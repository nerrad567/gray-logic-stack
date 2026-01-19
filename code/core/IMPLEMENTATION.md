# Gray Logic Core - Implementation Log

> Tracking progress on the Go Core implementation (Year 1 Roadmap)

---

## 🚀 RESUME HERE — Next Session

**Last session:** 2026-01-19 (Session 5)
**Current milestone:** M1.1 Core Infrastructure (95% complete)

### Next Task: Wire Infrastructure into main.go

**What to do:**
1. Initialise database in `cmd/graylogic/main.go`
2. Initialise MQTT client with config
3. Initialise InfluxDB client (if enabled)
4. Add health check endpoints
5. Graceful shutdown for all connections

**Docker services running:**
```bash
docker compose ps  # Should show graylogic-mosquitto and graylogic-influxdb healthy
```

**Start command:**
```bash
cd /home/darren/Development/Projects/gray-logic-stack/code/core
make build && make test  # Verify everything still works
```

**Reference docs:**
- `docs/technical/packages/` — Package design documentation
- `docs/technical/decisions/` — Implementation decision records

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
| 7 | Wire database + config → main.go | ⬜ Pending | Tasks 2, 5, 6 |
| 8 | Basic structured logging | ⬜ Pending | - |

---

## Current Status

| Milestone | Status | Progress |
|-----------|--------|----------|
| **M1.1** Core Infrastructure | 🟡 In Progress | 95% |
| M1.2 KNX Bridge | ⬜ Not Started | 0% |
| M1.3 Device Registry | ⬜ Not Started | 0% |
| M1.4 REST API + WebSocket | ⬜ Not Started | 0% |
| M1.5 Flutter Wall Panel | ⬜ Not Started | 0% |
| M1.6 Basic Scenes | ⬜ Not Started | 0% |

---

## M1.1: Core Infrastructure Setup

### Completed Tasks

#### 2026-01-18: Project Initialisation

**Go Module Setup**
- [x] Initialised Go module: `github.com/nerrad567/gray-logic-core`
- [x] Go version: 1.23.4
- [x] Created directory structure per CODING-STANDARDS.md

**Directory Structure Created:**
```
code/core/
├── cmd/graylogic/main.go       # Application entry point
├── internal/
│   ├── infrastructure/
│   │   ├── config/             # Configuration loading ✓
│   │   ├── database/           # SQLite + migrations ✓
│   │   ├── mqtt/               # MQTT client (pending)
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

### Remaining Tasks (M1.1)

- [x] SQLite database package with migrations ✅
- [x] Database tests (80.4% coverage) ✅
- [x] Docker Compose for Mosquitto + InfluxDB ✅
- [ ] MQTT client package with auto-reconnect
- [ ] InfluxDB client package
- [ ] Basic structured logging
- [ ] Wire everything into main.go

---

## Technical Decisions

### Why These Choices?

| Decision | Rationale |
|----------|-----------|
| **Go 1.23** | Latest stable, no bleeding-edge features |
| **gopkg.in/yaml.v3** | Mature, well-maintained, supports comments |
| **golangci-lint** | Industry standard, catches security issues |
| **Makefile** | Universal, works in 10 years, no dependencies |
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
   go version  # Confirmed Go 1.23.4 installed
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
   | Additive-only migrations | 10-year stability — never ALTER/DROP in production |

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

## Notes

- Project started: 2026-01-18
- Using UK English throughout (per project standards)
- All timestamps in UTC (per CODING-STANDARDS.md)
- Configuration supports environment variable overrides for secrets

