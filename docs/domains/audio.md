---
title: Audio Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - data-model/entities.md
  - automation/automation.md
---

# Audio Domain Specification

This document specifies how Gray Logic manages multi-room audio, including zone control, source management, announcements, and integration with various audio systems.

---

## Overview

### Philosophy

Gray Logic is an **orchestration layer** for audio, not a streaming platform:

| Gray Logic Does | Gray Logic Does NOT |
|-----------------|---------------------|
| Control volume, source, power | Stream music data |
| Group/ungroup zones | Host music libraries |
| Route announcements | Process audio signals |
| Coordinate with scenes/modes | Replace Sonos/Spotify |
| Provide unified control UI | Provide music browsing UI |

### Design Principles

1. **Control, not streaming** — Gray Logic orchestrates; audio systems do the heavy lifting
2. **Zone-centric** — Users think in rooms, not devices or amp channels
3. **Announcement priority** — Safety and doorbell announcements always get through
4. **Source abstraction** — Unified source model regardless of backend system
5. **Graceful degradation** — Audio keeps working if Gray Logic is offline

### System Types Supported

| Type | Examples | Control Method |
|------|----------|----------------|
| **Streaming speakers** | Sonos, HEOS, BluOS, Chromecast | Native API |
| **Matrix amplifiers** | Monoprice, HTD, Russound, Nuvo | RS-232, IP |
| **AV receivers** | Denon, Marantz, Yamaha, Sony | IP, RS-232 |
| **Soundbars** | Sonos Beam/Arc, Samsung | Native API |
| **Professional DSP** | Biamp, QSC, Symetrix | IP, RS-232 |

---

## Device Types

### Audio Zone

A logical audio output area (room or part of room).

```yaml
AudioZone:
  id: uuid
  site_id: uuid
  name: string                      # "Kitchen", "Master Bedroom"
  slug: string                      # "kitchen", "master-bedroom"
  
  # Physical mapping
  rooms: [uuid]                     # Rooms this zone covers
  
  # Hardware mapping (one of)
  hardware:
    type: "sonos" | "matrix" | "avr" | "standalone"
    
    # For Sonos/streaming
    player_id: string | null        # Sonos player UUID
    
    # For matrix systems
    matrix_id: string | null        # Matrix device ID
    zone_number: integer | null     # Zone/channel number
    
    # For AV receivers
    avr_id: string | null           # Receiver device ID
    avr_zone: string | null         # "main", "zone2", "zone3"
  
  # Configuration
  config:
    default_source: string | null   # Default source on power-on
    max_volume: integer             # Volume limit (0-100)
    startup_volume: integer         # Volume when powered on
    announcement_volume: integer    # Volume for announcements
    fixed_volume: boolean           # Disable volume control (e.g., powered speakers)
    
  # Current state
  state:
    power: boolean
    source_id: string | null
    volume: integer                 # 0-100
    mute: boolean
    
  # Optional EQ
  eq:
    bass: integer | null            # -10 to +10
    treble: integer | null          # -10 to +10
    balance: integer | null         # -10 (left) to +10 (right)
```

### Audio Source

An input that can be played in zones.

```yaml
AudioSource:
  id: string                        # "spotify", "turntable", "tv-arc"
  name: string                      # "Spotify", "Turntable", "TV Audio"
  type: enum                        # Source type (see below)
  icon: string                      # UI icon
  
  # Hardware mapping
  hardware:
    # For matrix inputs
    matrix_input: integer | null    # Input number on matrix
    
    # For streaming
    service: string | null          # "spotify", "airplay", "tidal"
    
    # For AV receiver inputs
    avr_input: string | null        # "HDMI1", "PHONO", "TV"
  
  # Capabilities
  capabilities:
    transport: boolean              # Can play/pause/skip
    metadata: boolean               # Provides now-playing info
    favorites: boolean              # Has preset favorites
    
  # Availability
  available_in: [string] | null     # Zone IDs, null = all zones
```

**Source Types:**

| Type | Description | Examples |
|------|-------------|----------|
| `streaming` | Network streaming service | Spotify, Tidal, AirPlay |
| `line_in` | Analog line input | Turntable, CD player |
| `hdmi_arc` | TV audio return | TV audio via ARC/eARC |
| `optical` | Digital optical input | TV, game console |
| `bluetooth` | Bluetooth input | Phone pairing |
| `usb` | USB audio | USB drive, DAC |
| `tuner` | Radio tuner | FM, DAB, Internet radio |
| `internal` | Built-in source | Sonos favorites |

### Audio Matrix

Multi-zone amplifier/distribution system.

```yaml
AudioMatrix:
  id: string
  name: string
  type: string                      # "monoprice_6zone", "htd_lync"
  
  # Connection
  connection:
    type: "ip" | "serial"
    host: string | null             # IP address
    port: integer | null            # IP port or serial baud
    serial_device: string | null    # "/dev/ttyUSB0"
    
  # Capabilities
  zones: integer                    # Number of zones
  sources: integer                  # Number of inputs
  
  # Zone mapping
  zone_map:
    - zone: 1
      zone_id: "zone-kitchen"
    - zone: 2
      zone_id: "zone-dining"
```

### AV Receiver

Home theater receiver with multi-zone capability.

```yaml
AVReceiver:
  id: string
  name: string
  manufacturer: string              # "denon", "marantz", "yamaha"
  model: string
  
  # Connection
  connection:
    type: "ip" | "serial"
    host: string
    port: integer
    
  # Zones
  zones:
    main:
      zone_id: "zone-living"
    zone2:
      zone_id: "zone-patio"
      
  # Input mapping
  inputs:
    - input: "HDMI1"
      source_id: "apple-tv"
    - input: "HDMI2"
      source_id: "playstation"
    - input: "PHONO"
      source_id: "turntable"
```

---

## State Model

### Zone State

```yaml
ZoneState:
  power: boolean                    # On/off
  source_id: string | null          # Current source
  volume: integer                   # 0-100
  mute: boolean                     # Muted
  
  # Group state (for groupable systems)
  group:
    is_coordinator: boolean         # Is this the group leader
    coordinator_id: string | null   # Group leader zone ID
    members: [string]               # Zone IDs in this group
    
  # Now playing (if available)
  now_playing:
    title: string | null
    artist: string | null
    album: string | null
    artwork_url: string | null
    duration_ms: integer | null
    position_ms: integer | null
    state: "playing" | "paused" | "stopped" | "idle"
```

### Source State

```yaml
SourceState:
  available: boolean                # Source is available
  active_in: [string]               # Zone IDs currently using this source
  
  # Transport state (if capable)
  transport:
    state: "playing" | "paused" | "stopped"
    shuffle: boolean
    repeat: "off" | "one" | "all"
```

---

## Commands

### Zone Commands

#### Power Control

```yaml
# Power on
- zone_id: "zone-kitchen"
  command: "power"
  parameters:
    power: true

# Power off
- zone_id: "zone-kitchen"
  command: "power"
  parameters:
    power: false

# Toggle
- zone_id: "zone-kitchen"
  command: "power_toggle"
```

#### Volume Control

```yaml
# Set absolute volume
- zone_id: "zone-kitchen"
  command: "volume"
  parameters:
    level: 50

# Relative volume
- zone_id: "zone-kitchen"
  command: "volume_up"
  parameters:
    step: 5                         # Default: 5

- zone_id: "zone-kitchen"
  command: "volume_down"
  parameters:
    step: 5

# Mute
- zone_id: "zone-kitchen"
  command: "mute"
  parameters:
    mute: true

# Mute toggle
- zone_id: "zone-kitchen"
  command: "mute_toggle"
```

#### Source Selection

```yaml
# Select source
- zone_id: "zone-kitchen"
  command: "source"
  parameters:
    source_id: "spotify"

# Play source (power on + select)
- zone_id: "zone-kitchen"
  command: "play_source"
  parameters:
    source_id: "spotify"
    volume: 40                      # Optional: set volume too
```

### Zone Grouping

For systems that support grouping (Sonos, HEOS, etc.):

```yaml
# Group zones together
- command: "group_zones"
  parameters:
    coordinator: "zone-living"      # Leader zone
    members: ["zone-kitchen", "zone-dining"]

# Ungroup a zone
- zone_id: "zone-kitchen"
  command: "ungroup"

# Ungroup all
- command: "ungroup_all"
```

### Transport Commands

For sources with transport control:

```yaml
- zone_id: "zone-living"
  command: "play"

- zone_id: "zone-living"
  command: "pause"

- zone_id: "zone-living"
  command: "stop"

- zone_id: "zone-living"
  command: "next"

- zone_id: "zone-living"
  command: "previous"

- zone_id: "zone-living"
  command: "seek"
  parameters:
    position_ms: 60000              # Seek to 1 minute
```

### Favorites/Presets

```yaml
# Play a favorite
- zone_id: "zone-kitchen"
  command: "play_favorite"
  parameters:
    favorite_id: "morning-jazz"

# Play a playlist
- zone_id: "zone-kitchen"
  command: "play_playlist"
  parameters:
    playlist_id: "spotify:playlist:abc123"
```

---

## Announcements

### Priority System

Announcements have priority levels that determine behavior:

| Priority | Level | Behavior | Examples |
|----------|-------|----------|----------|
| `critical` | 1 | Immediate, full volume, all zones | Fire alarm, security breach |
| `high` | 2 | Interrupt music, play, resume | Doorbell, intercom call |
| `normal` | 3 | Duck music, play, restore | "Dinner is ready" |
| `low` | 4 | Only if music is quiet/off | Reminders, notifications |

### Announcement Command

```yaml
- command: "announce"
  parameters:
    # Content (one of)
    tts_text: string | null         # Text to speak
    audio_url: string | null        # Audio file URL
    chime: string | null            # Built-in chime: "doorbell", "alert", "info"
    
    # Targeting
    zones: [string] | null          # Specific zones, null = all
    
    # Priority
    priority: "critical" | "high" | "normal" | "low"
    
    # Volume (optional)
    volume: integer | null          # Override volume, null = zone default
    
    # Behavior
    duck_percent: integer           # Reduce music to X% (default: 20)
    resume: boolean                 # Resume music after (default: true)
```

### Examples

#### Doorbell Announcement

```yaml
- command: "announce"
  parameters:
    chime: "doorbell"
    tts_text: "Someone is at the front door"
    zones: ["zone-kitchen", "zone-living", "zone-master"]
    priority: "high"
    volume: 70
```

#### General TTS

```yaml
- command: "announce"
  parameters:
    tts_text: "Dinner is ready"
    zones: null                     # All zones
    priority: "normal"
```

#### Emergency Alert

```yaml
- command: "announce"
  parameters:
    chime: "alert"
    tts_text: "Fire alarm activated. Please evacuate."
    zones: null
    priority: "critical"
    volume: 100
```

### Announcement Flow

```
1. Request received with priority
2. For each target zone:
   a. If priority > current activity:
      - Store current state (volume, source)
      - Duck or pause music
      - Set announcement volume
      - Play announcement
      - Wait for completion
      - Restore previous state (if resume=true)
   b. If priority <= current activity:
      - Queue or skip based on priority
```

---

## Integration Patterns

### Sonos Integration

```yaml
# Bridge configuration
bridge:
  id: "sonos-bridge"
  type: "sonos"
  
  discovery:
    method: "ssdp"                  # Auto-discover
    # Or manual:
    # players:
    #   - host: "192.168.1.50"
    
  polling_interval_ms: 1000
  
  # TTS settings
  tts:
    provider: "piper"               # Local Piper TTS
    voice: "en_GB-alba-medium"
    cache_dir: "/var/cache/graylogic/tts"
```

**Capabilities:**
- Zone grouping/ungrouping
- Favorites and playlists
- Now playing metadata
- TTS announcements (via Sonos audio clip API or stream)
- Volume, mute, transport control

### Matrix Audio Systems

#### Monoprice 6-Zone

```yaml
bridge:
  id: "matrix-bridge"
  type: "monoprice_6zone"
  
  connection:
    type: "serial"
    device: "/dev/ttyUSB0"
    baud: 9600
    
  zones:
    - number: 1
      zone_id: "zone-kitchen"
    - number: 2
      zone_id: "zone-dining"
    - number: 3
      zone_id: "zone-living"
    - number: 4
      zone_id: "zone-master"
    - number: 5
      zone_id: "zone-patio"
    - number: 6
      zone_id: "zone-office"
      
  sources:
    - number: 1
      source_id: "streaming-1"
    - number: 2
      source_id: "turntable"
    - number: 3
      source_id: "tv-audio"
```

**Capabilities:**
- Per-zone source selection
- Per-zone volume, bass, treble, balance
- Per-zone power
- No grouping (hardware limitation)
- No transport control (sources are external)

#### HTD Lync

```yaml
bridge:
  id: "htd-bridge"
  type: "htd_lync"
  
  connection:
    type: "ip"
    host: "192.168.1.60"
    port: 10006
```

### AV Receivers

#### Denon/Marantz (HEOS)

```yaml
bridge:
  id: "avr-bridge"
  type: "denon"
  
  connection:
    type: "ip"
    host: "192.168.1.70"
    port: 23                        # Telnet control
    
  zones:
    main:
      zone_id: "zone-living"
    zone2:
      zone_id: "zone-patio"
```

**Capabilities:**
- Multi-zone control
- Input selection
- Volume, mute
- Surround mode selection
- HEOS streaming integration

#### Yamaha MusicCast

```yaml
bridge:
  id: "yamaha-bridge"
  type: "yamaha_musiccast"
  
  connection:
    type: "ip"
    host: "192.168.1.71"
```

---

## Automation Integration

### Scene Integration

Audio can be part of any scene:

```yaml
# Cinema Mode scene
actions:
  # Set AV receiver to correct input
  - zone_id: "zone-living"
    command: "source"
    parameters:
      source_id: "apple-tv"
      
  # Set volume
  - zone_id: "zone-living"
    command: "volume"
    parameters:
      level: 35
      
  # Turn off other zones
  - zone_id: "zone-kitchen"
    command: "power"
    parameters:
      power: false
    parallel: true
```

```yaml
# Party Mode scene
actions:
  # Group all indoor zones
  - command: "group_zones"
    parameters:
      coordinator: "zone-living"
      members: ["zone-kitchen", "zone-dining"]
      
  # Play party playlist
  - zone_id: "zone-living"
    command: "play_playlist"
    parameters:
      playlist_id: "party-mix"
      
  # Set volume
  - zone_id: "zone-living"
    command: "volume"
    parameters:
      level: 60
```

### Mode Integration

Modes affect audio behavior:

```yaml
# Night mode
mode: "night"
behaviours:
  audio:
    max_volume: 30                  # Cap all zones at 30%
    announcements: false            # Disable non-critical announcements
    
# Away mode
mode: "away"
behaviours:
  audio:
    power_off_all: true             # Turn off all audio
    announcements_only: true        # Only allow security announcements
```

### Event-Driven Audio

```yaml
# Doorbell → Announcement
trigger:
  type: "device_state_changed"
  source:
    device_id: "doorbell-front"
    event: "pressed"
execute:
  type: "actions"
  actions:
    - command: "announce"
      parameters:
        chime: "doorbell"
        tts_text: "Someone is at the front door"
        priority: "high"
        zones: ["zone-kitchen", "zone-living"]

# Motion in room → Resume music (if was playing)
trigger:
  type: "motion_detected"
  source:
    room_id: "room-kitchen"
execute:
  type: "actions"
  actions:
    - zone_id: "zone-kitchen"
      command: "resume"
conditions:
  - type: "mode"
    operator: "eq"
    value: "home"

# Last person leaves → All audio off
trigger:
  type: "presence_changed"
  source:
    event: "last_person_left"
execute:
  type: "actions"
  actions:
    - command: "all_off"
      parameters:
        domain: "audio"
```

### Intercom/Paging

```yaml
# Intercom call from door station
trigger:
  type: "intercom_call"
  source:
    device_id: "doorstation-front"
execute:
  type: "actions"
  actions:
    # Announce incoming call
    - command: "announce"
      parameters:
        chime: "intercom"
        tts_text: "Incoming call from front door"
        priority: "high"
        
    # Route audio to specific zones if configured
    - command: "route_intercom"
      parameters:
        source: "doorstation-front"
        zones: ["zone-kitchen", "zone-living"]
```

---

## Commercial Audio

### Background Music

Commercial spaces often need background music management:

```yaml
BackgroundMusic:
  zone_groups:
    - id: "public-areas"
      zones: ["zone-reception", "zone-cafe", "zone-corridor"]
      
  schedules:
    - name: "Business Hours"
      time: "08:00-18:00"
      days: ["mon", "tue", "wed", "thu", "fri"]
      source_id: "background-playlist"
      volume: 25
      
    - name: "Evening"
      time: "18:00-22:00"
      days: ["mon", "tue", "wed", "thu", "fri"]
      source_id: "evening-playlist"
      volume: 20
```

### Paging System

```yaml
Paging:
  zones:
    all_call: ["zone-reception", "zone-office", "zone-warehouse"]
    office_only: ["zone-office-1", "zone-office-2"]
    
  priority: "high"                  # Paging priority level
  
  # Pre-announce chime
  chime: "paging-tone"
  chime_duration_ms: 1000
```

### Emergency Announcements

```yaml
EmergencyAudio:
  # Fire alarm integration
  fire_alarm:
    trigger:
      type: "fire_alarm_event"
      event: "activated"
    action:
      command: "announce"
      parameters:
        audio_url: "/audio/fire-evacuation.mp3"
        zones: null                 # All zones
        priority: "critical"
        volume: 100
        loop: true                  # Keep playing until cancelled
        
  # Lockdown
  lockdown:
    trigger:
      type: "security_event"
      event: "lockdown"
    action:
      command: "announce"
      parameters:
        audio_url: "/audio/lockdown.mp3"
        zones: null
        priority: "critical"
```

---

## MQTT Topics

### Zone State (Bridge → Core)

```yaml
topic: graylogic/bridge/audio-bridge/state/{zone_id}
payload:
  zone_id: "zone-kitchen"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    power: true
    source_id: "spotify"
    volume: 45
    mute: false
  now_playing:
    title: "Song Title"
    artist: "Artist Name"
    state: "playing"
```

### Zone Command (Core → Bridge)

```yaml
topic: graylogic/bridge/audio-bridge/command/{zone_id}
payload:
  zone_id: "zone-kitchen"
  command: "volume"
  parameters:
    level: 50
  request_id: "req-123"
```

### Announcement Command

```yaml
topic: graylogic/bridge/audio-bridge/command/announce
payload:
  command: "announce"
  parameters:
    tts_text: "Doorbell rang"
    zones: ["zone-kitchen", "zone-living"]
    priority: "high"
  request_id: "req-456"
```

---

## PHM Integration

Gray Logic applies Predictive Health Monitoring to audio systems to detect equipment degradation and communication issues. See [PHM Specification](../intelligence/phm.md) for the full PHM framework.

### PHM Value for Audio

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Amplifiers | ★★★☆☆ | Temperature, current draw, response time |
| Matrix systems | ★★☆☆☆ | Command latency, error rate |
| Streaming players | ★★☆☆☆ | Connectivity, response time |
| AV receivers | ★★☆☆☆ | Temperature, response time |

### Zone Health State

```yaml
ZoneHealth:
  zone_id: "zone-kitchen"
  status: "online" | "offline" | "degraded"
  last_seen: timestamp
  health_score: 95                        # 0-100
  
  issues:
    - type: "speaker_offline"
      severity: "warning"
      message: "Kitchen speaker not responding"
      since: "2026-01-12T10:00:00Z"
      
    - type: "source_unavailable"
      severity: "info"
      source_id: "spotify"
      message: "Spotify service unavailable"
```

### Monitored Parameters

| Parameter | Source | Baseline Method | Alert Indicator |
|-----------|--------|-----------------|-----------------|
| Amplifier temperature | Device query | Context-aware (by power) | Overheating, cooling issues |
| Power consumption | Power meter | Context-aware (by volume) | Component degradation |
| Command latency | Response timing | Rolling mean | Communication issues |
| Error rate | Command failures | Rolling mean | Hardware/connection problems |
| Reconnect count | Connection events | Daily count | Network instability |

### PHM Configuration

```yaml
phm_audio:
  devices:
    # Amplifier monitoring
    - device_id: "amp-zone-1"
      type: "amplifier"
      parameters:
        - name: "heatsink_temp_c"
          source: "device:temperature"
          baseline_method: "context_aware"
          context_parameter: "output_power_percent"
          deviation_threshold_c: 15
          absolute_limit_c: 85
          severity: "warning"
          recommendation: "Amplifier overheating - check ventilation"
          
        - name: "idle_current_a"
          source: "device:current"
          baseline_method: "rolling_mean"
          window_hours: 168
          deviation_threshold_percent: 20
          severity: "info"
          recommendation: "Idle current deviation - monitor for degradation"
          
        - name: "response_latency_ms"
          baseline_method: "rolling_percentile_95"
          threshold_ms: 500
          severity: "warning"
          recommendation: "Response latency elevated - check network/device"
          
    # Matrix system monitoring  
    - device_id: "matrix-main"
      type: "audio_matrix"
      parameters:
        - name: "command_error_rate"
          calculation: "errors / total_commands"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 200
          severity: "warning"
          recommendation: "Matrix error rate elevated - check RS-232/IP connection"
          
        - name: "reconnect_count"
          baseline_method: "daily_count"
          threshold_per_day: 3
          severity: "warning"
          recommendation: "Frequent reconnections - check network stability"
          
    # Streaming player monitoring
    - device_id: "sonos-kitchen"
      type: "streaming_player"
      parameters:
        - name: "offline_duration_minutes"
          threshold_minutes: 15
          severity: "warning"
          recommendation: "Speaker offline - check power and network"
          
        - name: "wifi_signal_strength"
          threshold_dbm: -70
          severity: "info"
          recommendation: "Weak WiFi signal - consider network improvements"
```

### Alerting

```yaml
phm_audio_alerts:
  # Device offline
  - condition: "zone_offline"
    for_minutes: 15
    severity: "warning"
    message: "Audio zone offline: ${zone_name}"
    actions:
      - notify:
          channels: ["push"]
      - log_phm: true
      
  # Amplifier overheating
  - condition: "amplifier_overtemp"
    severity: "warning"
    message: "Amplifier ${device_name} temperature elevated (${temp}°C)"
    actions:
      - notify:
          channels: ["push", "email"]
      - log_phm: true
      - recommended_action: "Check ventilation and ambient temperature"
      
  # Communication issues
  - condition: "high_error_rate"
    severity: "warning"
    message: "Audio device ${device_name} communication errors elevated"
    actions:
      - notify:
          channels: ["email"]
      - log_phm: true
```

### Dashboard Metrics

```yaml
phm_audio_dashboard:
  summary:
    total_zones: 8
    online: 7
    offline: 0
    degraded: 1
    
  health_score: 91
  
  issues:
    - zone: "zone-patio"
      status: "degraded"
      issue: "Weak WiFi signal (-72 dBm)"
      
  equipment_health:
    - device: "amp-zone-1"
      type: "amplifier"
      health_score: 95
      status: "healthy"
      
    - device: "matrix-main"
      type: "audio_matrix"
      health_score: 88
      status: "healthy"
      note: "Slight increase in command latency"
```

---

## Configuration Examples

### Residential: Sonos-Based

```yaml
audio:
  type: "sonos"
  
  zones:
    - id: "zone-living"
      name: "Living Room"
      rooms: ["room-living"]
      hardware:
        type: "sonos"
        player_id: "RINCON_XXX"
      config:
        max_volume: 80
        announcement_volume: 60
        
    - id: "zone-kitchen"
      name: "Kitchen"
      rooms: ["room-kitchen"]
      hardware:
        type: "sonos"
        player_id: "RINCON_YYY"
      config:
        max_volume: 70
        announcement_volume: 65
        
  sources:
    - id: "spotify"
      name: "Spotify"
      type: "streaming"
      icon: "spotify"
      
    - id: "tv-audio"
      name: "TV Audio"
      type: "hdmi_arc"
      available_in: ["zone-living"]
```

### Residential: Matrix + Streaming

```yaml
audio:
  bridges:
    - id: "matrix-bridge"
      type: "monoprice_6zone"
      connection:
        type: "serial"
        device: "/dev/ttyUSB0"
        
    - id: "streaming-bridge"
      type: "sonos"
      
  zones:
    # Matrix zones (whole-home)
    - id: "zone-kitchen"
      name: "Kitchen"
      hardware:
        type: "matrix"
        matrix_id: "matrix-bridge"
        zone_number: 1
        
    # Sonos zone (living room with better quality)
    - id: "zone-living"
      name: "Living Room"
      hardware:
        type: "sonos"
        player_id: "RINCON_XXX"
        
  sources:
    # Streaming source feeds matrix input 1
    - id: "streaming"
      name: "Streaming"
      type: "streaming"
      hardware:
        matrix_input: 1
        
    - id: "turntable"
      name: "Turntable"
      type: "line_in"
      hardware:
        matrix_input: 2
```

### Commercial: DSP-Based

```yaml
audio:
  type: "biamp_tesira"
  
  connection:
    host: "192.168.1.100"
    port: 23
    
  zones:
    - id: "zone-reception"
      name: "Reception"
      hardware:
        type: "dsp"
        instance_tag: "Zone1Level"
      config:
        max_volume: 60
        fixed_volume: false
        
    - id: "zone-meeting-1"
      name: "Meeting Room 1"
      hardware:
        type: "dsp"
        instance_tag: "Zone2Level"
      config:
        max_volume: 70
        
  paging:
    enabled: true
    priority_tag: "PagePriority"
    chime_tag: "PageChime"
```

---

## Best Practices

### Do's

1. **Set sensible volume limits** — Prevent accidental full-volume playback
2. **Configure announcement volumes** — Doorbell should be heard over music
3. **Use zone groups for parties** — Synchronized playback across rooms
4. **Integrate with presence** — Turn off audio when nobody's home
5. **Test announcements** — Verify doorbell/TTS works before relying on it

### Don'ts

1. **Don't rely on audio for safety** — Fire alarms should have dedicated sounders
2. **Don't set critical priority lightly** — Reserve for true emergencies
3. **Don't forget volume limits in night mode** — Protect sleeping family members
4. **Don't bypass local playback** — Sonos should work even if Gray Logic is down

---

## Related Documents

- [Data Model: Entities](../data-model/entities.md) — AudioZone entity definition
- [Automation Specification](../automation/automation.md) — Scene and mode integration
- [PHM Specification](../intelligence/phm.md) — Predictive health monitoring framework
- [Access Control Integration](../integration/access-control.md) — Doorbell announcements
- [REST API Specification](../interfaces/api.md) — Audio API endpoints
- [Core Internals](../architecture/core-internals.md) — Audio service architecture
