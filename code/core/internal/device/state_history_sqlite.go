package device

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
)

// SQLiteStateHistoryRepository implements StateHistoryRepository using SQLite.
//
// It stores state snapshots as JSON in the state_history table.
type SQLiteStateHistoryRepository struct {
	db *sql.DB
}

// NewSQLiteStateHistoryRepository creates a new SQLite state history repository.
//
// Parameters:
//   - db: Open SQLite connection used for queries
//
// Returns:
//   - *SQLiteStateHistoryRepository: Repository instance ready for use
func NewSQLiteStateHistoryRepository(db *sql.DB) *SQLiteStateHistoryRepository {
	return &SQLiteStateHistoryRepository{db: db}
}

// RecordStateChange inserts a new state history entry for a device.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceID: Unique device identifier
//   - state: State snapshot to persist
//   - source: Origin of the change (mqtt, command, scene)
//
// Returns:
//   - error: nil on success, otherwise the underlying database error
func (r *SQLiteStateHistoryRepository) RecordStateChange(ctx context.Context, deviceID string, state State, source string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}
	if source == "" {
		source = StateHistorySourceMQTT
	}
	if state == nil {
		state = State{}
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshalling state: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		"INSERT INTO state_history (device_id, state, source) VALUES (?, ?, ?)",
		deviceID,
		string(stateJSON),
		source,
	)
	if err != nil {
		return fmt.Errorf("inserting state history: %w", err)
	}

	return nil
}

// GetHistory returns recent state history entries for a device, ordered newest first.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - deviceID: Unique device identifier
//   - limit: Maximum entries to return (default 50, max 200)
//
// Returns:
//   - []StateHistoryEntry: History entries ordered by created_at DESC
//   - error: nil on success, otherwise the underlying query error
func (r *SQLiteStateHistoryRepository) GetHistory(ctx context.Context, deviceID string, limit int) ([]StateHistoryEntry, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, device_id, state, source, created_at
		 FROM state_history
		 WHERE device_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		deviceID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying state history: %w", err)
	}
	defer rows.Close()

	entries := make([]StateHistoryEntry, 0, limit)
	for rows.Next() {
		var entry StateHistoryEntry
		var stateJSON string
		var createdAt string

		if err := rows.Scan(&entry.ID, &entry.DeviceID, &stateJSON, &entry.Source, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning state history: %w", err)
		}

		if err := json.Unmarshal([]byte(stateJSON), &entry.State); err != nil {
			return nil, fmt.Errorf("unmarshalling state: %w", err)
		}

		timestamp, err := parseHistoryTimestamp(createdAt)
		if err != nil {
			return nil, err
		}
		entry.CreatedAt = timestamp

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating state history: %w", err)
	}

	return entries, nil
}

// PruneHistory deletes history entries older than the given duration.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - olderThan: Duration to retain (entries older than now-olderThan are deleted)
//
// Returns:
//   - int64: Number of rows deleted
//   - error: nil on success, otherwise the underlying database error
func (r *SQLiteStateHistoryRepository) PruneHistory(ctx context.Context, olderThan time.Duration) (int64, error) {
	if olderThan <= 0 {
		return 0, fmt.Errorf("olderThan must be positive")
	}

	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM state_history WHERE created_at < ?",
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("deleting state history: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("checking rows affected: %w", err)
	}

	return rowsAffected, nil
}

// parseHistoryTimestamp parses a timestamp stored in SQLite.
func parseHistoryTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("created_at is empty")
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return timestamp, nil
	}

	fallback, fallbackErr := time.Parse("2006-01-02T15:04:05Z", value)
	if fallbackErr == nil {
		return fallback, nil
	}

	return time.Time{}, fmt.Errorf("parsing created_at: %w", err)
}
