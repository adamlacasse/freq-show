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
)

type ipState struct {
	// token bucket
	tokens   float64
	lastSeen time.Time
	// daily cap
	dayCount int
	dayReset time.Time
}

type discoverLimiter struct {
	mu    sync.Mutex
	state map[string]*ipState
}

func newDiscoverLimiter() *discoverLimiter {
	return &discoverLimiter{state: make(map[string]*ipState)}
}

// allow returns true if the request from ip should proceed. It is safe for
// concurrent use.
func (l *discoverLimiter) allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	s, ok := l.state[ip]
	if !ok {
		s = &ipState{
			tokens:   discoverBurst,
			lastSeen: now,
			dayReset: now.Add(24 * time.Hour),
		}
		l.state[ip] = s
	}

	// Refill tokens based on elapsed time since last request.
	elapsed := now.Sub(s.lastSeen).Seconds()
	s.tokens += elapsed * discoverRatePerSec
	if s.tokens > discoverBurst {
		s.tokens = discoverBurst
	}
	s.lastSeen = now

	// Reset daily counter when the window has passed.
	if now.After(s.dayReset) {
		s.dayCount = 0
		s.dayReset = now.Add(24 * time.Hour)
	}

	// Daily cap is the harder limit — check it first.
	if s.dayCount >= discoverDailyCap {
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

// discoverRateLimit wraps a handler with per-IP rate limiting for the
// discovery endpoint. Both limits must pass for the request to proceed.
func discoverRateLimit(l *discoverLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(realIP(r), time.Now()) {
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
