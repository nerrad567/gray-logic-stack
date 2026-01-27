---
title: Blinds Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - data-model/entities.md
  - protocols/knx.md
  - domains/climate.md
---

# Blinds Domain Specification

This document specifies all shading and blind control features in Gray Logic, including roller blinds, venetian blinds, curtains, awnings, and louvres.

---

## Overview

Shading control provides comfort, energy efficiency, security, and privacy. Gray Logic coordinates blinds with lighting, climate, and security systems.

### Design Principles

1. **Manual override** — Physical controls always work via KNX
2. **Safety first** — Wind/rain protection cannot be overridden
3. **Energy coordination** — Integrate with climate for solar gain control
4. **Privacy protection** — Night mode closes blinds
5. **Smooth operation** — Minimize motor wear with intelligent control

### Device Types

| Type | Capabilities | Examples |
|------|-------------|----------|
| `blind_switch` | Open/close/stop | Simple motor relay |
| `blind_position` | Position 0-100% | Roller blind, screen |
| `blind_tilt` | Position + slat tilt | Venetian blind |
| `curtain` | Position 0-100% | Motorized curtain track |
| `awning` | Extended/retracted | Exterior awning |
| `louvre` | Blade angle | Adjustable louvres |

---

## Feature Matrix

### Core Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Position Control** | Must-have | Set blind to specific position (0-100%) |
| **Tilt Control** | Must-have | Slat angle for venetian blinds |
| **Open/Close/Stop** | Must-have | Basic motor control |
| **Preset Positions** | Must-have | Favorite position recall |
| **Status Feedback** | Must-have | Current position, tilt, moving |
| **Group Control** | Must-have | Control multiple blinds together |

### Intelligent Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Sun Tracking** | Should-have | Follow sun to optimize shading |
| **Solar Gain Control** | Should-have | Coordinate with climate |
| **Glare Protection** | Should-have | Automatic anti-glare |
| **Privacy Mode** | Should-have | Close at sunset/night |
| **Scene Integration** | Must-have | Include in scenes |
| **Wake/Sleep Automation** | Should-have | Gradual open/close |

### Safety Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Wind Protection** | Must-have | Retract awnings in high wind |
| **Rain Protection** | Should-have | Retract exterior blinds |
| **Frost Protection** | Should-have | Prevent freeze damage |
| **Obstruction Detection** | Should-have | Stop on obstruction |
| **Manual Override** | Must-have | Physical control always works |

---

## State Model

### Blind State

```yaml
BlindState:
  # Position (0 = fully open, 100 = fully closed)
  position: integer                 # 0-100%
  
  # Tilt for venetian blinds (0 = horizontal, 100 = fully closed)
  tilt: integer | null              # 0-100%, null if not applicable
  
  # Movement
  moving: boolean
  direction: enum | null            # up | down | null (stopped)
  
  # Calculated
  is_open: boolean                  # position == 0
  is_closed: boolean                # position == 100
  
  # Actuator
  motor_running: boolean
```

**Position Convention:**
- `0` = Fully open (raised, retracted)
- `100` = Fully closed (lowered, extended)
- This matches "percentage closed" convention

**Tilt Convention (Venetian):**
- `0` = Slats horizontal (full view)
- `50` = Slats at 45° angle
- `100` = Slats fully closed (overlapping)

---

## Device Configuration

### Roller Blind

```yaml
blinds:
  - id: "blind-living-1"
    name: "Living Room Blind 1"
    room_id: "room-living"
    type: "blind_position"
    
    # Protocol addressing
    protocol: "knx"
    address:
      move: "3/1/1"                # Up/Down/Stop
      position: "3/1/2"            # Position setpoint
      position_status: "3/1/3"     # Position feedback
      
    # Capabilities
    capabilities:
      - "position"
      
    # Timing
    timing:
      full_travel_up_seconds: 30
      full_travel_down_seconds: 28
      
    # Orientation (for sun calculations)
    orientation:
      facade: "south"
      azimuth: 180                 # degrees from north
      window_height: 2.2           # meters
      
    # Limits
    limits:
      min_position: 0
      max_position: 100
```

### Venetian Blind

```yaml
blinds:
  - id: "blind-office-1"
    name: "Office Venetian"
    room_id: "room-office"
    type: "blind_tilt"
    
    protocol: "knx"
    address:
      move: "3/2/1"
      stop: "3/2/2"
      position: "3/2/3"
      position_status: "3/2/4"
      tilt: "3/2/5"
      tilt_status: "3/2/6"
      
    capabilities:
      - "position"
      - "tilt"
      
    timing:
      full_travel_up_seconds: 45
      full_travel_down_seconds: 42
      full_tilt_seconds: 3
      
    orientation:
      facade: "east"
      azimuth: 90
      
    # Slat geometry (for sun tracking)
    slat:
      width_mm: 50
      spacing_mm: 45
      concave: true
```

### Exterior Awning

```yaml
awnings:
  - id: "awning-terrace"
    name: "Terrace Awning"
    area_id: "area-outdoor"
    type: "awning"
    
    protocol: "knx"
    address:
      move: "3/5/1"
      position: "3/5/2"
      position_status: "3/5/3"
      
    capabilities:
      - "position"
      
    timing:
      full_extend_seconds: 25
      full_retract_seconds: 20
      
    # Weather protection (required for exterior)
    weather_protection:
      wind:
        enabled: true
        sensor_id: "sensor-wind-speed"
        retract_threshold_kmh: 35
        lockout_minutes: 30
        
      rain:
        enabled: true
        sensor_id: "sensor-rain"
        retract_on_rain: true
        lockout_minutes: 60
        
      frost:
        enabled: true
        sensor_id: "sensor-outdoor-temp"
        threshold_celsius: 3
        action: "retract"
```

---

## Commands

### Basic Commands

| Command | Parameters | Description |
|---------|------------|-------------|
| `up` | - | Move to open position |
| `down` | - | Move to closed position |
| `stop` | - | Stop movement |
| `position` | `position` (0-100) | Move to position |
| `tilt` | `tilt` (0-100) | Set slat tilt |
| `preset` | `preset_name` | Recall preset |

### Command Examples

**Move to position:**
```yaml
target:
  device_id: "blind-living-1"
command: "position"
parameters:
  position: 50
  
# With fade (gradual movement)
parameters:
  position: 50
  transition_seconds: 5
```

**Set tilt:**
```yaml
target:
  device_id: "blind-office-1"
command: "tilt"
parameters:
  tilt: 30                         # 30% closed
```

**Room command:**
```yaml
target:
  type: "room"
  room_id: "room-living"
  domain: "blinds"
command: "position"
parameters:
  position: 0                      # Open all blinds in room
```

---

## Groups

### Group Configuration

```yaml
blind_groups:
  - id: "group-south-facade"
    name: "South Facade"
    members:
      - "blind-living-1"
      - "blind-living-2"
      - "blind-dining-1"
      
  - id: "group-all-exterior"
    name: "All Exterior Blinds"
    members:
      - "blind-living-1"
      - "blind-living-2"
      - "blind-dining-1"
      - "blind-kitchen-1"
      - "blind-bedroom-1"
      - "blind-bedroom-2"
```

---

## Automation

### Sun Tracking

Automatically adjust blinds to track the sun for optimal shading:

```yaml
sun_tracking:
  enabled: true
  
  # Facades to track
  facades:
    - name: "south"
      azimuth_range: [135, 225]    # Degrees from north
      blinds:
        - "blind-living-1"
        - "blind-living-2"
        
    - name: "east"
      azimuth_range: [45, 135]
      blinds:
        - "blind-office-1"
        
  # Tracking mode
  mode: "glare_protection"         # glare_protection | shade | view
  
  # Calculation parameters
  solar:
    location:
      latitude: 51.5074
      longitude: -0.1278
      
  # Update interval
  update_interval_minutes: 15
  
  # Conditions to pause tracking
  pause_conditions:
    - manual_override_active: true
    - wind_speed_above: 30         # km/h
    - rain_detected: true
```

### Solar Gain Control

Coordinate with climate for energy efficiency:

```yaml
solar_gain:
  enabled: true
  
  # Operating modes
  modes:
    winter:
      # Maximize solar gain when heating needed
      open_on_sunshine: true
      solar_radiation_threshold: 150  # W/m²
      only_when_heating: true
      
    summer:
      # Minimize solar gain when cooling needed
      close_on_sunshine: true
      solar_radiation_threshold: 300  # W/m²
      only_when_cooling: true
      
  # Integration with climate
  climate_zone_id: "zone-living"
  
  # Blind positions
  positions:
    open_for_gain: 0
    closed_for_shade: 80
    partial_shade: 50
```

### Glare Protection

Protect from direct sunlight glare:

```yaml
glare_protection:
  enabled: true
  
  zones:
    - room_id: "room-office"
      blinds: ["blind-office-1"]
      
      # Work areas to protect
      work_positions:
        - name: "desk"
          azimuth: 90               # Facing direction
          sensitivity: "high"
          
      # Tilt to block direct sun
      tilt_for_glare: 70
      
    - room_id: "room-living"
      blinds: ["blind-living-1", "blind-living-2"]
      
      # TV/screen positions
      work_positions:
        - name: "tv_area"
          azimuth: 180
          sensitivity: "medium"
```

### Privacy Automation

Close blinds for privacy:

```yaml
privacy_automation:
  enabled: true
  
  triggers:
    # Sunset-based
    - type: "sunset"
      offset_minutes: 30           # 30 min after sunset
      action:
        blinds: "all_interior"
        position: 100
        
    # Mode-based
    - type: "mode"
      mode: "night"
      action:
        blinds: "bedroom_group"
        position: 100
        
    # Lighting-based
    - type: "lights_on"
      rooms: ["room-living", "room-dining"]
      after_sunset: true
      action:
        position: 100
```

### Wake/Sleep Simulation

Gradual blind movement for natural wake-up:

```yaml
wake_automation:
  enabled: true
  
  schedules:
    - name: "weekday_wake"
      days: ["mon", "tue", "wed", "thu", "fri"]
      time: "06:30"
      blinds: ["blind-bedroom-1"]
      
      # Gradual opening
      sequence:
        - delay_minutes: 0
          position: 80             # Slight crack
        - delay_minutes: 5
          position: 50
        - delay_minutes: 10
          position: 20
        - delay_minutes: 15
          position: 0              # Fully open
          
    - name: "weekend_wake"
      days: ["sat", "sun"]
      time: "08:00"
      blinds: ["blind-bedroom-1"]
      sequence:
        - position: 50
          transition_minutes: 20
        - delay_minutes: 20
          position: 0
          transition_minutes: 10
```

### Scene Integration

```yaml
scenes:
  - id: "scene-cinema"
    name: "Cinema Mode"
    actions:
      - type: "blind"
        target: "room-living"
        command: "position"
        parameters:
          position: 100            # Fully closed
          
  - id: "scene-morning"
    name: "Good Morning"
    actions:
      - type: "blind"
        target: "group-bedroom"
        command: "position"
        parameters:
          position: 0              # Open
```

---

## Weather Protection

### Wind Protection (Mandatory for Exterior)

```yaml
wind_protection:
  sensor_id: "sensor-wind-speed"
  
  # Thresholds
  thresholds:
    warning: 25                    # km/h - start retracting awnings
    retract: 35                    # km/h - fully retract all exterior
    lockout: 45                    # km/h - prevent any extension
    
  # Device-specific
  devices:
    awning-terrace:
      retract_threshold: 30        # More sensitive
      lockout_minutes: 60
      
    blind-exterior-1:
      retract_threshold: 40
      lockout_minutes: 30
      
  # Behavior
  behavior:
    retract_position: 0            # Fully retracted
    lockout_notify: true
    priority: "safety"             # Cannot be overridden
```

### Rain Protection

```yaml
rain_protection:
  sensor_id: "sensor-rain"
  
  # Devices to protect
  devices:
    - "awning-terrace"
    - "awning-balcony"
    
  # Behavior
  behavior:
    retract_on_rain: true
    retract_position: 0
    delay_before_extend_minutes: 30
```

### Frost Protection

```yaml
frost_protection:
  sensor_id: "sensor-outdoor-temp"
  
  threshold_celsius: 3
  
  # Devices affected
  devices:
    - id: "awning-terrace"
      action: "retract"
      
  # Prevent movement when frozen
  lockout_below_celsius: 0
```

---

## Presets

### Preset Configuration

```yaml
presets:
  - device_id: "blind-living-1"
    presets:
      - name: "favorite"
        position: 30
        
      - name: "half"
        position: 50
        
      - name: "shade"
        position: 70
        
  - device_id: "blind-office-1"
    presets:
      - name: "work"
        position: 40
        tilt: 60
        
      - name: "meeting"
        position: 0                # Open
        tilt: 0
```

---

## Mode Integration

```yaml
mode_behaviors:
  home:
    # Normal operation, respect schedules
    sun_tracking: true
    solar_gain: true
    glare_protection: true
    
  away:
    # Security simulation
    position: 50                   # Partially closed
    randomize: true                # Slight random variation
    
  night:
    # Privacy
    bedroom_position: 100
    living_position: 100
    
  holiday:
    # Security simulation
    simulate_presence: true
    random_open_close: true
    times:
      open: "08:00"
      close: "sunset+30"
```

---

## Voice Integration

```yaml
voice_intents:
  - intent: "blinds.position"
    examples:
      - "open the blinds"
      - "close the blinds"
      - "set the blinds to 50 percent"
      - "lower the shades"
      - "raise the blinds in the ${room}"
    action:
      command: "position"
      target: "${room}"
      
  - intent: "blinds.tilt"
    examples:
      - "tilt the blinds"
      - "angle the slats"
    action:
      command: "tilt"
      target: "${room}"
      
  - intent: "blinds.stop"
    examples:
      - "stop the blinds"
      - "stop"
    action:
      command: "stop"
```

---

## KNX Integration Details

### Datapoint Types

| Function | DPT | Description |
|----------|-----|-------------|
| Move Up/Down | DPT 1.008 | 0 = Up, 1 = Down |
| Stop | DPT 1.007 | Step/Stop |
| Position | DPT 5.001 | 0-100% |
| Tilt Position | DPT 5.001 | 0-100% |

### Status Feedback

```yaml
knx_blind:
  device_id: "blind-living-1"
  
  objects:
    - name: "move"
      dpt: "1.008"
      address: "3/1/1"
      direction: "write"
      
    - name: "stop"
      dpt: "1.007"
      address: "3/1/2"
      direction: "write"
      
    - name: "position_set"
      dpt: "5.001"
      address: "3/1/3"
      direction: "write"
      
    - name: "position_status"
      dpt: "5.001"
      address: "3/1/4"
      direction: "read"
      flags: ["read", "transmit"]
```

---

## PHM Integration

```yaml
phm_blinds:
  devices:
    - device_id: "blind-living-1"
      metrics:
        - name: "motor_runtime"
          type: "counter"
          threshold_hours: 2000
          alert: "Motor runtime high - consider inspection"
          
        - name: "travel_time"
          type: "duration"
          baseline_method: "initial_calibration"
          deviation_threshold_percent: 20
          alert: "Travel time deviation - check mechanism"
          
        - name: "operations_count"
          type: "counter"
          threshold: 100000
          alert: "High operation count - preventive maintenance due"
```

---

## Calibration

### Travel Time Calibration

```yaml
calibration:
  device_id: "blind-living-1"
  
  # Measured during commissioning
  travel_times:
    up_seconds: 30
    down_seconds: 28
    
  # Position calibration
  position:
    use_limit_switches: true
    recalibrate_interval_days: 30
    
  # Tilt calibration (venetian)
  tilt:
    full_tilt_seconds: 3
    steps_per_full_tilt: 6
```

---

## Related Documents

- [KNX Protocol Specification](../protocols/knx.md) — KNX blind actuators
- [PHM Specification](../intelligence/phm.md) — Predictive health monitoring framework
- [Climate Domain Specification](climate.md) — Solar gain coordination
- [Lighting Domain Specification](lighting.md) — Scene integration
- [Data Model: Entities](../data-model/entities.md) — Device entity

