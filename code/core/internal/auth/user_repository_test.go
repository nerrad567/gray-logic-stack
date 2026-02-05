package auth

import (
	"context"
	"errors"
	"testing"
)

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := HashPassword("password123")
	user := &User{
		Username:     "testuser",
		DisplayName:  "Test User",
		Email:        "test@example.com",
		PasswordHash: hash,
		Role:         RoleUser,
		IsActive:     true,
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.ID == "" {
		t.Fatal("Create() should generate an ID")
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Username != "testuser" {
		t.Errorf("Username = %q, want %q", got.Username, "testuser")
	}
	if got.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test User")
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}
	if got.Role != RoleUser {
		t.Errorf("Role = %q, want %q", got.Role, RoleUser)
	}
	if !got.IsActive {
		t.Error("IsActive should be true")
	}
	if got.PasswordHash == "" {
		t.Error("PasswordHash should be populated")
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := HashPassword("password123")
	user := &User{
		Username:     "admin",
		DisplayName:  "Admin",
		PasswordHash: hash,
		Role:         RoleAdmin,
		IsActive:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %q, want %q", got.ID, user.ID)
	}
}

func TestUserRepository_GetByUsername_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	_, err := repo.GetByUsername(context.Background(), "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepository_DuplicateUsername(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := HashPassword("password123")
	user1 := &User{
		Username:     "duplicate",
		DisplayName:  "User 1",
		PasswordHash: hash,
		Role:         RoleUser,
		IsActive:     true,
	}
	if err := repo.Create(ctx, user1); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	user2 := &User{
		Username:     "duplicate",
		DisplayName:  "User 2",
		PasswordHash: hash,
		Role:         RoleUser,
		IsActive:     true,
	}
	err := repo.Create(ctx, user2)
	if !errors.Is(err, ErrUsernameExists) {
		t.Errorf("error = %v, want ErrUsernameExists", err)
	}
}

func TestUserRepository_List(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Empty list
	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 0 {
		t.Errorf("List() should return empty, got %d", len(users))
	}

	// Add users
	hash, _ := HashPassword("password123")
	for _, name := range []string{"alice", "bob", "charlie"} {
		u := &User{Username: name, DisplayName: name, PasswordHash: hash, Role: RoleUser, IsActive: true}
		if err := repo.Create(ctx, u); err != nil { //nolint:govet // shadow: err re-declared in test loop
			t.Fatalf("Create(%s) error = %v", name, err)
		}
	}

	users, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 3 {
		t.Errorf("List() returned %d users, want 3", len(users))
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := HashPassword("password123")
	user := &User{
		Username:     "updateme",
		DisplayName:  "Original",
		PasswordHash: hash,
		Role:         RoleUser,
		IsActive:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	user.DisplayName = "Updated"
	user.Role = RoleAdmin
	user.IsActive = false

	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, user.ID)
	if got.DisplayName != "Updated" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Updated")
	}
	if got.Role != RoleAdmin {
		t.Errorf("Role = %q, want %q", got.Role, RoleAdmin)
	}
	if got.IsActive {
		t.Error("IsActive should be false after update")
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := HashPassword("old-password")
	user := &User{
		Username:     "passchange",
		DisplayName:  "Pass Change",
		PasswordHash: hash,
		Role:         RoleUser,
		IsActive:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	newHash, _ := HashPassword("new-password")
	if err := repo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, user.ID)
	ok, _ := VerifyPassword("new-password", got.PasswordHash)
	if !ok {
		t.Error("new password should verify after UpdatePassword")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	hash, _ := HashPassword("password123")
	user := &User{
		Username:     "deleteme",
		DisplayName:  "Delete Me",
		PasswordHash: hash,
		Role:         RoleUser,
		IsActive:     true,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.GetByID(ctx, user.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("after delete, GetByID error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepository_Count(t *testing.T) {
	db := testDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	hash, _ := HashPassword("password123")
	for _, name := range []string{"one", "two"} {
		u := &User{Username: name, DisplayName: name, PasswordHash: hash, Role: RoleUser, IsActive: true}
		repo.Create(ctx, u) //nolint:errcheck // test setup
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}
