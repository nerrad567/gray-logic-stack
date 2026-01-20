# Logging Package Design

> `internal/infrastructure/logging/` — Structured logging using Go's slog

## Purpose

Provides structured logging for Gray Logic Core with:
- JSON output for production (machine-parsable)
- Text output for development (human-readable)
- Default fields on all log entries (service, version)
- Level-based filtering (debug, info, warn, error)
- Thread-safe concurrent use

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    logging.New()                             │
│  ┌─────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   Config    │ → │  slog.Handler   │ → │ Default Fields  │  │
│  │  (format,   │   │  (JSON/Text)    │   │ (service, ver)  │  │
│  │   level)    │   └─────────────────┘   └─────────────────┘  │
│  └─────────────┘                                              │
└─────────────────────────────────────────────────────────────┘
                               │
                               ▼
               ┌──────────────────────────────┐
               │       logging.Logger         │
               │   (wraps slog.Logger)        │
               └──────────────────────────────┘
```

### Key Types

| Type | Purpose |
|------|---------|
| `Logger` | Wrapper around slog.Logger with Gray Logic defaults |

### External Dependencies

None — uses Go standard library `log/slog` (Go 1.21+).

---

## How It Works

### Initialization

```go
logger := logging.New(cfg.Logging, version)

// Use structured logging
logger.Info("starting service", "port", 8080)
logger.Error("connection failed", "error", err)
```

**New() performs:**
1. Selects output destination (stdout/stderr)
2. Parses log level (debug/info/warn/error)
3. Creates handler (JSON or Text format)
4. Adds default fields (service=graylogic, version)

### Creating Component Loggers

```go
// Create child logger with additional context
mqttLogger := logger.With("component", "mqtt")
mqttLogger.Info("connected", "broker", "localhost:1883")
// Output includes: component=mqtt, service=graylogic, version=...
```

### Default Logger

For early startup before config is loaded:

```go
logger := logging.Default()
// Uses info level, JSON format, stdout
```

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **slog (stdlib)** | No dependencies, standard Go | zap (external), zerolog (external) |
| **JSON default** | Machine-parsable for production | Text (harder to parse) |
| **Immutable config** | No runtime log level changes | Dynamic (complexity) |
| **Default fields** | Consistent context on all logs | Per-call fields (repetitive) |

---

## Interactions

### Dependencies (what this package uses)

| Package | Purpose |
|---------|---------|
| `config` | Reads LoggingConfig |

### Dependents (what uses this package)

| Package | Purpose |
|---------|---------|
| `cmd/graylogic/main.go` | Creates logger at startup |
| `mqtt` | Optional logger via SetLogger() |

---

## Error Handling

No errors returned — invalid config values use sensible defaults:
- Unknown level → info
- Unknown format → json
- Unknown output → stdout

---

## Thread Safety

All Logger methods are safe for concurrent use (inherited from slog).

---

## Configuration

From `configs/config.yaml`:

```yaml
logging:
  level: "info"     # debug, info, warn, error
  format: "json"    # json, text
  output: "stdout"  # stdout, stderr
```

---

## Testing

```bash
make test PKG=./internal/infrastructure/logging/...
```

**Coverage:** 100%

Test scenarios:
- JSON and text output formats
- All log levels
- Default logger fallback
- With() chained loggers

---

## Security

**Never log secrets.** Use field redaction:

```go
// Bad
logger.Info("API call", "token", apiToken)

// Good
logger.Info("API call", "token_prefix", apiToken[:8]+"...")
```

See `docs/development/CODING-STANDARDS.md` for logging guidelines.

---

## Related Documents

- [doc.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/logging/doc.go) — Package-level godoc
- [CODING-STANDARDS.md](file:///home/graylogic-dev/gray-logic-stack/docs/development/CODING-STANDARDS.md) — Logging guidelines
