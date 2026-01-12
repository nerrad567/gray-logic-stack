---
title: Lighting Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - data-model/entities.md
  - protocols/knx.md
  - protocols/dali.md
---

# Lighting Domain Specification

This document specifies all lighting control features in Gray Logic, including capabilities, state management, automation, and integration patterns.

---

## Overview

Lighting is the most visible and frequently used aspect of building automation. Gray Logic provides comprehensive lighting control while ensuring **physical controls always work**.

### Design Principles

1. **Physical switches work without Gray Logic** — Direct bus links in KNX/DALI
2. **Smooth transitions** — Professional-grade fading, no flicker
3. **Energy efficiency** — Daylight harvesting, occupancy-based control
4. **Human-centric** — Circadian rhythm support, tunable white
5. **Scene-based control** — One-touch atmosphere selection

### Device Types

| Type | Capabilities | Typical Hardware |
|------|-------------|------------------|
| `light_switch` | On/Off | Relay actuator |
| `light_dimmer` | On/Off, Brightness | Dimmer actuator, DALI driver |
| `light_ct` | On/Off, Brightness, Color Temperature | DALI DT8 Tc driver |
| `light_rgb` | On/Off, Brightness, RGB Color | DALI DT8 RGB driver |
| `light_rgbw` | On/Off, Brightness, RGBW Color | DALI DT8 RGBW driver |
| `light_rgbwaf` | On/Off, Brightness, Full Color | DALI DT8 RGBWAF driver |

---

## Feature Matrix

### Priority Levels

- **Must-have**: Required for Year 1 deployment
- **Should-have**: Target for Year 2
- **Nice-to-have**: Future enhancement

### Core Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **On/Off Control** | Must-have | Basic switching |
| **Dimming (0-100%)** | Must-have | Brightness adjustment |
| **Scene Recall** | Must-have | Activate predefined states |
| **Scene Storage** | Must-have | Create and modify scenes |
| **Fade/Transition Times** | Must-have | Smooth state changes |
| **Status Feedback** | Must-have | Know actual device state |
| **Group Control** | Must-have | Control multiple lights together |
| **Room All On/Off** | Must-have | Quick room control |

### Color Control

| Feature | Priority | Description |
|---------|----------|-------------|
| **Color Temperature (CCT)** | Must-have | Warm to cool white (2700K-6500K) |
| **RGB Color** | Must-have | Full color mixing |
| **RGBW Color** | Should-have | RGB + dedicated white |
| **CIE xy Color** | Nice-to-have | Precise color matching |
| **Color Presets** | Should-have | Named colors (Warm, Cool, etc.) |

### Automation Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Occupancy Control** | Must-have | Auto on/off with presence |
| **Vacancy Control** | Must-have | Auto off only (manual on) |
| **Time-Based Scenes** | Must-have | Different defaults by time |
| **Mode Integration** | Must-have | Behavior changes by mode |
| **Circadian Rhythm** | Should-have | Auto CCT by time of day |
| **Daylight Harvesting** | Should-have | Dim based on natural light |
| **Wake Simulation** | Should-have | Gradual sunrise effect |
| **Sleep Simulation** | Should-have | Gradual sunset effect |
| **Pathway Lighting** | Should-have | Night navigation mode |
| **Vacation Simulation** | Nice-to-have | Random patterns when away |

### Advanced Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Human-Centric Lighting** | Nice-to-have | Melanopic optimization |
| **WELL Building Compliance** | Nice-to-have | Certification support |
| **Emergency Integration** | Nice-to-have | Monitor emergency lighting |
| **Luminaire Maintenance** | Nice-to-have | Runtime tracking, replacement alerts |

---

## State Model

### Light State Properties

```yaml
LightState:
  # Common to all types
  on: boolean                       # Power state
  
  # For dimmable lights
  brightness: integer               # 0-100 percent
  
  # For color temperature lights
  color_temp_kelvin: integer        # 2000-10000 K
  color_temp_mirek: integer         # 100-500 (calculated)
  
  # For RGB/RGBW lights
  color:
    mode: "rgb" | "hs" | "xy"       # Color mode
    r: integer                      # 0-255
    g: integer                      # 0-255
    b: integer                      # 0-255
    w: integer                      # 0-255 (RGBW only)
    h: float                        # 0-360 hue (hs mode)
    s: float                        # 0-100 saturation (hs mode)
    x: float                        # 0-1 CIE x (xy mode)
    y: float                        # 0-1 CIE y (xy mode)
  
  # Transition state
  transitioning: boolean            # Currently fading
  transition_end: timestamp         # When transition completes
  
  # Effect state (future)
  effect: string | null             # Active effect name
```

### Example States

**Simple switch:**
```yaml
device_id: "light-hall-pendant"
state:
  on: true
```

**Dimmer:**
```yaml
device_id: "light-living-main"
state:
  on: true
  brightness: 75
```

**Tunable white:**
```yaml
device_id: "light-kitchen-under"
state:
  on: true
  brightness: 100
  color_temp_kelvin: 4000
```

**RGB light:**
```yaml
device_id: "light-living-accent"
state:
  on: true
  brightness: 50
  color:
    mode: "rgb"
    r: 255
    g: 100
    b: 50
```

---

## Commands

### Basic Commands

| Command | Parameters | Description |
|---------|------------|-------------|
| `turn_on` | `brightness?`, `color_temp?`, `color?`, `transition_ms?` | Turn on with optional settings |
| `turn_off` | `transition_ms?` | Turn off |
| `toggle` | - | Toggle current state |
| `set` | Various | Set specific properties |

### Command Examples

**Turn on to 100%:**
```yaml
command: "turn_on"
parameters: {}
```

**Turn on to 50% with transition:**
```yaml
command: "turn_on"
parameters:
  brightness: 50
  transition_ms: 1000
```

**Set brightness (light already on):**
```yaml
command: "set"
parameters:
  brightness: 75
```

**Set color temperature:**
```yaml
command: "set"
parameters:
  color_temp_kelvin: 3000
  transition_ms: 2000
```

**Set RGB color:**
```yaml
command: "set"
parameters:
  color:
    r: 255
    g: 200
    b: 100
  brightness: 80
```

**Turn off with fade:**
```yaml
command: "turn_off"
parameters:
  transition_ms: 5000
```

### Group Commands

Apply to all lights in a group:

```yaml
target:
  type: "group"
  group_id: "group-living-room-all"
command: "turn_on"
parameters:
  brightness: 100
```

### Room Commands

Apply to all lights in a room:

```yaml
target:
  type: "room"
  room_id: "room-living"
command: "turn_off"
parameters:
  transition_ms: 3000
```

---

## Scenes

### Scene Structure

```yaml
Scene:
  id: "scene-living-cinema"
  name: "Cinema Mode"
  room_id: "room-living"
  
  lighting:
    - device_id: "light-living-main"
      state:
        on: true
        brightness: 5
        color_temp_kelvin: 2700
      transition_ms: 3000
      
    - device_id: "light-living-floor"
      state:
        on: true
        brightness: 10
        color_temp_kelvin: 2700
      transition_ms: 3000
      
    - device_id: "light-living-ceiling"
      state:
        on: false
      transition_ms: 3000
      
    - group_id: "group-living-accent"
      state:
        on: true
        brightness: 15
        color:
          r: 180
          g: 100
          b: 50
      transition_ms: 3000
```

### Standard Scene Types

| Scene Type | Description | Typical Settings |
|------------|-------------|------------------|
| **Bright** | Full illumination | 100% brightness, 4000K |
| **Relax** | Evening relaxation | 50% brightness, 2700K |
| **Concentrate** | Task lighting | 80% brightness, 5000K |
| **Energize** | Morning boost | 100% brightness, 6500K |
| **Dimmed** | Low light | 20% brightness, 2700K |
| **Cinema** | Movie watching | 5% brightness, 2700K, accents |
| **Dining** | Dinner ambiance | 60% brightness, 3000K |
| **Night** | Minimal light | 5% brightness, 2200K |
| **Off** | All off | Off |

### Scene Recall Behavior

1. Core loads scene definition
2. Core calculates target states for each device
3. Core sends commands to bridges (parallel)
4. Each device transitions to target state
5. Core publishes scene activation event

**Timing:**
- Scene recall should complete within 500ms of trigger
- Transitions execute according to scene-defined fade times
- Status feedback confirms final states

---

## Automation

### Occupancy-Based Control

```yaml
automation:
  id: "auto-kitchen-occupancy"
  name: "Kitchen Occupancy Lighting"
  
  trigger:
    type: "device_state"
    device_id: "sensor-kitchen-motion"
    condition: "motion == true"
    
  conditions:
    - type: "mode"
      operator: "in"
      value: ["home", "night"]
    - type: "device_state"
      device_id: "light-kitchen-main"
      condition: "on == false"
      
  actions:
    - target:
        type: "room"
        room_id: "room-kitchen"
      command: "turn_on"
      parameters:
        brightness: 100
        # Use time-appropriate color temp
        color_temp_kelvin: "${circadian_temp}"
```

### Vacancy Off Timer

```yaml
automation:
  id: "auto-kitchen-vacancy"
  name: "Kitchen Vacancy Off"
  
  trigger:
    type: "device_state"
    device_id: "sensor-kitchen-motion"
    condition: "motion == false"
    for_minutes: 10  # Debounce
    
  conditions:
    - type: "device_state"
      device_id: "light-kitchen-main"
      condition: "on == true"
      
  actions:
    - target:
        type: "room"
        room_id: "room-kitchen"
      command: "turn_off"
      parameters:
        transition_ms: 30000  # 30 second fade
```

### Mode-Based Defaults

```yaml
# When entering Night mode
automation:
  id: "auto-night-mode-lighting"
  name: "Night Mode Lighting"
  
  trigger:
    type: "mode_change"
    to: "night"
    
  actions:
    # Turn off non-essential lights
    - target:
        type: "group"
        group_id: "group-common-areas"
      command: "turn_off"
      parameters:
        transition_ms: 60000  # 1 minute fade
        
    # Enable pathway lighting
    - target:
        type: "group"
        group_id: "group-pathway"
      command: "set"
      parameters:
        brightness: 10
        color_temp_kelvin: 2200
```

---

## Circadian Rhythm

### Color Temperature Schedule

Automatic color temperature adjustment throughout the day:

```yaml
circadian:
  enabled: true
  
  # Color temperature by time
  schedule:
    - time: "06:00"
      color_temp_kelvin: 4000
      
    - time: "08:00"
      color_temp_kelvin: 5000
      
    - time: "12:00"
      color_temp_kelvin: 6000
      
    - time: "16:00"
      color_temp_kelvin: 4500
      
    - time: "19:00"
      color_temp_kelvin: 3000
      
    - time: "21:00"
      color_temp_kelvin: 2700
      
    - time: "23:00"
      color_temp_kelvin: 2200
      
  # Transition between points
  interpolation: "linear"  # linear | step
  
  # Apply to these devices
  devices:
    - group_id: "group-circadian-enabled"
    
  # Override conditions
  override:
    # Don't apply if scene active
    skip_if_scene: true
    # Don't apply if manually adjusted
    skip_if_manual: true
    manual_timeout_minutes: 60
```

### Implementation

1. Core calculates current circadian temperature
2. When light turns on, apply circadian CCT
3. Interpolate between schedule points
4. Skip if device was manually adjusted (timeout configurable)
5. Skip if scene is active
6. Reapply on mode change if appropriate

---

## Daylight Harvesting

### Light Level Sensor Integration

```yaml
daylight_harvesting:
  enabled: true
  
  zones:
    - zone_id: "zone-living-window"
      sensor_id: "sensor-living-lux"
      
      # Target illuminance (lux)
      target_lux: 500
      
      # Devices to dim
      devices:
        - device_id: "light-living-window-1"
          contribution_percent: 100  # How much this light affects sensor
        - device_id: "light-living-window-2"
          contribution_percent: 80
          
      # Control parameters
      min_brightness: 0    # Allow full off
      max_brightness: 100
      deadband_lux: 50     # Hysteresis
      adjustment_step: 5   # Percent per adjustment
      adjustment_interval_seconds: 30
      
  # Global settings
  settings:
    enable_in_modes: ["home"]
    disable_if_scene_active: true
    disable_at_night: true
```

### Algorithm

1. Read light level sensor
2. Compare to target illuminance
3. If below target - deadband: increase brightness
4. If above target + deadband: decrease brightness
5. Apply changes gradually
6. Respect manual overrides

---

## Wake/Sleep Simulation

### Wake Simulation

Gradual sunrise effect before alarm:

```yaml
wake_simulation:
  id: "wake-master-bedroom"
  name: "Master Bedroom Wake"
  
  # When to complete (alarm time)
  schedule:
    type: "user_alarm"
    user_id: "user-darren"
    # Or fixed time:
    # type: "fixed"
    # time: "07:00"
    # days: ["mon", "tue", "wed", "thu", "fri"]
    
  # Duration of sunrise
  duration_minutes: 30
  
  # Target state at completion
  target:
    brightness: 80
    color_temp_kelvin: 4000
    
  # Start state
  start:
    brightness: 1
    color_temp_kelvin: 2200
    
  # Devices to include
  devices:
    - device_id: "light-bedroom-main"
    - device_id: "light-bedroom-bed-1"
    - device_id: "light-bedroom-bed-2"
    
  # Conditions
  conditions:
    - type: "mode"
      operator: "neq"
      value: "away"
```

### Sleep Simulation

Gradual dim to off:

```yaml
sleep_simulation:
  id: "sleep-master-bedroom"
  name: "Master Bedroom Sleep"
  
  trigger:
    type: "scene"
    scene_id: "scene-good-night"
    
  duration_minutes: 15
  
  # Fade to off
  target:
    on: false
    
  devices:
    - device_id: "light-bedroom-main"
    - device_id: "light-bedroom-bed-1"
    - device_id: "light-bedroom-bed-2"
```

---

## Pathway Lighting

Low-level lighting for nighttime navigation:

```yaml
pathway_lighting:
  enabled: true
  
  # When active
  active_modes: ["night"]
  active_times:
    start: "22:00"
    end: "07:00"
    
  # Motion triggers
  sensors:
    - sensor_id: "sensor-landing-motion"
      lights:
        - device_id: "light-landing-1"
        - device_id: "light-stairs-1"
        - device_id: "light-stairs-2"
      timeout_seconds: 120
      
    - sensor_id: "sensor-hall-motion"
      lights:
        - device_id: "light-hall-1"
        - device_id: "light-hall-2"
      timeout_seconds: 60
      
  # Pathway light settings
  settings:
    brightness: 10
    color_temp_kelvin: 2200
    transition_on_ms: 500
    transition_off_ms: 5000
    
  # Don't activate if main lights are on
  skip_if_room_lit: true
  room_lit_threshold: 20  # Brightness percent
```

---

## Device Configuration

### Light Device Definition

```yaml
device:
  id: "light-living-main"
  name: "Living Room Main Light"
  room_id: "room-living"
  
  type: "light_dimmer"
  domain: "lighting"
  
  # Protocol configuration
  protocol: "knx"
  address:
    switch_ga: "1/0/1"
    switch_status_ga: "6/0/1"
    brightness_ga: "2/0/1"
    brightness_status_ga: "6/0/2"
    
  # Capabilities
  capabilities:
    - "on_off"
    - "dim"
    
  # Device-specific config
  config:
    min_brightness: 5        # Minimum dim level (percent)
    max_brightness: 100      # Maximum brightness
    default_transition_ms: 500
    power_on_behavior: "last"  # last | on | off | default
    power_on_level: 100        # If power_on_behavior is "default"
```

### Tunable White Configuration

```yaml
device:
  id: "light-kitchen-island"
  name: "Kitchen Island Light"
  room_id: "room-kitchen"
  
  type: "light_ct"
  domain: "lighting"
  
  protocol: "dali"
  address:
    gateway: "dali-gw-01"
    bus: 1
    short_address: 5
    
  capabilities:
    - "on_off"
    - "dim"
    - "color_temp"
    
  config:
    min_brightness: 1
    color_temp_range:
      min_kelvin: 2700
      max_kelvin: 6500
    circadian_enabled: true
```

### RGB Configuration

```yaml
device:
  id: "light-living-accent"
  name: "Living Room Accent"
  room_id: "room-living"
  
  type: "light_rgb"
  domain: "lighting"
  
  protocol: "dali"
  address:
    gateway: "dali-gw-01"
    bus: 1
    short_address: 10
    
  capabilities:
    - "on_off"
    - "dim"
    - "color_rgb"
    
  config:
    min_brightness: 1
    color_modes: ["rgb", "hs"]
```

---

## Groups

### Group Types

| Type | Description |
|------|-------------|
| **Room Group** | All lights in a room |
| **Functional Group** | Lights by function (main, accent, task) |
| **Zone Group** | Daylight harvesting zone |
| **Circadian Group** | Lights with circadian control |
| **Custom Group** | User-defined grouping |

### Group Definition

```yaml
groups:
  - id: "group-living-all"
    name: "Living Room All Lights"
    type: "room"
    room_id: "room-living"
    devices:
      - "light-living-main"
      - "light-living-ceiling"
      - "light-living-floor"
      - "light-living-accent-1"
      - "light-living-accent-2"
      
  - id: "group-living-main"
    name: "Living Room Main Lights"
    type: "functional"
    room_id: "room-living"
    devices:
      - "light-living-main"
      - "light-living-ceiling"
      
  - id: "group-living-accent"
    name: "Living Room Accent Lights"
    type: "functional"
    room_id: "room-living"
    devices:
      - "light-living-floor"
      - "light-living-accent-1"
      - "light-living-accent-2"
```

---

## Integration Patterns

### Wall Switch Integration

KNX switch directly controls actuator, Gray Logic observes:

```
Switch Press → KNX Telegram → Actuator (light turns on)
                    ↓
              KNX Bridge sees telegram
                    ↓
              Publishes state to MQTT
                    ↓
              Core updates state
                    ↓
              UI reflects change
```

### Keypad Integration

Multi-button keypad triggers scenes:

```yaml
keypad:
  device_id: "keypad-living-01"
  
  buttons:
    - button: 1
      action: "scene"
      scene_id: "scene-living-bright"
      led_feedback: true
      
    - button: 2
      action: "scene"
      scene_id: "scene-living-relax"
      led_feedback: true
      
    - button: 3
      action: "scene"
      scene_id: "scene-living-cinema"
      led_feedback: true
      
    - button: 4
      action: "room_off"
      room_id: "room-living"
      led_feedback: true
```

### Voice Integration

Voice command processing for lighting:

```yaml
voice_intents:
  - intent: "lighting.on"
    examples:
      - "turn on the lights"
      - "lights on"
      - "switch on the light"
    action:
      command: "turn_on"
      target: "${room}"  # From context
      
  - intent: "lighting.off"
    examples:
      - "turn off the lights"
      - "lights off"
    action:
      command: "turn_off"
      target: "${room}"
      
  - intent: "lighting.dim"
    examples:
      - "dim the lights"
      - "dim to 50 percent"
      - "set brightness to ${level}"
    action:
      command: "set"
      parameters:
        brightness: "${level}"
      target: "${room}"
      
  - intent: "lighting.scene"
    examples:
      - "set cinema mode"
      - "activate relax scene"
      - "movie time"
    action:
      command: "scene"
      scene_id: "${matched_scene}"
```

---

## Health Monitoring

### Metrics to Track

| Metric | Source | Purpose |
|--------|--------|---------|
| Runtime hours | DALI query | Maintenance planning |
| Switch cycles | Counter | Relay lifespan |
| Lamp failures | DALI status | Immediate alert |
| Error rate | Bridge metrics | Communication health |
| Response time | Core timing | Performance monitoring |

### DALI Diagnostics

```yaml
diagnostics:
  polling_interval_seconds: 60
  
  queries:
    - type: "status"
      detect: "lamp_failure"
      alert_severity: "warning"
      
    - type: "actual_level"
      compare_to: "commanded"
      deviation_threshold: 10
      alert_severity: "warning"
```

### Alerting

```yaml
alerts:
  - condition: "lamp_failure"
    severity: "warning"
    message: "Lamp failure detected: ${device_name}"
    actions:
      - notify_user: true
      - log_phm: true
      
  - condition: "device_offline"
    severity: "warning"
    for_minutes: 5
    message: "Light offline: ${device_name}"
```

---

## Commercial Lighting

This section covers lighting patterns specific to commercial and office environments.

### Emergency Lighting (DALI Part 202)

Gray Logic monitors emergency lighting but **never controls** safety-critical functions. Emergency lighting operates independently via certified hardware.

```yaml
emergency_lighting:
  # Gray Logic role: MONITORING ONLY
  role: "monitor"                   # Never "control"
  
  # DALI Part 202 gateway
  gateway:
    type: "dali_202"
    device_id: "dali-gw-emergency"
    address: "192.168.1.55"
    
  # Emergency luminaires
  luminaires:
    - device_id: "emergency-exit-01"
      name: "Exit Sign - Reception"
      type: "exit_sign"
      location: "Reception"
      dali_address: 1
      
    - device_id: "emergency-corridor-01"
      name: "Emergency Light - Corridor 1"
      type: "emergency_light"
      location: "Ground Floor Corridor"
      dali_address: 2
      battery_type: "3hr"           # 1hr or 3hr
      
  # Monitoring features
  monitoring:
    # Battery status polling
    battery_check_interval_hours: 24
    
    # Automatic test scheduling
    function_test:
      enabled: true
      interval_days: 30
      preferred_time: "02:00"       # Outside occupied hours
      notify_before_minutes: 60
      
    duration_test:
      enabled: true
      interval_days: 365
      preferred_day: "sunday"
      preferred_time: "02:00"
      notify_before_hours: 24
      
  # Alerting
  alerts:
    - condition: "lamp_failure"
      severity: "high"
      notify: ["facility_manager", "maintenance"]
      sla_hours: 24                 # Must fix within 24h
      
    - condition: "battery_failure"
      severity: "high"
      notify: ["facility_manager", "maintenance"]
      sla_hours: 24
      
    - condition: "communication_loss"
      severity: "medium"
      notify: ["maintenance"]
      
    - condition: "test_failed"
      severity: "high"
      notify: ["facility_manager"]
      
  # Compliance reporting
  reporting:
    enabled: true
    format: "pdf"
    schedule: "monthly"
    recipients: ["compliance@company.com"]
    include:
      - test_history
      - fault_log
      - luminaire_inventory
```

**Emergency Light State Model:**
```yaml
EmergencyLightState:
  # Lamp status
  lamp_status: enum                 # ok | failure | unknown
  lamp_runtime_hours: integer
  
  # Battery status
  battery_status: enum              # ok | low | failure | charging
  battery_level_percent: integer | null
  battery_duration_remaining: integer | null  # minutes
  
  # Mode
  mode: enum                        # normal | emergency | test | inhibited
  
  # Test results
  last_function_test:
    date: timestamp
    result: enum                    # pass | fail | skipped
    duration_seconds: integer
    
  last_duration_test:
    date: timestamp
    result: enum
    duration_minutes: integer
    battery_end_voltage: float
    
  # Faults
  faults: [string]                  # List of active faults
```

### Daylight Harvesting (Commercial)

Commercial spaces benefit significantly from daylight harvesting.

```yaml
daylight_harvesting:
  zone_id: "zone-open-plan-north"
  
  # Light sensors
  sensors:
    - device_id: "lux-sensor-window-1"
      location: "perimeter"
      weight: 1.0
    - device_id: "lux-sensor-core-1"
      location: "core"
      weight: 0.8
      
  # Target illuminance (lux)
  targets:
    task_area: 500                  # Desks
    circulation: 200                # Corridors
    meeting_room: 400
    
  # Daylight zones (distance from windows)
  zones:
    - name: "perimeter"
      depth_meters: 3
      lights:
        - "light-row-1"
        - "light-row-2"
      min_level: 10                 # Never fully off during occupied
      response_time_ms: 30000       # Slow response
      
    - name: "mid"
      depth_meters: 6
      lights:
        - "light-row-3"
        - "light-row-4"
      min_level: 20
      
    - name: "core"
      depth_meters: null            # Beyond windows
      lights:
        - "light-row-5"
        - "light-row-6"
      daylight_contribution: 0      # No daylight reaches here
      
  # Control parameters
  control:
    mode: "closed_loop"             # closed_loop | open_loop
    algorithm: "proportional"       # proportional | stepped
    deadband_lux: 50
    response_rate: "slow"           # slow | medium | fast
    
  # Blind coordination
  blind_coordination:
    enabled: true
    priority: "glare_then_daylight" # Glare control takes precedence
```

### Occupancy-Based Lighting (Commercial)

```yaml
occupancy_lighting:
  area_id: "area-open-plan"
  
  # Occupancy sensors
  sensors:
    - device_id: "pir-zone-1"
      coverage: ["light-row-1", "light-row-2"]
    - device_id: "pir-zone-2"
      coverage: ["light-row-3", "light-row-4"]
    - device_id: "mmwave-desk-area"
      type: "presence"              # More sensitive than PIR
      coverage: ["light-row-5"]
      
  # Behavior by time
  schedules:
    occupied_hours:
      start: "07:00"
      end: "19:00"
      on_trigger: "occupancy"       # Lights on when occupied
      off_trigger: "vacancy"        # Lights off when vacant
      vacancy_timeout_minutes: 15
      on_level: 100                 # Or auto from daylight
      
    after_hours:
      start: "19:00"
      end: "07:00"
      on_trigger: "occupancy"
      off_trigger: "vacancy"
      vacancy_timeout_minutes: 5    # Shorter timeout
      on_level: 70                  # Reduced level
      pathway_lights: ["light-exit-route"]  # Keep minimal on
      
  # Grace period to prevent flicker
  hold_time_seconds: 30             # Minimum on time
  
  # Notification for security
  after_hours_notification:
    enabled: true
    notify: ["security"]
    message: "Movement detected in ${area} after hours"
```

### Corridor and Circulation Lighting

```yaml
corridor_lighting:
  area_id: "corridor-ground-floor"
  
  # Sections (for sequential control)
  sections:
    - id: "section-1"
      lights: ["light-corr-1", "light-corr-2"]
      sensor: "pir-corr-section-1"
    - id: "section-2"
      lights: ["light-corr-3", "light-corr-4"]
      sensor: "pir-corr-section-2"
    - id: "section-3"
      lights: ["light-corr-5", "light-corr-6"]
      sensor: "pir-corr-section-3"
      
  # Pathway lighting behavior
  behavior:
    # During occupied hours
    occupied:
      base_level: 50                # Always on at this level
      presence_level: 100           # Full when occupied
      
    # After hours
    after_hours:
      base_level: 10                # Minimal security lighting
      presence_level: 80            # Bright enough for safety
      timeout_seconds: 120          # Off after 2 min
      sequential_on: true           # Light path ahead of occupant
```

### Meeting Room Lighting Integration

```yaml
meeting_room_lighting:
  room_id: "meeting-room-large"
  
  # Booking integration
  booking_source: "calendar"
  
  # Pre-meeting setup
  pre_meeting:
    lead_time_minutes: 5
    scene: "scene-meeting-default"
    
  # Scenes for different uses
  scenes:
    - id: "scene-meeting-default"
      name: "Meeting"
      description: "Standard meeting lighting"
      lights:
        - group: "all"
          brightness: 100
          color_temp: 4000
          
    - id: "scene-meeting-presentation"
      name: "Presentation"
      description: "Front dim for screen"
      lights:
        - group: "front"
          brightness: 30
        - group: "rear"
          brightness: 100
        - group: "table"
          brightness: 50
          
    - id: "scene-meeting-video"
      name: "Video Call"
      description: "Even lighting for cameras"
      lights:
        - group: "all"
          brightness: 100
          color_temp: 5000           # Daylight white
          
  # AV integration
  av_triggers:
    - trigger: "projector_on"
      action: "recall_scene"
      scene: "scene-meeting-presentation"
      
    - trigger: "video_call_start"
      action: "recall_scene"
      scene: "scene-meeting-video"
      
  # End of meeting
  post_meeting:
    delay_minutes: 10               # If room unoccupied
    action: "off"
```

### Personal Control and Task Lighting

Allow occupants limited personal control while maintaining efficiency.

```yaml
personal_control:
  area_id: "open-plan-east"
  
  # Zone breakdown
  control_zones:
    - zone_id: "desk-cluster-1"
      lights: ["light-cluster-1"]
      occupants: 4
      adjustment_range:
        brightness: [-20, +20]      # ±20% from auto level
        color_temp: [-500, +500]    # ±500K from auto
        
  # Personal control interface
  interface:
    type: "app"                     # app | desk_sensor | wall_keypad
    require_presence: true          # Must be at desk
    reset_on_vacancy: true          # Return to auto when leave
    
  # Limits
  limits:
    min_brightness: 30              # Health and safety minimum
    max_brightness: 100
    min_color_temp: 2700
    max_color_temp: 6500
```

### Commercial Commissioning Checklist

- [ ] All luminaires responding to commands
- [ ] DALI addresses correctly assigned
- [ ] Emergency lighting monitored (Part 202)
- [ ] Emergency test schedule configured
- [ ] Lux sensors calibrated
- [ ] Daylight harvesting zones tuned
- [ ] Occupancy sensors coverage verified
- [ ] Occupancy timeouts appropriate
- [ ] Scenes programmed for all rooms
- [ ] AV integration tested
- [ ] Booking system integration tested
- [ ] After-hours behavior verified
- [ ] Energy baseline recorded

---

## Related Documents

- [KNX Protocol Specification](../protocols/knx.md) — KNX lighting integration
- [DALI Protocol Specification](../protocols/dali.md) — DALI lighting integration
- [Scenes and Automation](../automation/scenes.md) — Scene management
- [Climate Domain Specification](climate.md) — Occupancy schedule sharing
- [Data Model: Entities](../data-model/entities.md) — Device entity definition
- [Voice Pipeline](../intelligence/voice-pipeline.md) — Voice control

