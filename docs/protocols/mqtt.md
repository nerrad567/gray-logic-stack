---
title: MQTT Protocol Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - architecture/bridge-interface.md
---

# MQTT Protocol Specification

This document specifies how Gray Logic uses MQTT as its internal message bus for communication between Core and Protocol Bridges.

---

## Overview

### What is MQTT?

MQTT (Message Queuing Telemetry Transport) is a lightweight publish/subscribe messaging protocol:

- **Simple** — Minimal overhead, easy to implement
- **Reliable** — QoS levels for delivery guarantees
- **Efficient** — Low bandwidth, ideal for IoT
- **Scalable** — Supports many clients
- **Decoupled** — Publishers and subscribers independent

### Why MQTT for Gray Logic?

| Requirement | MQTT Capability |
|-------------|-----------------|
| **Bridge isolation** | Bridges communicate without direct coupling |
| **Event-driven** | Real-time state updates |
| **Debuggable** | Easy to monitor with standard tools |
| **Reliable** | QoS ensures message delivery |
| **Mature** | Well-supported, stable implementations |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     GRAY LOGIC CORE                              │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Device      │  │ Automation  │  │ Intelligence            │  │
│  │ Layer       │  │ Layer       │  │ Layer                   │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│         │                │                     │                │
│         └────────────────┴─────────────────────┘                │
│                          │                                      │
│                   ┌──────▼──────┐                               │
│                   │ MQTT Client │                               │
│                   └──────┬──────┘                               │
└──────────────────────────┼──────────────────────────────────────┘
                           │
                    ┌──────▼──────┐
                    │  Mosquitto  │
                    │   Broker    │
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
   ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
   │ KNX Bridge  │  │ DALI Bridge │  │Modbus Bridge│
   └─────────────┘  └─────────────┘  └─────────────┘
```

---

## Broker Configuration

### Mosquitto

Gray Logic uses [Eclipse Mosquitto](https://mosquitto.org/) as the MQTT broker.

**`/etc/mosquitto/mosquitto.conf`:**

```conf
# Mosquitto Configuration for Gray Logic

# Persistence
persistence true
persistence_location /var/lib/mosquitto/

# Logging
log_dest file /var/log/mosquitto/mosquitto.log
log_type error
log_type warning
log_type notice
log_type information
log_timestamp true
log_timestamp_format %Y-%m-%dT%H:%M:%S

# Listeners
listener 1883 127.0.0.1
protocol mqtt

# For TLS (production)
# listener 8883 0.0.0.0
# protocol mqtt
# cafile /etc/mosquitto/certs/ca.crt
# certfile /etc/mosquitto/certs/server.crt
# keyfile /etc/mosquitto/certs/server.key
# require_certificate false

# WebSocket listener (for web UI)
listener 9001 127.0.0.1
protocol websockets

# Authentication
password_file /etc/mosquitto/passwd
allow_anonymous false

# ACL (Access Control)
acl_file /etc/mosquitto/acl

# Connection limits
max_connections 100
max_queued_messages 1000
message_size_limit 262144

# Keepalive
keepalive_interval 60
```

### Authentication

**`/etc/mosquitto/passwd`** (create with `mosquitto_passwd`):

```
graylogic-core:<hashed_password>
knx-bridge-01:<hashed_password>
dali-bridge-01:<hashed_password>
modbus-bridge-01:<hashed_password>
```

### Access Control

**`/etc/mosquitto/acl`:**

```acl
# Gray Logic Core - full access
user graylogic-core
topic readwrite graylogic/#

# Bridges - access to their own topics
user knx-bridge-01
topic readwrite graylogic/bridge/knx-bridge-01/#
topic read graylogic/system/#

user dali-bridge-01
topic readwrite graylogic/bridge/dali-bridge-01/#
topic read graylogic/system/#

user modbus-bridge-01
topic readwrite graylogic/bridge/modbus-bridge-01/#
topic read graylogic/system/#
```

---

## Topic Structure

### Hierarchy

```
graylogic/
├── bridge/
│   └── {bridge_id}/
│       ├── state/{device_id}       # Device state updates
│       ├── command/{device_id}     # Commands to devices
│       ├── response/{request_id}   # Command responses
│       ├── health                  # Bridge health status
│       └── discovery               # Device discovery
├── core/
│   ├── device/{device_id}/state    # Canonical device state
│   ├── event/{event_type}          # System events
│   ├── scene/{scene_id}/activated  # Scene activations
│   ├── scene/{scene_id}/progress   # Scene execution progress
│   ├── automation/{rule_id}/fired  # Automation rule triggered
│   ├── alert/{alert_id}            # System alerts
│   └── mode                        # Mode changes
├── system/
│   ├── status                      # System status
│   ├── time                        # Time sync
│   └── shutdown                    # Shutdown signal
└── ui/
    └── {client_id}/                # UI-specific topics
        ├── notification            # Push notifications
        └── presence                # UI presence updates
```

### Topic Naming Conventions

| Rule | Example |
|------|---------|
| Lowercase only | `graylogic/bridge/knx-bridge-01` |
| Hyphens for word separation | `knx-bridge-01`, `living-room` |
| No spaces | Never use spaces |
| Meaningful hierarchy | `bridge/{id}/state/{device}` |
| Device IDs are slugs | `light-living-main` |

---

## Message Formats

All messages are JSON-encoded. Timestamps use ISO 8601 format with timezone.

### State Update (Bridge → Core)

When a bridge receives a state change from a physical device:

```yaml
topic: graylogic/bridge/knx-bridge-01/state/light-living-main
qos: 1
retain: false
payload:
  device_id: "light-living-main"
  timestamp: "2026-01-12T14:30:00.123Z"
  source: "knx"
  
  state:
    on: true
    brightness: 75
    # Domain-specific state properties
    
  raw:
    # Protocol-specific raw data (optional, for debugging)
    - ga: "6/0/1"
      dpt: "1.001"
      value: 1
```

### Command (Core → Bridge)

When Core needs to control a device:

```yaml
topic: graylogic/bridge/knx-bridge-01/command/light-living-main
qos: 1
retain: false
payload:
  device_id: "light-living-main"
  command: "set"
  parameters:
    on: true
    brightness: 50
    transition_ms: 1000
  request_id: "req-abc123"
  timestamp: "2026-01-12T14:30:00.000Z"
  source: "core"
  context:
    trigger: "scene"
    scene_id: "cinema-mode"
    user_id: "user-001"
```

### Command Response (Bridge → Core)

Bridge acknowledges command execution:

```yaml
topic: graylogic/bridge/knx-bridge-01/response/req-abc123
qos: 1
retain: false
payload:
  request_id: "req-abc123"
  device_id: "light-living-main"
  success: true
  timestamp: "2026-01-12T14:30:00.150Z"
  # On failure:
  # success: false
  # error:
  #   code: "DEVICE_OFFLINE"
  #   message: "Device did not respond"
```

### Bridge Health (Bridge → Core)

Periodic health status from each bridge:

```yaml
topic: graylogic/bridge/knx-bridge-01/health
qos: 1
retain: true
payload:
  bridge_id: "knx-bridge-01"
  status: "online"  # online | degraded | offline
  timestamp: "2026-01-12T14:30:00Z"
  
  metrics:
    uptime_seconds: 86400
    messages_rx: 15432
    messages_tx: 8721
    errors_last_hour: 0
    queue_depth: 0
    
  protocol:
    connected: true
    last_activity: "2026-01-12T14:29:55Z"
    # Protocol-specific metrics
    
  devices:
    total: 45
    online: 44
    offline: 1
    error: 0
```

### Device Discovery (Bridge → Core)

When a bridge discovers new devices:

```yaml
topic: graylogic/bridge/knx-bridge-01/discovery
qos: 1
retain: false
payload:
  bridge_id: "knx-bridge-01"
  timestamp: "2026-01-12T14:30:00Z"
  action: "discovered"  # discovered | lost | updated
  
  devices:
    - protocol_address: "1/2/3"
      type: "unknown"  # Or detected type
      manufacturer: "MDT"
      model: "BE-04001.02"
      capabilities:
        - "switch"
        - "dimming"
```

### Canonical Device State (Core publishes)

Core maintains canonical state for all UIs:

```yaml
topic: graylogic/core/device/light-living-main/state
qos: 1
retain: true
payload:
  device_id: "light-living-main"
  timestamp: "2026-01-12T14:30:00Z"
  
  state:
    on: true
    brightness: 75
    
  health:
    status: "online"
    last_seen: "2026-01-12T14:30:00Z"
    
  meta:
    name: "Living Room Main Light"
    room_id: "room-living"
    type: "light_dimmer"
```

### System Events (Core publishes)

Core broadcasts significant events:

```yaml
topic: graylogic/core/event/device_state_changed
qos: 1
retain: false
payload:
  event_type: "device_state_changed"
  timestamp: "2026-01-12T14:30:00Z"
  
  data:
    device_id: "light-living-main"
    previous_state:
      on: false
      brightness: 0
    new_state:
      on: true
      brightness: 75
    trigger:
      type: "physical"
      source: "knx"
```

### Scene Activation (Core publishes)

```yaml
topic: graylogic/core/scene/cinema-mode
qos: 1
retain: false
payload:
  event_type: "scene_activated"
  timestamp: "2026-01-12T14:30:00Z"
  
  scene_id: "cinema-mode"
  scene_name: "Cinema Mode"
  
  trigger:
    type: "manual"  # manual | scheduled | automation | voice
    source: "ui"
    user_id: "user-001"
    
  actions_count: 5
```

### Mode Change (Core publishes)

```yaml
topic: graylogic/core/mode
qos: 1
retain: true
payload:
  event_type: "mode_changed"
  timestamp: "2026-01-12T14:30:00Z"

  previous_mode: "home"
  new_mode: "night"

  trigger:
    type: "scheduled"
    schedule_id: "schedule-night-mode"
```

### Scene Execution Progress (Core publishes)

For multi-step scenes, Core publishes progress updates:

```yaml
topic: graylogic/core/scene/cinema-mode/progress
qos: 1
retain: false
payload:
  scene_id: "cinema-mode"
  execution_id: "exec-abc123"         # Unique per execution
  timestamp: "2026-01-12T14:30:01Z"

  status: "running"                   # pending | running | completed | failed | cancelled

  progress:
    total_actions: 5
    completed_actions: 3
    current_action: 4
    current_action_device: "blind-living-01"

  errors: []                          # Array of failed actions
    # - device_id: "light-broken"
    #   error: "Device offline"

  started_at: "2026-01-12T14:30:00Z"
  eta_seconds: 2                      # Estimated time remaining
```

### Automation Rule Fired (Core publishes)

When an automation rule triggers:

```yaml
topic: graylogic/core/automation/rule-sunrise-blinds/fired
qos: 1
retain: false
payload:
  rule_id: "rule-sunrise-blinds"
  rule_name: "Open blinds at sunrise"
  timestamp: "2026-01-12T08:01:00Z"

  trigger:
    type: "event"                     # event | schedule | condition
    source: "sun_position"
    event_data:
      event: "sunrise"

  conditions_evaluated:
    - condition: "mode == home"
      result: true
    - condition: "weekday"
      result: true

  actions_executed: 3
  success: true
```

### System Alert (Core publishes)

For system-wide alerts:

```yaml
topic: graylogic/core/alert/alert-dali-offline
qos: 1
retain: true                          # Persist until cleared
payload:
  alert_id: "alert-dali-offline"
  timestamp: "2026-01-12T14:30:00Z"

  severity: "warning"                 # info | warning | critical
  category: "connectivity"            # connectivity | hardware | performance | security

  component: "bridges.dali"

  title: "DALI bridge offline"
  message: "The DALI lighting bridge has disconnected"
  customer_message: "Some lights unavailable"  # Simplified for end users

  active: true                        # false when cleared
  acknowledged: false
  acknowledged_by: null

  first_occurred: "2026-01-12T14:30:00Z"
  occurrence_count: 1

  actions:                            # Suggested remediation
    - label: "Restart bridge"
      api: "POST /api/v1/bridges/dali-bridge-01/restart"
    - label: "Contact support"
      url: "tel:+441onal23456789"
```

**Alert Lifecycle:**
1. Alert created → published with `active: true`, `retain: true`
2. Alert cleared → re-published with `active: false`, then broker retention cleared
3. UIs subscribe to `graylogic/core/alert/+` for all alerts

### System Status (Core publishes)

```yaml
topic: graylogic/system/status
qos: 1
retain: true
payload:
  status: "running"  # starting | running | degraded | stopping
  timestamp: "2026-01-12T14:30:00Z"
  
  version: "1.0.0"
  uptime_seconds: 604800
  
  bridges:
    - id: "knx-bridge-01"
      status: "online"
    - id: "dali-bridge-01"
      status: "online"
    - id: "modbus-bridge-01"
      status: "degraded"
      
  mode: "home"
  
  metrics:
    devices_total: 150
    devices_online: 148
    automations_active: 25
    memory_mb: 28
    cpu_percent: 2.5
```

### Time Sync (Core publishes)

For bridges without NTP:

```yaml
topic: graylogic/system/time
qos: 0
retain: true
payload:
  timestamp: "2026-01-12T14:30:00Z"
  timezone: "Europe/London"
  sunrise: "2026-01-12T08:01:00Z"
  sunset: "2026-01-12T16:15:00Z"
```

### Shutdown Signal (Core publishes)

Graceful shutdown notification:

```yaml
topic: graylogic/system/shutdown
qos: 1
retain: false
payload:
  event: "shutdown"
  timestamp: "2026-01-12T14:30:00Z"
  reason: "restart"  # restart | upgrade | manual | error
  delay_seconds: 5
```

---

## QoS Levels

| Level | Name | Guarantee | Use Case |
|-------|------|-----------|----------|
| **0** | At most once | Fire and forget | Time sync, non-critical |
| **1** | At least once | Guaranteed delivery | Commands, state updates |
| **2** | Exactly once | No duplicates | Financial transactions (not used) |

### Gray Logic QoS Usage

| Topic Pattern | QoS | Rationale |
|---------------|-----|-----------|
| `bridge/*/state/*` | 1 | State must arrive |
| `bridge/*/command/*` | 1 | Commands must execute |
| `bridge/*/health` | 1 | Health is important |
| `core/device/*/state` | 1 | UI needs accurate state |
| `core/event/*` | 1 | Events trigger automation |
| `core/scene/*/activated` | 1 | Scene activation is important |
| `core/scene/*/progress` | 1 | Progress updates for UI |
| `core/automation/*/fired` | 1 | Audit trail |
| `core/alert/*` | 1 | Alerts must be delivered |
| `system/time` | 0 | Periodic, loss acceptable |
| `system/status` | 1 | Status is important |

---

## Retained Messages

Retained messages persist on the broker and are delivered to new subscribers:

| Topic Pattern | Retained | Rationale |
|---------------|----------|-----------|
| `bridge/*/health` | Yes | New Core sees bridge status |
| `core/device/*/state` | Yes | UI gets current state on connect |
| `core/mode` | Yes | Mode is persistent state |
| `core/alert/*` | Yes | Active alerts persist until cleared |
| `system/status` | Yes | New clients see system state |
| `system/time` | Yes | Time reference for new clients |
| `bridge/*/state/*` | No | Historical states not needed |
| `bridge/*/command/*` | No | Commands are ephemeral |
| `core/event/*` | No | Events are transient |
| `core/scene/*/activated` | No | Scene events are transient |
| `core/scene/*/progress` | No | Progress is ephemeral |
| `core/automation/*/fired` | No | Rule events are transient |

---

## Last Will and Testament (LWT)

Bridges use LWT to notify Core if they disconnect unexpectedly:

```yaml
# Bridge connection with LWT
will_topic: graylogic/bridge/knx-bridge-01/health
will_payload:
  bridge_id: "knx-bridge-01"
  status: "offline"
  timestamp: ""  # Will be stale
  reason: "unexpected_disconnect"
will_qos: 1
will_retain: true
```

---

## Message Flow Examples

### User Turns On Light via UI

```
1. UI sends HTTP POST /api/devices/light-living-main/state
   { "on": true, "brightness": 100 }

2. Core validates request, checks authorization

3. Core publishes to MQTT:
   Topic: graylogic/bridge/knx-bridge-01/command/light-living-main
   Payload: { device_id, command: "set", parameters: { on: true, brightness: 100 }, request_id }

4. KNX Bridge receives command

5. KNX Bridge sends KNX telegram to actuator

6. Actuator turns on light

7. Actuator sends status telegram

8. KNX Bridge publishes to MQTT:
   Topic: graylogic/bridge/knx-bridge-01/state/light-living-main
   Payload: { device_id, state: { on: true, brightness: 100 } }

9. Core receives state update

10. Core updates internal state

11. Core publishes canonical state:
    Topic: graylogic/core/device/light-living-main/state
    Payload: { device_id, state: { on: true, brightness: 100 } }

12. Core broadcasts state change via WebSocket to all UIs

13. UIs update display
```

### Physical Switch Pressed

```
1. User presses wall switch

2. KNX switch sends telegram to actuator (direct bus link)

3. Actuator turns on light

4. Actuator sends status telegram

5. KNX Bridge publishes to MQTT:
   Topic: graylogic/bridge/knx-bridge-01/state/light-living-main
   Payload: { device_id, state: { on: true, brightness: 100 } }

6. Core receives state update

7. Core publishes event:
   Topic: graylogic/core/event/device_state_changed
   Payload: { device_id, previous_state, new_state, trigger: { type: "physical" } }

8. Automation engine evaluates triggers

9. Core updates canonical state

10. Core broadcasts to UIs via WebSocket
```

### Scene Activation

```
1. User activates "Good Night" scene via voice

2. Voice pipeline extracts intent: activate_scene(good-night)

3. Core loads scene definition

4. Core publishes commands to each bridge:
   - graylogic/bridge/knx-bridge-01/command/light-bedroom-main
   - graylogic/bridge/knx-bridge-01/command/blind-bedroom-01
   - graylogic/bridge/dali-bridge-01/command/light-bathroom-01

5. Bridges execute commands in parallel

6. Core publishes scene event:
   Topic: graylogic/core/scene/good-night
   Payload: { scene_id, trigger: { type: "voice" } }

7. Core updates mode if scene affects mode

8. Core collects state updates from bridges

9. Core broadcasts final states to UIs
```

---

## Bridge Implementation Guidelines

### Connection Lifecycle

```go
// Pseudocode
func (b *Bridge) Start() {
    // 1. Connect to MQTT broker
    client := mqtt.Connect(brokerURL, clientID, credentials)
    
    // 2. Set LWT
    client.SetWill(healthTopic, offlinePayload, QoS1, Retained)
    
    // 3. Subscribe to command topics
    client.Subscribe("graylogic/bridge/{id}/command/+", handleCommand)
    client.Subscribe("graylogic/system/#", handleSystem)
    
    // 4. Publish online status
    client.Publish(healthTopic, onlinePayload, QoS1, Retained)
    
    // 5. Start protocol connection
    protocol.Connect()
    
    // 6. Start health reporting
    go reportHealth()
    
    // 7. Request initial state from devices
    protocol.QueryAllDevices()
}

func (b *Bridge) Shutdown() {
    // 1. Stop protocol polling
    protocol.Stop()
    
    // 2. Publish offline status (graceful)
    client.Publish(healthTopic, gracefulOfflinePayload, QoS1, Retained)
    
    // 3. Disconnect MQTT
    client.Disconnect()
}
```

### Message Handling

```go
func handleCommand(topic string, payload []byte) {
    var cmd Command
    json.Unmarshal(payload, &cmd)
    
    // Validate command
    if err := validate(cmd); err != nil {
        publishError(cmd.RequestID, err)
        return
    }
    
    // Execute on protocol
    err := protocol.Execute(cmd)
    
    // Publish response
    if err != nil {
        publishError(cmd.RequestID, err)
    } else {
        publishSuccess(cmd.RequestID)
    }
}

func handleStateChange(device Device, state State) {
    payload := StateUpdate{
        DeviceID:  device.ID,
        Timestamp: time.Now().UTC(),
        Source:    "knx",
        State:     state,
    }
    
    client.Publish(
        fmt.Sprintf("graylogic/bridge/%s/state/%s", bridgeID, device.ID),
        json.Marshal(payload),
        QoS1,
        NotRetained,
    )
}
```

### Health Reporting

```go
func reportHealth() {
    ticker := time.NewTicker(30 * time.Second)
    
    for range ticker.C {
        health := HealthStatus{
            BridgeID:  bridgeID,
            Status:    calculateStatus(),
            Timestamp: time.Now().UTC(),
            Metrics:   collectMetrics(),
            Protocol:  protocolStatus(),
            Devices:   deviceCounts(),
        }
        
        client.Publish(healthTopic, json.Marshal(health), QoS1, Retained)
    }
}

func calculateStatus() string {
    if !protocol.Connected() {
        return "offline"
    }
    if errors.LastHour() > 10 || offlineDevices() > totalDevices()/10 {
        return "degraded"
    }
    return "online"
}
```

---

## Security

### Transport Security

For production deployments:

1. **TLS** — Enable TLS on broker (port 8883)
2. **Client certificates** — Mutual TLS for bridges
3. **Password rotation** — Regular credential updates

### Message Validation

- Validate JSON schema on all messages
- Reject malformed messages
- Log security-relevant events
- Rate-limit publishers

---

## Monitoring and Debugging

### Mosquitto Logging

```bash
# View logs
tail -f /var/log/mosquitto/mosquitto.log

# Enable debug logging (temporary)
mosquitto_ctrl dynamic-security setDefaultACLAction publish allow
```

### MQTT CLI Tools

```bash
# Subscribe to all topics
mosquitto_sub -h localhost -u graylogic-core -P password -t "graylogic/#" -v

# Subscribe to bridge states
mosquitto_sub -h localhost -t "graylogic/bridge/+/state/+" -v

# Publish test command
mosquitto_pub -h localhost -t "graylogic/bridge/knx-bridge-01/command/light-test" \
  -m '{"device_id":"light-test","command":"set","parameters":{"on":true},"request_id":"test-1"}'

# View retained messages
mosquitto_sub -h localhost -t "graylogic/core/device/+/state" -v --retained-only
```

### GUI Tools

- **MQTT Explorer** — Desktop app for browsing topics
- **MQTT.fx** — Feature-rich desktop client
- **HiveMQ Web Client** — Browser-based

---

## Performance Considerations

### Message Size

- Keep messages under 256KB (broker limit)
- Typical state messages: 500-2000 bytes
- Use compression for large payloads (rare)

### Throughput

- Mosquitto handles 100k+ messages/second
- Gray Logic typical: 10-100 messages/second
- Optimize polling intervals to reduce load

### Latency

- Local broker: <1ms
- Target end-to-end: <100ms for commands

---

## Related Documents

- [KNX Protocol Specification](knx.md) — KNX bridge details
- [DALI Protocol Specification](dali.md) — DALI bridge details
- [Modbus Protocol Specification](modbus.md) — Modbus bridge details
- [Bridge Interface Specification](../architecture/bridge-interface.md) — Bridge contract
- [Core Internals](../architecture/core-internals.md) — Core architecture

