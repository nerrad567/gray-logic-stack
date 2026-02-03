# Changelog – Gray Logic

All notable changes to this project will be documented in this file.

---

## 1.0.15 – Dev Environment Fixes & KNX Bridge Device Config (2026-02-03)

**Focus: Fix dev environment connectivity and enable KNX state flow**

Dev environment had three issues: knxd pointed at a hardcoded Windows VM IP instead of the knxsim container, the API server bound to all interfaces (broadcasting on the network), and the KNX bridge had zero device mappings so all telegrams from knxsim were silently dropped.

### Fixed

- **knxd backend host**: Changed from `192.168.4.34` (old Windows KNX Virtual VM) to `localhost` — connects to knxsim container exposed on `127.0.0.1:3671/udp`. Override via `GRAYLOGIC_KNXD_BACKEND_HOST` for physical gateways.
- **API bind address**: Changed from `0.0.0.0` to `127.0.0.1` — dev stays local-only. Override via `GRAYLOGIC_API_HOST=0.0.0.0` for production.

### Added

- **KNX bridge device mappings**: Populated `knx-bridge.yaml` with 15 devices matching knxsim config — 5 light switches (DPT 1.001), 5 thermostats (DPT 9.001 temp + setpoint, DPT 5.001 valve), 5 presence sensors (DPT 1.018 + DPT 9.004 lux). All status GAs have `transmit` flag for incoming telegram processing.

### Notes

- Identified that `knx-bridge.yaml` static device config is redundant with the ETS import + registry-based device loading. Plan to remove static YAML device config in favour of single source of truth: ETS import or Flutter admin panel.
- knxsim thermal simulation proactive UDP telegrams not reliably reaching knxd — boolean telegrams (presence) work, 2-byte float telegrams (temperature) don't. Needs investigation in knxsim tunnel server.

### Files Modified

- `code/core/configs/config.yaml` — knxd backend host, API bind address
- `code/core/configs/knx-bridge.yaml` — 15 device mappings with GAs, DPTs, flags

---

## 1.0.14 – Unified MQTT Topic Scheme (2026-02-03)

**Focus: Eliminate incompatible MQTT topic schemes, fix scene command routing**

Two incompatible MQTT topic schemes coexisted — the flat scheme (`graylogic/{category}/{protocol}/{address}`) used by the KNX bridge, and a bridge-wrapped scheme (`graylogic/bridge/{bridge_id}/{category}/{device_id}`) used by topic helpers and the scene engine. This caused a live bug where scene commands vanished because the engine published to topics the bridge wasn't subscribed to.

### Fixed

- **Scene command routing**: `engine.go` now publishes commands to `graylogic/command/{protocol}/{device_id}` — matching what bridges actually subscribe to. Previously used `graylogic/bridge/{bridge_id}/command/{device_id}` which no bridge listened to.

### Changed

- **MQTT topic helpers** (`topics.go`): All bridge methods now take `(protocol, address)` instead of `(bridgeID, deviceID)`. Output uses flat scheme: `graylogic/{category}/{protocol}/{address}`.
- **Removed `deriveBridgeID()`** from `engine.go` — no longer needed; protocol name is the routing key.
- **3 new topic methods**: `BridgeAck(protocol, address)`, `BridgeConfig(protocol)`, `BridgeRequest(protocol, requestID)`.
- **5 new wildcard methods**: `AllBridgeAcks()`, `AllBridgeDiscovery()`, `AllBridgeRequests()`, `AllBridgeResponses()`, `AllBridgeConfigs()`.
- **Documentation**: All 16 doc/config files updated to reference flat topic scheme. Zero remaining `graylogic/bridge/` references.

### Files Modified

- `internal/infrastructure/mqtt/topics.go` — Rewritten bridge methods
- `internal/automation/engine.go` — Fixed scene command topic + removed deriveBridgeID
- `internal/infrastructure/mqtt/publish.go`, `subscribe.go` — Updated doc comments
- `internal/device/validation.go` — Updated GatewayID comment
- `internal/infrastructure/mqtt/client_test.go` — Updated + expanded topic tests
- `internal/automation/engine_test.go` — Updated expectations, removed TestDeriveBridgeID
- 16 documentation and config files across `docs/`, `code/core/docs/`, `notes/`

---

## 1.0.13 – Dev vs Prod Workflow Restructure (2026-02-03)

**Focus: Fast native Go development with containerised support services**

Eliminates the 2-3 minute Docker rebuild cycle. Go core now runs natively on the host (~2-3s rebuild) while support services (Mosquitto, InfluxDB, KNXSim) run in Docker. Config defaults to localhost; production Docker overrides via environment variables.

### Added

- **Filesystem Panel Serving**: `panel.Handler(dir)` accepts a directory path to serve Flutter assets from the filesystem instead of embedded `go:embed` (dev mode). Falls back to embedded assets when dir is empty or missing (production).
- **PanelDir Config Field**: `Config.PanelDir` + `GRAYLOGIC_PANEL_DIR` env override wires filesystem panel serving through config → API server → router.
- **Makefile Dev Targets**: `dev-services`, `dev-services-down`, `dev-run`, `dev-run-quick` for fast native development loop.
- **Makefile Docker Targets**: `docker-build`, `docker-up`, `docker-down` replace placeholder TODOs.
- **`docker-compose.prod.yml`**: Full 4-service stack (mosquitto, influxdb, knxsim, graylogic) with Docker network for production testing and CI.
- **`docs/development/dev-workflow.md`**: Canonical dev workflow reference with prerequisites, quick start, troubleshooting.
- **Dev workflow sections** in `CLAUDE.md` and `AGENTS.md` — dev mode is the documented default for every session.

### Changed

- **`configs/config.yaml`**: MQTT broker host changed from `"mosquitto"` to `"localhost"` — works natively without Docker networking. Production overrides via `GRAYLOGIC_MQTT_HOST=mosquitto`.
- **`docker-compose.dev.yml`**: Stripped to 3 support services only (mosquitto, influxdb, knxsim). Removed `graylogic` service, Docker network, and unused volume. Added knxsim port bindings (`localhost:3671/udp`, `localhost:9090`).
- **Panel tests**: Updated all 5 existing calls from `Handler()` to `Handler("")`. Added `TestHandlerFilesystemMode` and `TestHandlerInvalidDirFallsBackToEmbed`.

### Removed

- **`code/core/docker-compose.yml`**: Deleted duplicate compose file with diverged credentials. Only root-level compose files now exist.

### Environment

- **knxd** installed on host (`apt-get install knxd`) — required for native dev mode. System service disabled (Gray Logic manages knxd as a subprocess).
- **Flutter SDK** installed at `/opt/flutter` (v3.38.9) — standalone tarball, no snap.

---

## 1.0.12 – ETS Import, Admin UI & KNXSim Thermostats (2026-02-03)

**Focus: Commissioning workflow and realistic climate simulation**

Major commissioning milestone: ETS project files can now be imported to auto-create devices, locations, and group addresses. New admin interface provides system visibility. KNXSim gains thermostat devices with internal PID control for realistic heating simulation.

### Added

- **ETS Import Commissioning** (`internal/commissioning/etsimport/`):
  - Parser for ETS 5/6 XML project files (.knxproj)
  - Device detection with confidence scoring (80%+ match required)
  - Group address extraction with 3-level hierarchy (main/middle/sub)
  - Room suggestion from ETS project structure
  - Auto-create locations (areas/rooms) from imported topology
  - 6 Go files: parser.go, types.go, detection.go, errors.go, doc.go + tests

- **Admin Interface** (`code/ui/admin/`):
  - Metrics tab with system health overview
  - Devices tab with device listing and state
  - Import tab for ETS project upload
  - Discovery tab with KNX bus scan data

- **GA Recorder** (`internal/bridges/knx/ga_recorder.go`):
  - Records group address traffic with timestamps
  - Replaces BusMonitor for commissioning discovery
  - Integrates with health check cycling

- **Flutter Panel Enhancements**:
  - ETS import and onboarding screens
  - Thermostat tile UI with setpoint control
  - Room selection pre-populated from ETS suggested_room
  - Auto-detect API host for embedded web panel

- **KNXSim Thermostat** (`sim/knxsim/devices/thermostat.py`):
  - Room thermostat with internal PID controller
  - Valve actuator control output (0-100%)
  - Setpoint adjustment via KNX commands
  - Thermal simulation scenario for realistic heating behaviour
  - Valve percentage display on heating actuator channels

- **KNXSim Device Templates**:
  - 28 new device templates added
  - Multi-channel actuator support with LED indicators
  - Loads system for physical equipment simulation
  - Channel state synchronisation improvements

### Changed

- `internal/bridges/knx/bridge.go` — Reload device mappings after ETS import
- `sim/knxsim/static/index.html` — Collapsible telegram inspector
- `sim/knxsim/static/js/store.js` — Extended GA format handling
- Device panel scroll fixes (absolute positioning)
- Template browser layout improvements (wider cards, modal)

### Fixed

- CGO/SQLite build configuration
- Extended GA format in multi-channel template generation
- Button state normalisation in channel updates
- Channel state routing by GA
- Stale pressed state in button normalisation

---

## 1.0.11 – KNXSim Wall Switch Support & Shared GA Handling (2026-01-26)

**Focus: Interactive wall switch controls and multi-field GA support**

Added interactive push button controls for wall switches in KNXSim Edit Mode, with proper handling of multiple buttons sharing the same Group Address (a common KNX pattern).

### Added

- **Wall Switch Controls** (`static/index.html`, `static/js/store.js`):
  - Push button controls in Edit Mode panel for template devices
  - `toggleButton()` — Toggle button state and send command to linked devices
  - `getButtonGAs()` — Extract button GAs from device for UI rendering
  - Buttons display ON/OFF state with visual toggle styling

- **DPT Detection** (`static/js/store.js`):
  - `guessDPT()` now recognises `button_*` and `led_*` patterns → returns "1.001"

### Fixed

- **Multiple fields per GA** (`devices/template_device.py`):
  - Changed `_ga_info` from single tuple to list of tuples per GA
  - Structure: `{ga: [(slot_name, field, dpt, direction), ...]}`
  - `on_group_write()` now iterates all mappings, updating all linked fields
  - Fixes: button_1 and button_2 on same GA now both update correctly

- **API command handler** (`api/routes_devices.py`):
  - Updated to handle new `_ga_info` list structure (was expecting 3-tuple, now 4-tuple in list)
  - Finds matching slot by command name for correct DPT lookup

- **Live GA editing** (`core/premise_manager.py`):
  - `update_device()` now rebuilds `_ga_info` when GAs are edited
  - Added inline `_parse_ga()` function (was importing from non-existent module)
  - GA edits take effect immediately without restart

- **Telegram callback** (`knxsim.py`):
  - `on_telegram` handles list of devices per GA (not just single device)

### Changed

- `devices/template_device.py` — `_ga_info` structure from `dict[int, tuple]` to `dict[int, list[tuple]]`
- `core/premise_manager.py` — Rebuild `_ga_info` with new 4-tuple list structure
- `api/routes_devices.py` — Iterate `_ga_info` list to find matching slot
- `static/css/style.css` — Added `.push-btn-ctrl` styles for wall switch buttons

---

## 1.0.10 – KNXSim Phase 2.6: Alpine.js Refactor & Export (2026-01-25)

**Focus: Complete UI refactor to Alpine.js with project export capabilities**

Major refactor of the KNXSim UI from vanilla JavaScript to Alpine.js for reactive, declarative UI. Added KNX project export functionality and development tooling improvements.

### Added

- **Alpine.js UI Framework** (`static/js/`):
  - `vendor/alpine.min.js` — Bundled Alpine 3.x (~15KB, no CDN dependency)
  - `store.js` — Global Alpine store for reactive state management
  - Declarative templates with `x-data`, `x-for`, `x-show`, `@click`, `:class`

- **KNX Project Export** (`api/routes_export.py`):
  - `GET /api/v1/premises/{id}/export/knxproj` — ETS-compatible project file (.knxproj)
  - `GET /api/v1/premises/{id}/export/esf` — ETS Symbol File (.esf) for group addresses
  - Export dropdown in UI header (Edit Mode only)

- **Logo Integration**:
  - `static/img/knxsim.svg` — Gray Logic KNXSim branding
  - `static/img/glcore.svg` — Gray Logic Core logo

- **Development Tooling**:
  - `ruff.toml` — Ruff linter configuration (Python 3.12, 100 char lines)
  - `pyrightconfig.json` — Pyright/basedpyright config for IDE support
  - `.gitignore` — Excludes `.venv/`
  - Added `ruff>=0.8.0` to requirements.txt

### Changed

- **UI Label**: "Engineer Mode" → "Edit Mode" (clearer terminology)
- **Header Height**: Increased to 80px to accommodate 64px logo
- **Python Type Hints**: Modernised to Python 3.10+ syntax (`X | None` instead of `Optional[X]`)
- **Import Style**: `from collections.abc import Callable` (modern pattern)

### Removed

- `static/js/components/room-grid.js` — Replaced by Alpine template
- `static/js/components/device-panel.js` — Replaced by Alpine template  
- `static/js/components/telegram-inspector.js` — Replaced by Alpine template
- `static/js/websocket.js` — Merged into store.js

### Fixed

- **133 Ruff lint issues** auto-fixed across Python codebase:
  - Unused imports removed
  - Import sorting standardised
  - Unused variables prefixed with `_`
  - Modern type annotation syntax

---

## 1.0.9 – KNXSim Engineer UI & Core Sync Fixes (2026-01-25)

**Focus: KNX Simulator enhancement and bidirectional state synchronisation**

Major improvements to the KNX/IP simulator with a fully functional Engineer UI, plus critical fixes to ensure Flutter ↔ KNXSim ↔ Core state synchronisation works correctly.

### Added

- **KNXSim Engineer UI** (`sim/knxsim/static/`):
  - Device panel with interactive controls (lights, blinds, presence, sensors)
  - Telegram Inspector with live WebSocket streaming
  - GA inspection with auto-detected DPT types
  - Presence sensor controls: motion trigger button + lux slider (0-2000 lx)
  - Real-time state updates via WebSocket

- **KNXSim API enhancements**:
  - `POST /devices/{id}/command` — Send commands with KNX telegram broadcast
  - Support for `presence` (DPT1) and `lux` (DPT9) commands
  - Commands now send status telegrams onto KNX bus (not just internal state)

- **DPT Codec expansion** (`sim/knxsim/dpt/`):
  - DPT9 (2-byte float) encode/decode for temperature, lux, humidity
  - Proper scaling and sign handling per KNX specification

### Fixed

- **KNXSim command API not sending telegrams** — The `/command` endpoint only updated internal state; now sends actual KNX telegrams via `_send_telegram_with_hook()` so Core sees changes
- **Core state merge overwrites** — `UpdateState` was replacing entire state JSON; now uses SQLite `json_patch()` for proper field merging (e.g., `on` + `level` accumulate)
- **Registry cache merge** — Cache update now merges state fields instead of replacing
- **Light dimmer multi-response** — `on_group_write()` now returns both `switch_status` AND `brightness_status` telegrams (like real KNX dimmers)
- **Engineer UI toggle not updating** — `updateState()` now calls `_renderControls()` to refresh button state

### Changed

- `sim/knxsim/devices/light_dimmer.py` — Return list of response telegrams
- `sim/knxsim/core/premise.py` — Handle list of cEMI responses
- `sim/knxsim/knxip/server.py` — Send multiple response telegrams
- `code/core/internal/device/repository.go` — Use `json_patch()` for state merge
- `code/core/internal/device/registry.go` — Merge state in cache, not replace
- `sim/knxsim/VISION.md` — Updated with Phase 2.3, 2.4 completion status

---

## 1.0.8 – M1.5 Flutter Wall Panel (2026-01-24)

**Milestone: M1.5 Flutter Wall Panel — Complete (Year 1 Foundation Done!)**

A Flutter-based wall panel UI served as an embedded web app from the Go binary, with real-time device control, scene activation, and optimistic UI updates.

### Added

- **Flutter wall panel app** (`code/ui/wallpanel/`):
  - Riverpod state management with Dio HTTP client
  - JWT auth flow with WebSocket ticket exchange
  - Room device grid layout (responsive)
  - SwitchTile (on/off toggle) and DimmerTile (slider) widgets
  - Scene activation bar (triggers scene engine)
  - Real-time state updates via WebSocket subscription
  - Optimistic UI with subtle opacity pulse animation (0.2–1.0) for pending states
  - Exponential backoff WebSocket reconnection
  - Location data caching for offline resilience
  - 12 test files (models, providers, services, widgets)

- **`internal/location/` package** — Area/room location hierarchy:
  - `Repository` interface with SQLite implementation
  - `ListAreas`, `ListAreasBySite`, `GetArea`, `ListRooms`, `ListRoomsByArea`, `GetRoom`
  - 270-line integration test suite

- **`internal/panel/` package** — Embedded web serving:
  - `go:embed` for Flutter web build assets
  - SPA fallback handler (client-side routing support)
  - Cache-control headers (no-cache for mutable assets)

- **API additions**:
  - `GET /api/v1/areas` — List areas (optionally by site)
  - `GET /api/v1/rooms` — List rooms (optionally by area)
  - `GET /panel/*` — Serve Flutter web UI
  - Dev-mode device command simulation (800ms delay + WS broadcast)

### Changed

- `internal/api/devices.go` — Enhanced command handler with dev-mode simulation
- `internal/api/middleware.go` — CORS configuration updates for dev origins
- `internal/api/websocket.go` — Improved broadcast for device state changes
- `configs/config.yaml` — Added dev mode flag, CORS origins
- `cmd/graylogic/main.go` — Wired location repository and panel handler

---

## 1.0.7 – M1.6 Basic Scenes (2026-01-23)

**Milestone: M1.6 Basic Scenes — Complete**

A scene engine for the Gray Logic Core: named collections of device commands that execute together (parallel or sequential) with delay/fade support, execution logging, and WebSocket event broadcasting.

**What Was Built**

- `internal/automation/` package — 7 files, ~1,800 lines:
  - **types.go** — `Scene`, `SceneAction`, `SceneExecution` structs with `DeepCopy()` methods
  - **errors.go** — Domain error sentinels (`ErrSceneNotFound`, `ErrSceneDisabled`, `ErrMQTTUnavailable`)
  - **validation.go** — `ValidateScene`, `ValidateAction`, `GenerateSlug` with category/priority bounds
  - **repository.go** — `SQLiteRepository` with full CRUD, execution logging, JSON-encoded actions
  - **registry.go** — Thread-safe in-memory cache wrapping Repository (RWMutex, deep-copy semantics)
  - **engine.go** — Scene execution engine: parallel/sequential action groups, delay_ms, fade_ms, MQTT publish, execution tracking
  - **doc.go** — Package documentation

- `internal/api/scenes.go` — 7 HTTP handlers for scene endpoints
- `migrations/20260123_150000_scenes.up.sql` / `.down.sql` — Schema for `scenes` and `scene_executions` tables

**Scene Engine Architecture**

```
POST /scenes/{id}/activate
        │
        ▼
  Engine.ActivateScene(ctx, sceneID, triggerType, triggerSource)
        │
        ├─ Load scene from Registry (cached)
        ├─ Check enabled → ErrSceneDisabled if false
        ├─ Create SceneExecution record (status: pending)
        ├─ Group actions by parallel flag
        ├─ Execute groups sequentially:
        │      Each group → goroutines + WaitGroup
        │        Per action: delay_ms → resolve device → MQTT publish (with fade_ms)
        ├─ Update execution (status, counts, duration_ms)
        └─ Broadcast "scene.activated" via WebSocket hub
```

**Action grouping:** First action starts group 1. Subsequent actions with `parallel=true` join the current group; `parallel=false` starts a new group.

**Narrow Interface Pattern (Engine Dependencies)**

```go
type DeviceRegistry interface {
    GetDevice(ctx context.Context, id string) (DeviceInfo, error)
}
type MQTTClient interface {
    Publish(topic string, payload []byte, qos byte, retained bool) error
}
type WSHub interface {
    Broadcast(channel string, payload any)
}
```

Adapters in `main.go` bridge concrete types (`*device.Registry`, `*mqtt.Client`, `*api.Hub`) to these interfaces.

**API Endpoints**

| Method | Path | Description | Response |
|--------|------|-------------|----------|
| GET | `/api/v1/scenes` | List scenes (filter: category, room_id, area_id, enabled) | 200 `{scenes, count}` |
| POST | `/api/v1/scenes` | Create scene | 201 Scene |
| GET | `/api/v1/scenes/{id}` | Get scene by ID | 200 Scene |
| PATCH | `/api/v1/scenes/{id}` | Update scene | 200 Scene |
| DELETE | `/api/v1/scenes/{id}` | Delete scene | 204 |
| POST | `/api/v1/scenes/{id}/activate` | Activate scene (async) | 202 `{execution_id, status}` |
| GET | `/api/v1/scenes/{id}/executions` | List execution history | 200 `{executions, count}` |

**main.go Wiring**

- WebSocket hub created before both Engine and API Server (shared via `Deps.ExternalHub`)
- `sceneDeviceRegistryAdapter` wraps `*device.Registry` → `automation.DeviceInfo`
- `sceneMQTTClientAdapter` wraps `*mqtt.Client` → `automation.MQTTClient`

**Files Modified**

- `internal/api/server.go` — Added `SceneEngine`, `SceneRegistry`, `SceneRepo`, `ExternalHub` to `Deps`
- `internal/api/router.go` — Added scene route group under protected routes
- `cmd/graylogic/main.go` — Wired automation package with adapters

**Tests Added**

- `internal/automation/validation_test.go` — Validation unit tests
- `internal/automation/repository_test.go` — SQLite repository tests
- `internal/automation/registry_test.go` — Registry tests with mock repo
- `internal/automation/engine_test.go` — Engine tests (parallel, delays, performance benchmark)
- `internal/automation/integration_test.go` — Full lifecycle + persistence tests
- `internal/api/scenes_test.go` — 18 HTTP handler tests (CRUD, filtering, activation, executions)

**Test Coverage**

- 91.6% coverage on `internal/automation/` package (target: 80%)
- 60+ new tests across 6 test files
- 10-device parallel scene activation verified <500ms
- All packages build, test, and lint clean

**Design Decisions**

- **External hub injection**: Hub created in `main.go` rather than inside `Server.Start()`, enabling engine to broadcast without circular dependencies
- **Adapter pattern**: Engine uses narrow interfaces; main.go provides adapters — keeps automation package decoupled from device/mqtt/api packages
- **JSON-encoded actions**: Scene actions stored as JSON array in SQLite `actions` column — avoids join table complexity for M1.6 scope
- **202 Accepted**: Scene activation is asynchronous — MQTT commands are published, device state updates arrive via WebSocket
- **UK English**: All field names use British spelling (`colour`, not `color`) — project controls both server and clients

**Next: M1.5 — Flutter Wall Panel (or Auth hardening)**

---

## 1.0.6 – M1.4 REST API + WebSocket (2026-01-23)

**Milestone: M1.4 REST API + WebSocket — Complete**

The HTTP API server and WebSocket real-time layer are fully implemented, tested, and integrated into the Gray Logic Core. User interfaces (Flutter wall panels, mobile apps, web admin) can now interact with the device registry via REST endpoints and receive real-time state updates via WebSocket.

**What Was Built**

- `internal/api/` package — 9 files, ~2,000 lines:
  - **Server lifecycle**: `New()`, `Start()`, `Close()`, `HealthCheck()` following existing infrastructure patterns
  - **Chi router** (`go-chi/chi/v5`): Stdlib-compatible, minimal HTTP routing
  - **Device CRUD**: List (with filtering), Get, Create, Update, Delete
  - **Device state**: GET current state, PUT command (publishes to MQTT → KNX bridge)
  - **WebSocket hub**: Real-time broadcast of `device.state_changed` events
  - **Auth placeholder**: JWT login (hardcoded dev user), ticket-based WebSocket auth
  - **Middleware stack**: Request ID, structured logging, panic recovery, CORS
  - **TLS support**: Optional `ListenAndServeTLS` when config enables it

**Command Flow (API → Physical Device → WebSocket)**

```
Client → PUT /api/v1/devices/{id}/state
    → API validates + publishes to MQTT: graylogic/command/{protocol}/{deviceID}
    → KNX Bridge receives command, sends KNX telegram
    → Physical device responds → KNX telegram back
    → Bridge publishes state update to MQTT
    → API server receives via MQTT subscription
    → WebSocket Hub broadcasts device.state_changed to subscribers
```

**Dependencies Added**

- `github.com/go-chi/chi/v5` v5.2.4 — HTTP router
- `github.com/gorilla/websocket` v1.5.3 — WebSocket upgrade
- `github.com/golang-jwt/jwt/v5` v5.3.0 — JWT token handling

**API Endpoints**

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| POST | `/api/v1/auth/login` | JWT login (dev: admin/admin) |
| POST | `/api/v1/auth/ws-ticket` | Single-use WebSocket ticket |
| GET | `/api/v1/devices` | List devices (filterable) |
| GET | `/api/v1/devices/{id}` | Get device by ID |
| POST | `/api/v1/devices` | Create device |
| PATCH | `/api/v1/devices/{id}` | Update device |
| DELETE | `/api/v1/devices/{id}` | Delete device |
| GET | `/api/v1/devices/{id}/state` | Get current state |
| PUT | `/api/v1/devices/{id}/state` | Send command (async) |
| GET | `/api/v1/ws` | WebSocket connection |

**Design Decisions**

- **MQTT optional**: Server degrades gracefully without MQTT — reads and WebSocket work, only commands fail
- **Ticket-based WebSocket auth**: Prevents JWT leakage in URL query params/logs
- **Deep-copy semantics**: Registry returns copies to prevent cache pollution (consistent with M1.3)
- **Package location**: `internal/api/` (application-level concern, not infrastructure)

**Tests Added**

- `internal/api/server_test.go` — 23 tests:
  - Health endpoint (status, content-type)
  - Middleware (request ID, CORS, 404 handling)
  - Device CRUD (create, get, update, delete, filter by domain)
  - Device state (get state, missing command, not found)
  - Auth (login success/failure, ticket single-use, ticket expiry)
  - WebSocket hub (broadcast to subscribed, no message for unsubscribed, client count)

**Test Coverage**

- All 12 packages pass (0 failures)
- Build and lint clean
- 23 new API tests + all existing tests continue to pass

**Next: M1.5 — Flutter Wall Panel (or Auth hardening)**

---

## 1.0.5 – M1.3 Device Registry Complete (2026-01-22)

**Milestone: M1.3 Device Registry — Complete**

The device registry is fully implemented, tested, and integrated into the Gray Logic Core. It provides the central catalogue of all building devices with thread-safe in-memory caching and SQLite persistence.

**What Was Built**

- `internal/device/` package — 9 files, ~1,200 lines:
  - 50+ device types across 12+ domains (lighting, climate, blinds, audio, etc.)
  - 14 protocol types (KNX, DALI, Modbus RTU/TCP, MQTT, etc.)
  - 45+ capabilities (on_off, dim, temperature_read, etc.)
  - Thread-safe registry with RWMutex-protected cache and deep-copy semantics
  - SQLite persistence via Repository interface pattern
  - Protocol-specific address validation (KNX group addresses, DALI short addresses, Modbus host+unit)
  - Automatic slug generation from device names

**Integration**

- Device registry wired into `main.go`: initialised at startup, cache refreshed from database
- KNX bridge integration via adapter pattern: state and health updates flow from bus telegrams to registry
- `deviceRegistryAdapter` bridges typed `device.HealthStatus` to KNX bridge's `string` interface

**Tests Added**

- `internal/device/integration_test.go` — 4 integration tests:
  - Full device lifecycle (create → state → health → persist → restart → update → delete)
  - Multi-device queries (by room, domain, protocol, slug)
  - Cache consistency after simulated restart
  - Rapid state updates (dimmer ramp simulation)
- `internal/knxd/manager_test.go` — 14 test functions:
  - NewManager with defaults, custom config, and invalid configs
  - ConnectionURL for TCP and Unix socket modes
  - BuildArgs for USB, IPT, and IP routing backends
  - Group/individual address parsing and formatting round-trips
  - HealthError type and recoverability
  - Config validation (addresses, ports, USB IDs)
- `internal/process/manager_test.go` — 13 test functions:
  - Construction with defaults and custom config
  - Initial state, start/stop lifecycle, invalid binary
  - Exponential backoff delay calculation
  - RecoverableError interface
  - OnStart callback

**Test Coverage**

- All 11 packages pass (0 failures)
- `knxd` and `process` packages previously had `[no test files]` — now fully covered
- Build and lint clean

**Next: M1.4 — REST API + WebSocket**

---

## 1.0.4 – KNX Bridge Integration with Device Registry (2026-01-22)

**Enhancement: Device Registry Wiring**

Connected the device registry to the KNX bridge so that incoming bus telegrams automatically update device state and health in the Core's central device catalogue.

**Files Modified**

- `cmd/graylogic/main.go` — Added device registry initialisation and `deviceRegistryAdapter`
- `internal/bridges/knx/bridge.go` — Added optional `DeviceRegistry` interface calls in `handleKNXTelegram()`

**Design Pattern**

The adapter pattern bridges the type difference between the device package (typed `HealthStatus`) and the KNX bridge (plain `string`), keeping both packages decoupled.

---

## 1.0.3 – knxd Subprocess Management & Multi-Layer Health Checks (2026-01-21)

**Feature: Managed knxd Lifecycle with Watchdog**

Added the ability for Gray Logic Core to spawn and manage the knxd daemon as a subprocess, with comprehensive multi-layer health monitoring and automatic recovery.

**Rationale**

- **Container-friendly deployment**: knxd runs inside the same container as Gray Logic Core, simplifying deployment
- **No sudo required on site**: Engineers configure everything via YAML, not system files
- **Self-healing**: Automatic restart on failure with configurable backoff
- **Observability**: Health checks, uptime monitoring, and restart tracking
- **Multi-decade deployments**: Critical infrastructure should be monitored and automatically recovered

**Multi-Layer Health Check System**

| Layer | Check | Purpose |
|-------|-------|---------|
| 0 | USB device presence | Verify physical interface exists (USB backends only) |
| 1 | /proc/PID/stat | Verify process exists and is in runnable state |
| 2 | TCP connection | Verify knxd is accepting connections on port 6720 |
| 3 | EIB protocol handshake | Verify knxd speaks EIB protocol correctly |
| 4 | Bus-level device read | End-to-end verification (knxd → interface → bus → device) |

Layer 4 is optional but recommended — configure a KNX device (e.g., PSU at 1/7/0) that always responds to READ requests.

**Watchdog Behaviour**

- Configurable health check interval (default: 30s)
- After 3 consecutive failures: knxd is killed and restarted
- Configurable max restart attempts (default: 10, 0 = unlimited)
- For USB backends: optional USB reset before restart (recovers from LIBUSB_ERROR_BUSY)

**New Packages**

- `internal/process/` — Generic subprocess lifecycle management (~590 lines, reusable for DALI, Modbus, etc.)
  - Graceful shutdown (SIGTERM → timeout → SIGKILL)
  - Automatic restart with configurable delay
  - stdout/stderr capture for logging
  - Health monitoring callbacks with consecutive failure tracking

- `internal/knxd/` — knxd-specific wrapper (~720 lines)
  - KNX address validation (physical addresses, client address pools)
  - Command-line argument building from YAML config
  - TCP readiness polling before bridge startup
  - Multi-layer health checks (Layers 0-4)
  - Bus-level health check via KNX READ request

**Configuration**

New `knxd:` section under `protocols.knx` in config.yaml:

```yaml
protocols:
  knx:
    enabled: true
    knxd:
      managed: true                    # If true, Gray Logic spawns knxd
      binary: "/usr/bin/knxd"
      physical_address: "0.0.1"
      client_addresses: "0.0.2:8"
      backend:
        type: "usb"                    # usb, ipt, or ip
        # host: "192.168.1.100"        # For ipt mode
        # multicast_address: "224.0.23.12"  # For ip mode
      restart_on_failure: true
      restart_delay_seconds: 5
      max_restart_attempts: 10

      # Watchdog health checks
      health_check_interval: 30s       # How often to run checks

      # Bus-level health check (Layer 4) - optional but recommended
      # Specify a KNX device that always responds to READ requests
      health_check_device_address: "1/7/0"   # e.g., PSU status
      health_check_device_timeout: 3s
```

**Files Created**

- `internal/process/doc.go` — Package documentation
- `internal/process/manager.go` — Generic subprocess manager (~490 lines)
- `internal/knxd/doc.go` — Package documentation
- `internal/knxd/config.go` — Configuration and arg building (~320 lines)
- `internal/knxd/manager.go` — knxd-specific manager (~270 lines)

**Files Modified**

- `internal/infrastructure/config/config.go` — Added `KNXDConfig` and `KNXDBackendConfig` types
- `configs/config.yaml` — Added `knxd:` section with full documentation
- `cmd/graylogic/main.go` — Added `startKNXD()` and updated startup sequence

**Startup Sequence**

```
1. Load configuration
2. Connect to database, MQTT
3. If knx.enabled && knxd.managed:
   a. Spawn knxd subprocess with args from config
   b. Wait for TCP port 6720 to accept connections (up to 30s)
   c. Log connection URL
4. Start KNX bridge using managed connection URL
5. On shutdown: Stop bridge first, then stop knxd
```

**Backwards Compatibility**

When `knxd.managed: false`, the system behaves exactly as before — expects knxd to be running externally (e.g., as a systemd service) and connects to the configured `knxd_host:knxd_port`.

---

## 1.0.2 – AI Assistant Context (2026-01-17)

**Tooling & Guidance**

Added specialized guidance for AI assistants to ensure alignment with project principles and coding standards.

**Documents**

- `CLAUDE.md`: Primary project guidance for Claude Code (claude.ai/code) — architecture overview, repository structure, common commands, development approach, and AI assistant guidelines. This is the main AI context file used throughout development.
- `GEMINI.md`: Project-specific guidance for the Gemini CLI agent, including architecture overview, philosophy, coding standards, and interaction rules.
- `code/core/AGENTS.md`: Go-specific development guidance for AI assistants working on the Core codebase — package conventions, testing standards, error handling patterns.

---

## 1.0.1 – Plant Room & Commercial Expansion (2026-01-12)

**Documentation Expansion**

Expanded Gray Logic to properly support plant room environments and commercial/office deployments.

**Updated Documents**

- `docs/data-model/entities.md`:
  - Added Area types: `wing`, `zone` for commercial deployments
  - Added hierarchy guidance table (Residential vs Office vs Multi-tenant)
  - Added Room types: `open_plan`, `meeting_room`, `boardroom`, `reception`, `break_room`, `hot_desk`, `server_room`, `storage`, `loading_bay`, `washroom`, `corridor`, `workshop`, `changing_room`, `wine_cellar`, `spa`
  - Expanded Plant Equipment device types: `chiller`, `cooling_tower`, `ahu`, `fcu`, `vav_box`, `vfd`, `fan`, `compressor`, `humidifier`, `dehumidifier`, `water_heater`, `water_softener`, `generator`, `ups`
  - Added Plant Sensors & Actuators: `flow_meter`, `pressure_sensor`, `differential_pressure`, `valve_2way`, `valve_3way`, `damper`, `vibration_sensor`, `bearing_temp`
  - Added Emergency & Safety (monitoring only): `emergency_light`, `exit_sign`, `fire_input`, `gas_detector`
  - Added Access Control: `card_reader`, `door_controller`, `turnstile`, `intercom`
  - Added Protocols: `bacnet_ip`, `bacnet_mstp`, `onvif`, `ocpp`
  - Added Domain: `safety`
  - Expanded Capabilities with categories: Basic Control, Climate & Environment, Presence & Occupancy, Audio/Video, Security & Access, Plant & Equipment, Energy Monitoring, Condition Monitoring (PHM), Emergency Lighting, Booking & Scheduling

- `docs/domains/climate.md`:
  - Added Commercial HVAC section
  - VAV zone control with state model
  - Fan coil unit zones
  - Occupancy scheduling with pre-conditioning and optimum start
  - Meeting room booking integration
  - Out-of-hours override requests
  - Night setback and purge sequences
  - Commercial commissioning checklist

- `docs/domains/lighting.md`:
  - Added Commercial Lighting section
  - Emergency lighting monitoring (DALI Part 202) with state model
  - Function and duration test scheduling
  - Compliance reporting for emergency lighting
  - Daylight harvesting for commercial spaces
  - Occupancy-based lighting (commercial patterns)
  - Corridor and circulation lighting
  - Meeting room lighting with AV integration
  - Personal control and task lighting
  - Commercial commissioning checklist

**New Documents**

- `docs/protocols/bacnet.md` — Year 2 roadmap placeholder for BACnet/IP and MS/TP integration with commercial HVAC systems (AHUs, chillers, VAVs)

- `docs/domains/plant.md` — Comprehensive plant domain specification covering:
  - Pumps, VFDs, AHUs, chillers, boilers, heat pumps
  - State models for all equipment types
  - Commands and sequences of operation
  - Lead/lag, economizer, staging sequences
  - Alarm management (priorities, state machine, shelving)
  - Predictive Health Monitoring (PHM) for rotating equipment
  - Energy optimization strategies
  - Modbus and BACnet protocol mapping

- `docs/integration/fire-alarm.md` — Fire alarm system integration (monitoring only):
  - Critical safety rules (observe, never control)
  - Integration architecture via auxiliary contacts
  - Signal types and wiring
  - Automation responses (lights, blinds, notifications)
  - UI display and wall panel behavior
  - Audit logging and compliance reporting
  - Testing and commissioning procedures

- `docs/integration/access-control.md` — Access control integration:
  - Integration levels (monitoring, triggering, control)
  - Door controllers, card readers, intercoms, gates
  - State models and access events
  - **Residential Access Control** (new):
    - Video intercom integration (2N, Doorbird, Akuvox)
    - Keypad entry with scheduled cleaner/tradesperson access
    - Smart lock integration (auto-lock, battery monitoring)
    - Garage door control with safety sensors and auto-close
    - Driveway gate automation with ANPR option
    - Guest/temporary access code management
    - Holiday mode access restrictions
    - Pool gate safety compliance (monitoring only)
  - Welcome home and visitor automation
  - Departure/arrival presence detection
  - Emergency egress coordination
  - Remote unlock security requirements
  - SIP intercom integration
  - **Commercial Access Control**:
    - Turnstiles and speed gates
    - Visitor management integration

- `docs/integration/cctv.md` — CCTV and video surveillance integration:
  - **Residential CCTV**:
    - Typical camera placement (front door, driveway, garden, garage)
    - Doorbell/intercom camera integration with wall panels
    - Motion detection automation triggers
    - Package delivery detection
    - Vehicle approaching notifications
    - Privacy controls per camera
  - **Commercial CCTV**:
    - Camera groups and multi-site layouts
    - Analytics integration (line crossing, people counting, ANPR)
    - Access control linking (record on denied access)
    - Video wall configuration
  - Stream management (main/sub stream selection)
  - MQTT event topics and payloads
  - Privacy zones and audit logging
  - Supported hardware (Uniview, Hikvision, Dahua, Axis)

- `docs/deployment/office-commercial.md` — Commercial deployment guide:
  - Commercial vs residential comparison
  - Network architecture and IT integration
  - Occupancy schedules and holiday calendars
  - Open plan and meeting room lighting
  - VAV system configuration
  - Meeting room booking (M365, Google Workspace)
  - Tenant energy billing
  - Active Directory integration
  - Role-based permissions
  - Commissioning checklist
  - Ongoing operations procedures

---

## 1.0.0 – Architecture Pivot to Custom Gray Logic Core (2026-01-12)

**Major Architectural Change**

This release marks a significant pivot from the openHAB-based approach (v0.4 and earlier) to a **custom-built Gray Logic Core** in Go. This decision was made after careful analysis of the project goals:

- **Multi-decade deployment stability** — Control over the entire stack, no dependency on third-party project decisions
- **True offline operation** — Leaner runtime, faster startup, lower resource usage
- **Custom UI** — Wall panels and mobile apps built to our specifications, not constrained by openHAB UI
- **Native AI integration** — Local voice control without Alexa/Google dependency
- **Full feature parity with Crestron/Savant/Loxone** — Scene complexity, multi-room audio, video distribution

**What Changed**

- **Archived** all v0.4 documentation to `docs/archive/v0.4-openhab-era.zip`
- **Archived** Docker Compose stack to `code/archive/v0.4-openhab-era.zip`
- **Created** new modular documentation structure:
  - `docs/overview/` — Vision, principles, glossary
  - `docs/architecture/` — System design
  - `docs/data-model/` — Entity definitions and schemas
  - `docs/domains/` — Per-domain specifications (to be created)
  - `docs/automation/` — Scenes, schedules, modes (to be created)
  - `docs/intelligence/` — Voice, PHM, AI (to be created)
  - `docs/resilience/` — Offline operation, satellite weather (to be created)
  - `docs/protocols/` — Protocol bridges (to be created)
  - `docs/interfaces/` — API and UI specs (to be created)
  - `docs/deployment/` — Installation and commissioning (to be created)
  - `docs/business/` — Business case and pricing (to be created)

**New Foundation Documents**

- `docs/overview/vision.md` — What Gray Logic is and why we're building it
- `docs/overview/principles.md` — Hard rules that can never be broken
- `docs/overview/glossary.md` — Standard terminology
- `docs/architecture/system-overview.md` — Complete system architecture
- `docs/data-model/entities.md` — Core data model (Site, Area, Room, Device, Scene, etc.)

**Technology Decisions**

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Core | Go | Single binary, no runtime, cross-compiles, multi-decade stability |
| Database | SQLite | Embedded, zero maintenance |
| Time-Series | InfluxDB | PHM data, energy monitoring |
| Message Bus | MQTT | Simple, proven, debuggable |
| Wall Panel/Mobile | Flutter | Cross-platform native |
| Voice STT | Whisper | Local, accurate, open |
| Voice TTS | Piper | Local, natural |
| Local AI | Llama/Phi | On-device intelligence |

**Principles Carried Forward**

All core principles from v0.4 remain valid:
- Offline-first (99%+ without internet)
- Physical controls always work
- Life safety independent
- Open standards at field layer (KNX, DALI, Modbus)
- No vendor lock-in
- Customer owns their system

**What's Next**

This is a 5-year part-time project:
- Year 1: Core + KNX + lighting in own home
- Year 2: Full scenes, modes, blinds, climate
- Year 3: Audio, video, security, CCTV
- Year 4: Voice control, PHM, AI
- Year 5: Commissioning tools, first customer

---

## 0.4 – Predictive Health, Doomsday Pack & Golden Handcuffs (2025-12-02)

> **Note**: This version used openHAB + Node-RED. Documentation archived to `docs/archive/v0.4-openhab-era.zip`

**Docs**

- Updated `docs/gray-logic-stack.md` to **Working Draft v0.4**:

  - Refined the **Design Principles** and module structure to explicitly support:
    - **Predictive Health Monitoring (PHM)** for suitable plant (pumps, boilers/heat pumps, AHUs, pool kit).
    - A clearer split between:
      - Short/medium-term **local logging** on-site.
      - Long-term **trend retention** as a remote premium bonus.
  - Added a dedicated **Plant & PHM** section (within modules and principles) covering:
    - The idea of a plant “**heartbeat**” (current, temp, run hours, vibration, etc.).
    - Using rolling averages and deviation thresholds as **early warning**, not magic:
      - e.g. “If pump current or temp deviates from its 7-day average by ≥20% for >2h, raise a ‘maintenance warning’, not a full fault.”
    - Asset categories where PHM makes sense (pumps, AHUs, boilers/heat pumps, etc.).
    - Examples of how PHM logic is split between:
      - openHAB (device states, items, basic rules).
      - Node-RED (cross-system logic and PHM flows).
  - Tightened the **Security & CCTV** section to:
    - Reiterate that cloud-only CCTV/doorbells (Ring/Nest-style) are **out of scope** for core logic.
    - Explicitly list examples of **integrator-grade** doorbells (Amcrest, DoorBird, Uniview) as _illustrative_ of the required capabilities (local RTSP/ONVIF, predictable behaviour).
  - Reinforced the **Consumer Overlay** rules:
    - Overlay devices remain non-critical and best-effort.
    - Overlay logic cannot become a dependency for plant, core lighting safety, or security.
  - Clarified that the stack is designed to avoid **“Golden Handcuffs”**:
    - Open standards at the field layer.
    - Documented configs and runbooks.
    - A clear handover path if someone else needs to take over.

- Updated `docs/business-case.md` to **Business Case v0.4**:

  - Extended the **Problem Statement** to include:
    - Demand for **BMS-like early warning** on small sites without full BMS cost.
    - The “Golden Handcuffs” problem of proprietary platforms (high cost to leave an ecosystem).
  - Strengthened the **Solution Overview**:
    - Added Predictive Health Monitoring (PHM) as a headline capability:
      - The system “learns” plant behaviour and flags deviations as early warnings.
    - Made the **internal vs external** split more explicit around data:
      - On-site: short/medium history and PHM rules that still work offline.
      - Remote: long-term retention, pretty dashboards, multi-year comparisons.
  - Reframed the **Value Proposition**:
    - For clients: emphasised **early warning** and evidence-based maintenance,
      not promises of zero failures.
    - For the business: PHM is a signed, billable differentiator that justifies Enhanced/Premium tiers.
  - Reworked **Support Tiers** to map directly to PHM capability:

    - **Core Support**:

      - Safe, documented system with basic binary monitoring.
      - Offline-first, no VPS required.

    - **Enhanced Support**:

      - Adds PHM alerts using trend deviation logic.
      - Short/medium-term trend views per site.
      - Clear “we spot issues earlier” story.

    - **Premium Support**:
      - Adds deeper VFD/Modbus data, multi-year history and reporting.
      - Multi-site or estate-level overviews where relevant.

  - Added a new **Risk & Mitigation** subsection for PHM:

    - Risk of over-promising.
    - Mitigation: honest “early warning, not magic” messaging and tunable rules.

  - Added a new **Trust / One-man band risk** section:

    - Introduced the **Doomsday / Handover Package** concept:
      - “God mode” exports (root access, KNX project, configs, Docker compose).
      - Printed **Safety Breaker** instructions to gracefully shut down the stack and revert to physical controls.
      - A “Yellow Pages” list of alternative integrators.
      - A “Dead Man’s Switch” clause allowing clients to open the sealed pack if Gray Logic is unresponsive for an agreed period.
    - Framed this as a positive trust signal and a counter to proprietary **Golden Handcuffs**.

  - Updated **Success Criteria**:
    - Success now explicitly includes:
      - At least one **PHM pattern** that has proven useful in the real world (e.g. pool pump early warning).
      - Clients seeing real value from offline-first behaviour **and** early warning on plant.

---

## 0.3 – Offline-first model, internal vs external (2025-12-02)

**Docs**

- Updated `docs/gray-logic-stack.md` to **Working Draft v0.3**:

  - Clarified the distinction between **Gray Logic** (company/brand) and the **Gray Logic Stack** (product/architecture).
  - Added an explicit **internal (on-site) vs external (remote)** model:
    - Internal = the on-site Linux/Docker node that must keep running if the internet/VPN is down.
    - External = optional remote services on a VPS, reachable via WireGuard, treated as non-critical bonuses.
  - Stated a clear **offline-first target**:
    - At least **99% of everyday functionality** (lighting, scenes, schedules, modes, local dashboards, plant logic) must work with **no internet connection**.
  - Defined a **Consumer Overlay**:
    - Segregated, non-critical module for consumer-grade IoT (Hue, random smart plugs, etc.).
    - Overlay logic is not allowed to become a dependency for core lighting safety, life safety, or plant operation.
  - Expanded **disaster recovery and rebuild**:
    - Config in Docker volumes/bind mounts.
    - Version-controlled `docker-compose.yml`.
    - Simple host rebuild process described in the spec.

- Updated `docs/business-case.md` to reflect the same model:
  - Internal vs external split baked into the solution overview and value proposition.
  - Support tiers aligned with on-site vs remote:
    - Core = on-site focus and basic support.
    - Enhanced = adds remote monitoring and alerts.
    - Premium = adds remote changes/updates and richer reporting.
  - Success criteria updated to include “clients actually experience the offline reliability benefit”.

---

## 0.2 – Spec + business case foundation

- `docs/gray-logic-stack.md`:

  - First proper written spec for the **Gray Logic Stack**:
    - Goals, principles, hard rules, and module breakdown.
    - Target domains: high-end homes, pools/leisure, small mixed-use/light commercial.
    - “Mini-BMS” positioning vs proprietary smart home platforms and full BMS/SCADA.
  - Described key functional modules:
    - Core (Traefik, dashboard, metrics).
    - Environment monitoring.
    - Lighting & scenes.
    - Media / cinema.
    - Security, alarms, and CCTV.
  - Added a roadmap from v0.1 → v1.0.

- `docs/business-case.md`:
  - Defined the **technical and business problem** as I see it.
  - Identified target customers and stakeholders.
  - Outlined a **project + recurring revenue** model.
  - Wrote down early success criteria and explicit “permission to walk away” conditions.

---

## 0.1 – Initial repo structure

- Created the initial repo layout:
  - `docs/` for specs and business case.
  - `code/` for future Docker Compose and config.
  - `notes/` for brainstorming and meeting notes.
- Dropped in early drafts of the Gray Logic Stack concept and basic architecture ideas.
