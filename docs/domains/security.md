---
title: Security Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/principles.md
  - architecture/core-internals.md
  - data-model/entities.md
  - integration/access-control.md
  - integration/cctv.md
---

# Security Domain Specification

This document specifies how Gray Logic integrates with intruder alarm systems, providing monitoring, status display, and controlled interaction while respecting the principle that **the alarm panel is the primary system**.

---

## Overview

### Philosophy

Gray Logic is a **secondary interface** to the alarm system, not a replacement:

| Alarm Panel (Primary) | Gray Logic (Secondary) |
|-----------------------|------------------------|
| Certified hardware | Monitoring and display |
| Independent operation | Automation triggers |
| Physical keypads | Mobile/UI arm/disarm |
| Siren control | Notifications |
| Sensor processing | Scene integration |
| Zone configuration | Status dashboard |

### Design Principles

1. **Panel is primary** — The alarm panel operates independently; Gray Logic failure must not affect security
2. **Physical always works** — Keypads, fobs, and sirens work regardless of Gray Logic state
3. **Secure by default** — All remote arm/disarm requires authentication + confirmation
4. **Complete audit trail** — Every security action is logged with user, time, and method
5. **Fail-safe** — Communication loss = alarm panel continues operating normally

### Safety Rules

**Gray Logic SHALL:**
- Monitor alarm state and zone status
- Display alerts and notifications
- Trigger automations based on alarm events
- Provide authenticated arm/disarm interface

**Gray Logic SHALL NOT:**
- Be the only method to arm/disarm
- Bypass or disable alarm functionality
- Suppress or silence alarm events without user action
- Control sirens directly (panel controls sirens)

---

## Supported Alarm Systems

### Integration Methods

| Method | Description | Examples |
|--------|-------------|----------|
| **Native API** | Direct integration via manufacturer API | Texecom Connect, Ajax |
| **Serial/IP** | Direct protocol connection | Texecom, Honeywell Galaxy, Paradox |
| **Contact closure** | Simple I/O for basic status | Any panel with outputs |
| **Monitoring receiver** | SIA/Contact ID protocol | Professional monitoring |

### Supported Panels

| Manufacturer | Model Range | Integration |
|--------------|-------------|-------------|
| **Texecom** | Premier Elite, Veritas | Serial (Crestron protocol), IP |
| **Honeywell** | Galaxy Dimension, Galaxy Flex | Serial (RS-232) |
| **Paradox** | EVO, SP, MG Series | Serial (PRT3), IP (IP150) |
| **Ajax** | Hub, Hub Plus, Hub 2 | Cloud API, local polling |
| **Risco** | LightSYS, ProSYS | Serial, IP |
| **Pyronix** | Enforcer, Euro | Serial (HomeControl+) |
| **DSC** | PowerSeries, PowerSeries Neo | IT-100/Envisalink |

### Integration Depth

| Level | Capabilities | Requirements |
|-------|--------------|--------------|
| **Basic** | Arm state only | Single output contact |
| **Standard** | Arm state + zones + alerts | Serial/IP connection |
| **Full** | All above + arm/disarm + bypass | Authenticated integration |

---

## Data Model

### Alarm Panel

```yaml
AlarmPanel:
  id: uuid
  site_id: uuid
  name: string                        # "Main Alarm Panel"
  
  # Hardware
  manufacturer: string                # "texecom"
  model: string                       # "Premier Elite 48"
  firmware_version: string | null
  
  # Connection
  connection:
    type: "serial" | "ip" | "api"
    host: string | null               # IP address
    port: integer | null              # Port number
    serial_device: string | null      # "/dev/ttyUSB0"
    baud_rate: integer | null         # 9600
    
  # Partitions (areas)
  partitions: [AlarmPartition]
  
  # Capabilities
  capabilities:
    arm_disarm: boolean               # Can Gray Logic arm/disarm
    bypass_zones: boolean             # Can bypass zones
    zone_status: boolean              # Can read zone status
    real_time_events: boolean         # Receives events in real-time
    
  # Current state
  state:
    connection_status: "online" | "offline" | "error"
    last_seen: timestamp
    last_event: timestamp
    
  created_at: timestamp
  updated_at: timestamp
```

### Alarm Partition

Most panels support multiple partitions (areas) that can be armed independently.

```yaml
AlarmPartition:
  id: uuid
  panel_id: uuid
  partition_number: integer           # 1-8 typically
  name: string                        # "Downstairs", "Upstairs"
  
  # Mapping
  areas: [uuid]                       # Gray Logic areas covered
  
  # Current state
  state:
    arm_state: enum                   # See arm states below
    arm_method: string | null         # "keypad", "fob", "app", "schedule"
    armed_by: string | null           # User ID or identifier
    armed_at: timestamp | null
    
    # Status flags
    ready: boolean                    # All zones secure, can arm
    trouble: boolean                  # System trouble condition
    alarm_memory: boolean             # Alarm occurred since last arm
    
    # Active conditions
    alarm_active: boolean             # Alarm currently sounding
    entry_delay: boolean              # Entry delay in progress
    exit_delay: boolean               # Exit delay in progress
```

**Arm States:**

| State | Description |
|-------|-------------|
| `disarmed` | System disarmed, monitoring only |
| `arm_away` | Full arm, all zones active |
| `arm_stay` | Perimeter only, interior zones bypassed |
| `arm_night` | Night mode (varies by panel) |
| `arm_instant` | Armed with no entry delay |
| `arming` | Exit delay in progress |
| `pending` | Entry delay in progress |
| `alarm` | Alarm condition active |

### Alarm Zone

```yaml
AlarmZone:
  id: uuid
  panel_id: uuid
  partition_id: uuid
  zone_number: integer                # Zone number on panel
  name: string                        # "Front Door", "PIR Hallway"
  
  # Zone type
  type: enum                          # See zone types below
  
  # Mapping
  device_id: uuid | null              # Associated Gray Logic device
  room_id: uuid | null                # Room containing zone
  
  # Configuration
  config:
    entry_zone: boolean               # Starts entry delay
    exit_zone: boolean                # Active during exit delay
    chime: boolean                    # Chimes when opened (disarmed)
    bypass_allowed: boolean           # Can be bypassed
    
  # Current state
  state:
    status: "secure" | "open" | "alarm" | "tamper" | "fault" | "bypassed"
    last_change: timestamp
    
  created_at: timestamp
  updated_at: timestamp
```

**Zone Types:**

| Type | Description | Examples |
|------|-------------|----------|
| `door` | Entry/exit door contact | Front door, back door |
| `window` | Window contact | Bedroom window |
| `pir` | Passive infrared motion | Hallway PIR |
| `glass_break` | Glass break detector | Living room GB |
| `smoke` | Smoke detector (if wired to panel) | Kitchen smoke |
| `panic` | Panic button | Bedroom panic |
| `tamper` | Tamper/anti-masking | PIR tamper |
| `technical` | Technical alarm (freeze, flood) | Boiler room flood |
| `24hr` | 24-hour zone (always active) | Safe tamper |

### Alarm Event

```yaml
AlarmEvent:
  id: uuid
  panel_id: uuid
  partition_id: uuid | null
  zone_id: uuid | null
  
  timestamp: timestamp
  
  # Event classification
  event_type: enum                    # See event types below
  event_code: string                  # Panel-specific code
  
  # Details
  description: string                 # Human-readable description
  user_id: string | null              # User who caused event (if applicable)
  user_name: string | null            # User name from panel
  
  # Processing
  acknowledged: boolean
  acknowledged_by: uuid | null
  acknowledged_at: timestamp | null
```

**Event Types:**

| Type | Description |
|------|-------------|
| `arm` | System armed |
| `disarm` | System disarmed |
| `alarm` | Alarm activated |
| `restore` | Alarm/trouble restored |
| `tamper` | Tamper detected |
| `trouble` | System trouble |
| `bypass` | Zone bypassed |
| `unbypass` | Zone unbypass |
| `zone_open` | Zone opened |
| `zone_close` | Zone closed |
| `entry` | Entry delay started |
| `exit` | Exit delay started |
| `panic` | Panic alarm |
| `duress` | Duress code used |
| `low_battery` | Low battery (zone or panel) |
| `ac_fail` | AC power failure |
| `ac_restore` | AC power restored |
| `comm_fail` | Communication failure |
| `test` | System test |

---

## Commands

### Arm/Disarm

All arm/disarm commands require authentication and are logged.

```yaml
# Arm away
- panel_id: "alarm-panel-1"
  command: "arm"
  parameters:
    partition: 1                      # Or "all"
    mode: "away"                      # away, stay, night, instant
    user_pin: "1234"                  # Required for remote arm/disarm
    
# Arm stay (perimeter only)
- panel_id: "alarm-panel-1"
  command: "arm"
  parameters:
    partition: 1
    mode: "stay"
    user_pin: "1234"

# Disarm
- panel_id: "alarm-panel-1"
  command: "disarm"
  parameters:
    partition: 1
    user_pin: "1234"
```

### Zone Bypass

```yaml
# Bypass a zone
- panel_id: "alarm-panel-1"
  command: "bypass_zone"
  parameters:
    zone: 5
    user_pin: "1234"
    reason: "Window open for ventilation"

# Unbypass a zone
- panel_id: "alarm-panel-1"
  command: "unbypass_zone"
  parameters:
    zone: 5
    user_pin: "1234"
```

### Status Query

```yaml
# Get partition status
- panel_id: "alarm-panel-1"
  command: "get_status"
  parameters:
    partition: 1

# Get zone status
- panel_id: "alarm-panel-1"
  command: "get_zone_status"
  parameters:
    zone: 5                           # Or "all"
```

---

## Automation Integration

### Mode Integration

Security arm state is tied to Gray Logic modes:

```yaml
modes:
  - id: "home"
    behaviours:
      security:
        arm_state: "disarm"
        
  - id: "away"
    behaviours:
      security:
        arm_state: "arm_away"
        auto_arm_delay_minutes: 5     # Delay before auto-arm
        
  - id: "night"
    behaviours:
      security:
        arm_state: "arm_stay"
        
  - id: "vacation"
    behaviours:
      security:
        arm_state: "arm_away"
```

### Event-Driven Automation

```yaml
# Alarm triggered → Emergency response
triggers:
  - type: "security_event"
    source:
      panel_id: "alarm-panel-1"
      event_type: "alarm"
    execute:
      type: "actions"
      actions:
        - domain: "lighting"
          command: "all_on"
          parameters:
            brightness: 100
        - domain: "audio"
          command: "announce"
          parameters:
            tts_text: "Alarm activated"
            priority: "critical"
        - domain: "notification"
          command: "send"
          parameters:
            title: "ALARM"
            message: "Intruder alarm activated at ${zone_name}"
            priority: "critical"
            channels: ["push", "sms"]

# Arm away → Automation
triggers:
  - type: "security_state_changed"
    source:
      panel_id: "alarm-panel-1"
      event: "armed_away"
    execute:
      type: "scene"
      scene_id: "scene-leaving-home"

# Disarm → Welcome home
triggers:
  - type: "security_state_changed"
    source:
      panel_id: "alarm-panel-1"
      event: "disarmed"
    execute:
      type: "scene"
      scene_id: "scene-welcome-home"
    conditions:
      - type: "time"
        operator: "between"
        value: ["06:00", "22:00"]

# Entry delay → Path lighting
triggers:
  - type: "security_event"
    source:
      event_type: "entry"
    execute:
      type: "actions"
      actions:
        - device_id: "light-hallway"
          command: "on"
          parameters:
            brightness: 50
```

### Zone-Based Triggers

```yaml
# Front door opened (while disarmed) → Doorbell notification
triggers:
  - type: "zone_state_changed"
    source:
      zone_id: "zone-front-door"
      from: "secure"
      to: "open"
    conditions:
      - type: "security_state"
        operator: "eq"
        value: "disarmed"
    execute:
      type: "actions"
      actions:
        - domain: "notification"
          command: "send"
          parameters:
            message: "Front door opened"

# Motion in living room + away mode → Alert
triggers:
  - type: "zone_state_changed"
    source:
      zone_id: "zone-pir-living"
      to: "open"
    conditions:
      - type: "mode"
        operator: "eq"
        value: "away"
    execute:
      type: "actions"
      actions:
        - domain: "cctv"
          command: "record_clip"
          parameters:
            camera_id: "camera-living"
            duration_seconds: 30
```

---

## User Interface

### Status Display

```yaml
SecurityDashboard:
  partitions:
    - partition_id: "partition-1"
      name: "Main House"
      arm_state: "disarmed"
      ready: true
      zones:
        total: 24
        open: 1
        bypass: 0
        trouble: 0
        
  recent_events:
    - timestamp: "2026-01-12T08:30:00Z"
      event: "Disarmed by John (Keypad)"
    - timestamp: "2026-01-12T07:15:00Z"
      event: "Armed Stay by Schedule"
      
  zone_status:
    - zone: "Front Door"
      type: "door"
      status: "secure"
    - zone: "Back Door"
      type: "door"
      status: "open"
      note: "Preventing arm"
```

### Arm/Disarm Interface

UI requirements for arm/disarm:

1. **Clear state indication** — Current arm state prominently displayed
2. **Ready status** — Show if system is ready to arm
3. **Open zones** — List zones preventing arm
4. **PIN entry** — Secure PIN entry for arm/disarm
5. **Confirmation** — Require confirmation for arm/disarm
6. **Exit delay countdown** — Show countdown during exit delay
7. **Entry delay warning** — Prominent entry delay indication

```yaml
ArmDisarmFlow:
  # Step 1: Show current state and options
  current_state: "disarmed"
  ready_to_arm: true
  options:
    - "Arm Away"
    - "Arm Stay"
    - "Arm Night"
    
  # Step 2: PIN entry
  prompt: "Enter PIN to arm"
  pin_entry: true
  
  # Step 3: Confirmation
  confirm: "Arm Away? Exit delay: 60 seconds"
  
  # Step 4: Exit delay countdown
  exit_delay: 60
  status: "Leave now"
```

---

## MQTT Topics

### State Updates (Bridge → Core)

```yaml
# Partition state
topic: graylogic/bridge/security-bridge/state/partition/{partition_id}
payload:
  partition_id: "partition-1"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    arm_state: "arm_away"
    armed_by: "user-john"
    armed_at: "2026-01-12T14:25:00Z"
    ready: true
    trouble: false
    alarm_active: false

# Zone state
topic: graylogic/bridge/security-bridge/state/zone/{zone_id}
payload:
  zone_id: "zone-front-door"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    status: "secure"
    last_change: "2026-01-12T14:25:00Z"

# Event
topic: graylogic/bridge/security-bridge/event
payload:
  panel_id: "alarm-panel-1"
  partition_id: "partition-1"
  zone_id: "zone-front-door"
  timestamp: "2026-01-12T14:30:00Z"
  event_type: "arm"
  event_code: "3401"
  description: "Armed Away by User 1"
  user_name: "John"
```

### Commands (Core → Bridge)

```yaml
# Arm command
topic: graylogic/bridge/security-bridge/command
payload:
  command: "arm"
  parameters:
    partition: 1
    mode: "away"
    user_code: "1234"
  request_id: "req-12345"

# Response
topic: graylogic/bridge/security-bridge/response
payload:
  request_id: "req-12345"
  success: true
  message: "Armed away, exit delay started"
```

---

## Security Bridge

### Architecture

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  Alarm Panel    │◄────►│  Security Bridge │◄────►│  Gray Logic     │
│  (Texecom, etc) │      │  (Go process)    │      │  Core           │
└─────────────────┘      └─────────────────┘      └─────────────────┘
     Serial/IP              State/Events            MQTT
                            Commands
```

### Bridge Responsibilities

1. **Protocol translation** — Panel protocol ↔ Gray Logic messages
2. **State polling** — Query panel for current state
3. **Event reception** — Receive and forward events
4. **Command execution** — Execute arm/disarm/bypass commands
5. **Connection monitoring** — Detect and report connection issues
6. **PIN handling** — Securely handle PIN codes (never logged)

### Configuration

```yaml
# /etc/graylogic/bridges/security.yaml
bridge:
  id: "security-bridge"
  type: "security"
  
  panel:
    manufacturer: "texecom"
    model: "premier_elite_48"
    
  connection:
    type: "serial"
    device: "/dev/ttyUSB0"
    baud_rate: 19200
    protocol: "crestron"              # Texecom Crestron protocol
    
  # Panel configuration
  config:
    udl_password: "${TEXECOM_UDL}"    # From environment
    partitions: 2
    zones: 48
    
  # Polling intervals
  polling:
    state_interval_ms: 1000
    zone_interval_ms: 5000
    
  mqtt:
    broker: "localhost"
    port: 1883
    client_id: "security-bridge"
    username: "security"
    password_env: "MQTT_PASSWORD"
```

---

## PHM Integration

Security systems have limited PHM applicability, but some monitoring is valuable.

### PHM Value for Security

| Component | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Panel communication | ★★★☆☆ | Connection stability, response time |
| Zone sensors | ★★☆☆☆ | False alarm rate, battery status |
| Backup battery | ★★★★☆ | Voltage, charge cycles |

### Monitored Parameters

```yaml
phm_security:
  devices:
    - device_id: "alarm-panel-1"
      type: "alarm_panel"
      parameters:
        - name: "comm_latency_ms"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 100
          alert: "Communication latency elevated"
          
        - name: "reconnect_count"
          baseline_method: "daily_count"
          threshold_per_day: 3
          alert: "Frequent panel reconnections"
          
        - name: "battery_voltage"
          threshold_low: 12.0
          threshold_critical: 11.5
          alert: "Panel backup battery low"
          
    - device_id: "zone-pir-hallway"
      type: "security_zone"
      parameters:
        - name: "false_alarm_count"
          baseline_method: "weekly_count"
          threshold: 5
          alert: "Excessive false alarms - check sensor"
          
        - name: "battery_low_events"
          threshold: 1
          alert: "Zone sensor battery low"
```

---

## Commercial Security

### Multi-Partition Sites

Commercial sites often have multiple partitions:

```yaml
commercial_security:
  panel_id: "alarm-panel-office"
  
  partitions:
    - partition: 1
      name: "Reception"
      areas: ["area-reception", "area-meeting"]
      schedule:
        arm: "19:00"
        disarm: "07:00"
        days: ["mon", "tue", "wed", "thu", "fri"]
        
    - partition: 2
      name: "Server Room"
      areas: ["area-server"]
      config:
        always_armed: true            # 24-hour protection
        
    - partition: 3
      name: "Warehouse"
      areas: ["area-warehouse"]
      schedule:
        arm: "18:00"
        disarm: "06:00"
```

### Access Control Integration

Link access control with alarm:

```yaml
access_security_integration:
  # Valid access disarms zone
  - trigger:
      type: "access_granted"
      reader_id: "reader-main-entry"
    action:
      command: "disarm"
      partition: 1
      method: "access_card"
      user_id: "${access_user}"
      
  # Invalid access + alarm = escalate
  - trigger:
      type: "access_denied"
      conditions:
        - security_state: "armed"
    action:
      command: "alert"
      priority: "high"
      message: "Invalid access attempt while armed"
```

### Monitoring Integration

For monitored sites:

```yaml
monitoring:
  enabled: true
  
  # Monitoring station connection
  station:
    protocol: "sia"                   # SIA/Contact ID
    primary:
      type: "ip"
      host: "monitoring.example.com"
      port: 12345
      account: "1234A"
    backup:
      type: "pstn"
      number: "01onal23456789"
      
  # Event filtering (what to send)
  report_events:
    - "alarm"
    - "panic"
    - "duress"
    - "fire"
    - "tamper"
    - "ac_fail"
    - "comm_fail"
```

---

## Configuration Examples

### Residential: Texecom

```yaml
security:
  panel:
    manufacturer: "texecom"
    model: "Premier Elite 24"
    
  connection:
    type: "serial"
    device: "/dev/ttyUSB0"
    baud_rate: 19200
    
  partitions:
    - partition: 1
      name: "Main"
      
  zones:
    - zone: 1
      name: "Front Door"
      type: "door"
      entry_zone: true
      room: "room-hallway"
      
    - zone: 2
      name: "Back Door"
      type: "door"
      room: "room-kitchen"
      
    - zone: 3
      name: "PIR Hallway"
      type: "pir"
      room: "room-hallway"
      
    - zone: 4
      name: "PIR Living Room"
      type: "pir"
      room: "room-living"
      
  modes:
    home: "disarm"
    away: "arm_away"
    night: "arm_stay"
```

### Residential: Ajax (Cloud)

```yaml
security:
  panel:
    manufacturer: "ajax"
    model: "Hub 2 Plus"
    
  connection:
    type: "api"
    api:
      type: "ajax_cloud"
      hub_id: "ABCD1234"
      token_env: "AJAX_API_TOKEN"
      
  polling:
    interval_ms: 5000                 # Cloud polling interval
```

### Commercial: Honeywell Galaxy

```yaml
security:
  panel:
    manufacturer: "honeywell"
    model: "Galaxy Dimension GD-520"
    
  connection:
    type: "ip"
    host: "192.168.1.50"
    port: 10001
    
  partitions:
    - partition: 1
      name: "Reception & Offices"
    - partition: 2
      name: "Server Room"
    - partition: 3
      name: "Warehouse"
      
  users:
    - code: 1
      name: "Manager"
      partitions: [1, 2, 3]
    - code: 2
      name: "Office Staff"
      partitions: [1]
    - code: 3
      name: "Warehouse Staff"
      partitions: [3]
```

---

## Best Practices

### Do's

1. **Test integration thoroughly** — Verify all states report correctly
2. **Use separate user codes** — Track who armed/disarmed
3. **Configure notifications** — Alert on all alarm events
4. **Document zone mapping** — Clear mapping between panel zones and rooms
5. **Test fail-safe** — Verify panel works when Gray Logic is offline
6. **Regular testing** — Test sensors and integration periodically

### Don'ts

1. **Don't rely solely on app** — Ensure physical keypads work
2. **Don't auto-bypass zones** — Manual bypass only, with logging
3. **Don't suppress alarms** — Alert immediately on any alarm
4. **Don't store PINs in plain text** — Use secure credential storage
5. **Don't log PIN codes** — Log user ID, not the actual PIN
6. **Don't disable logging** — Audit trail is mandatory

---

## Audit Logging

All security actions are logged with full detail:

```yaml
SecurityAuditLog:
  id: uuid
  timestamp: timestamp
  
  # Action
  action: string                      # arm, disarm, bypass, acknowledge
  success: boolean
  
  # Context
  panel_id: uuid
  partition_id: uuid | null
  zone_id: uuid | null
  
  # User
  user_id: uuid | null                # Gray Logic user (if via app)
  panel_user_code: integer | null     # Panel user code
  panel_user_name: string | null      # Panel user name
  
  # Method
  method: string                      # keypad, fob, app, schedule, api
  source_ip: string | null            # If remote
  
  # Details
  details: object                     # Additional context
```

**Retention:** Security audit logs are retained for minimum 2 years.

---

## Related Documents

- [Principles](../overview/principles.md) — Safety and security philosophy
- [Data Model: Entities](../data-model/entities.md) — Security device types
- [Access Control Integration](../integration/access-control.md) — Door access coordination
- [CCTV Integration](../integration/cctv.md) — Camera integration
- [Fire Alarm Integration](../integration/fire-alarm.md) — Emergency coordination
- [PHM Specification](../intelligence/phm.md) — Health monitoring framework
- [Automation Specification](../automation/automation.md) — Event-driven triggers
