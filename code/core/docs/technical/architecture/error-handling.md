# Error Handling Patterns

> Consistent error handling across Gray Logic Core

## Principles

1. **Wrap with context** — Always add context when propagating errors
2. **Domain errors** — Define package-specific error types
3. **Fail fast** — Validate early, return immediately on error
4. **No silent failures** — Every error must be logged or returned

---

## Error Wrapping

Use `%w` verb to maintain error chain for `errors.Is()` and `errors.As()`:

```go
// Good: Context + wrapped error
if err := db.Ping(); err != nil {
    return fmt.Errorf("verifying database connection: %w", err)
}

// Caller can check underlying error
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

---

## Domain Error Types

Each package defines sentinel errors in `errors.go`:

### database package

```go
// No custom errors yet — uses wrapped standard library errors
```

### mqtt package

```go
var (
    ErrNotConnected     = errors.New("mqtt: not connected")
    ErrConnectionFailed = errors.New("mqtt: connection failed")
    ErrPublishFailed    = errors.New("mqtt: publish failed")
    ErrSubscribeFailed  = errors.New("mqtt: subscribe failed")
)
```

**Usage:**
```go
// Creating
return fmt.Errorf("%w: timeout after %v", ErrConnectionFailed, timeout)

// Checking
if errors.Is(err, mqtt.ErrNotConnected) {
    // Attempt reconnect
}
```

---

## Error Handling Patterns

### Constructor Functions

Return error as second value:

```go
func Open(cfg Config) (*DB, error) {
    // Validate
    if cfg.Path == "" {
        return nil, fmt.Errorf("database path is required")
    }
    
    // Attempt operation
    db, err := sql.Open("sqlite3", connStr)
    if err != nil {
        return nil, fmt.Errorf("opening database: %w", err)
    }
    
    return &DB{DB: db}, nil
}
```

### Method Errors

Wrap with method context:

```go
func (db *DB) HealthCheck(ctx context.Context) error {
    var result int
    err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
    if err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }
    return nil
}
```

### Cleanup on Error

Use defer for cleanup, ignore cleanup errors on error path:

```go
func Open(cfg Config) (*DB, error) {
    db, err := sql.Open("sqlite3", connStr)
    if err != nil {
        return nil, fmt.Errorf("opening database: %w", err)
    }
    
    // Cleanup if later steps fail
    if err := db.Ping(); err != nil {
        db.Close() //nolint:errcheck // Best effort on error path
        return nil, fmt.Errorf("verifying connection: %w", err)
    }
    
    return &DB{DB: db}, nil
}
```

---

## Context Propagation

Always accept and respect `context.Context`:

```go
func (db *DB) HealthCheck(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return fmt.Errorf("health check: %w", ctx.Err())
    default:
    }
    
    // Proceed with check
    return db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
}
```

---

## Logging vs Returning

| Situation | Action |
|-----------|--------|
| Recoverable, caller can retry | Return error |
| Informational, doesn't affect operation | Log and continue |
| Fatal, application should exit | Log and return (let main exit) |
| Handler callback error | Log (can't return from callback) |

```go
// Handler errors are logged, not returned
func (c *Client) wrapHandler(handler MessageHandler) pahomqtt.MessageHandler {
    return func(_ pahomqtt.Client, msg pahomqtt.Message) {
        if err := handler(msg.Topic(), msg.Payload()); err != nil {
            // TODO: Log when logging package available
            _ = err
        }
    }
}
```

---

## Anti-Patterns

### Don't: Lose error context

```go
// Bad
return errors.New("failed")

// Good
return fmt.Errorf("connecting to broker: %w", err)
```

### Don't: Check error type by string

```go
// Bad
if err.Error() == "not connected" { ... }

// Good
if errors.Is(err, ErrNotConnected) { ... }
```

### Don't: Return raw errors from external libraries

```go
// Bad
return err  // Raw paho error

// Good
return fmt.Errorf("mqtt publish failed: %w", err)
```

---

## Related Documents

- [data-flow.md](data-flow.md) — System data flow
- [../packages/mqtt.md](../packages/mqtt.md) — MQTT error types
