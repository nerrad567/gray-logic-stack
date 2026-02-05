package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RoomAccessRepository defines the interface for user room access persistence.
type RoomAccessRepository interface {
	SetRoomAccess(ctx context.Context, userID string, grants []RoomAccessGrant, createdBy string) error
	GetRoomAccess(ctx context.Context, userID string) ([]RoomAccess, error)
	GetAccessibleRoomIDs(ctx context.Context, userID string) ([]string, error)
	GetSceneManageRoomIDs(ctx context.Context, userID string) ([]string, error)
	ClearRoomAccess(ctx context.Context, userID string) error
	ResolveRoomScope(ctx context.Context, userID string) (*RoomScope, error)
}

// RoomAccessGrant is the input for setting room access (used by API handlers).
type RoomAccessGrant struct {
	RoomID          string `json:"room_id"`
	CanManageScenes bool   `json:"can_manage_scenes"`
}

// SQLiteRoomAccessRepository implements RoomAccessRepository using SQLite.
type SQLiteRoomAccessRepository struct {
	db *sql.DB
}

// NewRoomAccessRepository creates a new SQLite-backed room access repository.
func NewRoomAccessRepository(db *sql.DB) *SQLiteRoomAccessRepository {
	return &SQLiteRoomAccessRepository{db: db}
}

// SetRoomAccess replaces all room access grants for a user.
// Pass an empty slice to revoke all room access (user becomes locked out).
func (r *SQLiteRoomAccessRepository) SetRoomAccess(ctx context.Context, userID string, grants []RoomAccessGrant, createdBy string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	if _, err := tx.ExecContext(ctx, "DELETE FROM user_room_access WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("clearing room access: %w", err)
	}

	for _, g := range grants {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO user_room_access (user_id, room_id, can_manage_scenes, created_by) VALUES (?, ?, ?, ?)",
			userID, g.RoomID, boolToInt(g.CanManageScenes), nullString(createdBy)); err != nil {
			return fmt.Errorf("granting room %s: %w", g.RoomID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing room access: %w", err)
	}
	return nil
}

// GetRoomAccess returns all room access grants for a user.
func (r *SQLiteRoomAccessRepository) GetRoomAccess(ctx context.Context, userID string) ([]RoomAccess, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, room_id, can_manage_scenes, created_by, created_at
		 FROM user_room_access WHERE user_id = ? ORDER BY room_id`, userID)
	if err != nil {
		return nil, fmt.Errorf("getting room access: %w", err)
	}
	defer rows.Close()

	var access []RoomAccess
	for rows.Next() {
		var ra RoomAccess
		var canManage int
		var createdBy sql.NullString
		var createdAt string

		if err := rows.Scan(&ra.UserID, &ra.RoomID, &canManage, &createdBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning room access: %w", err)
		}

		ra.CanManageScenes = canManage != 0
		if createdBy.Valid {
			ra.CreatedBy = createdBy.String
		}
		ra.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled

		access = append(access, ra)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating room access: %w", err)
	}

	if access == nil {
		access = []RoomAccess{}
	}
	return access, nil
}

// GetAccessibleRoomIDs returns just the room IDs a user can access.
//
//nolint:dupl // structurally similar to GetSceneManageRoomIDs
func (r *SQLiteRoomAccessRepository) GetAccessibleRoomIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT room_id FROM user_room_access WHERE user_id = ? ORDER BY room_id", userID)
	if err != nil {
		return nil, fmt.Errorf("getting accessible rooms: %w", err)
	}
	defer rows.Close()

	var roomIDs []string
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, fmt.Errorf("scanning room ID: %w", err)
		}
		roomIDs = append(roomIDs, roomID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating room IDs: %w", err)
	}

	if roomIDs == nil {
		roomIDs = []string{}
	}
	return roomIDs, nil
}

// GetSceneManageRoomIDs returns room IDs where the user can manage scenes.
//
//nolint:dupl // structurally similar to GetAccessibleRoomIDs
func (r *SQLiteRoomAccessRepository) GetSceneManageRoomIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT room_id FROM user_room_access WHERE user_id = ? AND can_manage_scenes = 1 ORDER BY room_id", userID)
	if err != nil {
		return nil, fmt.Errorf("getting scene-manage rooms: %w", err)
	}
	defer rows.Close()

	var roomIDs []string
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, fmt.Errorf("scanning room ID: %w", err)
		}
		roomIDs = append(roomIDs, roomID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating room IDs: %w", err)
	}

	if roomIDs == nil {
		roomIDs = []string{}
	}
	return roomIDs, nil
}

// ClearRoomAccess removes all room access for a user (locks them out).
func (r *SQLiteRoomAccessRepository) ClearRoomAccess(ctx context.Context, userID string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM user_room_access WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("clearing room access: %w", err)
	}
	return nil
}

// ResolveRoomScope builds a RoomScope for a user by querying their room access grants.
// Returns a RoomScope with the accessible room IDs and scene-manage room IDs.
// For users with no grants, returns an empty RoomScope (no access).
func (r *SQLiteRoomAccessRepository) ResolveRoomScope(ctx context.Context, userID string) (*RoomScope, error) {
	access, err := r.GetRoomAccess(ctx, userID)
	if err != nil {
		return nil, err
	}

	scope := &RoomScope{}
	for _, ra := range access {
		scope.RoomIDs = append(scope.RoomIDs, ra.RoomID)
		if ra.CanManageScenes {
			scope.SceneManageRoomIDs = append(scope.SceneManageRoomIDs, ra.RoomID)
		}
	}

	if scope.RoomIDs == nil {
		scope.RoomIDs = []string{}
	}
	if scope.SceneManageRoomIDs == nil {
		scope.SceneManageRoomIDs = []string{}
	}

	return scope, nil
}
