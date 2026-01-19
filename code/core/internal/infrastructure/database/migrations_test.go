package database

import (
	"context"
	"embed"
	"testing"
	"time"
)

// testMigrationsDir is the directory containing test migration files.
const testMigrationsDir = "testdata"

//go:embed testdata/*.sql
var testMigrationsFS embed.FS

// TestMigrate verifies migration application.
func TestMigrate(t *testing.T) {
	// Save original values
	origFS := MigrationsFS
	origDir := MigrationsDir
	defer func() {
		MigrationsFS = origFS
		MigrationsDir = origDir
	}()

	// Use test migrations
	MigrationsFS = testMigrationsFS
	MigrationsDir = testMigrationsDir

	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Verify table was created
	var tableName string
	err := db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='test_users'",
	).Scan(&tableName)
	if err != nil {
		t.Fatalf("table test_users not created: %v", err)
	}

	// Verify migration was recorded
	applied, pending, err := db.GetMigrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if len(applied) != 1 {
		t.Errorf("expected 1 applied migration, got %d", len(applied))
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending migrations, got %d", len(pending))
	}

	// Running again should be idempotent
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
}

// TestMigrateDown verifies migration rollback.
func TestMigrateDown(t *testing.T) {
	// Save original values
	origFS := MigrationsFS
	origDir := MigrationsDir
	defer func() {
		MigrationsFS = origFS
		MigrationsDir = origDir
	}()

	// Use test migrations
	MigrationsFS = testMigrationsFS
	MigrationsDir = testMigrationsDir

	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Apply migrations first
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Rollback
	if err := db.MigrateDown(ctx); err != nil {
		t.Fatalf("MigrateDown() error = %v", err)
	}

	// Verify table was dropped
	var count int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_users'",
	).Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 0 {
		t.Error("table test_users should have been dropped")
	}

	// Verify migration record removed
	applied, _, err := db.GetMigrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 applied migrations after rollback, got %d", len(applied))
	}
}

// TestMigrateNoMigrations verifies behaviour with no migrations.
func TestMigrateNoMigrations(t *testing.T) {
	// Save original values
	origFS := MigrationsFS
	origDir := MigrationsDir
	defer func() {
		MigrationsFS = origFS
		MigrationsDir = origDir
	}()

	// Use empty FS
	var emptyFS embed.FS
	MigrationsFS = emptyFS
	MigrationsDir = "."

	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()

	// Should succeed with no migrations
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() with no migrations error = %v", err)
	}
}

// TestGetMigrationStatus verifies status reporting.
func TestGetMigrationStatus(t *testing.T) {
	// Save original values
	origFS := MigrationsFS
	origDir := MigrationsDir
	defer func() {
		MigrationsFS = origFS
		MigrationsDir = origDir
	}()

	// Use test migrations
	MigrationsFS = testMigrationsFS
	MigrationsDir = testMigrationsDir

	db := openTestDB(t)
	defer db.Close() //nolint:errcheck // Test cleanup

	ctx := context.Background()

	// Create migrations table
	if err := db.createMigrationsTable(ctx); err != nil {
		t.Fatalf("createMigrationsTable() error = %v", err)
	}

	// Before migration: should show pending
	applied, pending, err := db.GetMigrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(applied))
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}
}

// TestParseMigrationFilename verifies filename parsing.
func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantVersion string
		wantIsUp    bool
		wantOk      bool
	}{
		{
			name:        "valid up migration",
			filename:    "20260118_120000_create_users.up.sql",
			wantVersion: "20260118_120000",
			wantIsUp:    true,
			wantOk:      true,
		},
		{
			name:        "valid down migration",
			filename:    "20260118_120000_create_users.down.sql",
			wantVersion: "20260118_120000",
			wantIsUp:    false,
			wantOk:      true,
		},
		{
			name:     "not sql file",
			filename: "readme.txt",
			wantOk:   false,
		},
		{
			name:     "missing direction",
			filename: "20260118_120000_create_users.sql",
			wantOk:   false,
		},
		{
			name:     "invalid format",
			filename: "invalid.up.sql",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, isUp, ok := parseMigrationFilename(tt.filename)
			if ok != tt.wantOk {
				t.Errorf("ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if version != tt.wantVersion {
					t.Errorf("version = %v, want %v", version, tt.wantVersion)
				}
				if isUp != tt.wantIsUp {
					t.Errorf("isUp = %v, want %v", isUp, tt.wantIsUp)
				}
			}
		})
	}
}

// TestExtractMigrationName verifies name extraction.
func TestExtractMigrationName(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"20260118_120000_create_users.up.sql", "create_users"},
		{"20260118_120000_initial_schema.down.sql", "initial_schema"},
		{"20260118_120000_add_email_to_users.up.sql", "add_email_to_users"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := extractMigrationName(tt.filename)
			if got != tt.want {
				t.Errorf("extractMigrationName(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
