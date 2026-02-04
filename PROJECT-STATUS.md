# Gray Logic — Project Status

> **Last Updated:** 2026-02-04
> **Current Phase:** Implementation (Year 1 Complete + Commissioning Tools)

---

## RESUME HERE — Next Session

**Last session:** 2026-02-04 (Session 31 - KNXSim Topology Refactor)
**Current milestone:** Year 1 Foundation Complete + KNXSim Phase 2 Complete + Topology Refactor

**What's done:**
- M1.1 Core Infrastructure (SQLite, MQTT, InfluxDB, Config, Logging) ✅
- M1.2 KNX Bridge (telegram parsing, knxd client, MQTT translation, 4 audit cycles) ✅
- M1.3 Device Registry (50+ types, thread-safe cache, SQLite persistence, KNX integration) ✅
- M1.4 REST API + WebSocket (Chi router, device CRUD, state commands, WebSocket hub, JWT auth) ✅
- M1.5 Flutter Wall Panel (Riverpod, Dio, WebSocket, optimistic UI, embedded web serving) ✅
- M1.6 Basic Scenes (automation package, scene engine, parallel execution, REST API, 91.6% coverage) ✅
- KNXSim Phase 1 ✅ (KNXnet/IP tunnelling, 57 templates, DPT codec, scenarios, multi-premise)
- KNXSim Phase 2 ✅ COMPLETE — Web Dashboard:
  - 2.1 Building Overview (premise selector, floor nav, room grid, building summary stats)
  - 2.3 Device Controls (entity tiles, quick controls, multi-channel)
  - 2.4 Technical Inspection (device panel, telegram inspector, bus stats, filtering)
  - 2.5 Device/Room Management (full CRUD, template browser, topology, GA hierarchy, conflict detection)
  - Codex consolidation (107 tests: DPT codec, cEMI frames, GA normalisation)
- **KNXSim Topology Refactor ✅ NEW** — One Premise = One TP Line:
  - Premise now stores `area_number`/`line_number`, gateway/client addresses derived
  - Removed Area/Line CRUD (20+ methods, 12 API endpoints, UI modals)
  - Flat topology view with educational info banner
  - IA remapping on sample load (Codex fix)
  - ETS export uses premise area/line (Codex fix)
  - Reference guide expanded: physical topology, address meaning, port vs IP
  - 146 tests passing, async httpx client (Codex improvement)
- ETS Import Commissioning ✅ (parser, device detection, location auto-creation, room assignment fix)
- ETS Device Classification Pipeline ✅ (Tier 1 Function Types + Tier 2 DPT fallback, manufacturer metadata, 14 new tests)
- KNX Pipeline Robustness ✅ (structured DPT storage, canonical function registry, import normalisation, pipeline integration tests)
- Admin Interface ✅ (metrics, devices, import, discovery tabs)
- GA Recorder ✅ (replaces BusMonitor for commissioning)
- Dev Workflow Restructure ✅ (native Go dev, Docker support services, filesystem panel serving)
- MQTT Topic Scheme Unification ✅ (flat scheme, fixed scene routing bug, 23 files updated)
- Dev Environment Fixes ✅ (knxd→knxsim, localhost API bind, bridge device mappings, end-to-end state flow)
- Flutter Dep Upgrade ✅ (Riverpod v3, file_picker v10, dio v5.9, all 55 tests passing, 3s analyze)

**What's next:**
- **Priority 1**: Auth hardening (production JWT, refresh tokens, role-based access)
- **Priority 2**: Year 2 kick-off (M2.1 Area/Room hierarchy, M2.2 advanced scenes)
- **Parking lot**: ETS domain filter i18n, KNXSim config.yaml schema validation
- **Future architecture**: KNXSim multi-container mode (one container per premise = separate IPs like real hardware)

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
| M2.5 | DALI bridge (Lunatone IoT Gateway REST API integration) |
| M2.6 | Blind control |
| M2.7 | Climate integration (Modbus → HVAC) |
| M2.8 | Flutter mobile app |
| M2.9 | Matter bridge (Apple/Google/Alexa via Matterbridge) |

### Year 3 — Integration & Resilience (2028)
| Milestone | Goal |
|-----------|------|
| M3.1 | Audio matrix (HTD/Russound) |
| M3.2 | Video matrix (Atlona) |
| M3.3 | Security panel (monitoring only) |
| M3.4 | CCTV (RTSP/ONVIF) |
| M3.5 | BACnet bridge (commercial building systems) |
| M3.6 | Advanced conditional logic |
| M3.7 | System Supervisor (Level 1) |
| M3.8 | Machine-readable error catalog |
| M3.9 | Multi-site installer dashboard |
| M3.10 | DALI PHM analytics (trending, predictive alerts) |

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

### Session 18: 2026-01-25 — KNXSim Phase 2.6 Alpine.js Refactor

**Goal:** Refactor UI to Alpine.js, add project export, improve dev tooling

**Alpine.js Refactor** (`sim/knxsim/static/`):
- Added `vendor/alpine.min.js` — Bundled locally (no CDN dependency)
- Created `store.js` — Global Alpine store for reactive state management
- Rewrote `index.html` with declarative Alpine templates (`x-data`, `x-for`, `x-show`, `@click`)
- Deleted vanilla JS components: `room-grid.js`, `device-panel.js`, `telegram-inspector.js`, `websocket.js`
- Updated `app.js` — Alpine init + WebSocket integration with store

**KNX Project Export** (`api/routes_export.py`):
- `GET /premises/{id}/export/knxproj` — ETS-compatible .knxproj file (ZIP with XML)
- `GET /premises/{id}/export/esf` — ETS Symbol File (.esf) for group addresses
- Added export dropdown button in UI header (Edit Mode only)

**UI Improvements**:
- Added `knxsim.svg` logo to header (64px, header height increased to 80px)
- Changed "Engineer Mode" label to "Edit Mode"
- Added `glcore.svg` for future use

**Development Tooling**:
- Added `ruff.toml` — Linter config (Python 3.12, line-length 100)
- Added `pyrightconfig.json` — IDE type checking support
- Added `.gitignore` — Excludes `.venv/`
- Ran `ruff --fix` — Auto-fixed 133 lint issues across Python codebase:
  - `Optional[X]` → `X | None` (modern syntax)
  - `from typing import Callable` → `from collections.abc import Callable`
  - Removed unused imports, sorted imports, prefixed unused vars with `_`
- Created local `.venv` for Pyright/Zed IDE support

**Files Created**: 8 new files
**Files Modified**: 25+ Python/JS/CSS files
**Files Deleted**: 4 vanilla JS component files

---

### Session 19: 2026-01-26 — KNXSim Wall Switch Support

**Goal:** Add interactive wall switch controls and fix shared GA handling

**Wall Switch Controls** (`sim/knxsim/static/`):
- Added push button controls to Edit Mode panel for template devices
- `toggleButton()` — Toggle button state and send command via shared GA
- `getButtonGAs()` — Extract button GAs from device for UI rendering
- `guessDPT()` — Recognises `button_*` and `led_*` patterns → returns "1.001"

**Shared GA Fix** (`sim/knxsim/devices/template_device.py`):
- Changed `_ga_info` from `dict[int, tuple]` to `dict[int, list[tuple]]`
- Multiple buttons sharing same GA now all update when telegram arrives
- Structure: `{ga: [(slot_name, field, dpt, direction), ...]}`

**API & Manager Fixes**:
- `api/routes_devices.py` — Updated to handle new `_ga_info` list structure
- `core/premise_manager.py` — Added `_parse_ga()` inline (was importing non-existent module)
- `core/premise_manager.py` — `update_device()` rebuilds `_ga_info` for live GA editing

**Files Modified**: 6 files (`template_device.py`, `routes_devices.py`, `premise_manager.py`, `index.html`, `store.js`, `style.css`)

**Result**: Wall switches work in Edit Mode — pressing button_1 or button_2 sends telegram to shared GA, all linked buttons and lights update correctly.

---

### Session 20: 2026-02-01 — KNXSim Topology Restructure Phase 1

**Goal:** Add proper KNX topology (Area/Line/Device) alongside building view (Floor/Room)

**Backend — Schema & Migration** (`sim/knxsim/persistence/db.py`):
- New `areas` table: id, premise_id, area_number (0-15), name, description
- New `lines` table: id, area_id, line_number (0-15), name, description
- Added `line_id`, `device_number` columns to `devices`
- Auto-migration: parses existing individual_address → creates topology
- `ensure_topology_for_ia()` auto-creates Area/Line when IA provided

**Backend — API Endpoints** (`sim/knxsim/api/routes_topology.py`):
- `GET /premises/{id}/topology` — Full tree (Areas → Lines → Devices)
- CRUD for `/premises/{id}/areas/*` and `/areas/{id}/lines/*`
- Duplicate area_number/line_number detection (409 Conflict)

**Backend — Reference API** (`sim/knxsim/api/routes_reference.py`):
- `/reference/individual-address` — IA structure guide
- `/reference/ga-structure` — GA hierarchy patterns
- `/reference/flags` — Communication flags (CRWTUI) documentation
- `/reference/dpts` — Complete DPT catalog (50+ types, categorized)
- `/reference/device-templates` — Recommended GAs by device type

**Frontend — View Switcher** (`sim/knxsim/static/`):
- New view switcher tabs: "🏠 Building View" | "🔌 Topology View"
- Building View: Existing Floor/Room layout unchanged
- Topology View: Collapsible tree of Areas → Lines → Devices

**Frontend — Topology View**:
- Area cards with expand/collapse, line count
- Line headers with device count, expand/collapse
- Device rows showing: IA, icon, name, type, room assignment
- Engineer Mode: Add/Edit/Delete buttons for Areas, Lines
- Add Device button on each line with suggested IA

**Frontend — Reference Store** (`sim/knxsim/static/js/store.js`):
- `$store.reference` — DPT catalog, device templates, GA/IA helpers
- `searchDpts(query)` — Search DPTs by id/name/use_case
- `suggestNextIA(areaLine)` — Suggest next device number on line
- `suggestNextGA(mainMiddle)` — Suggest next GA in group
- `getDevicesByLine()` — Group devices for reference panel

**Frontend — Device Modal Help Panel**:
- Tabbed interface: Recommended | All Devices | All GAs | IA Help | GA Help | DPTs | Flags
- "Suggest" button for Individual Address
- Reference data from backend loaded on demand

**Files Created**: 5 (`routes_topology.py`, `routes_reference.py`, `TOPOLOGY-RESTRUCTURE.md`, `CODEBASE.md`, `alpine-collapse.min.js`)
**Files Modified**: 12 (db.py, app.py, store.js, api.js, index.html, style.css, and others)
**Total Lines Added**: ~4,300

**Result**: Phase 1 complete — Topology schema, API, and UI working. Devices auto-migrated to topology on startup.

---

### Session 21: 2026-02-03 — ETS Import, Admin UI & KNXSim Thermostats

**Goal:** Commissioning workflow and realistic climate simulation

**ETS Import Commissioning** (`internal/commissioning/etsimport/`):
- Created ETS 5/6 XML parser for .knxproj files
- Device detection with confidence scoring (80%+ match threshold)
- 3-level GroupRange hierarchy for room extraction
- Auto-create locations (areas/rooms) from imported topology
- Suggested room pre-populated in Flutter import UI
- 6 files: parser.go, types.go, detection.go, errors.go, doc.go + tests

**Admin Interface** (`code/ui/admin/`):
- New admin UI with 4 tabs: Metrics, Devices, Import, Discovery
- Discovery tab shows KNX bus scan data from GA Recorder
- Import tab for ETS project file upload

**GA Recorder** (`internal/bridges/knx/ga_recorder.go`):
- Records group address traffic with timestamps
- Replaces BusMonitor for commissioning discovery
- Integrates with health check cycling

**KNXSim Thermostats** (`sim/knxsim/devices/thermostat.py`):
- Room thermostat with internal PID controller
- Valve actuator control output (0-100%)
- Setpoint adjustment via KNX commands
- Thermal simulation scenario for realistic heating behaviour
- Valve percentage display on heating actuator channels

**KNXSim Enhancements**:
- 28 new device templates added
- Multi-channel actuator LED indicators
- Loads system for physical equipment simulation
- Channel state synchronisation improvements
- Collapsible telegram inspector
- Template browser layout improvements

**Flutter Panel Improvements**:
- ETS import and onboarding screens
- Thermostat tile UI with setpoint control
- Auto-detect API host for embedded panel
- Room selection from suggested_room

**Files Created**: 10+ (commissioning package, admin UI, GA recorder)
**Files Modified**: 30+ across Go, Flutter, Python
**Commits**: 75 since last changelog entry

---

### Session 22: 2026-02-03 — Dev vs Prod Workflow Restructure

**Goal:** Eliminate Docker rebuild cycle for Go development

Switched to native Go development with containerised support services. Go core now runs natively (~2-3s rebuild) while Mosquitto, InfluxDB, and KNXSim run in Docker. Added filesystem panel serving for Flutter hot reload without Go recompile.

**Key changes**: Makefile dev/prod targets, `docker-compose.dev.yml` stripped to 3 services, `panel.Handler(dir)` for filesystem serving, config defaults to localhost.

**Files Modified**: 15+ across Go, Docker, docs
**See**: CHANGELOG.md v1.0.13 for full details

---

### Session 26: 2026-02-03 — Flutter Dependency Upgrade & Riverpod v3 Migration

**Goal:** Upgrade all outdated Flutter dependencies and migrate Riverpod v2→v3; fix all pre-existing test failures

**Part 1 — Stack Recovery & Nuke-Rebuild Command** (`ce2dead`):
- Recovered from crashed session — brought down stale services, purged databases
- Diagnosed GLCore "crash" as shell timeout issue (must use `nohup` + `&`)
- Created `/nuke-rebuild` Claude command — 5-phase scorched-earth stack reset
- Rebuilt and verified full stack from scratch (all services healthy, API responding)

**Part 2 — Flutter Dependency Upgrade** (`4f41654`):
- Upgraded: `flutter_riverpod` ^2.6.1→^3.2.0, `dio` ^5.7.0→^5.9.1, `file_picker` ^5.5.0→^10.3.10
- Migrated 6 StateNotifier classes to Riverpod v3 Notifier pattern:
  - `PendingDevicesNotifier`, `RoomDevicesNotifier` (most complex — WebSocket lifecycle in `build()`)
  - `AuthNotifier`, `LocationDataNotifier`, `ETSImportNotifier`, `RoomScenesNotifier`
- Fixed 10 `AsyncValue.valueOrNull` → `.value` occurrences across 5 screen files
- Fixed 2 device_provider tests (expected wrong optimistic update behaviour)
- Added `build-panel-dev` Makefile target (`--no-tree-shake-icons` for faster builds)
- Added `internal/panel/web/` to `.gitignore` (caught generated build output before it stayed in repo)
- Analysis time: 32s → ~3s (file_picker v10 eliminates heavy transitive dependency crawl)

**Part 3 — Pre-existing Test Failures** (`4c057a0`):
- Removed `dart:html` import and 80-line `_pickFileWeb()` from `ets_import_screen.dart` — `file_picker` v10 handles web natively, making the `dart:html` workaround redundant
- Fixed websocket test: port 99999 (exceeds TCP max) → port 19, added exception handling, strengthened lifecycle assertion
- Test results: 51/55 → **55/55 passing**

**Result:** All Flutter dependencies current, Riverpod v3 fully migrated, zero analyzer errors, 55/55 tests passing, production web build verified.

**Files Modified**: 16 Dart/config files + Makefile + .gitignore
**Commits**: 3

---

### Session 27: 2026-02-03 — KNX Device Classification Pipeline Upgrade

**Goal:** Pass device metadata (manufacturer, model, application program, ETS Function Types) through the entire pipeline — templates → export → parse → import → DB

**Step 1 (prior session)**: Added `manufacturer:` blocks to all 47 YAML templates with realistic KNX manufacturer data (ABB, MDT, Siemens, Gira, Elsner, Theben).

**Step 2 — Thread metadata through KNXSim runtime**:
- `loader.py`: Parse `manufacturer:` block in `DeviceTemplate.__init__()` with legacy flat-format fallback
- `routes_templates.py`: Inject manufacturer metadata into device config dict on `/from-template` API
- `routes_premises.py`: Template lookup for `config.yaml` devices on `reset-sample`

**Step 3 — Upgrade .knxproj export** (`routes_export.py`):
- Added `<Topology>` section with `<DeviceInstance>` (IndividualAddress, ProductRefId, ApplicationProgramRef, ComObjectInstanceRefs)
- Added `<ManufacturerData>` section (Manufacturer, Hardware, Product, ApplicationProgram)
- Replaced `FT-*` codes with standard ETS Function Types (SwitchableLight, DimmableLight, Sunblind, etc.)
- Added Comment attribute for Custom function types carrying template ID

**Step 4 — Upgrade GLCore parser** (`parser.go`):
- Added XML structs for DeviceInstance, Function, Manufacturer, etc.
- Added `extractFunctionDevices()` for Tier 1 classification from Functions/Topology/ManufacturerData
- Added `removeConsumedGAs()` and `buildGAIDToAddrMap()` helpers
- Refactored `parseKNXProj` → `parseKNXProjWithXML` returning raw XML for multi-pass parsing
- Updated `ParseBytes()`: Tier 1 runs first, consumed GAs filtered before Tier 2

**Step 5 — Upgrade GLCore import** (`commissioning.go`):
- Added metadata fields to `ETSDeviceImport`: Manufacturer, ProductModel, ApplicationProgram, IndividualAddress
- `buildDeviceFromImport()` populates dev.Manufacturer, dev.Model, address map entries
- Expanded `deriveCapabilitiesFromAddresses()` from ~10 to 27 function→capability mappings

**Step 6 — Function Type mapping** (`detection.go`):
- `functionTypeToDeviceType()`: 7 standard ETS types → GLCore types (0.95–0.99 confidence)
- `commentToDeviceType()`: 40+ KNXSim template IDs → GLCore types (0.98 confidence)

**Step 7 — New device types** (`types.go`):
- Added 12 types: scene_controller, push_button, binary_input, room_controller, logic_module, ip_router, line_coupler, power_supply, timer_switch, load_controller, multi_sensor, wind_sensor

**Step 8 — DB migration**: NOT needed — `application_program` stored in `address` JSON map; `manufacturer`/`model` columns already exist.

**Step 9 — Tests** (`parser_test.go`):
- 8 unit tests for `functionTypeToDeviceType`
- Table-driven `commentToDeviceType` test (14 subtests)
- Full integration test with Topology + ManufacturerData + Functions XML
- Tier 2 fallback + CSV regression tests
- All 15 Go packages pass, golangci-lint clean

**Files Modified**: 57 files (47 YAML templates + 7 Go files + 3 Python files)

---

### Session 25: 2026-02-03 — Registry-Only Devices & ETS Auto Location

**Goal:** Remove redundant static YAML device config; implement automatic location extraction from ETS import

**Part 1 — Registry-Only Device Config** (`a7f261a`):
- Cleared `knx-bridge.yaml` devices to `[]` — bridge starts empty
- Removed `seedDeviceRegistry()` — no more YAML→registry pollution on startup
- Removed `BuildDeviceIndex()` call from `NewBridge()` — maps start empty
- Made `loadDevicesFromRegistry()` the sole source of device→GA mappings
- Removed "skip if already in config" check — registry is primary, not secondary
- Fixed health reporting to use registry-loaded count
- Removed 8 dead helper functions and 6 dead test functions (~570 net lines removed)
- Added `createTestBridge()` test helper, updated 18 tests

**Part 2 — ETS Location Extraction** (`6e28ca8`):
- Implemented `extractLocations()` — builds `Location` objects from GA hierarchy paths
- Algorithm: collect paths → build tree → filter domain names (EN+DE) → classify leaf=room, non-leaf=floor
- Deduplicates by slug across domains (same room under Lighting + HVAC = 1 location)
- Sorts areas before rooms for correct parent creation order
- Added `autoMapDeviceLocations()` — maps `SuggestedRoom`→`RoomID` after location creation
- Added `SuggestedRoom`/`SuggestedArea` fields to `ETSDeviceImport`
- 6 new tests covering domain-first, location-first, 3-level, dedup, empty, device source

**Result:** Fresh system + `.knxproj` import → areas, rooms, and devices auto-populated with zero manual intervention.

**Files Modified**: 7 Go files + 1 YAML config
**Commits**: 2

---

### Session 24: 2026-02-03 — Dev Environment Fixes & Bridge Config

**Goal:** Fix dev environment connectivity and enable end-to-end KNX state flow

Fixed three dev environment issues: knxd connected to old Windows VM IP instead of knxsim container, API broadcast on all interfaces, and KNX bridge had empty device mappings. Also seeded areas/rooms via API for Flutter panel room navigation.

**Key changes**: `config.yaml` knxd host→localhost, API bind→127.0.0.1, populated `knx-bridge.yaml` with 15 devices, seeded 5 rooms via API, assigned devices to rooms.

**Files Modified**: 2 config files
**Commits**: 2 (`fix(config): dev defaults` + `docs: sync project logs`)

---

### Session 23: 2026-02-03 — Unified MQTT Topic Scheme

**Goal:** Eliminate incompatible MQTT topic schemes and fix scene command routing bug

Two MQTT topic schemes coexisted: flat (`graylogic/{category}/{protocol}/{address}`) used by the KNX bridge, and bridge-wrapped (`graylogic/bridge/{bridge_id}/{category}/{device_id}`) used by topic helpers and the scene engine. This caused a live bug — scene commands vanished because the engine published to topics no bridge subscribed to.

**Standardised on flat scheme** across all Go code and documentation:
- Rewrote `topics.go` bridge methods: `(bridgeID, deviceID)` → `(protocol, address)`
- Fixed `engine.go`: scene commands now route to `graylogic/command/{protocol}/{device_id}`
- Removed `deriveBridgeID()` — protocol name is the routing key
- Added 3 new topic methods + 5 new wildcard methods to match full 8-category scheme
- Updated all 16 documentation/config files — zero `graylogic/bridge/` references remain
- Updated DALI bridge planning notes to reflect resolved topic scheme

**Verification**: `go test ./...` all pass, `go vet` clean, comprehensive grep confirms zero old-scheme references.

**Files Modified**: 7 Go files, 16 doc/config files (23 total)
**Commits**: 2 (`fix(mqtt): unify topic scheme` + `docs(mqtt): update all topic references`)

---

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

### Session 29: 2026-02-04 — KNX Pipeline Robustness Refactor

**Goal:** Eliminate fragile DPT inference and function name coupling across the KNX mapping pipeline

The import → registry → bridge pipeline was a "house of cards" — DPTs were discarded at import time (`addresses[fn] = addr.GA`), forcing the bridge to guess them via substring matching. Any unrecognised function name → empty DPT → raw bytes in state → broken UI. Function names were also the single coupling point across layers with no shared definition.

**Phase A — Structured DPT storage:**
- New device address format: `{"functions": {"switch": {"ga": "1/0/1", "dpt": "1.001", "flags": ["write"]}}}`
- Removed redundant `group_address` top-level key (stored only one arbitrary GA)
- `buildDeviceFromImport()` now preserves DPT/flags from ETS import
- Bridge reads stored DPT/flags, falling back to inference only for pre-migration devices
- One-time `MigrateKNXAddressFormat()` converts existing devices on startup
- Updated validation, all test fixtures (5 test files), Flutter Add Device form

**Phase B — Canonical function registry:**
- New `functions.go`: ~55 function definitions with Name, StateKey, DPT, Flags, Aliases
- `NormalizeFunction()` resolves aliases (`actual_temperature` → `temperature`, `on_off` → `switch`)
- `NormalizeChannelFunction()` handles channel prefixes (`ch_a_switch` → prefix + canonical)
- `StateKeyForFunction()` replaced 30-line hardcoded `functionToStateKey` map
- Bridge's `inferDPT/inferFlags` now use canonical registry first, heuristic fallback second
- Normalisation wired into `buildDeviceFromImport()` at import time
- 13 comprehensive tests (consistency, aliases, channels, duplicates)

**Phase C — Pipeline integration tests:**
- `pipeline_test.go`: 12 telegram→state tests (switch, dimmer, blind, thermostat, PIR, valve, infra channels)
- 4 command→telegram tests (on, off, dim, set_position)
- Device count + channel state key verification
- All run with mock MQTT/knxd — no external dependencies

**Phase D — KNXSim validation:**
- Thermal telegram encoding verified correct (cEMI long-frame encoding is sound, DPT 9.xxx round-trips perfectly)
- `_guess_dpt()` extended from ~20 to ~45 patterns, fixed `stop` DPT (1.010→1.007), fixed ordering bugs
- Canonical name alignment verified: all KNXSim normalised names resolve via Go's `NormalizeFunction()`

**New files:** `functions.go`, `functions_test.go`, `pipeline_test.go`, `device_types.dart`
**Modified:** 20+ files across Go core, Flutter, KNXSim
**Tests:** All pass (`go test ./...` green, `flutter analyze` clean)

---

### Session 28: 2026-02-03 — ETS Import Room Assignment Fix

**Goal:** Fix devices not being assigned to GLCore rooms/areas after ETS import

Devices imported via the Flutter panel were ending up with empty `room_id` and `area_id` in the database, even though KNXSim correctly assigned them in the ETS hierarchy. The Go backend was correct — `createLocationsFromETS()` → `autoMapDeviceLocations()` → `createNewDevice()` — but Flutter was silently dropping location data.

**Root cause — three Flutter data-passing gaps:**
1. `ETSParseResult` didn't have a `locations` field — parse response locations silently discarded
2. `ETSImportRequest.toJson()` didn't include `locations` — Go received `locations: 0`
3. `_copyParseResultWithDevices()` didn't preserve `locations` — lost on any device toggle/edit

**Additional discovery:**
- `cp -r build/web ../../core/internal/panel/web` was creating nested `web/web/` — GLCore served old build without fixes

**Fix:**
- Added `locations` field + `_parseLocations()` to `ETSParseResult`
- Added `locations` to `ETSImportRequest.toJson()`
- Added `suggested_room`/`suggested_area` to `ETSDetectedDevice.toImportJson()`
- Extracted `_copyParseResultWithDevices()` helper preserving all fields including `locations`
- Added summary log to `commissioning.go` for import request diagnostics
- Fixed panel deployment: `rm -rf` + `cp -r build/web/*` instead of `cp -r build/web`

**Verified:** Fresh nuke-rebuild → ETS import → all 17 devices assigned to correct rooms and areas.

**Files Modified**: 3 (ets_import.dart, ets_import_provider.dart, commissioning.go) + panel web assets
**Commits**: 2 (`2fbe1f7`, `8321a07`)

---

### Session 30: 2026-02-04 — KNXSim Phase 2 Complete

**Goal:** Finish remaining KNXSim Phase 2 dashboard items and close out the simulator milestone

**Part 1 — VISION.md Audit & Correction:**
- Cross-referenced VISION.md checkboxes against actual codebase (JS stores, HTML, backend routes)
- Found Phase 2 was ~95% complete but VISION.md showed it as ~50% — massive checkbox debt
- Corrected all Phase 2.1, 2.4, 2.5 checkboxes to reflect true state
- Moved low-priority items (sparklines, bulk ops, custom templates) to Phase 3 backlog

**Part 2 — Bus Statistics Panel (2.4):**
- Backend: Enhanced `TelegramInspector` with per-premise TX/RX counters and GA frequency maps
- Backend: `get_stats()` now returns `total_recorded`, `rx_count`, `tx_count`, `unique_gas`, `top_gas` (top 5)
- Frontend: Collapsible stats panel in telegram inspector with 5 stat cards + busiest GA chips
- Auto-refreshes every 5s while visible via `setInterval`

**Part 3 — Telegram Filtering (2.4):**
- Backend: Added `direction`, `device`, `ga` query params to `GET /telegrams` endpoint
- Backend: `get_history()` filters by direction, device_id substring, GA substring server-side
- Frontend: Added debounced text search input (300ms) filtering across device, GA, source, value
- Two-tier filtering: server-side for REST API, client-side for live WebSocket stream

**Part 4 — Building Summary Stats (2.1):**
- Added `getBuildingSummary()` to app store — aggregates lights on, avg temp, presence rooms, blinds open
- Computed client-side from existing device state (reactive, no new endpoint)
- Enhanced footer stats bar with live building status + visual separators

**Part 5 — Address Conflict Warnings (2.5):**
- Device ID conflict: proactive warning in create modal when ID already exists
- IA conflict: proactive warning in create/edit modal when individual address in use
- Client-side validation using in-memory store (instant feedback, no round-trip)
- Backend already had `ConflictError` on save — these warnings prevent the user reaching that point

**Files Modified**: 6 (`VISION.md`, `telegram_inspector.py`, `routes_telegrams.py`, `store.js`, `index.html`, `style.css`)
**Lines Changed**: +479, -70
**Tests**: 107 passing (all existing tests unaffected)
**Commit**: `24efb2d`

**Result**: KNXSim Phase 2 Web Dashboard is now complete. All checklist items done. Phase 3 (scenarios, thermal models, testing tools) is next when KNXSim work resumes.

---

## Notes

- Project started: 2026-01-18
- Architecture pivoted from openHAB/Node-RED to custom Go Core (v1.0.0)
- Old documentation archived in `docs/archive/v0.4-openhab-era.zip`
- This is a 5-year part-time project
- First real deployment target: own home (Year 1)
- Using UK English throughout (per project standards)
- All timestamps in UTC (per CODING-STANDARDS.md)
