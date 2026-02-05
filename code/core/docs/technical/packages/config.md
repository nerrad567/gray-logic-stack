# Config Package Design

> `internal/infrastructure/config/` — YAML configuration with environment variable overrides

## Purpose

Provides configuration loading for Gray Logic Core with:
- YAML file parsing with sensible defaults
- Environment variable overrides for secrets
- Validation of required fields and value ranges
- Type-safe configuration structs

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      config.Load()                           │
│  ┌─────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  YAML File  │ → │ Env Overrides  │ → │   Validation    │  │
│  └─────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌──────────────────────────────┐
              │        config.Config          │
              │  (type-safe struct tree)      │
              └──────────────────────────────┘
```

### Key Types

| Type | Purpose |
|------|---------|
| `Config` | Root configuration structure (includes `DevMode` flag and `PanelDir` for dev) |
| `SiteConfig` | Site identity and location |
| `DatabaseConfig` | SQLite settings |
| `MQTTConfig` | MQTT broker connection |
| `APIConfig` | HTTP server settings |
| `VictoriaMetricsConfig` | Time-series database |
| `LoggingConfig` | Logging configuration |
| `SecurityConfig` | Auth and JWT settings |
| `ProtocolsConfig` | Protocol bridge settings |
| `KNXConfig` | KNX bridge and knxd settings |
| `KNXDConfig` | knxd daemon management |
| `KNXDBackendConfig` | KNX bus connection (USB/IPT/IP) |

### External Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing |

---

## How It Works

### Initialization

```go
cfg, err := config.Load("configs/config.yaml")
if err != nil {
    log.Fatal(err)
}

// Access configuration
fmt.Println(cfg.Site.Name)
fmt.Println(cfg.Database.Path)
```

**Load() performs:**
1. Sets default values for all fields
2. Reads and parses YAML file
3. Applies environment variable overrides
4. Validates required fields and ranges
5. Returns complete, validated config

### Environment Overrides

Secrets and deployment-specific values can be overridden:

```bash
export GRAYLOGIC_MQTT_USERNAME=admin
export GRAYLOGIC_MQTT_PASSWORD=secret
export GRAYLOGIC_JWT_SECRET=production-secret-key
export GRAYLOGIC_MQTT_HOST=mosquitto       # Override for Docker networking
export GRAYLOGIC_PANEL_DIR=/path/to/flutter/build/web  # Dev only: filesystem panel serving
```

**Pattern:** `GRAYLOGIC_<SECTION>_<FIELD>` (uppercase, underscores)

### Validation

Validation ensures:
- Required fields are non-empty
- Ports are in valid range (1-65535)
- Timeouts are positive
- Paths are specified

```go
if cfg.Site.ID == "" {
    return nil, fmt.Errorf("site.id is required")
}
if cfg.API.Port < 1 || cfg.API.Port > 65535 {
    return nil, fmt.Errorf("api.port must be 1-65535")
}
```

---

## Configuration Hierarchy

```yaml
dev_mode: false           # Enable dev-mode (simulates bridge confirmations without hardware)
panel_dir: ""             # Dev only: serve Flutter panel from filesystem (set via GRAYLOGIC_PANEL_DIR)

site:                    # SiteConfig
  id: "site-001"
  name: "My Home"
  timezone: "Europe/London"
  location:              # LocationConfig
    latitude: 51.5074
    longitude: -0.1278

database:                # DatabaseConfig
  path: "./data/graylogic.db"
  wal_mode: true
  busy_timeout: 5

mqtt:                    # MQTTConfig
  broker:                # MQTTBrokerConfig
    host: "localhost"
    port: 1883
    tls: false
    client_id: "graylogic-core"
  auth:                  # MQTTAuthConfig
    username: ""
    password: ""
  qos: 1
  reconnect:             # MQTTReconnectConfig
    initial_delay: 1
    max_delay: 60
    max_attempts: 0

api:                     # APIConfig
  host: "0.0.0.0"
  port: 8080
  tls:                   # TLSConfig
    enabled: false
    cert_file: ""
    key_file: ""
  timeouts:              # APITimeoutConfig
    read: 30
    write: 30
    idle: 60
  cors:                  # CORSConfig
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["Content-Type", "Authorization"]

websocket:               # WebSocketConfig
  path: "/ws"
  max_message_size: 8192
  ping_interval: 30
  pong_timeout: 10

tsdb:                # VictoriaMetricsConfig
  enabled: false
  url: "http://localhost:8428"
  token: ""              # Set via GRAYLOGIC_TSDB_URL
  org: "graylogic"
  bucket: "metrics"
  batch_size: 100        # Points per batch (max 100,000)
  flush_interval: 10     # Seconds between flushes (max 3,600)

logging:                 # LoggingConfig
  level: "info"
  format: "json"
  output: "stdout"
  file:                  # FileLoggingConfig (optional)
    path: ""             # File path for log output
    max_size: 100        # MB before rotation
    max_backups: 3       # Rotated files to keep
    max_age: 28          # Days before deletion
    compress: false      # Gzip rotated files

security:                # SecurityConfig
  jwt:                   # JWTConfig
    secret: ""           # Set via GRAYLOGIC_JWT_SECRET
    access_token_ttl: 24       # Hours
    refresh_token_ttl: 168     # Hours (7 days)
  api_keys:              # APIKeyConfig
    enabled: false
  rate_limit:            # RateLimitConfig
    enabled: false
    requests_per_minute: 60

protocols:               # ProtocolsConfig
  knx:                   # KNXConfig
    enabled: true
    config_file: "configs/knx-bridge.yaml"
    knxd_host: "localhost"
    knxd_port: 6720
    knxd:                # KNXDConfig
      managed: true      # Gray Logic manages knxd lifecycle
      binary: "/usr/bin/knxd"
      physical_address: "0.0.1"
      client_addresses: "0.0.2:8"
      backend:           # KNXDBackendConfig
        type: "usb"      # "usb", "ipt", or "ip"
        # USB settings
        usb_vendor_id: "0e77"
        usb_product_id: "0104"
        usb_reset_on_retry: true
        usb_reset_on_bus_failure: true
        # IPT settings (when type: "ipt")
        # host: "192.168.1.100"
        # port: 3671
      restart_on_failure: true
      restart_delay_seconds: 5
      max_restart_attempts: 10
      group_cache: true    # Enable knxd group comm cache (-c flag)
      health_check_interval: 30s
      health_check_device_address: ""  # Optional: "1/7/0"
      health_check_device_timeout: 3s
```

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **YAML format** | Human-readable, supports comments | TOML (less common), JSON (no comments) |
| **Env overrides** | Secrets shouldn't be in files | Vault (complexity), encrypted YAML |
| **Flat env names** | Simple, predictable | Nested (GRAYLOGIC_MQTT_BROKER_HOST harder) |
| **Validation at load** | Fail fast, clear errors | Runtime validation (delayed failures) |
| **Strongly typed** | Compile-time safety, IDE support | map[string]interface{} (no safety) |

---

## Interactions

### Dependencies

None — config is the root of the dependency tree.

### Dependents

| Package | Purpose |
|---------|---------|
| `cmd/graylogic/main.go` | Loads configuration at startup |
| `database` | Uses `cfg.Database` for connection |
| `mqtt` | Uses `cfg.MQTT` for broker connection |
| `tsdb` | Uses `cfg.TSDB` for time-series |
| `internal/knxd` | Uses `cfg.Protocols.KNX.KNXD` for daemon management |
| `internal/bridges/knx` | Uses `cfg.Protocols.KNX` for bridge settings |
| `api` | Uses `cfg.API`, `cfg.Security`, `cfg.PanelDir` for server |

---

## USB Configuration (KNX)

For USB KNX interfaces (Weinzierl, etc.), the config supports automatic device reset:

```yaml
protocols:
  knx:
    knxd:
      backend:
        type: "usb"
        usb_vendor_id: "0e77"      # Hex without 0x prefix
        usb_product_id: "0104"     # Hex without 0x prefix
        usb_reset_on_retry: true   # Reset before restart attempts
        usb_reset_on_bus_failure: true  # Reset when health checks fail
```

**USB Reset Features:**

| Setting | Purpose |
|---------|---------|
| `usb_vendor_id` | Vendor ID for USB device identification (e.g., "0e77" for Weinzierl) |
| `usb_product_id` | Product ID for USB device identification (e.g., "0104") |
| `usb_reset_on_retry` | Reset USB device before each restart attempt (recovers from LIBUSB_ERROR_BUSY) |
| `usb_reset_on_bus_failure` | Proactively reset USB when Layer 3/4 health checks fail |

**Requirements:**
- `usbreset` utility installed (standard on most Linux systems)
- udev rule for write access:
  ```
  SUBSYSTEM=="usb", ATTR{idVendor}=="0e77", ATTR{idProduct}=="0104", MODE="0666"
  ```

---

## Error Handling

Errors include file path and field context:

```go
return nil, fmt.Errorf("loading config from %s: %w", path, err)
return nil, fmt.Errorf("parsing YAML: %w", err)
return nil, fmt.Errorf("validation: site.id is required")
```

---

## Thread Safety

Configuration is **immutable after Load()**. No synchronization needed.

---

## Testing

```bash
make test PKG=./internal/infrastructure/config/...
```

Test scenarios:
- Valid YAML parsing
- Environment variable overrides
- Validation failures
- Missing file handling
- Default values

---

## Related Documents

- [doc.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/config/doc.go) — Package-level godoc
- [configs/config.yaml](file:///home/graylogic-dev/gray-logic-stack/code/core/configs/config.yaml) — Default configuration file
- [knxd Manager](./knxd-manager.md) — knxd daemon management (uses `protocols.knx.knxd`)
- [KNX Bridge](./knx-bridge.md) — KNX protocol bridge (uses `protocols.knx`)
- [Process Manager](./process-manager.md) — Generic subprocess management
