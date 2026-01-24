# Gray Logic

A building automation platform I'm developing to teach myself BMS architecture and protocol integration. This is a personal learning project — not a commercial product — but it represents how I think through complex systems.

**Background:** I'm a qualified electrician (20 years) with strong IT skills, looking to move into building management systems. Rather than just reading about BMS, I decided to design and build one to properly understand how these systems work.

## What's Working

| Component | Status | Description |
|-----------|--------|-------------|
| **Core Infrastructure** | ✅ Complete | Config, logging, MQTT client, InfluxDB client, SQLite |
| **KNX Bridge** | ✅ Complete | Protocol bridge connecting KNX bus to internal MQTT |
| **knxd Manager** | ✅ Complete | Manages knxd daemon with health monitoring and auto-restart |
| **Device Registry** | ✅ Complete | Central device catalogue with SQLite persistence and caching |
| **REST API + WebSocket** | ✅ Complete | Full CRUD API with real-time state push via WebSocket |
| **Flutter Wall Panel** | ✅ Complete | Touch UI embedded in Go binary, SPA with device/scene control |
| **Scenes Engine** | ✅ Complete | Scene definitions, activation, execution tracking, transitions |
| **Location Model** | ✅ Complete | Site → Area → Room spatial hierarchy |

### Year 1 Foundation — Complete

All 6 milestones delivered:

| Milestone | What It Does |
|-----------|-------------|
| M1.1 Infrastructure | Config loading, SQLite, MQTT client, InfluxDB, structured logging |
| M1.2 KNX Bridge | Bidirectional KNX↔MQTT with DPT encoding, reconnection, health checks |
| M1.3 Device Registry | Device CRUD, state management, capability model, caching |
| M1.4 REST API | Chi router, JWT auth, WebSocket hub, CORS, middleware |
| M1.5 Wall Panel | Flutter web app embedded via go:embed, SPA fallback, optimistic UI |
| M1.6 Scenes Engine | Scene CRUD, multi-action activation, execution history, transitions |

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                     User Interfaces                                   │
│   Flutter Wall Panel (/panel/)  •  Mobile App (future)  •  Web Admin │
└──────────────────────────┬───────────────────────────────────────────┘
                           │ REST API + WebSocket
                           ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      GRAY LOGIC CORE (Go)                            │
│                                                                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────────┐  │
│  │ API Server │  │  Scenes    │  │  Device    │  │   Location   │  │
│  │ + WebSocket│  │  Engine    │  │  Registry  │  │   Model      │  │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘  └──────────────┘  │
│        │                │                │                            │
│  ┌─────▼────────────────▼────────────────▼──────────────────────┐    │
│  │              Infrastructure Layer                             │    │
│  │   Config  •  SQLite  •  MQTT Client  •  InfluxDB  •  Logging │    │
│  └──────────────────────────┬────────────────────────────────────┘    │
└─────────────────────────────┼────────────────────────────────────────┘
                              │ Internal MQTT Bus
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       PROTOCOL BRIDGES                               │
│                                                                      │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────┐   │
│  │ KNX Bridge  │     │ DALI Bridge │     │  Modbus Bridge      │   │
│  │ (Complete)  │     │ (Planned)   │     │  (Planned)          │   │
│  └──────┬──────┘     └─────────────┘     └─────────────────────┘   │
│         │                                                            │
│    ┌────▼────┐                                                       │
│    │  knxd   │                                                       │
│    └────┬────┘                                                       │
└─────────┼────────────────────────────────────────────────────────────┘
          │
  ════════╧═══════════  Physical Bus (KNX 30V DC twisted pair)
      │         │
  ┌───┴───┐ ┌───┴───┐
  │Dimmer │ │Switch │  ... actual devices
  └───────┘ └───────┘
```

## Technology Choices

| Choice | Why |
|--------|-----|
| **Go** | Single binary, no runtime dependencies, multi-decade stability |
| **SQLite** | Embedded database, zero maintenance, works for decades |
| **MQTT** | Simple pub/sub, easy to debug, proven in IoT |
| **Flutter** | Cross-platform UI (wall panels, mobile, web) from one codebase |
| **knxd** | Mature open-source KNX daemon, handles USB/IP interfaces |
| **go:embed** | Flutter web build baked into Go binary — zero external file deps |

## Design Principles

These aren't just nice-to-haves — they're hard rules I've documented and designed around:

1. **Physical controls always work** — Wall switches must function even if all software is down
2. **Life safety is independent** — Fire alarms use certified hardware; software observes, never controls
3. **No cloud dependencies** — Core functionality works without internet
4. **Open standards** — KNX, DALI, Modbus — no proprietary lock-in
5. **Multi-decade deployment** — Version-pin everything, no forced upgrades
6. **Customer owns their system** — Full documentation, no dealer locks

## Protocols Covered

| Protocol | Status | Notes |
|----------|--------|-------|
| KNX | Working | Bridge complete, tested with real hardware |
| DALI | Documented | Addressable lighting, Year 2 target |
| Modbus | Documented | TCP and RTU for plant equipment |
| BACnet | Documented | For commercial BMS integration |

## Project Structure

```
gray-logic-stack/
├── code/
│   ├── core/                        # Gray Logic Core (Go)
│   │   ├── cmd/graylogic/          # Application entry point
│   │   ├── internal/
│   │   │   ├── api/                # REST API + WebSocket server
│   │   │   ├── automation/         # Scenes engine
│   │   │   ├── bridges/knx/        # KNX protocol bridge
│   │   │   ├── device/             # Device registry
│   │   │   ├── infrastructure/     # Config, MQTT, database, logging
│   │   │   ├── knxd/              # knxd daemon manager
│   │   │   ├── location/          # Area/Room spatial model
│   │   │   ├── panel/             # Flutter web UI embedding
│   │   │   └── process/           # Subprocess management
│   │   ├── configs/                # YAML configuration templates
│   │   └── migrations/            # SQL schema migrations
│   └── ui/wallpanel/              # Flutter wall panel app (Dart)
├── docs/                           # Architecture and domain specs
├── CHANGELOG.md                    # Project history
└── PROJECT-STATUS.md              # Current progress tracker
```

## Running the Code

```bash
# Build
cd code/core
go build -o bin/graylogic ./cmd/graylogic

# Run tests (14 packages, all passing)
go test -race ./...

# Lint
golangci-lint run

# Start dev services (MQTT + InfluxDB)
docker compose -f docker-compose.dev.yml up -d
```

Requires Go 1.25+ and knxd for KNX functionality.

## Current Focus

**Year 1 Foundation: Complete.** All 6 milestones delivered and audited.

**Next:** Refinement and simulation environment — building a Docker-based KNX simulator for full end-to-end testing without physical hardware.

## Why I Built This

I've been maintaining control systems for years — fault-finding through panels, replacing contactors, tracing circuits — without fully understanding how they're designed from the ground up. Building my own system is how I learn best: by thinking through the architecture decisions, understanding why protocols work the way they do, and writing code that actually runs.

Building my own system has taught me more about building automation than any course could. The domain thinking and architecture decisions are mine; I use AI tools to help with implementation.

## Links

- **Technical Docs:** [code/core/docs/technical/](code/core/docs/technical/)
- **KNX Bridge Design:** [code/core/docs/technical/packages/knx-bridge.md](code/core/docs/technical/packages/knx-bridge.md)
- **System Architecture:** [docs/architecture/system-overview.md](docs/architecture/system-overview.md)
- **Development Log:** [PROJECT-STATUS.md](PROJECT-STATUS.md)

---

*This is a personal learning project by Darren Gray, an electrician exploring building management systems.*
