# Codex Task: Phase 3 — SQLite State Audit Trail

## Summary

Add a local state change audit trail to SQLite. Every device state change that flows through the MQTT → WebSocket pipeline should also be logged to a `state_history` table with timestamp, device_id, and the state snapshot.

This gives queryable local history even when VictoriaMetrics is down or disabled, and supports the future REST endpoint `GET /api/v1/devices/{id}/history`.

## Context

The state pipeline was wired in commit `5cf245e`:

```
MQTT state message
  → WebSocket broadcast (hub.Broadcast)
  → Device Registry update (registry.SetDeviceState)   ← Phase 1
  → VictoriaMetrics write (tsdb.WriteDeviceMetric)     ← Phase 2
  → SQLite audit trail                                 ← THIS TASK (Phase 3)
```

The insertion point is in `internal/api/websocket.go` inside `subscribeStateUpdates()`, after the existing registry update and TSDB write blocks (around line 220).

## Requirements

### 1. Create `state_history` table

**File: `migrations/008_state_history.go`** (follow existing migration pattern)

```sql
CREATE TABLE IF NOT EXISTS state_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,
    state TEXT NOT NULL,          -- JSON-encoded state snapshot
    source TEXT DEFAULT 'mqtt',   -- 'mqtt', 'command', 'scene'
    created_at DATETIME DEFAULT (datetime('now'))
);

CREATE INDEX idx_state_history_device ON state_history(device_id, created_at DESC);
CREATE INDEX idx_state_history_time ON state_history(created_at DESC);
```

### 2. Create state history repository

**File: `internal/device/state_history.go`** (~80 lines)

```go
// StateHistoryEntry represents a single state change record.
type StateHistoryEntry struct {
    ID        int64     `json:"id"`
    DeviceID  string    `json:"device_id"`
    State     State     `json:"state"`
    Source    string    `json:"source"`
    CreatedAt time.Time `json:"created_at"`
}

// StateHistoryRepository stores device state change history.
type StateHistoryRepository interface {
    RecordStateChange(ctx context.Context, deviceID string, state State, source string) error
    GetHistory(ctx context.Context, deviceID string, limit int) ([]StateHistoryEntry, error)
}
```

**File: `internal/device/state_history_sqlite.go`** (~100 lines)

Implement `StateHistoryRepository` using `*sql.DB`:
- `RecordStateChange`: INSERT with JSON-marshaled state
- `GetHistory`: SELECT with ORDER BY created_at DESC, LIMIT
- Follow existing repository patterns (see `internal/device/repository.go`)

### 3. Wire into state pipeline

**File: `internal/api/server.go`**

Add `stateHistory device.StateHistoryRepository` to `Server` struct and `Deps`.

**File: `internal/api/websocket.go`**

In `subscribeStateUpdates()`, after the TSDB write block, add:

```go
// Record state change to local audit trail
if s.stateHistory != nil {
    if err := s.stateHistory.RecordStateChange(context.Background(), deviceID, devState, "mqtt"); err != nil {
        s.logger.Debug("state history write failed", "device_id", deviceID, "error", err)
    }
}
```

**File: `cmd/graylogic/main.go`**

Create `StateHistorySQLiteRepository` and pass to API deps.

### 4. Add retention policy

The `state_history` table will grow continuously. Add a periodic cleanup:

```go
// PruneHistory deletes entries older than the given duration.
func (r *SQLiteStateHistoryRepository) PruneHistory(ctx context.Context, olderThan time.Duration) (int64, error)
```

Call this from a background goroutine (daily, delete entries older than 30 days).

### 5. Tests

**File: `internal/device/state_history_test.go`**

- `TestRecordStateChange` — write and verify
- `TestGetHistory` — write multiple, verify order (newest first) and limit
- `TestPruneHistory` — write old entries, prune, verify deleted
- Use in-memory SQLite (`":memory:"`) for all tests

## Files to Create/Modify

| File | Action |
|------|--------|
| `migrations/008_state_history.go` | CREATE |
| `internal/device/state_history.go` | CREATE (interface + types) |
| `internal/device/state_history_sqlite.go` | CREATE (implementation) |
| `internal/device/state_history_test.go` | CREATE (tests) |
| `internal/api/server.go` | MODIFY (add stateHistory field) |
| `internal/api/websocket.go` | MODIFY (add write after TSDB block) |
| `cmd/graylogic/main.go` | MODIFY (create repo, pass to deps) |

## Patterns to Follow

- Look at `internal/audit/repository.go` for SQLite repository pattern
- Look at `internal/device/repository.go` for device-specific repository patterns
- Look at `migrations/` for migration file naming convention
- Look at `internal/api/websocket.go:subscribeStateUpdates()` for the insertion point
- All repository methods take `context.Context` as first parameter
- Use `Debug` level logging for expected failures (e.g., device not found)

## Verification

```bash
cd code/core

# Build
go build ./...

# Test
make test-race

# Verify migration runs
# (start the app and check logs for "database migrations complete")
```

## Constraints

- No external dependencies — use `database/sql` + `encoding/json`
- Thread-safe — multiple MQTT messages may arrive concurrently
- Non-blocking — state history write failures must not block the pipeline
- Follow existing code style (check `golangci-lint run`)
