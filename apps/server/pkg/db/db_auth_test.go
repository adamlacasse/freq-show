package db

import (
	"context"
	"testing"
	"time"
)

// TestMemoryStoreGetOrCreateUserByEmailIsIdempotent covers the "creating the
// account on first login" contract shared by every AuthRepository
// implementation: the same address always resolves to the same user id,
// and matching is case-insensitive.
func TestMemoryStoreGetOrCreateUserByEmailIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	first, err := store.GetOrCreateUserByEmail(ctx, "User@Example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUserByEmail returned error: %v", err)
	}
	if first.ID == "" {
		t.Fatal("expected a non-empty user id")
	}
	if first.Email != "user@example.com" {
		t.Fatalf("expected normalized lowercase email, got %q", first.Email)
	}

	second, err := store.GetOrCreateUserByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("second GetOrCreateUserByEmail returned error: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected the same user id on repeat login, got %q and %q", first.ID, second.ID)
	}

	other, err := store.GetOrCreateUserByEmail(ctx, "other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUserByEmail returned error: %v", err)
	}
	if other.ID == first.ID {
		t.Fatal("expected a different user id for a different email")
	}
}

func TestMemoryStoreGetOrCreateUserByEmailRequiresEmail(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	if _, err := store.GetOrCreateUserByEmail(ctx, "   "); err == nil {
		t.Fatal("expected error for blank email")
	}
}

func TestMemoryStoreLoginTokenLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	now := time.Now()
	const hash = "token-hash-1"
	if err := store.SaveLoginToken(ctx, hash, "User@Example.com", now.Add(15*time.Minute)); err != nil {
		t.Fatalf("SaveLoginToken returned error: %v", err)
	}

	email, ok, err := store.ConsumeLoginToken(ctx, hash, now)
	if err != nil {
		t.Fatalf("ConsumeLoginToken returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected token to be valid on first consume")
	}
	if email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", email)
	}

	// Single-use: a second consume of the same token must fail.
	if _, ok, err := store.ConsumeLoginToken(ctx, hash, now); err != nil {
		t.Fatalf("second ConsumeLoginToken returned error: %v", err)
	} else if ok {
		t.Fatal("expected token reuse to be rejected")
	}
}

func TestMemoryStoreConsumeLoginTokenRejectsExpired(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	now := time.Now()
	const hash = "token-hash-expired"
	if err := store.SaveLoginToken(ctx, hash, "user@example.com", now.Add(-time.Minute)); err != nil {
		t.Fatalf("SaveLoginToken returned error: %v", err)
	}

	if _, ok, err := store.ConsumeLoginToken(ctx, hash, now); err != nil {
		t.Fatalf("ConsumeLoginToken returned error: %v", err)
	} else if ok {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestMemoryStoreConsumeLoginTokenRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	if _, ok, err := store.ConsumeLoginToken(ctx, "no-such-hash", time.Now()); err != nil {
		t.Fatalf("ConsumeLoginToken returned error: %v", err)
	} else if ok {
		t.Fatal("expected unknown token to be rejected")
	}
}

func TestMemoryStoreSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	now := time.Now()
	const hash = "session-hash-1"
	const userID = "user-1"

	if err := store.CreateSession(ctx, hash, userID, now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	session, err := store.GetSession(ctx, hash, now)
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if session == nil {
		t.Fatal("expected session to be found")
	}
	if session.UserID != userID {
		t.Fatalf("expected user id %q, got %q", userID, session.UserID)
	}

	if err := store.DeleteSession(ctx, hash); err != nil {
		t.Fatalf("DeleteSession returned error: %v", err)
	}

	afterDelete, err := store.GetSession(ctx, hash, now)
	if err != nil {
		t.Fatalf("GetSession after delete returned error: %v", err)
	}
	if afterDelete != nil {
		t.Fatalf("expected nil session after delete, got %#v", afterDelete)
	}
}

func TestMemoryStoreGetSessionRejectsExpired(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	now := time.Now()
	const hash = "session-hash-expired"
	if err := store.CreateSession(ctx, hash, "user-1", now.Add(-time.Minute)); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	session, err := store.GetSession(ctx, hash, now)
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if session != nil {
		t.Fatalf("expected nil for expired session, got %#v", session)
	}
}

func TestMemoryStoreGetSessionMissing(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	session, err := store.GetSession(ctx, "no-such-hash", time.Now())
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if session != nil {
		t.Fatalf("expected nil for missing session, got %#v", session)
	}
}

func TestMemoryStorePruneExpiredAuth(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	now := time.Now()

	// A consumed token, an expired-but-unconsumed token, and a still-valid
	// token: only the first two should be pruned.
	if err := store.SaveLoginToken(ctx, "consumed", "user@example.com", now.Add(time.Hour)); err != nil {
		t.Fatalf("SaveLoginToken(consumed) returned error: %v", err)
	}
	if _, _, err := store.ConsumeLoginToken(ctx, "consumed", now); err != nil {
		t.Fatalf("ConsumeLoginToken returned error: %v", err)
	}
	if err := store.SaveLoginToken(ctx, "expired", "user@example.com", now.Add(-time.Minute)); err != nil {
		t.Fatalf("SaveLoginToken(expired) returned error: %v", err)
	}
	if err := store.SaveLoginToken(ctx, "valid", "user@example.com", now.Add(time.Hour)); err != nil {
		t.Fatalf("SaveLoginToken(valid) returned error: %v", err)
	}

	// An expired session and a still-valid session: only the first should
	// be pruned.
	if err := store.CreateSession(ctx, "expired-session", "user-1", now.Add(-time.Minute)); err != nil {
		t.Fatalf("CreateSession(expired) returned error: %v", err)
	}
	if err := store.CreateSession(ctx, "valid-session", "user-1", now.Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession(valid) returned error: %v", err)
	}

	deleted, err := store.PruneExpiredAuth(ctx, now)
	if err != nil {
		t.Fatalf("PruneExpiredAuth returned error: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("expected 3 rows pruned, got %d", deleted)
	}

	if _, ok, _ := store.ConsumeLoginToken(ctx, "expired", now); ok {
		t.Fatal("expected expired token to have been pruned")
	}
	if _, ok, _ := store.ConsumeLoginToken(ctx, "valid", now); !ok {
		t.Fatal("expected valid token to survive pruning")
	}

	if session, _ := store.GetSession(ctx, "expired-session", now); session != nil {
		t.Fatal("expected expired session to have been pruned")
	}
	if session, _ := store.GetSession(ctx, "valid-session", now); session == nil {
		t.Fatal("expected valid session to survive pruning")
	}
}

func TestMemoryStoreSessionValidation(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	if err := store.CreateSession(ctx, "", "user-1", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("expected error for empty token hash")
	}
	if err := store.CreateSession(ctx, "hash", "", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("expected error for empty user id")
	}
}

func TestMemoryStoreGetUser(t *testing.T) {
	ctx := context.Background()
	store, err := NewMemoryStore(ctx)
	if err != nil {
		t.Fatalf(newStoreErrFmt, err)
	}

	created, err := store.GetOrCreateUserByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUserByEmail returned error: %v", err)
	}

	found, err := store.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if found == nil || found.ID != created.ID || found.Email != "user@example.com" {
		t.Fatalf("expected user %#v, got %#v", created, found)
	}

	missing, err := store.GetUser(ctx, "nonexistent-id")
	if err != nil {
		t.Fatalf("GetUser(missing) returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing user, got %#v", missing)
	}

	empty, err := store.GetUser(ctx, "")
	if err != nil {
		t.Fatalf("GetUser(\"\") returned error: %v", err)
	}
	if empty != nil {
		t.Fatalf("expected nil for empty user id, got %#v", empty)
	}
}
