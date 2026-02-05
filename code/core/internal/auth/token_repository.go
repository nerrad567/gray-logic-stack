package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TokenRepository defines the interface for refresh token persistence.
type TokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByID(ctx context.Context, id string) (*RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeFamily(ctx context.Context, familyID string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	RotateRefreshToken(ctx context.Context, oldID string, newToken *RefreshToken) error
	ListActiveByUser(ctx context.Context, userID string) ([]RefreshToken, error)
	DeleteExpired(ctx context.Context) (int64, error)
}

// SQLiteTokenRepository implements TokenRepository using SQLite.
type SQLiteTokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new SQLite-backed token repository.
func NewTokenRepository(db *sql.DB) *SQLiteTokenRepository {
	return &SQLiteTokenRepository{db: db}
}

// HashToken computes the SHA-256 hash of a raw token string for storage.
// Raw tokens are never stored — only their hashes.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// Create inserts a new refresh token. The ID is generated if empty.
func (r *SQLiteTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	if token.ID == "" {
		token.ID = "rt-" + uuid.NewString()[:16]
	}
	if token.FamilyID == "" {
		token.FamilyID = uuid.NewString()
	}

	now := time.Now().UTC().Format(time.RFC3339)
	token.CreatedAt, _ = time.Parse(time.RFC3339, now) //nolint:errcheck // format is controlled

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, family_id, token_hash, device_info, expires_at, revoked, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.UserID, token.FamilyID, token.TokenHash,
		nullString(token.DeviceInfo),
		token.ExpiresAt.UTC().Format(time.RFC3339),
		boolToInt(token.Revoked), now,
	)
	if err != nil {
		return fmt.Errorf("creating refresh token: %w", err)
	}

	return nil
}

// GetByID retrieves a refresh token by its ID.
//
//nolint:dupl // structurally similar to GetByTokenHash
func (r *SQLiteTokenRepository) GetByID(ctx context.Context, id string) (*RefreshToken, error) {
	var t RefreshToken
	var deviceInfo sql.NullString
	var revoked int
	var expiresAt, createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, family_id, token_hash, device_info, expires_at, revoked, created_at
		 FROM refresh_tokens WHERE id = ?`, id,
	).Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &deviceInfo,
		&expiresAt, &revoked, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("getting refresh token: %w", err)
	}

	t.Revoked = revoked != 0
	if deviceInfo.Valid {
		t.DeviceInfo = deviceInfo.String
	}
	t.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt) //nolint:errcheck // format is controlled
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled

	return &t, nil
}

// GetByTokenHash retrieves a refresh token by its SHA-256 hash.
// Used during token refresh/logout when the client sends the raw token.
//
//nolint:dupl // structurally similar to GetByID
func (r *SQLiteTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var t RefreshToken
	var deviceInfo sql.NullString
	var revoked int
	var expiresAt, createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, family_id, token_hash, device_info, expires_at, revoked, created_at
		 FROM refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &deviceInfo,
		&expiresAt, &revoked, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("getting refresh token by hash: %w", err)
	}

	t.Revoked = revoked != 0
	if deviceInfo.Valid {
		t.DeviceInfo = deviceInfo.String
	}
	t.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt) //nolint:errcheck // format is controlled
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled

	return &t, nil
}

// Revoke marks a single refresh token as revoked.
func (r *SQLiteTokenRepository) Revoke(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}
	return nil
}

// RevokeFamily marks all tokens in a family as revoked.
// This is used for theft detection: if a revoked token is reused,
// the entire family is invalidated.
func (r *SQLiteTokenRepository) RevokeFamily(ctx context.Context, familyID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE family_id = ?", familyID)
	if err != nil {
		return fmt.Errorf("revoking token family: %w", err)
	}
	return nil
}

// RevokeAllForUser marks all refresh tokens for a user as revoked.
// Used when changing password or admin force-logout.
func (r *SQLiteTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("revoking all tokens for user: %w", err)
	}
	return nil
}

// RotateRefreshToken atomically revokes the old token and creates a new one
// in the same family. This prevents TOCTOU races during token refresh.
func (r *SQLiteTokenRepository) RotateRefreshToken(ctx context.Context, oldID string, newToken *RefreshToken) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning rotation transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	// Revoke the consumed token
	if _, err := tx.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = 1 WHERE id = ?", oldID); err != nil {
		return fmt.Errorf("revoking old token: %w", err)
	}

	// Insert the new token
	if newToken.ID == "" {
		newToken.ID = "rt-" + uuid.NewString()[:16]
	}
	if newToken.FamilyID == "" {
		newToken.FamilyID = uuid.NewString()
	}

	now := time.Now().UTC().Format(time.RFC3339)
	newToken.CreatedAt, _ = time.Parse(time.RFC3339, now) //nolint:errcheck // format is controlled

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, family_id, token_hash, device_info, expires_at, revoked, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		newToken.ID, newToken.UserID, newToken.FamilyID, newToken.TokenHash,
		nullString(newToken.DeviceInfo),
		newToken.ExpiresAt.UTC().Format(time.RFC3339),
		boolToInt(newToken.Revoked), now,
	); err != nil {
		return fmt.Errorf("creating new token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing rotation: %w", err)
	}
	return nil
}

// ListActiveByUser returns all non-revoked, non-expired tokens for a user.
func (r *SQLiteTokenRepository) ListActiveByUser(ctx context.Context, userID string) ([]RefreshToken, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, family_id, token_hash, device_info, expires_at, revoked, created_at
		 FROM refresh_tokens
		 WHERE user_id = ? AND revoked = 0 AND expires_at > ?
		 ORDER BY created_at DESC`, userID, now)
	if err != nil {
		return nil, fmt.Errorf("listing active tokens: %w", err)
	}
	defer rows.Close()

	var tokens []RefreshToken
	for rows.Next() {
		var t RefreshToken
		var deviceInfo sql.NullString
		var revoked int
		var expiresAt, createdAt string

		if err := rows.Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &deviceInfo,
			&expiresAt, &revoked, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning token: %w", err)
		}

		t.Revoked = revoked != 0
		if deviceInfo.Valid {
			t.DeviceInfo = deviceInfo.String
		}
		t.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt) //nolint:errcheck // format is controlled
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled

		tokens = append(tokens, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tokens: %w", err)
	}

	if tokens == nil {
		tokens = []RefreshToken{}
	}
	return tokens, nil
}

// DeleteExpired removes tokens that have expired, freeing storage.
// Returns the number of deleted rows.
func (r *SQLiteTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := r.db.ExecContext(ctx,
		"DELETE FROM refresh_tokens WHERE expires_at <= ?", now)
	if err != nil {
		return 0, fmt.Errorf("deleting expired tokens: %w", err)
	}

	count, _ := result.RowsAffected() //nolint:errcheck // always succeeds on SQLite
	return count, nil
}
