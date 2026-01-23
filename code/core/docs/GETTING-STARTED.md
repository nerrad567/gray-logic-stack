# Gray Logic Core — Getting Started Guide

> A comprehensive explanation of the Go project structure, patterns, and decisions made during initial setup.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Directory Structure Explained](#directory-structure-explained)
3. [File-by-File Breakdown](#file-by-file-breakdown)
4. [Go Concepts Explained](#go-concepts-explained)
5. [Build System](#build-system)
6. [Configuration System](#configuration-system)
7. [Code Quality Tools](#code-quality-tools)
8. [Common Commands](#common-commands)
9. [Next Steps](#next-steps)

---

## Project Overview

Gray Logic Core is a building automation system written in Go. The project follows Go's standard conventions and is designed for:

- **Multi-decade deployment stability** — Dependencies are pinned, code is simple
- **Offline-first operation** — No cloud dependencies for core functionality
- **Security** — Strict linting catches vulnerabilities early
- **Testability** — Structure enables comprehensive testing

### Technology Choices

| Choice | Why |
|--------|-----|
| **Go** | Compiles to single binary, no runtime needed, stable for decades |
| **YAML config** | Human-readable, supports comments, mature library |
| **SQLite** | Embedded database, zero maintenance, reliable |
| **MQTT** | Simple message bus, proven in IoT, debuggable |

---

## Directory Structure Explained

```
code/core/
├── cmd/                    ← EXECUTABLES (entry points)
│   └── graylogic/
│       └── main.go         ← The "main" function lives here
│
├── internal/               ← PRIVATE CODE (cannot be imported externally)
│   ├── infrastructure/     ← "Plumbing" — database, network, HTTP
│   │   ├── api/            ← REST API server (future)
│   │   ├── config/         ← Configuration loading ✓
│   │   ├── database/       ← SQLite connection (future)
│   │   └── mqtt/           ← MQTT client (future)
│   ├── device/             ← Device management
│   │   ├── command/        ← Command routing (future)
│   │   ├── registry/       ← Device CRUD (future)
│   │   └── state/          ← State tracking (future)
│   └── automation/         ← Automation logic
│       ├── scene/          ← Scene engine (future)
│       └── scheduler/      ← Time-based triggers (future)
│
├── pkg/                    ← PUBLIC CODE (can be imported by bridges)
│   └── models/             ← Shared data structures (future)
│
├── configs/                ← Default configuration files
│   └── config.yaml         ← Default settings ✓
│
├── migrations/             ← Database migrations (future)
├── tests/                  ← Test files
│   ├── integration/        ← End-to-end tests (future)
│   └── fixtures/           ← Test data (future)
│
├── build/                  ← Compiled binaries (gitignored)
│
├── go.mod                  ← Module definition ✓
├── go.sum                  ← Dependency checksums ✓
├── Makefile                ← Build automation ✓
├── .golangci.yml           ← Linter configuration ✓
├── .gitignore              ← Git exclusions ✓
└── PROJECT-STATUS.md       ← Progress tracking (repo root) ✓
```

### Why This Structure?

**`cmd/` — Entry Points**
Go convention: each subdirectory under `cmd/` produces one executable. If we later add a CLI tool or migration runner, they'd be `cmd/graylogic-ctl/` etc.

**`internal/` — Private Code**
This is **enforced by Go** — code in `internal/` cannot be imported by external packages. This protects implementation details and allows refactoring without breaking external users.

**`pkg/` — Public Code**
Code here is meant for reuse. Protocol bridges (KNX, DALI, Modbus) will import shared types from `pkg/models/`.

**`configs/` — Configuration**
Default configuration files. Users copy and modify for their installation.

---

## File-by-File Breakdown

### 1. `go.mod` — Module Definition

```go
module github.com/nerrad567/gray-logic-core

go 1.25.6

require gopkg.in/yaml.v3 v3.0.1
```

**What it does:**
- Declares this project's **module path** (unique identifier)
- Specifies the **minimum Go version** required
- Lists **dependencies** with exact versions

**Why it matters:**
- Anyone can run `go build` and get identical results
- No "works on my machine" problems
- In 2036, this exact version of the YAML library will still work

---

### 2. `go.sum` — Dependency Checksums

```
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
```

**What it does:**
- Contains **cryptographic hashes** of all dependencies
- Ensures no one can tamper with packages after we've verified them

**Why it matters:**
- Supply chain security — even if the package server is compromised, builds will fail if hashes don't match

---

### 3. `cmd/graylogic/main.go` — Application Entry Point

This is where execution begins. Let's break it down:

#### Package Declaration
```go
package main
```
The `main` package is special — it tells Go "this is an executable, not a library."

#### Imports
```go
import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
)
```
Go groups imports in parentheses. These are all from the standard library:
- `context` — Cancellation propagation
- `fmt` — Formatted printing
- `os` — Operating system interface
- `os/signal` — Signal handling (Ctrl+C)
- `syscall` — Low-level OS calls

#### Version Variables
```go
var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)
```
These are **overwritten at build time** via `-ldflags`. This lets us embed git info into the binary without hardcoding it.

#### The main() Function
```go
func main() {
    // Print startup banner
    fmt.Printf("Gray Logic Core %s (%s) built %s\n", version, commit, date)

    // Create cancellable context
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    // Run application
    if err := run(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

**Key patterns:**

1. **`signal.NotifyContext`** — Creates a context that automatically cancels when the process receives SIGINT (Ctrl+C) or SIGTERM (kill command). This is Go's idiomatic way to handle graceful shutdown.

2. **`defer cancel()`** — The `defer` keyword schedules `cancel()` to run when `main()` exits. This ensures cleanup happens even if we return early.

3. **`run(ctx)`** — We separate business logic into `run()` so it can be tested. You can't easily test `main()`, but you can test `run()` by passing a mock context.

4. **Error handling** — Go doesn't have exceptions. Functions return errors explicitly. We check `if err != nil` and handle it.

#### The run() Function
```go
func run(ctx context.Context) error {
    fmt.Println("Starting Gray Logic Core...")

    // Wait for shutdown signal
    <-ctx.Done()

    fmt.Println("Gray Logic Core stopped.")
    return nil
}
```

**Key patterns:**

1. **`context.Context`** — This parameter threads through the entire application. When `ctx` is cancelled (user presses Ctrl+C), all goroutines can detect it and shut down cleanly.

2. **`<-ctx.Done()`** — This "receives from a channel." It blocks until the context is cancelled. Think of it as "wait here until shutdown."

3. **Returning `error`** — Even though we currently always return `nil`, the signature is correct. When we add real logic, errors will propagate up to `main()`.

---

### 4. `Makefile` — Build Automation

The Makefile provides convenient commands for development:

#### Variables Section
```makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
```

**What this does:**
- `$(shell ...)` runs a shell command and captures output
- `git describe` gets the version tag (or commit hash if no tags)
- `--dirty` adds "-dirty" suffix if there are uncommitted changes
- `LDFLAGS` injects these values into the binary at compile time

#### Build Target
```makefile
build:
    @mkdir -p $(BUILD_DIR)
    $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
```

**What this does:**
- Creates `build/` directory if it doesn't exist
- Compiles Go code with version info injected
- Outputs binary to `build/graylogic`

#### Phony Targets
```makefile
.PHONY: help build run clean test
```

**Why this matters:**
Make normally checks if a file named "build" exists. If it does and is newer than dependencies, Make skips the target. `.PHONY` tells Make "these aren't files, always run them."

---

### 5. `.golangci.yml` — Linter Configuration

This configures `golangci-lint`, a meta-linter that runs 50+ individual linters. Key sections:

#### Enabled Linters
```yaml
linters:
  enable:
    - errcheck      # Ensures errors are never ignored
    - gosec         # Security scanner
    - gocyclo       # Complexity checker
    - misspell      # Spelling checker (UK English)
```

**Why these matter for Gray Logic:**

1. **`errcheck`** — In building automation, ignoring an error could mean a light doesn't turn off or a door doesn't lock. We catch this at compile time.

2. **`gosec`** — Finds security vulnerabilities like SQL injection, command injection, hardcoded credentials. Critical for a system that controls physical infrastructure.

3. **`gocyclo`** — Flags overly complex functions. Complex code has more bugs and is harder to maintain over decades.

4. **`misspell` (UK locale)** — Ensures consistent British English throughout the codebase.

#### Linter Settings
```yaml
linters-settings:
  gosec:
    excludes:
      - G104  # We use errcheck for error checking
    severity: medium
    confidence: medium

  misspell:
    locale: UK
```

**Why exclude G104:**
Both `gosec` and `errcheck` check for ignored errors. We use `errcheck` (more comprehensive) and disable the duplicate check in `gosec`.

#### Issue Exclusions
```yaml
issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - errcheck
```

**Why relax rules in tests:**
Test code often has intentional "bad" patterns (testing error cases, using magic numbers for test data). We allow this in `*_test.go` files.

---

### 6. `internal/infrastructure/config/config.go` — Configuration Loading

This package loads YAML configuration with environment variable overrides.

#### Struct Tags
```go
type Config struct {
    Site     SiteConfig     `yaml:"site"`
    Database DatabaseConfig `yaml:"database"`
}
```

**What `yaml:"site"` means:**
This is a "struct tag" — metadata attached to fields. The YAML library reads this to know that the `Site` field maps to the `site:` key in YAML.

#### The Load Function
```go
func Load(path string) (*Config, error) {
    cfg := defaultConfig()                    // 1. Start with defaults

    data, err := os.ReadFile(path)            // 2. Read file
    if err != nil {
        return nil, fmt.Errorf("reading config file: %w", err)
    }

    if err := yaml.Unmarshal(data, cfg); err != nil {  // 3. Parse YAML
        return nil, fmt.Errorf("parsing config file: %w", err)
    }

    applyEnvOverrides(cfg)                    // 4. Environment overrides

    if err := cfg.Validate(); err != nil {    // 5. Validate
        return nil, fmt.Errorf("validating config: %w", err)
    }

    return cfg, nil
}
```

**Key patterns:**

1. **Layered configuration** — Defaults → File → Environment variables. This allows sensible defaults, customisation via file, and secrets via environment.

2. **Error wrapping with `%w`** — The `%w` verb wraps errors, preserving the original error while adding context. This enables `errors.Is()` checks later.

3. **Pointer receiver** — `cfg` is a pointer (`*Config`), so `yaml.Unmarshal` modifies the original, not a copy.

#### Environment Variable Overrides
```go
func applyEnvOverrides(cfg *Config) {
    if v := os.Getenv("GRAYLOGIC_JWT_SECRET"); v != "" {
        cfg.Security.JWT.Secret = v
    }
}
```

**Why this pattern:**
Secrets (passwords, tokens, API keys) should NEVER be in config files that might be committed to git. Environment variables are the standard way to inject secrets in production.

---

### 7. `configs/config.yaml` — Default Configuration

This is the reference configuration with all options documented:

```yaml
site:
  id: "site-001"
  name: "My Home"
  timezone: "Europe/London"
  location:
    latitude: 51.5074
    longitude: -0.1278
```

**Why location matters:**
Gray Logic calculates sunrise/sunset for astronomical scheduling (e.g., "turn on lights at sunset"). This requires geographic coordinates.

```yaml
database:
  wal_mode: true
  busy_timeout: 5
```

**What is WAL mode?**
Write-Ahead Logging (WAL) is a SQLite mode that allows concurrent reads during writes. Essential for a building automation system where state changes constantly while the UI reads data.

```yaml
mqtt:
  qos: 1
```

**What is QoS?**
Quality of Service levels in MQTT:
- QoS 0: Fire and forget (might be lost)
- QoS 1: At least once (guaranteed delivery, might duplicate)
- QoS 2: Exactly once (guaranteed, no duplicates, slower)

We use QoS 1 — reliable enough for building automation without the overhead of QoS 2.

---

## Go Concepts Explained

### Packages

Go organises code into **packages**. Each directory is one package. The package name is declared at the top of each file:

```go
package config  // All files in this directory
```

To use code from another package, you import it:

```go
import "github.com/nerrad567/gray-logic-core/internal/infrastructure/config"

cfg, err := config.Load("config.yaml")
```

### Exported vs Unexported

Go uses **capitalisation** for visibility:
- `Config` (capital C) — Exported, visible outside package
- `defaultConfig` (lowercase d) — Unexported, private to package

This is why our structs use capital letters — they need to be visible to the YAML library and other packages.

### Error Handling

Go doesn't have exceptions. Functions that can fail return an error:

```go
func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config file: %w", err)
    }
    // Continue if no error...
}
```

This is verbose but explicit — you always know which operations can fail and how errors propagate.

### Context

`context.Context` is Go's way of managing:
- **Cancellation** — "Stop what you're doing"
- **Deadlines** — "Stop after 30 seconds"
- **Values** — "Here's some request-scoped data"

It flows through your application as the first parameter:

```go
func DoSomething(ctx context.Context, arg string) error {
    select {
    case <-ctx.Done():
        return ctx.Err()  // Cancelled or timed out
    default:
        // Do work...
    }
}
```

### Defer

`defer` schedules a function to run when the current function returns:

```go
func DoSomething() {
    f, _ := os.Open("file.txt")
    defer f.Close()  // Runs when DoSomething returns

    // Use file...
}  // f.Close() called here
```

This ensures cleanup happens even if the function returns early due to an error.

---

## Common Commands

```bash
# Navigate to the project
cd code/core

# See all available commands
make help

# Build the application
make build

# Build and run
make run

# Run all tests
make test

# Run linter
make lint

# Format code
make fmt

# All quality checks (format, vet, lint, test)
make check

# Clean build artifacts
make clean

# Full release build
make all
```

### Integration Tests

Some packages have integration tests that require external services (MQTT broker, InfluxDB).

```bash
# Start dev services first
docker compose -f docker-compose.dev.yml up -d

# Run integration tests (requires running broker)
go test -tags=integration -v ./internal/infrastructure/mqtt/...

# Run all tests including integration
go test -tags=integration -v ./...

# Skip integration tests (default behaviour)
go test -v ./...
```

Integration test files use the `//go:build integration` build tag and are skipped by default.

---

## Next Steps

The foundation is complete. Next we'll implement:

1. **SQLite Database Package**
   - Connection management
   - Migration runner
   - Initial schema (sites, areas, rooms, devices)

2. **MQTT Client Package**
   - Auto-reconnect with exponential backoff
   - QoS handling
   - Topic structure per specification

3. **Docker Compose**
   - Mosquitto MQTT broker
   - InfluxDB for time-series data

4. **Wire Up Configuration**
   - Load config in main.go
   - Pass to components

---

## Further Reading

- [Effective Go](https://go.dev/doc/effective_go) — Official Go style guide
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) — Common review feedback
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout) — Directory conventions
- Gray Logic docs: `docs/development/CODING-STANDARDS.md`
