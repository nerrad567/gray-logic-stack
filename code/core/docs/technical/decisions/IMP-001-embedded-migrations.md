# IMP-001: Embedded Migrations with embed.FS

## Status
Accepted

## Date
2026-01-19

## Context

Gray Logic Core requires database schema migrations for:
- Initial table creation (sites, areas, rooms, devices, audit_logs)
- Future schema evolution over 10+ year deployment horizon

Migrations must be:
- **Reliable** — Work identically in dev, test, and production
- **Self-contained** — No external file dependencies at runtime
- **Versioned** — Track which migrations have been applied

## Decision

Use Go 1.16+ `embed.FS` to compile SQL migration files directly into the binary.

```go
//go:embed *.sql
var MigrationsFS embed.FS
```

The migration runner reads files from `MigrationsFS` rather than the filesystem.

## Implementation

**File structure:**
```
migrations/
├── embed.go                              # //go:embed directive
├── 20260118_200000_initial_schema.up.sql
└── 20260118_200000_initial_schema.down.sql
```

**embed.go:**
```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

func init() {
    database.MigrationsFS = FS
    database.MigrationsDir = "."
}
```

**Migration loading** ([migrations.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/database/migrations.go#L298-L327)):
```go
func loadMigrations() ([]Migration, error) {
    entries, err := fs.ReadDir(MigrationsFS, MigrationsDir)
    // ... parse and load
}
```

## Consequences

### Advantages
- **Single binary deployment** — No need to copy migration files
- **Immutable at runtime** — Migrations can't be accidentally modified
- **Version consistency** — Binary version matches migration version
- **Cross-platform** — Works identically on Linux, macOS, Windows

### Disadvantages
- **Rebuild required** — New migrations require recompiling
- **Binary size** — SQL files increase binary size (negligible, ~10KB)

### Risks
- **Forgotten rebuild** — Deploying old binary won't have new migrations
  - Mitigation: CI/CD always builds fresh

## Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|----------------|
| File-based migrations | External dependency, deployment complexity |
| SQL in Go strings | Hard to read, no syntax highlighting |
| Migration library (golang-migrate) | External dependency, overkill for simple needs |
