# AI Agent Guidance — Gray Logic Core (Go)

> **Gray Logic Core development guidance for AI coding agents.**
>
> This file supplements the root `AGENTS.md` and assumes you have read `docs/CONSTRAINTS.md`.

---

## Prerequisites

Before working on Gray Logic Core, you MUST read:

1. **[docs/CONSTRAINTS.md](../../docs/CONSTRAINTS.md)** — Canonical rules (mandatory)
2. **[docs/overview/principles.md](../../docs/overview/principles.md)** — Foundational philosophy
3. **[docs/development/CODING-STANDARDS.md](../../docs/development/CODING-STANDARDS.md)** — Full coding standards

---

## Quick Reference

### Critical Boundaries (Summary)

These rules CANNOT be broken. See `docs/CONSTRAINTS.md` for full details.

1. **Physical controls always work** — Software failure ≠ building failure
2. **Life safety is independent** — Fire/E-stop systems are read-only
3. **No cloud dependencies** — 99%+ functionality works without internet
4. **Multi-decade horizon** — Systems deployed today work in 2046
5. **Open standards** — KNX, DALI, Modbus at field layer
6. **Customer owns system** — No dealer locks
7. **Privacy by design** — Voice/AI processed locally

### Project Structure

```
code/core/
├── cmd/
│   └── graylogic/          # Main entry point
│       └── main.go
├── internal/               # Private packages
│   ├── infrastructure/     # DB, MQTT, InfluxDB, HTTP, Config
│   ├── device/             # Device registry, state, commands
│   ├── automation/         # Scenes, schedules, modes, events
│   ├── intelligence/       # Voice, PHM, AI
│   └── domain/             # Core domain types
├── pkg/                    # Public packages
│   ├── models/             # Shared types
│   └── protocols/          # Protocol definitions
├── migrations/             # SQL migration files
├── configs/                # Configuration templates
└── tests/
    └── integration/        # Integration test suites
```

---

## Go Coding Standards

### 1. Documentation-First

Every package MUST have a `doc.go`:

```go
// Package lighting implements lighting control for Gray Logic.
//
// This package manages:
// - Device registry for lights, dimmers, and switches
// - Scene execution with transitions
// - Protocol-agnostic communication via MQTT
//
// # Security
//
// All commands are authorized before execution.
// Device states are validated before persistence.
//
// # Performance
//
// Scene execution targets <500ms for 10 devices.
// State updates are non-blocking.
//
// See docs/domains/lighting.md for domain specification.
package lighting
```

Every exported function MUST be documented:

```go
// ActivateScene activates the specified scene.
//
// The function loads the scene definition, validates all target devices,
// and publishes commands to the appropriate protocol bridges via MQTT.
//
// Parameters:
//   - ctx: Context for cancellation and timeout (30s recommended)
//   - sceneID: Unique scene identifier (slug format)
//   - options: Optional activation parameters (may be nil)
//
// Returns:
//   - error: nil on success, or:
//     - ErrSceneNotFound if scene doesn't exist
//     - ErrDeviceOffline if any target device is unreachable
//     - ErrPermissionDenied if caller lacks scenes:execute permission
//
// Security:
//
//	Caller must have 'scenes:execute' permission.
//	Scene activation is logged in audit trail.
//
// Example:
//
//	err := service.ActivateScene(ctx, "cinema-mode", nil)
//	if errors.Is(err, lighting.ErrSceneNotFound) {
//	    return fmt.Errorf("scene not found: %w", err)
//	}
func (s *Service) ActivateScene(ctx context.Context, sceneID string, options *ActivateOptions) error
```

### 2. Error Handling

Define domain-specific errors:

```go
// Domain errors for lighting package
var (
    ErrSceneNotFound   = errors.New("scene not found")
    ErrDeviceOffline   = errors.New("device offline")
    ErrInvalidState    = errors.New("invalid state value")
)
```

Wrap errors with context:

```go
if err := s.repo.GetByID(ctx, id); err != nil {
    return fmt.Errorf("lighting: failed to load scene %q: %w", id, err)
}
```

### 3. Testing Requirements

> For the full testing strategy including tiers, build tags, and coverage targets,
> see `docs/technical/testing-strategy.md`.

- **80% coverage minimum** for new code
- **100% coverage** for security-critical and error paths
- Use **table-driven tests**:

```go
func TestActivateScene(t *testing.T) {
    tests := []struct {
        name      string
        sceneID   string
        setupRepo func(*MockRepository)
        wantErr   error
    }{
        {
            name:    "valid scene activates successfully",
            sceneID: "cinema-mode",
            setupRepo: func(r *MockRepository) {
                r.AddScene("cinema-mode", validScene)
            },
            wantErr: nil,
        },
        {
            name:    "missing scene returns ErrSceneNotFound",
            sceneID: "nonexistent",
            setupRepo: func(r *MockRepository) {},
            wantErr: lighting.ErrSceneNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... test body
        })
    }
}
```

### 4. Concurrency Safety

Protect shared state:

```go
type DeviceRegistry struct {
    mu      sync.RWMutex
    devices map[string]*Device
}

func (r *DeviceRegistry) Get(id string) (*Device, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    d, ok := r.devices[id]
    return d, ok
}

func (r *DeviceRegistry) Update(id string, state State) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    // ... update logic
}
```

### 5. UK English Spelling

Use UK English throughout all code, comments, and documentation:

| ✅ UK (correct) | ❌ US (wrong) |
|-----------------|---------------|
| initialise | initialize |
| colour | color |
| behaviour | behavior |
| metre | meter |
| cancelled | canceled |
| organisation | organization |
| serialise | serialize |
| synchronise | synchronize |

golangci-lint with `misspell` linter is configured to catch US spellings.
```bash
# Check for spelling issues
golangci-lint run --enable=misspell
```

### 6. Context Usage

All long-running or cancellable operations MUST accept context as first parameter:
```go
// ✅ Correct — context as first parameter
func (s *Service) LoadDevices(ctx context.Context) ([]Device, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Pass context to downstream calls
    devices, err := s.repo.FindAll(ctx)
    if err != nil {
        return nil, fmt.Errorf("load devices: %w", err)
    }
    return devices, nil
}

// ❌ Wrong — no context support
func (s *Service) LoadDevices() ([]Device, error)
```

**Rules:**
- Pass context through the entire call chain
- Never store context in structs
- Use `context.WithTimeout` for operations with deadlines
- Check `ctx.Done()` in loops and long operations

### 7. Graceful Shutdown

Use `signal.NotifyContext` for OS signal handling. Close resources in **reverse order** of creation (LIFO):
```go
func main() {
    // Setup signal handling
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    // Open resources in order
    db, err := database.Open(cfg.Database)
    if err != nil {
        return fmt.Errorf("database: %w", err)
    }
    defer db.Close() // Closed last

    mqtt, err := mqtt.Connect(cfg.MQTT)
    if err != nil {
        return fmt.Errorf("mqtt: %w", err)
    }
    defer mqtt.Close() // Closed second

    influx, err := influxdb.Connect(cfg.InfluxDB)
    if err != nil {
        return fmt.Errorf("influxdb: %w", err)
    }
    defer influx.Close() // Closed first

    // Start services...

    // Wait for shutdown signal
    <-ctx.Done()
    log.Info("shutting down gracefully")
    return nil
}
```

**Rules:**
- `defer Close()` immediately after successful `Open()`
- Defers execute in LIFO order (last defer runs first)
- Log shutdown initiation for debugging
- Give connections time to drain if needed (use timeout context)

---

## Database Rules

### Additive-Only Migrations

```sql
-- ✅ Allowed: Add new nullable column
ALTER TABLE devices ADD COLUMN location TEXT;

-- ✅ Allowed: Add new column with default
ALTER TABLE devices ADD COLUMN protocol TEXT NOT NULL DEFAULT 'knx';

-- ❌ PROHIBITED: Drop column
ALTER TABLE devices DROP COLUMN old_field;

-- ❌ PROHIBITED: Rename column (add new instead)
ALTER TABLE devices RENAME COLUMN name TO display_name;
```

### Migration Naming

```
migrations/
├── 001_initial_schema.sql
├── 002_add_device_location.sql
└── 003_add_scene_transitions.sql
```

### Rollback Test (REQUIRED)

Every migration must pass:
1. Migrate from A to B
2. Add data using B
3. Run A binary against B database
4. A must operate normally

---

## MQTT Integration

### Publishing State Updates

```go
topic := fmt.Sprintf("graylogic/state/%s/%s", 
    device.Protocol,
    url.PathEscape(device.Address))

payload := StateMessage{
    DeviceID:  device.ID,
    Timestamp: time.Now().UTC(),
    State:     currentState,
    Protocol:  device.Protocol,
    Address:   device.Address,
}

if err := client.Publish(topic, payload, mqtt.QoS1, false); err != nil {
    log.Error("failed to publish state", "device", device.ID, "error", err)
}
```

### Handling Commands

```go
client.Subscribe("graylogic/command/+/+", func(msg mqtt.Message) {
    var cmd CommandMessage
    if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
        log.Error("invalid command message", "error", err)
        return
    }
    
    // Always acknowledge immediately
    ack := AckMessage{
        CommandID: cmd.ID,
        Status:    "accepted",
        Timestamp: time.Now().UTC(),
    }
    client.Publish(ackTopic(cmd), ack, mqtt.QoS1, false)
    
    // Execute asynchronously
    go executeCommand(cmd)
})
```

### Health Reporting

Publish health every 30 seconds:

```go
func (b *Bridge) publishHealth() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        health := HealthMessage{
            Bridge:    b.ID,
            Status:    b.getStatus(),
            Timestamp: time.Now().UTC(),
            // ... metrics
        }
        b.mqtt.Publish(healthTopic, health, mqtt.QoS1, true) // retained
    }
}
```

---

## Security Patterns

### Input Validation

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
    
    // Sanitize for UI safety
    req.Name = html.EscapeString(req.Name)
    
    // Continue...
}
```

### Parameterized Queries (ALWAYS)

```go
// ✅ Correct
db.Query("SELECT * FROM devices WHERE id = ?", deviceID)

// ❌ NEVER DO THIS
db.Query(fmt.Sprintf("SELECT * FROM devices WHERE id = '%s'", deviceID))
```

### Secrets from Environment

```go
// ✅ Correct
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    return errors.New("JWT_SECRET not set")
}

// ❌ NEVER hardcode secrets
const jwtSecret = "my-secret-key"
```

### Never Log Secrets

```go
// ✅ Correct - redacted
log.Info("API key used", "key_prefix", key[:8]+"...")

// ❌ NEVER log full secrets
log.Info("API key used", "key", key)
```

---

## Common Gotchas

### 1. MQTT Tests Need Broker

MQTT tests require a running broker. Start with Docker Compose:

```bash
docker compose -f docker-compose.dev.yml up -d
go test -tags=integration ./internal/infrastructure/mqtt/...
```

Or use the `--short` flag to skip integration tests:

```bash
go test -short ./...
```

### 2. SQLite Requires CGO

SQLite driver requires CGO:

```bash
CGO_ENABLED=1 go build ./cmd/graylogic
```

For cross-compilation, use the appropriate toolchain.

### 3. Config Location

Development config: `./configs/dev.yaml`
Production config: `/etc/graylogic/config.yaml`

### 4. Timestamps Always UTC

```go
// ✅ Correct
createdAt := time.Now().UTC()

// ❌ Wrong - uses local timezone
createdAt := time.Now()
```

---

## Commands

```bash
# Build
go build -o bin/graylogic ./cmd/graylogic

# Run tests
go test -v ./...

# Run with coverage
go test -cover -coverprofile=coverage.out ./...

# Lint
golangci-lint run

# Security audit
govulncheck ./...

# Pre-commit check
make precommit
```

---

## Key Files to Understand

| File | Purpose |
|------|---------|
| `cmd/graylogic/main.go` | Application entry point |
| `internal/infrastructure/config/` | Configuration loading |
| `internal/infrastructure/database/` | SQLite connection and migrations |
| `internal/infrastructure/mqtt/` | MQTT client wrapper |
| `internal/infrastructure/influxdb/` | InfluxDB client for PHM |

---

## Protocol References

When working on protocol bridges, you MUST consult the relevant specification:

| Protocol | Reference Document | Required For |
|----------|-------------------|--------------|
| **KNX** | [docs/protocols/knx-reference.md](../../docs/protocols/knx-reference.md) | Any `internal/bridges/knx/` or `internal/knxd/` work |
| **MQTT** | [docs/protocols/mqtt.md](../../docs/protocols/mqtt.md) | Topic structure, QoS, message formats |

**Rule:** Never implement protocol behaviour from memory or guesswork. Always verify against the specification document.

Each bridge package has its own `AGENTS.md` with package-specific guidance:
- `internal/bridges/knx/AGENTS.md` — KNX-specific rules and spec compliance checklist

---

## Related Documentation

- **[docs/CONSTRAINTS.md](../../docs/CONSTRAINTS.md)** — All constraints (MUST READ)
- **[docs/protocols/knx-reference.md](../../docs/protocols/knx-reference.md)** — KNX protocol specification
- **[docs/development/CODING-STANDARDS.md](../../docs/development/CODING-STANDARDS.md)** — Full standards
- **[docs/architecture/core-internals.md](../../docs/architecture/core-internals.md)** — Core architecture
- **[PROJECT-STATUS.md](../../PROJECT-STATUS.md)** — Current implementation status and session log
