package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/auth"
)


// sessionCookieName is the cookie that carries the opaque session token
// issued by GET /auth/verify.
const sessionCookieName = "freqshow_session"

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
			default:
				log.Printf("auth: failed to send login email to %s: %v", body.Email, err)
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

// authVerifyHandler implements GET /auth/verify?token=...: exchanges a
// one-time magic-link token for a session cookie. On success it either
// redirects to cfg.frontendURL (when configured) or serves a minimal
// confirmation page — there's no single canonical frontend origin wired up
// yet, so a hard-coded redirect target would be wrong for local dev.
func authVerifyHandler(svc AuthService, cfg cookieConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodGet) {
			return
		}
		if svc == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{"login is not configured"})
			return
		}

		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{"token query parameter is required"})
			return
		}

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
			http.Redirect(w, r, cfg.frontendURL, http.StatusFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!doctype html><html><body><p>You're signed in to FreqShow. You can close this tab.</p></body></html>"))
	})
}

func setSessionCookie(w http.ResponseWriter, cfg cookieConfig, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
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
