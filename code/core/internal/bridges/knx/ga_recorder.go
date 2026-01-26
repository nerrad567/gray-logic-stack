package knx

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// GARecorder passively records group addresses and device individual addresses
// seen on the KNX bus. It is called by the Bridge whenever a telegram is received,
// building a database of known addresses over time.
//
// This enables health checks to use discovered addresses without manual
// configuration - the recorder implements GroupAddressProvider and DeviceProvider
// for the knxd Manager.
//
// Thread Safety: All methods are safe for concurrent use.
type GARecorder struct {
	db     *sql.DB
	logger Logger

	// Prepared statements for upserts (created once, reused)
	gaUpsertStmt     *sql.Stmt
	deviceUpsertStmt *sql.Stmt
	stmtMu           sync.Mutex

	// Shutdown coordination
	closed bool
	mu     sync.RWMutex
}

// NewGARecorder creates a new recorder for group addresses and devices.
// The database must have the knx_group_addresses and knx_devices tables created.
func NewGARecorder(db *sql.DB) *GARecorder {
	return &GARecorder{
		db: db,
	}
}

// SetLogger sets the logger for the recorder.
func (r *GARecorder) SetLogger(logger Logger) {
	r.logger = logger
}

// Start prepares the recorder for use.
// Must be called before RecordTelegram.
func (r *GARecorder) Start() error {
	r.stmtMu.Lock()
	defer r.stmtMu.Unlock()

	if r.gaUpsertStmt != nil {
		return nil // Already started
	}

	// Prepare GA upsert statement
	gaStmt, err := r.db.Prepare(`
		INSERT INTO knx_group_addresses (group_address, last_seen, message_count, has_read_response)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(group_address) DO UPDATE SET
			last_seen = excluded.last_seen,
			message_count = message_count + 1,
			has_read_response = MAX(has_read_response, excluded.has_read_response)
	`)
	if err != nil {
		return fmt.Errorf("preparing GA upsert statement: %w", err)
	}

	// Prepare device upsert statement
	deviceStmt, err := r.db.Prepare(`
		INSERT INTO knx_devices (individual_address, last_seen, message_count)
		VALUES (?, ?, 1)
		ON CONFLICT(individual_address) DO UPDATE SET
			last_seen = excluded.last_seen,
			message_count = message_count + 1
	`)
	if err != nil {
		gaStmt.Close()
		return fmt.Errorf("preparing device upsert statement: %w", err)
	}

	r.gaUpsertStmt = gaStmt
	r.deviceUpsertStmt = deviceStmt
	r.log("GA recorder started")
	return nil
}

// Stop closes the recorder and releases resources.
func (r *GARecorder) Stop() {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()

	r.stmtMu.Lock()
	defer r.stmtMu.Unlock()

	if r.gaUpsertStmt != nil {
		r.gaUpsertStmt.Close()
		r.gaUpsertStmt = nil
	}
	if r.deviceUpsertStmt != nil {
		r.deviceUpsertStmt.Close()
		r.deviceUpsertStmt = nil
	}

	r.log("GA recorder stopped")
}

// RecordTelegram records the source device and destination GA from a telegram.
// Called by the Bridge for every received telegram.
//
// Parameters:
//   - source: Source individual address (e.g., "1.1.5") - the sending device
//   - ga: Destination group address (e.g., "1/2/3")
//   - isResponse: True if this was a GroupValue_Response (APCI 0x40)
func (r *GARecorder) RecordTelegram(source, ga string, isResponse bool) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return
	}
	r.mu.RUnlock()

	r.stmtMu.Lock()
	gaStmt := r.gaUpsertStmt
	deviceStmt := r.deviceUpsertStmt
	r.stmtMu.Unlock()

	if gaStmt == nil || deviceStmt == nil {
		return // Not started
	}

	now := time.Now().Unix()

	// Record source device (skip 0.0.0 which is invalid/broadcast)
	if source != "" && source != "0.0.0" {
		if _, err := deviceStmt.Exec(source, now); err != nil {
			r.logError("recording device", err)
		}
	}

	// Record destination GA
	hasResponse := 0
	if isResponse {
		hasResponse = 1
	}
	if _, err := gaStmt.Exec(ga, now, hasResponse); err != nil {
		r.logError("recording GA", err)
	}
}

// RecordGA records a group address seen on the bus (legacy method).
// Prefer RecordTelegram which also records the source device.
func (r *GARecorder) RecordGA(ga string, isResponse bool) {
	r.RecordTelegram("", ga, isResponse)
}

// GetHealthCheckGroupAddresses returns group addresses for Layer 3 health checks.
// The selection strategy cycles through addresses to:
//   - Spread load across multiple devices (don't always hit the same one)
//   - Discover read capabilities on more addresses over time
//
// Priority order:
//  1. Addresses with has_read_response=1, least recently checked first
//  2. Addresses with has_read_response=0, least recently checked first (discovery)
//
// Implements knxd.GroupAddressProvider interface.
func (r *GARecorder) GetHealthCheckGroupAddresses(ctx context.Context, limit int) ([]string, error) {
	// Query cycles through GAs: verified responders first, then discovery candidates
	// SQLite sorts NULL before other values in ASC order by default
	rows, err := r.db.QueryContext(ctx, `
		SELECT group_address FROM knx_group_addresses
		ORDER BY has_read_response DESC, last_health_check ASC, last_seen DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, rows.Err()
}

// MarkHealthCheckUsed records that a GA was just used for a health check.
// This enables cycling through addresses instead of always using the same one.
func (r *GARecorder) MarkHealthCheckUsed(ctx context.Context, ga string) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `
		UPDATE knx_group_addresses SET last_health_check = ? WHERE group_address = ?
	`, now, ga)
	return err
}

// GroupAddressCount returns the number of discovered group addresses.
func (r *GARecorder) GroupAddressCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knx_group_addresses`).Scan(&count)
	return count, err
}

// DeviceCount returns the number of discovered devices.
func (r *GARecorder) DeviceCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knx_devices`).Scan(&count)
	return count, err
}

// log logs an info message if logger is set.
func (r *GARecorder) log(msg string, keysAndValues ...any) {
	if r.logger != nil {
		r.logger.Info(msg, keysAndValues...)
	}
}

// logError logs an error if logger is set.
func (r *GARecorder) logError(msg string, err error) {
	if r.logger != nil {
		r.logger.Error(msg, "error", err)
	}
}
