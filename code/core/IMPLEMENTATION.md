# Gray Logic Core - Implementation Log

> Tracking progress on the Go Core implementation (Year 1 Roadmap)

---

## 🚀 RESUME HERE — Next Session

**Last session:** 2026-01-20 (Session 8 - KNX Bridge Phase 1)
**Current milestone:** M1.2 KNX Bridge (30% complete)

### Next Task: Continue KNX Bridge Implementation (Phase 2)

**Completed:**
- ✅ telegram.go — KNX telegram parsing/encoding
- ✅ knxd.go — knxd client with TCP/Unix socket support
- ✅ telegram_test.go — 21 tests
- ✅ knxd_test.go — 12 tests (includes MockKNXDServer)

**What to do next:**
1. Create `config.go` — Bridge configuration types (YAML loading)
2. Create `messages.go` — MQTT message types for state/command
3. Create `health.go` — Health reporting to MQTT
4. Create `bridge.go` — Main orchestration logic

**All tests passing:**
```bash
cd /home/graylogic-dev/gray-logic-stack/code/core
go test -v ./internal/bridges/knx/...  # 33 tests OK
```

**Reference docs:**
- `.claude/plans/dynamic-snacking-fiddle.md` — Full implementation plan
- `docs/technical/packages/knx-bridge.md` — Package design doc
- `docs/protocols/knx.md` — KNX protocol specification

### M1.1 Hardening Complete (Session 7)

**Edge case tests added:**
- Database: unwritable directory, context cancellation, busy timeout, invalid SQL
- Logging: invalid output defaults, empty config, chained loggers
- InfluxDB: disabled returns nil, health check after close, flush after close
- MQTT: broker refused, uninitialised state, multiple subscriptions, nil/large payloads

**Bugs fixed:**
- `influxdb.Flush()` panicked when called after `Close()` — now safe

**Improvements:**
- MQTT client now supports optional logger via `SetLogger()`
- Removed obsolete TODO comments
- Test coverage: config 100%, logging 100%, database 81.9%, influxdb 82.1%, mqtt 82.1%

---

### Remaining M1.1 Tasks (in order)

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | SQLite database package | ✅ Done | - |
| 2 | Initial schema migration | ✅ Done | Task 1 |
| 3 | Database tests | ✅ Done | Task 2 |
| 4 | Docker Compose (Mosquitto + InfluxDB) | ✅ Done | - |
| 5 | MQTT client package | ✅ Done | Task 4 |
| 6 | InfluxDB client package | ✅ Done | Task 4 |
| 7 | Wire database + config → main.go | ✅ Done | Tasks 2, 5, 6 |
| 8 | Basic structured logging | ✅ Done | - |

---

## 5-Year Roadmap Overview

> Full details in [DEVELOPMENT-STRATEGY.md](../../docs/development/DEVELOPMENT-STRATEGY.md)

### Year 1 — Foundation (2026) ← **Current**
| Milestone | Goal | Status |
|-----------|------|--------|
| **M1.1** | Core Infrastructure (SQLite, MQTT, InfluxDB) | ✅ Complete |
| **M1.2** | KNX Bridge | 🔨 30% |
| M1.3 | Device Registry | ⬜ Not Started |
| M1.4 | REST API + WebSocket | ⬜ Not Started |
| M1.5 | Flutter Wall Panel | ⬜ Not Started |
| M1.6 | Basic Scenes | ⬜ Not Started |

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

## Year 1 Task Breakdown

### M1.2: KNX Bridge

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | telegram.go — KNX telegram parsing/encoding | ✅ Done | - |
| 2 | knxd.go — knxd client (TCP/Unix) | ✅ Done | Task 1 |
| 3 | address.go — Group address parsing | ✅ Done | - |
| 4 | dpt.go — Datapoint type encoding/decoding | ✅ Done | - |
| 5 | config.go — Bridge configuration (YAML) | ⬜ Not Started | - |
| 6 | messages.go — MQTT message types | ⬜ Not Started | - |
| 7 | health.go — Health reporting | ⬜ Not Started | - |
| 8 | bridge.go — Main orchestration | ⬜ Not Started | Tasks 1-7 |
| 9 | Integration tests | ⬜ Not Started | Task 8 |
| 10 | Fix lint issues | ⬜ Not Started | Tasks 1-4 |

### M1.3: Device Registry

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | Database schema (devices, capabilities) | ⬜ Not Started | M1.1 |
| 2 | repository.go — CRUD operations | ⬜ Not Started | Task 1 |
| 3 | service.go — Business logic, validation | ⬜ Not Started | Task 2 |
| 4 | KNX device registration with GAs | ⬜ Not Started | M1.2, Task 3 |
| 5 | Seed script for test devices | ⬜ Not Started | Task 4 |
| 6 | Unit tests (80%+ coverage) | ⬜ Not Started | Tasks 2-3 |

### M1.4: REST API + WebSocket

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | server.go — Router setup, middleware | ⬜ Not Started | - |
| 2 | GET /api/v1/devices | ⬜ Not Started | M1.3 |
| 3 | GET /api/v1/devices/{id} | ⬜ Not Started | M1.3 |
| 4 | POST /api/v1/devices/{id}/command | ⬜ Not Started | M1.3 |
| 5 | websocket.go — Real-time state updates | ⬜ Not Started | Task 1 |
| 6 | TLS configuration (self-signed) | ⬜ Not Started | Task 1 |
| 7 | Basic auth placeholder | ⬜ Not Started | Task 1 |
| 8 | Load test (10 clients, 100 cmd/s) | ⬜ Not Started | Tasks 1-5 |

### M1.5: Flutter Wall Panel

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | Flutter project setup (ui/wallpanel) | ⬜ Not Started | - |
| 2 | Device models (matching Go structs) | ⬜ Not Started | M1.3 |
| 3 | Repository pattern for API calls | ⬜ Not Started | M1.4 |
| 4 | WebSocket connection for state updates | ⬜ Not Started | M1.4 |
| 5 | RoomView widget | ⬜ Not Started | Tasks 2-4 |
| 6 | SwitchTile (on/off) | ⬜ Not Started | Task 5 |
| 7 | DimmerTile (slider) | ⬜ Not Started | Task 5 |
| 8 | Response time <200ms validation | ⬜ Not Started | Tasks 5-7 |

### M1.6: Basic Scenes

| # | Task | Status | Depends On |
|---|------|--------|------------|
| 1 | Database schema (scenes, scene_actions) | ⬜ Not Started | M1.1 |
| 2 | Scene struct definition | ⬜ Not Started | Task 1 |
| 3 | scene_engine.go — Activate, parallel execution | ⬜ Not Started | Task 2 |
| 4 | GET /api/v1/scenes | ⬜ Not Started | M1.4 |
| 5 | POST /api/v1/scenes/{id}/activate | ⬜ Not Started | Task 3 |
| 6 | Scene execution <500ms for 10 devices | ⬜ Not Started | Task 3 |
| 7 | Persist scene state across restarts | ⬜ Not Started | Task 3 |

---

## M1.1: Core Infrastructure Setup

### Completed Tasks

#### 2026-01-18: Project Initialisation

**Go Module Setup**
- [x] Initialised Go module: `github.com/nerrad567/gray-logic-core`
- [x] Go version: 1.25.6
- [x] Created directory structure per CODING-STANDARDS.md

**Directory Structure Created:**
```
code/core/
├── cmd/graylogic/main.go       # Application entry point
├── internal/
│   ├── infrastructure/
│   │   ├── config/             # Configuration loading ✓
│   │   ├── database/           # SQLite + migrations ✓
│   │   ├── mqtt/               # MQTT client ✓
│   │   ├── influxdb/           # InfluxDB client ✓
│   │   ├── logging/            # Structured logging ✓
│   │   └── api/                # HTTP server (pending)
│   ├── device/
│   │   ├── registry/           # Device CRUD (pending)
│   │   ├── state/              # State management (pending)
│   │   └── command/            # Command routing (pending)
│   └── automation/
│       ├── scene/              # Scene engine (pending)
│       └── scheduler/          # Time triggers (pending)
├── pkg/models/                 # Shared data structures (pending)
├── migrations/                 # SQLite migrations ✓
│   ├── embed.go
│   └── *.sql
├── configs/config.yaml         # Default configuration ✓
├── tests/
│   ├── integration/            # Integration tests (pending)
│   └── fixtures/               # Test data (pending)
├── go.mod                      # Module definition ✓
├── go.sum                      # Dependency checksums ✓
├── Makefile                    # Build automation ✓
└── .golangci.yml               # Linter configuration ✓
```

**Build System**
- [x] Created Makefile with targets: build, run, test, lint, fmt, vet, check
- [x] Version injection via ldflags (commit hash, build date)
- [x] Installed golangci-lint v1.64.8
- [x] Configured strict linting (.golangci.yml) with:
  - Security linting (gosec)
  - Error checking (errcheck, errorlint)
  - Code quality (gocyclo, gocognit, goconst)
  - UK English spelling (misspell)

**Configuration System**
- [x] Created YAML configuration file (configs/config.yaml)
- [x] Implemented config loading package (internal/infrastructure/config/)
- [x] Environment variable override support (GRAYLOGIC_*)
- [x] Configuration validation
- [x] Added gopkg.in/yaml.v3 dependency

**Main Application**
- [x] Entry point with graceful shutdown (signal handling)
- [x] Context-based cancellation pattern
- [x] Testable structure (run function separated from main)

### Completed Tasks (M1.1)

- [x] SQLite database package with migrations ✅
- [x] Database tests (80.4% coverage) ✅
- [x] Docker Compose for Mosquitto + InfluxDB ✅
- [x] MQTT client package with auto-reconnect ✅
- [x] InfluxDB client package ✅
- [x] Basic structured logging ✅
- [x] Wire everything into main.go ✅

---

## Technical Decisions

### Why These Choices?

| Decision | Rationale |
|----------|-----------|
| **Go 1.25** | Latest stable, no bleeding-edge features |
| **gopkg.in/yaml.v3** | Mature, well-maintained, supports comments |
| **golangci-lint** | Industry standard, catches security issues |
| **Makefile** | Universal, works for decades, no dependencies |
| **internal/ directory** | Go-enforced encapsulation |

### Dependencies Added

| Package | Version | Purpose |
|---------|---------|---------|
| gopkg.in/yaml.v3 | v3.0.1 | YAML configuration parsing |
| github.com/mattn/go-sqlite3 | v1.14.33 | SQLite database driver (CGO) |

---

## Commands Reference

```bash
# Build the application
make build

# Run the application
make run

# Run all tests
make test

# Run linter
make lint

# Format code
make fmt

# All quality checks
make check

# Show all commands
make help
```

---

## Session Log

### Session 1: 2026-01-18 — Project Initialisation

**Goal:** Set up Go project structure (Step 1 of M1.1)

**Steps Taken:**

1. **Checked prerequisites**
   ```bash
   go version  # Confirmed Go 1.25.6 installed
   ls code/    # Found only archive/ directory existed
   ```

2. **Created directory structure**
   ```bash
   mkdir -p code/core/{cmd/graylogic,internal/{infrastructure/{database,mqtt,api,config},device/{registry,state,command},automation/{scene,scheduler}},pkg/models,migrations,configs,tests/{integration,fixtures}}
   ```
   - `cmd/` — Entry points (executables)
   - `internal/` — Private code (Go-enforced)
   - `pkg/` — Public code (for bridges to import)

3. **Initialised Go module**
   ```bash
   cd code/core
   go mod init github.com/nerrad567/gray-logic-core
   ```
   - Creates `go.mod` with module path and Go version

4. **Created main.go entry point**
   - Location: `cmd/graylogic/main.go`
   - Key patterns implemented:
     - `signal.NotifyContext` for graceful shutdown (Ctrl+C handling)
     - Separate `run()` function for testability
     - Version variables injected at build time via ldflags
   - Tested: `go build -o graylogic ./cmd/graylogic && ./graylogic`

5. **Created Makefile**
   - 15 targets: build, run, test, lint, fmt, vet, check, clean, etc.
   - Version injection: extracts git commit, tag, and build date
   - Tested: `make help`, `make build`

6. **Installed and configured golangci-lint**
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```
   - Created `.golangci.yml` with strict settings
   - Enabled security linter (gosec), error checking, UK spelling
   - Fixed issues found:
     - Changed "Initialize" → "Initialise" (UK spelling)
     - Added `//nolint:unparam` comment with explanation
     - Fixed deprecated config options

7. **Created configuration system**
   - `configs/config.yaml` — Default configuration with all options documented
   - `internal/infrastructure/config/config.go` — Loading logic
   - `internal/infrastructure/config/doc.go` — Package documentation
   - Added dependency: `go get gopkg.in/yaml.v3`
   - Features:
     - Layered loading: defaults → YAML file → environment variables
     - Validation of required fields and ranges
     - Environment overrides for secrets (GRAYLOGIC_*)

8. **Created supporting files**
   - `.gitignore` — Excludes build/, *.db, *.log, .env
   - `IMPLEMENTATION.md` — This progress tracking file
   - `docs/GETTING-STARTED.md` — Comprehensive explanation of all concepts

9. **Updated project status**
   - `PROJECT-STATUS.md` — Changed phase to "Implementation (M1.1)"
   - Updated code status from "Not started" to "M1.1 In Progress"

**Issues Encountered & Resolved:**

| Issue | Resolution |
|-------|------------|
| Linter flagged "Initialize" as misspelling | Changed to UK English "Initialise" |
| Linter warned about unused error return | Added `//nolint:unparam` with explanation |
| Empty if-branch warning | Converted to comment explaining the intent |
| Deprecated golangci-lint config options | Updated to new option names |
| Stray binary in project root | Added to .gitignore, cleaned up |

**Commands Used (for reference):**

```bash
# Check Go installation
go version

# Create project structure
mkdir -p code/core/{cmd/graylogic,internal/...}

# Initialise module
go mod init github.com/nerrad567/gray-logic-core

# Add dependency
go get gopkg.in/yaml.v3
go mod tidy

# Build and test
make build
make run
make lint

# Install linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Format code
gofmt -w .
```

**Time Spent:** ~1 hour

**Outcome:** Foundation complete. Project builds, lints clean, and runs with graceful shutdown.

---

### Session 2: 2026-01-19 — SQLite Database Package

**Goal:** Implement SQLite database package with migrations (Step 2 of M1.1)

**Steps Taken:**

1. **Read specification documents**
   - `docs/data-model/entities.md` — Entity definitions (Site, Area, Room, Device, etc.)
   - `docs/development/database-schema.md` — Schema design (STRICT mode, additive-only)

2. **Created database package**
   - Location: `internal/infrastructure/database/`
   - Files created:
     - `doc.go` — Package documentation
     - `database.go` — Connection manager with WAL mode
     - `migrations.go` — Migration runner with embed.FS support

3. **Key implementation decisions:**

   | Decision | Rationale |
   |----------|-----------|
   | `embed.FS` for migrations | Compiles SQL into binary, no external files needed at runtime |
   | WAL mode | Better concurrent access (readers don't block writers) |
   | STRICT mode | SQLite type enforcement (prevents silent data issues) |
   | Additive-only migrations | Multi-decade stability — never ALTER/DROP in production |

4. **Created initial schema migration**
   - File: `migrations/20260118_200000_initial_schema.up.sql`
   - Tables: `sites`, `areas`, `rooms`, `devices`, `audit_logs`
   - Includes indexes on foreign keys and common query fields
   - View: `devices_with_location` for hierarchical queries

5. **Created down migration**
   - File: `migrations/20260118_200000_initial_schema.down.sql`
   - Drops tables in reverse dependency order
   - For development/testing only

6. **Created embed package**
   - Location: `migrations/embed.go`
   - Uses `//go:embed *.sql` directive
   - Registers with database package via `init()`

7. **Fixed linter issues:**

   | Issue | Resolution |
   |-------|------------|
   | Magic numbers (0750, 0600, 1000) | Added named constants |
   | gofmt formatting | Ran `gofmt -w` |
   | Error wrapping (wrapcheck) | Added `fmt.Errorf("context: %w", err)` |
   | Cognitive complexity (>20) | Refactored into helper functions |
   | Unchecked os.Chmod error | Added `//nolint:errcheck` with explanation |

8. **Verified build**
   ```bash
   make lint   # All checks pass
   make build  # Compiles successfully
   ./build/graylogic --version  # Runs correctly
   ```

**Dependencies Added:**

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/mattn/go-sqlite3 | v1.14.33 | SQLite driver (CGO) |

**Files Created:**

```
internal/infrastructure/database/
├── doc.go         # Package documentation
├── database.go    # Connection manager (Config, Open, Close, HealthCheck)
└── migrations.go  # Migration runner (Migrate, MigrateDown, GetMigrationStatus)

migrations/
├── embed.go                                    # Embeds SQL files
├── 20260118_200000_initial_schema.up.sql      # Create tables
└── 20260118_200000_initial_schema.down.sql    # Drop tables
```

**Code Patterns Introduced:**

- **Embedded Filesystem (`embed.FS`)**: Compiles files into binary
- **Named Return Values**: `(upFiles, downFiles map[string]string)` for clarity
- **Helper Functions**: Reduce cognitive complexity, improve testability
- **Deferred Rollback**: `defer tx.Rollback()` is no-op after commit

**Outcome:** Database package complete. Migrations compile into binary. Ready for tests or Docker Compose setup.

---

### Session 3: 2026-01-19 — Database Tests & Docker Compose

**Goal:** Add database tests and set up Docker infrastructure

**Steps Taken:**

1. **Created database tests**
   - `database_test.go` — Connection, health check, transactions
   - `migrations_test.go` — Migration apply, rollback, status
   - `testdata/*.sql` — Embedded test fixtures
   - Coverage: 80.4% of database package
   - 21 test cases, all passing

2. **Created Docker Compose infrastructure**
   - File: `docker-compose.yml`
   - Services: Mosquitto (MQTT), InfluxDB (time-series)
   - **Isolation measures** (to protect existing media-stack):
     - Project name: `graylogic`
     - Network: `graylogic` (separate from `media-stack_default`)
     - Ports bound to `127.0.0.1` only
     - Container names: `graylogic-mosquitto`, `graylogic-influxdb`
     - Volume names: `graylogic_*` prefix

3. **Created Mosquitto configuration**
   - File: `docker/mosquitto/config/mosquitto.conf`
   - Listeners: 1883 (MQTT), 9001 (WebSocket)
   - Anonymous access (dev only)
   - Logging enabled for debugging

4. **Verified isolation**
   ```bash
   # Gray Logic containers on separate network
   docker network inspect graylogic  # Only graylogic containers

   # Media stack unaffected
   docker ps | grep jellyfin  # Still running

   # Ports bound to localhost only
   ss -tlnp | grep 1883  # Shows 127.0.0.1:1883
   ```

**Files Created:**

```
docker-compose.yml                    # Service definitions
docker/
├── mosquitto/
│   └── config/
│       └── mosquitto.conf            # Broker configuration
└── influxdb/                         # (volumes managed by Docker)

internal/infrastructure/database/
├── database_test.go                  # 210 lines
├── migrations_test.go                # 260 lines
└── testdata/
    ├── 20260101_000000_test.up.sql
    └── 20260101_000000_test.down.sql
```

**Docker Services Running:**

| Service | Container | Port | Status |
|---------|-----------|------|--------|
| Mosquitto | graylogic-mosquitto | 127.0.0.1:1883 | ✅ Healthy |
| InfluxDB | graylogic-influxdb | 127.0.0.1:8086 | ✅ Healthy |

**InfluxDB Credentials (dev):**
- URL: http://127.0.0.1:8086
- Username: admin
- Password: graylogic-dev-password
- Organisation: graylogic
- Bucket: metrics
- Token: graylogic-dev-token

**Outcome:** Infrastructure complete. MQTT broker and time-series DB running locally. Ready for MQTT client package.

---

### Session 4: 2026-01-19 — MQTT Client Package

**Goal:** Implement MQTT client package with auto-reconnect and subscription management

**Steps Taken:**

1. **Added paho.mqtt.golang dependency**
   ```bash
   go get github.com/eclipse/paho.mqtt.golang@v1.4.3
   ```
   - Eclipse Foundation library, 10+ years stable
   - MIT licensed, widely used in production

2. **Created package structure**
   ```
   internal/infrastructure/mqtt/
   ├── doc.go          # Package documentation
   ├── errors.go       # ErrNotConnected, etc.
   ├── options.go      # Connection options builder
   ├── topics.go       # Topic namespace helpers
   ├── client.go       # Main client with lifecycle
   ├── publish.go      # Publish with QoS
   ├── subscribe.go    # Subscribe with tracking
   └── client_test.go  # Integration tests
   ```

3. **Implemented key features:**

   | Feature | Implementation |
   |---------|----------------|
   | **Auto-reconnect** | Paho built-in with exponential backoff |
   | **LWT (Last Will)** | Published on unexpected disconnect |
   | **Subscription tracking** | `map[string]subscription` restored on reconnect |
   | **Handler wrapping** | Panic recovery prevents crash |
   | **Thread safety** | RWMutex for subscriptions and state |

4. **Topic namespace design:**
   ```go
   mqtt.Topics{}.BridgeState("knx-01", "light-living")
   // → "graylogic/bridge/knx-01/state/light-living"
   
   mqtt.Topics{}.SystemStatus()
   // → "graylogic/system/status"
   ```

5. **Fixed synchronous connected state issue:**
   - OnConnectHandler runs asynchronously
   - Set `connected = true` immediately after Connect() returns
   - Prevents race where IsConnected() returns false after successful connect

**Dependencies Added:**

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/eclipse/paho.mqtt.golang | v1.4.3 | MQTT client library |

**Files Created:**

| File | Lines | Purpose |
|------|-------|---------|
| doc.go | 57 | Package documentation with usage examples |
| errors.go | ~35 | Domain-specific error types |
| options.go | ~100 | Connection options and LWT configuration |
| topics.go | ~200 | Topic builders for namespace |
| client.go | 259 | Main client with connect/close/health |
| publish.go | ~60 | Publish with timeout and QoS |
| subscribe.go | ~100 | Subscribe/Unsubscribe with tracking |
| client_test.go | ~500 | Integration tests against Mosquitto |

**Technical Decisions:**

| Decision | Rationale |
|----------|-----------|
| Subscription tracking | Restore subscriptions after reconnect |
| Synchronous connected state | Avoid race condition with async callback |
| Wrapped handlers | Panic recovery prevents broker disconnect |
| Topic builders | Type-safe topic construction |

**Issues Encountered:**

| Issue | Resolution |
|-------|------------|
| Connect callback async | Set connected state synchronously after Connect() |
| Handler panic crashes | Wrap all handlers with recover() |

**Outcome:** MQTT package complete with auto-reconnect, subscription tracking, and topic helpers. M1.1 progress: 85% → 90%.

---

### Session 5: 2026-01-19 — InfluxDB Client Package

**Goal:** Implement InfluxDB client package for time-series storage

**Steps Taken:**

1. **Added dependency**
   ```bash
   go get github.com/influxdata/influxdb-client-go/v2@latest
   ```
   - Version v2.14.0 installed

2. **Created package structure**
   ```
   internal/infrastructure/influxdb/
   ├── doc.go          # Package documentation
   ├── errors.go       # ErrNotConnected, ErrDisabled, etc.
   ├── client.go       # Connect, Close, HealthCheck
   ├── write.go        # WriteDeviceMetric, WriteEnergyMetric, etc.
   └── client_test.go  # Integration tests (11 tests)
   ```

3. **Key implementation decisions:**

   | Decision | Rationale |
   |----------|-----------|
   | Non-blocking WriteAPI | High-frequency sensor data shouldn't block Core |
   | Batching (config-driven) | Reduce network overhead |
   | Error callback | Async writes can't return errors synchronously |
   | Ping health check | Simple, built-in, no custom queries |

4. **Fixed lint issues:**

   | Issue | Resolution |
   |-------|------------|
   | G115 integer overflow | Added validation for batch_size/flush_interval |
   | Magic number 1000 | Created `millisecondsPerSecond` constant |
   | Error comparison | Changed to `errors.Is()` |

5. **Created documentation:**
   - `docs/technical/packages/influxdb.md` — Package design doc
   - Updated `docs/technical/README.md` index
   - Updated `configs/config.yaml` bucket: `telemetry` → `metrics`

**Dependencies Added:**

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/influxdata/influxdb-client-go/v2 | v2.14.0 | Official InfluxDB v2 client |

**Files Created:**

| File | Lines | Purpose |
|------|-------|---------|
| doc.go | 45 | Package documentation |
| errors.go | 25 | Domain-specific errors |
| client.go | 200 | Main client with lifecycle |
| write.go | 140 | Metric writing helpers |
| client_test.go | 250 | Integration tests |

**Test Results:**
- 11 tests passing
- All lint checks clean
- Integration tests require running InfluxDB

**Outcome:** InfluxDB package complete with non-blocking writes, health check, and domain helpers. M1.1 progress: 90% → 95%.

---

### Session 6: 2026-01-20 — Infrastructure Wiring & Structured Logging

**Goal:** Complete M1.1 by wiring infrastructure into main.go and adding structured logging

**Steps Taken:**

1. **Created structured logging package**
   ```
   internal/infrastructure/logging/
   ├── doc.go          # Package documentation
   ├── logger.go       # Logger setup with JSON/Text output
   └── logger_test.go  # Unit tests (8 tests)
   ```
   - Uses Go stdlib `log/slog` (no external dependencies)
   - Supports JSON (production) and text (development) formats
   - Default fields: service="graylogic", version from build vars
   - Level-based filtering (debug, info, warn, error)

2. **Updated main.go with infrastructure wiring**
   - Load configuration from YAML file
   - Open SQLite database with `database.Open()`
   - Run migrations with `db.Migrate()`
   - Connect MQTT client with `mqtt.Connect()`
   - Connect InfluxDB client (if enabled) with `influxdb.Connect()`
   - Added health check for all connections
   - Graceful shutdown with defers in reverse order (LIFO)
   - MQTT connect/disconnect logging callbacks

3. **Fixed lint issues**
   - Resolved variable shadowing in defer closures
   - All lint checks pass

**Files Created/Modified:**

| File | Lines | Purpose |
|------|-------|---------|
| logging/doc.go | 32 | Package documentation |
| logging/logger.go | 107 | Logger implementation |
| logging/logger_test.go | 162 | Unit tests |
| cmd/graylogic/main.go | 197 | Infrastructure wiring |

**Outcome:** M1.1 Core Infrastructure complete (100%). Ready to start M1.2 KNX Bridge.

---

### Session 7: 2026-01-20 — M1.1 Final Code Audit

**Goal:** Complete code audit before starting M1.2

**Steps Taken:**

1. **Ran `/code-audit` command** (4 iterations)
   - All static checks passed (lint, vet, vulncheck)
   - Fixed JWT validation (cache + hmac.Equal)
   - Fixed InfluxDB shutdown order (flush before close)
   - Added context timeout validation
   - Added bounds validation for batch sizes
   - Added MQTT payload size limit

2. **Modified `/code-audit` command**
   - Added automatic run tracking (`.claude/audit-history.json`)
   - Auto-switches to Final Advisory mode on 3rd+ run
   - Resets count when git HEAD changes

**Files Created:**
- `.claude/audit-history.json` — Tracks audit runs
- Updated `.claude/commands/code-audit.md` — Auto-tracking

**Outcome:** M1.1 shipped with all audit findings addressed. Ready for M1.2.

---

### Session 8: 2026-01-20 — M1.2 KNX Bridge Phase 1

**Goal:** Implement low-level KNX communication (telegram parsing, knxd client)

**Steps Taken:**

1. **Created implementation plan**
   - Explored MQTT client patterns for template
   - Researched knxd protocol format
   - Plan saved to `.claude/plans/dynamic-snacking-fiddle.md`

2. **Created telegram.go (~180 lines)**
   - KNX telegram parsing and encoding
   - knxd message framing (EIB_OPEN_GROUPCON, EIB_GROUP_PACKET)
   - APCI types: Read (0x00), Response (0x40), Write (0x80)

3. **Created telegram_test.go (~340 lines)**
   - 21 test cases: parse, encode, roundtrip, helpers
   - Found and fixed bug: single-byte values > 0x3F were not being encoded correctly

4. **Created knxd.go (~350 lines)**
   - `KNXDClient` with TCP/Unix socket support
   - `Connect()` with handshake (EIB_OPEN_GROUPCON)
   - `Send()` and `SendRead()` for group telegrams
   - `SetOnTelegram()` callback for incoming telegrams
   - Atomic statistics (TelegramsTx, TelegramsRx, Errors)
   - `Connector` interface for mocking in tests

5. **Created knxd_test.go (~380 lines)**
   - `MockKNXDServer` simulates knxd for testing
   - 12 test cases: connect, send, receive, close, stats

6. **Created technical documentation**
   - `docs/technical/packages/knx-bridge.md` — Full package design doc
   - Updated `docs/technical/README.md` index

7. **Created `/explain-code` command**
   - For understanding recently written code
   - Includes ASCII diagrams, data flows, key concepts

**Files Created:**

```
internal/bridges/knx/
├── telegram.go        # NEW: KNX telegram parse/encode
├── telegram_test.go   # NEW: 21 tests
├── knxd.go           # NEW: knxd client
└── knxd_test.go      # NEW: 12 tests + MockKNXDServer

docs/technical/packages/
└── knx-bridge.md     # NEW: Package design doc

.claude/commands/
└── explain-code.md   # NEW: Code explanation command
```

**Bugs Fixed:**

| Bug | Fix |
|-----|-----|
| telegram.Encode() single-byte > 0x3F | Added `smallData` check to detect when value fits in APCI byte |

**Test Results:**
- All 33 KNX tests passing
- All lint checks clean

**Technical Patterns Used:**

| Pattern | Purpose |
|---------|---------|
| Atomic counters | Lock-free stats (TelegramsTx, etc.) |
| Connector interface | Enables mock testing |
| Callback with panic recovery | Prevents crashes from bad handlers |
| TCP + Unix socket support | Flexible deployment options |

**Outcome:** Phase 1 complete (telegram parsing + knxd client). M1.2 progress: 30%.

---

## Notes

- Project started: 2026-01-18
- Using UK English throughout (per project standards)
- All timestamps in UTC (per CODING-STANDARDS.md)
- Configuration supports environment variable overrides for secrets

