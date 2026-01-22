# Gray Logic

A building automation platform I'm developing to teach myself BMS architecture and protocol integration. This is a personal learning project — not a commercial product — but it represents how I think through complex systems.

**Background:** I'm a qualified electrician (20 years) with strong IT skills, looking to move into building management systems. Rather than just reading about BMS, I decided to design and build one to properly understand how these systems work.

## What's Working

| Component | Status | Description |
|-----------|--------|-------------|
| **KNX Bridge** | ✅ Complete | Tested protocol bridge connecting KNX bus to internal MQTT |
| **knxd Manager** | ✅ Complete | Manages knxd daemon with health monitoring and auto-restart |
| **Device Registry** | 🔄 In Progress | Central device catalogue with SQLite persistence |
| **Core Infrastructure** | ✅ Complete | Config, logging, MQTT client, InfluxDB client, SQLite |

### KNX Bridge Details

The KNX bridge is the most complete component — a working Go implementation that:

- Connects to knxd daemon (USB or IP tunnelling)
- Parses and encodes KNX telegrams (group communication)
- Translates between KNX and MQTT messages
- Supports multiple datapoint types (DPT1, DPT5, DPT9, etc.)
- Includes comprehensive unit tests (69% coverage)

```
KNX Bus ←→ knxd daemon ←→ Gray Logic KNX Bridge ←→ MQTT ←→ Core
```

I've tested this against both real hardware (Weinzierl USB interface) and KNX Virtual.

## Architecture

```
                        ┌─────────────────────────────────────────┐
                        │           Gray Logic Core (Go)           │
                        │                                          │
                        │  • Device Registry    • Scene Engine     │
                        │  • State Management   • Scheduler        │
                        │  • REST API           • WebSocket        │
                        └─────────────────┬─────────────────────────┘
                                          │
                                    Internal MQTT
                                          │
              ┌───────────────────────────┼───────────────────────────┐
              │                           │                           │
     ┌────────▼────────┐        ┌────────▼────────┐        ┌────────▼────────┐
     │   KNX Bridge    │        │   DALI Bridge   │        │  Modbus Bridge  │
     │   (Complete)    │        │   (Planned)     │        │   (Planned)     │
     └────────┬────────┘        └─────────────────┘        └─────────────────┘
              │
        ┌─────▼─────┐
        │   knxd    │
        └─────┬─────┘
              │
     ═══════════════════  KNX Bus (30V DC twisted pair)
         │         │
     ┌───┴───┐ ┌───┴───┐
     │Dimmer │ │Switch │  ... actual KNX devices
     └───────┘ └───────┘
```

## Technology Choices

| Choice | Why |
|--------|-----|
| **Go** | Single binary, no runtime dependencies, compiles for any platform |
| **MQTT** | Simple pub/sub, easy to debug with standard tools, proven in IoT |
| **SQLite** | Embedded database, zero maintenance, works for decades |
| **knxd** | Mature open-source KNX daemon, handles USB/IP interfaces |

## Design Principles

These aren't just nice-to-haves — they're hard rules I've documented and designed around:

1. **Physical controls always work** — Wall switches must function even if all software is down
2. **Life safety is independent** — Fire alarms use certified hardware; software observes, never controls
3. **No cloud dependencies** — Core functionality works without internet
4. **Open standards** — KNX, DALI, Modbus, BACnet — no proprietary lock-in

## Protocols Covered

| Protocol | Status | Notes |
|----------|--------|-------|
| KNX | Working | Bridge complete, tested with real hardware |
| DALI | Documented | Understand addressable lighting concepts |
| Modbus | Documented | TCP and RTU for plant equipment |
| BACnet | Documented | For commercial BMS integration |

## Documentation

The `/docs` folder contains detailed specifications I've written covering:

- **Architecture:** System design, component interactions, data flow
- **Protocols:** KNX, DALI, Modbus, BACnet specifications
- **Domains:** Lighting, climate, blinds, audio, security, energy, pool/plant
- **Data Model:** Device entities, state management, automation rules

This documentation represents how I think through complex systems before writing code.

## Project Structure

```
gray-logic-stack/
├── code/core/                    # Go implementation
│   ├── cmd/graylogic/           # Main application entry point
│   ├── internal/
│   │   ├── bridges/knx/         # KNX protocol bridge (complete)
│   │   ├── knxd/                # knxd daemon manager (complete)
│   │   ├── device/              # Device registry (in progress)
│   │   ├── infrastructure/      # Config, MQTT, database, logging
│   │   └── process/             # Subprocess management
│   ├── configs/                 # YAML configuration files
│   └── docs/technical/          # Package design documentation
├── docs/                        # Architecture and domain specs
└── CHANGELOG.md                 # Project history
```

## Running the Code

```bash
# Build
cd code/core
go build -o bin/graylogic ./cmd/graylogic

# Run tests
go test -v ./...

# Lint
golangci-lint run
```

Requires Go 1.25+ and knxd for KNX functionality.

## Current Focus

**Milestone M1.3:** Device Registry — CRUD operations, caching, SQLite persistence

**Next:** REST API and WebSocket for real-time state updates

## Why I Built This

I've been maintaining control systems for years — fault-finding through panels, replacing contactors, tracing circuits — without fully understanding how they're designed from the ground up. Building my own system is how I learn best: by thinking through the architecture decisions, understanding why protocols work the way they do, and writing code that actually runs.

This project won't replace commercial BMS platforms, but it's taught me more about building automation than any course could. The domain thinking and architecture decisions are mine; I use AI tools to help with implementation.

## Links

- **Technical Docs:** [code/core/docs/technical/](code/core/docs/technical/)
- **KNX Bridge Design:** [code/core/docs/technical/packages/knx-bridge.md](code/core/docs/technical/packages/knx-bridge.md)
- **System Architecture:** [docs/architecture/system-overview.md](docs/architecture/system-overview.md)
- **Development Log:** [code/core/IMPLEMENTATION.md](code/core/IMPLEMENTATION.md)

---

*This is a personal learning project by Darren Gray, an electrician exploring building management systems.*
