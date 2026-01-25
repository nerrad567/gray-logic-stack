# Gray Logic — Project Status

> **Last Updated:** 2026-01-25
> **Current Phase:** Implementation (M1.5 Complete — Year 1 Foundation Done!)

---

## RESUME HERE — Next Session

**Last session:** 2026-01-25 (Session 17 - KNXSim Engineer UI & Sync Fixes)
**Current milestone:** Year 1 Foundation Complete + KNXSim Phase 2 in progress

**What's done:**
- M1.1 Core Infrastructure (SQLite, MQTT, InfluxDB, Config, Logging) ✅
- M1.2 KNX Bridge (telegram parsing, knxd client, MQTT translation, 4 audit cycles) ✅
- M1.3 Device Registry (50+ types, thread-safe cache, SQLite persistence, KNX integration) ✅
- M1.4 REST API + WebSocket (Chi router, device CRUD, state commands, WebSocket hub, JWT auth) ✅
- M1.5 Flutter Wall Panel (Riverpod, Dio, WebSocket, optimistic UI, embedded web serving) ✅
- M1.6 Basic Scenes (automation package, scene engine, parallel execution, REST API, 91.6% coverage) ✅
- KNXSim Phase 2.3 Device Controls ✅ (lights, blinds, presence, sensors)
- KNXSim Phase 2.4 Engineer Mode ✅ (telegram inspector, device panel, GA inspection)

**What's next:**
- KNXSim Phase 2.1/2.2: Building overview, floor navigation, room grid
- Auth hardening (production JWT, refresh tokens, role-based access)
- Year 2 planning (M2.1 Area/Room hierarchy, M2.2 advanced scenes, M2.5 DALI bridge)

---

## Quick Summary

| Area | Status |
|------|--------|
| Core Documentation | ✅ Complete |
| Architecture | ✅ Complete |
| Domain Specs | ✅ 13/13 complete |
| Protocol Specs | ✅ Complete |
| Integration Specs | ✅ Complete |
| API Specification | ✅ Complete |
| Automation Spec | ✅ Complete |
| Intelligence Specs | ✅ Complete |
| Resilience Specs | ✅ Complete |
| Deployment Guides | ✅ Complete |
| Business Docs | ✅ Complete |
| Development Docs | ✅ Complete |
| Operations Docs | ✅ Complete |
| Commissioning Docs | ✅ Complete |
| Code | ✅ Year 1 Complete (all 6 milestones) |

---

## Documentation Status

All documentation is complete. See `CHANGELOG.md` entries from 2026-01-12 to 2026-01-18 for details.

#### Overview (`docs/overview/`)
- [x] `vision.md` — Product vision and goals
- [x] `principles.md` — Hard rules and design principles
- [x] `capabilities-and-benefits.md` — Capabilities summary v2.0
- [x] `glossary.md` — Standard terminology

#### Architecture (`docs/architecture/`)
- [x] `system-overview.md` — High-level architecture
- [x] `core-internals.md` — Go Core package structure
- [x] `bridge-interface.md` — MQTT bridge contract
- [x] `energy-model.md` — Bidirectional energy flows
- [x] `security-model.md` — Authentication and authorization
- [x] `cloud-relay.md` — Cloud services architecture v0.1.0

#### Data Model (`docs/data-model/`)
- [x] `entities.md` — Core entities (Site, Area, Room, Device, Scene, etc.)
- [x] `schemas/` — JSON Schema definitions for all entities

#### Protocols (`docs/protocols/`)
- [x] `knx.md` — KNX integration via knxd
- [x] `dali.md` — DALI lighting control
- [x] `modbus.md` — Modbus RTU/TCP for plant equipment
- [x] `mqtt.md` — Internal message bus
- [x] `bacnet.md` — BACnet roadmap (Year 2 placeholder)
- [x] `ocpp.md` — EV charging protocol

#### Integrations (`docs/integration/`)
- [x] `cctv.md` — CCTV and video surveillance
- [x] `access-control.md` — Door access, intercoms, gates
- [x] `fire-alarm.md` — Fire alarm monitoring (observation only)
- [x] `diy-integration.md` — DIY device integration spec (Matter/Zigbee)

#### Interfaces (`docs/interfaces/`)
- [x] `api.md` — REST and WebSocket API specification

#### Automation (`docs/automation/`)
- [x] `automation.md` — Scenes, schedules, modes, conditions, events

#### Domains (`docs/domains/`)
- [x] `lighting.md` — Lighting control + commercial
- [x] `climate.md` — HVAC + commercial
- [x] `blinds.md` — Shading and blind control
- [x] `plant.md` — Plant room equipment
- [x] `audio.md` — Multi-room audio
- [x] `security.md` — Alarm system integration
- [x] `energy.md` — Energy management
- [x] `video.md` — Video/AV distribution
- [x] `irrigation.md` — Garden and outdoor
- [x] `leak-protection.md` — Leak detection and auto-shutoff
- [x] `water-management.md` — Rainwater, greywater, treatment
- [x] `presence.md` — Occupancy and presence detection
- [x] `pool.md` — Pool chemistry, covers, water features

#### Deployment (`docs/deployment/`)
- [x] `office-commercial.md` — Commercial deployment guide
- [x] `residential.md` — Residential deployment guide
- [x] `handover-pack-template.md` — Customer handover template

#### Intelligence (`docs/intelligence/`)
- [x] `ai-premium-features.md` — AI feature boundaries
- [x] `phm.md` — Predictive Health Monitoring framework
- [x] `voice.md` — Voice pipeline specification (Whisper, NLU, Piper)
- [x] `weather.md` — Weather integration

#### Resilience (`docs/resilience/`)
- [x] `offline.md` — Offline behavior and graceful degradation
- [x] `backup.md` — Backup and recovery procedures
- [x] `satellite-weather.md` — Satellite weather decode
- [x] `mesh-comms.md` — LoRa/Meshtastic mesh communications

#### Operations (`docs/operations/`)
- [x] `updates.md` — Update and upgrade strategy
- [x] `monitoring.md` — Monitoring and alerting
- [x] `maintenance.md` — System maintenance procedures

#### Commissioning (`docs/commissioning/`)
- [x] `discovery.md` — Device discovery specification

#### Business (`docs/business/`)
- [x] `business-case.md` — Business case, market analysis, open source strategy
- [x] `pricing.md` — Installation tiers, hardware strategy, support tiers
- [x] `sales-spec.md` — Customer journey, proposals, contracts, installation
- [x] `go-to-market.md` — Phased growth strategy, marketing channels
- [x] `certification.md` — Training curriculum, partner benefits, quality control
- [x] `institutional-principles.md` — Building for generations, succession, knowledge preservation
- [x] `subscription-pricing.md` — Cloud subscription tier details

---

## Code Status

| Component | Status | Notes |
|-----------|--------|-------|
| Gray Logic Core (Go) | ✅ Year 1 Complete | M1.1 + M1.2 + M1.3 + M1.4 + M1.5 + M1.6 complete |
| REST API + WebSocket | ✅ Complete | Chi router, device CRUD, state commands, WebSocket hub, JWT auth placeholder |
| KNX Bridge | ✅ Complete | Wired into main.go, 4 audit cycles (15 issues fixed) |
| knxd Manager | ✅ Complete | Managed subprocess, multi-layer health checks, USB reset |
| Basic Scenes | ✅ Complete | Automation package, scene engine, parallel execution, REST API, 91.6% coverage |
| Device Registry | ✅ Complete | Types, repository, validation, wired to main.go + KNX bridge |
| Process Manager | ✅ Complete | Generic subprocess lifecycle (reusable for DALI, Modbus) |
| DALI Bridge | ❌ Not started | Spec complete (Year 2) |
| Modbus Bridge | ❌ Not started | Spec complete (Year 2) |
| Flutter Wall Panel | ✅ Complete | Riverpod, Dio, WebSocket, optimistic UI, embedded web serving |
| Location API | ✅ Complete | Area/room endpoints, SQLite repository |
| Voice Pipeline | ❌ Not started | Year 4 |

### M1.1 Progress (Core Infrastructure) — ✅ Complete
- [x] Go module initialised
- [x] Directory structure created
- [x] Makefile with build automation
- [x] golangci-lint configured
- [x] Configuration system (YAML + env vars)
- [x] SQLite database package with migrations
- [x] MQTT client package with auto-reconnect
- [x] InfluxDB client package
- [x] Docker Compose (Mosquitto, InfluxDB)
- [x] Structured logging
- [x] Infrastructure wired into main.go

### M1.2 Progress (KNX Bridge) — ✅ Complete
- [x] telegram.go — KNX telegram parsing/encoding
- [x] knxd.go — knxd client (TCP/Unix socket)
- [x] address.go — Group address parsing
- [x] dpt.go — Datapoint type encoding/decoding
- [x] config.go — Bridge configuration with YAML + env vars
- [x] messages.go — MQTT message types (command, ack, state, health)
- [x] health.go — Health status reporting to MQTT
- [x] bridge.go — Main orchestration (KNX ↔ MQTT translation)
- [x] Comprehensive unit tests (69% coverage, all passing)
- [x] 4 code audit cycles (15 issues fixed)
- [x] Wired into main.go with MQTT adapter
- [x] Sample knx-bridge.yaml configuration created

### M1.3 Progress (Device Registry) — ✅ Complete
- [x] types.go — 50+ device types, 12+ domains, 14 protocols, 45+ capabilities
- [x] registry.go — Thread-safe in-memory cache with CRUD operations
- [x] repository.go — SQLite persistence layer (Repository interface + SQLiteRepository)
- [x] validation.go — Device validation, slug generation, protocol-specific address checks
- [x] errors.go — Domain-specific error types with errors.Is support
- [x] Wired into main.go — RefreshCache on startup, adapter to KNX bridge
- [x] KNX bridge integration — State/health updates flow from bridge to registry
- [x] Integration tests — Full lifecycle (create → cache → state → health → persist → delete)
- [x] Unit tests for knxd package — Config validation, address parsing, BuildArgs
- [x] Unit tests for process package — Construction, backoff, state transitions

### M1.4 Progress (REST API + WebSocket) — ✅ Complete
- [x] server.go — HTTP server lifecycle (New, Start, Close, HealthCheck)
- [x] router.go — Chi router with middleware wiring and route registration
- [x] errors.go — HTTP error response helpers with typed error codes
- [x] middleware.go — Request ID, logging, recovery, CORS, auth placeholder
- [x] devices.go — Device CRUD + state/command handlers (MQTT publish)
- [x] websocket.go — WebSocket hub, client management, channel subscriptions
- [x] auth.go — JWT login (dev credentials), ticket-based WebSocket auth
- [x] MQTT → WebSocket bridge — State updates broadcast to subscribed clients
- [x] TLS support — Optional ListenAndServeTLS from config
- [x] Wired into main.go — Initialised after MQTT, before InfluxDB
- [x] server_test.go — 23 tests (health, middleware, CRUD, state, auth, WebSocket hub)
- [x] All 12 packages pass, build and lint clean

---

## Roadmap

### Year 1 — Foundation (2026) ← Current
| Milestone | Goal | Status |
|-----------|------|--------|
| **M1.1** | Core Infrastructure (SQLite, MQTT, InfluxDB) | ✅ Complete |
| **M1.2** | KNX Bridge | ✅ Complete |
| **M1.3** | Device Registry | ✅ Complete |
| **M1.4** | REST API + WebSocket | ✅ Complete |
| **M1.5** | Flutter Wall Panel | ✅ Complete |
| **M1.6** | Basic Scenes | ✅ Complete |

### Year 2 — Automation Expansion (2027)
| Milestone | Goal |
|-----------|------|
| M2.1 | Area/Room hierarchy |
| M2.2 | Scene engine (multi-device, timed) |
| M2.3 | Mode system (Home/Away/Night/Holiday) |
| M2.4 | Astronomical clock + scheduler |
| M2.5 | DALI bridge |
| M2.6 | Blind control |
| M2.7 | Climate integration (Modbus → HVAC) |
| M2.8 | Flutter mobile app |

### Year 3 — Integration & Resilience (2028)
| Milestone | Goal |
|-----------|------|
| M3.1 | Audio matrix (HTD/Russound) |
| M3.2 | Video matrix (Atlona) |
| M3.3 | Security panel (monitoring only) |
| M3.4 | CCTV (RTSP/ONVIF) |
| M3.5 | BACnet bridge |
| M3.6 | Advanced conditional logic |
| M3.7 | System Supervisor (Level 1) |
| M3.8 | Machine-readable error catalog |

### Year 4 — Intelligence (2029)
| Milestone | Goal |
|-----------|------|
| M4.1 | Voice pipeline (Whisper → NLU → Piper) |
| M4.2 | PHM baseline learning |
| M4.3 | Local AI engine |
| M4.4 | Occupancy prediction |
| M4.5 | Energy insights |
| M4.6 | System Supervisor with playbook recovery |
| M4.7 | Simulation/chaos testing |
| M4.8 | Circuit breaker patterns |
| M4.9 | Failure Memory system |
| M4.10 | Workflow Learning |

### Year 5 — Commercial Readiness (2030)
| Milestone | Goal |
|-----------|------|
| M5.1 | Web-based commissioning tool |
| M5.2 | Configuration backup/restore |
| M5.3 | Remote diagnostics (WireGuard) |
| M5.4 | Installer documentation |
| M5.5 | Customer handover pack generator |
| M5.6 | Automated testing framework |
| M5.7 | First customer deployment |

---

## Year 1 Task Breakdowns

### M1.5: Flutter Wall Panel — ✅ Complete

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | Flutter project setup (ui/wallpanel) | ✅ Complete | - |
| 2 | Device models (matching Go structs) | ✅ Complete | M1.3 |
| 3 | Repository pattern for API calls (Dio + Riverpod) | ✅ Complete | M1.4 |
| 4 | WebSocket connection for state updates | ✅ Complete | M1.4 |
| 5 | RoomView widget (device grid) | ✅ Complete | Tasks 2-4 |
| 6 | SwitchTile (on/off) | ✅ Complete | Task 5 |
| 7 | DimmerTile (slider, no snap-back) | ✅ Complete | Task 5 |
| 8 | Scene activation bar | ✅ Complete | M1.6 |
| 9 | Optimistic UI (opacity pulse for pending state) | ✅ Complete | Tasks 5-7 |
| 10 | Go: Location API (areas/rooms endpoints) | ✅ Complete | M1.3 |
| 11 | Go: Panel web embed (go:embed, SPA fallback) | ✅ Complete | Task 1 |
| 12 | Go: Dev mode device simulation (800ms delay + WS broadcast) | ✅ Complete | M1.4 |

### M1.6: Basic Scenes — ✅ Complete

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | Database schema (scenes, scene_executions) | ✅ Complete | M1.1 |
| 2 | Scene struct definition + validation | ✅ Complete | Task 1 |
| 3 | scene engine — Activate, parallel/sequential execution | ✅ Complete | Task 2 |
| 4 | Full scene CRUD API (GET/POST/PATCH/DELETE) | ✅ Complete | M1.4 |
| 5 | POST /api/v1/scenes/{id}/activate | ✅ Complete | Task 3 |
| 6 | Scene execution <500ms for 10 devices | ✅ Complete | Task 3 |
| 7 | Persist scene state across restarts (registry cache) | ✅ Complete | Task 3 |

---

## Technical Decisions

| Decision | Rationale |
|----------|-----------|
| **Go 1.25** | Latest stable, no bleeding-edge features |
| **gopkg.in/yaml.v3** | Mature, well-maintained, supports comments |
| **golangci-lint** | Industry standard, catches security issues |
| **Makefile** | Universal, works for decades, no dependencies |
| **internal/ directory** | Go-enforced encapsulation |
| **Chi router** | Stdlib-compatible, minimal, stable since 2015 |
| **Gorilla WebSocket** | Industry standard, proven in production |
| **Ticket-based WS auth** | Prevents JWT leakage in URL params/logs |

### Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| gopkg.in/yaml.v3 | v3.0.1 | YAML configuration parsing |
| github.com/mattn/go-sqlite3 | v1.14.33 | SQLite database driver (CGO) |
| github.com/eclipse/paho.mqtt.golang | v1.4.3 | MQTT client library |
| github.com/influxdata/influxdb-client-go/v2 | v2.14.0 | InfluxDB v2 client |
| github.com/go-chi/chi/v5 | v5.2.4 | HTTP router |
| github.com/gorilla/websocket | v1.5.3 | WebSocket upgrade |
| github.com/golang-jwt/jwt/v5 | v5.3.0 | JWT token handling |

---

## Session Log

### Session 1: 2026-01-18 — Project Initialisation (M1.1)

**Goal:** Set up Go project structure

- Initialised Go module (`github.com/nerrad567/gray-logic-core`, Go 1.25.6)
- Created directory structure per CODING-STANDARDS.md
- Created main.go with signal.NotifyContext for graceful shutdown
- Created Makefile (15 targets, version injection via ldflags)
- Installed golangci-lint v1.64.8, configured strict linting
- Created configuration system (YAML + env var overrides)
- Fixed UK spelling issues ("Initialize" → "Initialise")

### Session 2: 2026-01-19 — SQLite Database Package (M1.1)

**Goal:** Implement SQLite with embedded migrations

- Created `internal/infrastructure/database/` package
- embed.FS for compiled-in migrations, WAL mode, STRICT mode
- Initial schema: sites, areas, rooms, devices, audit_logs
- Added `github.com/mattn/go-sqlite3` dependency

### Session 3: 2026-01-19 — Database Tests & Docker Compose (M1.1)

**Goal:** Add tests and dev infrastructure

- 21 database test cases (80.4% coverage)
- Docker Compose: Mosquitto (MQTT) + InfluxDB (time-series)
- Isolated from existing media-stack (separate network, localhost-only ports)
- Container names: `graylogic-mosquitto`, `graylogic-influxdb`

### Session 4: 2026-01-19 — MQTT Client Package (M1.1)

**Goal:** MQTT client with auto-reconnect

- Created `internal/infrastructure/mqtt/` (7 files, ~1,200 lines)
- Auto-reconnect with subscription restoration
- Topic namespace helpers (type-safe topic construction)
- Panic recovery in handlers
- Added `github.com/eclipse/paho.mqtt.golang` v1.4.3

### Session 5: 2026-01-19 — InfluxDB Client Package (M1.1)

**Goal:** Time-series storage for PHM

- Created `internal/infrastructure/influxdb/` (5 files)
- Non-blocking WriteAPI with batching
- Domain helpers: WriteDeviceMetric, WriteEnergyMetric
- Added `github.com/influxdata/influxdb-client-go/v2` v2.14.0

### Session 6: 2026-01-20 — Infrastructure Wiring & Logging (M1.1 Complete)

**Goal:** Wire everything into main.go

- Created structured logging package (slog-based, JSON/text output)
- Updated main.go: config → DB → MQTT → InfluxDB → health checks
- Graceful shutdown with LIFO defer chain
- **M1.1 Core Infrastructure complete**

### Session 7: 2026-01-20 — M1.1 Final Code Audit

**Goal:** Harden before moving to M1.2

- 4 audit iterations, all static checks pass
- Fixed: JWT validation, InfluxDB shutdown order, context timeouts
- Added bounds validation, MQTT payload size limits
- Modified `/code-audit` command with auto-tracking

### Session 8: 2026-01-20 — M1.2 KNX Bridge Phase 1

**Goal:** Low-level KNX communication

- Created telegram.go (~180 lines): KNX telegram parsing/encoding
- Created knxd.go (~350 lines): TCP/Unix socket client with handshake
- MockKNXDServer for testing, 33 tests passing
- Fixed bug: single-byte values > 0x3F encoding

### Session 9: 2026-01-20 — M1.2 KNX Bridge Phase 2 (Complete)

**Goal:** Complete bridge package

- Created: config.go, messages.go, health.go, bridge.go, errors.go, doc.go
- 4 code audit cycles (15 issues fixed across audits #5-#8)
- Key patterns: closeOnce, bridge-level context, state cache pruning, password redaction
- Coverage: 69.4%, all tests passing
- **Total: ~3,500 lines of Go code + tests**

### Session 10: 2026-01-20 — M1.2 Wiring (M1.2 Complete)

**Goal:** Wire KNX bridge into main.go

- Created MQTT adapter (bridges interface differences)
- Added `config_file` field to KNXConfig
- Created `configs/knx-bridge.yaml` sample (180 lines)
- **M1.2 KNX Bridge complete**

### Session 11: 2026-01-21 — knxd Subprocess Management

**Goal:** Managed knxd lifecycle

- Created `internal/process/` (~590 lines): Generic subprocess management
- Created `internal/knxd/` (~720 lines): knxd-specific wrapper
- Multi-layer health checks (Layers 0-4: USB → process → TCP → EIB → bus)
- USB device reset, PID file management, exponential backoff
- Tested with physical Weinzierl KNX-USB interface

### Session 12: 2026-01-22 — Documentation Sync & Testing

**Goal:** Package docs and hardware testing

- Created package documentation (knxd-manager, device-registry, process-manager)
- Multi-layer health checks tested with KNX Virtual (IPT backend)
- USB backend tested with Weinzierl interface + Theben PSU

### Session 13: 2026-01-22 — M1.3 Device Registry (Complete)

**Goal:** Device catalogue with persistence

- Created `internal/device/` (9 files, ~1,200 lines)
- 50+ device types, 12+ domains, 14 protocols, 45+ capabilities
- Thread-safe registry (RWMutex, deep-copy semantics)
- SQLite persistence via Repository interface
- Integration tests: full lifecycle through real SQLite
- Added tests for knxd (14 functions) and process (13 functions) packages
- Wired into main.go with KNX bridge adapter
- **M1.3 Device Registry complete**

### Session 14: 2026-01-23 — M1.4 REST API + WebSocket (Complete)

**Goal:** HTTP API and real-time layer

- Created `internal/api/` (9 files, ~2,000 lines)
- Chi router with middleware stack (request ID, logging, recovery, CORS)
- Device CRUD + state/command endpoints (MQTT publish for commands)
- WebSocket hub with channel subscriptions and MQTT → WS bridge
- JWT auth placeholder + ticket-based WebSocket auth
- TLS support (optional ListenAndServeTLS)
- 23 tests (health, middleware, CRUD, state, auth, hub)
- All 12 packages pass, build and lint clean
- Added: chi v5.2.4, gorilla/websocket v1.5.3, golang-jwt v5.3.0
- **M1.4 REST API + WebSocket complete**

### Session 15: 2026-01-23 — M1.6 Basic Scenes (Complete)

**Goal:** Scene engine with parallel execution and REST API

- Created `internal/automation/` package (7 files, ~1,800 lines):
  - **types.go** — Scene, SceneAction, SceneExecution structs with DeepCopy
  - **errors.go** — Domain error sentinels (ErrSceneNotFound, ErrSceneDisabled, etc.)
  - **validation.go** — ValidateScene, ValidateAction, GenerateSlug
  - **repository.go** — SQLiteRepository with full CRUD + execution logging
  - **registry.go** — Thread-safe cache wrapping Repository (RWMutex, deep-copy)
  - **engine.go** — Scene execution engine (parallel/sequential groups, delays, fade, MQTT publish)
  - **doc.go** — Package documentation
- Created `internal/api/scenes.go` — 7 HTTP handlers (List, Get, Create, Update, Delete, Activate, ListExecutions)
- Created SQLite migrations (`20260123_150000_scenes.up.sql` / `.down.sql`)
- Modified `internal/api/server.go` — Added SceneEngine, SceneRegistry, SceneRepo, ExternalHub to Deps
- Modified `internal/api/router.go` — Added scene route group under protected routes
- Modified `cmd/graylogic/main.go` — Wired automation package with adapter pattern for DeviceRegistry and MQTTClient interfaces
- External hub injection: WebSocket hub created before both engine and API server, shared via Deps
- **Tests**: 60+ new tests across 6 test files (unit, integration, API handler, performance)
- **Coverage**: 91.6% on automation package (target was 80%)
- **Performance**: 10-device parallel scene activation <500ms verified
- UK English throughout (colour, not color) — enforced after lint review
- All packages build, test, and lint clean
- **M1.6 Basic Scenes complete**

### Session 16: 2026-01-24 — M1.5 Flutter Wall Panel (Complete)

**Goal:** Wall panel UI with embedded web serving

- Created Flutter wall panel app (`code/ui/wallpanel/`, 103 files):
  - **Riverpod** state management with Dio HTTP client
  - Auth flow: JWT login → WebSocket ticket exchange
  - Room device grid with SwitchTile and DimmerTile widgets
  - Scene activation bar (triggers scene engine via API)
  - Real-time state updates via WebSocket subscription
  - Optimistic UI: subtle opacity pulse (0.2–1.0) for pending state
  - Exponential backoff reconnection for WebSocket
  - Location data caching
- Created `internal/location/` package (4 files):
  - **types.go** — Area, Room structs with settings
  - **errors.go** — Domain error sentinels
  - **repository.go** — SQLiteRepository (ListAreas, ListRooms, by-site/by-area queries)
  - **repository_test.go** — 270-line integration test suite
- Created `internal/panel/` package (1 file + embedded web build):
  - **panel.go** — `go:embed` web assets, SPA fallback handler, cache-control headers
  - **panel_test.go** — Handler tests (static files, SPA fallback, 404)
  - **web/** — Flutter web build output (canvaskit, assets, service worker)
- Modified `internal/api/`:
  - **locations.go** — Area/room list endpoints (GET /areas, GET /rooms)
  - **router.go** — Panel mount at `/panel/`, location routes
  - **devices.go** — Dev-mode command simulation (800ms delay + WS broadcast)
  - **middleware.go** — CORS config updates for dev
  - **websocket.go** — Enhanced broadcast for device state changes
- Modified `cmd/graylogic/main.go` — Location repo + panel handler wiring
- Modified `configs/config.yaml` — Dev mode flag, CORS origins
- **Tests**: 12 test files in Flutter (models, providers, services, widgets)
- UX: Dimmer slider holds sent value until WebSocket confirms (no snap-back)
- **M1.5 Flutter Wall Panel complete — Year 1 Foundation done!**

### Session 17: 2026-01-25 — KNXSim Engineer UI & Sync Fixes

**Goal:** Complete KNXSim Engineer UI and fix bidirectional state synchronisation

**KNXSim Enhancements** (`sim/knxsim/`):
- **Engineer UI** with interactive device controls:
  - Light switches: ON/OFF toggle button
  - Dimmers: brightness slider (0-100%)
  - Blinds: position + slat angle sliders
  - Presence sensors: motion trigger button + lux slider (0-2000 lx)
  - Sensors: temperature display
- **Telegram Inspector**: Live bus traffic via WebSocket
- **Device Panel**: GA table with auto-detected DPT types, raw state JSON
- **Command API**: `POST /devices/{id}/command` now sends KNX telegrams on bus

**Critical Sync Fixes**:
- **KNXSim command → KNX telegram**: Commands now call `_send_telegram_with_hook()` to broadcast on bus (was only updating internal state)
- **Core state merge**: Changed SQLite `SET state = ?` to `json_patch()` — fields now accumulate instead of overwriting
- **Registry cache merge**: Cache update merges state fields instead of replacing entire state
- **Multi-response telegrams**: Dimmers return both `switch_status` AND `brightness_status` (like real KNX devices)
- **UI toggle refresh**: `updateState()` now calls `_renderControls()` to update button state

**Files Modified**:
- `sim/knxsim/static/js/components/device-panel.js` — Presence controls, slider suffix fix
- `sim/knxsim/api/routes_devices.py` — presence/lux commands, DPT9 encoding
- `sim/knxsim/devices/light_dimmer.py` — Return list of response telegrams
- `sim/knxsim/core/premise.py` — Handle list of cEMI responses
- `sim/knxsim/knxip/server.py` — Send multiple response telegrams
- `code/core/internal/device/repository.go` — json_patch for state merge
- `code/core/internal/device/registry.go` — Merge state in cache

**Documentation**:
- Updated `sim/knxsim/VISION.md` — Phase 2.3, 2.4 marked complete
- Updated `CHANGELOG.md` — Version 1.0.9 entry

**Result**: Full bidirectional sync working: Flutter ↔ Core ↔ KNXSim

---

## Notes

- Project started: 2026-01-18
- Architecture pivoted from openHAB/Node-RED to custom Go Core (v1.0.0)
- Old documentation archived in `docs/archive/v0.4-openhab-era.zip`
- This is a 5-year part-time project
- First real deployment target: own home (Year 1)
- Using UK English throughout (per project standards)
- All timestamps in UTC (per CODING-STANDARDS.md)
