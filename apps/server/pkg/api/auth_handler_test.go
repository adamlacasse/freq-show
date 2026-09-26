package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/auth"
)

// fakeAuthService is a hand-rolled AuthService for handler tests — separate
// from stubAuthService in ratelimit_test.go, which exists purely to
// exercise discoverRateLimit's AuthenticateSession call and errors out if
// its other two methods are ever reached.
type fakeAuthService struct {
	requestLoginFunc        func(ctx context.Context, email string) error
	verifyTokenFunc         func(ctx context.Context, rawToken string) (string, time.Time, error)
	authenticateSessionFunc func(ctx context.Context, rawSessionToken string) (string, bool)
}

func (f *fakeAuthService) RequestLogin(ctx context.Context, email string) error {
	if f.requestLoginFunc != nil {
		return f.requestLoginFunc(ctx, email)
	}
	return nil
}

func (f *fakeAuthService) VerifyToken(ctx context.Context, rawToken string) (string, time.Time, error) {
	if f.verifyTokenFunc != nil {
		return f.verifyTokenFunc(ctx, rawToken)
	}
	return "session-token", time.Now().Add(time.Hour), nil
}

func (f *fakeAuthService) AuthenticateSession(ctx context.Context, rawSessionToken string) (string, bool) {
	if f.authenticateSessionFunc != nil {
		return f.authenticateSessionFunc(ctx, rawSessionToken)
	}
	return "", false
}

func TestAuthRequestHandlerSendsLink(t *testing.T) {
	var gotEmail string
	svc := &fakeAuthService{
		requestLoginFunc: func(ctx context.Context, email string) error {
			gotEmail = email
			return nil
		},
	}

	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request", bytes.NewReader(body))
	res := httptest.NewRecorder()

	authRequestHandler(svc, nil).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}
	if gotEmail != "user@example.com" {
		t.Fatalf("expected RequestLogin to receive the submitted email, got %q", gotEmail)
	}
}

func TestAuthRequestHandlerRejectsInvalidEmail(t *testing.T) {
	svc := &fakeAuthService{
		requestLoginFunc: func(ctx context.Context, email string) error {
			return auth.ErrInvalidEmail
		},
	}

	body, _ := json.Marshal(map[string]string{"email": "not-an-email"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request", bytes.NewReader(body))
	res := httptest.NewRecorder()

	authRequestHandler(svc, nil).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf(status400Fmt, res.Code)
	}
}

func TestAuthRequestHandlerServiceUnavailableWhenNil(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request", bytes.NewReader(body))
	res := httptest.NewRecorder()

	authRequestHandler(nil, nil).ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}

func TestAuthRequestHandlerMailerUnconfigured(t *testing.T) {
	svc := &fakeAuthService{
		requestLoginFunc: func(ctx context.Context, email string) error {
			return auth.ErrMailerUnconfigured
		},
	}

	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request", bytes.NewReader(body))
	res := httptest.NewRecorder()

	authRequestHandler(svc, nil).ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}

func TestAuthRequestHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/request", nil)
	res := httptest.NewRecorder()

	authRequestHandler(&fakeAuthService{}, nil).ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", res.Code)
	}
}

func TestAuthRequestHandlerBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/request", bytes.NewReader([]byte("not json")))
	res := httptest.NewRecorder()

	authRequestHandler(&fakeAuthService{}, nil).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf(status400Fmt, res.Code)
	}
}

func TestAuthRequestHandlerRateLimited(t *testing.T) {
	limiter := newLimiter(0, 0, 100) // burst of 0 tokens: every request denied
	svc := &fakeAuthService{}

	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/request", bytes.NewReader(body))
	res := httptest.NewRecorder()

	authRequestHandler(svc, limiter).ServeHTTP(res, req)

	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", res.Code)
	}
}

// TestAuthVerifyHandlerGETDoesNotConsumeToken covers the fix for the
// email-scanner/prefetch problem: an automated GET against the link in the
// email (Microsoft Defender SafeLinks, Proofpoint, and some mobile
// browsers all do this before a user opens the message) must not call
// VerifyToken — if it did, the token would be burned before the real user
// ever clicked it.
func TestAuthVerifyHandlerGETDoesNotConsumeToken(t *testing.T) {
	svc := &fakeAuthService{
		verifyTokenFunc: func(ctx context.Context, rawToken string) (string, time.Time, error) {
			t.Fatal("GET must not call VerifyToken")
			return "", time.Time{}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=good-token", nil)
	res := httptest.NewRecorder()

	authVerifyHandler(svc, cookieConfig{secure: true}).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}
	if len(res.Result().Cookies()) != 0 {
		t.Fatal("expected no cookie to be set by GET")
	}
	body := res.Body.String()
	if !strings.Contains(body, `method="POST"`) {
		t.Fatalf("expected confirmation page with a POST form, got %q", body)
	}
	if !strings.Contains(body, `name="token" value="good-token"`) {
		t.Fatalf("expected the form to carry the token through as a hidden field, got %q", body)
	}
	if strings.Contains(body, `action="/auth/verify`) {
		t.Fatalf("expected a relative form action (not an absolute /auth/verify path, which bypasses the frontend's /api proxy), got %q", body)
	}
}

// TestAuthVerifyHandlerGETEscapesTokenInPage guards against reflected XSS:
// the token comes straight from the query string, and GET renders it back
// into an HTML attribute.
func TestAuthVerifyHandlerGETEscapesTokenInPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, `/auth/verify?token=%22%3E%3Cscript%3Ealert(1)%3C%2Fscript%3E`, nil)
	res := httptest.NewRecorder()

	authVerifyHandler(&fakeAuthService{}, cookieConfig{}).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}
	body := res.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("expected token to be escaped, got raw markup in body: %q", body)
	}
}

func TestAuthVerifyHandlerPOSTSetsSessionCookie(t *testing.T) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	svc := &fakeAuthService{
		verifyTokenFunc: func(ctx context.Context, rawToken string) (string, time.Time, error) {
			if rawToken != "good-token" {
				t.Fatalf("unexpected token %q", rawToken)
			}
			return "raw-session-token", expiresAt, nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/verify?token=good-token", nil)
	res := httptest.NewRecorder()

	authVerifyHandler(svc, cookieConfig{secure: true}).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf(status200Fmt, res.Code)
	}

	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie to be set, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName {
		t.Fatalf("unexpected cookie name %q", cookie.Name)
	}
	if cookie.Value != "raw-session-token" {
		t.Fatalf("unexpected cookie value %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected session cookie to be HttpOnly")
	}
	if !cookie.Secure {
		t.Fatal("expected session cookie to be Secure when cfg.secure is true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
	if cookie.MaxAge <= 0 {
		t.Fatalf("expected a positive MaxAge, got %d", cookie.MaxAge)
	}
}

func TestAuthVerifyHandlerPOSTRedirectsWhenFrontendURLConfigured(t *testing.T) {
	svc := &fakeAuthService{}

	req := httptest.NewRequest(http.MethodPost, "/auth/verify?token=good-token", nil)
	res := httptest.NewRecorder()

	authVerifyHandler(svc, cookieConfig{frontendURL: "https://app.example.com/welcome"}).ServeHTTP(res, req)

	if res.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", res.Code)
	}
	if loc := res.Header().Get("Location"); loc != "https://app.example.com/welcome" {
		t.Fatalf("unexpected redirect location %q", loc)
	}
}

func TestAuthVerifyHandlerPOSTRejectsInvalidToken(t *testing.T) {
	svc := &fakeAuthService{
		verifyTokenFunc: func(ctx context.Context, rawToken string) (string, time.Time, error) {
			return "", time.Time{}, auth.ErrTokenInvalid
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/verify?token=bad-token", nil)
	res := httptest.NewRecorder()

	authVerifyHandler(svc, cookieConfig{}).ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", res.Code)
	}
	if len(res.Result().Cookies()) != 0 {
		t.Fatal("expected no cookie to be set for an invalid token")
	}
}

func TestAuthVerifyHandlerRequiresTokenParam(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/auth/verify", nil)
		res := httptest.NewRecorder()

		authVerifyHandler(&fakeAuthService{}, cookieConfig{}).ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("%s: "+status400Fmt, method, res.Code)
		}
	}
}

func TestAuthVerifyHandlerServiceUnavailableWhenNil(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=x", nil)
	res := httptest.NewRecorder()

	authVerifyHandler(nil, cookieConfig{}).ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}

func TestAuthVerifyHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/auth/verify?token=x", nil)
	res := httptest.NewRecorder()

	authVerifyHandler(&fakeAuthService{}, cookieConfig{}).ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", res.Code)
	}
}

func TestReadSessionCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	if _, ok := readSessionCookie(req); ok {
		t.Fatal("expected no cookie to be found")
	}

	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "abc123"})
	token, ok := readSessionCookie(req)
	if !ok {
		t.Fatal("expected cookie to be found")
	}
	if token != "abc123" {
		t.Fatalf("unexpected token %q", token)
	}
}
