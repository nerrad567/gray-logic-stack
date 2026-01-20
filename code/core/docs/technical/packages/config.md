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
| `Config` | Root configuration structure |
| `SiteConfig` | Site identity and location |
| `DatabaseConfig` | SQLite settings |
| `MQTTConfig` | MQTT broker connection |
| `APIConfig` | HTTP server settings |
| `InfluxDBConfig` | Time-series database |
| `LoggingConfig` | Logging configuration |
| `SecurityConfig` | Auth and JWT settings |

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
    idle: 120
  cors:                  # CORSConfig
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["Content-Type", "Authorization"]

websocket:               # WebSocketConfig
  path: "/ws"
  max_message_size: 65536
  ping_interval: 30
  pong_timeout: 10

influxdb:                # InfluxDBConfig
  url: "http://localhost:8086"
  token: ""              # Set via GRAYLOGIC_INFLUXDB_TOKEN
  org: "graylogic"
  bucket: "metrics"

logging:                 # LoggingConfig
  level: "info"
  format: "json"
  output: "stdout"

security:                # SecurityConfig
  jwt_secret: ""         # Set via GRAYLOGIC_JWT_SECRET
  token_expiry: 24
  bcrypt_cost: 12
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
| `api` | (Future) Uses `cfg.API` for server |

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
