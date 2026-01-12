---
title: Access Control Integration
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/principles.md
  - data-model/entities.md
---

# Access Control Integration

This document specifies how Gray Logic integrates with access control systems for doors, gates, barriers, and intercom/entry systems.

---

## Overview

Access control integration enables:
- Unified view of door status and access events
- Automation triggers based on access events
- Intercom integration with video and audio
- Presence detection via card swipes
- Emergency egress coordination

### Integration Levels

| Level | Description | Gray Logic Role |
|-------|-------------|-----------------|
| **Monitoring** | See door status, access events | Read-only |
| **Triggering** | Trigger automation on events | Event consumer |
| **Control** | Remote unlock via Gray Logic | Control with restrictions |
| **Primary** | Gray Logic as access controller | Full control (rare) |

Most installations use **Monitoring + Triggering + Limited Control**.

---

## Safety Principles

### Emergency Egress

> **HARD RULE**: Emergency egress must **NEVER** be impeded by Gray Logic.

- Emergency exit doors must have mechanical release (crash bars, break glass)
- Fire alarm integration should trigger unlock sequence
- Power failure = doors fail to safe state (typically unlocked)
- Gray Logic cannot prevent emergency egress

### Security Boundaries

- Remote unlock requires PIN + confirmation
- Audit logging for all access events
- Role-based access to control functions
- Physical override always available

---

## Architecture

### Typical Integration

```
┌─────────────────────────────────────────────────────────────────────┐
│               ACCESS CONTROL SYSTEM                                  │
│                                                                      │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │
│  │ Controller  │    │ Controller  │    │ Controller  │             │
│  │  (Door 1)   │    │  (Door 2)   │    │  (Gate)     │             │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘             │
│         │                  │                  │                     │
│         └──────────────────┼──────────────────┘                     │
│                            │                                        │
│                    ┌───────▼───────┐                               │
│                    │  Head-End /   │                               │
│                    │   Software    │                               │
│                    └───────┬───────┘                               │
│                            │                                        │
└────────────────────────────┼────────────────────────────────────────┘
                             │ API / Integration
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                    GRAY LOGIC CORE                                   │
│                                                                      │
│  Access Devices → State Manager → Automation Triggers               │
│                → Event Router  → Notifications                      │
│                → Audit Logging → Presence Detection                 │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Direct Door Control (Where Appropriate)

For simpler installations (residential, small commercial):

```
┌───────────────────────────────────────────────────────────────┐
│                    DOOR HARDWARE                               │
│                                                                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐       │
│  │ Card Reader │    │ Door Contact│    │ Electric    │       │
│  │ (Wiegand)   │    │  (N/C)      │    │ Strike/Lock │       │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘       │
│         │                  │                  │               │
└─────────┼──────────────────┼──────────────────┼───────────────┘
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    KNX/MODBUS INTERFACE                          │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │ Wiegand to  │    │ Binary      │    │ Relay       │         │
│  │ KNX Gateway │    │ Input       │    │ Output      │         │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘         │
│         │                  │                  │                 │
└─────────┼──────────────────┼──────────────────┼─────────────────┘
          │                  │                  │
          └──────────────────┼──────────────────┘
                             │
                    ┌────────▼────────┐
                    │ GRAY LOGIC CORE │
                    └─────────────────┘
```

---

## Device Types

### Card Reader

```yaml
device:
  id: "reader-main-entry"
  name: "Main Entry Card Reader"
  type: "card_reader"
  domain: "access"
  
  protocol: "knx"                   # Or dedicated access protocol
  address:
    credential: "11/0/1"            # Card number received
    tamper: "11/0/2"                # Tamper alarm
    
  config:
    reader_type: "wiegand_26"       # wiegand_26 | wiegand_34 | osdp
    timeout_seconds: 5              # Card presentation timeout
    led_feedback: true
    beep_on_read: true
```

### Door Controller

```yaml
device:
  id: "door-main-entry"
  name: "Main Entry Door"
  type: "door_controller"
  domain: "access"
  
  protocol: "knx"
  address:
    lock_command: "11/1/1"          # Output to lock
    lock_feedback: "11/1/2"         # Lock state feedback
    door_contact: "11/1/3"          # Door open/closed
    rex_button: "11/1/4"            # Request to exit
    
  config:
    lock_type: "fail_secure"        # fail_secure | fail_safe
    unlock_duration_seconds: 5
    door_held_alarm_seconds: 30
    door_forced_alarm: true
    
  # Associated reader
  reader_id: "reader-main-entry"
```

### Intercom/Door Station

```yaml
device:
  id: "intercom-front-door"
  name: "Front Door Intercom"
  type: "door_station"
  domain: "access"
  
  protocol: "sip"
  address:
    sip_uri: "sip:frontdoor@192.168.1.60"
    
  config:
    video: true
    resolution: "1080p"
    camera_view: "wide"             # For package delivery view
    
    # SIP integration
    sip:
      server: "192.168.1.50"
      extension: "100"
      
    # Relay outputs
    relays:
      - id: 1
        name: "Door Strike"
        door_id: "door-main-entry"
      - id: 2
        name: "Gate"
        door_id: "gate-driveway"
        
  # Ring targets
  call_targets:
    - device: "panel-hallway"       # Wall panel
    - device: "panel-kitchen"
    - app: "all_residents"          # Mobile app
```

### Gate/Barrier

```yaml
device:
  id: "gate-driveway"
  name: "Driveway Gate"
  type: "door_controller"
  domain: "access"
  
  protocol: "modbus_tcp"
  address:
    host: "192.168.1.70"
    port: 502
    
  config:
    type: "sliding_gate"            # sliding_gate | swing_gate | barrier
    open_time_seconds: 15           # Full open cycle
    auto_close: true
    auto_close_delay_seconds: 30
    safety_edge: true               # Obstruction detection
    
  # Vehicle detection
  vehicle_detector:
    type: "loop"                    # loop | radar | camera
    address: "11/2/1"
```

---

## State Model

### Door State

```yaml
DoorState:
  # Lock status
  locked: boolean                   # Currently locked
  lock_mode: enum                   # normal | passage | lockdown
  
  # Door position
  door_open: boolean                # Contact state
  door_held: boolean                # Open too long
  door_forced: boolean              # Forced entry alarm
  
  # Last access
  last_access:
    timestamp: timestamp
    credential_id: string
    user_name: string
    result: enum                    # granted | denied | unknown_card
    
  # Alarms
  tamper_alarm: boolean
  communication_fault: boolean
```

### Intercom State

```yaml
IntercomState:
  # Call status
  call_active: boolean
  call_direction: enum              # incoming | outgoing | none
  call_source: string               # SIP URI or device ID
  call_duration_seconds: integer
  
  # Video
  video_available: boolean
  video_streaming: boolean
  
  # Door control
  last_unlock:
    timestamp: timestamp
    relay: integer
    triggered_by: string            # User or automation
```

---

## Access Events

### Event Types

| Event | Description | Automation Use |
|-------|-------------|----------------|
| `access_granted` | Valid credential, door unlocked | Welcome automation |
| `access_denied` | Invalid/expired credential | Security alert |
| `door_opened` | Door contact changed to open | Entry detection |
| `door_closed` | Door contact changed to closed | Arm delay reset |
| `door_held` | Door open too long | Alert occupant |
| `door_forced` | Door opened without unlock | Security alarm |
| `rex_activated` | Request to exit pressed | Normal egress |
| `intercom_ring` | Doorbell/call initiated | Notify residents |
| `intercom_answered` | Call answered | - |
| `remote_unlock` | Door unlocked via app/UI | Audit log |

### Event Structure

```yaml
AccessEvent:
  timestamp: timestamp
  event_type: string
  device_id: string
  door_name: string
  
  # Credential info (if applicable)
  credential:
    type: enum                      # card | pin | biometric | mobile
    id: string                      # Card number, etc.
    user_id: string | null
    user_name: string | null
    
  # Result
  result: enum                      # granted | denied | error
  reason: string | null             # "Card expired", "Unknown card", etc.
  
  # Location context
  room_id: string
  area_id: string
```

---

## Automation

### Welcome Home Automation

```yaml
automation:
  id: "welcome-home-front-door"
  name: "Welcome Home - Front Door"
  
  trigger:
    type: "access_event"
    event_type: "access_granted"
    device_id: "door-main-entry"
    
  conditions:
    - type: "time"
      operator: "between"
      value: ["17:00", "23:00"]
    - type: "mode"
      operator: "eq"
      value: "away"
      
  actions:
    # Change mode
    - command: "set_mode"
      mode: "home"
      
    # Hallway lights
    - target:
        device_id: "light-hallway"
      command: "turn_on"
      parameters:
        brightness: 100
        transition_ms: 0
        
    # Disarm security (if armed away)
    - target:
        device_id: "alarm-panel"
      command: "disarm"
      parameters:
        # PIN still required at panel for full disarm
        auto_disarm: true           # Only if user has permission
        
    # Climate
    - target:
        zone_id: "zone-ground-floor"
      command: "cancel_setback"
```

### Visitor Arrival

```yaml
automation:
  id: "visitor-intercom-ring"
  name: "Visitor at Front Door"
  
  trigger:
    type: "device_event"
    device_id: "intercom-front-door"
    event: "ring"
    
  actions:
    # Pause media
    - target:
        type: "site"
        domain: "audio"
      command: "pause"
      
    # Show intercom on wall panels
    - target:
        type: "site"
        device_type: "wall_panel"
      command: "show_intercom"
      parameters:
        intercom_id: "intercom-front-door"
        
    # Show on TV (if watching)
    - target:
        device_id: "tv-living-room"
      command: "show_pip"
      parameters:
        source: "intercom-front-door"
        position: "top_right"
        
    # Mobile notification
    - type: "notification"
      title: "🚪 Someone at the door"
      message: "Tap to view and answer"
      image: "intercom-front-door/snapshot"
      actions:
        - id: "answer"
          title: "Answer"
        - id: "unlock"
          title: "Unlock Door"
```

### Gate Vehicle Detection

```yaml
automation:
  id: "gate-vehicle-approach"
  name: "Vehicle Approaching Gate"
  
  trigger:
    type: "device_state"
    device_id: "gate-driveway"
    condition:
      property: "vehicle_detected"
      operator: "eq"
      value: true
      
  conditions:
    - type: "mode"
      operator: "in"
      value: ["home", "away"]
      
  actions:
    # Notify resident
    - type: "notification"
      title: "🚗 Vehicle at gate"
      message: "Vehicle detected at driveway gate"
      
    # Don't auto-open - require explicit action
```

### Emergency Egress (Fire Alarm)

```yaml
automation:
  id: "fire-alarm-unlock-doors"
  name: "Fire Alarm - Unlock Egress Doors"
  
  trigger:
    type: "device_state"
    device_id: "fire-panel-main"
    condition:
      property: "fire_active"
      operator: "eq"
      value: true
      
  actions:
    # Unlock all egress doors
    - target:
        type: "device_group"
        group: "egress_doors"
      command: "unlock"
      parameters:
        mode: "passage"             # Stay unlocked
        reason: "fire_alarm"
        
    # Log for audit
    - type: "audit_log"
      category: "safety"
      message: "Egress doors unlocked due to fire alarm"
```

---

## Remote Door Control

### Security Requirements

Remote unlock requires additional verification:

```yaml
remote_unlock:
  device_id: "door-main-entry"
  
  # Authentication requirements
  authentication:
    require_pin: true               # PIN in addition to app login
    require_2fa: false              # Optional second factor
    
  # Confirmation
  confirmation:
    required: true
    timeout_seconds: 10
    message: "Unlock front door for 5 seconds?"
    
  # Restrictions
  restrictions:
    allowed_roles: ["admin", "resident"]
    allowed_times: null             # Or restrict to certain hours
    require_video_view: true        # Must view camera first
    
  # Audit
  audit:
    log_all_attempts: true
    include_user_id: true
    include_ip_address: true
    retention: "1_year"
```

### UI Flow

```
1. User taps "Unlock" on intercom view
2. App shows camera feed (required)
3. App prompts for PIN
4. App shows confirmation: "Unlock front door?"
5. User confirms
6. Door unlocks for 5 seconds
7. Event logged with user ID, timestamp, source IP
```

---

## Presence Detection

Card swipes can indicate presence for automation:

```yaml
presence_from_access:
  - door_id: "door-main-entry"
    direction: "in"                 # Entry = arriving
    presence_action: "set_home"
    users:
      - credential: "12345678"
        user_id: "user-darren"
      - credential: "87654321"
        user_id: "user-partner"
        
  - door_id: "door-garage-internal"
    direction: "in"                 # Coming from garage
    presence_action: "set_home"
    
  # Exit detection (if reader on inside)
  - door_id: "door-main-entry"
    direction: "out"
    presence_action: "check_empty"  # If last person, set away
```

---

## Intercom Integration

### SIP Configuration

```yaml
sip_integration:
  # Gray Logic SIP client
  client:
    enabled: true
    server: "192.168.1.50"
    port: 5060
    transport: "udp"
    
    # Registration for incoming calls
    register:
      enabled: true
      extension: "200"
      username: "graylogic"
      password: "${SIP_PASSWORD}"
      
  # Door stations
  door_stations:
    - device_id: "intercom-front-door"
      sip_uri: "sip:100@192.168.1.50"
      
    - device_id: "intercom-gate"
      sip_uri: "sip:101@192.168.1.50"
      
  # Internal endpoints (wall panels, app)
  endpoints:
    - device_id: "panel-hallway"
      sip_uri: "sip:201@192.168.1.50"
      
    - device_id: "panel-kitchen"
      sip_uri: "sip:202@192.168.1.50"
      
    - type: "mobile_app"
      sip_uri: "sip:300@192.168.1.50"
```

### Call Handling

```yaml
call_routing:
  # Incoming call from front door
  - source: "intercom-front-door"
    ring_targets:
      - "panel-hallway"
      - "panel-kitchen"
      - "panel-living"
      - "mobile_app"
    ring_timeout_seconds: 45
    
    # Video routing
    video:
      enabled: true
      show_on_panels: true
      show_on_tv_pip: true
      
    # If not answered
    no_answer:
      action: "voicemail"           # Or "ignore"
      message: "No one is available. Please leave a message."
```

---

## Residential Access Control

This section covers access control patterns specific to residential properties.

### Typical Residential Entry Points

| Entry Point | Hardware | Integration |
|-------------|----------|-------------|
| **Front door** | Video intercom + electric strike | SIP + KNX relay |
| **Side/rear door** | Keypad or card + electric strike | KNX binary input/output |
| **Garage door** | Sectional door motor | Modbus or relay |
| **Driveway gate** | Sliding/swing gate motor | Modbus or relay |
| **Pedestrian gate** | Electric lock | KNX relay |
| **Pool gate** | Self-closing + magnetic lock | Safety compliance |

### Video Intercom Configuration

Most residential installs use video intercoms at the main entrance:

```yaml
residential_intercom:
  device_id: "intercom-front-door"
  
  # Hardware examples: 2N, Doorbird, Hikvision, Akuvox
  hardware:
    manufacturer: "2n"
    model: "ip_verso"
    
  # SIP configuration
  sip:
    server: "192.168.1.50"          # Local Asterisk/FreePBX or Gray Logic SIP
    extension: "100"
    
  # Features
  features:
    video: true
    hd_audio: true
    camera_count: 1                 # Or 2 for wide + close-up
    night_vision: true
    motion_detection: true
    tamper_detection: true
    
  # Door release outputs
  relays:
    - id: 1
      name: "Front Door Strike"
      door_id: "door-front"
      pulse_duration_ms: 5000
      
    - id: 2
      name: "Gate Release"
      door_id: "gate-pedestrian"
      pulse_duration_ms: 3000
      
  # Call routing
  ring_targets:
    at_home:
      - "panel-hallway"
      - "panel-kitchen"
      - "mobile_app"
    away:
      - "mobile_app"                # Only mobile when away
      
  # Motion-triggered snapshot
  motion_snapshot:
    enabled: true
    store_days: 30
    notify: false                   # Or true for security alerts
```

### Keypad Entry

For side doors, garages, or additional family members:

```yaml
keypad_entry:
  device_id: "keypad-side-door"
  
  # Hardware: KNX keypad, standalone keypad with Wiegand output
  protocol: "knx"
  address:
    code_entered: "11/0/10"
    valid_signal: "11/0/11"
    
  # PIN codes
  codes:
    - code: "1234"
      name: "Family"
      user_ids: ["user-darren", "user-partner"]
      valid_always: true
      
    - code: "5678"
      name: "Cleaner"
      user_id: "user-cleaner"
      valid_times:
        days: ["monday", "thursday"]
        start: "09:00"
        end: "13:00"
        
    - code: "9999"
      name: "Temporary"
      valid_from: "2026-01-15T00:00:00Z"
      valid_until: "2026-01-20T23:59:59Z"
      single_use: false
      
  # Failed attempts
  security:
    max_attempts: 3
    lockout_minutes: 5
    notify_on_lockout: true
```

### Smart Lock Integration

For properties using smart locks (typically retrofit):

```yaml
smart_lock:
  device_id: "lock-front-door"
  
  # Integration via local API (not cloud)
  protocol: "http"
  address:
    host: "192.168.1.80"
    api_path: "/api/lock"
    
  # Or via Z-Wave/Zigbee bridge (if used)
  # protocol: "mqtt"
  # topic: "zigbee2mqtt/front_door_lock"
  
  # Capabilities
  capabilities:
    - lock_unlock
    - battery_status
    - auto_lock
    - one_time_codes
    
  # Auto-lock
  auto_lock:
    enabled: true
    delay_seconds: 30
    only_when_mode: ["away", "night"]
    
  # Battery monitoring
  battery:
    low_threshold: 20
    critical_threshold: 10
    notify: ["admin"]
```

### Garage Door Control

```yaml
garage_door:
  device_id: "garage-main"
  
  # Motor controller (Modbus or relay-based)
  protocol: "modbus_tcp"
  address:
    host: "192.168.1.75"
    port: 502
    
  # Door states
  states:
    position:
      register: 100
      values:
        0: "closed"
        1: "opening"
        2: "open"
        3: "closing"
        4: "stopped"
        
  # Commands
  commands:
    toggle:
      register: 200
      value: 1
    stop:
      register: 201
      value: 1
      
  # Safety
  safety:
    obstruction_sensor: true
    auto_close:
      enabled: true
      delay_minutes: 10
      only_when_mode: ["away"]      # Don't auto-close when home
      warn_before_seconds: 30
      
  # Vehicle detection
  vehicle_sensor:
    type: "ultrasonic"              # Or loop, or none
    device_id: "sensor-garage-vehicle"
```

### Driveway Gate

```yaml
driveway_gate:
  device_id: "gate-driveway"
  
  type: "sliding"                   # sliding | swing_single | swing_double
  
  # Motor controller
  protocol: "modbus_rtu"
  address:
    port: "/dev/ttyUSB0"
    baud: 9600
    unit_id: 1
    
  # States
  states:
    position:
      register: 0
      type: "input"
    moving:
      register: 1
      type: "input"
    obstacle:
      register: 2
      type: "input"
      
  # Commands
  commands:
    open:
      register: 0
      type: "coil"
    close:
      register: 1
      type: "coil"
    stop:
      register: 2
      type: "coil"
      
  # Entry methods
  entry_methods:
    # Intercom release
    - type: "intercom"
      intercom_id: "intercom-gate"
      relay: 1
      
    # Vehicle loop detector
    - type: "vehicle_loop"
      direction: "exit"             # Auto-open on exit only
      
    # Remote control (via relay input)
    - type: "remote"
      input_address: "11/2/5"
      
    # Number plate recognition (optional)
    - type: "anpr"
      camera_id: "camera-gate"
      plates:
        - plate: "AB12 CDE"
          name: "Family Car 1"
          auto_open: true
        - plate: "XY34 FGH"
          name: "Family Car 2"
          auto_open: true
          
  # Safety
  safety:
    photocell: true
    safety_edge: true
    pedestrian_warning: true        # Beep before closing
    auto_close:
      enabled: true
      delay_seconds: 30
```

### Guest Access

Managing temporary access for guests, tradespeople, deliveries:

```yaml
guest_access:
  # Temporary codes
  temporary_codes:
    - name: "Electrician Visit"
      code: "2468"
      valid_from: "2026-01-15T08:00:00Z"
      valid_until: "2026-01-15T18:00:00Z"
      doors: ["door-side"]
      notify_on_use: true
      single_use: false
      
    - name: "Package Delivery"
      code: "1357"
      valid_from: "2026-01-15T00:00:00Z"
      valid_until: "2026-01-15T23:59:59Z"
      doors: ["gate-pedestrian"]
      single_use: true              # Expires after first use
      
  # Guest WiFi integration (optional)
  wifi_voucher:
    enabled: true
    ssid: "GuestNetwork"
    duration_hours: 24
    send_with_code: true
    
  # Automation on guest arrival
  on_guest_arrival:
    - notify_residents: true
    - log_event: true
```

### Departure and Arrival Detection

Combining access events with presence for automation:

```yaml
presence_detection:
  methods:
    # Phone WiFi/Bluetooth (existing)
    - type: "phone"
      devices:
        - mac: "AA:BB:CC:DD:EE:01"
          user_id: "user-darren"
        - mac: "AA:BB:CC:DD:EE:02"
          user_id: "user-partner"
          
    # Door access events
    - type: "door_access"
      entries:
        - door_id: "door-front"
          direction: "in"
        - door_id: "door-garage-internal"
          direction: "in"
      exits:
        - door_id: "door-front"
          direction: "out"          # If outbound reader
          
    # Vehicle detection
    - type: "vehicle"
      sensors:
        - sensor_id: "sensor-garage-vehicle"
          arrives: "vehicle_present"
          departs: "vehicle_absent"
          
  # Departure automation
  on_last_person_leaves:
    delay_minutes: 5                # Debounce
    actions:
      - set_mode: "away"
      - check_doors_locked: true
      - check_garage_closed: true
      - notify_if_open: true
      
  # Arrival automation (first person)
  on_first_person_arrives:
    actions:
      - set_mode: "home"
      - cancel_climate_setback: true
```

### Holiday/Vacation Mode

```yaml
vacation_mode:
  triggers:
    - mode: "holiday"
    - calendar_event: "vacation"
    
  access_changes:
    # Disable temporary codes
    disable_temp_codes: true
    
    # Tighter security
    door_held_alarm_seconds: 15     # Shorter timeout
    
    # Enhanced notifications
    notify_all_access: true
    notify_all_motion: true
    
    # Auto-lock everything
    auto_lock_all: true
    
  # Trusted access only
  allowed_during_vacation:
    - user_id: "user-house-sitter"
      doors: ["door-front", "door-side"]
    - user_id: "user-gardener"
      doors: ["gate-pedestrian"]
      times:
        days: ["wednesday"]
        start: "10:00"
        end: "14:00"
```

### Pool/Spa Gate (Safety Compliance)

Pool gates have specific safety requirements:

```yaml
pool_gate:
  device_id: "gate-pool"
  
  # Safety compliance (AS1926.1 / ASTM F1908 etc.)
  compliance:
    self_closing: true              # Spring-loaded hinges
    self_latching: true             # Magnetic lock at height
    no_auto_open: true              # NEVER auto-open
    
  # Manual release only
  release:
    type: "manual"                  # Physical latch operation
    # Gray Logic monitors but does NOT control
    
  # Monitoring
  monitoring:
    door_contact: "11/3/1"
    alert_if_open_minutes: 5
    alert_recipients: ["all_residents"]
    
  # Alarm integration
  alarm_when:
    - mode: "away"
    - mode: "night"
    - children_home: true           # Optional presence flag
```

### Residential Commissioning Checklist

- [ ] Front door intercom tested (video, audio, door release)
- [ ] Mobile app receiving intercom calls
- [ ] Wall panels showing intercom video
- [ ] All PIN codes programmed and tested
- [ ] Temporary code expiry verified
- [ ] Garage door safety sensors working
- [ ] Garage auto-close timing appropriate
- [ ] Driveway gate safety edge functional
- [ ] Gate photocells working
- [ ] Vehicle detection (if fitted) calibrated
- [ ] Departure detection triggering Away mode
- [ ] Arrival detection cancelling Away mode
- [ ] Remote unlock security (PIN required) tested
- [ ] Fire alarm unlock integration verified
- [ ] All doors fail to correct state (secure vs safe)
- [ ] Guest access workflow demonstrated to owner

---

## Commercial Access Control

### Multi-Door Systems

```yaml
commercial_access:
  system:
    type: "integrated"              # Enterprise access control
    vendor: "paxton"                # Or genetec, lenel, etc.
    integration: "api"
    
  # API connection
  api:
    base_url: "https://access.local/api"
    auth_type: "oauth2"
    
  # Sync configuration
  sync:
    doors:
      enabled: true
      interval_seconds: 60
      
    events:
      enabled: true
      method: "webhook"             # Or polling
      webhook_path: "/api/access/events"
      
    users:
      enabled: false                # Don't sync user database
```

### Turnstile/Speed Gate

```yaml
device:
  id: "turnstile-reception"
  name: "Reception Turnstile"
  type: "turnstile"
  domain: "access"
  
  config:
    type: "speed_gate"
    direction: "bidirectional"
    tailgate_detection: true
    
  # People counting
  counting:
    enabled: true
    publish_to: "occupancy"         # Feed to occupancy system
```

### Visitor Management

```yaml
visitor_management:
  integration:
    type: "api"
    system: "envoy"                 # Or proxyclick, etc.
    
  # On visitor check-in
  on_checkin:
    - notify_host: true
    - unlock_turnstile: true
    - issue_temp_credential: true
    
  # Automation trigger
  automation:
    trigger_type: "visitor_arrival"
    data:
      visitor_name: true
      host_name: true
      company: true
```

---

## Commissioning Checklist

### Door Controllers

- [ ] Lock/unlock command verified
- [ ] Door contact status correct
- [ ] REX button functional
- [ ] Fail-safe/fail-secure mode correct
- [ ] Unlock duration appropriate
- [ ] Door held alarm timing set
- [ ] Emergency egress tested
- [ ] Fire alarm unlock integration tested

### Card Readers

- [ ] Reader communicating
- [ ] Test cards read correctly
- [ ] Unknown card handling correct
- [ ] LED/beep feedback working

### Intercoms

- [ ] SIP registration successful
- [ ] Ring targets configured
- [ ] Video streaming working
- [ ] Door release relay mapped
- [ ] Mobile app receiving calls
- [ ] Audio quality acceptable

### Integration

- [ ] Events flowing to Gray Logic
- [ ] Automation triggers working
- [ ] Audit logging verified
- [ ] Remote unlock security tested
- [ ] Presence detection working

---

## Related Documents

- [Principles](../overview/principles.md) — Safety and security philosophy
- [Entities](../data-model/entities.md) — Access control device types
- [Fire Alarm Integration](fire-alarm.md) — Emergency egress coordination
- [CCTV Integration](cctv.md) — Camera integration, doorbell cameras
- [Security Domain](../domains/security.md) — Alarm system integration (to be created)

