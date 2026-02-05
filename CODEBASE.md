# CODEBASE.md — Gray Logic Stack

> Sir's working reference for the gray-logic-stack codebase.
> **Deep dive completed 2026-02-01** — Read ~65KB of docs + ~15KB of core implementation code.

---

## Dev Workflow & Model Strategy

### Cost Model (as of 2026-02)

| Model | Cost | When to Use |
|-------|------|-------------|
| **Claude Opus (me)** | Flat £90/mo Max sub | **Primary** — already paid for |
| **Claude Sonnet** | Flat £90/mo Max sub | Faster Claude tasks |
| **GPT-5.2 via Codex** | $1.75/1M in, $14/1M out | Parallel work, second opinions, Codex tooling |
| **GPT-5-mini** | $0.25/1M in, $2/1M out | Avoid — if paying, use 5.2 |

### Decision Tree

```
Task comes in
    │
    ├─► Default: I handle it (Claude - flat rate)
    │
    ├─► Need parallelism?
    │       └─► Spawn Codex for task B while I do task A
    │
    ├─► I'm stuck / want different perspective?
    │       └─► Try Codex (GPT-5.2)
    │
    └─► Well-defined task needing fast iteration?
            └─► Codex --full-auto (GPT-5.2)
```

---

## Project Overview

**Gray Logic** is a building intelligence/BMS platform. Rivals Crestron, Savant, Control4, Loxone — but open, offline-first, zero vendor lock-in.

### Hard Rules (NEVER BREAK)

1. **Physical controls always work** — Wall switches function without software
2. **Life safety is independent** — Fire alarms are read-only, never controlled
3. **No cloud dependencies** — 99%+ functionality works offline
4. **Multi-decade horizon** — Deployed systems work in 2046
5. **Open standards** — KNX, DALI, Modbus at field layer
6. **Customer owns system** — No dealer locks
7. **Privacy by design** — Voice/AI processed locally

### Tech Stack

| Component | Tech | Notes |
|-----------|------|-------|
| Core | Go 1.25+ | Single binary, CGO for SQLite |
| Database | SQLite (WAL mode) | Strict mode, additive-only migrations |
| Time-series | VictoriaMetrics | PHM data, energy monitoring |
| Message Bus | MQTT (Mosquitto) | Internal comms, QoS 1 |
| UI | Flutter | Wall panels + mobile |
| Voice | Whisper (STT), Piper (TTS) | Local processing |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER INTERFACES                              │
│   Wall Panels │ Mobile App │ Voice │ Web Admin │ Remote (VPN)       │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ REST API + WebSocket
┌───────────────────────────────▼─────────────────────────────────────┐
│                        GRAY LOGIC CORE (Go)                          │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ Intelligence: AI Engine │ Voice/NLU │ Presence │ Learning       ││
│  ├─────────────────────────────────────────────────────────────────┤│
│  │ Automation: Scenes │ Scheduler │ Modes │ Events │ Logic         ││
│  ├─────────────────────────────────────────────────────────────────┤│
│  │ Device: Registry │ State │ Commands │ Discovery │ PHM           ││
│  ├─────────────────────────────────────────────────────────────────┤│
│  │ Infrastructure: API │ WebSocket │ SQLite │ MQTT │ Auth          ││
│  └─────────────────────────────────────────────────────────────────┘│
└───────────────────────────────┬─────────────────────────────────────┘
                                │ Internal MQTT Bus
┌───────────────────────────────▼─────────────────────────────────────┐
│                        PROTOCOL BRIDGES                              │
│   KNX │ DALI │ Modbus │ Audio Matrix │ Video │ Security │ CCTV      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Code Structure (Detailed)

```
gray-logic-stack/
├── code/core/                    # Gray Logic Core (Go) — 19K+ lines
│   ├── cmd/graylogic/main.go     # Entry point — wires everything together
│   │                             # - Loads config, opens DB, runs migrations
│   │                             # - Creates registries (device, scene)
│   │                             # - Connects MQTT, starts API server
│   │                             # - Starts knxd + KNX bridge
│   │                             # - Graceful shutdown via signal.NotifyContext
│   │
│   ├── internal/
│   │   ├── api/                  # REST + WebSocket server
│   │   │   ├── server.go         # Chi router setup, JWT auth
│   │   │   ├── router.go         # Route definitions
│   │   │   ├── devices.go        # GET/POST /api/v1/devices
│   │   │   ├── scenes.go         # Scene CRUD + activation
│   │   │   ├── websocket.go      # WS hub for real-time updates
│   │   │   └── middleware.go     # Auth, logging, CORS
│   │   │
│   │   ├── automation/           # Scene engine
│   │   │   ├── types.go          # Scene, SceneAction, SceneExecution
│   │   │   ├── engine.go         # ActivateScene() — groups actions, parallel exec
│   │   │   ├── registry.go       # Scene cache with thread-safe access
│   │   │   └── repository.go     # SQLite persistence
│   │   │
│   │   ├── device/               # Device management
│   │   │   ├── types.go          # Device struct, Domain, Protocol, Capability enums
│   │   │   │                     # DeepCopy() for cache isolation
│   │   │   ├── registry.go       # Thread-safe cache + Repository wrapper
│   │   │   │                     # RWMutex for concurrent access
│   │   │   │                     # All returns are deep copies
│   │   │   ├── repository.go     # SQLite CRUD
│   │   │   └── validation.go     # Device validation rules
│   │   │
│   │   ├── bridges/knx/          # KNX protocol bridge
│   │   │   ├── bridge.go         # Bidirectional KNX ↔ MQTT translation
│   │   │   │                     # - Receives MQTT commands → KNX telegrams
│   │   │   │                     # - Receives KNX telegrams → MQTT state
│   │   │   ├── health.go         # Health reporting (30s interval)
│   │   │   ├── config.go         # Bridge configuration
│   │   │   ├── dpt.go            # Data Point Type encoding/decoding
│   │   │   └── ga_recorder.go    # Passive group address discovery
│   │   │
│   │   ├── infrastructure/
│   │   │   ├── config/           # YAML config loading
│   │   │   ├── database/         # SQLite with WAL, migrations
│   │   │   ├── mqtt/             # Paho MQTT wrapper with reconnect
│   │   │   ├── tsdb/              # VictoriaMetrics client for time-series
│   │   │   └── logging/          # Structured slog logging
│   │   │
│   │   ├── knxd/                 # knxd subprocess manager
│   │   │   └── manager.go        # Start/stop/restart, health checks
│   │   │
│   │   ├── location/             # Areas, rooms, sites
│   │   ├── panel/                # Flutter web panel serving
│   │   └── process/              # Generic subprocess lifecycle
│   │
│   ├── configs/                  # Config templates
│   └── migrations/               # SQL migrations (additive only!)
│
├── sim/knxsim/                   # KNX/IP Simulator (Python) — 300+ lines
│   ├── knxsim.py                 # Main — loads config, starts servers
│   ├── config.yaml               # Virtual device definitions
│   ├── api/                      # FastAPI REST + WebSocket
│   │   ├── app.py                # Route setup
│   │   └── websocket_hub.py      # Live telegram/state streams
│   ├── core/
│   │   ├── premise_manager.py    # Device lifecycle
│   │   ├── telegram_inspector.py # Bus traffic logging
│   │   └── knxip_server.py       # UDP 3671 KNXnet/IP tunnelling
│   ├── devices/                  # Virtual device types
│   └── persistence/db.py         # SQLite for state persistence
│
├── docs/                         # Full documentation (~80 files)
│   ├── CONSTRAINTS.md            # ⭐ MUST READ — all rules in one place
│   ├── overview/                 # Vision, principles, glossary
│   ├── architecture/             # System design, bridge interface
│   ├── data-model/entities.md    # ⭐ All entity definitions
│   ├── protocols/                # KNX, DALI, Modbus, MQTT specs
│   ├── domains/                  # Lighting, climate, blinds, etc.
│   ├── automation/               # Scenes, schedules, modes
│   └── development/              # Coding standards, security
│
└── docker-compose.dev.yml        # Dev stack
```

---

## Key Patterns

### Thread Safety

The codebase uses consistent patterns for concurrency:

```go
// Registry pattern (device/registry.go, automation/registry.go)
type Registry struct {
    cache   map[string]*Entity
    cacheMu sync.RWMutex
}

// Read operations: RLock
func (r *Registry) Get(id string) (*Entity, error) {
    r.cacheMu.RLock()
    defer r.cacheMu.RUnlock()
    return r.cache[id].DeepCopy(), nil  // Always return deep copy!
}

// Write operations: Lock
func (r *Registry) Set(entity *Entity) {
    r.cacheMu.Lock()
    defer r.cacheMu.Unlock()
    r.cache[entity.ID] = entity.DeepCopy()  // Store deep copy!
}
```

### Deep Copy for Cache Isolation

All cached entities use DeepCopy() to prevent external mutation:

```go
func (d *Device) DeepCopy() *Device {
    cpy := *d  // Shallow copy value fields
    cpy.Address = deepCopyMap(d.Address)
    cpy.State = deepCopyMap(d.State)
    cpy.Capabilities = append([]Capability{}, d.Capabilities...)
    return &cpy
}
```

### Scene Execution (Parallel Groups)

Actions are grouped by `Parallel` flag and executed:

```go
// groupActions([A, B(parallel), C(parallel), D]) → [[A,B,C], [D]]
// Group 1 executes concurrently, then Group 2
```

### MQTT Topics

```
graylogic/
├── state/{protocol}/{device_id}    # Bridge → Core (device state)
├── command/{protocol}/{device_id}  # Core → Bridge (commands)
├── health/{protocol}               # Bridge health (retained)
├── discovery/{protocol}            # Discovered devices
├── core/
│   ├── device/{id}/state           # Canonical state
│   └── scene/{id}/activated        # Scene events
└── system/
    └── status                      # System status
```

---

## Data Flow

### Command Flow (User → Device)

```
1. User taps "Cinema Mode" on wall panel
2. POST /api/v1/scenes/cinema/activate
3. Scene Engine loads scene from registry
4. Engine groups actions by parallel flag
5. For each action group:
   - Lookup device → get protocol/gateway
   - Build MQTT payload with command
   - Publish to: graylogic/command/{protocol}/{device}
6. KNX Bridge receives MQTT message
7. Bridge translates to KNX telegram, sends via knxd
8. Device responds with status telegram
9. Bridge publishes state to MQTT
10. Core updates device registry
11. WebSocket broadcasts to all UIs
```

### State Update Flow (Device → Core)

```
1. Physical button pressed on KNX switch
2. Actuator sends status telegram on bus
3. knxd receives, forwards to Bridge
4. Bridge decodes DPT, builds state object
5. Bridge publishes: graylogic/state/knx/{device}
6. Core receives, calls registry.SetDeviceState()
7. Registry updates cache + DB
8. WebSocket hub broadcasts state change
9. All connected UIs update
```

---

## Device Model

### Key Types (from device/types.go)

```go
type Device struct {
    ID           string       // UUID
    Name         string       // Human readable
    Slug         string       // URL-safe
    RoomID       *string      // Optional room
    Type         DeviceType   // light_dimmer, thermostat, etc.
    Domain       Domain       // lighting, climate, blinds, etc.
    Protocol     Protocol     // knx, dali, modbus_tcp, etc.
    Address      Address      // Protocol-specific (map[string]any)
    Capabilities []Capability // on_off, dim, temperature_read, etc.
    State        State        // Current state (map[string]any)
    HealthStatus HealthStatus // online, offline, degraded, unknown
}
```

### Domains

`lighting`, `climate`, `blinds`, `audio`, `video`, `security`, `access`, `energy`, `plant`, `irrigation`, `safety`, `sensor`

### Protocols

`knx`, `dali`, `modbus_rtu`, `modbus_tcp`, `bacnet_ip`, `mqtt`, `http`, `sip`, `rtsp`, `onvif`, `ocpp`

---

## KNXSim (Simulator)

Virtual KNX/IP tunnelling server for development without hardware.

### What it does

- Runs UDP server on port 3671 (KNXnet/IP)
- Accepts tunnelling connections from knxd, ETS, Core
- Virtual devices respond to GroupValue_Read/Write
- Scenarios simulate real behaviour (temperature drift, presence)
- Web UI at :9090/ui/ for control and inspection

### Virtual Device Types

- `light_switch`, `light_dimmer` — On/off, brightness
- `blind` — Position, slat angle
- `presence_sensor` — Motion, lux
- `temperature_sensor` — Temperature values
- `wall_switch` — Push buttons with LEDs

### API

```bash
# Health
GET /api/v1/health

# Devices
GET /api/v1/premises/{id}/devices
POST /api/v1/premises/{id}/devices/{did}/command

# WebSocket
WS /ws/telegrams?premise={id}  # Live telegram stream
WS /ws/state?premise={id}      # Live state updates
```

---

## Database Rules

### Additive Only (CRITICAL)

Once deployed, NEVER:
- Drop columns
- Rename columns
- Change column types

Always:
- Add nullable columns, or
- Add columns with DEFAULT

### Rollback Test

Every migration must pass:
1. Migrate A → B
2. Add data with B
3. Run A binary against B schema
4. A must work normally

---

## Security Patterns

### Authentication

- Passwords: Argon2id
- JWT: HS256, validate `alg`, enforce expiry
- API Keys: Hashed, 1-year default expiry
- PIN: 3-attempt lockout, exponential backoff

### Input Validation

- Whitelist approach
- Parameterized SQL only
- Never pass unsanitized input to shell

### Never Log

- Passwords, secrets, PINs
- Full JWT tokens (last 8 chars only)
- Audio/video content

---

## Testing

### Coverage Targets

| Type | Target |
|------|--------|
| New code | 80% |
| Security-critical | 100% |
| Error paths | 100% |

### Commands

```bash
cd code/core
go test -v ./...
go test -cover -coverprofile=coverage.out ./...
golangci-lint run
```

---

## Performance Targets

| Operation | Target | Max |
|-----------|--------|-----|
| UI tap → command | 50ms | 100ms |
| Command → device | 100ms | 200ms |
| Scene (10 devices) | 200ms | 500ms |
| Voice → action | 1500ms | 2000ms |

| Resource | Baseline | Max |
|----------|----------|-----|
| RAM | 30MB | 100MB |
| CPU (idle) | <1% | 20% |

---

## Dev Environment

```bash
# Location
~/workspace/gray-logic-stack

# Start stack
docker compose -f docker-compose.dev.yml up -d

# Access
KNXSim UI: http://10.6.0.6:9090/ui/
Core API: http://10.6.0.6:8090/api/v1/
Mosquitto: 127.0.0.1:1883
VictoriaMetrics: 127.0.0.1:8428

# Build
cd code/core && make build

# Test
cd code/core && go test ./...
```

---

## Current Status (2026-01)

**Year 1 Foundation: COMPLETE ✅**

| Milestone | Status |
|-----------|--------|
| M1.1 Infrastructure | ✅ SQLite, MQTT, VictoriaMetrics, Config, Logging |
| M1.2 KNX Bridge | ✅ Telegram parsing, knxd client, MQTT translation |
| M1.3 Device Registry | ✅ 50+ types, thread-safe cache, SQLite |
| M1.4 REST API + WebSocket | ✅ Chi router, JWT auth, WS hub |
| M1.5 Flutter Wall Panel | ✅ Riverpod, Dio, WebSocket, optimistic UI |
| M1.6 Basic Scenes | ✅ Engine, parallel execution, 91.6% coverage |

**Current Focus:** KNXSim enhancements

**Next:** Year 2 — Area/Room hierarchy, advanced scenes, DALI bridge

---

## Files to Read First

| Priority | File | Purpose |
|----------|------|---------|
| 1 | `docs/CONSTRAINTS.md` | All rules in one place |
| 2 | `code/core/AGENTS.md` | Go coding standards |
| 3 | `docs/data-model/entities.md` | Data model definitions |
| 4 | `cmd/graylogic/main.go` | How it all wires together |
| 5 | `docs/protocols/mqtt.md` | MQTT topic structure |

---

## What I (Sir) Understand

### Strong Understanding ✅

- Architecture layers and data flow
- Device registry pattern (thread-safe cache, deep copy)
- Scene engine (action grouping, parallel execution)
- MQTT topic structure and message formats
- KNX bridge operation (bidirectional translation)
- Database constraints (additive only)
- Security patterns (auth, validation, logging)
- KNXSim operation and API

### Can Confidently Do

- Add new API endpoints following existing patterns
- Add new device types/capabilities
- Modify scene execution logic
- Add new MQTT message handlers
- Write tests following table-driven pattern
- Debug data flow issues

### Would Read More First

- DALI bridge implementation (not started)
- PHM (Predictive Health Monitoring) details
- Voice/NLU pipeline
- Flutter panel internals

---

*Last updated: 2026-02-01 by Sir — Deep dive complete*
