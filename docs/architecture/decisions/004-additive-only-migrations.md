---
title: "ADR-004: Additive-Only Database Migrations"
status: accepted
date: 2026-01-15
---

# ADR-004: Additive-Only Database Migrations

## Status

Accepted

## Context

Gray Logic is designed for 10-year deployments with these requirements:
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

This gives us ~10 years before cruft becomes a problem, by which time a v2 rewrite may be warranted anyway.

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

## Related

- [ADR-003: SQLite as Primary Database](003-sqlite-primary-database.md)
- [Database Schema Strategy](../../development/database-schema.md)
