package device

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupStateHistoryTestDB creates an in-memory SQLite database with the state_history table.
func setupStateHistoryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE state_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
			state TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'mqtt',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		) STRICT;
		CREATE INDEX idx_state_history_device ON state_history(device_id, created_at DESC);
		CREATE INDEX idx_state_history_time ON state_history(created_at DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// insertStateHistoryRow inserts a state history row with a specific timestamp.
func insertStateHistoryRow(t *testing.T, db *sql.DB, deviceID, stateJSON, source string, createdAt time.Time) {
	t.Helper()

	_, err := db.Exec(
		"INSERT INTO state_history (device_id, state, source, created_at) VALUES (?, ?, ?, ?)",
		deviceID,
		stateJSON,
		source,
		createdAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("failed to insert state history row: %v", err)
	}
}

// TestRecordStateChange verifies state history writes and retrieval.
func TestRecordStateChange(t *testing.T) {
	db := setupStateHistoryTestDB(t)
	repo := NewSQLiteStateHistoryRepository(db)
	ctx := context.Background()

	state := State{"on": true, "level": 75}
	if err := repo.RecordStateChange(ctx, "dev-1", state, StateHistorySourceMQTT); err != nil {
		t.Fatalf("RecordStateChange() error = %v", err)
	}

	entries, err := repo.GetHistory(ctx, "dev-1", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.DeviceID != "dev-1" {
		t.Errorf("DeviceID = %q, want %q", entry.DeviceID, "dev-1")
	}
	if entry.Source != StateHistorySourceMQTT {
		t.Errorf("Source = %q, want %q", entry.Source, StateHistorySourceMQTT)
	}
	if entry.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want non-zero")
	}
	if on, ok := entry.State["on"].(bool); !ok || !on {
		t.Errorf("State[\"on\"] = %v, want true", entry.State["on"])
	}
	if level, ok := entry.State["level"].(float64); !ok || level != 75 {
		t.Errorf("State[\"level\"] = %v, want 75", entry.State["level"])
	}
}

// TestGetHistory verifies ordering and limit enforcement.
func TestGetHistory(t *testing.T) {
	db := setupStateHistoryTestDB(t)
	repo := NewSQLiteStateHistoryRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	insertStateHistoryRow(t, db, "dev-1", `{"on":false}`, StateHistorySourceCommand, now.Add(-2*time.Hour))
	insertStateHistoryRow(t, db, "dev-1", `{"on":true}`, StateHistorySourceMQTT, now.Add(-1*time.Hour))
	insertStateHistoryRow(t, db, "dev-1", `{"on":true}`, StateHistorySourceScene, now)
	insertStateHistoryRow(t, db, "dev-2", `{"on":true}`, StateHistorySourceMQTT, now)

	entries, err := repo.GetHistory(ctx, "dev-1", 2)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries length = %d, want 2", len(entries))
	}

	if !entries[0].CreatedAt.Equal(now) {
		t.Errorf("entry[0] CreatedAt = %s, want %s", entries[0].CreatedAt, now)
	}
	if !entries[1].CreatedAt.Equal(now.Add(-1 * time.Hour)) {
		t.Errorf("entry[1] CreatedAt = %s, want %s", entries[1].CreatedAt, now.Add(-1*time.Hour))
	}
}

// TestPruneHistory verifies old entries are removed.
func TestPruneHistory(t *testing.T) {
	db := setupStateHistoryTestDB(t)
	repo := NewSQLiteStateHistoryRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	insertStateHistoryRow(t, db, "dev-1", `{"on":true}`, StateHistorySourceMQTT, now.Add(-40*24*time.Hour))
	insertStateHistoryRow(t, db, "dev-1", `{"on":false}`, StateHistorySourceMQTT, now.Add(-12*time.Hour))

	deleted, err := repo.PruneHistory(ctx, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("PruneHistory() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	entries, err := repo.GetHistory(ctx, "dev-1", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	if !entries[0].CreatedAt.Equal(now.Add(-12 * time.Hour)) {
		t.Errorf("remaining CreatedAt = %s, want %s", entries[0].CreatedAt, now.Add(-12*time.Hour))
	}
}
