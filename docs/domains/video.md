---
title: Video/AV Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - data-model/entities.md
  - domains/audio.md
---

# Video/AV Domain Specification

This document specifies how Gray Logic manages video distribution, display control, and AV switching. For multi-room audio, see [Audio Domain](audio.md).

---

## Overview

### Philosophy

Gray Logic is an **orchestration layer** for video/AV, not a video processor:

| Gray Logic Does | Gray Logic Does NOT |
|-----------------|---------------------|
| Switch video matrices | Process video signals |
| Control displays (on/off, input) | Stream video content |
| Select sources | Upscale/downscale video |
| Coordinate with scenes | Handle HDCP negotiation |
| Control projector screens | Encode/decode video |

### Design Principles

1. **Source-centric** — Users select what to watch, not which input number
2. **Display-centric** — Control is about displays, not matrix outputs
3. **Unified control** — One interface regardless of underlying equipment
4. **Scene integration** — Video is part of whole-room experiences
5. **Graceful degradation** — Local controls always work

### Video vs Audio

| Aspect | Video Domain | Audio Domain |
|--------|--------------|--------------|
| Focus | Picture distribution | Sound distribution |
| Typical equipment | Matrix, displays, projectors | Amps, speakers, streamers |
| Grouping | One source per display | Multi-room sync possible |
| Content | External sources | May include streaming |

---

## System Types

### Distribution Architectures

| Type | Description | Scale | Examples |
|------|-------------|-------|----------|
| **Direct** | Source → Display | 1-3 displays | HDMI cables |
| **Matrix** | N sources → M displays | 4-16 displays | HDMI matrix |
| **HDBaseT** | Matrix + Cat6 distribution | 8-64 displays | Crestron, Extron |
| **IP Video** | Encoders/decoders over network | Unlimited | Just Add Power, ZeeVee |
| **AV Receiver** | Local switching per zone | 1-3 zones | Denon, Yamaha |

### Supported Equipment

| Category | Examples | Control Method |
|----------|----------|----------------|
| **Matrices** | Atlona, Extron, Crestron, Blustream | RS-232, IP, IR |
| **Displays** | LG, Samsung, Sony TVs | RS-232, IP, CEC, IR |
| **Projectors** | Epson, Sony, JVC, BenQ | RS-232, IP |
| **Projector Screens** | Screen Innovations, Elite | RS-232, Contact closure |
| **AV Receivers** | Denon, Marantz, Yamaha | IP, RS-232 |
| **Streaming Devices** | Apple TV, Nvidia Shield | Limited (CEC, IR) |

---

## Data Model

### Video Zone

A display or viewing area.

```yaml
VideoZone:
  id: uuid
  site_id: uuid
  name: string                        # "Living Room TV", "Cinema"
  
  # Physical mapping
  room_id: uuid
  
  # Display device(s)
  displays:
    - id: "display-living-tv"
      type: "tv"                      # tv, projector, monitor
      
  # Projector screen (if applicable)
  screen:
    device_id: uuid | null            # Motorized screen device
    
  # Matrix output (if applicable)
  matrix:
    matrix_id: string | null
    output_number: integer | null
    
  # Audio zone link
  audio_zone_id: uuid | null          # Associated audio zone
  
  # Configuration
  config:
    default_source: string | null     # Default source on power-on
    auto_off_minutes: integer | null  # Auto-off after inactivity
    cec_enabled: boolean              # Use HDMI-CEC
    
  # Current state
  state:
    power: boolean
    source_id: string | null
    display_state:
      power: boolean
      input: string | null
      volume: integer | null          # If display has speakers
      mute: boolean | null
```

### Video Source

An input that can be displayed.

```yaml
VideoSource:
  id: string                          # "apple-tv-living", "sky-q"
  name: string                        # "Apple TV", "Sky Q"
  type: enum                          # See source types
  icon: string                        # UI icon
  
  # Hardware mapping
  hardware:
    # For matrix systems
    matrix_id: string | null
    matrix_input: integer | null
    
    # For direct connection
    display_input: string | null      # "HDMI1", "HDMI2"
    
    # For AV receiver routing
    avr_input: string | null
    
  # Source device control (if controllable)
  device:
    control_method: string | null     # "ip", "ir", "cec"
    device_id: string | null          # Reference to controllable device
    
  # Availability
  available_in: [string] | null       # Zone IDs, null = all zones
  
  # Content type hint
  content_type: string | null         # "streaming", "live_tv", "gaming", "computer"
```

**Source Types:**

| Type | Description | Examples |
|------|-------------|----------|
| `streaming_box` | Streaming device | Apple TV, Fire TV, Roku |
| `cable_sat` | Cable/Satellite box | Sky Q, Virgin V6 |
| `game_console` | Gaming console | PS5, Xbox, Switch |
| `blu_ray` | Disc player | Blu-ray, UHD player |
| `computer` | Computer/HTPC | Mac Mini, NUC |
| `camera` | Camera feed | CCTV, doorbell |
| `signage` | Digital signage | Info displays |

### Video Matrix

Centralized video switching.

```yaml
VideoMatrix:
  id: string
  name: string                        # "Main Video Matrix"
  
  # Hardware
  manufacturer: string                # "atlona", "extron"
  model: string
  
  # Connection
  connection:
    type: "ip" | "serial"
    host: string | null
    port: integer | null
    serial_device: string | null
    
  # Size
  inputs: integer                     # Number of inputs
  outputs: integer                    # Number of outputs
  
  # Current routing
  routing:
    1: 3                              # Output 1 shows Input 3
    2: 1                              # Output 2 shows Input 1
    3: 3                              # Output 3 shows Input 3
    4: 0                              # Output 4 off
```

### Display Device

TV, projector, or monitor.

```yaml
DisplayDevice:
  id: uuid
  name: string
  type: "tv" | "projector" | "monitor"
  
  # Hardware
  manufacturer: string                # "lg", "samsung", "sony", "epson"
  model: string | null
  
  # Connection
  connection:
    type: "ip" | "serial" | "cec" | "ir"
    host: string | null               # For IP control
    mac_address: string | null        # For WoL
    serial_device: string | null
    cec_address: string | null        # CEC logical address
    ir_device: string | null          # IR blaster device
    
  # Capabilities
  capabilities:
    power_control: boolean
    input_select: boolean
    volume_control: boolean           # If has speakers
    picture_mode: boolean
    
  # Inputs
  inputs:
    - id: "HDMI1"
      name: "Matrix"
      source_id: null                 # Mapped dynamically from matrix
    - id: "HDMI2"
      name: "Apple TV"
      source_id: "apple-tv-living"
    - id: "HDMI3"
      name: "PlayStation"
      source_id: "ps5-living"
      
  # Current state
  state:
    power: boolean
    input: string | null
    volume: integer | null
    mute: boolean | null
    picture_mode: string | null
```

### Projector Screen

Motorized projection screen.

```yaml
ProjectorScreen:
  id: uuid
  name: string                        # "Cinema Screen"
  
  # Control
  connection:
    type: "relay" | "serial" | "ip"
    device_id: string | null          # Relay module or controller
    
  # Configuration
  config:
    travel_time_seconds: 30           # Time to fully deploy
    aspect_ratio: "16:9" | "2.35:1" | "variable"
    masking: boolean                  # Has masking panels
    
  # Current state
  state:
    position: "up" | "down" | "moving"
    position_percent: integer | null  # If position feedback available
```

---

## Commands

### Zone Commands

```yaml
# Power on zone (display + screen if applicable)
- zone_id: "zone-cinema"
  command: "power"
  parameters:
    power: true

# Select source
- zone_id: "zone-living"
  command: "source"
  parameters:
    source_id: "apple-tv"

# Power on + select source
- zone_id: "zone-living"
  command: "watch"
  parameters:
    source_id: "sky-q"
```

### Display Commands

```yaml
# Power control
- display_id: "display-living-tv"
  command: "power"
  parameters:
    power: true

# Input selection
- display_id: "display-living-tv"
  command: "input"
  parameters:
    input: "HDMI2"

# Volume (if display has speakers)
- display_id: "display-living-tv"
  command: "volume"
  parameters:
    level: 30

# Picture mode
- display_id: "display-living-tv"
  command: "picture_mode"
  parameters:
    mode: "cinema"                    # cinema, vivid, game, etc.
```

### Matrix Commands

```yaml
# Route source to output
- matrix_id: "matrix-main"
  command: "route"
  parameters:
    input: 3
    output: 1

# Route source to multiple outputs
- matrix_id: "matrix-main"
  command: "route"
  parameters:
    input: 3
    outputs: [1, 2, 4]

# Disconnect output
- matrix_id: "matrix-main"
  command: "disconnect"
  parameters:
    output: 3
```

### Projector Screen Commands

```yaml
# Lower screen
- screen_id: "screen-cinema"
  command: "down"

# Raise screen
- screen_id: "screen-cinema"
  command: "up"

# Stop (if supported)
- screen_id: "screen-cinema"
  command: "stop"
```

### Source Commands

For controllable sources:

```yaml
# Power on source device
- source_id: "apple-tv-living"
  command: "power"
  parameters:
    power: true

# Navigation (via IR or IP)
- source_id: "apple-tv-living"
  command: "navigate"
  parameters:
    action: "menu"                    # menu, up, down, left, right, select, back

# Transport
- source_id: "apple-tv-living"
  command: "transport"
  parameters:
    action: "play"                    # play, pause, stop, ff, rw
```

---

## Automation Integration

### Scene Integration

Video as part of scenes:

```yaml
# Cinema Mode
scenes:
  - id: "scene-cinema"
    name: "Cinema Mode"
    actions:
      # Lower projector screen
      - device_id: "screen-cinema"
        command: "down"
        
      # Power on projector
      - device_id: "projector-cinema"
        command: "power"
        parameters:
          power: true
          
      # Select source
      - zone_id: "zone-cinema"
        command: "source"
        parameters:
          source_id: "apple-tv-cinema"
          
      # Dim lights
      - domain: "lighting"
        scope: "room-cinema"
        command: "dim"
        parameters:
          level: 10
          
      # Close blinds
      - domain: "blinds"
        scope: "room-cinema"
        command: "close"
        
      # Set audio
      - zone_id: "zone-cinema-audio"
        command: "source"
        parameters:
          source_id: "avr-hdmi-arc"
          volume: 35

# TV Mode (casual viewing)
  - id: "scene-tv"
    name: "Watch TV"
    actions:
      - zone_id: "zone-living"
        command: "watch"
        parameters:
          source_id: "sky-q"
          
      - domain: "lighting"
        scope: "room-living"
        command: "scene"
        parameters:
          scene: "dim"
```

### Mode Integration

```yaml
modes:
  - id: "away"
    behaviours:
      video:
        power_off_all: true
        
  - id: "night"
    behaviours:
      video:
        auto_off_minutes: 120         # Turn off after 2 hours inactivity
        
  - id: "guest"
    behaviours:
      video:
        restrict_sources: ["apple-tv", "netflix"]  # Limited sources
```

### Event-Driven Automation

```yaml
triggers:
  # Doorbell → Show camera on TV
  - type: "device_event"
    source:
      device_id: "doorbell-front"
      event: "pressed"
    execute:
      type: "actions"
      actions:
        # Show doorbell camera
        - zone_id: "zone-living"
          command: "watch"
          parameters:
            source_id: "camera-doorbell"
            
        # Pause current content (if supported)
        - source_id: "${current_source}"
          command: "transport"
          parameters:
            action: "pause"
            
    conditions:
      - type: "video_zone_active"
        zone_id: "zone-living"

  # Motion in room → Wake display
  - type: "motion_detected"
    source:
      room_id: "room-living"
    conditions:
      - type: "time"
        operator: "between"
        value: ["06:00", "23:00"]
    execute:
      - display_id: "display-living-tv"
        command: "power"
        parameters:
          power: true

  # No motion → Power off after timeout
  - type: "no_motion"
    source:
      room_id: "room-living"
    for_minutes: 30
    execute:
      - zone_id: "zone-living"
        command: "power"
        parameters:
          power: false
```

---

## HDMI-CEC Integration

### CEC Capabilities

HDMI-CEC allows limited control over HDMI:

| Capability | Support |
|------------|---------|
| Power on/off displays | ✅ Common |
| Input switching | ✅ Common |
| Volume control | ⚠️ Varies |
| Transport (play/pause) | ⚠️ Varies |
| Source device control | ⚠️ Limited |

### CEC Configuration

```yaml
cec:
  enabled: true
  adapter: "/dev/cec0"                # USB-CEC adapter
  
  # Device mapping
  devices:
    - cec_address: "0"                # TV
      device_id: "display-living-tv"
      
    - cec_address: "4"                # Playback device 1
      device_id: "apple-tv-living"
      
  # Behavior
  config:
    power_on_active_source: true      # TV powers on when source activates
    standby_propagation: true         # Standby command to all devices
    volume_forwarding: true           # Forward volume to AVR
```

### CEC Limitations

```yaml
# CEC is supplementary, not primary
cec_policy:
  use_for:
    - "display_power"                 # Wake/sleep displays
    - "input_follow"                  # Display follows active source
    
  do_not_rely_on_for:
    - "matrix_switching"              # Use matrix commands
    - "source_transport"              # Use direct IP/IR
    - "critical_operations"           # Always have fallback
```

---

## MQTT Topics

### Zone State

```yaml
topic: graylogic/video/zone/{zone_id}/state
payload:
  zone_id: "zone-living"
  timestamp: "2026-01-12T20:30:00Z"
  state:
    power: true
    source_id: "apple-tv-living"
    display:
      power: true
      input: "HDMI1"
```

### Display State

```yaml
topic: graylogic/video/display/{display_id}/state
payload:
  display_id: "display-living-tv"
  timestamp: "2026-01-12T20:30:00Z"
  state:
    power: true
    input: "HDMI1"
    volume: 25
    mute: false
```

### Matrix State

```yaml
topic: graylogic/video/matrix/{matrix_id}/state
payload:
  matrix_id: "matrix-main"
  timestamp: "2026-01-12T20:30:00Z"
  routing:
    1: 3
    2: 1
    3: 0
    4: 3
```

### Commands

```yaml
topic: graylogic/video/command
payload:
  target: "zone-living"
  command: "source"
  parameters:
    source_id: "apple-tv-living"
  request_id: "req-12345"
```

---

## Video Bridge

### Architecture

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  Video Matrix   │◄────►│   Video Bridge  │◄────►│  Gray Logic     │
│  (Atlona, etc)  │      │   (Go process)  │      │  Core           │
├─────────────────┤      │                 │      │                 │
│  Displays       │◄────►│                 │      │                 │
│  (LG, Samsung)  │      │                 │      │                 │
├─────────────────┤      │                 │      │                 │
│  CEC Adapter    │◄────►│                 │      │                 │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

### Bridge Configuration

```yaml
# /etc/graylogic/bridges/video.yaml
bridge:
  id: "video-bridge"
  type: "video"
  
  matrix:
    manufacturer: "atlona"
    model: "at-uhd-pro3-88m"
    connection:
      type: "ip"
      host: "192.168.1.80"
      port: 23
      
  displays:
    - id: "display-living-tv"
      manufacturer: "lg"
      model: "oled65"
      connection:
        type: "ip"
        host: "192.168.1.81"
        
    - id: "display-bedroom-tv"
      manufacturer: "samsung"
      connection:
        type: "ip"
        host: "192.168.1.82"
        
  cec:
    enabled: true
    adapter: "/dev/ttyACM0"
    
  polling:
    matrix_interval_ms: 2000
    display_interval_ms: 5000
    
  mqtt:
    broker: "localhost"
    port: 1883
```

---

## PHM Integration

### PHM Value for Video Equipment

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Projector | ★★★★☆ | Lamp hours, filter status, temp |
| Matrix | ★★☆☆☆ | Communication errors, HDCP failures |
| Displays | ★★☆☆☆ | Power hours, backlight (if reported) |

### PHM Configuration

```yaml
phm_video:
  devices:
    - device_id: "projector-cinema"
      type: "projector"
      parameters:
        - name: "lamp_hours"
          source: "device:lamp_hours"
          threshold: 3000             # Typical lamp life
          alert_at_percent: 80
          alert: "Projector lamp approaching end of life"
          
        - name: "filter_hours"
          source: "device:filter_hours"
          threshold: 1000
          alert: "Projector filter needs cleaning"
          
        - name: "temperature_c"
          source: "device:temperature"
          threshold: 45
          alert: "Projector running hot - check ventilation"
          
    - device_id: "matrix-main"
      type: "video_matrix"
      parameters:
        - name: "hdcp_failures"
          baseline_method: "daily_count"
          threshold_per_day: 10
          alert: "HDCP handshake failures - check cables/devices"
          
        - name: "comm_errors"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 200
          alert: "Matrix communication errors elevated"
```

---

## Commercial Video

### Digital Signage

```yaml
Signage:
  zones:
    - id: "signage-reception"
      name: "Reception Display"
      display_id: "display-reception"
      
      content:
        default: "welcome-loop"
        schedules:
          - content: "morning-news"
            time: "08:00-09:00"
          - content: "welcome-loop"
            time: "09:00-17:00"
          - content: "off"
            time: "17:00-08:00"
            
      triggers:
        - event: "meeting_starting"
          room: "room-meeting-1"
          content: "room-meeting-1-info"
```

### Video Conferencing

```yaml
VideoConference:
  rooms:
    - id: "vc-meeting-1"
      name: "Meeting Room 1"
      
      equipment:
        display: "display-meeting-1"
        camera: "camera-vc-meeting-1"
        codec: "zoom-room-1"          # Or Teams Room, etc.
        
      presets:
        - id: "presentation"
          actions:
            - source: "laptop-hdmi"
            - camera_preset: "wide"
            
        - id: "video-call"
          actions:
            - source: "vc-codec"
            - camera_preset: "speaker-track"
            
      integration:
        calendar: "room-meeting-1@company.com"
        auto_prepare_minutes: 5       # Prepare room 5 min before meeting
```

### Multi-Display Walls

```yaml
VideoWall:
  id: "wall-control-room"
  name: "Control Room Video Wall"
  
  layout:
    rows: 2
    columns: 3
    displays:
      - [1, 2, 3]                     # Top row: outputs 1, 2, 3
      - [4, 5, 6]                     # Bottom row: outputs 4, 5, 6
      
  presets:
    - id: "single-source"
      description: "One source across all displays"
      routing:
        all: 1                        # All outputs show input 1
        
    - id: "quad"
      description: "4 sources in corners, 2 large center"
      routing:
        1: 1                          # Top-left
        3: 2                          # Top-right
        4: 3                          # Bottom-left
        6: 4                          # Bottom-right
        2: 5                          # Top-center (large)
        5: 5                          # Bottom-center (large)
```

---

## Configuration Examples

### Residential: Simple

```yaml
video:
  zones:
    - id: "zone-living"
      name: "Living Room"
      display_id: "display-living-tv"
      audio_zone_id: "zone-living-audio"
      
  sources:
    - id: "apple-tv"
      name: "Apple TV"
      type: "streaming_box"
      hardware:
        display_input: "HDMI1"
        
    - id: "sky-q"
      name: "Sky Q"
      type: "cable_sat"
      hardware:
        display_input: "HDMI2"
        
    - id: "ps5"
      name: "PlayStation 5"
      type: "game_console"
      hardware:
        display_input: "HDMI3"
```

### Residential: Matrix System

```yaml
video:
  matrix:
    id: "matrix-main"
    manufacturer: "atlona"
    model: "at-uhd-pro3-44m"
    connection:
      type: "ip"
      host: "192.168.1.80"
    inputs: 4
    outputs: 4
    
  sources:
    - id: "apple-tv"
      name: "Apple TV"
      hardware:
        matrix_input: 1
        
    - id: "sky-q"
      name: "Sky Q"
      hardware:
        matrix_input: 2
        
    - id: "blu-ray"
      name: "Blu-ray Player"
      hardware:
        matrix_input: 3
        
    - id: "camera-front"
      name: "Front Door Camera"
      hardware:
        matrix_input: 4
        
  zones:
    - id: "zone-living"
      name: "Living Room"
      display_id: "display-living"
      matrix:
        matrix_id: "matrix-main"
        output: 1
        
    - id: "zone-master"
      name: "Master Bedroom"
      display_id: "display-master"
      matrix:
        matrix_id: "matrix-main"
        output: 2
        
    - id: "zone-kitchen"
      name: "Kitchen"
      display_id: "display-kitchen"
      matrix:
        matrix_id: "matrix-main"
        output: 3
```

### Residential: Cinema

```yaml
video:
  zones:
    - id: "zone-cinema"
      name: "Home Cinema"
      
      displays:
        - id: "projector-cinema"
          type: "projector"
          manufacturer: "jvc"
          connection:
            type: "serial"
            device: "/dev/ttyUSB0"
            
      screen:
        device_id: "screen-cinema"
        
      audio_zone_id: "zone-cinema-audio"
      
      config:
        default_source: "apple-tv-cinema"
        projector_warmup_seconds: 30
        screen_deploy_before_projector: true
        
  sources:
    - id: "apple-tv-cinema"
      name: "Apple TV 4K"
      hardware:
        avr_input: "HDMI1"
        
    - id: "kodi"
      name: "Kodi"
      hardware:
        avr_input: "HDMI2"
        
    - id: "blu-ray-cinema"
      name: "UHD Blu-ray"
      hardware:
        avr_input: "HDMI3"
```

---

## Best Practices

### Do's

1. **Label everything** — Cables, ports, devices
2. **Use quality cables** — Certified HDMI cables for 4K/HDR
3. **Plan for sources** — More inputs than you think you need
4. **Test HDCP** — Verify all sources work through matrix
5. **Document routing** — Which input is which source
6. **Consider CEC carefully** — Can help or cause issues

### Don'ts

1. **Don't rely solely on CEC** — Always have direct control
2. **Don't mix cable grades** — Inconsistent quality causes issues
3. **Don't forget cooling** — Matrices and projectors need ventilation
4. **Don't over-complicate** — Simple is more reliable
5. **Don't forget audio sync** — Video and audio must stay in sync

---

## Related Documents

- [Audio Domain](audio.md) — Multi-room audio integration
- [Automation Specification](../automation/automation.md) — Scene integration
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring
- [Data Model: Entities](../data-model/entities.md) — Video device types
- [CCTV Integration](../integration/cctv.md) — Camera feeds as video sources
