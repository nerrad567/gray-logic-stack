package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Database configuration constants.
const (
	// dirPermissions is the permission mode for the database directory.
	dirPermissions = 0750

	// filePermissions is the permission mode for the database file.
	filePermissions = 0600

	// msPerSecond converts seconds to milliseconds.
	msPerSecond = 1000

	// connectionTimeout is the timeout for verifying database connectivity.
	connectionTimeout = 5 * time.Second

	// connMaxIdleTime is how long idle connections are kept open.
	connMaxIdleTime = 30 * time.Minute
)

// DB wraps a sql.DB connection with Gray Logic-specific functionality.
// It provides migration support, health checks, and proper lifecycle management.
type DB struct {
	*sql.DB
	path string
}

// Config contains database configuration options.
// These map to the database section of config.yaml.
type Config struct {
	// Path is the filesystem path to the SQLite database file.
	// The directory will be created if it doesn't exist.
	Path string

	// WALMode enables Write-Ahead Logging for better concurrent access.
	// Recommended: true (allows concurrent reads during writes).
	WALMode bool

	// BusyTimeout is the maximum time to wait for a database lock (seconds).
	// Prevents "database is locked" errors under contention.
	BusyTimeout int
}

// Open creates a new database connection with the specified configuration.
//
// It performs the following setup:
//  1. Creates the database directory if it doesn't exist
//  2. Opens the database file (creates if not present)
//  3. Configures WAL mode and busy timeout
//  4. Sets appropriate file permissions (0600)
//  5. Verifies the connection with a ping
//
// Parameters:
//   - cfg: Database configuration
//
// Returns:
//   - *DB: Connected database wrapper
//   - error: If connection or configuration fails
func Open(cfg Config) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	// Build connection string with pragmas
	// See: https://github.com/mattn/go-sqlite3#connection-string
	connStr := fmt.Sprintf("file:%s?_busy_timeout=%d&_foreign_keys=on",
		cfg.Path,
		cfg.BusyTimeout*msPerSecond,
	)

	// Add WAL mode if enabled
	if cfg.WALMode {
		connStr += "&_journal_mode=WAL&_synchronous=NORMAL"
	}

	// Open database
	sqlDB, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure connection pool
	// SQLite works best with a single writer, but multiple readers
	sqlDB.SetMaxOpenConns(1)            // SQLite only supports one writer
	sqlDB.SetMaxIdleConns(1)            // Keep one connection ready
	sqlDB.SetConnMaxLifetime(time.Hour) // Refresh connections hourly
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	db := &DB{
		DB:   sqlDB,
		path: cfg.Path,
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		sqlDB.Close() //nolint:errcheck // Best effort cleanup on error path
		return nil, fmt.Errorf("verifying database connection: %w", err)
	}

	// Set file permissions (owner read/write only)
	// Ignore error - file might not exist yet on first run, will be set after first write
	_ = os.Chmod(cfg.Path, filePermissions) //nolint:errcheck // Intentional: first run creates file later

	return db, nil
}

// Close closes the database connection gracefully.
// It should be called when the application shuts down.
//
// Returns:
//   - error: If closing fails
func (db *DB) Close() error {
	if db.DB == nil {
		return nil
	}
	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}
	return nil
}

// Path returns the filesystem path to the database file.
func (db *DB) Path() string {
	return db.path
}

// HealthCheck verifies the database is accessible and functioning.
// It performs a simple query to ensure the connection is alive.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: nil if healthy, error describing the issue otherwise
func (db *DB) HealthCheck(ctx context.Context) error {
	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

// Stats returns database connection pool statistics.
// Useful for monitoring and debugging connection issues.
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// ExecContext executes a query that doesn't return rows (INSERT, UPDATE, DELETE).
// This is a convenience wrapper that provides consistent error handling.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//   - query: SQL query with ? placeholders
//   - args: Arguments for placeholders
//
// Returns:
//   - sql.Result: Contains LastInsertId and RowsAffected
//   - error: If execution fails
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	return result, nil
}

// QueryRowContext executes a query that returns at most one row.
// This is a convenience wrapper for single-row queries.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//   - query: SQL query with ? placeholders
//   - args: Arguments for placeholders
//
// Returns:
//   - *sql.Row: Row to scan results from
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a new transaction with the given options.
// Always use transactions for operations that modify multiple rows/tables.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//   - opts: Transaction options (nil for defaults)
//
// Returns:
//   - *sql.Tx: Transaction to execute queries on
//   - error: If starting transaction fails
//
// Example:
//
//	tx, err := db.BeginTx(ctx, nil)
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback() // No-op if committed
//
//	// ... execute queries on tx ...
//
//	return tx.Commit()
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	return tx, nil
}
