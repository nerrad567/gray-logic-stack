# Changelog – Gray Logic

All notable changes to this project will be documented in this file.

---

## 1.0.28 – Capability-Aware Scene Editor & Flutter Fixes (2026-02-06)

### Fixed

- **Scene editor command selection** — Rewrote `_commandsForDevice()` from domain-based switch to capability-driven logic. Devices now only show commands matching their actual capabilities:
  - `heating_actuator` (on_off only): shows on/off/toggle instead of set_setpoint
  - `blind_switch` (binary move): shows on/off/stop instead of set_position
  - Sensors (`presence_sensor`, etc.): hidden from device dropdown entirely (read-only, not commandable)
- **Scene presets** — Capability-aware downgrading: `set_level` on switch-only lights becomes `on`, `set_position` on blind switches becomes `on`/`off` based on target position
- **`blind_switch` default capabilities** — Changed from `['position']` to `['on_off']` in device type config (only affects manual device creation; ETS imports use Go commissioning)
- **Flutter web startup error** — Added `WidgetsFlutterBinding.ensureInitialized()` before `BrowserContextMenu.disableContextMenu()` call in `main.dart` to prevent `OptionalMethodChannel` crash during engine init

### Added

- **Device capability getters** — `hasPosition`, `hasTilt`, `hasTemperatureSet`, `hasColorTemp`, `hasColorRGB`, `hasSpeed`, `isCommandable` on Device model
- **Tilt slider UI** — `set_tilt` command with tilt angle slider (0-100%) for blinds with slat control

---

## 1.0.27 – M2.1 Location Hierarchy, Device Groups & Infrastructure Zones (2026-02-05)

**Milestone: M2.1 Location Hierarchy — Complete (Phases 1-5)**

Added device tags, device groups (static/dynamic/hybrid), unified infrastructure zones, a single-call hierarchy endpoint, and referential safety on room deletion. Multi-agent delivery: Codex implemented Phases 1-3 (repository layer), Claude implemented Phases 4-5 (API handlers + wiring) — in parallel with zero file overlap.

### Added

- **Device Tags** (`internal/device/tags.go`):
  - `TagRepository` interface + SQLite implementation (7 methods)
  - Free-form string labels for filtering and exception-based operations
  - Tags normalised to lowercase + trimmed, bulk-loaded into registry cache on `RefreshCache`
  - `Registry.GetDevicesByTag()` for cache-based tag filtering
  - API: `GET /tags`, `GET /devices/{id}/tags`, `PUT /devices/{id}/tags`, `GET /devices?tag=X`

- **Device Groups** (`internal/device/group_*.go`):
  - `DeviceGroup`, `GroupType` (static/dynamic/hybrid), `FilterRules`, `GroupMember` types
  - `GroupRepository` interface + SQLite implementation (Create, GetByID, List, Update, Delete, SetMembers, GetMembers, GetMemberDeviceIDs)
  - `ResolveGroup()` — runtime resolution expanding groups to concrete device lists with scope, domain, capability, tag, and exclude_tag filters
  - API: Full CRUD (`GET/POST/PATCH/DELETE /device-groups`), `PUT /device-groups/{id}/members`, `GET /device-groups/{id}/members`, `GET /device-groups/{id}/resolve`

- **Unified Infrastructure Zones** (`internal/location/zone_*.go`):
  - Single `infrastructure_zones` table with domain discriminator covering climate, audio, lighting, power, security, video
  - Domain-specific config in JSON `settings` column — no schema migration needed per domain
  - `ZoneRepository` interface + SQLite implementation with one-zone-per-domain enforcement
  - Room membership via `infrastructure_zone_rooms` junction table (replaces deprecated `rooms.climate_zone_id`/`audio_zone_id` columns)
  - API: Full CRUD (`GET/POST/PATCH/DELETE /zones`), `GET /zones?domain=X`, `PUT /zones/{id}/rooms`, `GET /zones/{id}/rooms`

- **Hierarchy Endpoint** (`internal/api/hierarchy.go`):
  - `GET /hierarchy` — single-call site → areas → rooms tree
  - Includes per-room device count, scene count, and zone membership
  - Room-scoped users see only their granted rooms; empty areas omitted

- **Referential Safety** (`internal/api/locations.go`):
  - `DELETE /rooms/{id}` now returns 409 Conflict if devices or scenes reference the room

- **Database Migration** (`20260206_100000_tags_groups_zones`):
  - 5 new tables: `device_tags`, `device_groups`, `device_group_members`, `infrastructure_zones`, `infrastructure_zone_rooms`
  - Indexes for tag lookup, group member reverse lookup, zone domain filtering

### Changed

- **`api/router.go`** — 22 new routes across `device:read`, `device:configure`, and `location:manage` permission groups
- **`api/server.go`** — Added `TagRepo`, `GroupRepo`, `ZoneRepo` to Deps/Server
- **`api/devices.go`** — Added `?tag=` query filter to device listing
- **`cmd/graylogic/main.go`** — Creates tag, group, zone repositories; wires tag repo into registry before `RefreshCache`
- **`device/registry.go`** — Added `SetTagRepository()`, bulk tag loading in `RefreshCache`, `GetDevicesByTag()`
- **`device/types.go`** — Added `Tags []string` field with `DeepCopy` support

---

## 1.0.26 – M1.7 Auth Hardening — Multi-Level Authentication & Authorisation (2026-02-05)

**Milestone: M1.7 Auth Hardening — Complete**

Replaced the placeholder auth system (hardcoded `admin/admin`, pass-through middleware) with a production-grade multi-level authentication and authorisation system. 4 role tiers, Argon2id password hashing, JWT access/refresh token rotation with theft detection, panel device identity, per-user room scoping, and comprehensive audit logging.

### Added

- **`internal/auth/` package** (19 files, 3,009 lines):
  - **types.go** — `User`, `RefreshToken`, `Panel`, `RoomAccess`, `RoomScope` structs; `Role` constants (panel/user/admin/owner); 7 error sentinels
  - **password.go** — Argon2id hash/verify with PHC string format (m=64MiB, t=3, p=1, OWASP 2025)
  - **claims.go** — Typed `CustomClaims` replacing `MapClaims`; `GenerateAccessToken`, `GenerateRefreshToken`, `ParseToken` with role, session ID, JTI
  - **permissions.go** — 10 permission constants, static `rolePermissions` map, `HasPermission()`, `IsRoomScoped()` for 4-tier RBAC
  - **user_repository.go** — `UserRepository` interface + SQLite: Create, GetByID, GetByUsername, List, Update, Delete, Count, UpdatePassword
  - **token_repository.go** — `TokenRepository` interface + SQLite: Create, GetByID, Revoke, RevokeFamily (theft detection), RevokeAllForUser, ListActiveByUser, DeleteExpired
  - **panel_repository.go** — `PanelRepository` interface + SQLite: Create, GetByID, GetByTokenHash, List, Delete, UpdateLastSeen, SetRooms, GetRoomIDs
  - **room_access.go** — `RoomAccessRepository` interface + SQLite: SetRoomAccess, GetRoomAccess, GetAccessibleRoomIDs, GetSceneManageRoomIDs, ClearRoomAccess
  - **seed.go** — `SeedOwner()` creates default owner with crypto-random password on first boot (printed to console)
  - **doc.go** — Package documentation
  - **test_helpers_test.go** — Shared test setup (in-memory SQLite with migrations)
  - 8 test files with comprehensive coverage (password round-trip, JWT generate+parse, RBAC permissions, CRUD operations, family revocation, room scoping)

- **Auth API endpoints** (`api/auth.go` — rewritten):
  - `POST /auth/login` — DB lookup + Argon2id verify → access token + refresh token (replaces hardcoded `admin/admin`)
  - `POST /auth/refresh` — Token rotation with family tracking + theft detection (reuse → revoke entire family)
  - `POST /auth/logout` — Revoke refresh token family
  - `POST /auth/change-password` — Verify current, hash new, revoke all sessions

- **User management API** (`api/users.go` — new):
  - `GET /POST /users` — List/create users (`user:manage` permission)
  - `GET /PATCH /DELETE /users/{id}` — Read/update/delete with self-protection guards
  - `GET /DELETE /users/{id}/sessions` — List/revoke active refresh tokens
  - `GET /PUT /users/{id}/rooms` — Per-user room access management (explicit-grant model)
  - Owner-only guard: only owners can create/modify owner-role users

- **Panel management API** (`api/panels.go` — new):
  - `GET /POST /panels` — List/register panels with room assignments (`system:admin` permission)
  - `GET /PATCH /DELETE /panels/{id}` — Read/update/revoke panel tokens
  - `GET /PUT /panels/{id}/rooms` — Panel room assignment management
  - Panel token shown once on creation (256-bit, SHA-256 hashed for storage)

- **Real auth middleware** (`api/middleware.go` — rewritten):
  - `authMiddleware` — Validates Bearer JWT OR `X-Panel-Token` header, injects claims/panel context
  - `requirePermission(perm)` — Checks role has permission via static map
  - `requireRoomScope()` — Resolves `user_room_access` for `user` role; admin/owner bypass
  - Context helpers: `claimsFromContext()`, `panelFromContext()`, `roomScopeFromContext()`
  - Permission denied events logged to audit trail

- **Audit logging** (`api/audit.go` — new):
  - `auditLog()` helper — async, best-effort via goroutine with `context.Background()`
  - Events: login success/failure, token reuse (theft detection), password change, user CRUD, panel CRUD, room access changes, permission denied (403)
  - `audit.Repository.Create()` method added to existing audit package

- **Database migration** (`20260205_150000_auth_users`):
  - `users` table with Argon2id password hash, role, active status
  - `refresh_tokens` table with family-based theft detection
  - `panels` table with hashed bearer tokens
  - `panel_room_access` table (multi-room panel assignments)
  - `user_room_access` table (explicit-grant room scoping with `can_manage_scenes` flag)

- **Background token cleanup**: Hourly goroutine purges expired refresh tokens from SQLite

- **`writeForbidden`** helper in `api/errors.go` for 403 responses

- **`golang.org/x/crypto`** dependency for Argon2id

### Changed

- **`api/router.go`** — All routes restructured by permission level (public → authenticated → permission-gated)
- **`api/server.go`** — Added `UserRepo`, `TokenRepo`, `PanelRepo`, `RoomAccessRepo`, `AuditRepo` to Deps/Server
- **`api/websocket.go`** — WebSocket ticket now carries `CustomClaims` through to WSClient
- **`cmd/graylogic/main.go`** — Creates auth repos, calls `SeedOwner()`, starts token cleanup loop, passes repos to API
- **`config/config.go`** — Added `RefreshTokenTTLPanel`, `PanelElevationTimeout` fields

### Removed

- **Hardcoded `admin/admin` credentials** — Login now validates against `users` table with Argon2id
- **Pass-through `authMiddleware`** — Was a no-op; now enforces real JWT/panel-token validation

### Security Model

```
4-Tier Role Hierarchy:
  panel  → Device identity (no login, X-Panel-Token, room-scoped)
  user   → Household member (JWT, explicit room grants, optional scene management)
  admin  → Full system control (JWT, bypasses room scoping)
  owner  → Emergency-only (JWT, factory reset, manage other owners)

Room Scoping (user role only):
  Zero rooms assigned = no access (locked out)
  Admin must explicitly grant rooms per user
  Per-room can_manage_scenes flag for scene creation/editing

Token Security:
  Access tokens: 15min TTL, no DB hit (signature + expiry only)
  Refresh tokens: 7-day TTL, SQLite-backed, family rotation
  Theft detection: reused refresh token → entire family revoked
  Panel tokens: permanent until revoked, SHA-256 hashed storage
```

### Technical Notes

- Auth package: 19 files, 3,009 lines (including tests)
- New API handlers: users.go (220 lines), panels.go (200 lines), audit.go (30 lines)
- Migration: 78 lines (up) + 5 lines (down)
- All 16 test packages pass with `-race`
- golangci-lint: 0 warnings
- First-boot owner password printed to console and logged — must be changed immediately
- Async audit logging ensures auth events never block request processing

---

## 1.0.25 – Zero-Warning Lint Baseline (2026-02-05)

**Focus: Comprehensive golangci-lint cleanup — 174 warnings reduced to 0**

### Fixed

- **wrapcheck**: Wrapped ~23 bare `return err` with `fmt.Errorf` context across location/repository, state_history, ga_recorder — error chains now carry debugging context
- **goconst**: Extracted domain, type, and location string literals to named constants in commissioning/etsimport
- **revive (naming)**: Renamed `SqlDB()` → `SQLDB()` for Go acronym conventions (interface + implementation + docs)
- **unused**: Deleted dead `checkBusHealth()` (159 lines, superseded by `checkBusHealthWithGroupAddresses`) and `parseKNXProj()` wrapper

### Changed

- **`.golangci.yml`**: Reverted blanket "color" ignore-word, replaced with per-line `//nolint:misspell` on 11 lines where American spelling is required by KNX DPT standard
- **Per-line `//nolint` directives**: Added 108 targeted suppressions with justification comments for confirmed false positives (shadow, mnd, gocognit/gocyclo, misspell, dupl, revive)

### Technical Notes

- Net line change: +292 / -376 (code removed > code added)
- All 15 test packages pass with `-race`
- Future lint warnings are genuine issues — zero-noise baseline established

---

## 1.0.24 – State History Audit Trail + Metrics Endpoints (2026-02-05)

**Focus: SQLite state audit trail and REST endpoints for device history and VictoriaMetrics metrics**

Codex-assisted implementation (gpt-5.2-codex, xhigh reasoning, 20 minutes autonomous). Adds a local state change audit trail to SQLite and REST endpoints for querying device history and time-series metrics.

### Added

- **State history audit trail**: `state_history` SQLite table records every device state change with device_id, JSON state snapshot, source (mqtt/command/scene), and timestamp. STRICT mode with FK cascade.
- **`StateHistoryRepository`**: Interface + SQLite implementation with `RecordStateChange`, `GetHistory` (newest-first, configurable limit), and `PruneHistory` (duration-based retention)
- **Background prune loop**: Daily cleanup of state history entries older than 30 days, context-cancellable on shutdown
- **`GET /devices/{id}/history`**: Returns state change history from SQLite audit trail with `limit` and `since` query params
- **`GET /devices/{id}/metrics`**: Proxies PromQL range queries to VictoriaMetrics for device telemetry (start/end/step params)
- **`GET /devices/{id}/metrics/summary`**: Returns available metric fields and latest values for a device via PromQL instant query
- **TSDB query methods**: `QueryRange` and `QueryInstant` on tsdb.Client — thin HTTP wrappers returning `json.RawMessage` (zero deserialise/re-serialise overhead)
- **PromQL injection prevention**: Label values escaped via `strconv.Quote` in query builder
- **11 new test functions**: Repository tests (record, history ordering, prune), handler tests (history, metrics proxy, summary parsing, TSDB disabled), TSDB query tests (range, instant, errors, timeout)

### Technical Notes

- Migration: `20260205_090000_state_history.up.sql` with indexes on (device_id, created_at DESC) and (created_at DESC)
- State history is independent of VictoriaMetrics — works even when TSDB is disabled
- Metrics endpoints return 503 (not 500) when TSDB is nil/disabled — graceful degradation
- All 15 test packages pass with `-race`

---

## 1.0.23 – VictoriaMetrics Migration + State Pipeline Wiring (2026-02-05)

**Focus: Replace InfluxDB with VictoriaMetrics and connect the state collection pipeline**

ADR-004: Replaced InfluxDB 2.7 with VictoriaMetrics as the time-series database. VictoriaMetrics provides free OSS clustering (InfluxDB 3 locks HA behind paid Enterprise), 10x lower RAM usage, and accepts InfluxDB line protocol — making it a drop-in wire-protocol replacement. The Go client was rewritten with zero external dependencies (net/http only).

Additionally, wired the previously-disconnected state pipeline: MQTT state messages from the KNX bridge now flow through WebSocket broadcast → Device Registry update → VictoriaMetrics telemetry writes.

### Added

- **State pipeline wiring**: MQTT device state updates now write through to Device Registry (SQLite) and VictoriaMetrics (if enabled). Previously, incoming bus telegrams (physical switch presses, sensor updates) only broadcast to WebSocket — the registry and TSDB were never updated.
- **TSDB client in API server**: `tsdb.Client` passed as optional dependency to API server for telemetry writes during state ingestion
- **Boolean-to-float conversion**: Boolean state fields (on/off) written to VictoriaMetrics as 0.0/1.0 for time-series analytics

### Changed

- **Docker Compose**: `influxdb:2.7-alpine` → `victoriametrics/victoria-metrics:v1.135.0` (dev + prod)
- **Config**: `InfluxDBConfig` → `TSDBConfig` — removed Token, Org, Bucket fields (not needed for VM single-node)
- **Package rename**: `internal/infrastructure/influxdb/` → `internal/infrastructure/tsdb/`
- **TSDB client rewrite**: Pure `net/http` client with internal batching (configurable batch size + flush interval), InfluxDB line protocol formatting, health checks via `GET /health`, writes via `POST /write`
- **go.mod**: Removed `influxdb-client-go/v2` and `line-protocol` dependencies (zero external TSDB deps)
- **Makefile**: Updated dev-services comment for victoriametrics
- **main.go**: TSDB connection moved before API server start (required for state pipeline wiring)

### Removed

- **InfluxDB 2.7 dependency**: Docker image, Go client library, and all associated configuration
- **influxdb-client-go/v2**: Eliminated MEDIUM-HIGH audit risk item from 2026-02-04 report

### Technical Notes

- VictoriaMetrics accepts InfluxDB line protocol on `POST /write` — write-side code pattern stays similar
- Queries use PromQL via `GET /api/v1/query_range` (better for time-series analytics than Flux/SQL)
- Deployment tiers: Tier 1 (residential, VM on NUC), Tier 2 (commercial, dedicated VM), Tier 3 (campus, free OSS VM cluster)
- TSDB is value-add for analytics/PHM — KNX field layer works independently, TSDB can be disabled without breaking control

---

## 1.0.22 – KNXSim Topology Refactor: One Premise = One TP Line (2026-02-04)

**Focus: Align KNXSim's topology model with real-world KNX hardware**

Refactored KNXSim so each premise represents exactly one physical KNX TP line with its own KNXnet/IP interface — matching how real hardware works. Removed the ability to create multiple areas/lines within a single premise. The individual address now correctly encodes physical location (Area.Line.Device).

### Added

- **`area_number` / `line_number` fields on Premise**: Stored on creation, `gateway_address` and `client_address` now derived automatically (e.g., area=2, line=1 → gateway `2.1.0`, client `2.1.255`)
- **Flat topology view**: Replaced Area→Line→Device tree with a simple device list + educational info banner explaining the one-line-per-premise model
- **IA remapping on sample load**: When loading sample devices onto a premise, IAs are remapped from `1.1.x` to match the target premise's `{area}.{line}.x`
- **Comprehensive reference guide updates**: "Individual Address" section now explains physical topology, what each part means (Area=building zone, Line=physical TP cable), what happens when addresses don't match physical location, and how KNXSim maps to real hardware (port vs IP distinction)
- **Create premise form**: Area and Line number inputs with live gateway address preview

### Changed

- **`config.yaml`**: `gateway:` section renamed to `topology:` with `area_number`/`line_number` fields (Codex fix)
- **`premise_manager.py`**: Bootstrap reads `topology.area_number`/`topology.line_number` instead of gateway/client addresses
- **`routes_export.py`**: ETS XML export now uses `premise.area_number`/`premise.line_number` instead of hardcoded "1"/"1" (Codex fix)
- **`routes_premises.py`**: `reset_to_sample()` remaps device IAs to target premise's line (Codex fix)
- **`routes_topology.py`**: Gutted from ~230 lines to ~40 lines — only `GET /topology` and `GET /next-device-number` remain
- **`db.py`**: Removed ~20 area/line CRUD methods, simplified `get_topology()` to return flat structure, `create_device()` no longer requires `line_id`
- **`api/models.py`**: `PremiseCreate` uses `area_number`/`line_number` instead of `gateway_address`/`client_address`; removed `line_id` from device models
- **Frontend (`api.js`, `store.js`, `index.html`)**: Removed all area/line CRUD methods (~180 lines), simplified topology state management
- **CSS**: Removed old `.topology-area`, `.topology-line` styles; added `.info-banner` for educational content
- **Tests**: Updated 6 test files to use new premise payload format and topology assertions; Codex switched to async httpx client to fix threadpool issues

### Removed

- **Area/Line CRUD endpoints**: `POST/GET/PATCH/DELETE /premises/{id}/areas`, `/areas/{id}/lines` — all removed
- **Area/Line UI**: "Add Area", "Add Line" buttons and modals removed from topology view
- **`line_id` on devices**: No longer stored or required — device number is sufficient within a premise
- **`gateway_address`/`client_address` in `PremiseCreate`**: Now derived from area/line numbers

### Technical Notes

- Database migration adds `area_number`/`line_number` columns to existing premises (defaults to 1/1)
- For fresh installations, simply nuke the Docker volume: `docker compose -f docker-compose.dev.yml down knxsim -v`
- 146 tests passing

---

## 1.0.21 – KNXSim Phase 2 Complete (2026-02-04)

**Focus: Close out all remaining KNXSim Phase 2 dashboard items**

KNXSim Phase 2 (Web Dashboard) is now fully complete. The VISION.md checklist was significantly out of date — a cross-reference audit revealed most Phase 2.1 and 2.5 items were already built. The remaining gaps (bus statistics, telegram filtering, building summary, conflict warnings) have been implemented.

### Added

- **Bus Statistics Panel** (Phase 2.4): Collapsible panel in telegram inspector showing total telegrams, TX/RX split, TPS, unique GAs, and top 5 busiest GAs. Backend tracks per-premise direction counts and GA frequency maps. Auto-refreshes every 5s while visible
- **Telegram Filtering** (Phase 2.4): Backend accepts `direction`, `device`, `ga` query params on history endpoint. Frontend adds debounced text search (300ms) filtering across device ID, GA, source address, and decoded value
- **Building Summary Stats** (Phase 2.1): Footer bar shows lights on, average temperature, presence rooms, and blinds open — computed reactively from live device state, no new endpoint needed
- **Address Conflict Warnings** (Phase 2.5): Proactive warnings in device create/edit modal for duplicate individual addresses and device IDs. Client-side validation against in-memory store for instant feedback

### Changed

- **VISION.md**: Full audit and correction of all Phase 2 checkboxes. Phase 2 marked complete. Low-priority items (sparklines, bulk ops, custom templates, template import/export, address range view) deferred to Phase 3
- **Stats bar**: Redesigned with building status indicators and visual separators between static counts and live metrics
- **TelegramInspector.get_stats()**: Now returns `total_recorded`, `rx_count`, `tx_count`, `unique_gas`, `top_gas` for per-premise queries
- **TelegramInspector.clear()**: Resets direction counters and GA frequency maps alongside ring buffer

---

## 1.0.20 – KNX Pipeline Robustness Refactor (2026-02-04)

**Focus: Eliminate fragile DPT inference and function name coupling across the KNX mapping pipeline**

The KNX import → registry → bridge pipeline was "a house of cards": DPTs were discarded at import time, forcing the bridge to reconstruct them via fragile substring matching. Function names were the single coupling point across all layers with no shared definition. This release stores structured DPT/flag metadata from import through to the bridge, creates a canonical function registry, and adds end-to-end pipeline integration tests.

### Added

- **Canonical function registry** (`functions.go`): Single authoritative list of ~55 KNX functions with name, state key, default DPT, flags, and aliases. Lookup maps built at init for O(1) resolution. Handles channel-prefixed functions (`ch_a_switch` → prefix `ch_a_`, base `switch`)
- **`NormalizeFunction()` / `NormalizeChannelFunction()`**: Resolve aliases at import time — `actual_temperature` → `temperature`, `on_off` → `switch`, `colour_temperature` → `color_temperature`
- **`StateKeyForFunction()`**: Unified state key lookup replacing the old hardcoded 30-line `functionToStateKey` map. Handles canonical names, aliases, and channel prefixes in a single call
- **Pipeline integration tests** (`pipeline_test.go`): 12 telegram→state test cases (lights, dimmers, blinds, thermostats, PIR, valves, infrastructure channels) + 4 command→telegram tests + device count + channel state key verification. All run in `go test` with no external dependencies
- **`KNXFunctionConfig` struct + `GetKNXFunctions()` helper** (`types.go`): Parse structured `map[string]any` from JSON into typed structs for the bridge
- **One-time data migration** (`registry.go`): `MigrateKNXAddressFormat()` converts existing flat-format devices to structured format on startup using inference as a one-time bootstrap
- **DPT and flags on Flutter `AddressFunction`** (`device_types.dart`): All 20+ device type templates now carry correct DPT and flags, sent through to the API

### Changed

- **Device address format**: From flat `{"group_address": "1/0/1", "switch": "1/0/1"}` to structured `{"functions": {"switch": {"ga": "1/0/1", "dpt": "1.001", "flags": ["write"]}}}`
- **`buildDeviceFromImport()`** (`commissioning.go`): Now preserves DPT/flags from ETS import and normalises function names via canonical registry
- **`loadDevicesFromRegistry()`** (`bridge.go`): Reads stored DPT and flags from `functions` map, falling back to inference only for pre-migration devices
- **`inferDPTFromFunction()` / `inferFlagsFromFunction()`** (`bridge.go`): Now use canonical registry as primary lookup, with simplified heuristic fallback
- **`buildStateUpdate()`** (`bridge.go`): Uses `StateKeyForFunction()` instead of hardcoded map
- **`validateKNXAddress()`** (`validation.go`): Validates `functions` map (at least one entry with non-empty `ga`) instead of checking for redundant `group_address` key
- **`deviceRegistryAdapter`** (`main.go`): Parses structured `functions` map into typed `FunctionMapping` structs
- **Flutter Add Device form** (`devices_tab.dart`): Sends structured `functions` format with DPT/flags per function
- **KNXSim `_guess_dpt()`** (`routes_export.py`): Extended from ~20 to ~45 function patterns, fixed `stop` DPT (1.010→1.007), added color temperature, RGB/RGBW, scenes, energy metering, wind/rain, boolean controls

### Removed

- **`group_address` top-level key**: Redundant — stored only one arbitrary GA. Replaced by `functions` map validation
- **`functionToStateKey` map** (`bridge.go`): 30-line hardcoded map replaced by canonical registry lookup

### Fixed

- **`stop` DPT mismatch** in KNXSim export: Was `1.010` (Start/Stop), now `1.007` (Step) matching KNX blind stop convention
- **Unreachable `brightness` → `9.004` rule** in `_guess_dpt()`: `"lux" or "brightness"` for lux was unreachable because `brightness` was already caught by the dimming check. Changed to `"lux" or "illuminance"`
- **`color_temperature` DPT fallback**: Was incorrectly matching `"temperature"` rule (→ `9.001`). Now matched before temperature check (→ `7.600`)

---

## 1.0.19 – ETS Import Room Assignment Fix (2026-02-03)

**Focus: Fix ETS import so devices are correctly assigned to GLCore rooms and areas**

Devices imported from ETS were ending up with no room or area assignments in the GLCore database. The Go backend was correct — it creates locations then auto-maps devices — but Flutter was silently dropping the `locations` array from the parse response and never sending it back on import. Without locations, no rooms were created, and all device `room_id`/`area_id` were cleared as invalid.

### Fixed

- **Flutter `ETSParseResult`**: Added `locations` field and `_parseLocations()` to capture location hierarchy from Go parse response (previously silently discarded)
- **Flutter `ETSImportRequest.toJson()`**: Now includes `locations` in the import payload (previously omitted — Go received `locations: 0`)
- **Flutter `ETSDetectedDevice.toImportJson()`**: Now includes `suggested_room` and `suggested_area` fields for auto-mapping fallback
- **Flutter `_copyParseResultWithDevices()`**: Extracted helper method that preserves `locations` when rebuilding parse result state (previously lost on any device toggle/edit)
- **Flutter panel deployment**: Fixed `cp -r build/web` creating nested `web/web/` directory — old build was being served instead of new one

### Changed

- **`commissioning.go`**: Added Info-level summary log on import request (device count, location count, options) for diagnostics
- **`config.yaml`**: Type alias additions from previous session now included

### Architecture

```
Flutter Parse Response → ETSParseResult.fromJson() → locations captured ✅
                                                        (was: silently dropped ❌)

Flutter Import Request → ETSImportRequest.toJson() → locations sent ✅
                                                        (was: never included ❌)

Go createLocationsFromETS() → areas + rooms created → autoMapDeviceLocations() → devices assigned ✅
                                (was: empty list, nothing created ❌)
```

**Commits**: `2fbe1f7`, `8321a07`

---

## 1.0.18 – KNX Device Classification — Full Pipeline Upgrade (2026-02-03)

**Focus: Two-tier device classification with ETS Function Types + manufacturer metadata through the entire pipeline**

KNXSim knew exactly what each device was (manufacturer, model, application program) but exported bare-bones .knxproj files with only group addresses. GLCore then guessed device types from DPT patterns. This release passes real metadata through the entire pipeline: YAML templates → KNXSim runtime → .knxproj export → GLCore parser → GLCore import → database.

### Added

- **Manufacturer metadata on all 47 YAML templates** (`sim/knxsim/templates/**/*.yaml`):
  - Realistic manufacturer data: ABB, MDT, Siemens, Gira, Elsner, Theben
  - Fields: `manufacturer.id` (M-number), `name`, `product_model`, `application_program`, `hardware_type`
  - Covers all device categories: lighting, blinds, climate, sensors, energy, controls, system

- **Tier 1 Function Type classification** (`code/core/internal/commissioning/etsimport/`):
  - `functionTypeToDeviceType()` — Maps 7 standard ETS Function Types to GLCore device types (0.95–0.99 confidence)
  - `commentToDeviceType()` — Maps 40+ KNXSim template IDs via Comment attribute (0.98 confidence)
  - Tier 1 runs before existing Tier 2 (DPT pattern matching) — consumed GAs are filtered out
  - Standard types: `SwitchableLight`, `DimmableLight`, `Sunblind`, `HeatingRadiator`, `HeatingFloor`, `HeatingContinuousVariable`, `HeatingSwitchingVariable`
  - Custom types use `Type="Custom" Comment="presence_detector"` pattern

- **ETS-standard .knxproj export sections** (`sim/knxsim/api/routes_export.py`):
  - `<Topology>` with `<DeviceInstance>` elements: IndividualAddress, ProductRefId, ApplicationProgramRef, ComObjectInstanceRefs
  - `<ManufacturerData>` with `<Manufacturer>`, `<Hardware>`, `<Product>`, `<ApplicationProgram>`
  - Standard ETS Function Types on `<Function>` elements (replaces `FT-*` codes)
  - Comment attribute for Custom function types carrying template ID

- **Metadata fields on DetectedDevice** (`types.go`):
  - `manufacturer`, `product_model`, `application_program`, `individual_address`, `function_type`

- **12 new device types** (`code/core/internal/device/types.go`):
  - KNX Controls: `scene_controller`, `push_button`, `binary_input`, `room_controller`, `logic_module`
  - KNX System: `ip_router`, `line_coupler`, `power_supply`, `timer_switch`, `load_controller`
  - Additional Sensors: `multi_sensor`, `wind_sensor`

- **14 new parser tests** (`parser_test.go`):
  - 8 unit tests for `functionTypeToDeviceType` mapping
  - Table-driven test for `commentToDeviceType` (14 subtests)
  - Full integration test with Topology + ManufacturerData + Functions XML
  - Tier 2 fallback regression test + CSV regression test

- **Expanded capability derivation** (`commissioning.go`):
  - 27 function→capability mappings (up from ~10): colour_temp, rgb, fan_speed, co2, contact, energy, power, voltage, current, hvac_mode, etc.

### Changed

- **KNXSim template loader** (`loader.py`): Parses `manufacturer:` block with legacy flat-format fallback
- **KNXSim device creation** (`routes_templates.py`, `routes_premises.py`): Injects manufacturer metadata into device config dict
- **KNXSim export** (`routes_export.py`): `_device_type_to_function_type()` returns `tuple[str, str]` (function_type, comment)
- **GLCore parser** (`parser.go`): Runs Tier 1 extraction before Tier 2; refactored `parseKNXProj` → `parseKNXProjWithXML`
- **GLCore import** (`commissioning.go`): `buildDeviceFromImport()` populates manufacturer, model, app_program, individual_address

### Architecture

```
KNXSim Template YAML        GLCore DB
(manufacturer, model,   →   (manufacturer, model,
 application_program)        application_program,
        │                    individual_address)
        ▼                           ▲
   .knxproj Export              Import
   ┌─────────────────┐    ┌─────────────────┐
   │ <DeviceInstance> │    │ Tier 1: FuncType │
   │ <ManufacturerData│───▶│ Tier 2: DPT rules│
   │ <Function Type>  │    │ (fallback only)  │
   └─────────────────┘    └─────────────────┘
```

### Files Modified

- `sim/knxsim/templates/**/*.yaml` (47 files) — Manufacturer metadata blocks
- `sim/knxsim/templates/loader.py` — Parse manufacturer metadata
- `sim/knxsim/api/routes_templates.py` — Inject metadata into device config
- `sim/knxsim/api/routes_premises.py` — Template lookup for config.yaml devices
- `sim/knxsim/api/routes_export.py` — Topology, ManufacturerData, standard Function Types
- `code/core/internal/commissioning/etsimport/parser.go` — Tier 1 extraction, XML structs
- `code/core/internal/commissioning/etsimport/types.go` — 5 metadata fields on DetectedDevice
- `code/core/internal/commissioning/etsimport/detection.go` — functionTypeToDeviceType(), commentToDeviceType()
- `code/core/internal/commissioning/etsimport/parser_test.go` — 14 new tests
- `code/core/internal/api/commissioning.go` — Import metadata, expanded capabilities
- `code/core/internal/device/types.go` — 12 new device types

---

## 1.0.17 – Flutter Dependency Upgrade & Riverpod v3 Migration (2026-02-03)

**Focus: Upgrade all outdated Flutter deps, migrate Riverpod v2→v3, fix all pre-existing test failures**

### Changed

- **flutter_riverpod**: ^2.6.1 → ^3.2.0 — Full StateNotifier→Notifier migration across 6 provider classes
- **dio**: ^5.7.0 → ^5.9.1
- **file_picker**: ^5.5.0 → ^10.3.10 — Eliminated ~24s analysis overhead from old transitive dependencies
- **Riverpod migration pattern** (applied to all 6 notifiers):
  - `StateNotifier<T>` → `Notifier<T>` with `build()` override returning initial state
  - Removed explicit `Ref` field — using built-in `ref` on `Notifier`
  - `dispose()` → `ref.onDispose()` in `build()`
  - `mounted` → `ref.mounted`
  - `AsyncValue.valueOrNull` → `.value` (returns `T?` in Riverpod v3)
  - `StateProvider` imports from `package:flutter_riverpod/legacy.dart`

### Fixed

- **ETS import `dart:html` removal**: Removed obsolete web-specific file picker using `dart:html` `FileUploadInputElement`/`FileReader`. `file_picker` v10 handles web natively — unified `_pickFile()` works on all platforms. Eliminates `dart:html` compile failure on Dart VM test runner.
- **WebSocket test invalid port**: Changed port 99999 (exceeds TCP max 65535) → port 19. Added try/catch for rethrown exception. Strengthened assertion to verify full `connecting→disconnected` lifecycle.
- **Device provider tests**: Fixed 2 tests that incorrectly expected optimistic state updates — code uses pending-confirmation pattern. Tests now verify device marked as pending while awaiting WebSocket confirmation.
- **Test results**: 51/55 → **55/55 passing** (4 pre-existing failures fixed)
- **Analysis**: 32s → **~3s** (file_picker v10 eliminates heavy transitive dependency crawl)

### Added

- **`build-panel-dev` Makefile target**: Quick Flutter web build with `--no-tree-shake-icons` for faster dev iteration
- **`/nuke-rebuild` Claude command**: Scorched-earth stack teardown and rebuild (5 phases: Kill → Purge → Rebuild → Launch → Verify)
- **`.gitignore` for panel build output**: `internal/panel/web/` excluded from git tracking

### Files Modified

- `code/ui/wallpanel/pubspec.yaml` — 3 dependency version bumps
- `code/ui/wallpanel/lib/providers/device_provider.dart` — 2 StateNotifiers → Notifiers + dispose→onDispose
- `code/ui/wallpanel/lib/providers/auth_provider.dart` — StateNotifier → Notifier
- `code/ui/wallpanel/lib/providers/location_provider.dart` — StateNotifier → Notifier + legacy import
- `code/ui/wallpanel/lib/providers/ets_import_provider.dart` — StateNotifier → Notifier
- `code/ui/wallpanel/lib/providers/scene_provider.dart` — StateNotifier → Notifier + mounted fix
- `code/ui/wallpanel/lib/screens/ets_import_screen.dart` — Removed dart:html, unified file picker, valueOrNull→value
- `code/ui/wallpanel/lib/screens/room_view.dart` — valueOrNull→value
- `code/ui/wallpanel/lib/screens/admin/devices_tab.dart` — valueOrNull→value
- `code/ui/wallpanel/lib/screens/app_shell.dart` — valueOrNull→value
- `code/ui/wallpanel/test/providers/device_provider_test.dart` — Fixed optimistic update expectations
- `code/ui/wallpanel/test/services/websocket_service_test.dart` — Fixed port + exception handling
- `code/ui/wallpanel/test/widgets/scene_button_test.dart` — Removed unused import
- `code/core/Makefile` — Added build-panel-dev target
- `code/core/.gitignore` — Added panel web output
- `.claude/commands/nuke-rebuild.md` — New command (created in prior session, updated build flag)

---

## 1.0.16 – Registry-Only Devices & ETS Auto Location (2026-02-03)

**Focus: Single source of truth for devices; zero-intervention ETS import**

### Removed

- **Static YAML device config**: Cleared `knx-bridge.yaml` devices to `[]`. Bridge starts empty — no more YAML→registry seeding on startup. Removed `seedDeviceRegistry()`, `buildDeviceSeed()`, and 6 helper functions (~570 net lines removed).
- **Config-takes-precedence logic**: Removed "skip if already in config" check from `loadDevicesFromRegistry()`. Registry is now the sole source of device→GA mappings.

### Added

- **ETS location extraction**: Implemented `extractLocations()` in the ETS parser — builds `Location` objects (areas/rooms) from GroupRange hierarchy paths. Filters domain names (Lighting, HVAC, Blinds, etc. EN+DE), deduplicates by slug, classifies leaf nodes as rooms and non-leaf as areas.
- **Auto room assignment**: Added `autoMapDeviceLocations()` in the import endpoint — maps `SuggestedRoom`/`SuggestedArea` to `RoomID`/`AreaID` after location creation. User-provided values take precedence.
- **`SuggestedRoom`/`SuggestedArea` fields** on `ETSDeviceImport` — carries parser suggestions through to import phase.
- **`createTestBridge()` test helper** — pre-loads device maps from config for bridge tests.
- **6 new location extraction tests** — domain-first, location-first, 3-level, dedup, empty, device source.

### Changed

- **Bridge startup**: `NewBridge()` starts with empty GA maps. `Start()` calls `loadDevicesFromRegistry()` as sole loading path. Health reporting uses registry count.
- **`extractLocations()` call site**: Moved from `parseGroupAddressesXML()` to `ParseBytes()` — works for all formats (knxproj, XML, CSV).

### Files Modified

- `code/core/configs/knx-bridge.yaml` — cleared devices
- `code/core/internal/bridges/knx/bridge.go` — registry-only loading
- `code/core/internal/bridges/knx/bridge_test.go` — test helper, removed dead tests
- `code/core/internal/bridges/knx/config.go` — updated doc comment
- `code/core/internal/commissioning/etsimport/parser.go` — location extraction
- `code/core/internal/commissioning/etsimport/parser_test.go` — 6 new tests
- `code/core/internal/api/commissioning.go` — auto-map device locations

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
