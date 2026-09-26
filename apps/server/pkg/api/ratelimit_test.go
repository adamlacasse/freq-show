package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscoverLimiterAllowsUpToBurst(t *testing.T) {
	l := newDiscoverLimiter()
	now := time.Now()
	for i := range discoverBurst {
		if !l.allow("1.2.3.4", now) {
			t.Fatalf("request %d should be allowed (within burst)", i+1)
		}
	}
	if l.allow("1.2.3.4", now) {
		t.Fatal("request beyond burst should be denied")
	}
}

func TestDiscoverLimiterRefillsOverTime(t *testing.T) {
	l := newDiscoverLimiter()
	now := time.Now()

	// Exhaust the bucket.
	for range discoverBurst {
		l.allow("1.2.3.4", now)
	}
	if l.allow("1.2.3.4", now) {
		t.Fatal("expected denial immediately after burst")
	}

	// Advance 6 seconds — enough to earn back 1 token at 10/min.
	later := now.Add(6 * time.Second)
	if !l.allow("1.2.3.4", later) {
		t.Fatal("expected 1 token to refill after 6 seconds")
	}
}

func TestDiscoverLimiterIsolatesIPs(t *testing.T) {
	l := newDiscoverLimiter()
	now := time.Now()

	// Exhaust IP A.
	for range discoverBurst {
		l.allow("1.1.1.1", now)
	}
	if l.allow("1.1.1.1", now) {
		t.Fatal("1.1.1.1 should be rate limited")
	}

	// IP B should still have a full bucket.
	if !l.allow("2.2.2.2", now) {
		t.Fatal("2.2.2.2 should not be affected by 1.1.1.1's limit")
	}
}

func TestDiscoverLimiterDailyCap(t *testing.T) {
	l := newDiscoverLimiter()

	// Drain the daily cap by advancing time 6 seconds between each request
	// (enough to keep the token bucket full but still hit the daily cap).
	now := time.Now()
	for i := range discoverDailyCap {
		if !l.allow("1.2.3.4", now) {
			t.Fatalf("request %d should be allowed (daily cap not yet reached)", i+1)
		}
		now = now.Add(6 * time.Second)
	}
	if l.allow("1.2.3.4", now) {
		t.Fatal("request beyond daily cap should be denied")
	}
}

func TestDiscoverLimiterDailyCapResetsAfter24Hours(t *testing.T) {
	l := newDiscoverLimiter()
	now := time.Now()

	// Hit the daily cap.
	for i := range discoverDailyCap {
		l.allow("1.2.3.4", now.Add(time.Duration(i)*6*time.Second))
	}

	// 24 hours later, the window resets.
	tomorrow := now.Add(25 * time.Hour)
	if !l.allow("1.2.3.4", tomorrow) {
		t.Fatal("daily cap should reset after 24 hours")
	}
}

func TestRealIPPrefersXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/discover", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	r.RemoteAddr = "10.0.0.1:54321"

	if got := realIP(r); got != "203.0.113.5" {
		t.Fatalf("expected 203.0.113.5, got %q", got)
	}
}

func TestRealIPFallsBackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/discover", nil)
	r.RemoteAddr = "192.168.1.1:12345"

	if got := realIP(r); got != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %q", got)
	}
}

func TestDiscoverRateLimitMiddlewareReturns429(t *testing.T) {
	l := newDiscoverLimiter()
	now := time.Now()

	// Pre-exhaust the limiter for this IP via the internal method so the
	// middleware sees an already-limited client.
	for range discoverBurst {
		l.allow("9.9.9.9", now)
	}

	handler := discoverRateLimit(l, newDiscoverAuthedLimiter(), nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/discover", nil)
	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

// stubAuthService lets discoverRateLimit tests control session
// authentication without depending on the real auth package.
type stubAuthService struct {
	authenticateFunc func(ctx context.Context, rawSessionToken string) (string, bool)
}

func (s *stubAuthService) RequestLogin(ctx context.Context, email string) error {
	return errUnexpectedStubCall
}

func (s *stubAuthService) VerifyToken(ctx context.Context, rawToken string) (string, time.Time, error) {
	return "", time.Time{}, errUnexpectedStubCall
}

func (s *stubAuthService) AuthenticateSession(ctx context.Context, rawSessionToken string) (string, bool) {
	if s.authenticateFunc != nil {
		return s.authenticateFunc(ctx, rawSessionToken)
	}
	return "", false
}

var errUnexpectedStubCall = errors.New("unexpected call on stubAuthService")

// TestDiscoverRateLimitPrefersSessionOverIP verifies the optional-auth
// behavior required by BACKLOG.md: a request with a valid session cookie is
// limited per-user, even when the request's IP has already exhausted its
// anonymous quota.
func TestDiscoverRateLimitPrefersSessionOverIP(t *testing.T) {
	ipLimiter := newDiscoverLimiter()
	now := time.Now()
	for range discoverBurst {
		ipLimiter.allow("9.9.9.9", now)
	}
	if ipLimiter.allow("9.9.9.9", now) {
		t.Fatal("test setup: expected IP limiter to be exhausted")
	}

	authn := &stubAuthService{
		authenticateFunc: func(ctx context.Context, rawSessionToken string) (string, bool) {
			if rawSessionToken != "valid-session" {
				return "", false
			}
			return "user-1", true
		},
	}

	handler := discoverRateLimit(ipLimiter, newDiscoverAuthedLimiter(), authn, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/discover", nil)
	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "valid-session"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected authenticated request to bypass the exhausted IP limit, got %d", w.Code)
	}
}

// TestDiscoverRateLimitFallsBackToIPForInvalidSession covers an absent,
// unknown, or expired session cookie: AuthenticateSession returns ok=false,
// and the request must fall back to (and be bound by) the anonymous IP
// limit rather than sailing through unlimited.
func TestDiscoverRateLimitFallsBackToIPForInvalidSession(t *testing.T) {
	ipLimiter := newDiscoverLimiter()
	now := time.Now()
	for range discoverBurst {
		ipLimiter.allow("9.9.9.9", now)
	}

	authn := &stubAuthService{
		authenticateFunc: func(ctx context.Context, rawSessionToken string) (string, bool) {
			return "", false
		},
	}

	handler := discoverRateLimit(ipLimiter, newDiscoverAuthedLimiter(), authn, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/discover", nil)
	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "stale-or-unknown"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected an invalid session to fall back to the exhausted IP limit, got %d", w.Code)
	}
}

// TestDiscoverRateLimitNilAuthServiceActsAnonymous covers the pre-magic-link
// deployment state (no RESEND_API_KEY, so RouterConfig.Auth is nil): every
// request must be treated as anonymous, exactly as /discover behaved before
// this feature existed.
func TestDiscoverRateLimitNilAuthServiceActsAnonymous(t *testing.T) {
	ipLimiter := newDiscoverLimiter()
	now := time.Now()
	for range discoverBurst {
		ipLimiter.allow("9.9.9.9", now)
	}

	handler := discoverRateLimit(ipLimiter, newDiscoverAuthedLimiter(), nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPost, "/discover", nil)
	r.Header.Set("X-Forwarded-For", "9.9.9.9")
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "whatever"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected nil AuthService to behave anonymously (IP-limited), got %d", w.Code)
	}
}

// TestDiscoverRateLimitAuthedUserHasOwnCap verifies the authenticated path
// still enforces its own (more generous) cap rather than being unlimited.
func TestDiscoverRateLimitAuthedUserHasOwnCap(t *testing.T) {
	ipLimiter := newDiscoverLimiter()
	userLimiter := newLimiter(1, discoverAuthedRatePerSec, discoverAuthedDailyCap)

	authn := &stubAuthService{
		authenticateFunc: func(ctx context.Context, rawSessionToken string) (string, bool) {
			return "user-1", true
		},
	}

	handler := discoverRateLimit(ipLimiter, userLimiter, authn, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	newReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/discover", nil)
		r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "valid-session"})
		return r
	}

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, newReq())
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first authenticated request to succeed, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, newReq())
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second authenticated request to exhaust the 1-token user bucket, got %d", w2.Code)
	}
}
