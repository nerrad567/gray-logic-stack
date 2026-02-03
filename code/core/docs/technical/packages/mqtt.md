# MQTT Package Design

> `internal/infrastructure/mqtt/` — MQTT client with auto-reconnect and subscription management

## Purpose

Provides MQTT connectivity for Gray Logic Core with:
- Connection to Mosquitto broker with auto-reconnect
- Message publishing with QoS guarantees
- Topic subscriptions with automatic restoration on reconnect
- Last Will and Testament (LWT) for offline detection
- Connection health monitoring

**Why MQTT?** See [ADR-002: MQTT Internal Bus](../../../../../docs/architecture/decisions/002-mqtt-internal-bus.md)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       mqtt.Client                            │
│  ┌─────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ paho.Client │  │  Subscriptions  │  │  Connection     │  │
│  │  (wrapped)  │  │  map[topic]sub  │  │  State (mutex)  │  │
│  └─────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ TCP/TLS
              ┌──────────────────────────────┐
              │      Mosquitto Broker        │
              │   (127.0.0.1:1883 / :8883)   │
              └──────────────────────────────┘
                              │
                              ▼
              ┌──────────────────────────────┐
              │      Protocol Bridges        │
              │   (KNX, DALI, Modbus, etc.)  │
              └──────────────────────────────┘
```

### Key Types

| Type | File | Purpose |
|------|------|---------|
| `Client` | [client.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/mqtt/client.go#L21-L37) | Main client wrapping paho with reconnect handling |
| `MessageHandler` | [client.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/mqtt/client.go#L46-L57) | Callback for received messages |
| `Topics` | [topics.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/mqtt/topics.go) | Topic builder utilities |
| `subscription` | [client.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/mqtt/client.go#L40-L44) | Internal tracking for reconnect restoration |

### External Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/eclipse/paho.mqtt.golang` | v1.4.3 | Eclipse MQTT client (10+ years stable) |

---

## How It Works

### Initialization

```go
// 1. Connect using config (typically from config.yaml)
client, err := mqtt.Connect(cfg.MQTT)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 2. Subscribe to topics
err = client.Subscribe(mqtt.Topics{}.AllStates(), 1,
    func(topic string, payload []byte) error {
        log.Printf("Received: %s = %s", topic, payload)
        return nil
    })
```

**Connect() performs:**
1. Builds client options (broker URL, auth, TLS)
2. Configures LWT for crash detection
3. Sets up auto-reconnect with exponential backoff
4. Attempts initial connection with timeout
5. Publishes online status to `graylogic/system/status`

### Core Operations

**Publishing:**
```go
topic := mqtt.Topics{}.Command("knx", "light-living")
err := client.Publish(topic, []byte(`{"on":true}`), 1, false)
```

**Subscribing:**
```go
// Subscribe is tracked for automatic restoration on reconnect
err := client.Subscribe("graylogic/state/+/+", 1, handler)
```

**Topic builders:**
```go
mqtt.Topics{}.State("knx", "light-living")
// → "graylogic/state/knx/light-living"

mqtt.Topics{}.AllStates()
// → "graylogic/state/+/+"

mqtt.Topics{}.SystemStatus()
// → "graylogic/system/status"
```

### Shutdown

```go
if err := client.Close(); err != nil {
    log.Printf("Error closing MQTT: %v", err)
}
```

**Close() performs:**
1. Publishes graceful offline status (different from LWT crash status)
2. Waits for pending publish operations (quiesce)
3. Disconnects from broker

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **Paho library** | Eclipse Foundation, 10+ years stable, MIT licensed | Mochi (newer, less proven) |
| **Subscription tracking** | Restore subscriptions after reconnect | Rely on clean session (loses state) |
| **Synchronous connected state** | Set immediately after Connect() returns | Wait for callback (race condition) |
| **Wrapped handlers** | Panic recovery prevents crash | Raw handlers (crash on panic) |
| **LWT for offline** | Broker publishes on unexpected disconnect | Polling (latency, traffic) |

See also:
- [IMP-003: Subscription Tracking](../decisions/IMP-003-subscription-tracking.md)

---

## Interactions

### Dependencies (what this package uses)

| Package | Purpose |
|---------|---------|
| `config` | Reads MQTT configuration |

### Dependents (what uses this package)

| Package | Purpose |
|---------|---------|
| `cmd/graylogic/main.go` | Creates and manages lifecycle |
| `device/command/` | (Future) Publishes commands to bridges |
| `device/state/` | (Future) Subscribes to bridge state updates |

---

## Error Handling

Package defines domain-specific errors in `errors.go`:

```go
var (
    ErrNotConnected    = errors.New("mqtt: not connected")
    ErrConnectionFailed = errors.New("mqtt: connection failed")
    ErrPublishFailed   = errors.New("mqtt: publish failed")
    ErrSubscribeFailed = errors.New("mqtt: subscribe failed")
)
```

**Usage:**
```go
if errors.Is(err, mqtt.ErrNotConnected) {
    // Handle disconnected state
}
```

---

## Thread Safety

| Operation | Thread-Safe | Notes |
|-----------|-------------|-------|
| `Connect()` | N/A | Called once at startup |
| `Close()` | Yes | Safe from signal handler |
| `Publish()` | Yes | Paho handles internally |
| `Subscribe()` | Yes | Protected by `subMu` mutex |
| `Unsubscribe()` | Yes | Protected by `subMu` mutex |
| `IsConnected()` | Yes | Protected by `connMu` mutex |
| `HealthCheck()` | Yes | Read-only check |
| `SetLogger()` | Yes | Protected by mutex |

**Handler invocation:** Handlers run in paho goroutines. They should not block.

---

## Configuration

From `configs/config.yaml`:

```yaml
mqtt:
  broker:
    host: "localhost"
    port: 1883
    tls: false                    # Enable for production
    client_id: "graylogic-core"
  
  auth:
    username: ""                  # Set via GRAYLOGIC_MQTT_USERNAME
    password: ""                  # Set via GRAYLOGIC_MQTT_PASSWORD
  
  qos: 1                          # 0=at-most-once, 1=at-least-once, 2=exactly-once
  
  reconnect:
    initial_delay: 1              # Seconds before first reconnect
    max_delay: 60                 # Maximum backoff delay
    max_attempts: 0               # 0 = infinite
```

---

## Testing

**Unit tests:** Require running Mosquitto broker.

```bash
# Start Docker services
cd /home/graylogic-dev/gray-logic-stack/code/core
docker compose up -d

# Run tests
make test PKG=./internal/infrastructure/mqtt/...
```

**Test coverage:** Includes connection, publish/subscribe, reconnection scenarios.

---

## Known Limitations

1. **No QoS 2 testing** — Theoretically supported but not tested
2. **No TLS client certificates** — Only username/password auth
3. **No message persistence** — Relies on broker persistence

---

## Related Documents

- [doc.go](file:///home/graylogic-dev/gray-logic-stack/code/core/internal/infrastructure/mqtt/doc.go) — Package-level godoc
- [docs/protocols/mqtt.md](file:///home/graylogic-dev/gray-logic-stack/docs/protocols/mqtt.md) — Topic structure and message formats
