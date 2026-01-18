# Gray Logic Core - Implementation Log

> Tracking progress on the Go Core implementation (Year 1 Roadmap)

---

## 🚀 RESUME HERE — Next Session

**Last session:** 2026-01-18 (Session 1)
**Current milestone:** M1.1 Core Infrastructure (40% complete)

### Next Task: SQLite Database Package

**What to do:**
1. Create `internal/infrastructure/database/` package
2. Implement connection manager with WAL mode
3. Create migration runner
4. Write `migrations/001_initial_schema.sql` with:
   - `sites` table
   - `areas` table
   - `rooms` table
   - `devices` table
   - `audit_logs` table

**Start command:**
```bash
cd /home/darren/Development/Projects/gray-logic-stack/code/core
make build  # Verify everything still works
```

**Reference docs:**
- `docs/data-model/entities.md` — Entity definitions
- `docs/development/database-schema.md` — Schema design
- `docs/development/CODING-STANDARDS.md` — Migration file format

---

### Remaining M1.1 Tasks (in order)

| # | Task | Depends On |
|---|------|------------|
| 1 | **SQLite database package** ← START HERE | - |
| 2 | Initial schema migration | Task 1 |
| 3 | Docker Compose (Mosquitto + InfluxDB) | - |
| 4 | MQTT client package | Task 3 |
| 5 | InfluxDB client package | Task 3 |
| 6 | Wire config → main.go | Tasks 1, 4 |
| 7 | Basic structured logging | - |

---

## Current Status

| Milestone | Status | Progress |
|-----------|--------|----------|
| **M1.1** Core Infrastructure | 🟡 In Progress | 40% |
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
│   │   ├── database/           # SQLite (pending)
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
├── migrations/                 # SQLite migrations (pending)
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

- [ ] SQLite database package with migrations
- [ ] MQTT client package with auto-reconnect
- [ ] Docker Compose for Mosquitto + InfluxDB
- [ ] InfluxDB client package
- [ ] Basic structured logging

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

## Notes

- Project started: 2026-01-18
- Using UK English throughout (per project standards)
- All timestamps in UTC (per CODING-STANDARDS.md)
- Configuration supports environment variable overrides for secrets
