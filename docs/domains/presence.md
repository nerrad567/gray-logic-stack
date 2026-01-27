---
title: Presence Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - data-model/entities.md
  - automation/automation.md
---

# Presence Domain Specification

This document specifies how Gray Logic detects, tracks, and utilizes presence and occupancy information for automation and analytics.

---

## Overview

### Philosophy

Presence is the foundation of intelligent automation — knowing who is where enables:

| Capability | Benefit |
|------------|---------|
| **Away detection** | Security arm, energy savings |
| **Room awareness** | Follow-me lighting, climate |
| **Occupancy counting** | Commercial HVAC optimization |
| **Behavior learning** | Predictive automation |

### Design Principles

1. **Privacy-first** — All processing local, minimal data retention
2. **Multi-source** — Combine sensors for accuracy
3. **Graceful degradation** — Works with any sensor subset
4. **Non-intrusive** — No cameras for presence (see CCTV for security)
5. **Real-time** — Fast detection for responsive automation

### Presence Hierarchy

```
Site Level
├── Someone home / Nobody home
├── Which users are home
│
Room Level
├── Room occupied / Room vacant
├── Occupancy count (commercial)
│
Zone Level (optional)
├── Which area of room
└── Desk occupancy (commercial)
```

---

## Detection Methods

### Method Comparison

| Method | Range | Accuracy | Privacy | Battery |
|--------|-------|----------|---------|---------|
| **WiFi** | Site | Medium | Low concern | N/A |
| **Bluetooth** | Room | High | Low concern | Phone |
| **PIR** | Room | Medium | None | Low |
| **mmWave radar** | Room/zone | High | None | N/A |
| **Ultrasonic** | Zone | High | None | N/A |
| **CO2** | Room | Low (delayed) | None | N/A |
| **Beacon** | Room | High | Low concern | Beacon |

### WiFi-Based Detection

Detect phone presence on network:

```yaml
WiFiPresence:
  enabled: true
  
  # Detection method
  method: "router_api" | "arp_scan" | "dhcp_lease"
  
  # Router integration
  router:
    type: "unifi" | "openwrt" | "mikrotik" | "generic"
    host: "192.168.1.1"
    api_key_env: "ROUTER_API_KEY"
    
  # Tracked devices
  devices:
    - mac: "AA:BB:CC:DD:EE:FF"
      user_id: "user-john"
      name: "John's iPhone"
      
    - mac: "11:22:33:44:55:66"
      user_id: "user-jane"
      name: "Jane's iPhone"
      
  # Timing
  config:
    poll_interval_seconds: 30
    consider_away_after_minutes: 5
    consider_home_after_seconds: 30
```

### Bluetooth-Based Detection

Room-level presence via Bluetooth beacons or phone scanning:

```yaml
BluetoothPresence:
  enabled: true
  
  # BLE scanners (one per room/area)
  scanners:
    - id: "ble-scanner-living"
      room_id: "room-living"
      device_id: "esp32-living"       # ESP32 with BLE
      
    - id: "ble-scanner-kitchen"
      room_id: "room-kitchen"
      device_id: "esp32-kitchen"
      
  # Track phone BLE
  phones:
    - identifier: "irk:XXXX"          # iOS Identity Resolving Key
      user_id: "user-john"
      
  # Track beacons
  beacons:
    - uuid: "FDA50693-A4E2-4FB1-AFCF-C6EB07647825"
      major: 1
      minor: 1
      user_id: "user-john"
      name: "John's Tile"
      
  # RSSI thresholds
  config:
    near_threshold_dbm: -60
    far_threshold_dbm: -85
    smoothing_samples: 3
```

### PIR Motion Sensors

Motion-based occupancy:

```yaml
PIRPresence:
  sensors:
    - device_id: "pir-living"
      room_id: "room-living"
      coverage: "main_area"
      
    - device_id: "pir-kitchen"
      room_id: "room-kitchen"
      coverage: "full_room"
      
  config:
    # Motion indicates presence
    motion_triggers_occupied: true
    
    # No motion indicates vacant after timeout
    vacant_timeout_minutes: 15
    
    # Per-room timeout overrides
    room_overrides:
      - room_id: "room-bedroom"
        vacant_timeout_minutes: 60    # Longer for bedroom
        
      - room_id: "room-bathroom"
        vacant_timeout_minutes: 30
```

### mmWave Radar

High-accuracy presence including stationary detection:

```yaml
RadarPresence:
  sensors:
    - device_id: "radar-living"
      room_id: "room-living"
      manufacturer: "aqara"
      model: "fp2"
      
      # Zone detection (if supported)
      zones:
        - id: "zone-sofa"
          name: "Sofa Area"
        - id: "zone-desk"
          name: "Desk Area"
          
  config:
    # Sensitivity
    sensitivity: "medium"             # low, medium, high
    
    # Detection range
    max_range_m: 5
    
    # Stationary detection
    stationary_detection: true
    stationary_timeout_seconds: 30
```

### CO2-Based Occupancy

Infer occupancy from CO2 levels:

```yaml
CO2Presence:
  sensors:
    - device_id: "co2-living"
      room_id: "room-living"
      
  config:
    # Baseline (empty room)
    baseline_ppm: 450
    
    # Per-person increase
    per_person_ppm: 30
    
    # Thresholds
    occupied_above_ppm: 500
    
    # Delay (CO2 changes slowly)
    detection_delay_minutes: 10
    clearing_delay_minutes: 30
    
  # Useful for HVAC optimization, not real-time presence
  use_for:
    - "hvac_demand_control"
    - "occupancy_estimation"
```

---

## Data Model

### User Presence

```yaml
UserPresence:
  user_id: uuid
  name: string
  
  # Site-level presence
  home: boolean
  last_seen: timestamp
  last_location: string | null        # Room name
  
  # Detection sources
  sources:
    wifi:
      detected: boolean
      last_seen: timestamp
    bluetooth:
      detected: boolean
      room_id: uuid | null
      rssi: integer | null
    
  # Arrival/departure
  last_arrival: timestamp | null
  last_departure: timestamp | null
  
  # Duration tracking
  home_today_minutes: integer
  home_this_week_minutes: integer
```

### Room Occupancy

```yaml
RoomOccupancy:
  room_id: uuid
  name: string
  
  # Occupancy state
  occupied: boolean
  occupancy_count: integer | null     # If counting supported
  
  # Timing
  last_motion: timestamp | null
  occupied_since: timestamp | null
  vacant_since: timestamp | null
  
  # Users present (if known)
  users_present: [uuid]
  
  # Detection sources
  sources:
    pir: boolean | null
    radar: boolean | null
    co2: boolean | null
    bluetooth: [uuid] | null          # User IDs detected
    
  # Zone occupancy (if supported)
  zones:
    - zone_id: "zone-sofa"
      occupied: true
    - zone_id: "zone-desk"
      occupied: false
```

### Site Occupancy

```yaml
SiteOccupancy:
  site_id: uuid
  
  # Overall state
  anyone_home: boolean
  
  # User summary
  users_home: [uuid]
  users_away: [uuid]
  
  # Occupancy count (if tracked)
  total_occupants: integer | null
  
  # Last change
  last_arrival: timestamp | null
  last_departure: timestamp | null
  
  # Mode suggestion
  suggested_mode: "home" | "away" | null
```

---

## Occupancy Logic

### Multi-Source Fusion

Combine multiple sources for accuracy:

```yaml
OccupancyFusion:
  room_id: "room-living"
  
  sources:
    - type: "pir"
      weight: 0.3
      
    - type: "radar"
      weight: 0.5
      
    - type: "bluetooth"
      weight: 0.4
      
  logic:
    # Occupied if any high-confidence source
    occupied_if:
      - source: "radar"
        condition: "detected"
        
      - source: "pir"
        condition: "motion_within"
        minutes: 5
        
      - source: "bluetooth"
        condition: "any_user"
        
    # Vacant requires all sources agree
    vacant_if:
      - source: "radar"
        condition: "not_detected"
        for_minutes: 5
        
      - source: "pir"
        condition: "no_motion"
        for_minutes: 15
```

### Arrival Detection

```yaml
ArrivalDetection:
  methods:
    # WiFi first arrival
    - source: "wifi"
      triggers: "first_user_home"
      
    # Bluetooth for room arrival
    - source: "bluetooth"
      triggers: "user_in_room"
      
    # PIR/radar for room arrival
    - source: "motion"
      triggers: "room_occupied"
      
  # Geofencing (optional)
  geofence:
    enabled: true
    app: "home_assistant_companion"   # Or native app
    radius_m: 200
    
    triggers:
      entering: "pre_arrival"         # Start warm-up
      exiting: "departure_likely"
```

### Departure Detection

```yaml
DepartureDetection:
  # WiFi gone = likely left
  wifi:
    away_after_minutes: 5
    
  # Confirm with no motion
  motion:
    no_motion_minutes: 15
    
  # Combined logic
  logic:
    type: "all"
    conditions:
      - wifi_away: true
        for_minutes: 5
      - all_rooms_vacant: true
        for_minutes: 10
```

---

## Automation Integration

### Mode Triggers

```yaml
triggers:
  # Last person left → Away mode
  - type: "presence_changed"
    condition: "last_person_left"
    execute:
      type: "mode"
      mode: "away"
      
  # First person home → Home mode
  - type: "presence_changed"
    condition: "first_person_arrived"
    execute:
      type: "mode"
      mode: "home"
      
  # Specific user arrived
  - type: "user_arrived"
    user_id: "user-john"
    execute:
      type: "scene"
      scene_id: "johns-welcome"
```

### Room-Level Triggers

```yaml
triggers:
  # Room occupied → Lighting
  - type: "room_occupancy_changed"
    room_id: "room-living"
    condition: "occupied"
    execute:
      - domain: "lighting"
        scope: "room-living"
        command: "scene"
        parameters:
          scene: "default"
    conditions:
      - type: "sun"
        operator: "below_horizon"
        
  # Room vacant → Lights off
  - type: "room_occupancy_changed"
    room_id: "room-living"
    condition: "vacant"
    execute:
      - domain: "lighting"
        scope: "room-living"
        command: "off"
        
  # Room occupied → HVAC
  - type: "room_occupancy_changed"
    room_id: "room-office"
    condition: "occupied"
    execute:
      - domain: "climate"
        zone_id: "zone-office"
        command: "set_mode"
        parameters:
          mode: "comfort"
```

### Follow-Me Automation

```yaml
FollowMe:
  # Lighting follows user between rooms
  lighting:
    enabled: true
    
    behavior:
      - previous_room: "off_after_minutes"
        minutes: 2
        
      - new_room: "scene"
        scene: "default"
        
    excluded_rooms:
      - "room-bedroom"                # Don't auto-light at night
      
  # Audio follows user
  audio:
    enabled: true
    
    behavior:
      transfer_playback: true
      fade_duration_seconds: 3
      
  # Climate follows user
  climate:
    enabled: true
    
    behavior:
      occupied_setpoint: 21
      unoccupied_setpoint: 18
      preheat_minutes: 10
```

---

## Commercial Occupancy

### Desk/Workstation Detection

```yaml
DeskOccupancy:
  workstations:
    - id: "desk-a1"
      name: "Desk A1"
      sensors:
        - type: "pir_under_desk"
          device_id: "pir-desk-a1"
        - type: "power_monitor"       # Computer power
          device_id: "power-desk-a1"
          
  config:
    occupied_if:
      - motion_detected: true
      - power_above_w: 50             # Computer on
      
    vacant_after_minutes: 30
```

### Room/Zone Counting

```yaml
PeopleCount:
  zones:
    - id: "zone-open-office"
      name: "Open Office"
      sensors:
        - type: "thermal_counter"
          device_id: "counter-entry-1"
          direction: "in_out"
          
      capacity: 50
      
  # HVAC integration
  hvac:
    demand_ventilation: true
    cfm_per_person: 20
    min_occupancy_for_hvac: 1
    
  # Space utilization
  reporting:
    track_utilization: true
    peak_hours: ["09:00", "17:00"]
```

### Meeting Room Occupancy

```yaml
MeetingRoomOccupancy:
  rooms:
    - id: "room-meeting-1"
      name: "Meeting Room 1"
      capacity: 8
      
      sensors:
        - type: "radar"
          device_id: "radar-meeting-1"
          count_capable: true
          
      calendar:
        integration: "microsoft_365"
        room_email: "meeting1@company.com"
        
      automation:
        # Book started but no one arrived
        no_show_minutes: 10
        no_show_action: "release_room"
        
        # Room in use without booking
        walk_in_detection: true
        walk_in_action: "create_booking"
```

---

## Privacy

### Data Minimization

```yaml
Privacy:
  # What we store
  storage:
    user_location:
      resolution: "room"              # Not zone/coordinates
      retention_hours: 24
      
    occupancy_history:
      resolution: "occupied/vacant"
      retention_days: 30
      
    # Never store
    prohibited:
      - "video"
      - "audio"
      - "exact_coordinates"
      - "movement_patterns"
      
  # User consent
  consent:
    require_opt_in: true
    allow_opt_out: true
    anonymous_mode: true              # Track occupancy, not identity
```

### Access Control

```yaml
PresenceAccess:
  # Who can see what
  permissions:
    - role: "family_member"
      can_see: ["own_presence", "home_status"]
      
    - role: "admin"
      can_see: ["all_presence", "history"]
      
    - role: "guest"
      can_see: ["none"]
      
  # External access
  external:
    expose_to_api: false              # By default
    expose_home_status_only: true
```

---

## MQTT Topics

### User Presence

```yaml
topic: graylogic/presence/user/{user_id}/state
payload:
  user_id: "user-john"
  timestamp: "2026-01-12T14:30:00Z"
  home: true
  location: "room-living"
  sources:
    wifi: true
    bluetooth: true
```

### Room Occupancy

```yaml
topic: graylogic/presence/room/{room_id}/state
payload:
  room_id: "room-living"
  timestamp: "2026-01-12T14:30:00Z"
  occupied: true
  occupancy_count: 2
  users: ["user-john", "user-jane"]
  last_motion: "2026-01-12T14:29:30Z"
```

### Site Status

```yaml
topic: graylogic/presence/site/state
payload:
  site_id: "site-home"
  timestamp: "2026-01-12T14:30:00Z"
  anyone_home: true
  users_home: ["user-john", "user-jane"]
  users_away: ["user-guest"]
```

---

## Configuration Examples

### Residential: Basic

```yaml
presence:
  # Site-level via WiFi
  wifi:
    enabled: true
    router_type: "unifi"
    devices:
      - mac: "AA:BB:CC:DD:EE:FF"
        user: "john"
      - mac: "11:22:33:44:55:66"
        user: "jane"
        
  # Room-level via PIR
  rooms:
    - room_id: "room-living"
      sensor: "pir-living"
      vacant_timeout_minutes: 15
      
    - room_id: "room-kitchen"
      sensor: "pir-kitchen"
      vacant_timeout_minutes: 10
```

### Residential: Advanced

```yaml
presence:
  # Multi-source
  wifi:
    enabled: true
    router_type: "openwrt"
    
  bluetooth:
    enabled: true
    scanners:
      - room: "living"
        device: "esp32-living"
      - room: "kitchen"
        device: "esp32-kitchen"
      - room: "bedroom"
        device: "esp32-bedroom"
        
  radar:
    sensors:
      - room: "living"
        device: "radar-living"
      - room: "office"
        device: "radar-office"
        zones: ["desk", "reading"]
        
  # User tracking
  users:
    - id: "john"
      wifi_mac: "AA:BB:CC:DD:EE:FF"
      bluetooth_irk: "xxxxx"
      
  # Automation
  automation:
    away_mode_after_minutes: 10
    follow_me_lighting: true
```

### Commercial: Office

```yaml
presence:
  # Per-floor counting
  floors:
    - id: "floor-1"
      entry_counters:
        - device: "counter-entry-1"
          
  # Meeting rooms
  meeting_rooms:
    - room: "meeting-1"
      radar: "radar-meeting-1"
      calendar: "meeting1@company.com"
      
    - room: "meeting-2"
      radar: "radar-meeting-2"
      calendar: "meeting2@company.com"
      
  # Open office zones
  zones:
    - id: "zone-sales"
      co2_sensor: "co2-sales"
      
    - id: "zone-engineering"
      co2_sensor: "co2-engineering"
      
  # Integration
  integration:
    hvac_demand_control: true
    lighting_scheduling: true
    space_utilization_reporting: true
```

---

## Best Practices

### Do's

1. **Multiple sources** — Combine for reliability
2. **Appropriate timeouts** — Balance responsiveness and false triggers
3. **Room-specific tuning** — Bedroom needs longer timeout than hallway
4. **Test thoroughly** — Walk through all scenarios
5. **Privacy by design** — Minimize stored data
6. **User opt-out** — Respect preferences

### Don'ts

1. **Don't use cameras for presence** — Privacy concern (use CCTV for security only)
2. **Don't rely on single source** — WiFi alone has latency
3. **Don't set timeouts too short** — Leads to light flickering
4. **Don't track guests without consent** — Privacy issue
5. **Don't store movement history** — Unnecessary for automation

---

## Related Documents

- [Lighting Domain](lighting.md) — Presence-based lighting
- [Climate Domain](climate.md) — Occupancy-based HVAC
- [Security Domain](security.md) — Away mode integration
- [Energy Domain](energy.md) — Occupancy-based energy saving
- [Automation Specification](../automation/automation.md) — Presence triggers
- [Data Model: Entities](../data-model/entities.md) — User and device entities
