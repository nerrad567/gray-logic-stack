# Process Manager Package Design

> `internal/process/` — Generic subprocess lifecycle management

## Purpose

Provides generic subprocess management for long-running child processes like protocol daemons (knxd, DALI gateways, etc.) that Gray Logic depends on. Features include:
- Start/stop subprocess with graceful shutdown (SIGTERM → SIGKILL)
- Automatic restart on failure with exponential backoff
- Health monitoring via configurable watchdog
- Log capture from subprocess stdout/stderr
- Context-based cancellation for clean shutdown

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Process Manager                                  │
│                                                                          │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐   │
│  │     Manager      │───▶│    subprocess    │    │   Log Capture    │   │
│  │   (manager.go)   │    │   (exec.Cmd)     │    │   (goroutines)   │   │
│  │                  │    │                  │    │                  │   │
│  │ • Start/Stop     │    │ • Binary + args  │    │ • stdout pipe    │   │
│  │ • Monitor loop   │    │ • Process group  │    │ • stderr pipe    │   │
│  │ • Restart logic  │    │ • Environment    │    │                  │   │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘   │
│           │                                                              │
│           ▼                                                              │
│  ┌──────────────────┐    ┌──────────────────┐                           │
│  │     Config       │    │    Watchdog      │                           │
│  │                  │    │  (health check)  │                           │
│  │ • Binary path    │    │                  │                           │
│  │ • Restart policy │    │ • Interval check │                           │
│  │ • Callbacks      │    │ • Kill on fail   │                           │
│  └──────────────────┘    └──────────────────┘                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### Key Types

| Type | Purpose |
|------|---------|
| `Manager` | Main process lifecycle controller |
| `Config` | Process configuration and callbacks |
| `Status` | Process state: stopped, starting, running, failed |
| `Stats` | Runtime statistics |
| `RecoverableError` | Interface for errors that indicate if restart helps |

### External Dependencies

None — uses only Go standard library (`os/exec`, `syscall`).

---

## How It Works

### Basic Usage

```go
mgr := process.NewManager(process.Config{
    Name:               "knxd",
    Binary:             "/usr/bin/knxd",
    Args:               []string{"-e", "0.0.1", "-b", "usb:"},
    RestartOnFailure:   true,
    RestartDelay:       5 * time.Second,
    MaxRestartAttempts: 10,
})
mgr.SetLogger(logger)

if err := mgr.Start(ctx); err != nil {
    log.Fatal(err)
}
defer mgr.Stop()

// Process is now running
fmt.Printf("PID: %d, Uptime: %v\n", mgr.PID(), mgr.Uptime())
```

### With Health Check (Watchdog)

```go
mgr := process.NewManager(process.Config{
    Name:   "knxd",
    Binary: "/usr/bin/knxd",
    Args:   []string{"-e", "0.0.1", "-b", "usb:"},

    // Health monitoring
    HealthCheckInterval: 30 * time.Second,
    HealthCheckFunc: func(ctx context.Context) error {
        // Custom health check logic
        // Return error to trigger restart
        return checkKNXDResponding()
    },
})
```

The watchdog will:
1. Run health check at configured interval
2. After 3 consecutive failures, kill the process
3. Trigger automatic restart (if configured)

### With Lifecycle Callbacks

```go
mgr := process.NewManager(process.Config{
    Name:   "knxd",
    Binary: "/usr/bin/knxd",

    OnStart: func() {
        log.Info("knxd started", "pid", mgr.PID())
    },
    OnStop: func(err error) {
        if err != nil {
            log.Warn("knxd crashed", "error", err)
        }
    },
    OnRestart: func(attempt int) {
        log.Info("restarting knxd", "attempt", attempt)
        // Opportunity to reset hardware, clear state, etc.
    },
})
```

---

## Configuration

```go
type Config struct {
    // Identity
    Name   string   // Human-readable name for logs
    Binary string   // Path to executable
    Args   []string // Command-line arguments
    Env    []string // Additional environment variables
    WorkDir string  // Working directory

    // Restart Policy
    RestartOnFailure   bool          // Auto-restart on crash
    RestartDelay       time.Duration // Initial delay before restart
    MaxRestartDelay    time.Duration // Maximum delay (exponential backoff cap)
    MaxRestartAttempts int           // 0 = unlimited
    StableThreshold    time.Duration // Reset backoff after this uptime

    // Shutdown
    GracefulTimeout time.Duration // Wait for SIGTERM before SIGKILL

    // Health Monitoring
    HealthCheckFunc     func(ctx context.Context) error
    HealthCheckInterval time.Duration

    // Callbacks
    OnStart   func()
    OnStop    func(err error)
    OnRestart func(attempt int)
}
```

### Defaults

```go
cfg := process.DefaultConfig("knxd", "/usr/bin/knxd", args)
// RestartOnFailure:    true
// RestartDelay:        5s
// MaxRestartDelay:     5m
// MaxRestartAttempts:  10
// StableThreshold:     2m
// GracefulTimeout:     10s
// HealthCheckInterval: 30s
```

---

## Restart Behaviour

### Exponential Backoff

```
Attempt 1: 5s delay
Attempt 2: 10s delay
Attempt 3: 20s delay
Attempt 4: 40s delay
...
Attempt N: capped at MaxRestartDelay (5m default)
```

### Stability Reset

If the process runs longer than `StableThreshold` before failing, the restart count and backoff are reset:

```
Process starts → runs for 3 hours → crashes
→ Restart count reset to 0 (was stable)
→ First restart uses initial delay (5s)
```

### RecoverableError Interface

For smart restart decisions, errors can implement `RecoverableError`:

```go
type RecoverableError interface {
    error
    IsRecoverable() bool
}

// Example: Hardware missing - don't restart
type HardwareError struct {
    Err error
}

func (e *HardwareError) Error() string { return e.Err.Error() }
func (e *HardwareError) IsRecoverable() bool { return false }
```

When a non-recoverable error is detected:
- The manager logs it but continues monitoring
- Restart is NOT triggered (would fail again)
- If hardware comes back, the next health check passes

---

## Process Group Management

The manager creates a process group for clean shutdown:

```go
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
```

Benefits:
- `SIGTERM` signals entire process group (including children)
- No orphaned child processes
- Clean shutdown of daemons that spawn workers

Shutdown sequence:
1. `SIGTERM` to process group (`kill -pid`)
2. Wait for `GracefulTimeout`
3. `SIGKILL` if not exited

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **Process groups** | Cleanly signal all children | Just signal main process (orphans) |
| **Exponential backoff** | Prevents tight restart loops | Fixed delay (slow recovery) |
| **Stability threshold** | Prevents backoff accumulation over time | Never reset (punishes stable processes) |
| **RecoverableError** | Hardware issues shouldn't trigger restarts | Always restart (wastes resources) |
| **Log capture via pipes** | Non-blocking, captures all output | File redirection (needs cleanup) |
| **Done channel pattern** | Clean synchronization for Stop() | Mutex polling (complex) |

---

## Interactions

### Dependencies

None — uses only Go standard library.

### Dependents

| Package | Purpose |
|---------|---------|
| `internal/knxd` | Manages knxd daemon |
| Future bridges | DALI gateways, audio matrices, etc. |

---

## Error Handling

The `LastError()` method returns the most recent failure:

```go
if !mgr.IsRunning() {
    if err := mgr.LastError(); err != nil {
        log.Error("process failed", "error", err)
    }
}
```

Health check failures include context:

```go
// "killed due to health check failure after 3 consecutive failures"
```

---

## Thread Safety

The Manager is safe for concurrent use:

| Operation | Thread-safe |
|-----------|-------------|
| `Start()` | Yes (prevents double-start) |
| `Stop()` | Yes (idempotent) |
| `Status()`, `IsRunning()` | Yes |
| `PID()`, `Uptime()` | Yes |
| `Stats()` | Yes |

---

## Testing

```bash
cd code/core
go test -v ./internal/process/...

# With race detector
go test -race ./internal/process/...
```

Test scenarios:
- Start/stop lifecycle
- Automatic restart on failure
- Exponential backoff calculation
- Stability threshold reset
- Graceful vs forced shutdown
- Health check failure handling

---

## Statistics

```go
stats := mgr.Stats()
fmt.Printf("Name: %s\n", stats.Name)
fmt.Printf("Status: %s\n", stats.Status)
fmt.Printf("PID: %d\n", stats.PID)
fmt.Printf("Uptime: %v\n", stats.Uptime)
fmt.Printf("Restart count: %d\n", stats.RestartCount)
fmt.Printf("Last error: %s\n", stats.LastError)
```

---

## Related Documents

- [knxd Manager](./knxd-manager.md) — Uses process manager for knxd
- [Config Package](./config.md) — Process configuration in YAML
