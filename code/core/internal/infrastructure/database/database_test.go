package database

import (
	"context"
	"os"
	"path/filepath"
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
