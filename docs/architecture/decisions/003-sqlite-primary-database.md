---
title: "ADR-003: SQLite as Primary Database"
status: accepted
date: 2026-01-15
---

# ADR-003: SQLite as Primary Database

## Status

Accepted

## Context

Gray Logic needs persistent storage for:
- Device configuration and metadata
- Scene and schedule definitions
- User accounts and permissions
- System configuration
- Audit logs

Requirements:
- Must work reliably for 10+ years
- Zero external dependencies (no database server to manage)
- Survive unexpected power loss
- Easy backup and restore
- Support typical home automation scale (100-500 devices)

We considered:

| Option | Pros | Cons |
|--------|------|------|
| **SQLite** | Embedded, zero maintenance, reliable, single file | Single writer, limited concurrency |
| **PostgreSQL** | Full-featured, concurrent, scalable | External server, maintenance, complexity |
| **MySQL/MariaDB** | Widely used, good tooling | External server, maintenance |
| **BoltDB/BadgerDB** | Embedded, fast | Key-value only, less queryable |
| **Flat files (JSON/YAML)** | Simple | No transactions, no queries |

## Decision

**Use SQLite as the primary database with STRICT mode and WAL journaling.**

Key factors:
1. **Embedded** — No external database server to install, configure, or maintain
2. **Single file** — Easy backup (just copy the file), easy restore
3. **Battle-tested** — SQLite is used in billions of devices worldwide
4. **Reliable** — Survives power loss with WAL journaling
5. **Sufficient performance** — More than adequate for 500 devices, thousands of entities
6. **10-year stability** — SQLite has excellent backward compatibility
7. **STRICT mode** — Enforces type safety (SQLite 3.37+)

## Consequences

### Positive
- Zero database administration
- Backup is just file copy (with `VACUUM INTO` for hot backup)
- No connection management or pooling complexity
- Small footprint (library embedded in binary)
- Queries are standard SQL

### Negative
- Single writer limit (mitigated by WAL mode)
- No clustering or replication built-in
- Large-scale analytics better suited to time-series DB

### Configuration

```sql
-- Enable WAL mode for better concurrency
PRAGMA journal_mode = WAL;

-- NORMAL sync is safe with WAL and much faster
PRAGMA synchronous = NORMAL;

-- Enable foreign key enforcement
PRAGMA foreign_keys = ON;

-- Use STRICT tables for type safety
CREATE TABLE devices (...) STRICT;
```

### Time-Series Data

SQLite is used for configuration and state. Time-series data (energy readings, temperature history) goes to **InfluxDB** — a separate system optimized for that purpose.

See: [Infrastructure & Data Strategy](../../operations/infrastructure.md)

## Related

- [ADR-004: Additive-Only Migrations](004-additive-only-migrations.md)
- [Database Schema Strategy](../../development/database-schema.md)
- [Infrastructure Strategy](../../operations/infrastructure.md)
