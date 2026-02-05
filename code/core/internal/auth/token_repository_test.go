package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTokenRepository_CreateAndGetByID(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "tokenuser", RoleUser)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	token := &RefreshToken{
		UserID:     user.ID,
		TokenHash:  HashToken("raw-refresh-token"),
		DeviceInfo: "Chrome on macOS",
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}

	if err := repo.Create(ctx, token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if token.ID == "" {
		t.Fatal("Create() should generate an ID")
	}
	if token.FamilyID == "" {
		t.Fatal("Create() should generate a FamilyID")
	}

	got, err := repo.GetByID(ctx, token.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", got.UserID, user.ID)
	}
	if got.DeviceInfo != "Chrome on macOS" {
		t.Errorf("DeviceInfo = %q, want %q", got.DeviceInfo, "Chrome on macOS")
	}
	if got.Revoked {
		t.Error("new token should not be revoked")
	}
}

func TestTokenRepository_Revoke(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "revokeuser", RoleUser)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	token := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("revoke-me"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	repo.Create(ctx, token) //nolint:errcheck // test setup

	if err := repo.Revoke(ctx, token.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, token.ID)
	if !got.Revoked {
		t.Error("token should be revoked after Revoke()")
	}
}

func TestTokenRepository_RevokeFamily(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "familyuser", RoleUser)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	// Create two tokens in the same family
	familyID := "test-family-001"
	t1 := &RefreshToken{
		UserID:    user.ID,
		FamilyID:  familyID,
		TokenHash: HashToken("family-token-1"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	t2 := &RefreshToken{
		UserID:    user.ID,
		FamilyID:  familyID,
		TokenHash: HashToken("family-token-2"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	// And one in a different family
	t3 := &RefreshToken{
		UserID:    user.ID,
		FamilyID:  "other-family",
		TokenHash: HashToken("other-token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	repo.Create(ctx, t1) //nolint:errcheck // test setup
	repo.Create(ctx, t2) //nolint:errcheck // test setup
	repo.Create(ctx, t3) //nolint:errcheck // test setup

	if err := repo.RevokeFamily(ctx, familyID); err != nil {
		t.Fatalf("RevokeFamily() error = %v", err)
	}

	got1, _ := repo.GetByID(ctx, t1.ID)
	got2, _ := repo.GetByID(ctx, t2.ID)
	got3, _ := repo.GetByID(ctx, t3.ID)

	if !got1.Revoked {
		t.Error("t1 should be revoked (same family)")
	}
	if !got2.Revoked {
		t.Error("t2 should be revoked (same family)")
	}
	if got3.Revoked {
		t.Error("t3 should NOT be revoked (different family)")
	}
}

func TestTokenRepository_RevokeAllForUser(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "revokeall", RoleUser)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	for i := range 3 {
		tk := &RefreshToken{
			UserID:    user.ID,
			TokenHash: HashToken("token-" + string(rune('a'+i))),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		repo.Create(ctx, tk) //nolint:errcheck // test setup
	}

	if err := repo.RevokeAllForUser(ctx, user.ID); err != nil {
		t.Fatalf("RevokeAllForUser() error = %v", err)
	}

	active, _ := repo.ListActiveByUser(ctx, user.ID)
	if len(active) != 0 {
		t.Errorf("ListActiveByUser() returned %d, want 0 after RevokeAll", len(active))
	}
}

func TestTokenRepository_ListActiveByUser(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "listactive", RoleUser)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	// Active token
	active := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("active-token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	repo.Create(ctx, active) //nolint:errcheck // test setup

	// Expired token
	expired := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("expired-token"),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	repo.Create(ctx, expired) //nolint:errcheck // test setup

	// Revoked token
	revoked := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("revoked-token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	repo.Create(ctx, revoked)    //nolint:errcheck // test setup
	repo.Revoke(ctx, revoked.ID) //nolint:errcheck // test setup

	tokens, err := repo.ListActiveByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListActiveByUser() error = %v", err)
	}

	if len(tokens) != 1 {
		t.Errorf("ListActiveByUser() returned %d, want 1", len(tokens))
	}
	if len(tokens) > 0 && tokens[0].ID != active.ID {
		t.Errorf("active token ID = %q, want %q", tokens[0].ID, active.ID)
	}
}

func TestTokenRepository_DeleteExpired(t *testing.T) {
	db := testDB(t)
	user := seedTestUser(t, db, "cleanup", RoleUser)
	repo := NewTokenRepository(db)
	ctx := context.Background()

	// Expired token
	expired := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("old-token"),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	repo.Create(ctx, expired) //nolint:errcheck // test setup

	// Active token
	active := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken("new-token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	repo.Create(ctx, active) //nolint:errcheck // test setup

	count, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}
	if count != 1 {
		t.Errorf("DeleteExpired() deleted %d, want 1", count)
	}

	// Active token should still exist
	_, err = repo.GetByID(ctx, active.ID)
	if err != nil {
		t.Errorf("active token should still exist after cleanup, got error: %v", err)
	}

	// Expired token should be gone
	_, err = repo.GetByID(ctx, expired.ID)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("expired token should be deleted, got error: %v", err)
	}
}

func TestHashToken(t *testing.T) {
	hash1 := HashToken("raw-token")
	hash2 := HashToken("raw-token")
	hash3 := HashToken("different-token")

	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different input should produce different hash")
	}
	if len(hash1) != 64 { //nolint:mnd // SHA-256 hex = 64 characters
		t.Errorf("hash length = %d, want 64 (SHA-256 hex)", len(hash1))
	}
}
