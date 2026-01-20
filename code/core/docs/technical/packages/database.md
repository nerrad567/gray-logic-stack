# Database Package Design

> `internal/infrastructure/database/` — SQLite connectivity with migrations

## Purpose

Provides SQLite database connectivity for Gray Logic Core with:
- WAL mode for concurrent read/write access
- Embedded schema migrations (compiled into binary)
- Connection pooling and lifecycle management
- STRICT mode enforcement for type safety

**Why SQLite?** See [ADR-003: SQLite for Local Storage](../../../../../docs/architecture/decisions/003-sqlite-local-storage.md)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       database.DB                            │
│  ┌─────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   *sql.DB   │  │   Migrations    │  │   HealthCheck   │  │
│  │  (embedded) │  │  (embed.FS)     │  │   (SELECT 1)    │  │
│  └─────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌──────────────────────────────┐
              │     SQLite Database File      │
              │   (WAL mode, STRICT tables)   │
              └──────────────────────────────┘
```

### Key Types

| Type | File | Purpose |
|------|------|---------|
| `DB` | [database.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/database/database.go#L34-L37) | Wraps `*sql.DB` with migrations and health checks |
| `Config` | [database.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/database/database.go#L41-L53) | Database configuration options |
| `Migration` | [migrations.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/database/migrations.go#L42-L55) | Single migration with up/down SQL |
| `MigrationRecord` | [migrations.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/database/migrations.go#L58-L61) | Applied migration tracking |

### External Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/mattn/go-sqlite3` | v1.14.33 | SQLite driver (CGO) |

---

## How It Works

### Initialization

```go
// 1. Create config (typically from config.yaml)
cfg := database.Config{
    Path:        "/var/lib/graylogic/core.db",
    WALMode:     true,
    BusyTimeout: 5,
}

// 2. Open connection (context for cancellation/timeout)
ctx := context.Background()
db, err := database.Open(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// 3. Run migrations
if err := db.Migrate(ctx); err != nil {
    log.Fatal(err)
}
```

**Open(ctx, cfg) performs:**
1. Creates database directory if not exists (`0750` permissions)
2. Opens database with connection string pragmas
3. Configures connection pool (1 writer, 1 idle)
4. Verifies connectivity with ping (5s timeout enforced, even if context has no deadline)
5. Sets file permissions (`0600`)

### Core Operations

**Query execution:**
```go
// Single row
var name string
err := db.QueryRowContext(ctx, "SELECT name FROM sites WHERE id = ?", id).Scan(&name)

// Multiple rows
rows, err := db.QueryContext(ctx, "SELECT * FROM devices WHERE area_id = ?", areaID)

// Mutations
result, err := db.ExecContext(ctx, "UPDATE devices SET state = ? WHERE id = ?", state, id)
```

**Transactions:**
```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback() // No-op if committed

// Execute statements on tx...
tx.ExecContext(ctx, "INSERT INTO ...")
tx.ExecContext(ctx, "UPDATE ...")

return tx.Commit()
```

### Shutdown

```go
if err := db.Close(); err != nil {
    log.Printf("Error closing database: %v", err)
}
```

Closes underlying `*sql.DB`, releasing file handles.

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **WAL mode** | Concurrent reads during writes; better crash recovery | DELETE mode (simpler but blocks) |
| **Single writer** | SQLite limitation; prevents lock contention | Multiple connections (causes "database locked") |
| **embed.FS migrations** | Single binary deployment; no external files | File-based (deployment complexity) |
| **Additive-only schema** | Safe rollbacks over multi-decade horizon | ALTER/DROP (risk of data loss) |
| **STRICT mode** | Type enforcement prevents silent corruption | Flexible types (data integrity risk) |
| **Per-migration atomicity** | Allows partial progress, matches SQLite model | All-or-nothing (lock timeout risk) |

### Migration Atomicity

Migrations use **per-migration atomicity**: each migration runs in its own transaction.

If migration N fails:
- Migrations 1 to N-1 **remain committed**
- Migration N is **rolled back**
- Migrations N+1 onwards are **not attempted**

This design is intentional:
- **Partial progress**: Large migration batches can make progress even with one failure
- **SQLite model**: Matches single-writer semantics; avoids lock timeouts
- **Debuggability**: `schema_migrations` table shows exactly which migration failed
- **Resumable**: Re-running `Migrate()` after fixing the issue continues from N

For all-or-nothing semantics, wrap the `Migrate()` call in your own transaction, but be aware this may hit SQLite busy timeouts on large migrations.

See also:
- [IMP-001: Embedded Migrations](../decisions/IMP-001-embedded-migrations.md)
- [IMP-002: WAL Mode](../decisions/IMP-002-wal-mode.md)

---

## Interactions

### Dependencies (what this package uses)

| Package | Purpose |
|---------|---------|
| `migrations/` | Embedded SQL files via `embed.FS` |

### Dependents (what uses this package)

| Package | Purpose |
|---------|---------|
| `cmd/graylogic/main.go` | Creates and manages lifecycle |
| `device/registry/` | (Future) Device CRUD operations |
| `automation/scene/` | (Future) Scene persistence |

---

## Error Handling

All errors are wrapped with context using `%w`:

```go
return nil, fmt.Errorf("opening database: %w", err)
return nil, fmt.Errorf("creating database directory: %w", err)
return fmt.Errorf("database health check failed: %w", err)
```

**Caller pattern:**
```go
db, err := database.Open(cfg)
if err != nil {
    if errors.Is(err, os.ErrPermission) {
        // Handle permission error
    }
    return fmt.Errorf("initializing database: %w", err)
}
```

---

## Thread Safety

| Operation | Thread-Safe | Notes |
|-----------|-------------|-------|
| `Open()` | N/A | Called once at startup |
| `Close()` | Yes | Safe to call from signal handler |
| `HealthCheck()` | Yes | Read-only query |
| `Migrate()` | No | Call only during startup |
| `ExecContext()` | Yes | Connection pool handles |
| `QueryContext()` | Yes | Connection pool handles |
| `BeginTx()` | Yes | Returns isolated transaction |

**Connection Pool:** Single writer means concurrent writes are serialized. Reads remain concurrent.

---

## Configuration

From `configs/config.yaml`:

```yaml
database:
  # Path to SQLite database file
  # Default: ./data/graylogic.db
  path: "./data/graylogic.db"
  
  # Enable Write-Ahead Logging for concurrent access
  # Default: true (strongly recommended)
  wal_mode: true
  
  # Maximum wait time for database lock (seconds)
  # Default: 5
  busy_timeout: 5
```

**Validation:**
- `path`: Required, non-empty
- `wal_mode`: Boolean (default true)
- `busy_timeout`: Integer > 0 (default 5)

---

## Testing

**Unit tests:** Require no external dependencies.

```bash
cd /home/graylogic-dev/gray-logic-stack/code/core
make test PKG=./internal/infrastructure/database/...
```

**Test files:**
- `database_test.go` — Connection, health check, transactions
- `migrations_test.go` — Migration apply, rollback, status
- `testdata/*.sql` — Embedded test fixtures

**Coverage:** 80.4%

---

## Known Limitations

1. **Single writer** — SQLite limitation; high-write workloads may contend
2. **CGO required** — `go-sqlite3` needs C compiler; cross-compilation harder
3. **No JSON functions** — SQLite JSON1 available but not yet used
4. **No encryption at rest** — Planned for future (SQLCipher)

---

## Related Documents

- [doc.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/database/doc.go) — Package-level godoc
- [docs/development/database-schema.md](file:///home/graylogic-dev/gray-logic-stack/docs/development/database-schema.md) — Schema design strategy
- [docs/data-model/entities.md](file:///home/graylogic-dev/gray-logic-stack/docs/data-model/entities.md) — Entity definitions
