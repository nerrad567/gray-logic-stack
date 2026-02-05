# IMP-002: WAL Mode for SQLite

## Status
Accepted

## Date
2026-01-19

## Context

Gray Logic Core uses SQLite for persistent storage. The default rollback journal mode has limitations:
- Writers block all readers
- Only one connection can write at a time
- Readers block writers

This causes contention when:
- API serves multiple concurrent read requests
- State updates write frequently
- VictoriaMetrics sync reads while automations write

## Decision

Enable Write-Ahead Logging (WAL) mode with `SYNCHRONOUS=NORMAL`.

```go
if cfg.WALMode {
    connStr += "&_journal_mode=WAL&_synchronous=NORMAL"
}
```

## Implementation

**Connection string** ([database.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/database/database.go#L77-L86)):
```go
connStr := fmt.Sprintf("file:%s?_busy_timeout=%d&_foreign_keys=on",
    cfg.Path,
    cfg.BusyTimeout*msPerSecond,
)

if cfg.WALMode {
    connStr += "&_journal_mode=WAL&_synchronous=NORMAL"
}
```

**Configuration:**
```yaml
database:
  wal_mode: true  # Default: true
```

## Consequences

### Advantages
- **Concurrent reads during writes** — Readers don't block, writers don't block readers
- **Better crash recovery** — WAL is more robust than rollback journal
- **Improved write performance** — Sequential writes to WAL file

### Disadvantages
- **WAL file overhead** — Additional `-wal` and `-shm` files alongside database
- **Checkpointing** — Periodic checkpoint operations (SQLite handles automatically)
- **Network filesystem incompatible** — WAL requires local filesystem

### Risks
- **Large WAL file** — If checkpointing fails, WAL grows unbounded
  - Mitigation: SQLite auto-checkpoints; monitor file size
- **Corruption on network storage** — WAL doesn't work over NFS/SMB
  - Mitigation: Document requirement for local storage

## Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|----------------|
| DELETE mode (rollback) | Readers block writers, poor concurrency |
| MEMORY mode | No persistence, unsuitable for production |
| SYNCHRONOUS=FULL | Excessive fsync calls, performance penalty |
| SYNCHRONOUS=OFF | Risk of corruption on power loss |

## References

- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
- [Appropriate Uses for SQLite](https://www.sqlite.org/whentouse.html)
