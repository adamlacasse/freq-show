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
	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
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
	GetCurrentUser(ctx context.Context, rawSessionToken string) (*data.User, error)
	Logout(ctx context.Context, rawSessionToken string) error
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
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sign in to FreqShow!</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: #0f172a;
      color: #f5f1e0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Inter, Helvetica, Arial, sans-serif;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 1.5rem;
      background-image:
        radial-gradient(circle at 20% 20%, rgba(251, 113, 133, 0.12), transparent 50%),
        radial-gradient(circle at 80% 80%, rgba(45, 212, 191, 0.1), transparent 50%);
    }
    .card {
      background: rgba(17, 24, 39, 0.9);
      border: 1px solid rgba(255, 255, 255, 0.12);
      border-radius: 1.5rem;
      padding: 2.5rem 2rem;
      max-width: 26rem;
      width: 100%;
      text-align: center;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
    }
    .icon-badge {
      width: 3.5rem;
      height: 3.5rem;
      background: rgba(45, 212, 191, 0.15);
      color: #2dd4bf;
      border-radius: 1rem;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 1.25rem;
    }
    h1 {
      font-size: 1.5rem;
      font-weight: 700;
      letter-spacing: -0.025em;
      color: #ffffff;
      margin-bottom: 0.5rem;
    }
    p {
      color: rgba(245, 241, 224, 0.7);
      font-size: 0.925rem;
      line-height: 1.5;
      margin-bottom: 1.75rem;
    }
    .btn {
      display: inline-block;
      width: 100%;
      padding: 0.875rem 1.5rem;
      background: #2dd4bf;
      color: #0f172a;
      font-weight: 600;
      font-size: 0.95rem;
      border: none;
      border-radius: 0.75rem;
      cursor: pointer;
      text-decoration: none;
      transition: background-color 0.15s ease, transform 0.1s ease;
    }
    .btn:hover {
      background: #14b8a6;
    }
    .btn:active {
      transform: scale(0.98);
    }
    .footer {
      margin-top: 1.5rem;
      font-size: 0.75rem;
      color: rgba(245, 241, 224, 0.4);
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon-badge">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M13.8 12H3"/>
      </svg>
    </div>
    <h1>Sign in to FreqShow!</h1>
    <p>Click below to complete your sign-in and open your session.</p>
    <form method="POST" action="">
      <input type="hidden" name="token" value="{{.}}">
      <button class="btn" type="submit">Complete sign-in</button>
    </form>
    <div class="footer">FreqShow &bull; Liner notes for music lovers</div>
  </div>
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
	_, _ = w.Write(verifySuccessPage)
}

var verifySuccessPage = []byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Signed In — FreqShow</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: #0f172a;
      color: #f5f1e0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Inter, Helvetica, Arial, sans-serif;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 1.5rem;
      background-image:
        radial-gradient(circle at 20% 20%, rgba(251, 113, 133, 0.12), transparent 50%),
        radial-gradient(circle at 80% 80%, rgba(45, 212, 191, 0.1), transparent 50%);
    }
    .card {
      background: rgba(17, 24, 39, 0.9);
      border: 1px solid rgba(255, 255, 255, 0.12);
      border-radius: 1.5rem;
      padding: 2.5rem 2rem;
      max-width: 26rem;
      width: 100%;
      text-align: center;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
    }
    .icon-badge {
      width: 3.5rem;
      height: 3.5rem;
      background: rgba(45, 212, 191, 0.15);
      color: #2dd4bf;
      border-radius: 1rem;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 1.25rem;
    }
    h1 {
      font-size: 1.5rem;
      font-weight: 700;
      letter-spacing: -0.025em;
      color: #ffffff;
      margin-bottom: 0.5rem;
    }
    p {
      color: rgba(245, 241, 224, 0.7);
      font-size: 0.925rem;
      line-height: 1.5;
      margin-bottom: 1.75rem;
    }
    .btn {
      display: inline-block;
      width: 100%;
      padding: 0.875rem 1.5rem;
      background: #2dd4bf;
      color: #0f172a;
      font-weight: 600;
      font-size: 0.95rem;
      border: none;
      border-radius: 0.75rem;
      cursor: pointer;
      text-decoration: none;
      transition: background-color 0.15s ease;
    }
    .btn:hover {
      background: #14b8a6;
    }
    .footer {
      margin-top: 1.5rem;
      font-size: 0.75rem;
      color: rgba(245, 241, 224, 0.4);
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon-badge">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M20 6L9 17l-5-5"/>
      </svg>
    </div>
    <h1>You're signed in</h1>
    <p>Your session has started. You can return to FreqShow or close this tab.</p>
    <a href="/" class="btn">Open FreqShow</a>
    <div class="footer">FreqShow &bull; Liner notes for music lovers</div>
  </div>
</body>
</html>`)

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

// authMeHandler implements GET /auth/me: inspects the session cookie and returns
// the authenticated user record, or 401 if unauthenticated.
func authMeHandler(svc AuthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if svc == nil {
			writeJSON(w, http.StatusUnauthorized, errorResponse{"not authenticated"})
			return
		}

		token, ok := readSessionCookie(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{"not authenticated"})
			return
		}

		user, err := svc.GetCurrentUser(r.Context(), token)
		if err != nil || user == nil {
			writeJSON(w, http.StatusUnauthorized, errorResponse{"not authenticated"})
			return
		}

		writeJSON(w, http.StatusOK, user)
	})
}

// authLogoutHandler implements POST /auth/logout: terminates the session in the database
// and clears the session cookie.
func authLogoutHandler(svc AuthService, cfg cookieConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodPost) {
			return
		}
		w.Header().Set("Cache-Control", "no-store")

		if svc != nil {
			if token, ok := readSessionCookie(r); ok {
				_ = svc.Logout(r.Context(), token)
			}
		}

		clearSessionCookie(w, cfg)
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "signed out",
		})
	})
}

func clearSessionCookie(w http.ResponseWriter, cfg cookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.secure,
		SameSite: http.SameSiteLaxMode,
	})
}
