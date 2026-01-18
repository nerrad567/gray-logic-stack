---
title: Database Schema Strategy
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - architecture/core-internals.md
  - operations/infrastructure.md
---

# Database Schema Strategy

This document defines the rules for database schema design and evolution. To support the "multi-decade stability" goal and reliable rollbacks, we enforce strict **Additive-Only** schema policies.

---

## The Golden Rule: Additive Changes Only

Once a column or table is created in production, it **must never be dropped or renamed** during the active life of that version series (v1.x).

### Why?
1.  **Rollback Safety:** If v1.5 adds a column and we rollback to v1.4, the v1.4 binary will simply ignore the extra column. If we *renamed* a column, v1.4 would crash.
2.  **Data Preservation:** We never want to accidentally delete user data during an upgrade/downgrade cycle.

---

## Schema Evolution Rules

### 1. Adding Features
*   **Allowed:** `CREATE TABLE`, `ADD COLUMN`.
*   **Constraint:** New columns must be `NULLABLE` or have a `DEFAULT` value.

### 2. Deprecating Features
*   **Action:** Mark the column as deprecated in documentation/code.
*   **Constraint:** Do **not** `DROP COLUMN`. The column remains in the DB forever (or until v2.0 major migration).
*   **Constraint:** Do **not** re-purpose the column.

### 3. Renaming (Don't)
*   **Scenario:** You want to rename `user.fullname` to `user.display_name`.
*   **Wrong Way:** `ALTER TABLE users RENAME COLUMN fullname TO display_name;` (Breaks rollback).
*   **Right Way:**
    1.  Add `display_name` column.
    2.  Write a migration to copy data: `UPDATE users SET display_name = fullname;`
    3.  Update code to read/write `display_name`.
    4.  Keep `fullname` populated (via application logic or trigger) if backward compatibility is strictly required, otherwise leave it as a "tombstone".

### 4. Constraints
*   **Allowed:** Adding indexes (performance optimization).
*   **Careful:** Adding `UNIQUE` or `FOREIGN KEY` constraints to existing tables requires ensuring data integrity first.

---

## SQLite Specifics

### Mode
All tables must be created with **`STRICT`** mode (available in SQLite 3.37+) to enforce type safety.

```sql
CREATE TABLE devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
) STRICT;
```

### Journaling
*   **Mode:** `WAL` (Write-Ahead Logging) enabled.
*   **Synchronous:** `NORMAL` (Safe enough for home automation, much faster).

---

## Migration Testing

Every schema change must pass the **Rollback Test**:

1.  Start DB at Version A.
2.  Migrate to Version B.
3.  Add data using Version B logic.
4.  **Run Version A binary against Version B database.**
5.  **Success Condition:** Version A starts and operates normally (ignoring new fields).

---

## Version 2.0 Strategy

If the schema accumulates too much "cruft" (unused columns) over decades:
1.  **Major Release (v2.0):** This is the only time destructive changes are allowed.
2.  **Migration Tool:** A specialized tool exports v1 data, transforms it, and imports it into a clean v2 schema.
3.  **No In-Place Upgrade:** v2 creates a *new* database file.
