package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// discoverBurst is the maximum number of tokens the bucket can hold,
	// which is also the maximum number of requests a client can make in
	// quick succession before the steady rate kicks in.
	discoverBurst = 10

	// discoverRatePerSec is the token refill rate: 10 requests per minute.
	discoverRatePerSec = 10.0 / 60.0

	// discoverDailyCap is the maximum requests a single IP may make per day.
	// Protects against sustained low-and-slow attacks that stay under the
	// per-minute limit.
	discoverDailyCap = 100

	// discoverAuthedBurst/RatePerSec/DailyCap apply once a request carries a
	// valid session instead of the anonymous IP limit above. A verified
	// email is harder to churn through than an IP address, so a logged-in
	// user gets a more generous — but still bounded — quota, keyed by user
	// id rather than IP. This is the "tighten discovery cost protection"
	// half of magic-link auth: it rewards signing in without removing the
	// anonymous path.
	discoverAuthedBurst      = 30
	discoverAuthedRatePerSec = 30.0 / 60.0
	discoverAuthedDailyCap   = 500

	// authRequestBurst/RatePerSec/DailyCap throttle POST /auth/request per
	// IP. Unlike /discover, a login request has no natural per-call cost
	// signal to lean on — the cost is entirely "did this send an email" —
	// so the limit is tight: a real user asks for at most a couple of links
	// in a sitting.
	authRequestBurst      = 5
	authRequestRatePerSec = 5.0 / (15.0 * 60.0)
	authRequestDailyCap   = 20
)

type ipState struct {
	// token bucket
	tokens   float64
	lastSeen time.Time
	// daily cap
	dayCount int
	dayReset time.Time
}

// rateLimiter is a generic per-key token-bucket-plus-daily-cap limiter. It
// backs the anonymous IP limit on /discover, the per-user limit on
// /discover for authenticated requests, and the per-IP limit on
// /auth/request — each with its own burst/rate/cap via newLimiter, and its
// own key space (IP address vs. "user:"+userID) so the three never collide
// in the same map.
type rateLimiter struct {
	mu         sync.Mutex
	state      map[string]*ipState
	burst      float64
	ratePerSec float64
	dailyCap   int
}

func newLimiter(burst float64, ratePerSec float64, dailyCap int) *rateLimiter {
	return &rateLimiter{
		state:      make(map[string]*ipState),
		burst:      burst,
		ratePerSec: ratePerSec,
		dailyCap:   dailyCap,
	}
}

func newDiscoverLimiter() *rateLimiter {
	return newLimiter(discoverBurst, discoverRatePerSec, discoverDailyCap)
}

func newDiscoverAuthedLimiter() *rateLimiter {
	return newLimiter(discoverAuthedBurst, discoverAuthedRatePerSec, discoverAuthedDailyCap)
}

func newAuthRequestLimiter() *rateLimiter {
	return newLimiter(authRequestBurst, authRequestRatePerSec, authRequestDailyCap)
}

// allow returns true if the request keyed by key should proceed. It is safe
// for concurrent use.
func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	s, ok := l.state[key]
	if !ok {
		s = &ipState{
			tokens:   l.burst,
			lastSeen: now,
			dayReset: now.Add(24 * time.Hour),
		}
		l.state[key] = s
	}

	// Refill tokens based on elapsed time since last request.
	elapsed := now.Sub(s.lastSeen).Seconds()
	s.tokens += elapsed * l.ratePerSec
	if s.tokens > l.burst {
		s.tokens = l.burst
	}
	s.lastSeen = now

	// Reset daily counter when the window has passed.
	if now.After(s.dayReset) {
		s.dayCount = 0
		s.dayReset = now.Add(24 * time.Hour)
	}

	// Daily cap is the harder limit — check it first.
	if s.dayCount >= l.dailyCap {
		return false
	}

	// Token bucket check.
	if s.tokens < 1.0 {
		return false
	}

	s.tokens--
	s.dayCount++
	return true
}

// discoverRateLimit wraps a handler with rate limiting for the discovery
// endpoint. A request carrying a valid session is limited per-user via
// userLimiter (and passes even if the request's IP is otherwise exhausted);
// anonymous requests — including one with an absent, expired, or unknown
// session cookie — fall back to the existing per-IP limit via ipLimiter.
// authn may be nil, which disables the authenticated path entirely (every
// request is treated as anonymous) — this is how the server behaves before
// magic-link auth is configured (no RESEND_API_KEY).
func discoverRateLimit(ipLimiter, userLimiter *rateLimiter, authn AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authn != nil {
			if token, ok := readSessionCookie(r); ok {
				if userID, ok := authn.AuthenticateSession(r.Context(), token); ok {
					if !userLimiter.allow("user:"+userID, time.Now()) {
						writeJSON(w, http.StatusTooManyRequests, errorResponse{"rate limit exceeded — try again shortly"})
						return
					}
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		if !ipLimiter.allow(realIP(r), time.Now()) {
			writeJSON(w, http.StatusTooManyRequests, errorResponse{"rate limit exceeded — try again shortly"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// realIP returns the client's IP address, preferring X-Forwarded-For (set by
// Render's load balancer) over RemoteAddr.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may be a comma-separated list; the first entry is
		// the originating client IP.
		if idx := strings.Index(xff, ","); idx != -1 {
			xff = xff[:idx]
		}
		if ip := strings.TrimSpace(xff); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
