# InfluxDB Package Design

> `internal/infrastructure/influxdb/` — Time-series storage for PHM and energy telemetry

## Purpose

Provides InfluxDB v2 connectivity for Gray Logic Core with:
- Non-blocking batched writes for high-frequency data
- Connection management with ping-based health checks
- Domain-specific helpers for device, energy, and PHM metrics
- Graceful shutdown with pending write flush

**Why InfluxDB?** See [ADR-004: InfluxDB for Time-Series](../../../../../docs/architecture/decisions/004-influxdb-time-series.md) *(planned)*

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     influxdb.Client                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │ influxdb2.Client│  │   WriteAPI      │  │ HealthCheck │  │
│  │   (wrapped)     │  │  (non-blocking) │  │   (Ping)    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ HTTP
              ┌──────────────────────────────┐
              │       InfluxDB Server        │
              │    (127.0.0.1:8086 / :8087)  │
              └──────────────────────────────┘
                              │
                              ▼
              ┌──────────────────────────────┐
              │         metrics bucket       │
              │  device_metrics, energy, phm │
              └──────────────────────────────┘
```

### Key Types

| Type | File | Purpose |
|------|------|---------|
| `Client` | [client.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/influxdb/client.go#L28-L39) | Wraps InfluxDB client with lifecycle and health |
| `ErrNotConnected` | [errors.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/influxdb/errors.go) | Sentinel error types |

### External Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/influxdata/influxdb-client-go/v2` | v2.14.0 | Official InfluxDB v2 client |

---

## How It Works

### Initialization

```go
// 1. Create config (typically from config.yaml)
cfg := config.InfluxDBConfig{
    Enabled:       true,
    URL:           "http://localhost:8086",
    Token:         "your-token",
    Org:           "graylogic",
    Bucket:        "metrics",
    BatchSize:     100,
    FlushInterval: 10,
}

// 2. Connect to InfluxDB
client, err := influxdb.Connect(cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

**Connect() performs:**
1. Validates config (disabled returns ErrDisabled)
2. Creates client with token authentication
3. Verifies connectivity via ping
4. Creates non-blocking write API with batching

### Core Operations

**Device metrics:**
```go
client.WriteDeviceMetric("light-living-01", "power_watts", 23.5)
client.WriteDeviceMetric("thermostat-01", "temperature_c", 21.0)
```

**Energy tracking:**
```go
client.WriteEnergyMetric("hvac-main", 450.0, 12.34) // power_watts, energy_kwh
```

**PHM metrics:**
```go
client.WritePHMMetric("compressor-01", "runtime_hours", 1234.5)
```

**Custom points:**
```go
client.WritePoint("system_stats",
    map[string]string{"host": "core-01"},
    map[string]interface{}{"cpu_percent": 45.2})
```

### Shutdown

```go
if err := client.Close(); err != nil {
    log.Printf("Error closing InfluxDB: %v", err)
}
```

**Close() performs:**
1. Flushes all pending batched writes
2. Closes the underlying client

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **Non-blocking WriteAPI** | High-frequency sensor data shouldn't block Core | Blocking writes (too slow) |
| **Batching** | Reduce network overhead; configured via config.yaml | Per-point writes (inefficient) |
| **Error callback** | Async writes can't return errors synchronously | Ignore errors (data loss risk) |
| **Ping health check** | Simple, built-in, no custom queries | Query-based (more complex) |

---

## Interactions

### Dependencies (what this package uses)

| Package | Purpose |
|---------|---------|
| `config` | Reads InfluxDB configuration |

### Dependents (what uses this package)

| Package | Purpose |
|---------|---------|
| `cmd/graylogic/main.go` | (Future) Creates and manages lifecycle |
| `device/state/` | (Future) Records device state changes |
| `phm/` | (Future) Records health monitoring metrics |

---

## Error Handling

Package defines domain-specific errors in `errors.go`:

```go
var (
    ErrNotConnected     = errors.New("influxdb: not connected")
    ErrConnectionFailed = errors.New("influxdb: connection failed")
    ErrWriteFailed      = errors.New("influxdb: write failed")
    ErrDisabled         = errors.New("influxdb: disabled in configuration")
)
```

**Async write errors:**
```go
client.SetOnError(func(err error) {
    log.Printf("InfluxDB write error: %v", err)
})
```

---

## Thread Safety

| Operation | Thread-Safe | Notes |
|-----------|-------------|-------|
| `Connect()` | N/A | Called once at startup |
| `Close()` | Yes | Flushes and disconnects |
| `HealthCheck()` | Yes | Ping-based check |
| `WriteDeviceMetric()` | Yes | Non-blocking WriteAPI |
| `WritePoint()` | Yes | Non-blocking WriteAPI |
| `Flush()` | Yes | Forces batch flush |
| `SetOnError()` | Yes | Protected by mutex |

---

## Configuration

From `configs/config.yaml`:

```yaml
influxdb:
  # Enable InfluxDB integration
  enabled: false

  # Connection settings
  url: "http://localhost:8086"
  token: ""  # Set via GRAYLOGIC_INFLUXDB_TOKEN
  org: "graylogic"
  bucket: "metrics"

  # Write batch settings
  batch_size: 100       # Points per batch
  flush_interval: 10    # Seconds
```

---

## Testing

**Integration tests:** Require running InfluxDB (via Docker Compose).

```bash
# Start Docker services
cd /home/darren/Development/Projects/gray-logic-stack/code/core
docker compose up -d

# Run tests
make test PKG=./internal/infrastructure/influxdb/...
```

**Test coverage:** 11 tests covering connection, health check, and write operations.

---

## Known Limitations

1. **No query API** — Only writes implemented; queries deferred to PHM layer
2. **No retention policy management** — Uses default bucket retention
3. **Single bucket** — All metrics go to configured bucket
4. **No TLS client certs** — Token-only authentication

---

## Related Documents

- [doc.go](file:///home/darren/Development/Projects/gray-logic-stack/code/core/internal/infrastructure/influxdb/doc.go) — Package-level godoc
- [docker-compose.yml](file:///home/darren/Development/Projects/gray-logic-stack/code/core/docker-compose.yml) — InfluxDB container config
