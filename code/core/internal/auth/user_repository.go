package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user account persistence.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// SQLiteUserRepository implements UserRepository using SQLite.
type SQLiteUserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new SQLite-backed user repository.
func NewUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

// Create inserts a new user account. The ID is generated if empty.
func (r *SQLiteUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = "usr-" + uuid.NewString()[:8]
	}

	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt, _ = time.Parse(time.RFC3339, now) //nolint:errcheck // format is controlled
	user.UpdatedAt = user.CreatedAt

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, display_name, email, password_hash, role, is_active, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.DisplayName, nullString(user.Email),
		user.PasswordHash, string(user.Role), boolToInt(user.IsActive),
		nullString(user.CreatedBy), now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUsernameExists
		}
		return fmt.Errorf("creating user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their unique ID.
func (r *SQLiteUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	return r.getUser(ctx, "SELECT id, username, display_name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users WHERE id = ?", id)
}

// GetByUsername retrieves a user by their username.
func (r *SQLiteUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	return r.getUser(ctx, "SELECT id, username, display_name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users WHERE username = ?", username)
}

// List returns all users ordered by creation date.
func (r *SQLiteUserRepository) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, username, display_name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users ORDER BY created_at ASC")
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating users: %w", err)
	}

	if users == nil {
		users = []User{}
	}
	return users, nil
}

// Update modifies a user's mutable fields (display_name, email, role, is_active).
func (r *SQLiteUserRepository) Update(ctx context.Context, user *User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, now) //nolint:errcheck // format is controlled

	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, email = ?, role = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		user.DisplayName, nullString(user.Email), string(user.Role), boolToInt(user.IsActive), now, user.ID,
	)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	rows, _ := result.RowsAffected() //nolint:errcheck // always succeeds on SQLite
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdatePassword changes a user's password hash.
func (r *SQLiteUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, now, id,
	)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	rows, _ := result.RowsAffected() //nolint:errcheck // always succeeds on SQLite
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Delete removes a user account by ID.
func (r *SQLiteUserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	rows, _ := result.RowsAffected() //nolint:errcheck // always succeeds on SQLite
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Count returns the total number of user accounts.
func (r *SQLiteUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

// getUser executes a query and scans a single user result.
func (r *SQLiteUserRepository) getUser(ctx context.Context, query string, args ...any) (*User, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	u, err := scanUserRow(row)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// scanner is an interface for sql.Row and sql.Rows Scan methods.
type scanner interface {
	Scan(dest ...any) error
}

// scanUser scans a user from sql.Rows.
func scanUser(rows *sql.Rows) (*User, error) {
	return scanUserFrom(rows)
}

// scanUserRow scans a user from sql.Row.
func scanUserRow(row *sql.Row) (*User, error) {
	return scanUserFrom(row)
}

// scanUserFrom scans a user from any scanner (Row or Rows).
func scanUserFrom(s scanner) (*User, error) {
	var u User
	var email, createdBy sql.NullString
	var role string
	var isActive int
	var createdAt, updatedAt string

	err := s.Scan(&u.ID, &u.Username, &u.DisplayName, &email,
		&u.PasswordHash, &role, &isActive, &createdBy,
		&createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("scanning user: %w", err)
	}

	u.Role = Role(role)
	u.IsActive = isActive != 0
	if email.Valid {
		u.Email = email.String
	}
	if createdBy.Valid {
		u.CreatedBy = createdBy.String
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt) //nolint:errcheck // format is controlled

	return &u, nil
}

// Helper functions.

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isUniqueViolation checks if a SQLite error is a UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") ||
		contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
