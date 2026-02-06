---
title: Mesh Communications Specification
version: 1.0.0
status: active
last_updated: 2026-01-13
depends_on:
  - resilience/offline.md
  - architecture/system-overview.md
---

# Mesh Communications Specification

This document specifies Gray Logic's out-of-band mesh communication capability — enabling critical alerts and basic control between buildings when primary networks are unavailable.

---

## Overview

### Why Mesh Communications?

Primary communication (WiFi, Ethernet) can fail:

| Failure Scenario | Impact Without Mesh | With Mesh |
|------------------|---------------------|-----------|
| **Internet outage** | No remote alerts | Alerts via mesh |
| **Power outage** | Network switches dead | Battery-powered mesh continues |
| **Network failure** | Buildings isolated | Inter-building communication |
| **Emergency** | Can't reach property | Critical alerts delivered |

### Use Cases

1. **Estate-wide alerts** — Fire alarm in pool house alerts main house
2. **Multi-building coordination** — HVAC status across outbuildings
3. **Emergency communication** — Critical alerts when network is down
4. **Remote properties** — Areas with poor cellular/internet coverage

### Technology: LoRa + Meshtastic

Gray Logic uses **Meshtastic** — an open-source mesh networking protocol over LoRa radio:

| Feature | Specification |
|---------|---------------|
| **Frequency** | 868 MHz (EU), 915 MHz (US) |
| **Range** | 1-10 km (line of sight) |
| **Power** | 1W typical |
| **License** | ISM band, license-free |
| **Battery** | Days to weeks on battery |
| **Encryption** | AES-256 |

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                       MAIN HOUSE                                 │
│                                                                  │
│  ┌──────────────────┐     ┌──────────────────┐                  │
│  │   Gray Logic     │────▶│   Mesh Gateway   │◀──── LoRa Radio  │
│  │   Core           │     │   (Meshtastic)   │                  │
│  └──────────────────┘     └──────────────────┘                  │
│                                    │                             │
└────────────────────────────────────┼─────────────────────────────┘
                                     │
                         LoRa Radio (868/915 MHz)
                                     │
       ┌─────────────────────────────┼─────────────────────────────┐
       │                             │                             │
       ▼                             ▼                             ▼
┌──────────────┐          ┌──────────────┐          ┌──────────────┐
│  POOL HOUSE  │          │   GARAGE     │          │  GATE POST   │
│              │          │              │          │              │
│ Mesh Node    │◀────────▶│ Mesh Node    │◀────────▶│ Mesh Node    │
│ (Meshtastic) │          │ (Meshtastic) │          │ (Meshtastic) │
└──────────────┘          └──────────────┘          └──────────────┘
```

### Mesh Topology

```yaml
mesh_topology:
  # Gateway node (connected to Gray Logic)
  gateway:
    location: "Main house"
    connection: "USB to Gray Logic server"
    role: "router + gateway"
    power: "Mains + battery backup"
    
  # Router nodes (extend range)
  routers:
    - location: "Pool house"
      role: "router"
      power: "Mains + battery"
      sensors: ["fire_alarm_input"]
      
    - location: "Garage"
      role: "router"
      power: "Mains + battery"
      sensors: ["temperature", "door_contact"]
      
  # End nodes (simple sensors)
  end_nodes:
    - location: "Gate post"
      role: "client"
      power: "Solar + battery"
      sensors: ["button", "camera_trigger"]
```

---

## Hardware Requirements

### Meshtastic Devices

```yaml
device_options:
  # Recommended for gateway
  gateway:
    device: "Heltec LoRa 32 V3"
    features:
      - "ESP32 + LoRa SX1262"
      - "USB connection"
      - "Small display"
    cost: "£25-35"
    
  # Alternative gateway
  gateway_alt:
    device: "LilyGO T-Beam"
    features:
      - "ESP32 + LoRa"
      - "GPS (optional)"
      - "18650 battery holder"
    cost: "£35-50"
    
  # Router/relay nodes
  router:
    device: "RAK WisBlock"
    features:
      - "Low power"
      - "Modular I/O"
      - "Industrial enclosure available"
    cost: "£40-60"
    
  # Solar-powered node
  solar_node:
    device: "LilyGO T-Beam with solar panel"
    features:
      - "Solar charging"
      - "Long battery life"
    cost: "£60-80 with panel"
```

### Antennas

```yaml
antennas:
  # Indoor/gateway
  indoor:
    type: "Stubby antenna (included)"
    range: "100-500m"
    
  # Outdoor nodes
  outdoor:
    type: "External antenna (868/915 MHz)"
    options:
      - "Fiberglass collinear (best range)"
      - "Yagi (directional, longest range)"
      - "Ground plane (omnidirectional)"
    range: "1-10 km"
    
  # Cable
  cable:
    type: "LMR-200 or better"
    connector: "SMA"
    length: "As short as practical"
```

### Enclosures

```yaml
enclosures:
  # Outdoor nodes
  outdoor:
    rating: "IP65 or better"
    material: "UV-resistant plastic"
    mounting: "Wall or pole mount"
    
  # Power
  power:
    mains: "5V USB adapter"
    battery: "18650 Li-ion (3.7V)"
    solar: "6V 1W panel (for always-on nodes)"
```

---

## Message Types

### Priority Levels

```yaml
message_priorities:
  critical:
    level: 1
    examples:
      - "Fire alarm triggered"
      - "Intrusion detected"
      - "Water leak detected"
    delivery: "Immediate, retry until acknowledged"
    notification: "Push + SMS + TTS"
    
  high:
    level: 2
    examples:
      - "Power failure"
      - "Temperature alarm"
      - "Security system armed/disarmed"
    delivery: "Immediate"
    notification: "Push"
    
  normal:
    level: 3
    examples:
      - "Equipment status"
      - "Sensor readings"
    delivery: "Best effort"
    notification: "Log only"
    
  low:
    level: 4
    examples:
      - "Heartbeat"
      - "Battery status"
    delivery: "Best effort, no retry"
    notification: "None"
```

### Message Format

```yaml
mesh_message:
  # Header
  header:
    source_node: string             # "pool-house"
    destination: string             # "gateway" | "broadcast"
    priority: int                   # 1-4
    sequence: int                   # Message sequence number
    timestamp: datetime
    
  # Payload
  payload:
    type: string                    # "alert" | "status" | "command" | "ack"
    
    # Alert payload
    alert:
      alert_type: string            # "fire" | "intrusion" | "leak" | "power"
      severity: string              # "critical" | "warning" | "info"
      message: string               # Human-readable
      device_id: string             # Source device
      
    # Status payload
    status:
      measurements:
        - parameter: string
          value: float
          unit: string
          
    # Command payload
    command:
      action: string                # "arm" | "disarm" | "reset"
      target: string
      parameters: object
      
    # Acknowledgment
    ack:
      original_sequence: int
      result: string                # "received" | "executed" | "failed"
```

### Example Messages

```yaml
examples:
  # Fire alarm
  fire_alert:
    source_node: "pool-house"
    destination: "gateway"
    priority: 1
    payload:
      type: "alert"
      alert:
        alert_type: "fire"
        severity: "critical"
        message: "Fire alarm triggered in pool house"
        device_id: "fire-panel-pool"
        
  # Temperature reading
  temperature_status:
    source_node: "garage"
    destination: "gateway"
    priority: 4
    payload:
      type: "status"
      status:
        measurements:
          - parameter: "temperature"
            value: 8.5
            unit: "celsius"
            
  # Arm command
  arm_command:
    source_node: "gateway"
    destination: "pool-house"
    priority: 3
    payload:
      type: "command"
      command:
        action: "arm"
        target: "security-panel"
```

---

## Gray Logic Integration

### Mesh Gateway Service

```yaml
mesh_gateway:
  # Service configuration
  service:
    name: "graylogic-mesh"
    connection: "serial"
    device: "/dev/ttyUSB0"
    baud: 115200
    
  # Meshtastic configuration
  meshtastic:
    region: "EU_868"                # or "US" for 915 MHz
    channel: 0
    channel_name: "GrayLogic"
    psk: "${MESH_PSK}"              # Pre-shared key (AES-256)
    
  # Message handling
  handling:
    # Inbound
    inbound:
      topic: "graylogic/mesh/inbound"
      parse: true
      
    # Outbound
    outbound:
      topic: "graylogic/mesh/outbound"
      queue: true
      max_queue_size: 100
```

### MQTT Topics

```yaml
mqtt_topics:
  # Inbound messages (mesh → Core)
  inbound:
    topic: "graylogic/mesh/inbound/{node_id}"
    payload:
      source_node: "pool-house"
      priority: 1
      message_type: "alert"
      data: {...}
      
  # Outbound messages (Core → mesh)
  outbound:
    topic: "graylogic/mesh/outbound/{node_id}"
    payload:
      priority: 2
      message_type: "command"
      data: {...}
      
  # Node status
  node_status:
    topic: "graylogic/mesh/nodes/{node_id}/status"
    payload:
      online: true
      last_seen: "2026-01-13T10:30:00Z"
      battery_percent: 85
      signal_snr: -5
```

### Alert Integration

```yaml
alert_integration:
  # Map mesh alerts to Gray Logic events
  mappings:
    - mesh_alert: "fire"
      gray_logic_event: "fire_alarm.triggered"
      actions:
        - "notify_all"
        - "scene_activate: emergency-lights"
        - "log_audit"
        
    - mesh_alert: "intrusion"
      gray_logic_event: "security.alarm"
      actions:
        - "notify_owner"
        - "record_cctv"
        
    - mesh_alert: "power_failure"
      gray_logic_event: "power.outage"
      actions:
        - "notify_owner"
        - "log_event"
```

### Automation Triggers

```yaml
automation:
  # Trigger automation from mesh messages
  - name: "Pool house fire alert"
    trigger:
      type: "mesh"
      node: "pool-house"
      message_type: "alert"
      alert_type: "fire"
      
    actions:
      - action: "notify"
        channels: ["push", "sms", "voice_call"]
        message: "FIRE ALERT: Pool house fire alarm triggered"
        
      - action: "scene"
        scene: "emergency-all-lights-on"
        
      - action: "tts"
        text: "Fire alarm in pool house. Please evacuate."
        rooms: "all"
```

---

## Node Configuration

### Gateway Node

```yaml
# Gateway node (connected to Gray Logic)
gateway_config:
  node_id: "gateway"
  role: "router"
  
  # Meshtastic settings
  meshtastic:
    region: "EU_868"
    modem_preset: "LONG_FAST"       # Range vs speed tradeoff
    hop_limit: 3                    # Max hops
    
  # Encryption
  encryption:
    enabled: true
    psk: "${MESH_PSK}"
    
  # Power
  power:
    source: "usb"
    battery_backup: true
    
  # Heartbeat
  heartbeat:
    interval_seconds: 300
    include_status: true
```

### Router Node

```yaml
# Router node (e.g., pool house)
router_config:
  node_id: "pool-house"
  role: "router"
  
  # Sensors connected to node
  sensors:
    - id: "fire-input"
      type: "digital_input"
      gpio: 4
      trigger: "low"                # Active low
      alert_type: "fire"
      
    - id: "temperature"
      type: "ds18b20"
      gpio: 5
      interval_seconds: 300
      
  # Power
  power:
    source: "usb"
    battery: "18650"
    solar: false
    
  # Reporting
  reporting:
    temperature_interval: 300       # Every 5 minutes
    heartbeat_interval: 600
```

### Solar End Node

```yaml
# Solar-powered end node (e.g., gate post)
solar_config:
  node_id: "gate"
  role: "client"                    # End device, doesn't route
  
  # Power optimization
  power:
    source: "solar"
    battery: "18650"
    sleep:
      enabled: true
      wake_interval_seconds: 300    # Wake every 5 minutes
      wake_on_interrupt: true       # Wake on GPIO change
      
  # Sensors
  sensors:
    - id: "button"
      type: "digital_input"
      gpio: 4
      trigger: "rising"
      alert_type: "gate_bell"
      
  # Minimal reporting (save power)
  reporting:
    battery_interval: 3600          # Every hour
    heartbeat_interval: 1800
```

---

## Security

### Encryption

```yaml
encryption:
  # Channel encryption
  channel:
    method: "AES-256"
    psk_source: "environment"       # MESH_PSK env variable
    key_rotation: "manual"          # Change when adding/removing nodes
    
  # Message integrity
  integrity:
    method: "HMAC"
    included: true
    
  # Node authentication
  authentication:
    method: "psk"                   # Pre-shared key
    note: "All nodes must have same PSK"
```

### Security Considerations

```yaml
security_considerations:
  # Physical security
  physical:
    - "Nodes contain PSK — secure enclosures recommended"
    - "Gateway has network access — treat as trusted device"
    
  # Radio security
  radio:
    - "LoRa can be intercepted — encryption essential"
    - "Jamming possible — not suitable for life safety"
    
  # Message security
  messages:
    - "Limit sensitive data in payloads"
    - "Use message IDs to prevent replay attacks"
```

---

## Monitoring

### Node Health

```yaml
node_health:
  # Metrics per node
  metrics:
    - last_seen
    - battery_percent
    - signal_snr
    - message_count
    - error_count
    
  # Health checks
  health:
    check_interval_seconds: 60
    offline_threshold_seconds: 900  # 15 minutes
    
  # Alerts
  alerts:
    node_offline:
      threshold_minutes: 15
      severity: "warning"
      notify: ["push"]
      
    low_battery:
      threshold_percent: 20
      severity: "warning"
      notify: ["push"]
```

### Mesh Status API

```http
GET /api/v1/mesh/status
```

**Response:**
```json
{
  "mesh": {
    "status": "healthy",
    "channel": "GrayLogic",
    "region": "EU_868"
  },
  "nodes": [
    {
      "node_id": "gateway",
      "status": "online",
      "role": "router",
      "last_seen": "2026-01-13T10:30:00Z"
    },
    {
      "node_id": "pool-house",
      "status": "online",
      "role": "router",
      "last_seen": "2026-01-13T10:29:00Z",
      "battery_percent": 95,
      "signal_snr": -5
    },
    {
      "node_id": "gate",
      "status": "online",
      "role": "client",
      "last_seen": "2026-01-13T10:25:00Z",
      "battery_percent": 78,
      "signal_snr": -12
    }
  ],
  "recent_messages": 24,
  "errors_24h": 0
}
```

---

## Configuration

### Complete Mesh Configuration

```yaml
# /etc/graylogic/mesh.yaml
mesh:
  enabled: true
  
  # Gateway configuration
  gateway:
    device: "/dev/ttyUSB0"
    baud: 115200
    
  # Meshtastic settings
  meshtastic:
    region: "EU_868"
    channel_name: "GrayLogic"
    psk_env: "MESH_PSK"
    modem_preset: "LONG_FAST"
    hop_limit: 3
    
  # Nodes
  nodes:
    - id: "pool-house"
      name: "Pool House"
      sensors:
        - type: "fire_input"
          
    - id: "garage"
      name: "Garage"
      sensors:
        - type: "temperature"
        - type: "door_contact"
        
    - id: "gate"
      name: "Gate Post"
      solar: true
      
  # Alert handling
  alerts:
    fire:
      priority: 1
      notify: ["push", "sms"]
      actions: ["emergency_lights"]
      
    intrusion:
      priority: 1
      notify: ["push", "sms"]
      
  # Health monitoring
  health:
    check_interval_seconds: 60
    offline_threshold_seconds: 900
    low_battery_percent: 20
```

---

## Commissioning

### Setup Checklist

```yaml
commissioning:
  gateway:
    - [ ] Flash Meshtastic firmware
    - [ ] Configure region and channel
    - [ ] Set PSK
    - [ ] Connect to Gray Logic server
    - [ ] Verify MQTT communication
    
  nodes:
    - [ ] Flash Meshtastic firmware
    - [ ] Configure same region/channel/PSK
    - [ ] Connect sensors (if any)
    - [ ] Install in location
    - [ ] Verify mesh connectivity
    - [ ] Test alert messages
    
  integration:
    - [ ] Configure node mappings in Gray Logic
    - [ ] Test alert → notification flow
    - [ ] Test command → node flow
    - [ ] Verify battery monitoring
```

### Testing

```yaml
testing:
  # Connectivity test
  connectivity:
    action: "Send test message from each node"
    verify: "Message received at gateway"
    
  # Alert test
  alert_test:
    action: "Trigger test alert on remote node"
    verify: "Alert appears in Gray Logic, notifications sent"
    
  # Range test
  range_test:
    action: "Walk mesh boundary with portable node"
    verify: "Note signal strength at various locations"
    
  # Power failure test
  power_failure:
    action: "Disconnect mains power to node"
    verify: "Node continues on battery, alerts received"
```

---

## Retro Panel — Integrated Mesh Node

### Panel as Meshtastic Endpoint

The Elecrow CrowPanel Advance 3.5" has a modular wireless slot that accepts
a LoRa SX1262 radio module. This means **every retro panel can be a
Meshtastic node** — not just a display, but a full mesh participant that can
route messages, relay commands, and operate as a repeater.

```
                    LoRa Mesh (868/915 MHz, AES-256)
                              │
    ┌─────────────┐    ┌──────┴──────┐    ┌─────────────┐
    │ Core Server │    │   Retro     │    │   Retro     │
    │ + USB       │◄──►│   Panel     │◄──►│   Panel     │
    │   Gateway   │    │ (Kitchen)   │    │ (Garage)    │
    └─────────────┘    │ WiFi + LoRa │    │ LoRa only   │
                       └─────────────┘    └──────┬──────┘
                                                 │
                                          ┌──────┴──────┐
                                          │ Solar Node  │
                                          │ (Gate Post) │
                                          └─────────────┘
```

**Key capabilities:**

| Capability | Description |
|------------|-------------|
| **Secure control** | Send device commands (toggle, dim, scene) over encrypted LoRa when WiFi is down |
| **Mesh repeater** | Panel routes messages between other nodes, extending estate coverage |
| **Fallback comms** | WiFi fails → panel auto-switches to LoRa for command/state traffic |
| **Alert display** | Fire/intrusion/leak alerts from remote nodes displayed on panel screen |
| **Battery resilience** | Panel's LiFePO4 battery keeps mesh alive during power outages |
| **Remote rooms** | Garage, pool house, garden room — panels in areas with no WiFi coverage operate via LoRa |

### Secure Control Channel

The panel can send commands to Gray Logic Core over LoRa when the primary
network (WiFi/Ethernet) is unavailable. This uses a compact binary protocol
over Meshtastic's encrypted channel (AES-256).

```yaml
mesh_control:
  # Authentication — panel must prove identity over mesh
  auth:
    method: "pre-shared panel token"
    token_source: "NVS flash (same as REST X-Panel-Token)"
    challenge_response: true       # Prevents replay attacks

  # Command format — compact for LoRa bandwidth
  command_message:
    panel_id: uint16               # Panel short ID (registered at Core)
    sequence: uint16               # Anti-replay sequence counter
    hmac: bytes[8]                 # Truncated HMAC-SHA256 of payload
    payload:
      type: "command"              # command | scene | query
      device_id_hash: uint16       # CRC16 of device UUID (saves bytes)
      command: uint8               # 0=off, 1=on, 2=toggle, 3=set_level, etc.
      value: uint8                 # Level/position 0-100, or scene index

  # State response format
  state_message:
    device_id_hash: uint16
    state_flags: uint8             # Bit field: on, level_present, pos_present
    level: uint8                   # 0-100 if present
    position: uint8                # 0-100 if present
```

**Why this is safe:**
1. LoRa channel encrypted with AES-256 (Meshtastic standard)
2. Panel token authenticates the sender (same token as REST API)
3. HMAC prevents message tampering
4. Sequence counter prevents replay attacks
5. Core validates panel's room scope — can only control authorised rooms
6. Read-only by default — command capability requires explicit grant

### Network Fallback Logic

```
Panel boot / network change:
  1. Check Ethernet (docked?) → use if available
  2. Check WiFi → use if available
  3. Check LoRa module present?
     a. YES → activate Meshtastic, join mesh
     b. NO  → offline mode (local UI only)

Command routing:
  IF Ethernet/WiFi connected:
    → REST API (fast, full bandwidth)
  ELSE IF LoRa mesh active:
    → Mesh command protocol (slow, essential commands only)
  ELSE:
    → Queue commands, retry when connectivity returns

State updates:
  IF MQTT connected:
    → Full real-time state via MQTT subscription
  ELSE IF LoRa mesh active:
    → Request state on demand (poll, not push — saves bandwidth)
    → Critical alerts pushed automatically via mesh
```

### Panel Mesh Roles

| Mode | Mesh Role | Behaviour |
|------|-----------|-----------|
| **Docked (WiFi + LoRa)** | Router | Routes mesh traffic for other nodes. WiFi for own commands |
| **Portable (WiFi + LoRa)** | Router | Dual-path: WiFi primary, LoRa fallback for commands |
| **Portable (LoRa only)** | Client | No WiFi — all commands via mesh. State polling only |
| **No LoRa module** | None | Standard panel — WiFi/Ethernet only |

### Repeater Functionality

Any panel with a LoRa module automatically acts as a **mesh repeater** when
configured as a router node. This means:

- Installing a panel in the garage (mains powered, LoRa module) extends mesh
  coverage to the garden shed, gate, and neighbouring outbuildings
- A panel on every floor of a large house creates a reliable indoor mesh
- Battery-powered panels keep the mesh alive during power outages (hours to
  days depending on backlight/WiFi state)

Meshtastic handles all routing automatically — no manual configuration needed
beyond setting the same channel/PSK on all nodes.

### Hardware: LoRa Module for Elecrow CrowPanel

The CrowPanel's modular wireless slot accepts:

| Module | Frequency | Range | Cost |
|--------|-----------|-------|------|
| SX1262 LoRa (868MHz EU) | 868 MHz | 1-10 km LOS | ~$5-8 |
| SX1262 LoRa (915MHz US) | 915 MHz | 1-10 km LOS | ~$5-8 |

The module plugs into the CrowPanel's expansion slot — no soldering required.
Flash Meshtastic firmware onto the LoRa module, configure same channel/PSK
as the gateway, and the panel joins the mesh.

For the Waveshare board (no wireless slot), an external SX1262 module
connects via SPI (4 GPIO pins). This requires a small breakout board but
is functionally identical.

---

## Limitations

### What Mesh Comms Can Do

- Deliver critical alerts when primary network is down
- Provide basic sensor data between buildings
- Continue operating on battery during power outages
- Cover large estates (km range)
- Work completely independently of internet

### What Mesh Comms Cannot Do

- High bandwidth (LoRa is slow: ~1 kbps)
- Real-time streaming (video, audio)
- Replace primary network for normal operation
- Guarantee delivery (best effort, with retries)
- Provide life-safety grade reliability

### Important Notes

```yaml
important_notes:
  - note: "Mesh is supplementary, not primary"
    explanation: "Use for alerts and backup, not main communication"
    
  - note: "Not life-safety certified"
    explanation: "Fire alarms should have dedicated monitoring"
    
  - note: "Battery maintenance required"
    explanation: "Batteries need periodic replacement"
    
  - note: "Range varies"
    explanation: "Hills, buildings, and vegetation affect range"
```

---

## Related Documents

- [Offline Behavior](offline.md) — System resilience
- [Security Domain](../domains/security.md) — Alarm integration
- [Fire Alarm Integration](../integration/fire-alarm.md) — Fire system monitoring
- [System Overview](../architecture/system-overview.md) — Architecture
