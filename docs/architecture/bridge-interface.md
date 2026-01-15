---
title: Bridge Interface Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - protocols/mqtt.md
---

# Bridge Interface Specification

This document defines the contract between Gray Logic Core and Protocol Bridges. All bridges must implement this interface to integrate with the system.

---

## Overview

Protocol Bridges are independent processes that translate between Gray Logic's internal representation and specific hardware protocols (KNX, DALI, Modbus, etc.).

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           GRAY LOGIC CORE                                        │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                    MQTT Client                                           │    │
│  │                                                                          │    │
│  │   Subscribes to:                    Publishes to:                        │    │
│  │   - graylogic/state/+/+            - graylogic/command/+/+               │    │
│  │   - graylogic/health/+             - graylogic/config/+                  │    │
│  │   - graylogic/discovery/+          - graylogic/request/+/+               │    │
│  │   - graylogic/response/+/+                                               │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ MQTT (mosquitto)
                                      │
      ┌───────────────────────────────┼───────────────────────────────┐
      │                               │                               │
      ▼                               ▼                               ▼
┌─────────────┐                ┌─────────────┐                ┌─────────────┐
│  KNX Bridge │                │ DALI Bridge │                │Modbus Bridge│
│             │                │             │                │             │
│  Subscribes:│                │  Subscribes:│                │  Subscribes:│
│  command/knx│                │ command/dali│                │command/modbu│
│  request/knx│                │ request/dali│                │request/modbu│
│             │                │             │                │             │
│  Publishes: │                │  Publishes: │                │  Publishes: │
│  state/knx  │                │  state/dali │                │ state/modbus│
│  health/knx │                │  health/dali│                │health/modbus│
│  discovery/ │                │  discovery/ │                │  discovery/ │
└──────┬──────┘                └──────┬──────┘                └──────┬──────┘
       │                              │                               │
       ▼                              ▼                               ▼
   KNX Bus                      DALI Gateway                    Modbus Device
```

### Design Principles

1. **Independence** — Each bridge runs as a separate process
2. **Stateless Core** — Core doesn't know protocol details
3. **Bridge owns translation** — Protocol ↔ normalized state
4. **Health reporting** — Bridges report their status
5. **Graceful restart** — Bridges can restart without Core restart

---

## MQTT Topic Structure

### Topic Hierarchy

```
graylogic/
├── command/{protocol}/{address}     # Core → Bridge: Commands
├── ack/{protocol}/{address}         # Bridge → Core: Command acknowledgments
├── state/{protocol}/{address}       # Bridge → Core: State updates
├── request/{protocol}/{request_id}  # Core → Bridge: Requests
├── response/{protocol}/{request_id} # Bridge → Core: Responses
├── health/{protocol}                # Bridge → Core: Health status
├── discovery/{protocol}             # Bridge → Core: Discovered devices
└── config/{protocol}                # Core → Bridge: Configuration
```

### Address Encoding

Addresses in topics are URL-encoded protocol-specific identifiers.

| Protocol | Address Format | Topic Example |
|----------|----------------|---------------|
| KNX | Group address | `graylogic/state/knx/1%2F2%2F3` (1/2/3) |
| DALI | Gateway:short | `graylogic/state/dali/gw1%3A15` (gw1:15) |
| Modbus | Host:port:unit | `graylogic/state/modbus/192.168.1.50%3A502%3A1` |

---

## Message Formats

All messages are JSON with a standard envelope.

### Command Message (Core → Bridge)

```json
{
  "id": "cmd-uuid-123",
  "timestamp": "2026-01-12T10:30:00Z",
  "device_id": "light-living-1",
  "command": "dim",
  "parameters": {
    "level": 50,
    "transition_ms": 1000
  },
  "source": "api",
  "user_id": "user-darren"
}
```

**Topic:** `graylogic/command/{protocol}/{address}`

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique command ID for correlation |
| `timestamp` | ISO8601 | Yes | When command was issued |
| `device_id` | string | Yes | Gray Logic device ID |
| `command` | string | Yes | Command name |
| `parameters` | object | No | Command parameters |
| `source` | string | Yes | Origin: `api`, `automation`, `voice`, `scene` |
| `user_id` | string | No | User who triggered (if applicable) |

### Command Acknowledgment Message (Bridge → Core)

Bridges MUST acknowledge every command received. This enables Core to track command delivery and detect failures.

```json
{
  "command_id": "cmd-uuid-123",
  "timestamp": "2026-01-12T10:30:00.500Z",
  "device_id": "light-living-1",
  "status": "accepted",
  "protocol": "knx",
  "address": "1/2/3"
}
```

**Topic:** `graylogic/ack/{protocol}/{address}`

**QoS:** 1 (at least once)

**Retained:** No (acknowledgments are transient)

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command_id` | string | Yes | ID from the original command |
| `timestamp` | ISO8601 | Yes | When acknowledgment was sent |
| `device_id` | string | Yes | Gray Logic device ID |
| `status` | string | Yes | Acknowledgment status (see below) |
| `protocol` | string | Yes | Protocol identifier |
| `address` | string | Yes | Protocol-specific address |
| `error` | object | No | Error details if status is `failed` |

**Status Values:**

| Status | Description |
|--------|-------------|
| `accepted` | Command received and sent to device |
| `queued` | Command received, waiting to send (device busy) |
| `failed` | Command could not be executed |
| `timeout` | Device did not respond within timeout |

**Error Object (when status is `failed` or `timeout`):**

```json
{
  "command_id": "cmd-uuid-123",
  "timestamp": "2026-01-12T10:30:05Z",
  "device_id": "light-living-1",
  "status": "timeout",
  "protocol": "knx",
  "address": "1/2/3",
  "error": {
    "code": "DEVICE_UNREACHABLE",
    "message": "Device did not respond within 5000ms",
    "retries": 3
  }
}
```

**Core Behavior:**

- Core tracks pending commands by `command_id`
- If no acknowledgment within 10 seconds, Core logs warning
- Scene engine uses acknowledgments to track action execution
- UI can display command status to users

---

### State Message (Bridge → Core)

```json
{
  "device_id": "light-living-1",
  "timestamp": "2026-01-12T10:30:01Z",
  "state": {
    "on": true,
    "level": 50
  },
  "protocol": "knx",
  "address": "1/2/3"
}
```

**Topic:** `graylogic/state/{protocol}/{address}`

**QoS:** 1 (at least once)

**Retained:** Yes (for current state)

**Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `device_id` | string | Yes | Gray Logic device ID |
| `timestamp` | ISO8601 | Yes | When state was observed |
| `state` | object | Yes | Current device state |
| `protocol` | string | Yes | Protocol identifier |
| `address` | string | Yes | Protocol-specific address |

### State Object Schemas

#### Lighting State

```json
{
  "on": true,
  "level": 50,
  "color_temp": 4000,
  "color": {
    "r": 255,
    "g": 200,
    "b": 150
  }
}
```

#### Blind State

```json
{
  "position": 50,
  "tilt": 30,
  "moving": false,
  "direction": null
}
```

#### Climate Sensor State

```json
{
  "temperature": 21.5,
  "humidity": 45.0,
  "co2": 650
}
```

#### Switch/Contact State

```json
{
  "state": true
}
```

#### Motion Sensor State

```json
{
  "motion": true,
  "last_motion": "2026-01-12T10:29:55Z"
}
```

---

## Request/Response Pattern

For operations requiring acknowledgment or data retrieval.

### Request Message (Core → Bridge)

```json
{
  "request_id": "req-uuid-456",
  "timestamp": "2026-01-12T10:30:00Z",
  "action": "read_state",
  "device_id": "sensor-temp-1",
  "parameters": {}
}
```

**Topic:** `graylogic/request/{protocol}/{request_id}`

### Response Message (Bridge → Core)

```json
{
  "request_id": "req-uuid-456",
  "timestamp": "2026-01-12T10:30:01Z",
  "success": true,
  "data": {
    "temperature": 21.5
  },
  "error": null
}
```

**Topic:** `graylogic/response/{protocol}/{request_id}`

### Request Types

| Action | Description | Parameters |
|--------|-------------|------------|
| `read_state` | Request current state | `device_id` |
| `read_all` | Read all device states | - |
| `discover` | Trigger discovery | `scope` (optional) |
| `reconfigure` | Reload configuration | - |
| `restart` | Restart bridge | - |

---

## Health Reporting

Bridges must report health status periodically.

### Health Message

```json
{
  "bridge": "knx",
  "timestamp": "2026-01-12T10:30:00Z",
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "connection": {
    "status": "connected",
    "address": "192.168.1.10:3671",
    "connected_since": "2026-01-11T10:30:00Z"
  },
  "statistics": {
    "messages_received": 12345,
    "messages_sent": 6789,
    "errors": 2
  },
  "devices_managed": 42
}
```

**Topic:** `graylogic/health/{protocol}`

**QoS:** 1

**Retained:** Yes

**Publish Interval:** Every 30 seconds

### Last Will and Testament (LWT)

Bridges must configure LWT for disconnect detection:

```json
{
  "bridge": "knx",
  "timestamp": "2026-01-12T10:30:00Z",
  "status": "offline",
  "reason": "unexpected_disconnect"
}
```

**LWT Topic:** `graylogic/health/{protocol}`

**LWT Retain:** Yes

### Status Values

| Status | Description |
|--------|-------------|
| `healthy` | Operating normally |
| `degraded` | Operating with issues |
| `unhealthy` | Not operating correctly |
| `offline` | Not connected (LWT) |
| `starting` | Bridge starting up |
| `stopping` | Bridge shutting down |

---

## Device Discovery

Bridges can announce discovered devices.

### Discovery Message

```json
{
  "timestamp": "2026-01-12T10:30:00Z",
  "bridge": "knx",
  "devices": [
    {
      "protocol": "knx",
      "address": "1/2/3",
      "type": "light_dimmer",
      "capabilities": ["on_off", "dim"],
      "manufacturer": "ABB",
      "product": "DG/S 1.64.5.1",
      "suggested_name": "Dimmer 1/2/3"
    }
  ]
}
```

**Topic:** `graylogic/discovery/{protocol}`

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `protocol` | string | Protocol identifier |
| `address` | string | Protocol-specific address |
| `type` | string | Device type (from standard list) |
| `capabilities` | array | Capability identifiers |
| `manufacturer` | string | Device manufacturer |
| `product` | string | Product model |
| `suggested_name` | string | Suggested display name |

---

## Configuration

Core can push configuration to bridges.

### Configuration Message

```json
{
  "timestamp": "2026-01-12T10:30:00Z",
  "bridge": "knx",
  "devices": [
    {
      "device_id": "light-living-1",
      "protocol": "knx",
      "address": {
        "control": "1/2/3",
        "status": "1/2/4"
      },
      "type": "light_dimmer",
      "capabilities": ["on_off", "dim"],
      "config": {
        "min_level": 5,
        "max_level": 100
      }
    }
  ],
  "settings": {
    "poll_interval_ms": 0,
    "reconnect_delay_ms": 5000
  }
}
```

**Topic:** `graylogic/config/{protocol}`

**Retained:** Yes

---

## Bridge Implementation Requirements

### Mandatory Behaviors

1. **Connect to MQTT broker on startup**
2. **Subscribe to required topics:**
   - `graylogic/command/{protocol}/#`
   - `graylogic/request/{protocol}/#`
   - `graylogic/config/{protocol}`

3. **Set LWT on MQTT connection**

4. **Publish health status:**
   - On startup (status: `starting`)
   - Every 30 seconds (status: `healthy` or other)
   - On shutdown (status: `stopping`)

5. **Publish state on change:**
   - When device state changes
   - Retained message for current state

6. **Handle commands:**
   - Translate command to protocol
   - Publish acknowledgment immediately (`accepted`, `queued`, or `failed`)
   - Execute on device
   - Publish follow-up acknowledgment if timeout/failure occurs
   - State update will confirm final execution

7. **Handle configuration:**
   - Load device mappings
   - Apply settings

### Error Handling

```json
{
  "request_id": "req-uuid-456",
  "timestamp": "2026-01-12T10:30:01Z",
  "success": false,
  "data": null,
  "error": {
    "code": "DEVICE_UNREACHABLE",
    "message": "Device 1/2/3 did not respond",
    "details": {
      "timeout_ms": 5000,
      "retries": 3
    }
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `DEVICE_UNREACHABLE` | Device not responding |
| `INVALID_COMMAND` | Command not supported |
| `INVALID_PARAMETERS` | Parameters invalid |
| `PROTOCOL_ERROR` | Protocol-level error |
| `TIMEOUT` | Operation timed out |
| `NOT_CONFIGURED` | Device not in configuration |
| `BRIDGE_ERROR` | Internal bridge error |

---

## Bridge Lifecycle

### Startup Sequence

```
1. Load configuration file
2. Connect to MQTT broker
   - Set LWT
   - Subscribe to topics
3. Publish health status (starting)
4. Connect to protocol (KNX/DALI/Modbus)
5. Request configuration from Core (optional)
6. Initialize device state cache
7. Publish health status (healthy)
8. Begin normal operation
```

### Shutdown Sequence

```
1. Receive shutdown signal (SIGTERM/SIGINT)
2. Publish health status (stopping)
3. Complete pending operations (timeout 5s)
4. Disconnect from protocol
5. Disconnect from MQTT (graceful)
6. Exit
```

### Reconnection

```
On protocol disconnect:
1. Publish health status (degraded)
2. Attempt reconnect with exponential backoff
3. On success: publish health status (healthy)
4. On persistent failure: publish health status (unhealthy)

On MQTT disconnect:
1. Attempt reconnect with exponential backoff
2. On success: re-subscribe, publish health
3. On persistent failure: log and continue attempting
```

---

## Example: KNX Bridge

### Subscriptions

```
graylogic/command/knx/#
graylogic/request/knx/#
graylogic/config/knx
```

### Command Processing

```go
func handleCommand(msg CommandMessage) {
    // 1. Look up KNX address from device_id
    address := deviceToAddress[msg.DeviceID]
    
    // 2. Translate command to KNX telegram
    var telegram knx.GroupEvent
    switch msg.Command {
    case "on":
        telegram = knx.GroupEvent{
            Command:     knx.GroupWrite,
            Destination: address,
            Data:        []byte{0x01},
        }
    case "dim":
        level := msg.Parameters["level"].(float64)
        dptValue := byte(level * 255 / 100)
        telegram = knx.GroupEvent{
            Command:     knx.GroupWrite,
            Destination: address,
            Data:        []byte{dptValue},
        }
    }
    
    // 3. Send to KNX bus
    err := knxClient.Send(telegram)
    
    // 4. Log result (state update comes from bus)
    if err != nil {
        log.Error("Failed to send KNX telegram", "error", err)
    }
}
```

### State Publishing

```go
func onKNXTelegram(event knx.GroupEvent) {
    // 1. Look up device_id from KNX address
    deviceID := addressToDevice[event.Destination]
    if deviceID == "" {
        return // Unknown device, ignore
    }
    
    // 2. Translate KNX data to normalized state
    state := translateKNXToState(event)
    
    // 3. Build state message
    msg := StateMessage{
        DeviceID:  deviceID,
        Timestamp: time.Now().UTC(),
        State:     state,
        Protocol:  "knx",
        Address:   event.Destination.String(),
    }
    
    // 4. Publish to MQTT
    topic := fmt.Sprintf("graylogic/state/knx/%s", 
        url.PathEscape(event.Destination.String()))
    mqtt.Publish(topic, msg, QoS1, Retained)
}
```

---

## Example: Modbus Bridge

### Polling Architecture

```go
type ModbusBridge struct {
    devices    []ModbusDevice
    pollTicker *time.Ticker
}

type ModbusDevice struct {
    DeviceID   string
    Address    ModbusAddress
    Registers  []RegisterSpec
    LastState  map[string]any
}

func (b *ModbusBridge) pollLoop() {
    for range b.pollTicker.C {
        for _, device := range b.devices {
            state, err := b.pollDevice(device)
            if err != nil {
                log.Error("Poll failed", "device", device.DeviceID)
                continue
            }
            
            // Only publish if changed
            if !reflect.DeepEqual(state, device.LastState) {
                b.publishState(device.DeviceID, state)
                device.LastState = state
            }
        }
    }
}
```

---

## Security Considerations

### Authentication

Bridges authenticate to MQTT with credentials:

```yaml
# Bridge configuration
mqtt:
  broker: "tcp://localhost:1883"
  username: "bridge-knx"
  password: "${BRIDGE_KNX_PASSWORD}"
```

### Authorization (ACL)

MQTT broker restricts bridge access:

```
# Mosquitto ACL
user bridge-knx
topic read graylogic/command/knx/#
topic read graylogic/request/knx/#
topic read graylogic/config/knx
topic write graylogic/ack/knx/#
topic write graylogic/state/knx/#
topic write graylogic/health/knx
topic write graylogic/discovery/knx
topic write graylogic/response/knx/#

user graylogic-core
topic readwrite graylogic/#
```

### Network Isolation

Bridges should run on the same host or isolated network segment as the MQTT broker.

---

## Monitoring

### Metrics per Bridge

Each bridge should expose Prometheus metrics:

```
# Protocol metrics
graylogic_bridge_messages_received_total{bridge="knx"} 12345
graylogic_bridge_messages_sent_total{bridge="knx"} 6789
graylogic_bridge_errors_total{bridge="knx", error="timeout"} 5

# Connection metrics
graylogic_bridge_connected{bridge="knx"} 1
graylogic_bridge_connection_uptime_seconds{bridge="knx"} 86400
graylogic_bridge_reconnects_total{bridge="knx"} 2

# Device metrics
graylogic_bridge_devices_managed{bridge="knx"} 42
graylogic_bridge_device_last_seen_timestamp{bridge="knx", device="1/2/3"} 1736674200
```

### Logging

Structured JSON logging:

```json
{
  "level": "info",
  "ts": "2026-01-12T10:30:00Z",
  "bridge": "knx",
  "msg": "Command executed",
  "device_id": "light-living-1",
  "command": "dim",
  "address": "1/2/3"
}
```

---

## Related Documents

- [Core Internals Architecture](core-internals.md) — Core architecture
- [MQTT Protocol Specification](../protocols/mqtt.md) — MQTT details
- [KNX Protocol Specification](../protocols/knx.md) — KNX bridge specifics
- [DALI Protocol Specification](../protocols/dali.md) — DALI bridge specifics
- [Modbus Protocol Specification](../protocols/modbus.md) — Modbus bridge specifics

