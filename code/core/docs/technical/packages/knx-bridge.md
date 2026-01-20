# KNX Bridge Package Design

> `internal/bridges/knx/` — KNX protocol bridge via knxd daemon

## Purpose

Provides bidirectional communication between the KNX building bus and Gray Logic Core:
- Connection to knxd daemon (TCP or Unix socket)
- Parsing and encoding of KNX group telegrams
- Translation between KNX and MQTT message formats
- Group address handling with 3-level notation (Main/Middle/Sub)
- Datapoint Type (DPT) encoding/decoding

**Why KNX?** See [docs/protocols/knx.md](../../../../../docs/protocols/knx.md) — European building automation standard (ISO/IEC 14543-3) for lighting, HVAC, blinds, and sensors.

---

## Architecture

```
┌───────────────────────────────────────────────────────────────────────────┐
│                            KNX Bridge Binary                               │
│                                                                            │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐     │
│  │    KNXDClient    │    │     Bridge       │    │   MQTT Client    │     │
│  │  (knxd.go)       │◄──►│   (bridge.go)    │◄──►│   (from core)    │     │
│  │                  │    │                  │    │                  │     │
│  │ • Connect        │    │ • GA→Device map  │    │ • Publish state  │     │
│  │ • Send telegram  │    │ • State aggreg.  │    │ • Subscribe cmd  │     │
│  │ • OnTelegram cb  │    │ • Health report  │    │                  │     │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘     │
│           │                                                │               │
└───────────│────────────────────────────────────────────────│───────────────┘
            │                                                │
            ▼ TCP/Unix                                       ▼ TCP
┌──────────────────────┐                        ┌──────────────────────┐
│    knxd Daemon       │                        │   Mosquitto Broker   │
│  (localhost:6720)    │                        │  (localhost:1883)    │
└──────────────────────┘                        └──────────────────────┘
            │
            ▼
┌──────────────────────┐
│   KNX/IP Gateway     │
│   (Physical Bus)     │
└──────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                          Physical KNX Devices                             │
│  [Wall Switch]  [Dimmer Actuator]  [Blind Motor]  [Temperature Sensor]   │
└──────────────────────────────────────────────────────────────────────────┘
```

### Key Types

| Type | File | Purpose |
|------|------|---------|
| `GroupAddress` | address.go | KNX group address (Main/Middle/Sub format) |
| `Telegram` | telegram.go | KNX message with destination, APCI, and data |
| `KNXDClient` | knxd.go | Connection manager for knxd daemon |
| `KNXDConfig` | knxd.go | Connection configuration |
| `Connector` | knxd.go | Interface for testability (mocking) |

### APCI Types (Application Protocol Control Information)

| Constant | Value | Meaning |
|----------|-------|---------|
| `APCIRead` | 0x00 | Request current value from device |
| `APCIResponse` | 0x40 | Device reply to read request |
| `APCIWrite` | 0x80 | Send value to devices on group address |

---

## How It Works

### knxd Protocol

The knxd daemon uses a simple binary protocol over TCP or Unix socket:

```
┌──────────────────────────────────────────────────────────────────┐
│                    knxd Message Format                            │
├──────────────────────────────────────────────────────────────────┤
│ Byte 0-1: Total message size (big-endian, includes header)       │
│ Byte 2-3: Message type (big-endian)                              │
│ Byte 4+:  Payload (variable length)                              │
└──────────────────────────────────────────────────────────────────┘

Message Types:
  0x0026 = EIB_OPEN_GROUPCON — Open group communication mode
  0x0027 = EIB_GROUP_PACKET  — Send/receive group telegram
  0x0006 = EIB_CLOSE         — Close connection gracefully
```

### Connection Flow

```go
// 1. Create configuration
cfg := knx.KNXDConfig{
    Connection:     "tcp://localhost:6720",  // or "unix:///run/knxd"
    ConnectTimeout: 10 * time.Second,
    ReadTimeout:    30 * time.Second,
}

// 2. Connect to knxd
client, err := knx.Connect(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 3. Set telegram handler
client.SetOnTelegram(func(t knx.Telegram) {
    log.Printf("Received: GA=%s, APCI=0x%02X, Data=%X",
        t.Destination, t.APCI, t.Data)
})

// 4. Send a telegram
ga := knx.GroupAddress{Main: 1, Middle: 0, Sub: 1}
data := []byte{0x01}  // DPT 1.001: ON
err = client.Send(ctx, ga, data)
```

**Connect() performs:**
1. Parse connection URL (tcp:// or unix://)
2. Dial with timeout
3. Send EIB_OPEN_GROUPCON to enter group mode
4. Wait for acknowledgement
5. Start receive goroutine

### Telegram Flow: Physical Switch → App

```
[Wall Switch Pressed]
        │
        ▼
[KNX Bus: Telegram to GA 1/0/1, APCI=Write, Data=0x01]
        │
        ▼
[knxd daemon receives telegram]
        │
        ▼ TCP
[KNXDClient.receiveLoop() reads message]
        │
        ▼
[ParseTelegram() decodes bytes]
        │
        ▼
[onTelegram callback invoked]
        │
        ▼
[Bridge: Lookup device by GA, publish MQTT]
        │
        ▼ MQTT
[Core receives state update]
        │
        ▼
[UI refreshes]
```

### Telegram Flow: App → Physical Device

```
[User taps "ON" in app]
        │
        ▼
[Core publishes MQTT command]
        │
        ▼ MQTT
[Bridge: Subscribe receives command]
        │
        ▼
[Bridge: Lookup GA by device, encode DPT]
        │
        ▼
[KNXDClient.Send(ga, data)]
        │
        ▼
[Telegram.Encode() → knxd wire format]
        │
        ▼ TCP
[knxd sends to KNX bus]
        │
        ▼
[Dimmer actuator receives, turns on light]
```

### Shutdown

```go
if err := client.Close(); err != nil {
    log.Printf("Error closing: %v", err)
}
```

**Close() performs:**
1. Signal receive loop to stop (close `done` channel)
2. Mark disconnected
3. Close TCP/Unix connection (unblocks pending reads)
4. Wait for goroutine to finish (`wg.Wait()`)

---

## Design Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **Separate binary** | Can restart bridge without affecting Core; isolates KNX crashes | Part of Core (simpler but coupled) |
| **knxd daemon** | Proven, handles all KNX/IP variants; we don't reinvent transport | Direct KNX/IP (complex multicast, tunneling) |
| **TCP default** | Works in Docker without socket mounts | Unix socket (faster but complex setup) |
| **Callback pattern** | Simple, non-blocking receive | Channel (more Go-idiomatic but harder to use) |
| **Atomic stats** | Lock-free performance counters | Mutex (simpler but contention) |
| **Connector interface** | Enables mock testing without real knxd | Concrete type only (harder to test) |

---

## Interactions

### Dependencies (what this package uses)

| Package | Purpose |
|---------|---------|
| `encoding/binary` | Big-endian wire format |
| `net` | TCP/Unix socket connections |
| `net/url` | Parse connection URLs |

### Dependents (what uses this package)

| Package | Purpose |
|---------|---------|
| `cmd/knxbridge/main.go` | Entry point, lifecycle management |
| `bridge.go` (internal) | Orchestrates KNX ↔ MQTT translation |

---

## Error Handling

Package defines domain-specific errors in `errors.go`:

```go
var (
    ErrInvalidGroupAddress = errors.New("knx: invalid group address")
    ErrInvalidDPT          = errors.New("knx: invalid datapoint type")
    ErrInvalidTelegram     = errors.New("knx: invalid telegram")
    ErrConnectionFailed    = errors.New("knx: connection failed")
    ErrNotConnected        = errors.New("knx: not connected")
    ErrTelegramFailed      = errors.New("knx: telegram send failed")
)
```

**Usage:**
```go
if errors.Is(err, knx.ErrNotConnected) {
    // Handle disconnected state
}
```

---

## Thread Safety

| Operation | Thread-Safe | Notes |
|-----------|-------------|-------|
| `Connect()` | N/A | Called once at startup |
| `Close()` | Yes | Safe from signal handler |
| `Send()` | Yes | Protected by `connMu` |
| `SendRead()` | Yes | Protected by `connMu` |
| `SetOnTelegram()` | Yes | Protected by `callbackMu` |
| `IsConnected()` | Yes | Protected by `connMu` |
| `Stats()` | Yes | Uses atomic counters |
| `HealthCheck()` | Yes | Read-only check |

**Handler invocation:** Callbacks run in dedicated goroutines. Panics are recovered.

---

## Configuration

From `configs/knx-bridge.yaml` (to be created):

```yaml
bridge:
  id: "knx-bridge-01"
  health_interval: 30  # seconds

knxd:
  connection: "tcp://localhost:6720"  # or "unix:///run/knxd"
  connect_timeout: 10                  # seconds
  read_timeout: 30                     # seconds
  reconnect_interval: 5                # seconds

mqtt:
  broker: "tcp://localhost:1883"
  client_id: "knx-bridge-01"
  qos: 1

devices:
  - device_id: "light-living-main"
    type: "light_dimmer"
    addresses:
      switch:
        ga: "1/0/1"
        dpt: "1.001"
        flags: ["write"]
      brightness_status:
        ga: "6/0/2"
        dpt: "5.001"
        flags: ["read", "transmit"]
```

---

## Testing

**Unit tests:** Can run without external dependencies (mock server included).

```bash
# Run all KNX tests
cd /home/graylogic-dev/gray-logic-stack/code/core
go test -v ./internal/bridges/knx/...

# Run specific test
go test -v -run TestKNXDClientConnectAndSend ./internal/bridges/knx/...
```

**Test files:**

| File | Tests | Coverage |
|------|-------|----------|
| `telegram_test.go` | 21 tests | Parsing, encoding, roundtrip |
| `knxd_test.go` | 12 tests | Connect, send, receive, stats |
| `address_test.go` | (existing) | Group address parsing |
| `dpt_test.go` | (existing) | Datapoint encoding |

**Mock server:** `MockKNXDServer` in `knxd_test.go` simulates knxd for testing.

---

## Statistics

The client tracks operational metrics:

```go
stats := client.Stats()
// stats.TelegramsTx  — Telegrams sent
// stats.TelegramsRx  — Telegrams received
// stats.ErrorsTotal  — Cumulative errors
// stats.LastActivity — Timestamp of last telegram
// stats.Connected    — Current connection state
```

---

## Known Limitations

1. **No auto-reconnect yet** — Planned for bridge.go orchestration layer
2. **No ETS project import** — Manual device configuration required
3. **No KNX Secure** — Plain KNX only (sufficient for isolated home network)
4. **No multiple lines** — Single knxd connection (covers most installations)
5. **No tunneling mode** — Group communication only (most common use case)

---

## Files Overview

| File | Lines | Purpose |
|------|-------|---------|
| `address.go` | ~100 | Group address parsing (existing) |
| `dpt.go` | ~200 | Datapoint encoding (existing) |
| `errors.go` | ~30 | Domain errors (existing) |
| `telegram.go` | ~180 | Telegram parse/encode (new) |
| `knxd.go` | ~350 | knxd client (new) |
| `telegram_test.go` | ~340 | Telegram tests (new) |
| `knxd_test.go` | ~380 | Client tests (new) |

---

## Related Documents

- [doc.go](../../../internal/bridges/knx/doc.go) — Package-level godoc
- [docs/protocols/knx.md](../../../../../docs/protocols/knx.md) — KNX protocol specification
- [docs/architecture/bridge-interface.md](../../../../../docs/architecture/bridge-interface.md) — Bridge contract

---

*Last updated: 2026-01-20*
