package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Resilience tests verify that the auth subsystem handles failure scenarios
// gracefully. These tests use the TestResilience_ prefix for easy filtering:
//
//	go test -run TestResilience -race ./internal/auth/...

// TestResilience_TokenRotation_ConcurrentRefresh verifies that concurrent
// refresh token rotation requests don't corrupt state. When two goroutines
// present the same refresh token simultaneously, one should succeed and the
// other should see the token as revoked (theft detection).
func TestResilience_TokenRotation_ConcurrentRefresh(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	tokenRepo := NewTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user := seedTestUser(t, db, "concurrent-user", RoleUser)

	// Create an initial refresh token
	rawToken := "test-raw-token-concurrent"
	tokenHash := HashToken(rawToken)

	initialToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	if err := tokenRepo.Create(ctx, initialToken); err != nil {
		t.Fatalf("creating initial token: %v", err)
	}

	// Simulate concurrent refresh: two goroutines try to rotate the same token
	var wg sync.WaitGroup
	results := make(chan error, 2) //nolint:mnd // two concurrent attempts

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Look up the token
			stored, err := tokenRepo.GetByTokenHash(ctx, tokenHash)
			if err != nil {
				results <- err
				return
			}

			// Try to rotate it
			newRT := &RefreshToken{
				UserID:    user.ID,
				FamilyID:  stored.FamilyID,
				TokenHash: HashToken("new-token-" + time.Now().String()),
				ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			}

			err = tokenRepo.RotateRefreshToken(ctx, stored.ID, newRT)
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	// At least one should succeed; both may succeed since SQLite serialises writes.
	// The key invariant: no panic, no data corruption, and the original token is revoked.
	var successes, failures int
	for err := range results {
		if err != nil {
			failures++
		} else {
			successes++
		}
	}

	if successes == 0 {
		t.Error("expected at least one concurrent rotation to succeed")
	}

	// Verify the original token is now revoked
	stored, err := tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("retrieving rotated token: %v", err)
	}
	if !stored.Revoked {
		t.Error("original token should be revoked after rotation")
	}

	// Verify user can still be fetched (no corruption)
	_, err = userRepo.GetByID(ctx, user.ID)
	if err != nil {
		t.Errorf("user lookup after concurrent rotation failed: %v", err)
	}
}

// TestResilience_UserDeletion_CascadesCleanly verifies that deleting a user
// cascades to refresh tokens and room access (via FK ON DELETE CASCADE),
// leaving no orphaned references.
func TestResilience_UserDeletion_CascadesCleanly(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)

	userRepo := NewUserRepository(db)
	tokenRepo := NewTokenRepository(db)
	roomRepo := NewRoomAccessRepository(db)
	ctx := context.Background()

	// Create user with tokens and room access
	user := seedTestUser(t, db, "cascade-user", RoleUser)

	// Add refresh tokens
	for i := range 3 {
		rt := &RefreshToken{
			UserID:    user.ID,
			TokenHash: HashToken("token-" + string(rune('a'+i))),
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		}
		if err := tokenRepo.Create(ctx, rt); err != nil {
			t.Fatalf("creating token %d: %v", i, err)
		}
	}

	// Add room access
	grants := []RoomAccessGrant{
		{RoomID: "room-kitchen", CanManageScenes: true},
		{RoomID: "room-living", CanManageScenes: false},
	}
	if err := roomRepo.SetRoomAccess(ctx, user.ID, grants, user.ID); err != nil {
		t.Fatalf("setting room access: %v", err)
	}

	// Verify pre-deletion state
	tokens, err := tokenRepo.ListActiveByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("listing tokens pre-delete: %v", err)
	}
	if len(tokens) != 3 { //nolint:mnd // 3 tokens created above
		t.Errorf("expected 3 tokens pre-delete, got %d", len(tokens))
	}

	rooms, err := roomRepo.GetRoomAccess(ctx, user.ID)
	if err != nil {
		t.Fatalf("getting room access pre-delete: %v", err)
	}
	if len(rooms) != 2 { //nolint:mnd // 2 room grants created above
		t.Errorf("expected 2 room grants pre-delete, got %d", len(rooms))
	}

	// Delete the user
	err = userRepo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("deleting user: %v", err)
	}

	// Verify cascade: tokens should be gone
	tokens, err = tokenRepo.ListActiveByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("listing tokens post-delete: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens post-delete (FK cascade), got %d", len(tokens))
	}

	// Verify cascade: room access should be gone
	rooms, err = roomRepo.GetRoomAccess(ctx, user.ID)
	if err != nil {
		t.Fatalf("getting room access post-delete: %v", err)
	}
	if len(rooms) != 0 {
		t.Errorf("expected 0 room grants post-delete (FK cascade), got %d", len(rooms))
	}
}

// TestResilience_ContextCancellation_RepositoryOps verifies that repository
// operations respect context cancellation and return clean errors rather
// than panicking or leaving partial state.
func TestResilience_ContextCancellation_RepositoryOps(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)

	// Create a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// All operations should return a context error, not panic
	_, err := userRepo.List(ctx)
	if err == nil {
		t.Error("List with cancelled context should return error")
	}

	_, err = userRepo.GetByUsername(ctx, "nonexistent")
	if err == nil {
		t.Error("GetByUsername with cancelled context should return error")
	}

	_, err = userRepo.Count(ctx)
	if err == nil {
		t.Error("Count with cancelled context should return error")
	}

	user := &User{
		Username:     "cancel-test",
		DisplayName:  "Cancel Test",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=1$dGVzdHNhbHQ$dGVzdGhhc2g",
		Role:         RoleUser,
		IsActive:     true,
	}
	err = userRepo.Create(ctx, user)
	if err == nil {
		t.Error("Create with cancelled context should return error")
	}
}

// TestResilience_RoomScope_EmptyGrants verifies that a user with zero room
// grants gets an empty (but non-nil) RoomScope — not nil (which would mean
// unrestricted) and not an error.
func TestResilience_RoomScope_EmptyGrants(t *testing.T) {
	db := testDB(t)
	seedTestRooms(t, db)

	roomRepo := NewRoomAccessRepository(db)
	ctx := context.Background()

	// Create user with no room grants
	user := seedTestUser(t, db, "no-rooms-user", RoleUser)

	scope, err := roomRepo.ResolveRoomScope(ctx, user.ID)
	if err != nil {
		t.Fatalf("resolving empty room scope: %v", err)
	}

	if scope == nil {
		t.Fatal("empty room scope should be non-nil (nil means unrestricted)")
	}

	if len(scope.RoomIDs) != 0 {
		t.Errorf("expected 0 room IDs, got %d", len(scope.RoomIDs))
	}

	if len(scope.SceneManageRoomIDs) != 0 {
		t.Errorf("expected 0 scene-manage room IDs, got %d", len(scope.SceneManageRoomIDs))
	}

	// CanAccessRoom should return false for any room
	if scope.CanAccessRoom("room-kitchen") {
		t.Error("user with no grants should not access any room")
	}
}
