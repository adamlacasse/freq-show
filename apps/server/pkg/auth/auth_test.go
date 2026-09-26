package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
)

// fakeLoginToken/fakeSession/fakeRepo give tests a tiny in-memory
// Repository without pulling in package db (which would make this an
// integration test of the SQLite/Memory stores rather than of Service's own
// logic — those get their own tests in pkg/db).
type fakeLoginToken struct {
	email      string
	expiresAt  time.Time
	consumedAt time.Time
}

type fakeSession struct {
	userID    string
	expiresAt time.Time
}

type fakeRepo struct {
	usersByEmail map[string]*data.User
	loginTokens  map[string]*fakeLoginToken
	sessions     map[string]*fakeSession
	nextUserID   int

	// failGetOrCreate, when non-nil, is returned by GetOrCreateUserByEmail
	// instead of the normal lookup/create logic.
	failGetOrCreate error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		usersByEmail: make(map[string]*data.User),
		loginTokens:  make(map[string]*fakeLoginToken),
		sessions:     make(map[string]*fakeSession),
	}
}

func (r *fakeRepo) GetOrCreateUserByEmail(ctx context.Context, email string) (*data.User, error) {
	if r.failGetOrCreate != nil {
		return nil, r.failGetOrCreate
	}
	if u, ok := r.usersByEmail[email]; ok {
		return u, nil
	}
	r.nextUserID++
	u := &data.User{ID: strings.ToLower(email) + "-id", Email: email}
	r.usersByEmail[email] = u
	return u, nil
}

func (r *fakeRepo) SaveLoginToken(ctx context.Context, tokenHash, email string, expiresAt time.Time) error {
	r.loginTokens[tokenHash] = &fakeLoginToken{email: email, expiresAt: expiresAt}
	return nil
}

func (r *fakeRepo) ConsumeLoginToken(ctx context.Context, tokenHash string, now time.Time) (string, bool, error) {
	tok, ok := r.loginTokens[tokenHash]
	if !ok {
		return "", false, nil
	}
	if !tok.consumedAt.IsZero() || now.After(tok.expiresAt) {
		return "", false, nil
	}
	tok.consumedAt = now
	return tok.email, true, nil
}

func (r *fakeRepo) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	r.sessions[tokenHash] = &fakeSession{userID: userID, expiresAt: expiresAt}
	return nil
}

func (r *fakeRepo) GetSession(ctx context.Context, tokenHash string, now time.Time) (*data.Session, error) {
	s, ok := r.sessions[tokenHash]
	if !ok || now.After(s.expiresAt) {
		return nil, nil
	}
	return &data.Session{UserID: s.userID}, nil
}

func (r *fakeRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	delete(r.sessions, tokenHash)
	return nil
}

type fakeMailer struct {
	sentTo   string
	sentLink string
	err      error
	calls    int
}

func (m *fakeMailer) SendMagicLink(ctx context.Context, toEmail, link string) error {
	m.calls++
	m.sentTo = toEmail
	m.sentLink = link
	return m.err
}

func TestRequestLoginSendsLinkForValidEmail(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "  User@Example.com  "); err != nil {
		t.Fatalf("RequestLogin returned error: %v", err)
	}

	if mailer.calls != 1 {
		t.Fatalf("expected mailer to be called once, got %d", mailer.calls)
	}
	if mailer.sentTo != "user@example.com" {
		t.Fatalf("expected normalized lowercase email, got %q", mailer.sentTo)
	}
	if !strings.HasPrefix(mailer.sentLink, "https://api.example.com/auth/verify?token=") {
		t.Fatalf("unexpected link: %q", mailer.sentLink)
	}
	if len(repo.loginTokens) != 1 {
		t.Fatalf("expected one login token to be saved, got %d", len(repo.loginTokens))
	}
}

func TestRequestLoginRejectsInvalidEmail(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	cases := []string{"", "   ", "not-an-email", "@example.com"}
	for _, email := range cases {
		if err := svc.RequestLogin(context.Background(), email); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("email %q: expected ErrInvalidEmail, got %v", email, err)
		}
	}
	if mailer.calls != 0 {
		t.Fatalf("expected no emails sent for invalid addresses, got %d", mailer.calls)
	}
}

func TestRequestLoginRequiresMailer(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "user@example.com"); !errors.Is(err, ErrMailerUnconfigured) {
		t.Fatalf("expected ErrMailerUnconfigured, got %v", err)
	}
}

func TestVerifyTokenIssuesSessionAndCreatesUser(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("RequestLogin returned error: %v", err)
	}

	rawToken := extractToken(t, mailer.sentLink)

	sessionToken, expiresAt, err := svc.VerifyToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if sessionToken == "" {
		t.Fatal("expected a non-empty session token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expected session expiry in the future, got %v", expiresAt)
	}
	if _, ok := repo.usersByEmail["user@example.com"]; !ok {
		t.Fatal("expected user account to be created on first verify")
	}

	userID, ok := svc.AuthenticateSession(context.Background(), sessionToken)
	if !ok {
		t.Fatal("expected the freshly issued session to authenticate")
	}
	if userID != repo.usersByEmail["user@example.com"].ID {
		t.Fatalf("unexpected user id from session: %q", userID)
	}
}

func TestVerifyTokenIsSingleUse(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("RequestLogin returned error: %v", err)
	}
	rawToken := extractToken(t, mailer.sentLink)

	if _, _, err := svc.VerifyToken(context.Background(), rawToken); err != nil {
		t.Fatalf("first VerifyToken returned error: %v", err)
	}

	if _, _, err := svc.VerifyToken(context.Background(), rawToken); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid on reuse, got %v", err)
	}
}

func TestVerifyTokenRejectsExpiredToken(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("RequestLogin returned error: %v", err)
	}
	rawToken := extractToken(t, mailer.sentLink)

	// Force every stored token to look expired regardless of TokenTTL.
	for _, tok := range repo.loginTokens {
		tok.expiresAt = time.Now().Add(-time.Minute)
	}

	if _, _, err := svc.VerifyToken(context.Background(), rawToken); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid for expired token, got %v", err)
	}
}

func TestVerifyTokenRejectsUnknownToken(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, &fakeMailer{}, "https://api.example.com")

	if _, _, err := svc.VerifyToken(context.Background(), "not-a-real-token"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}

	if _, _, err := svc.VerifyToken(context.Background(), ""); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid for empty token, got %v", err)
	}
}

func TestAuthenticateSessionRejectsUnknownOrEmptyToken(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, &fakeMailer{}, "https://api.example.com")

	if _, ok := svc.AuthenticateSession(context.Background(), ""); ok {
		t.Fatal("expected empty token to fail authentication")
	}
	if _, ok := svc.AuthenticateSession(context.Background(), "unknown"); ok {
		t.Fatal("expected unknown token to fail authentication")
	}
}

func TestAuthenticateSessionRejectsExpiredSession(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("RequestLogin returned error: %v", err)
	}
	rawToken := extractToken(t, mailer.sentLink)

	sessionToken, _, err := svc.VerifyToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}

	for _, s := range repo.sessions {
		s.expiresAt = time.Now().Add(-time.Minute)
	}

	if _, ok := svc.AuthenticateSession(context.Background(), sessionToken); ok {
		t.Fatal("expected expired session to fail authentication")
	}
}

func TestLogoutDeletesSession(t *testing.T) {
	repo := newFakeRepo()
	mailer := &fakeMailer{}
	svc := New(repo, mailer, "https://api.example.com")

	if err := svc.RequestLogin(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("RequestLogin returned error: %v", err)
	}
	rawToken := extractToken(t, mailer.sentLink)
	sessionToken, _, err := svc.VerifyToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}

	if err := svc.Logout(context.Background(), sessionToken); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	if _, ok := svc.AuthenticateSession(context.Background(), sessionToken); ok {
		t.Fatal("expected session to be gone after logout")
	}
}

func TestNewOpaqueTokenIsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		raw, hash, err := newOpaqueToken(tokenBytes)
		if err != nil {
			t.Fatalf("newOpaqueToken returned error: %v", err)
		}
		if raw == "" || hash == "" {
			t.Fatal("expected non-empty raw token and hash")
		}
		if seen[raw] {
			t.Fatalf("duplicate raw token generated: %q", raw)
		}
		seen[raw] = true
		if hashToken(raw) != hash {
			t.Fatal("hash mismatch between newOpaqueToken and hashToken")
		}
	}
}

// extractToken pulls the ?token= value out of a magic-link URL emitted by
// RequestLogin, mirroring what a browser following the emailed link would
// send back to GET /auth/verify.
func extractToken(t *testing.T, link string) string {
	t.Helper()
	idx := strings.Index(link, "token=")
	if idx == -1 {
		t.Fatalf("link has no token query param: %q", link)
	}
	return link[idx+len("token="):]
}
