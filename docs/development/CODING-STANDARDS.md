---
title: Coding Standards
version: 1.0.0
status: active
last_updated: 2026-01-16
depends_on:
  - development/DEVELOPMENT-STRATEGY.md
  - overview/principles.md
  - architecture/core-internals.md
---

# Gray Logic Coding Standards

This document defines how we write, document, and test Gray Logic code.

---

## Philosophy

### Documentation-First

**Every module, every function, every non-trivial block must be documented.**

Why?
1. **Multi-decade horizon** — Future maintainers (including future you) need to understand intent
2. **Handover requirement** — Customers get full source; it must be comprehensible
3. **Security audits** — Auditors need to understand what code does without running it
4. **Onboarding** — New developers learn faster from good docs

**Rule:** If it's worth writing, it's worth documenting.

---

## Go Code Standards

### Package Documentation

Every package must have a `doc.go` file with:

```go
// Package lighting implements the lighting control subsystem for Gray Logic.
//
// This package manages:
//   - Light device registration and discovery
//   - Dimming, switching, and color control
//   - Scene recall for lighting
//   - Integration with KNX and DALI protocols
//
// Security Considerations:
//   - All commands are validated against device capabilities
//   - Rate limiting prevents flood attacks
//   - Dimming transitions are bounded to prevent epileptic triggers
//
// Performance Characteristics:
//   - Command processing: <10ms per command
//   - Scene recall: <500ms for 20 devices
//   - Memory: ~100KB per 100 devices
//
// Related Documents:
//   - docs/domains/lighting.md — Lighting domain specification
//   - docs/protocols/knx.md — KNX protocol details
//   - docs/protocols/dali.md — DALI protocol details
package lighting
```

**Required sections:**
1. **Package purpose** (one-line summary)
2. **What it manages** (bullet list)
3. **Security considerations** (attack surface, mitigations)
4. **Performance characteristics** (latency, throughput, memory)
5. **Related documents** (cross-references)

---

### Function Documentation

Every exported function must have:

```go
// ActivateScene recalls a lighting scene by ID, applying saved device states.
//
// The scene is executed in two phases:
//  1. Fast devices (switches) execute immediately
//  2. Slow devices (dimmers) ramp over the scene's transition time
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - sceneID: UUID of the scene to activate
//   - options: Optional overrides (e.g., transition time)
//
// Returns:
//   - error: nil on success, or:
//     - ErrSceneNotFound if sceneID doesn't exist
//     - ErrDeviceOffline if a critical device is unreachable
//     - context.DeadlineExceeded if activation takes >5s
//
// Security:
//   - Requires 'scenes:activate' permission
//   - Rate limited to 10 activations per minute per user
//
// Example:
//   err := ActivateScene(ctx, sceneID, nil)
//   if err != nil {
//       log.Errorf("Scene activation failed: %v", err)
//   }
func ActivateScene(ctx context.Context, sceneID uuid.UUID, options *SceneOptions) error {
    // Implementation...
}
```

**Required sections:**
1. **Purpose** (what the function does)
2. **How it works** (algorithm or phases, if non-obvious)
3. **Parameters** (each parameter with description)
4. **Returns** (success case and all error types)
5. **Security** (auth requirements, rate limits)
6. **Example** (typical usage)

---

### Internal Functions and Complex Logic

Non-exported functions with non-trivial logic must be documented:

```go
// validateSceneDevices checks that all devices in a scene are:
//  1. Registered in the device registry
//  2. Online and reachable
//  3. Support the requested capability (e.g., dimming)
//
// Returns ErrDeviceNotFound, ErrDeviceOffline, or ErrCapabilityNotSupported.
func validateSceneDevices(scene *Scene) error {
    // Implementation...
}
```

---

### Inline Comments

**Comment intent, not implementation.**

❌ **Bad:**
```go
// Loop through devices
for _, device := range devices {
    // Check if device is online
    if device.Online {
        // Send command
        device.SendCommand(cmd)
    }
}
```

✅ **Good:**
```go
// Only send commands to online devices to avoid timeouts.
// Offline devices will receive the command when they reconnect
// via the state synchronization mechanism.
for _, device := range devices {
    if device.Online {
        device.SendCommand(cmd)
    }
}
```

**When to comment:**
- Non-obvious algorithms (e.g., "Using exponential backoff to avoid thundering herd")
- Security-relevant code (e.g., "Validate input to prevent SQL injection")
- Performance trade-offs (e.g., "Cache invalidation instead of DB query for 10x speedup")
- Edge cases (e.g., "Handle empty list to prevent index out of range")
- Workarounds (e.g., "KNX devices don't report status; polling every 30s instead")

**When NOT to comment:**
- Obvious code (e.g., `i++` doesn't need "increment i")
- Code that can be made self-documenting with better naming

---

### Struct Documentation

Document every exported struct and its fields:

```go
// Scene represents a saved state of multiple devices to be recalled together.
//
// Scenes support:
//   - Multi-device coordination (lights, blinds, climate)
//   - Timed transitions (fade times, ramp rates)
//   - Conditional activation (e.g., only if mode=Night)
type Scene struct {
    // ID is the unique identifier for this scene (UUID v4).
    ID uuid.UUID `json:"id" db:"id"`

    // Name is the human-readable scene name (e.g., "Good Night").
    // Must be unique within the room or area scope.
    Name string `json:"name" db:"name" validate:"required,max=50"`

    // Scope defines where the scene is available (Room, Area, Site).
    Scope ScopeType `json:"scope" db:"scope"`

    // Actions are the device commands to execute when scene is activated.
    // Executed in order; failures are logged but don't stop execution.
    Actions []SceneAction `json:"actions" db:"-"`

    // TransitionTimeMs is the default fade/ramp time in milliseconds.
    // Individual actions can override this.
    TransitionTimeMs int `json:"transition_time_ms" db:"transition_time_ms" validate:"min=0,max=60000"`

    // CreatedAt is the scene creation timestamp (UTC).
    CreatedAt time.Time `json:"created_at" db:"created_at"`

    // UpdatedAt is the last modification timestamp (UTC).
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
```

**Document:**
- What the struct represents
- What it supports (capabilities)
- Field purpose and constraints
- Database mappings and validation rules

---

## Error Handling

### Error Types

Define domain-specific errors:

```go
package lighting

import "errors"

var (
    // ErrSceneNotFound is returned when a scene ID doesn't exist in the database.
    ErrSceneNotFound = errors.New("scene not found")

    // ErrDeviceOffline is returned when a scene activation requires a device
    // that is currently unreachable. Scenes can be partially activated if
    // non-critical devices are offline (determined by scene definition).
    ErrDeviceOffline = errors.New("device offline")

    // ErrInvalidTransitionTime is returned when a transition time is negative
    // or exceeds the 60-second maximum (to prevent locking the system).
    ErrInvalidTransitionTime = errors.New("invalid transition time")
)
```

### Error Wrapping

Use `fmt.Errorf` with `%w` to preserve error chains:

```go
func ActivateScene(ctx context.Context, sceneID uuid.UUID) error {
    scene, err := s.repo.GetSceneByID(ctx, sceneID)
    if err != nil {
        return fmt.Errorf("failed to load scene %s: %w", sceneID, err)
    }

    if err := s.validateDevices(scene); err != nil {
        return fmt.Errorf("scene validation failed: %w", err)
    }

    // Execute scene...
    return nil
}
```

**Benefits:**
- Preserves root cause for debugging
- Allows `errors.Is()` and `errors.As()` checks
- Provides context at each layer

### Error Logging

```go
// Log errors with context
log.WithFields(log.Fields{
    "scene_id": sceneID,
    "user_id":  userID,
    "error":    err,
}).Error("Scene activation failed")
```

**Include:**
- Entity IDs (scene, device, user)
- Operation being attempted
- Error details

**Don't include:**
- Passwords or secrets
- Full request payloads (log fingerprint instead)

---

## Testing Standards

### Test Coverage

**Minimum requirements:**
- 80% coverage for new code
- 100% coverage for security-critical code (auth, validation, crypto)
- 100% coverage for error paths

**Measure coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Unit Tests

Test files live alongside code (`*_test.go`).

**Structure:**
```go
package lighting_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gray-logic/internal/domain/lighting"
)

// TestActivateScene_Success validates scene activation with all devices online.
func TestActivateScene_Success(t *testing.T) {
    // Arrange
    ctx := context.Background()
    scene := &lighting.Scene{
        ID:   uuid.New(),
        Name: "Test Scene",
        Actions: []lighting.SceneAction{
            {DeviceID: "device1", Command: "on"},
            {DeviceID: "device2", Command: "dim", Level: 50},
        },
    }
    service := lighting.NewService(mockRepo, mockBridge)

    // Act
    err := service.ActivateScene(ctx, scene.ID, nil)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, "on", mockBridge.LastCommands["device1"])
    assert.Equal(t, 50, mockBridge.LastLevels["device2"])
}

// TestActivateScene_DeviceOffline validates partial activation when devices are offline.
func TestActivateScene_DeviceOffline(t *testing.T) {
    // Arrange
    ctx := context.Background()
    scene := &lighting.Scene{
        ID:   uuid.New(),
        Actions: []lighting.SceneAction{
            {DeviceID: "online1", Command: "on"},
            {DeviceID: "offline1", Command: "on"},
        },
    }
    mockBridge.SetDeviceOffline("offline1")
    service := lighting.NewService(mockRepo, mockBridge)

    // Act
    err := service.ActivateScene(ctx, scene.ID, nil)

    // Assert
    assert.ErrorIs(t, err, lighting.ErrDeviceOffline)
    // Online device should still have been activated
    assert.Equal(t, "on", mockBridge.LastCommands["online1"])
}
```

**Test naming convention:**
```
Test<FunctionName>_<Scenario>
```

Examples:
- `TestActivateScene_Success`
- `TestActivateScene_DeviceOffline`
- `TestActivateScene_InvalidSceneID`
- `TestActivateScene_RateLimitExceeded`

**Use table-driven tests for multiple scenarios:**
```go
func TestValidateTransitionTime(t *testing.T) {
    tests := []struct {
        name        string
        input       int
        expectError bool
    }{
        {"zero is valid", 0, false},
        {"1 second is valid", 1000, false},
        {"60 seconds is valid", 60000, false},
        {"negative is invalid", -1, true},
        {"over 60s is invalid", 60001, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateTransitionTime(tt.input)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Integration Tests

Integration tests live in `tests/integration/`:

```go
//go:build integration

package integration

import (
    "context"
    "testing"
    "gray-logic/internal/core"
)

// TestEndToEnd_SceneActivation validates scene activation from API to device.
func TestEndToEnd_SceneActivation(t *testing.T) {
    // Start Core, MQTT, mock KNX bridge
    testEnv := setupTestEnvironment(t)
    defer testEnv.Teardown()

    // Create scene via API
    sceneID := testEnv.CreateScene("Test Scene", []SceneAction{...})

    // Activate scene via API
    resp := testEnv.APIClient.Post("/api/scenes/" + sceneID + "/activate")
    require.Equal(t, 200, resp.StatusCode)

    // Verify device received command
    assert.Eventually(t, func() bool {
        return testEnv.MockKNXBridge.ReceivedCommand("1.2.3", "on")
    }, 1*time.Second, 100*time.Millisecond)
}
```

**Run integration tests:**
```bash
go test -tags=integration ./tests/integration/...
```

### Benchmark Tests

For performance-critical code:

```go
func BenchmarkActivateScene(b *testing.B) {
    ctx := context.Background()
    service := setupBenchmarkService()
    sceneID := setupBenchmarkScene()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = service.ActivateScene(ctx, sceneID, nil)
    }
}
```

**Run benchmarks:**
```bash
go test -bench=. -benchmem ./...
```

---

## Code Style

### Formatting

Use `gofmt` and `goimports`:

```bash
gofmt -w .
goimports -w .
```

**Enforced in CI:**
```bash
gofmt -d . | tee /dev/stderr | test -z "$(cat)"
```

### Linting

Use `golangci-lint` with strict settings:

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - stylecheck
    - gosec  # Security linter
    - gocritic
    - revive

linters-settings:
  gosec:
    excludes:
      - G104  # Don't fail on unhandled errors in defer
  gocritic:
    enabled-tags:
      - diagnostic
      - performance
      - style
```

**Run linter:**
```bash
golangci-lint run ./...
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| **Package** | lowercase, single word | `lighting`, `climate`, `api` |
| **Exported type** | PascalCase | `Scene`, `DeviceRegistry` |
| **Unexported type** | camelCase | `sceneValidator`, `deviceCache` |
| **Exported function** | PascalCase | `ActivateScene`, `GetDeviceByID` |
| **Unexported function** | camelCase | `validateDevices`, `sendCommand` |
| **Interface** | PascalCase, often `-er` suffix | `SceneActivator`, `DeviceRepository` |
| **Constant** | PascalCase or UPPER_SNAKE | `MaxDevices`, `DEFAULT_TIMEOUT` |
| **Variable** | camelCase | `sceneID`, `deviceList` |

**Acronyms:**
- Use all caps for 2-letter acronyms: `ID`, `API`, `UI`
- Use PascalCase for longer: `Http`, `Json`, `Mqtt`

Examples:
- `DeviceID` ✅
- `DeviceId` ❌
- `HTTPClient` ✅
- `HttpClient` ✅ (acceptable)

---

## File Organization

### Project Structure

**This structure matches [core-internals.md](../architecture/core-internals.md):**

```
gray-logic-core/
├── cmd/
│   └── graylogic/             # Main application entry point
│       └── main.go
│
├── internal/                   # Private application code
│   ├── infrastructure/         # Infrastructure Layer
│   │   ├── api/                # REST API server
│   │   │   ├── server.go
│   │   │   ├── routes.go
│   │   │   ├── middleware/
│   │   │   └── handlers/
│   │   ├── websocket/          # WebSocket server
│   │   │   ├── server.go
│   │   │   ├── hub.go
│   │   │   └── client.go
│   │   ├── database/           # SQLite database
│   │   │   ├── database.go
│   │   │   ├── migrations/
│   │   │   └── queries/
│   │   ├── mqtt/               # MQTT client
│   │   │   ├── client.go
│   │   │   └── topics.go
│   │   ├── config/             # Configuration management
│   │   │   ├── config.go
│   │   │   └── loader.go
│   │   ├── auth/               # Authentication & authorization
│   │   │   ├── auth.go
│   │   │   ├── jwt.go
│   │   │   └── rbac.go
│   │   ├── logging/            # Structured logging
│   │   │   └── logger.go
│   │   └── metrics/            # Prometheus metrics
│   │       └── metrics.go
│   │
│   ├── device/                 # Device Layer
│   │   ├── registry/           # Device registry
│   │   │   ├── registry.go
│   │   │   └── device.go
│   │   ├── state/              # State management
│   │   │   ├── manager.go
│   │   │   ├── store.go
│   │   │   └── history.go
│   │   ├── command/            # Command processing
│   │   │   ├── processor.go
│   │   │   ├── validator.go
│   │   │   └── router.go
│   │   ├── association/        # Device associations
│   │   │   └── resolver.go
│   │   ├── discovery/          # Device discovery
│   │   │   └── discovery.go
│   │   └── health/             # PHM health monitoring
│   │       ├── monitor.go
│   │       ├── baseline.go
│   │       └── alerter.go
│   │
│   ├── automation/             # Automation Layer
│   │   ├── scene/              # Scene engine
│   │   │   ├── engine.go
│   │   │   ├── scene.go
│   │   │   └── action.go
│   │   ├── scheduler/          # Time-based triggers
│   │   │   ├── scheduler.go
│   │   │   ├── schedule.go
│   │   │   └── astronomical.go
│   │   ├── mode/               # Mode management
│   │   │   ├── manager.go
│   │   │   └── mode.go
│   │   ├── event/              # Event routing
│   │   │   ├── router.go
│   │   │   ├── event.go
│   │   │   └── handlers.go
│   │   └── logic/              # Conditional logic
│   │       ├── engine.go
│   │       ├── condition.go
│   │       └── evaluator.go
│   │
│   ├── intelligence/           # Intelligence Layer
│   │   ├── ai/                 # AI engine
│   │   │   ├── engine.go
│   │   │   └── llm.go
│   │   ├── voice/              # Voice processing
│   │   │   ├── processor.go
│   │   │   ├── stt.go          # Speech-to-text
│   │   │   ├── nlu.go          # Natural language understanding
│   │   │   └── tts.go          # Text-to-speech
│   │   ├── presence/           # Presence detection
│   │   │   ├── engine.go
│   │   │   └── tracker.go
│   │   └── learning/           # Pattern learning
│   │       ├── engine.go
│   │       └── patterns.go
│   │
│   └── domain/                 # Domain-specific logic
│       ├── lighting/
│       │   ├── service.go
│       │   └── circadian.go
│       ├── climate/
│       │   ├── service.go
│       │   ├── zone.go
│       │   └── adaptive.go
│       ├── blinds/
│       │   ├── service.go
│       │   └── suntracking.go
│       ├── audio/
│       │   └── service.go
│       ├── security/
│       │   └── service.go
│       └── energy/
│           ├── service.go
│           └── flows.go
│
├── pkg/                        # Public packages (reusable)
│   ├── models/                 # Data models
│   │   ├── site.go
│   │   ├── device.go
│   │   ├── scene.go
│   │   └── ...
│   ├── protocol/               # Protocol definitions
│   │   ├── knx/
│   │   ├── dali/
│   │   └── modbus/
│   └── mqtt/                   # MQTT message types
│       ├── messages.go
│       └── topics.go
│
├── migrations/                 # Database migrations
│   ├── 001_initial.sql
│   └── ...
│
├── configs/                    # Default configurations
│   ├── config.yaml
│   └── ...
│
├── tests/
│   ├── integration/            # Integration tests
│   └── fixtures/               # Test data
│
└── web/                        # Embedded web assets (optional)
    └── admin/
```

### File Naming

| Type | Convention | Example |
|------|------------|---------|
| **Source** | `lowercase_with_underscores.go` | `scene_service.go`, `device_registry.go` |
| **Test** | `<source>_test.go` | `scene_service_test.go` |
| **Mock** | `mock_<interface>.go` or `<interface>_mock.go` | `mock_repository.go` |
| **Documentation** | `doc.go` | `doc.go` (in each package) |

---

## Database Migrations

### Migration File Format

Database migrations are versioned SQL files:

```
migrations/
├── 20260115_120000_initial_schema.up.sql
├── 20260115_120000_initial_schema.down.sql
├── 20260120_143000_add_scene_executions.up.sql
├── 20260120_143000_add_scene_executions.down.sql
└── ...
```

**Naming convention:** `YYYYMMDD_HHMMSS_description.{up|down}.sql`

### Migration Rules

1. **Always provide both up and down** — Every migration must be reversible
2. **Idempotent where possible** — Use `IF NOT EXISTS`, `IF EXISTS`
3. **Small, focused migrations** — One logical change per migration
4. **Test migrations on copy** — Never test on production database
5. **Backup before migrate** — Automated by migration tool

### Example Migration

```sql
-- 20260120_143000_add_scene_executions.up.sql
-- Add scene execution tracking

CREATE TABLE IF NOT EXISTS scene_executions (
    id TEXT PRIMARY KEY,
    scene_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'partial', 'failed', 'cancelled')),
    triggered_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    trigger_type TEXT NOT NULL,
    trigger_user_id TEXT,
    actions_total INTEGER NOT NULL DEFAULT 0,
    actions_completed INTEGER NOT NULL DEFAULT 0,
    actions_failed INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scene_executions_scene_id ON scene_executions(scene_id);
CREATE INDEX IF NOT EXISTS idx_scene_executions_triggered_at ON scene_executions(triggered_at);
```

```sql
-- 20260120_143000_add_scene_executions.down.sql
-- Remove scene execution tracking

DROP INDEX IF EXISTS idx_scene_executions_triggered_at;
DROP INDEX IF EXISTS idx_scene_executions_scene_id;
DROP TABLE IF EXISTS scene_executions;
```

### Migration Workflow

```bash
# Check migration status
graylogic-ctl migrate status

# Apply pending migrations
graylogic-ctl migrate up

# Rollback last migration
graylogic-ctl migrate down 1

# Create new migration
graylogic-ctl migrate create add_feature_x
```

---

## Structured Logging

### Log Format

All components use JSON structured logging:

```json
{
  "timestamp": "2026-01-15T10:30:00.123Z",
  "level": "info",
  "component": "scene_engine",
  "msg": "Scene activated",
  "scene_id": "scene-cinema",
  "scene_name": "Cinema Mode",
  "trigger": "voice",
  "user_id": "usr-001",
  "actions_count": 8,
  "duration_ms": 245
}
```

### Required Fields

Every log entry MUST include:

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO8601 | When the event occurred (UTC) |
| `level` | string | Log level: debug, info, warn, error, fatal |
| `component` | string | Source component: core, api, scene_engine, bridge_knx, etc. |
| `msg` | string | Human-readable message |

### Context Fields

Add context relevant to the operation:

```go
log.Info("Command sent to device",
    "component", "bridge_knx",
    "device_id", device.ID,
    "command", cmd.Name,
    "address", device.Address,
    "latency_ms", latency.Milliseconds(),
)
```

### Sensitive Data

**NEVER log:**
- Passwords or secrets
- Full JWT tokens (log last 8 chars only: `token: ...abc12345`)
- PII unless required for audit
- Audio/video content
- Credit card numbers

**Redact when necessary:**
```go
log.Info("API key used",
    "key_id", key.ID,
    "key_prefix", key.Value[:8]+"...",  // Redacted
)
```

### Log Levels

| Level | Use Case | Example |
|-------|----------|---------|
| `debug` | Development tracing | "Received MQTT message", "Query executed" |
| `info` | Normal operations | "Scene activated", "Device registered" |
| `warn` | Recoverable issues | "Retry attempt 2/3", "Rate limit approaching" |
| `error` | Failures | "Database write failed", "Device unreachable" |
| `fatal` | Unrecoverable | "Database corrupted", "Critical config missing" |

---

## Timestamp and Timezone Handling

### UTC Everywhere Internally

**All timestamps stored and transmitted internally MUST be UTC.**

```go
// ✅ Good - Always use UTC for storage and transmission
createdAt := time.Now().UTC()

// ✅ Good - Parse incoming timestamps as UTC
t, err := time.Parse(time.RFC3339, "2026-01-15T10:30:00Z")

// ❌ Bad - Using local time for storage
createdAt := time.Now()  // Don't do this
```

### Display Timezone

Convert to local timezone only at the UI layer:

```go
// API response - always UTC (Z suffix)
type DeviceState struct {
    UpdatedAt time.Time `json:"updated_at"`  // "2026-01-15T10:30:00Z"
}

// UI layer converts for display
func formatForDisplay(t time.Time, tz *time.Location) string {
    return t.In(tz).Format("15:04 02/01/2006")  // "10:30 15/01/2026" for UK
}
```

### Site Timezone Configuration

Each site has a configured timezone for:
- Schedule evaluation (e.g., "turn on at 18:00" means 18:00 local)
- Astronomical calculations (sunrise/sunset for the location)
- Log display in admin interface

```yaml
site:
  timezone: "Europe/London"  # IANA timezone database
```

### Database Storage

```sql
-- All timestamps stored as UTC
CREATE TABLE device_state (
    updated_at TIMESTAMP NOT NULL  -- Always UTC, no timezone suffix needed
);

-- When inserting
INSERT INTO device_state (updated_at) VALUES (datetime('now'));  -- SQLite UTC
```

### Why UTC?

1. **No ambiguity** — No daylight saving time transitions
2. **Sortable** — Direct string comparison works
3. **Interoperable** — Standard for APIs and logs
4. **Debuggable** — One reference point for all timestamps

---

## Testing with Hardware

### Testing Strategy

Gray Logic uses a three-tier testing approach:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Unit Tests                                   │
│  - Mock interfaces                                                  │
│  - No external dependencies                                         │
│  - Fast (<1s per test)                                              │
│  - Run on every commit                                              │
└───────────────────────────────────┬─────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Integration Tests (Simulators)                    │
│  - Software simulators for protocols                                │
│  - Full component integration                                       │
│  - Medium speed (seconds per test)                                  │
│  - Run on PR/merge                                                  │
└───────────────────────────────────┬─────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  Hardware-in-Loop Tests (Real Devices)               │
│  - Actual KNX/DALI/Modbus devices                                   │
│  - Full end-to-end validation                                       │
│  - Slow (minutes per suite)                                         │
│  - Run before release / weekly                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Mock Interfaces

For unit testing, mock all external dependencies:

```go
// interfaces.go
type DeviceRepository interface {
    GetByID(ctx context.Context, id string) (*Device, error)
    Save(ctx context.Context, device *Device) error
}

type MQTTPublisher interface {
    Publish(topic string, payload []byte) error
}

// mock_repository.go
type MockDeviceRepository struct {
    Devices map[string]*Device
    SaveErr error
}

func (m *MockDeviceRepository) GetByID(ctx context.Context, id string) (*Device, error) {
    if d, ok := m.Devices[id]; ok {
        return d, nil
    }
    return nil, ErrDeviceNotFound
}
```

### Protocol Simulators

For integration testing without hardware:

```yaml
simulators:
  knx:
    tool: "knxd --fake"              # knxd can run in simulation mode
    config: "tests/fixtures/knx_simulation.yaml"
    devices:
      - address: "1/2/3"
        type: "switch"
        initial_state: "off"

  dali:
    tool: "dali-simulator"           # Custom or community simulator
    config: "tests/fixtures/dali_simulation.yaml"
    devices:
      - short_address: 0
        type: "led_driver"
        initial_level: 0

  modbus:
    tool: "diagslave"                # Modbus simulator
    config: "tests/fixtures/modbus_simulation.yaml"
    registers:
      - address: 40001
        value: 0
```

### Hardware Test Lab

For final validation, maintain a hardware test setup:

```yaml
test_lab:
  # Recommended test devices
  knx:
    - "ABB SA/S switch actuator"
    - "ABB UD/S dimmer"
    - "ABB sensor"

  dali:
    - "Tridonic gateway"
    - "2-3 DALI drivers"

  modbus:
    - "Eastron SDM630 energy meter"
    - "Any Modbus temperature sensor"

  # Test scenarios
  scenarios:
    - name: "Basic switching"
      steps:
        - "Send on command"
        - "Verify device state"
        - "Verify state in Core"

    - name: "Scene execution"
      steps:
        - "Activate scene with 5 devices"
        - "Verify all devices reached target state"
        - "Verify execution logged"
```

### When to Use Each

| Test Type | When to Run | Purpose |
|-----------|-------------|---------|
| Unit | Every commit | Catch logic errors |
| Simulator | Every PR | Catch integration issues |
| Hardware | Weekly / Pre-release | Catch protocol issues |

---

## Git Commit Standards

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat` — New feature
- `fix` — Bug fix
- `docs` — Documentation only
- `style` — Formatting, missing semi-colons, etc.
- `refactor` — Code change that neither fixes a bug nor adds a feature
- `perf` — Performance improvement
- `test` — Adding or updating tests
- `chore` — Maintenance (dependencies, build scripts)

**Scopes:**
- `lighting`, `climate`, `scenes`, `api`, `db`, `bridge`, `security`, etc.

**Examples:**

```
feat(lighting): add dimming transition support

Implements smooth dimming transitions for KNX and DALI devices.
Transition time is configurable per scene action.

Closes #42
```

```
fix(scenes): prevent concurrent scene activation race

Scene activation now uses mutex to prevent overlapping executions
that could cause devices to receive conflicting commands.

Fixes #87
```

```
docs(development): add coding standards document

Defines function documentation templates, error handling patterns,
and testing standards for Gray Logic development.
```

### Commit Size

**One logical change per commit:**
- ✅ "Add scene activation API"
- ✅ "Fix race condition in scene engine"
- ❌ "Add scenes, fix bugs, update docs" (too broad)

**Small commits preferred:**
- Easier to review
- Easier to revert
- Easier to bisect bugs

---

## Dependency Management

### Vendoring

Use Go modules with vendoring:

```bash
go mod vendor
```

**Commit `vendor/` directory:**
- Ensures reproducible builds in 10+ years
- No dependency on external package registries

### Dependency Selection

**Prefer:**
- Mature libraries (5+ years old)
- Active maintenance (commit in last 6 months)
- Permissive licenses (MIT, Apache 2.0, BSD)
- Minimal transitive dependencies

**Avoid:**
- Abandoned libraries
- GPL-incompatible licenses (Gray Logic is GPL v3)
- Heavy frameworks (we want small binary)

**Audit dependencies:**
```bash
go list -m all
go mod graph
govulncheck ./...
```

---

## Security Coding Practices

### Input Validation

Validate all external inputs:

```go
func CreateScene(req *CreateSceneRequest) (*Scene, error) {
    // Validate name
    if req.Name == "" {
        return nil, errors.New("scene name is required")
    }
    if len(req.Name) > 50 {
        return nil, errors.New("scene name too long (max 50 chars)")
    }

    // Validate transition time
    if req.TransitionTimeMs < 0 || req.TransitionTimeMs > 60000 {
        return nil, errors.New("transition time must be 0-60000ms")
    }

    // Sanitize name (prevent injection attacks if rendered in UI)
    req.Name = html.EscapeString(req.Name)

    // Continue with scene creation...
}
```

### SQL Injection Prevention

Always use parameterized queries:

❌ **Bad:**
```go
query := fmt.Sprintf("SELECT * FROM scenes WHERE name = '%s'", name)
db.Query(query)
```

✅ **Good:**
```go
db.Query("SELECT * FROM scenes WHERE name = ?", name)
```

### Command Injection Prevention

Never pass unsanitized input to shell:

❌ **Bad:**
```go
exec.Command("sh", "-c", "echo "+userInput)
```

✅ **Good:**
```go
// Don't use shell; use direct command execution
exec.Command("echo", userInput)

// Or validate/sanitize input
if !isAlphanumeric(userInput) {
    return errors.New("invalid input")
}
```

### Secret Management

**Never hardcode secrets:**
```go
// ❌ Bad
const apiKey = "sk_live_abc123"

// ✅ Good
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    return errors.New("API_KEY not set")
}
```

**Use environment variables or secure vaults.**

---

## Documentation Standards

### README.md (per package)

Optional but recommended for complex packages:

```markdown
# Lighting Package

Implements lighting control for Gray Logic.

## Features

- Device registry for lights, dimmers, switches
- Scene recall with transitions
- KNX and DALI protocol support

## Usage

\`\`\`go
import "gray-logic/internal/domain/lighting"

service := lighting.NewService(repo, bridge)
err := service.ActivateScene(ctx, sceneID, nil)
\`\`\`

## Testing

\`\`\`bash
go test ./internal/domain/lighting/...
\`\`\`

## Related Docs

- [Lighting Domain Spec](../../../docs/domains/lighting.md)
- [KNX Protocol](../../../docs/protocols/knx.md)
```

### External Documentation

Keep `docs/` in sync with code:
- Update domain docs when adding features
- Update protocol docs when adding support
- Update architecture docs on major refactors

**Link from code to docs:**
```go
// Scene represents a lighting scene.
//
// For scene design philosophy and business rules, see:
// docs/domains/lighting.md#scenes
type Scene struct {
    // ...
}
```

---

## Changelog

Maintain `CHANGELOG.md` in [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
# Changelog

## [Unreleased]

### Added
- Scene transition time configuration

### Fixed
- Race condition in scene activation

## [0.2.0] - 2026-01-10

### Added
- DALI bridge support
- Multi-room scenes

### Changed
- Improved KNX device discovery performance

### Fixed
- Scene activation timeout handling
```

---

## Related Documents

- [Development Strategy](DEVELOPMENT-STRATEGY.md) — Overall development approach and phased plan
- [Security Checklist](SECURITY-CHECKLIST.md) — Security verification and release gates
- [Principles](../overview/principles.md) — Hard rules governing development
- [Core Internals](../architecture/core-internals.md) — Technical architecture
