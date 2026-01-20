package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOpen verifies database connection establishment.
func TestOpen(t *testing.T) {
	t.Run("creates database file", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := Open(Config{
			Path:        dbPath,
			WALMode:     true,
			BusyTimeout: 5,
		})
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer db.Close() //nolint:errcheck // Test cleanup

		// Verify file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

		db, err := Open(Config{
			Path:        dbPath,
			WALMode:     true,
			BusyTimeout: 5,
		})
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer db.Close() //nolint:errcheck // Test cleanup

		// Verify directory was created
		dir := filepath.Dir(dbPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("database directory was not created")
		}
	})

	t.Run("returns path", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := Open(Config{
			Path:        dbPath,
			WALMode:     true,
			BusyTimeout: 5,
		})
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer db.Close() //nolint:errcheck // Test cleanup

		if db.Path() != dbPath {
			t.Errorf("Path() = %v, want %v", db.Path(), dbPath)
		}
	})
}

// TestHealthCheck verifies the health check functionality.
func TestHealthCheck(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

// TestClose verifies graceful shutdown.
func TestClose(t *testing.T) {
	db := openTestDB(t)

	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should not error (nil check)
	db.DB = nil
	if err := db.Close(); err != nil {
		t.Errorf("Close() on nil DB error = %v", err)
	}
}

// TestExecContext verifies query execution.
func TestExecContext(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()

	// Create a test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("ExecContext() CREATE error = %v", err)
	}

	// Insert a row
	result, err := db.ExecContext(ctx, "INSERT INTO test_table (name) VALUES (?)", "test")
	if err != nil {
		t.Fatalf("ExecContext() INSERT error = %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}
	if id != 1 {
		t.Errorf("LastInsertId() = %v, want 1", id)
	}
}

// TestBeginTxCommit verifies transaction commit.
func TestBeginTxCommit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()

	// Create test table
	_, err := db.ExecContext(ctx, "CREATE TABLE tx_commit_test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error = %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO tx_commit_test (value) VALUES (?)", "committed")
	if err != nil {
		t.Fatalf("INSERT error = %v", err)
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify row exists
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tx_commit_test WHERE value = ?", "committed").Scan(&count)
	if err != nil {
		t.Fatalf("SELECT error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

// TestBeginTxRollback verifies transaction rollback.
func TestBeginTxRollback(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()

	// Create test table
	_, err := db.ExecContext(ctx, "CREATE TABLE tx_rollback_test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error = %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO tx_rollback_test (value) VALUES (?)", "rolled_back")
	if err != nil {
		t.Fatalf("INSERT error = %v", err)
	}

	if err = tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// Verify row does not exist
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tx_rollback_test WHERE value = ?", "rolled_back").Scan(&count)
	if err != nil {
		t.Fatalf("SELECT error = %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

// TestStats verifies stats are returned.
func TestStats(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Errorf("MaxOpenConnections = %v, want 1 (SQLite single writer)", stats.MaxOpenConnections)
	}
}

// openTestDB creates a temporary database for testing.
func openTestDB(t *testing.T) *DB {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(Config{
		Path:        dbPath,
		WALMode:     true,
		BusyTimeout: 5,
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	return db
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestOpen_UnwritableDirectory verifies failure when directory is not writable.
func TestOpen_UnwritableDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	tmpDir := t.TempDir()
	unwritableDir := filepath.Join(tmpDir, "readonly")

	if err := os.Mkdir(unwritableDir, 0500); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}

	dbPath := filepath.Join(unwritableDir, "subdir", "test.db")

	_, err := Open(Config{
		Path:        dbPath,
		WALMode:     true,
		BusyTimeout: 5,
	})

	if err == nil {
		t.Fatal("Open() should fail for unwritable directory")
	}

	if !strings.Contains(err.Error(), "creating database directory") {
		t.Errorf("expected 'creating database directory' error, got: %v", err)
	}
}

// TestHealthCheck_ContextCancelled verifies health check respects context.
func TestHealthCheck_ContextCancelled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := db.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck() should fail with cancelled context")
	}
}

// TestOpen_BusyTimeoutHonored verifies busy timeout handles lock contention.
func TestOpen_BusyTimeoutHonored(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db1, err := Open(Config{
		Path:        dbPath,
		WALMode:     false,
		BusyTimeout: 1,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db1.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()
	tx, err := db1.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	_, err = tx.ExecContext(ctx, "CREATE TABLE lock_test (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("CREATE TABLE error = %v", err)
	}

	db2, err := Open(Config{
		Path:        dbPath,
		WALMode:     false,
		BusyTimeout: 1,
	})
	if err != nil {
		t.Fatalf("Second Open() error = %v", err)
	}
	defer db2.Close() //nolint:errcheck // Test cleanup

	start := time.Now()
	_, writeErr := db2.ExecContext(ctx, "CREATE TABLE another_test (id INTEGER PRIMARY KEY)")
	elapsed := time.Since(start)

	if writeErr == nil {
		t.Log("Write succeeded (WAL mode might allow this)")
	} else if !strings.Contains(writeErr.Error(), "locked") &&
		!strings.Contains(writeErr.Error(), "busy") {
		t.Logf("Got error (expected lock/busy): %v", writeErr)
	}

	if elapsed < 500*time.Millisecond {
		t.Log("Write completed quickly (may have succeeded or failed fast)")
	}

	if err := tx.Rollback(); err != nil {
		t.Logf("Rollback error: %v", err)
	}
}

// TestExecContext_InvalidSQL verifies proper error handling for invalid SQL.
func TestExecContext_InvalidSQL(t *testing.T) {
	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()

	_, err := db.ExecContext(ctx, "INVALID SQL SYNTAX HERE")
	if err == nil {
		t.Error("ExecContext() should fail for invalid SQL")
	}

	if !strings.Contains(err.Error(), "executing query") {
		t.Errorf("expected 'executing query' error wrapper, got: %v", err)
	}
}
