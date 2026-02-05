package auth

import (
	"context"
	"log/slog"
	"testing"
)

func TestSeedOwner_CreatesOnEmptyDB(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	logger := slog.Default()
	ctx := context.Background()

	password, err := SeedOwner(ctx, userRepo, logger)
	if err != nil {
		t.Fatalf("SeedOwner() error = %v", err)
	}

	if password == "" {
		t.Fatal("SeedOwner() should return generated password")
	}

	// Verify owner was created
	owner, err := userRepo.GetByUsername(ctx, "owner")
	if err != nil {
		t.Fatalf("GetByUsername(owner) error = %v", err)
	}

	if owner.Role != RoleOwner {
		t.Errorf("Role = %q, want %q", owner.Role, RoleOwner)
	}

	if !owner.IsActive {
		t.Error("seed owner should be active")
	}

	// Verify password works
	ok, err := VerifyPassword(password, owner.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Error("generated password should verify against stored hash")
	}
}

func TestSeedOwner_SkipsWhenUsersExist(t *testing.T) {
	db := testDB(t)
	userRepo := NewUserRepository(db)
	logger := slog.Default()
	ctx := context.Background()

	// Create an existing user first
	seedTestUser(t, db, "existing", RoleAdmin)

	password, err := SeedOwner(ctx, userRepo, logger)
	if err != nil {
		t.Fatalf("SeedOwner() error = %v", err)
	}

	if password != "" {
		t.Error("SeedOwner() should return empty password when users exist")
	}

	// Should still only have the one user
	count, _ := userRepo.Count(ctx)
	if count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}
}

func TestSeedOwner_UniquePasswords(t *testing.T) {
	db1 := testDB(t)
	db2 := testDB(t)
	logger := slog.Default()
	ctx := context.Background()

	pw1, _ := SeedOwner(ctx, NewUserRepository(db1), logger)
	pw2, _ := SeedOwner(ctx, NewUserRepository(db2), logger)

	if pw1 == pw2 {
		t.Error("seed passwords should be unique across instances")
	}
}
