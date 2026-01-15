---
title: Coding Standards
version: 1.0.0
status: active
last_updated: 2026-01-15
depends_on:
  - development/DEVELOPMENT-STRATEGY.md
  - overview/principles.md
---

# Gray Logic Coding Standards

This document defines how we write, document, and test Gray Logic code.

---

## Philosophy

### Documentation-First

**Every module, every function, every non-trivial block must be documented.**

Why?
1. **10-year horizon** — Future maintainers (including future you) need to understand intent
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
    "gray-logic/internal/lighting"
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
// +build integration

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

```
gray-logic-core/
├── cmd/
│   └── core/              # Main application entry point
│       └── main.go
├── internal/              # Private application code
│   ├── api/               # REST API handlers
│   ├── websocket/         # WebSocket server
│   ├── lighting/          # Lighting domain
│   ├── climate/           # Climate domain
│   ├── scenes/            # Scene engine
│   ├── devices/           # Device registry
│   ├── bridge/            # Bridge interface
│   └── db/                # Database layer
├── pkg/                   # Public libraries (reusable)
│   ├── mqtt/              # MQTT client
│   └── knx/               # KNX protocol utilities
├── tests/
│   ├── integration/       # Integration tests
│   └── fixtures/          # Test data
├── docs/                  # Documentation (see docs/)
├── scripts/               # Build and deployment scripts
├── configs/               # Configuration file templates
└── go.mod
```

### File Naming

| Type | Convention | Example |
|------|------------|---------|
| **Source** | `lowercase_with_underscores.go` | `scene_service.go`, `device_registry.go` |
| **Test** | `<source>_test.go` | `scene_service_test.go` |
| **Mock** | `mock_<interface>.go` or `<interface>_mock.go` | `mock_repository.go` |
| **Documentation** | `doc.go` | `doc.go` (in each package) |

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
- Ensures reproducible builds in 10 years
- No dependency on external package registries

### Dependency Selection

**Prefer:**
- Mature libraries (5+ years old)
- Active maintenance (commit in last 6 months)
- Permissive licenses (MIT, Apache 2.0, BSD)
- Minimal transitive dependencies

**Avoid:**
- Abandoned libraries
- GPL licenses (incompatible with dual-licensing goals)
- Heavy frameworks (we want small binary)

**Audit dependencies:**
```bash
go list -m all
go mod graph
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
import "gray-logic/internal/lighting"

service := lighting.NewService(repo, bridge)
err := service.ActivateScene(ctx, sceneID, nil)
\`\`\`

## Testing

\`\`\`bash
go test ./internal/lighting/...
\`\`\`

## Related Docs

- [Lighting Domain Spec](../../docs/domains/lighting.md)
- [KNX Protocol](../../docs/protocols/knx.md)
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
