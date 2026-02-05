package knx

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupRecorderDB creates an in-memory SQLite database with the required tables.
func setupRecorderDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS knx_group_addresses (
			group_address TEXT PRIMARY KEY,
			last_seen INTEGER NOT NULL,
			message_count INTEGER NOT NULL DEFAULT 1,
			has_read_response INTEGER NOT NULL DEFAULT 0,
			last_health_check INTEGER DEFAULT NULL
		) STRICT;

		CREATE INDEX IF NOT EXISTS idx_knx_group_addresses_health
			ON knx_group_addresses(has_read_response DESC, last_health_check ASC, last_seen DESC);

		CREATE TABLE knx_devices (
			individual_address TEXT PRIMARY KEY,
			last_seen INTEGER NOT NULL,
			message_count INTEGER NOT NULL DEFAULT 1
		) STRICT;

		CREATE INDEX idx_knx_devices_last_seen ON knx_devices(last_seen DESC);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func TestGARecorder_StartStop(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	// Start should succeed.
	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Double-start should be idempotent (no error).
	if err := rec.Start(); err != nil {
		t.Fatalf("second Start() error: %v", err)
	}

	// Stop should not panic.
	rec.Stop()

	// Double-stop should not panic.
	rec.Stop()
}

func TestGARecorder_RecordTelegram(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer rec.Stop()

	ctx := context.Background()

	// Record a telegram with a source device and GA.
	rec.RecordTelegram("1.1.5", "1/2/3", false)

	// Verify GA was recorded.
	gaCount, err := rec.GroupAddressCount(ctx)
	if err != nil {
		t.Fatalf("GroupAddressCount() error: %v", err)
	}
	if gaCount != 1 {
		t.Errorf("GroupAddressCount() = %d, want 1", gaCount)
	}

	// Verify device was recorded.
	devCount, err := rec.DeviceCount(ctx)
	if err != nil {
		t.Fatalf("DeviceCount() error: %v", err)
	}
	if devCount != 1 {
		t.Errorf("DeviceCount() = %d, want 1", devCount)
	}

	// Record same GA again — count should still be 1 (upsert).
	rec.RecordTelegram("1.1.5", "1/2/3", false)

	gaCount, err = rec.GroupAddressCount(ctx)
	if err != nil {
		t.Fatalf("GroupAddressCount() error: %v", err)
	}
	if gaCount != 1 {
		t.Errorf("GroupAddressCount() after duplicate = %d, want 1", gaCount)
	}

	// Verify message_count was incremented.
	var msgCount int
	err = db.QueryRow(`SELECT message_count FROM knx_group_addresses WHERE group_address = ?`, "1/2/3").Scan(&msgCount)
	if err != nil {
		t.Fatalf("querying message_count: %v", err)
	}
	if msgCount != 2 {
		t.Errorf("message_count = %d, want 2", msgCount)
	}
}

func TestGARecorder_RecordTelegram_SkipsBroadcast(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer rec.Stop()

	ctx := context.Background()

	// Source "0.0.0" should be skipped (broadcast/invalid).
	rec.RecordTelegram("0.0.0", "1/2/3", false)

	devCount, err := rec.DeviceCount(ctx)
	if err != nil {
		t.Fatalf("DeviceCount() error: %v", err)
	}
	if devCount != 0 {
		t.Errorf("DeviceCount() = %d, want 0 (broadcast should be skipped)", devCount)
	}

	// GA should still be recorded.
	gaCount, err := rec.GroupAddressCount(ctx)
	if err != nil {
		t.Fatalf("GroupAddressCount() error: %v", err)
	}
	if gaCount != 1 {
		t.Errorf("GroupAddressCount() = %d, want 1", gaCount)
	}
}

func TestGARecorder_RecordTelegram_ReadResponse(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer rec.Stop()

	// Record a read response.
	rec.RecordTelegram("1.1.5", "1/2/3", true)

	var hasReadResponse int
	err := db.QueryRow(`SELECT has_read_response FROM knx_group_addresses WHERE group_address = ?`, "1/2/3").Scan(&hasReadResponse)
	if err != nil {
		t.Fatalf("querying has_read_response: %v", err)
	}
	if hasReadResponse != 1 {
		t.Errorf("has_read_response = %d, want 1", hasReadResponse)
	}

	// Subsequent non-response should NOT downgrade has_read_response (MAX behaviour).
	rec.RecordTelegram("1.1.5", "1/2/3", false)

	err = db.QueryRow(`SELECT has_read_response FROM knx_group_addresses WHERE group_address = ?`, "1/2/3").Scan(&hasReadResponse)
	if err != nil {
		t.Fatalf("querying has_read_response after non-response: %v", err)
	}
	if hasReadResponse != 1 {
		t.Errorf("has_read_response after non-response = %d, want 1 (MAX should preserve)", hasReadResponse)
	}
}

func TestGARecorder_GetHealthCheckGroupAddresses(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer rec.Stop()

	ctx := context.Background()

	// Empty database should return empty list.
	addrs, err := rec.GetHealthCheckGroupAddresses(ctx, 5)
	if err != nil {
		t.Fatalf("GetHealthCheckGroupAddresses() error: %v", err)
	}
	if len(addrs) != 0 {
		t.Errorf("GetHealthCheckGroupAddresses() returned %d, want 0", len(addrs))
	}

	// Record some GAs — one with read response, one without.
	rec.RecordTelegram("1.1.1", "1/0/1", true)  // has read response
	rec.RecordTelegram("1.1.2", "2/0/1", false) // no read response

	addrs, err = rec.GetHealthCheckGroupAddresses(ctx, 5)
	if err != nil {
		t.Fatalf("GetHealthCheckGroupAddresses() error: %v", err)
	}
	if len(addrs) != 2 {
		t.Fatalf("GetHealthCheckGroupAddresses() returned %d, want 2", len(addrs))
	}

	// Verified responder (has_read_response=1) should come first.
	if addrs[0] != "1/0/1" {
		t.Errorf("first health check address = %q, want %q (verified responder should be first)", addrs[0], "1/0/1")
	}

	// Limit should be respected.
	addrs, err = rec.GetHealthCheckGroupAddresses(ctx, 1)
	if err != nil {
		t.Fatalf("GetHealthCheckGroupAddresses(limit=1) error: %v", err)
	}
	if len(addrs) != 1 {
		t.Errorf("GetHealthCheckGroupAddresses(limit=1) returned %d, want 1", len(addrs))
	}
}

func TestGARecorder_MarkHealthCheckUsed(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer rec.Stop()

	ctx := context.Background()

	// Record and mark a GA as used for health check.
	rec.RecordTelegram("1.1.1", "1/0/1", true)

	if err := rec.MarkHealthCheckUsed(ctx, "1/0/1"); err != nil {
		t.Fatalf("MarkHealthCheckUsed() error: %v", err)
	}

	// Verify last_health_check was set (not NULL).
	var lastCheck sql.NullInt64
	err := db.QueryRow(`SELECT last_health_check FROM knx_group_addresses WHERE group_address = ?`, "1/0/1").Scan(&lastCheck)
	if err != nil {
		t.Fatalf("querying last_health_check: %v", err)
	}
	if !lastCheck.Valid {
		t.Error("last_health_check should not be NULL after MarkHealthCheckUsed")
	}
}

func TestGARecorder_RecordAfterStop(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	if err := rec.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	rec.Stop()

	// Recording after stop should not panic (silently ignored).
	rec.RecordTelegram("1.1.5", "1/2/3", false)

	ctx := context.Background()
	gaCount, err := rec.GroupAddressCount(ctx)
	if err != nil {
		t.Fatalf("GroupAddressCount() error: %v", err)
	}
	if gaCount != 0 {
		t.Errorf("GroupAddressCount() = %d, want 0 (should be ignored after stop)", gaCount)
	}
}

func TestGARecorder_RecordBeforeStart(t *testing.T) {
	db := setupRecorderDB(t)
	rec := NewGARecorder(db)

	// Recording before start should not panic (silently ignored).
	rec.RecordTelegram("1.1.5", "1/2/3", false)

	ctx := context.Background()
	gaCount, err := rec.GroupAddressCount(ctx)
	if err != nil {
		t.Fatalf("GroupAddressCount() error: %v", err)
	}
	if gaCount != 0 {
		t.Errorf("GroupAddressCount() = %d, want 0 (should be ignored before start)", gaCount)
	}
}
