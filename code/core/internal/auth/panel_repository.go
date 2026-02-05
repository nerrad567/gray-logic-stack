package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PanelRepository defines the interface for panel device identity persistence.
type PanelRepository interface {
	Create(ctx context.Context, panel *Panel) error
	GetByID(ctx context.Context, id string) (*Panel, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*Panel, error)
	List(ctx context.Context) ([]Panel, error)
	UpdateName(ctx context.Context, id, name string) error
	Delete(ctx context.Context, id string) error
	UpdateLastSeen(ctx context.Context, id string) error
	SetRooms(ctx context.Context, panelID string, roomIDs []string) error
	GetRoomIDs(ctx context.Context, panelID string) ([]string, error)
}

// SQLitePanelRepository implements PanelRepository using SQLite.
type SQLitePanelRepository struct {
	db *sql.DB
}

// NewPanelRepository creates a new SQLite-backed panel repository.
func NewPanelRepository(db *sql.DB) *SQLitePanelRepository {
	// Ensure in-memory SQLite uses a single connection to avoid per-connection schemas in tests.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return &SQLitePanelRepository{db: db}
}

// Create inserts a new panel device identity. The ID is generated if empty.
func (r *SQLitePanelRepository) Create(ctx context.Context, panel *Panel) error {
	if panel.ID == "" {
		panel.ID = "pnl-" + uuid.NewString()[:16]
	}

	now := time.Now().UTC().Format(time.RFC3339)
	panel.CreatedAt, _ = time.Parse(time.RFC3339, now) //nolint:errcheck // format is controlled

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO panels (id, name, token_hash, is_active, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		panel.ID, panel.Name, panel.TokenHash,
		boolToInt(panel.IsActive), nullString(panel.CreatedBy), now,
	)
	if err != nil {
		return fmt.Errorf("creating panel: %w", err)
	}

	return nil
}

// GetByID retrieves a panel by its unique ID.
func (r *SQLitePanelRepository) GetByID(ctx context.Context, id string) (*Panel, error) {
	return r.scanPanel(r.db.QueryRowContext(ctx,
		`SELECT id, name, token_hash, is_active, last_seen_at, created_by, created_at
		 FROM panels WHERE id = ?`, id))
}

// GetByTokenHash retrieves a panel by its token hash (used during authentication).
func (r *SQLitePanelRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*Panel, error) {
	return r.scanPanel(r.db.QueryRowContext(ctx,
		`SELECT id, name, token_hash, is_active, last_seen_at, created_by, created_at
		 FROM panels WHERE token_hash = ?`, tokenHash))
}

// List returns all registered panels.
func (r *SQLitePanelRepository) List(ctx context.Context) ([]Panel, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, token_hash, is_active, last_seen_at, created_by, created_at
		 FROM panels ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing panels: %w", err)
	}
	defer rows.Close()

	var panels []Panel
	for rows.Next() {
		var p Panel
		var lastSeen, createdBy sql.NullString
		var isActive int
		var createdAt string

		if err := rows.Scan(&p.ID, &p.Name, &p.TokenHash, &isActive,
			&lastSeen, &createdBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning panel: %w", err)
		}

		p.IsActive = isActive != 0
		if lastSeen.Valid {
			t, _ := time.Parse(time.RFC3339, lastSeen.String) //nolint:errcheck // format is controlled
			p.LastSeenAt = &t
		}
		if createdBy.Valid {
			p.CreatedBy = createdBy.String
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled

		panels = append(panels, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating panels: %w", err)
	}

	if panels == nil {
		panels = []Panel{}
	}
	return panels, nil
}

// UpdateName changes a panel's display name.
func (r *SQLitePanelRepository) UpdateName(ctx context.Context, id, name string) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE panels SET name = ? WHERE id = ?", name, id)
	if err != nil {
		return fmt.Errorf("updating panel name: %w", err)
	}

	rows, _ := result.RowsAffected() //nolint:errcheck // always succeeds on SQLite
	if rows == 0 {
		return ErrPanelNotFound
	}
	return nil
}

// Delete removes a panel by ID. Room assignments are cascade-deleted.
func (r *SQLitePanelRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM panels WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting panel: %w", err)
	}

	rows, _ := result.RowsAffected() //nolint:errcheck // always succeeds on SQLite
	if rows == 0 {
		return ErrPanelNotFound
	}
	return nil
}

// UpdateLastSeen updates the panel's last_seen_at timestamp to now.
func (r *SQLitePanelRepository) UpdateLastSeen(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx,
		"UPDATE panels SET last_seen_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("updating last seen: %w", err)
	}
	return nil
}

// SetRooms replaces all room assignments for a panel. Pass an empty slice to remove all.
func (r *SQLitePanelRepository) SetRooms(ctx context.Context, panelID string, roomIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is no-op after commit

	if _, err := tx.ExecContext(ctx, "DELETE FROM panel_room_access WHERE panel_id = ?", panelID); err != nil {
		return fmt.Errorf("clearing panel rooms: %w", err)
	}

	for _, roomID := range roomIDs {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO panel_room_access (panel_id, room_id) VALUES (?, ?)",
			panelID, roomID); err != nil {
			return fmt.Errorf("assigning room %s to panel: %w", roomID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing panel rooms: %w", err)
	}
	return nil
}

// GetRoomIDs returns the room IDs assigned to a panel.
//
//nolint:dupl // structurally similar to room_access queries
func (r *SQLitePanelRepository) GetRoomIDs(ctx context.Context, panelID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT room_id FROM panel_room_access WHERE panel_id = ? ORDER BY room_id", panelID)
	if err != nil {
		return nil, fmt.Errorf("getting panel rooms: %w", err)
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

// scanPanel scans a panel from a single row query.
func (r *SQLitePanelRepository) scanPanel(row *sql.Row) (*Panel, error) {
	var p Panel
	var lastSeen, createdBy sql.NullString
	var isActive int
	var createdAt string

	err := row.Scan(&p.ID, &p.Name, &p.TokenHash, &isActive,
		&lastSeen, &createdBy, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPanelNotFound
		}
		return nil, fmt.Errorf("scanning panel: %w", err)
	}

	p.IsActive = isActive != 0
	if lastSeen.Valid {
		t, _ := time.Parse(time.RFC3339, lastSeen.String) //nolint:errcheck // format is controlled
		p.LastSeenAt = &t
	}
	if createdBy.Valid {
		p.CreatedBy = createdBy.String
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt) //nolint:errcheck // format is controlled

	return &p, nil
}
