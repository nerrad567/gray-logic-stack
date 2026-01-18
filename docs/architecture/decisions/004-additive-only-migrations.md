---
title: "ADR-004: Additive-Only Database Migrations"
status: accepted
date: 2026-01-15
last_updated: 2026-01-17
---

# ADR-004: Additive-Only Database Migrations

## Status

Accepted

## Context

Gray Logic is designed for multi-decade deployments with these requirements:
- **Safe rollback** — Ability to revert to previous version if upgrade fails
- **No data loss** — User data must never be accidentally deleted
- **Reliability** — Upgrades should be low-risk operations

Traditional migration approaches allow:
- `CREATE TABLE` / `DROP TABLE`
- `ADD COLUMN` / `DROP COLUMN`
- `RENAME COLUMN`

These destructive operations create rollback problems:

```
Scenario: Upgrade from v1.4 to v1.5

v1.5 migration: RENAME COLUMN devices.type TO devices.kind

Problem: If we rollback to v1.4, the v1.4 code looks for
         "type" column but finds "kind" → crash
```

## Decision

**Enforce additive-only migrations for the v1.x series.**

Rules:
1. **Allowed**: `CREATE TABLE`, `ADD COLUMN`
2. **Forbidden**: `DROP TABLE`, `DROP COLUMN`, `RENAME COLUMN`
3. **Required**: New columns must be `NULLABLE` or have `DEFAULT`
4. **Deprecation**: Mark columns deprecated in code, don't remove from DB

### How to "Rename" a Column (Correctly)

```sql
-- WRONG: Breaks rollback
ALTER TABLE users RENAME COLUMN fullname TO display_name;

-- RIGHT: Additive approach
-- Step 1: Add new column
ALTER TABLE users ADD COLUMN display_name TEXT;

-- Step 2: Migrate data
UPDATE users SET display_name = fullname WHERE display_name IS NULL;

-- Step 3: Update application to use display_name
-- Step 4: Keep fullname (deprecated, don't delete)
```

## Consequences

### Positive
- **Safe rollback** — v1.4 binary works against v1.5 database (ignores new columns)
- **No data loss** — Columns are never deleted
- **Simple upgrades** — Just add, never modify or remove
- **Predictable** — Developers know what's allowed

### Negative
- **Schema cruft** — Unused columns accumulate over time
- **Larger database** — Deprecated columns still take space
- **Documentation burden** — Must track what's deprecated

### Cruft Mitigation

At major version (v2.0), we allow:
1. Fresh schema with cleaned-up design
2. Migration tool exports v1 data and imports to v2
3. No in-place migration — v2 creates new database file

This gives us decades before cruft becomes a problem, by which time a v2 rewrite may be warranted anyway.

## Implementation

Migrations are managed with `golang-migrate`:

```
migrations/
├── 001_initial_schema.up.sql
├── 001_initial_schema.down.sql    # down = no-op for additive
├── 002_add_device_profiles.up.sql
├── 002_add_device_profiles.down.sql
```

Each migration is wrapped in a transaction. On failure, the transaction rolls back and Core exits with error.

---

## Migration Safety Requirements

Even with additive-only migrations, things can go wrong:
- Migration SQL has a bug (data corruption)
- Disk full during migration
- Power loss mid-migration
- Logic errors in data transformation

### Mandatory Pre-Migration Backup

**CRITICAL: Every migration MUST be preceded by a verified backup.**

```yaml
migration_safety:
  # Automatic backup before any migration
  pre_migration_backup:
    enabled: true
    type: "sqlite3 .backup"
    location: "/var/backup/graylogic/pre-migration"
    verify: true  # PRAGMA integrity_check after backup
    retention: 5   # Keep last 5 pre-migration backups

  # Abort migration if backup fails
  require_backup_success: true
```

**Startup Sequence (with Safety):**

```go
func (db *Database) MigrateWithSafety() error {
    // 1. Check pending migrations
    pending, err := db.PendingMigrations()
    if err != nil || len(pending) == 0 {
        return err // No migrations needed
    }

    // 2. Create pre-migration backup
    backupPath := fmt.Sprintf("/var/backup/graylogic/pre-migration/graylogic_%s_before_%s.db",
        time.Now().Format("20060102_150405"),
        pending[0].Version)

    if err := db.BackupTo(backupPath); err != nil {
        return fmt.Errorf("pre-migration backup failed: %w", err)
    }

    // 3. Verify backup integrity
    if err := db.VerifyBackup(backupPath); err != nil {
        return fmt.Errorf("backup verification failed: %w", err)
    }

    logger.Info("Pre-migration backup created",
        "path", backupPath,
        "pending_migrations", len(pending))

    // 4. Run migrations
    if err := db.RunMigrations(); err != nil {
        logger.Error("Migration failed - backup available for rollback",
            "backup", backupPath,
            "error", err)
        return err
    }

    return nil
}
```

### Dry-Run Mode

Test migrations without applying changes:

```bash
# Dry-run: copy database, run migration, verify, discard
graylogic migrate --dry-run

# Output:
# Creating test copy of database...
# Running migration 047_add_device_tags.up.sql on test copy...
# Migration completed successfully (0.3s)
# Verifying database integrity... OK
# Row counts: devices=156 (+0), scenes=42 (+0), tags=0 (+156 new)
# Dry-run complete. Production database unchanged.
```

**Implementation:**

```go
func (db *Database) DryRunMigration() error {
    // 1. Create temporary copy
    tempDB := filepath.Join(os.TempDir(), "graylogic_dryrun.db")
    defer os.Remove(tempDB)

    if err := db.BackupTo(tempDB); err != nil {
        return err
    }

    // 2. Open copy and run migrations
    testDB, err := sql.Open("sqlite3", tempDB)
    if err != nil {
        return err
    }
    defer testDB.Close()

    if err := runMigrationsOn(testDB); err != nil {
        return fmt.Errorf("dry-run migration failed: %w", err)
    }

    // 3. Verify integrity
    if err := verifyIntegrity(testDB); err != nil {
        return fmt.Errorf("dry-run verification failed: %w", err)
    }

    // 4. Report changes (row count diffs, etc.)
    reportChanges(db, testDB)

    return nil
}
```

### Rollback Procedure

When a migration fails or causes issues after deployment:

```yaml
rollback_procedure:
  # Step 1: Stop Core immediately
  step_1_stop:
    command: "sudo systemctl stop graylogic"
    reason: "Prevent further data corruption"

  # Step 2: Locate pre-migration backup
  step_2_find_backup:
    location: "/var/backup/graylogic/pre-migration/"
    naming: "graylogic_{timestamp}_before_{version}.db"

  # Step 3: Restore from backup
  step_3_restore:
    commands:
      - "mv /var/lib/graylogic/graylogic.db /var/lib/graylogic/graylogic.db.failed"
      - "cp /var/backup/graylogic/pre-migration/graylogic_TIMESTAMP_before_VERSION.db /var/lib/graylogic/graylogic.db"
      - "chown graylogic:graylogic /var/lib/graylogic/graylogic.db"

  # Step 4: Downgrade binary (if needed)
  step_4_downgrade:
    note: "If rollback is due to failed migration, keep current binary"
    note_2: "If rollback is due to application bug, install previous binary"
    commands:
      - "apt install graylogic=1.4.0"  # or dpkg -i previous.deb

  # Step 5: Restart with previous state
  step_5_restart:
    command: "sudo systemctl start graylogic"

  # Step 6: Verify operation
  step_6_verify:
    actions:
      - "Check web UI loads"
      - "Verify device count matches expected"
      - "Test device control"
      - "Check logs for errors"
```

### Down Migration Scripts

Although additive-only means down migrations are usually no-ops, we still require them for documentation and edge cases:

```sql
-- migrations/047_add_device_tags.up.sql
CREATE TABLE device_tags (
    device_id TEXT REFERENCES devices(id),
    tag TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (device_id, tag)
);

-- migrations/047_add_device_tags.down.sql
-- Additive-only: down migration is informational only.
-- The device_tags table will remain but be unused by older code.
-- To fully remove: restore from pre-migration backup.

-- INFORMATIONAL: This table was added in migration 047.
-- Previous versions do not use this table.
-- Safe to leave in place for rollback compatibility.

SELECT 'Migration 047 down: device_tags table left in place (additive-only policy)';
```

### Migration Failure Handling

```go
// On startup, if migration fails:
func handleMigrationFailure(err error, backupPath string) {
    logger.Error("MIGRATION FAILED",
        "error", err,
        "backup_available", backupPath,
        "recovery_steps", []string{
            "1. Do not restart without investigating",
            "2. Check /var/log/graylogic/core.log for details",
            "3. To rollback: see docs/architecture/decisions/004-additive-only-migrations.md",
            "4. Backup location: " + backupPath,
        })

    // Write recovery instructions to prominent location
    recoveryFile := "/var/lib/graylogic/RECOVERY_NEEDED.txt"
    ioutil.WriteFile(recoveryFile, []byte(fmt.Sprintf(`
GRAY LOGIC MIGRATION FAILED
===========================

Backup available at: %s

To rollback:
1. sudo systemctl stop graylogic
2. cp %s /var/lib/graylogic/graylogic.db
3. sudo chown graylogic:graylogic /var/lib/graylogic/graylogic.db
4. sudo systemctl start graylogic

See docs/architecture/decisions/004-additive-only-migrations.md for details.
`, backupPath, backupPath)), 0644)

    os.Exit(1)  // Do NOT continue with broken database
}
```

### Commissioning Test: Migration Safety

During commissioning, verify migration safety works:

```yaml
commissioning_test_migration:
  name: "Verify migration safety procedures"
  steps:
    - action: "Run dry-run migration (should report no pending)"
      command: "graylogic migrate --dry-run"
      expected: "No pending migrations"

    - action: "Verify pre-migration backups exist"
      command: "ls -la /var/backup/graylogic/pre-migration/"
      expected: "At least one backup file from installation"

    - action: "Test backup restore procedure"
      commands:
        - "sudo systemctl stop graylogic"
        - "cp /var/backup/graylogic/pre-migration/LATEST.db /tmp/restore-test.db"
        - "sqlite3 /tmp/restore-test.db 'PRAGMA integrity_check;'"
      expected: "ok"

    - action: "Document backup location in handover pack"
      verify: "Customer knows where backups are stored"
```

## Related

- [ADR-003: SQLite as Primary Database](003-sqlite-primary-database.md)
- [Database Schema Strategy](../../development/database-schema.md)
- [Backup and Recovery Specification](../../resilience/backup.md)
