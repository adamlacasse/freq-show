package api

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/auth"
)

// sessionCookieName is the cookie that carries the opaque session token
// issued by GET /auth/verify.
const sessionCookieName = "freqshow_session"

// maxAuthRequestBodyBytes caps the size of the POST /auth/request body.
// The real payload is a single email address; 64KiB is generous headroom
// while still ruling out a client trying to exhaust memory with an
// oversized request.
const maxAuthRequestBodyBytes = 64 * 1024

// AuthService captures the magic-link auth operations the router relies on.
// Implemented by *auth.Service; declared here (rather than imported
// directly) to match the DI style used for the other external dependencies
// in RouterConfig.
type AuthService interface {
	RequestLogin(ctx context.Context, email string) error
	VerifyToken(ctx context.Context, rawToken string) (sessionToken string, expiresAt time.Time, err error)
	AuthenticateSession(ctx context.Context, rawSessionToken string) (userID string, ok bool)
}

// cookieConfig captures the deployment-specific bits of the session cookie:
// whether it should be marked Secure (only sent over HTTPS — on in every
// environment except local development, where the dev server is plain
// HTTP) and where to send the browser after a successful /auth/verify.
type cookieConfig struct {
	secure      bool
	frontendURL string
}

// authRequestHandler implements POST /auth/request: accepts a JSON body of
// the form {"email": "..."} and, if a mailer is configured, sends a magic
// sign-in link. The response is the same regardless of whether the address
// already has an account — accounts are created lazily on first verify, so
// there's nothing to enumerate — but a malformed address still gets a 400
// so the frontend can show a validation error inline.
func authRequestHandler(svc AuthService, limiter *rateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodPost) {
			return
		}
		if svc == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{"login is not configured"})
			return
		}
		if limiter != nil && !limiter.allow(realIP(r), time.Now()) {
			writeJSON(w, http.StatusTooManyRequests, errorResponse{"too many login requests — try again later"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxAuthRequestBodyBytes)

		var body struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
			return
		}

		if err := svc.RequestLogin(r.Context(), body.Email); err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidEmail):
				writeJSON(w, http.StatusBadRequest, errorResponse{"a valid email address is required"})
			case errors.Is(err, auth.ErrMailerUnconfigured):
				writeJSON(w, http.StatusServiceUnavailable, errorResponse{"login is not configured"})
			case errors.Is(err, auth.ErrTooManyRequests):
				writeJSON(w, http.StatusTooManyRequests, errorResponse{"a sign-in link was already sent recently — check your email"})
			default:
				// Deliberately not logging body.Email alongside this: it's
				// the one piece of PII this handler ever sees, and the
				// failure itself (a Resend/network error) is diagnosable
				// without it.
				log.Printf("auth: failed to send login email: %v", err)
				writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to send login email"})
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "check your email for a sign-in link",
		})
	})
}

// verifyConfirmPage is served by GET /auth/verify?token=... and is
// deliberately inert: it names the token in a form that POSTs back to this
// same endpoint, but does not itself consume the token or set a cookie.
//
// This matters because GET is not actually a safe, no-side-effects request
// here in practice: enterprise email security gateways (Microsoft Defender
// SafeLinks, Proofpoint) and some mobile browsers automatically fetch links
// found in incoming email — via GET — before the recipient ever opens the
// message, to scan the destination for phishing/malware. If GET itself
// consumed the one-time token, that automated fetch would burn it, and the
// real user clicking the link afterward would see it as "already used."
// Requiring an explicit form submission (POST) to actually complete
// sign-in sidesteps this: scanners fetch and render the link, they don't
// submit forms.
var verifyConfirmPage = template.Must(template.New("verify-confirm").Parse(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>Sign in to FreqShow</title></head>
<body>
<p>Click below to finish signing in to FreqShow.</p>
<form method="POST" action="">
  <input type="hidden" name="token" value="{{.}}">
  <button type="submit">Complete sign-in</button>
</form>
</body>
</html>`))

// authVerifyHandler implements the magic-link verification endpoint.
// GET /auth/verify?token=... renders a non-destructive confirmation page
// (see verifyConfirmPage); POST /auth/verify?token=... — the form's own
// submission — is what actually consumes the token, creates the session,
// and sets the cookie. On success it either redirects to cfg.frontendURL
// (when configured) or serves a minimal confirmation page — there's no
// single canonical frontend origin wired up yet, so a hard-coded redirect
// target would be wrong for local dev.
func authVerifyHandler(svc AuthService, cfg cookieConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{"login is not configured"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			token := strings.TrimSpace(r.URL.Query().Get("token"))
			if token == "" {
				writeJSON(w, http.StatusBadRequest, errorResponse{"token query parameter is required"})
				return
			}
			// no-store: this page (and the token it carries) must never be
			// served from a shared/browser cache to a different visitor.
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = verifyConfirmPage.Execute(w, token)
		case http.MethodPost:
			// The confirm page's form posts back with action="" (this same
			// URL, query string included — see verifyConfirmPage) plus the
			// token as a hidden field, so read whichever is present rather
			// than assuming one: a relative empty action is not honored
			// identically everywhere, and either source is equally trusted.
			if err := r.ParseForm(); err != nil {
				writeJSON(w, http.StatusBadRequest, errorResponse{"invalid form submission"})
				return
			}
			token := strings.TrimSpace(r.FormValue("token"))
			if token == "" {
				token = strings.TrimSpace(r.URL.Query().Get("token"))
			}
			if token == "" {
				writeJSON(w, http.StatusBadRequest, errorResponse{"token is required"})
				return
			}
			completeVerify(w, r, svc, cfg, token)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// completeVerify performs the actual, destructive half of verification:
// consuming the token and issuing a session. Split out from
// authVerifyHandler so the GET path above can never reach it by accident.
func completeVerify(w http.ResponseWriter, r *http.Request, svc AuthService, cfg cookieConfig, token string) {
	// no-store applies to every response this handler can produce,
	// success or failure alike — none of them should be cached.
	w.Header().Set("Cache-Control", "no-store")

	sessionToken, expiresAt, err := svc.VerifyToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, auth.ErrTokenInvalid) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{"sign-in link is invalid or has expired"})
			return
		}
		log.Printf("auth: failed to verify token: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{"failed to complete sign-in"})
		return
	}

	setSessionCookie(w, cfg, sessionToken, expiresAt)

	if cfg.frontendURL != "" {
		// 303 (not 302): this redirect follows a state-changing POST, and
		// See Other is what tells the browser to GET the target regardless
		// of the original method — the standard Post/Redirect/Get pattern.
		http.Redirect(w, r, cfg.frontendURL, http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<!doctype html><html><body><p>You're signed in to FreqShow. You can close this tab.</p></body></html>"))
}

func setSessionCookie(w http.ResponseWriter, cfg cookieConfig, token string, expiresAt time.Time) {
	// MaxAge alongside Expires: browsers prefer MaxAge when both are
	// present, which sidesteps any client clock skew relative to the
	// Expires timestamp. A non-positive MaxAge would tell the browser to
	// delete the cookie immediately, so guard the (should-never-happen)
	// case where expiresAt is already in the past.
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cfg.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// readSessionCookie extracts the raw session token from the request, if
// present. It makes no claim about validity — that's AuthService's job.
func readSessionCookie(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}
