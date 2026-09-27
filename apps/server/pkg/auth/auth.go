// Package auth implements passwordless magic-link authentication: issuing
// short-lived email login tokens, exchanging them for sessions, and
// resolving a session cookie back to a user. See BACKLOG.md "Magic link
// authentication" for the feature this backs.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

const (
	// TokenTTL is how long an emailed magic-link token remains valid.
	// Kept short (an assumed but conventional value for this kind of
	// flow) since the token travels in plaintext over email and only
	// needs to survive the time it takes to open an inbox.
	TokenTTL = 15 * time.Minute

	// SessionTTL is how long an issued session cookie remains valid.
	// Deliberately long — magic-link exists to replace remembering a
	// password, not to force repeated logins — but still bounded so a
	// stolen cookie doesn't work forever.
	SessionTTL = 30 * 24 * time.Hour

	// tokenBytes/sessionBytes are the amount of randomness backing each
	// bearer value: 256 bits, well beyond brute-force range.
	tokenBytes   = 32
	sessionBytes = 32

	// RequestCooldown is the minimum time between two magic-link emails to
	// the same address, independent of the caller's IP. IP-based rate
	// limiting (pkg/api's per-IP rateLimiter on /auth/request) is easy to
	// evade by rotating IPs or forged X-Forwarded-For values, but doing so
	// doesn't change the target *email* — so a per-address cooldown is a
	// backstop that protects the one thing that actually costs money and
	// annoys a real person: how often any given inbox gets emailed.
	RequestCooldown = 60 * time.Second

	// requestCooldownPruneEvery amortizes cleanup of the per-email cooldown
	// map: roughly every this-many RequestLogin calls, entries idle longer
	// than requestCooldownPruneAge are dropped. Without this the map would
	// grow by one entry per unique address ever requested and never
	// shrink. A counter-driven sweep (rather than a wall-clock ticker
	// goroutine) keeps this deterministic and easy to unit test.
	requestCooldownPruneEvery = 512
	requestCooldownPruneAge   = 24 * time.Hour
)

var (
	// ErrInvalidEmail is returned by RequestLogin for an empty or
	// malformed address.
	ErrInvalidEmail = errors.New("auth: invalid email address")

	// ErrTokenInvalid is returned by VerifyToken for a token that is
	// missing, unknown, expired, or already used. These cases are
	// deliberately indistinguishable to the caller: all of them mean "this
	// link doesn't work anymore," and distinguishing them would let an
	// attacker learn which tokens once existed.
	ErrTokenInvalid = errors.New("auth: sign-in link is invalid or has expired")

	// ErrMailerUnconfigured is returned by RequestLogin when no Mailer was
	// wired up (e.g. RESEND_API_KEY is unset).
	ErrMailerUnconfigured = errors.New("auth: email delivery is not configured")

	// ErrTooManyRequests is returned by RequestLogin when the same address
	// was already sent a link within RequestCooldown.
	ErrTooManyRequests = errors.New("auth: a sign-in link was already sent recently")
)

// Repository captures the persistence operations Service depends on. It is
// satisfied by db.AuthRepository (and therefore by db.Store); declared
// locally to keep this package independent of pkg/db's import graph.
type Repository interface {
	GetOrCreateUserByEmail(ctx context.Context, email string) (*data.User, error)
	GetUser(ctx context.Context, id string) (*data.User, error)
	SaveLoginToken(ctx context.Context, tokenHash, email string, expiresAt time.Time) error
	ConsumeLoginToken(ctx context.Context, tokenHash string, now time.Time) (email string, ok bool, err error)
	CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error
	GetSession(ctx context.Context, tokenHash string, now time.Time) (*data.Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

// Mailer sends the magic-link email. Implemented by pkg/sources/resend.
type Mailer interface {
	SendMagicLink(ctx context.Context, toEmail, link string) error
}

// Service implements magic-link issuance/verification and session lookups.
// Besides its dependencies, it holds a small in-memory map for the
// per-email request cooldown (see RequestCooldown) — safe for concurrent
// use behind its own mutex, same as Repository and Mailer implementations
// are expected to be (as db.Store and resend.Client both are).
type Service struct {
	repo    Repository
	mailer  Mailer
	baseURL string // the origin embedded in the emailed verification link — see New's doc comment

	mu            sync.Mutex
	lastRequestAt map[string]time.Time // normalized email -> last RequestLogin time
	requestCalls  uint64               // guards the amortized prune in allowEmailRequest
}

// New constructs a Service. baseURL is embedded directly into the emailed
// /auth/verify link, so it must be whatever origin a BROWSER should use to
// reach it — not necessarily the backend's own direct URL. In this repo's
// deployment (see apps/frontend/server.ts and proxy.conf.json), the
// frontend proxies /api/* to the backend, and the browser only ever talks
// to the frontend's origin; a link built from the backend's own URL sets
// its Set-Cookie response on an origin the browser will never send back on
// later /discover calls. Pass the frontend's origin with an /api prefix
// (config.AuthConfig.BaseURL carries the resolved value).
func New(repo Repository, mailer Mailer, baseURL string) *Service {
	return &Service{
		repo:          repo,
		mailer:        mailer,
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		lastRequestAt: make(map[string]time.Time),
	}
}

// RequestLogin validates the address and, if it looks like an email, issues
// a fresh magic-link token and emails it. Account creation is deferred to
// VerifyToken (on first successful click) rather than happening here, so
// this method never reveals whether an address already has an account.
func (s *Service) RequestLogin(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if s.mailer == nil {
		return ErrMailerUnconfigured
	}
	if !s.allowEmailRequest(normalized, time.Now()) {
		return ErrTooManyRequests
	}

	raw, hash, err := newOpaqueToken(tokenBytes)
	if err != nil {
		return err
	}

	if err := s.repo.SaveLoginToken(ctx, hash, normalized, time.Now().UTC().Add(TokenTTL)); err != nil {
		return err
	}

	link := s.baseURL + "/auth/verify?token=" + raw
	return s.mailer.SendMagicLink(ctx, normalized, link)
}

// allowEmailRequest enforces RequestCooldown for a single normalized
// address, recording this attempt as the new "last requested at" time when
// it's allowed. It also periodically prunes idle entries so the map
// doesn't grow forever.
func (s *Service) allowEmailRequest(email string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requestCalls++
	if s.requestCalls%requestCooldownPruneEvery == 0 {
		for key, last := range s.lastRequestAt {
			if now.Sub(last) > requestCooldownPruneAge {
				delete(s.lastRequestAt, key)
			}
		}
	}

	if last, ok := s.lastRequestAt[email]; ok && now.Sub(last) < RequestCooldown {
		return false
	}
	s.lastRequestAt[email] = now
	return true
}

// VerifyToken consumes a one-time magic-link token — creating the user
// record on first login — and returns a freshly issued session token and
// its expiry. The raw session token is what belongs in the session cookie;
// only its hash is ever persisted.
func (s *Service) VerifyToken(ctx context.Context, rawToken string) (sessionToken string, expiresAt time.Time, err error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", time.Time{}, ErrTokenInvalid
	}

	email, ok, err := s.repo.ConsumeLoginToken(ctx, hashToken(rawToken), time.Now().UTC())
	if err != nil {
		return "", time.Time{}, err
	}
	if !ok {
		return "", time.Time{}, ErrTokenInvalid
	}

	user, err := s.repo.GetOrCreateUserByEmail(ctx, email)
	if err != nil {
		return "", time.Time{}, err
	}

	rawSession, sessionHash, err := newOpaqueToken(sessionBytes)
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt = time.Now().UTC().Add(SessionTTL)
	if err := s.repo.CreateSession(ctx, sessionHash, user.ID, expiresAt); err != nil {
		return "", time.Time{}, err
	}

	return rawSession, expiresAt, nil
}

// AuthenticateSession looks up the session behind a raw cookie value and
// returns the associated user id. ok is false for a missing, expired, or
// unknown token; callers should treat that identically to "not logged in"
// (falling back to anonymous/IP-based behavior) rather than surfacing an
// error for what is a routine, expected state.
func (s *Service) AuthenticateSession(ctx context.Context, rawSessionToken string) (userID string, ok bool) {
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return "", false
	}

	session, err := s.repo.GetSession(ctx, hashToken(rawSessionToken), time.Now().UTC())
	if err != nil || session == nil {
		return "", false
	}
	return session.UserID, true
}

// GetCurrentUser resolves a raw session token to its User entity.
// Returns (nil, nil) if the session is missing, expired, or invalid.
func (s *Service) GetCurrentUser(ctx context.Context, rawSessionToken string) (*data.User, error) {
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return nil, nil
	}

	session, err := s.repo.GetSession(ctx, hashToken(rawSessionToken), time.Now().UTC())
	if err != nil || session == nil {
		return nil, err
	}
	return s.repo.GetUser(ctx, session.UserID)
}

// Logout deletes the session behind a raw cookie value, if any.
func (s *Service) Logout(ctx context.Context, rawSessionToken string) error {
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, hashToken(rawSessionToken))
}

func normalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", ErrInvalidEmail
	}
	return strings.ToLower(addr.Address), nil
}

// newOpaqueToken generates n bytes of cryptographically random data and
// returns both its URL-safe raw encoding (what the caller hands out — as an
// email link or a cookie value) and its SHA-256 hash (what gets persisted).
func newOpaqueToken(n int) (raw string, hash string, err error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
