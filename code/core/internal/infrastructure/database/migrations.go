package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration filename parsing constants.
const (
	// migrationFilenameParts is the expected number of parts in a migration filename.
	// Format: YYYYMMDD_HHMMSS_description.up.sql (3 parts when split by "_")
	migrationFilenameParts = 3

	// minVersionParts is the minimum parts needed to extract a version.
	minVersionParts = 2
)

// MigrationsFS should be set by the main package to embed migration files.
// This allows the migrations to be compiled into the binary.
//
// Usage in main.go or a migrations package:
//
//	//go:embed migrations/*.sql
//	var migrationsFS embed.FS
//
//	func init() {
//	    database.MigrationsFS = migrationsFS
//	}
var MigrationsFS embed.FS

// MigrationsDir is the directory within MigrationsFS containing migration files.
// Can be set to "." if files are at the root of the embedded filesystem.
var MigrationsDir = "migrations"

// Migration represents a single database migration.
type Migration struct {
	// Version is the migration version number (extracted from filename).
	// Format: YYYYMMDD_HHMMSS (e.g., 20260118_120000)
	Version string

	// Name is the human-readable migration name.
	Name string

	// UpSQL contains the SQL to apply this migration.
	UpSQL string

	// DownSQL contains the SQL to rollback this migration.
	DownSQL string
}

// MigrationRecord represents a row in the schema_migrations table.
type MigrationRecord struct {
	Version   string
	AppliedAt time.Time
}

// Migrate applies all pending migrations to the database.
// Migrations are applied in version order (oldest first).
//
// # Atomicity
//
// Each migration runs in its own transaction. If migration N fails:
//   - Migrations 1 to N-1 remain committed
//   - Migration N is rolled back
//   - Migrations N+1 onwards are not attempted
//
// This per-migration atomicity is intentional:
//   - Allows partial progress on large migration batches
//   - Matches SQLite's single-writer model
//   - Enables debugging by seeing which migration failed
//   - Re-running Migrate() after fixing the issue continues from N
//
// For all-or-nothing semantics, wrap the call in your own transaction,
// but be aware this may hit SQLite lock timeouts on large migrations.
//
// This method:
//  1. Creates the schema_migrations table if it doesn't exist
//  2. Loads all migration files from MigrationsFS
//  3. Determines which migrations haven't been applied
//  4. Applies pending migrations in order, each in its own transaction
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: If any migration fails (that migration is rolled back)
func (db *DB) Migrate(ctx context.Context) error {
	// Ensure migrations table exists
	if err := db.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Load all migrations
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	if len(migrations) == 0 {
		return nil // No migrations to apply
	}

	// Get applied migrations
	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	// Determine pending migrations
	appliedSet := make(map[string]bool)
	for _, m := range applied {
		appliedSet[m.Version] = true
	}

	var pending []Migration
	for _, m := range migrations {
		if !appliedSet[m.Version] {
			pending = append(pending, m)
		}
	}

	if len(pending) == 0 {
		return nil // All migrations already applied
	}

	// Apply each pending migration
	for _, m := range pending {
		if err := db.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("applying migration %s (%s): %w", m.Version, m.Name, err)
		}
	}

	return nil
}

// MigrateDown rolls back the most recent migration.
// This is primarily for development and testing.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - error: If rollback fails
func (db *DB) MigrateDown(ctx context.Context) error {
	// Get the most recent applied migration
	applied, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return nil // Nothing to rollback
	}

	// Get the most recent
	latest := applied[len(applied)-1]

	// Load migrations to find the down SQL
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}

	var migration *Migration
	for _, m := range migrations {
		if m.Version == latest.Version {
			migration = &m
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration %s not found in filesystem", latest.Version)
	}

	if migration.DownSQL == "" {
		return fmt.Errorf("migration %s has no down SQL", latest.Version)
	}

	// Apply rollback in transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is no-op after commit

	// Execute down SQL
	if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
		return fmt.Errorf("executing down SQL: %w", err)
	}

	// Remove from migrations table
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM schema_migrations WHERE version = ?",
		migration.Version,
	); err != nil {
		return fmt.Errorf("removing migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing rollback: %w", err)
	}
	return nil
}

// GetMigrationStatus returns the current migration status.
// Useful for health checks and debugging.
//
// Parameters:
//   - ctx: Context for timeout/cancellation
//
// Returns:
//   - applied: List of applied migrations
//   - pending: List of pending migrations
//   - error: If status check fails
func (db *DB) GetMigrationStatus(ctx context.Context) (applied []MigrationRecord, pending []Migration, err error) {
	applied, err = db.getAppliedMigrations(ctx)
	if err != nil {
		return nil, nil, err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return nil, nil, err
	}

	appliedSet := make(map[string]bool)
	for _, m := range applied {
		appliedSet[m.Version] = true
	}

	for _, m := range migrations {
		if !appliedSet[m.Version] {
			pending = append(pending, m)
		}
	}

	return applied, pending, nil
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist.
func (db *DB) createMigrationsTable(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`)
	return err
}

// getAppliedMigrations returns all migrations that have been applied.
func (db *DB) getAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	rows, err := db.DB.QueryContext(ctx,
		"SELECT version, applied_at FROM schema_migrations ORDER BY version",
	)
	if err != nil {
		return nil, fmt.Errorf("querying migrations: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var r MigrationRecord
		var appliedAt string
		if err := rows.Scan(&r.Version, &appliedAt); err != nil {
			return nil, fmt.Errorf("scanning migration row: %w", err)
		}
		// Parse timestamp - ignore error as format is controlled by us
		r.AppliedAt, _ = time.Parse(time.RFC3339, appliedAt) //nolint:errcheck // Format is controlled
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating migrations: %w", err)
	}
	return records, nil
}

// applyMigration applies a single migration within a transaction.
func (db *DB) applyMigration(ctx context.Context, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is no-op after commit

	// Execute the up SQL
	if _, err := tx.ExecContext(ctx, m.UpSQL); err != nil {
		return fmt.Errorf("executing SQL: %w", err)
	}

	// Record the migration
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		m.Version,
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration: %w", err)
	}
	return nil
}

// loadMigrations loads all migration files from the embedded filesystem.
func loadMigrations() ([]Migration, error) {
	// Check if MigrationsFS has been set
	var empty embed.FS
	if MigrationsFS == empty {
		return nil, nil // No embedded migrations
	}

	entries, err := fs.ReadDir(MigrationsFS, MigrationsDir)
	if err != nil {
		// Directory might not exist if no migrations
		return nil, nil
	}

	// Categorise migration files by version
	upFiles, downFiles := categoriseMigrationFiles(entries)

	// Build migration list from categorised files
	migrations, err := buildMigrations(upFiles, downFiles)
	if err != nil {
		return nil, err
	}

	// Sort by version (oldest first)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// categoriseMigrationFiles groups migration files by version and direction.
func categoriseMigrationFiles(entries []fs.DirEntry) (upFiles, downFiles map[string]string) {
	upFiles = make(map[string]string)
	downFiles = make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		version, isUp, ok := parseMigrationFilename(name)
		if !ok {
			continue
		}

		if isUp {
			upFiles[version] = name
		} else {
			downFiles[version] = name
		}
	}

	return upFiles, downFiles
}

// parseMigrationFilename extracts version and direction from a migration filename.
// Returns version, isUp (true for .up.sql, false for .down.sql), and ok (true if valid).
func parseMigrationFilename(name string) (version string, isUp bool, ok bool) {
	if !strings.HasSuffix(name, ".sql") {
		return "", false, false
	}

	base := strings.TrimSuffix(name, ".sql")

	switch {
	case strings.HasSuffix(base, ".up"):
		isUp = true
		base = strings.TrimSuffix(base, ".up")
	case strings.HasSuffix(base, ".down"):
		isUp = false
		base = strings.TrimSuffix(base, ".down")
	default:
		return "", false, false
	}

	// Extract version (YYYYMMDD_HHMMSS from YYYYMMDD_HHMMSS_description)
	parts := strings.SplitN(base, "_", migrationFilenameParts)
	if len(parts) < minVersionParts {
		return "", false, false
	}

	version = parts[0] + "_" + parts[1]
	return version, isUp, true
}

// buildMigrations creates Migration structs from categorised files.
func buildMigrations(upFiles, downFiles map[string]string) ([]Migration, error) {
	var migrations []Migration

	for version, upFile := range upFiles {
		m, err := buildMigration(version, upFile, downFiles[version])
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	return migrations, nil
}

// buildMigration creates a single Migration from its files.
func buildMigration(version, upFile, downFile string) (Migration, error) {
	upSQL, err := fs.ReadFile(MigrationsFS, filepath.Join(MigrationsDir, upFile))
	if err != nil {
		return Migration{}, fmt.Errorf("reading %s: %w", upFile, err)
	}

	m := Migration{
		Version: version,
		Name:    extractMigrationName(upFile),
		UpSQL:   string(upSQL),
	}

	if downFile != "" {
		downSQL, err := fs.ReadFile(MigrationsFS, filepath.Join(MigrationsDir, downFile))
		if err != nil {
			return Migration{}, fmt.Errorf("reading %s: %w", downFile, err)
		}
		m.DownSQL = string(downSQL)
	}

	return m, nil
}

// extractMigrationName extracts a human-readable name from the filename.
// Example: "20260118_120000_initial_schema.up.sql" -> "initial_schema"
func extractMigrationName(filename string) string {
	base := strings.TrimSuffix(filename, ".sql")
	base = strings.TrimSuffix(base, ".up")
	base = strings.TrimSuffix(base, ".down")

	parts := strings.SplitN(base, "_", migrationFilenameParts)
	if len(parts) >= migrationFilenameParts {
		return parts[minVersionParts] // The description part
	}
	return base
}
