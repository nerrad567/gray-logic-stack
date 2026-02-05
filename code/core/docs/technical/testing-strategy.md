---
title: Testing Strategy — Gray Logic Core
version: 1.0.0
status: active
last_updated: 2026-01-23
references:
  - code/core/AGENTS.md
  - .claude/commands/milestone-audit.md
  - .claude/commands/code-audit.md
---

# Testing Strategy — Gray Logic Core

## Overview

Tests are the safety net for a multi-decade deployment system. Gray Logic Core must run reliably for 20+ years in production buildings, controlling lighting, climate, blinds, and security. A bug in this system means lights that don't turn on, heating that runs all night, or blinds that won't open.

### Philosophy

- **Standard library only**: No testify, no assertion libraries. `testing` package with `t.Errorf`, `t.Fatalf`, `errors.Is`. This ensures zero external test dependencies that could break over decades.
- **Deterministic**: Tests must produce the same result every run. No reliance on system time, random values, or network availability (except integration tier).
- **Self-documenting**: Test names describe the scenario. Table-driven tests enumerate all cases.
- **Fast by default**: Unit tests run in <30s. Integration tests are opt-in via build tags.
- **Coverage enforced**: Each package has a minimum coverage target based on its risk profile.

---

## Test Tiers

Gray Logic uses four test tiers, gated by Go build tags:

| Tier | Build Tag | When Run | Requirements | Duration |
|------|-----------|----------|--------------|----------|
| **Unit** | *(none)* | Every commit, CI, `/pre-commit` | Nothing | <30s |
| **Integration** | `//go:build integration` | `/milestone-audit`, CI with Docker | docker-compose.dev.yml healthy | <2min |
| **Hardware** | `//go:build hardware` | On-site / test bench | Real KNX/DALI interface + physical bus | <5min |
| **Endurance** | `//go:build endurance` | Pre-release soak test | Real hardware + extended time window | 24h+ |

### Tier Progression

```
Development:     Unit tests only (fast feedback)
Milestone:       Unit + Integration (coverage gates)
Pre-deployment:  Unit + Integration + Hardware (real protocols)
Pre-release:     All tiers including Endurance (stability proof)
```

---

## Coverage Targets

Coverage targets are risk-based. Higher-risk code (user-facing, state-changing, security-related) requires more thorough testing.

| Tier | Target (unit only) | Target (with hardware) | Packages | Rationale |
|------|-------------------|------------------------|----------|-----------|
| **Critical** | 85%+ | 85%+ | `internal/api`, `internal/device` | User-facing, state-changing, auth |
| **Infrastructure** | 75%+ | 75%+ | `internal/infrastructure/*`, `internal/knxd` | Foundation stability |
| **Bridge** | 50%+ | 70%+ | `internal/bridges/knx`, future DALI/Modbus | Hardware-dependent paths |
| **Integration** | 40%+ | 40%+ | `cmd/graylogic`, `internal/process` | Startup/shutdown wiring |
| **Exempt** | — | — | `migrations/`, `doc.go` files | No logic to test |

### Why These Numbers?

- **85% Critical**: Ensures all business logic, error paths, and auth flows are exercised. The remaining 15% covers transport-level code (TLS handshakes, WebSocket upgrades) better tested by integration.
- **75% Infrastructure**: Database, MQTT, and config are well-abstracted. Most logic is testable with in-memory backends.
- **50% Bridge (no hardware)**: Protocol bridges have significant hardware-dependent code. Mock-based tests cover the message handling and state management, but real protocol verification needs physical devices.
- **70% Bridge (with hardware)**: When KNX hardware is available, we can test telegram encoding/decoding, device communication, and reconnection behaviour.
- **40% Integration**: `main.go` and process manager are primarily wiring code. Testing them requires full service startup, which is covered by integration tests.

---

## How to Run Each Tier

```bash
cd code/core
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# --- Unit Tests (default, always safe to run) ---
go test -race -cover ./...

# Verbose with individual test names:
go test -race -cover -v ./...

# Single package:
go test -race -cover -v ./internal/api/...

# Coverage report (file):
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out        # Per-function breakdown
go tool cover -html=coverage.out         # Opens browser with visual coverage

# --- Integration Tests (requires Docker services) ---
# First, ensure services are running:
docker compose -f ../../docker-compose.dev.yml up -d

# Verify health:
mosquitto_pub -h localhost -p 1883 -t test -m "ping" && echo "MQTT: OK"
curl -sf http://localhost:8428/health && echo "VictoriaMetrics: OK"

# Run integration tests:
go test -tags=integration -race -cover -v ./...

# --- Hardware Tests (requires KNX interface) ---
export GRAYLOGIC_HARDWARE_TEST=true
go test -tags=hardware -v ./internal/bridges/knx/...

# --- Endurance Tests (long-running, requires hardware) ---
export GRAYLOGIC_ENDURANCE_TEST=true
go test -tags=endurance -timeout=25h -v ./...
```

---

## Docker Compose Requirements

Integration tests require the dev services stack:

| Service | Image | Port | Purpose | Health Check |
|---------|-------|------|---------|-------------|
| Mosquitto | `eclipse-mosquitto:2.0.18` | 1883 (MQTT), 9001 (WS) | MQTT broker | `mosquitto_pub -t test -m hello` |
| VictoriaMetrics | `victoriametrics/victoria-metrics:v1.135.0` | 8428 | Time-series for PHM/energy | `curl -s http://localhost:8428/health` |

### Starting Services

```bash
# From repository root:
docker compose -f docker-compose.dev.yml up -d

# Check status:
docker compose -f docker-compose.dev.yml ps

# View logs:
docker compose -f docker-compose.dev.yml logs -f mosquitto

# Stop services:
docker compose -f docker-compose.dev.yml down
```

### Default Credentials (Dev Only)

| Service | Username | Password | Notes |
|---------|----------|----------|-------|
| VictoriaMetrics | graylogic | dev-password-only | Set in docker-compose env vars |
| MQTT | *(none)* | *(none)* | Anonymous access for dev |

---

## Hardware Test Setup

For the hardware test tier, physical equipment is required:

| Component | Purpose | Notes |
|-----------|---------|-------|
| KNX USB interface | knxd connectivity | Weinzierl KNX USB Interface 332 or similar |
| KNX PSU | Bus power | 29VDC, 640mA minimum |
| KNX switching actuator | End-to-end command test | Any 1-channel actuator |
| KNX dimming actuator | Dimming command test | Any DALI/1-10V dimmer |
| knxd daemon | Bridge to KNX bus | Managed by process manager or standalone |

### Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `GRAYLOGIC_HARDWARE_TEST` | Enable hardware tier | `true` |
| `KNX_TEST_INTERFACE` | USB device path | `/dev/ttyACM0` |
| `KNX_TEST_PHYSICAL_ADDR` | knxd physical address | `1.0.1` |
| `KNX_TEST_GROUP_ADDR` | Test group address (switching) | `1/1/1` |
| `KNX_TEST_DIM_ADDR` | Test group address (dimming) | `1/1/2` |
| `KNXD_SOCKET` | knxd socket path | `/var/run/knxd` |

### What Hardware Tests Verify

- Telegram send/receive (switching on/off)
- Group address read requests
- Device descriptor read (individual addressing)
- Dimming commands (0-100%)
- Reconnection after cable pull
- Bus monitor frame parsing

---

## Test Patterns

### Table-Driven Tests

The standard pattern for testing multiple cases:

```go
func TestValidateName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr error
    }{
        {name: "valid name", input: "Living Room Dimmer", wantErr: nil},
        {name: "empty name", input: "", wantErr: ErrInvalidName},
        {name: "too long", input: strings.Repeat("x", 256), wantErr: ErrInvalidName},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateName(tt.input)
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("ValidateName(%q) = %v, want %v", tt.input, err, tt.wantErr)
            }
        })
    }
}
```

### Interface Mocking with Structs

No mock frameworks. Define mock structs that implement interfaces:

```go
type MockRepository struct {
    mu        sync.Mutex
    devices   map[string]*Device
    createErr error // Inject errors for testing
}

func (m *MockRepository) GetByID(_ context.Context, id string) (*Device, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if d, ok := m.devices[id]; ok {
        return d, nil
    }
    return nil, ErrDeviceNotFound
}

func (m *MockRepository) SetCreateError(err error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.createErr = err
}
```

### Test Helpers

Use `t.Helper()` for helper functions and `t.Cleanup()` for teardown:

```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("open test db: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}
```

### Concurrent Tests

Use `sync.WaitGroup` and race detector (`go test -race`):

```go
func TestRegistry_ConcurrentAccess(t *testing.T) {
    registry := setupTestRegistry(t)

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            _ = registry.SetDeviceState(ctx, "dev-1", State{"count": n})
        }(i)
    }
    wg.Wait()

    dev, err := registry.GetDevice(ctx, "dev-1")
    if err != nil {
        t.Fatalf("GetDevice after concurrent writes: %v", err)
    }
    if dev.State == nil {
        t.Error("expected non-nil state after concurrent writes")
    }
}
```

### Build Tag Gating

Integration tests use build tags as the first line:

```go
//go:build integration

package mqtt

import (
    "context"
    "testing"
    "time"
)

func TestMQTT_RealBrokerConnect(t *testing.T) {
    // This test requires a real MQTT broker at localhost:1883
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := Connect(ctx, Config{Host: "localhost", Port: 1883})
    if err != nil {
        t.Fatalf("Connect to real broker: %v", err)
    }
    t.Cleanup(func() { client.Close() })

    // Verify publish/subscribe round-trip
    // ...
}
```

### Context with Timeout

All integration tests should use bounded contexts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### In-Memory SQLite

For persistence tests that don't need Docker:

```go
db, err := sql.Open("sqlite3", ":memory:")
```

Or use `t.TempDir()` for file-based SQLite with WAL mode:

```go
tmpDir := t.TempDir()
db, err := Open(ctx, Config{Path: filepath.Join(tmpDir, "test.db"), WALMode: true})
```

---

## Writing New Tests

### Checklist

- [ ] Use standard library `testing` package only
- [ ] Follow file naming: `{source}_test.go` or `integration_test.go`
- [ ] Use `t.Helper()` in all helper functions
- [ ] Use `t.Cleanup()` for resource teardown
- [ ] Use `t.Parallel()` where safe (no shared mutable state)
- [ ] Name tests: `TestFunctionName_Scenario` (e.g., `TestGetDevice_NotFound`)
- [ ] Use `errors.Is()` for error comparison, not `==`
- [ ] Compare struct fields explicitly (avoid `reflect.DeepEqual`)
- [ ] Use UK English in test names and comments
- [ ] Run with `-race` flag before committing
- [ ] Add `//go:build integration` tag if test requires external services

### What Makes a Good Test

- **Tests behaviour, not implementation** — assert on outputs and side effects, not internal state
- **One assertion per subtest** — makes failures easy to diagnose
- **Descriptive failure messages** — include got vs want: `t.Errorf("GetDevice(%q) = %v, want %v", id, got, want)`
- **No `time.Sleep` in unit tests** — use channels, contexts, or polling helpers
- **Idempotent** — running twice produces the same result

### What NOT to Test

- Trivial getters/setters with no logic
- Code that simply delegates to another function without transformation
- Third-party library internals
- Generated code

---

## Integration Test Guidelines

### Rules

1. **First line must be the build tag**: `//go:build integration`
2. **Check availability early**: Skip if services are unavailable
   ```go
   func TestIntegration_MQTT(t *testing.T) {
       if _, err := net.DialTimeout("tcp", "localhost:1883", 2*time.Second); err != nil {
           t.Skip("MQTT broker unavailable, skipping integration test")
       }
       // ...
   }
   ```
3. **Use unique identifiers**: Avoid MQTT client ID conflicts
   ```go
   clientID := fmt.Sprintf("test-%s-%d", t.Name(), time.Now().UnixNano())
   ```
4. **Clean up test data**: Unsubscribe topics, delete test records
5. **Use generous timeouts**: 5-30s for network operations
6. **Document requirements**: Comment block at top of file
   ```go
   // This file requires:
   //   - MQTT broker at localhost:1883 (docker compose -f docker-compose.dev.yml up -d)
   //   - No authentication configured on broker
   ```

### Cross-Package Integration Flows to Test

| Flow | Packages Involved | What to Verify |
|------|-------------------|----------------|
| Device command | api → mqtt → bridge | MQTT message arrives with correct topic/payload |
| State broadcast | bridge → mqtt → api → websocket | WebSocket client receives state change event |
| Device lifecycle | api → device → database | Create/update/delete persists and cache updates |
| Health monitoring | bridge → mqtt → device | Health status updates propagate to registry |

---

## Milestone Audit Process

The `/milestone-audit` command enforces these testing standards at the end of each milestone:

1. Sweeps all packages in dependency order
2. Measures current coverage against tier targets
3. Writes missing tests to fill gaps
4. Runs integration tests (if Docker available)
5. Scans technical debt (TODOs, FIXMEs)
6. Produces a milestone readiness report

See `.claude/commands/milestone-audit.md` for the full procedure.

---

## Related Documentation

- `code/core/AGENTS.md` — Go development guidance and coding standards
- `.claude/commands/milestone-audit.md` — End-of-milestone quality gate
- `.claude/commands/code-audit.md` — Per-package bug-finding audit
- `.claude/commands/pre-commit.md` — Quick pre-commit checks
- `docker-compose.dev.yml` — Dev services (MQTT, VictoriaMetrics)
- `code/core/test-reports/TEST-AUDIT.md` — Historical test results
