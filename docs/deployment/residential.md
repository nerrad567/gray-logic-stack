---
title: Residential Deployment Guide
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/vision.md
  - overview/principles.md
  - architecture/system-overview.md
---

# Residential Deployment Guide

This guide covers deploying Gray Logic in high-end residential properties — single-family homes, estates, and properties with pools, spas, and leisure facilities. It addresses the unique requirements of residential installations versus commercial deployments.

---

## Overview

### Residential vs Commercial

| Aspect | Residential | Commercial |
|--------|-------------|------------|
| **Users** | Family/guests (few, known) | Employees/visitors (many, changing) |
| **Schedules** | Flexible, lifestyle-driven | Fixed occupancy hours |
| **Lighting** | Scene-based, mood-driven | Occupancy/daylight-based |
| **HVAC** | Comfort zones, pool/spa | VAV/FCU zones, efficiency |
| **Access** | Front door, garage, gates | Multiple entries, card readers |
| **Audio/Video** | Multi-room, cinema | Meeting rooms, announcements |
| **Energy** | Cost awareness | Cost + compliance reporting |
| **Network** | Home network | Enterprise VLANs |
| **Support** | Owner self-service | IT team + facility manager |

### Target Properties

Gray Logic residential deployment suits:

- High-end new builds (£500k+)
- Deep refurbishments with integrated systems
- Properties with pools, spas, or leisure facilities
- Estates with multiple buildings (main house, guest house, pool house)
- Properties requiring integrated security and CCTV
- Homes with multi-room audio/video distribution

---

## Architecture

### Residential Deployment Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          HOME NETWORK                                │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    GRAY LOGIC SERVER                         │    │
│  │  • Gray Logic Core                                          │    │
│  │  • Voice Bridge                                             │    │
│  │  • InfluxDB (PHM)                                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    CONTROL NETWORK                            │    │
│  │  • KNX/IP Interface                                          │    │
│  │  • DALI Gateways                                             │    │
│  │  • Modbus Gateways (pool plant)                              │    │
│  │  • Audio Matrix                                              │    │
│  │  • Video Matrix                                              │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    GUEST NETWORK                              │    │
│  │  • Visitor WiFi (isolated)                                  │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    SECURITY NETWORK                           │    │
│  │  • CCTV NVR                                                  │    │
│  │  • Alarm Panel                                              │    │
│  │  • Access Control                                           │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

### Network Segregation

For residential deployments, network segregation is simpler than commercial:

```yaml
network_segregation:
  # Main network (trusted)
  main:
    devices:
      - "Gray Logic Server"
      - "Wall Panels"
      - "Owner devices"
      - "Smart TVs"
    isolation: false
    
  # Control network (isolated)
  control:
    devices:
      - "KNX/IP Interface"
      - "DALI Gateways"
      - "Modbus Gateways"
      - "Audio/Video Matrices"
    isolation: true
    vlan: 100
    
  # Security network (isolated)
  security:
    devices:
      - "CCTV NVR"
      - "Alarm Panel"
      - "Access Control"
    isolation: true
    vlan: 200
    no_internet: true              # Security devices don't need internet
    
  # Guest network (isolated)
  guest:
    devices:
      - "Visitor WiFi"
    isolation: true
    vlan: 300
    internet_only: true
```

---

## Server Requirements

### Hardware

```yaml
server_requirements:
  # Minimum (small home, basic control)
  minimum:
    cpu: "4 cores"
    ram: "4GB"
    storage: "64GB SSD"
    network: "1x Gigabit"
    
  # Recommended (typical home)
  recommended:
    cpu: "8 cores"
    ram: "8GB"
    storage: "256GB SSD"
    network: "1x Gigabit"
    
  # High-end (large home, pool, multiple buildings)
  high_end:
    cpu: "12+ cores"
    ram: "16GB"
    storage: "512GB SSD"
    network: "2x Gigabit (bonded)"
    
  # Form factor
  options:
    - "Mini PC (NUC-style)"
    - "Rack-mount (if comms cabinet)"
    - "Wall-mount (if space constrained)"
```

### Location

```yaml
server_location:
  options:
    - "Comms cabinet (preferred)"
    - "Server room (if available)"
    - "Utility room (ventilated)"
    - "Garage (temperature controlled)"
    
  requirements:
    temperature: "0-35°C"
    humidity: "10-80%"
    ventilation: "Adequate airflow"
    power: "UPS protected (recommended)"
    access: "Secure, owner-accessible"
```

### Power Protection

```yaml
power_protection:
  ups:
    recommended: true
    runtime_minutes: 30             # Allow graceful shutdown
    battery_type: "Lead-acid or Li-ion"
    
  # Critical systems
  critical:
    - "Gray Logic Core"
    - "KNX/IP Interface"
    - "Alarm Panel (if integrated)"
    
  # Non-critical (can shut down)
  non_critical:
    - "Voice Bridge"
    - "InfluxDB (PHM data)"
    - "Audio/Video Matrices"
```

---

## Site Structure

### Typical Residential Hierarchy

```yaml
site_structure:
  site:
    id: "site-main-house"
    name: "Main House"
    type: "residential"
    
  areas:
    - id: "area-ground-floor"
      name: "Ground Floor"
      type: "floor"
      
    - id: "area-first-floor"
      name: "First Floor"
      type: "floor"
      
    - id: "area-pool-house"
      name: "Pool House"
      type: "outbuilding"
      
    - id: "area-garage"
      name: "Garage"
      type: "garage"
      
  rooms:
    # Ground floor
    - id: "room-living"
      name: "Living Room"
      area_id: "area-ground-floor"
      type: "living_room"
      
    - id: "room-kitchen"
      name: "Kitchen"
      area_id: "area-ground-floor"
      type: "kitchen"
      
    - id: "room-dining"
      name: "Dining Room"
      area_id: "area-ground-floor"
      type: "dining_room"
      
    - id: "room-cinema"
      name: "Cinema Room"
      area_id: "area-ground-floor"
      type: "cinema"
      
    # First floor
    - id: "room-master-bedroom"
      name: "Master Bedroom"
      area_id: "area-first-floor"
      type: "bedroom"
      
    - id: "room-guest-bedroom"
      name: "Guest Bedroom"
      area_id: "area-first-floor"
      type: "bedroom"
      
    # Pool house
    - id: "room-pool-house"
      name: "Pool House"
      area_id: "area-pool-house"
      type: "pool_house"
      
    - id: "room-plant-room"
      name: "Plant Room"
      area_id: "area-ground-floor"
      type: "plant_room"
```

---

## Lighting Control

### Scene-Based Lighting

Residential lighting is scene-driven, not occupancy-driven:

```yaml
lighting_scenes:
  # Living room scenes
  - id: "scene-living-day"
    name: "Day"
    room_id: "room-living"
    brightness: 100
    color_temp: 4000
    
  - id: "scene-living-evening"
    name: "Evening"
    room_id: "room-living"
    brightness: 60
    color_temp: 2700
    
  - id: "scene-living-cinema"
    name: "Cinema"
    room_id: "room-living"
    brightness: 5
    color_temp: 2000
    voice_phrase: "cinema mode"
    
  # Bedroom scenes
  - id: "scene-bedroom-night"
    name: "Night"
    room_id: "room-master-bedroom"
    brightness: 10
    color_temp: 2000
    
  - id: "scene-bedroom-reading"
    name: "Reading"
    room_id: "room-master-bedroom"
    brightness: 80
    color_temp: 3000
```

### Circadian Lighting

```yaml
circadian_lighting:
  enabled: true
  
  schedule:
    morning:
      start: "sunrise-30"
      color_temp: 2000              # Warm wake-up
      brightness: 20
      
    day:
      start: "09:00"
      color_temp: 4000               # Cool, energizing
      brightness: 100
      
    evening:
      start: "sunset"
      color_temp: 2700               # Warm, relaxing
      brightness: 60
      
    night:
      start: "22:00"
      color_temp: 2000               # Very warm
      brightness: 10
```

### Exterior Lighting

```yaml
exterior_lighting:
  # Security lighting
  security:
    - device_id: "light-front-door"
      trigger: "motion"
      timeout_minutes: 5
      brightness: 100
      
  # Path lighting
  paths:
    - device_id: "light-driveway"
      schedule: "sunset to 23:00"
      brightness: 50
      
  # Garden lighting
  garden:
    - device_id: "light-garden-1"
      schedule: "sunset to sunrise"
      brightness: 30
      color: "warm_white"
```

---

## Climate Control

### Zone-Based Heating

```yaml
climate_zones:
  - zone_id: "zone-ground-floor"
    name: "Ground Floor"
    rooms: ["room-living", "room-kitchen", "room-dining"]
    heating:
      type: "underfloor"
      setpoints:
        occupied: 21
        unoccupied: 18
        night: 16
        
  - zone_id: "zone-first-floor"
    name: "First Floor"
    rooms: ["room-master-bedroom", "room-guest-bedroom"]
    heating:
      type: "radiators"
      setpoints:
        occupied: 20
        unoccupied: 18
        night: 17
        
  - zone_id: "zone-pool-house"
    name: "Pool House"
    rooms: ["room-pool-house"]
    heating:
      type: "heat_pump"
      setpoints:
        occupied: 24
        unoccupied: 18
```

### Pool/Spa Climate

```yaml
pool_climate:
  pool_house:
    room_id: "room-pool-house"
    
    # Air temperature
    air_temp:
      setpoint: 28
      min: 18
      max: 32
      
    # Humidity control
    humidity:
      target: 60
      max: 70
      dehumidifier: "dehumidifier-pool-house"
      
    # Ventilation
    ventilation:
      fans: ["fan-pool-house-1", "fan-pool-house-2"]
      schedule: "continuous when pool in use"
```

### Smart Scheduling

```yaml
smart_scheduling:
  # Presence-based
  presence_aware:
    enabled: true
    occupancy_sensors: ["pir-living", "pir-kitchen"]
    unoccupied_delay_minutes: 30
    
  # Pre-conditioning
  pre_conditioning:
    enabled: true
    arrival_time: "17:30"
    start_heating_minutes: 60
    
  # Away mode
  away_mode:
    setpoints:
      heating: 16
      cooling: 28
    notify_on_return: true
```

---

## Audio/Video

### Multi-Room Audio

```yaml
audio_zones:
  - zone_id: "audio-living"
    name: "Living Room"
    room_id: "room-living"
    sources:
      - "Spotify"
      - "Apple Music"
      - "Local library"
      - "Radio"
      
  - zone_id: "audio-kitchen"
    name: "Kitchen"
    room_id: "room-kitchen"
    sources:
      - "Spotify"
      - "Radio"
      
  - zone_id: "audio-master-bedroom"
    name: "Master Bedroom"
    room_id: "room-master-bedroom"
    sources:
      - "Spotify"
      - "Local library"
      
  # Grouping
  groups:
    - name: "Downstairs"
      zones: ["audio-living", "audio-kitchen"]
      
    - name: "Everywhere"
      zones: ["audio-living", "audio-kitchen", "audio-master-bedroom"]
```

### Cinema Room

```yaml
cinema_room:
  room_id: "room-cinema"
  
  # Video
  video:
    projector: "projector-cinema"
    screen: "screen-cinema"
    sources:
      - "Apple TV"
      - "Blu-ray"
      - "Streaming"
      
  # Audio
  audio:
    system: "7.1 surround"
    zones: ["audio-cinema"]
    
  # Lighting integration
  lighting:
    scene: "scene-cinema"
    dim_on_start: true
    restore_on_stop: true
    
  # Climate
  climate:
    setpoint: 20
    fan: "quiet"
```

### Announcements

```yaml
announcements:
  # Doorbell
  doorbell:
    text: "Someone is at the front door"
    rooms: ["room-living", "room-kitchen"]
    priority: "high"
    interrupt_audio: true
    
  # General announcements
  general:
    voice_command: "announce {text}"
    rooms: "all"
    priority: "normal"
```

---

## Security & Access

### Alarm Integration

```yaml
alarm_integration:
  panel_id: "alarm-main"
  protocol: "texecom"               # or galaxy, pyronix
  
  # Zones
  zones:
    - zone_id: "zone-front-door"
      name: "Front Door"
      type: "door_contact"
      
    - zone_id: "zone-ground-floor"
      name: "Ground Floor PIRs"
      type: "pir"
      
  # Modes
  modes:
    - mode_id: "mode-away"
      arm_type: "full"
      
    - mode_id: "mode-night"
      arm_type: "part"
      exclude_zones: ["zone-first-floor"]
      
  # Automation
  automation:
    on_arm_away:
      - scene: "scene-all-off"
      - mode: "away"
      - notify: true
      
    on_alarm:
      - scene: "scene-all-lights-on"
      - notify: ["owner", "security_company"]
      - record_cctv: true
```

### Access Control

```yaml
access_control:
  # Front door
  front_door:
    type: "video_intercom"
    device_id: "intercom-front"
    integration: "2n"                # or doorbird, akuvox
    
    # Unlock methods
    unlock_methods:
      - "app"
      - "keypad"
      - "rfid_tag"
      - "voice_command"
      
    # Automation
    on_ring:
      - tts: "Someone is at the front door"
      - show_on_wall_panels: true
      - record_cctv: true
      
  # Garage door
  garage:
    type: "garage_door"
    device_id: "garage-main"
    
    # Safety
    safety:
      auto_close_minutes: 10
      obstruction_sensor: true
      
    # Automation
    on_open:
      - scene: "scene-garage-lights-on"
      
  # Gate
  gate:
    type: "driveway_gate"
    device_id: "gate-driveway"
    
    # Methods
    open_methods:
      - "rfid_tag"
      - "app"
      - "keypad"
      - "anpr"                       # Optional
```

### CCTV Integration

```yaml
cctv_integration:
  nvr_id: "nvr-main"
  protocol: "onvif"
  
  # Cameras
  cameras:
    - camera_id: "cam-front-door"
      name: "Front Door"
      location: "front_door"
      recording: "motion"
      
    - camera_id: "cam-driveway"
      name: "Driveway"
      location: "driveway"
      recording: "continuous"
      
    - camera_id: "cam-garden"
      name: "Garden"
      location: "garden"
      recording: "motion"
      
  # Automation
  automation:
    on_motion:
      - notify: true
      - record: true
      
    on_doorbell:
      - show_on_wall_panels: true
      - record: true
```

---

## Pool & Spa

### Pool Control

```yaml
pool_control:
  pool_id: "pool-main"
  
  # Filtration
  filtration:
    pump: "pump-pool-filtration"
    schedule:
      summer: "06:00-22:00"
      winter: "08:00-18:00"
    runtime_hours: 8
    
  # Heating
  heating:
    heat_pump: "heatpump-pool"
    setpoint: 28
    min: 18
    max: 32
    schedule: "on_demand"
    
  # Chemistry
  chemistry:
    ph_sensor: "sensor-pool-ph"
    chlorine_sensor: "sensor-pool-chlorine"
    dosing_pump: "pump-pool-dosing"
    
  # Cover
  cover:
    device_id: "cover-pool"
    auto_close: true
    close_time: "sunset"
    
  # Lighting
  lighting:
    device_id: "light-pool"
    schedule: "sunset to 23:00"
    color: "blue"
```

### Spa Control

```yaml
spa_control:
  spa_id: "spa-main"
  
  # Heating
  heating:
    heater: "heater-spa"
    setpoint: 38
    min: 35
    max: 40
    
  # Jets
  jets:
    pump: "pump-spa-jets"
    control: "manual"
    
  # Lighting
  lighting:
    device_id: "light-spa"
    color: "multi_color"
    scenes:
      - "relax"
      - "energize"
      - "romantic"
```

### Pool House Integration

```yaml
pool_house:
  room_id: "room-pool-house"
  
  # Climate
  climate:
    heating: "heat_pump"
    setpoint: 24
    dehumidifier: true
    
  # Lighting
  lighting:
    scenes:
      - "day"
      - "evening"
      - "party"
      
  # Audio
  audio:
    zone_id: "audio-pool-house"
    sources: ["Spotify", "Radio"]
    
  # Automation
  automation:
    on_pool_use:
      - scene: "scene-pool-house-day"
      - audio: "play_radio"
```

---

## Energy Management

### Energy Monitoring

```yaml
energy_monitoring:
  # Main meter
  main_meter:
    device_id: "meter-main"
    type: "electrical"
    location: "consumer_unit"
    
  # Solar
  solar:
    device_id: "inverter-solar"
    type: "solar_pv"
    monitoring: true
    
  # Battery storage (if present)
  battery:
    device_id: "battery-storage"
    type: "battery"
    monitoring: true
    
  # Device-level monitoring
  device_monitoring:
    - device_id: "heatpump-main"
      ct_clamp: "ct-heatpump"
      
    - device_id: "pool-pump"
      ct_clamp: "ct-pool-pump"
```

### Energy Optimization

```yaml
energy_optimization:
  # Solar self-consumption
  solar_optimization:
    enabled: true
    strategies:
      - "charge_battery_during_peak"
      - "run_pool_pump_during_solar"
      - "pre_heat_during_solar"
      
  # Time-of-use
  time_of_use:
    enabled: false                  # If TOU tariffs available
    peak_hours: ["16:00-20:00"]
    strategies:
      - "avoid_peak_usage"
      - "pre_heat_before_peak"
      
  # Load shifting
  load_shifting:
    enabled: true
    devices:
      - "pool_pump"
      - "ev_charger"                # If present
```

---

## Modes & Automation

### System Modes

```yaml
system_modes:
  - mode_id: "mode-home"
    name: "Home"
    description: "Normal operation"
    behaviors:
      lighting: "scene_based"
      climate: "comfort"
      security: "disarmed"
      audio: "available"
      
  - mode_id: "mode-away"
    name: "Away"
    description: "Property unoccupied"
    behaviors:
      lighting: "security_simulation"
      climate: "setback"
      security: "armed_full"
      audio: "off"
      
  - mode_id: "mode-night"
    name: "Night"
    description: "Sleeping hours"
    behaviors:
      lighting: "minimal"
      climate: "night_setback"
      security: "armed_partial"
      audio: "off"
      voice: "disabled"             # Don't wake people
      
  - mode_id: "mode-holiday"
    name: "Holiday"
    description: "Extended absence"
    behaviors:
      lighting: "security_simulation"
      climate: "frost_protection"
      security: "armed_full"
      pool: "maintenance_mode"
```

### Automation Examples

```yaml
automation_examples:
  # Good morning
  - name: "Good Morning"
    trigger:
      type: "schedule"
      time: "07:00"
      days: ["monday", "tuesday", "wednesday", "thursday", "friday"]
    actions:
      - mode: "mode-home"
      - scene: "scene-bedroom-day"
      - climate: "pre_heat"
      - tts: "Good morning. The weather today is {weather_summary}"
      
  # Good night
  - name: "Good Night"
    trigger:
      type: "voice"
      phrase: "good night"
    actions:
      - scene: "scene-all-off"
      - mode: "mode-night"
      - security: "arm_partial"
      - pool: "cover_close"
      
  # Arrival home
  - name: "Arrival Home"
    trigger:
      type: "presence"
      event: "arrival"
    actions:
      - mode: "mode-home"
      - scene: "scene-welcome"
      - climate: "comfort_setpoint"
      - tts: "Welcome home"
      
  # Departure
  - name: "Departure"
    trigger:
      type: "presence"
      event: "departure"
    actions:
      - mode: "mode-away"
      - scene: "scene-all-off"
      - security: "arm_full"
      - pool: "cover_close"
```

---

## Voice Control

### Voice Setup

```yaml
voice_setup:
  # Microphones
  microphones:
    - room_id: "room-living"
      device: "/dev/audio0"
      
    - room_id: "room-kitchen"
      device: "/dev/audio1"
      
    - room_id: "room-master-bedroom"
      device: "/dev/audio2"
      
  # Wake word
  wake_word:
    phrase: "hey gray"
    sensitivity: 0.7
    
  # Voice commands
  voice_commands:
    - "turn on the {room} lights"
    - "cinema mode"
    - "good night"
    - "set temperature to {degrees}"
    - "what's the temperature"
```

### Voice Integration

```yaml
voice_integration:
  # Scenes
  scene_activation:
    - scene: "scene-cinema"
      phrases: ["cinema mode", "movie time"]
      
    - scene: "scene-good-night"
      phrases: ["good night", "night mode"]
      
  # Device control
  device_control:
    - device: "lights"
      phrases: ["turn on the lights", "lights on"]
      
    - device: "blinds"
      phrases: ["open the blinds", "close the blinds"]
      
  # Status queries
  status_queries:
    - query: "temperature"
      phrases: ["what's the temperature", "how warm is it"]
```

---

## Commissioning

### Pre-Installation

- [ ] Site survey completed
- [ ] Network requirements confirmed
- [ ] Server location confirmed
- [ ] Power and UPS confirmed
- [ ] KNX/DALI project files available
- [ ] Device schedule prepared
- [ ] Owner preferences documented

### Installation

- [ ] Server installed and networked
- [ ] Gray Logic Core installed
- [ ] KNX/DALI gateways connected
- [ ] All devices discovered and named
- [ ] Areas and rooms defined
- [ ] Scenes created and tested
- [ ] Modes configured
- [ ] Voice control tested
- [ ] Audio/video integration verified
- [ ] Security integration verified
- [ ] Pool/spa control tested

### Integration Testing

- [ ] All lighting scenes work
- [ ] Climate control responds correctly
- [ ] Voice commands execute properly
- [ ] Security arming/disarming works
- [ ] CCTV integration verified
- [ ] Access control tested
- [ ] Pool automation working
- [ ] Energy monitoring accurate
- [ ] Modes switch correctly
- [ ] Automation triggers work

### Training

- [ ] Owner trained on basic operation
- [ ] Owner trained on mobile app
- [ ] Owner trained on voice commands
- [ ] Owner trained on scene creation
- [ ] Owner trained on mode management
- [ ] Owner trained on troubleshooting
- [ ] User guide provided

### Documentation

- [ ] As-built drawings updated
- [ ] Device schedule created
- [ ] Network diagram documented
- [ ] Scene list documented
- [ ] Mode behaviors documented
- [ ] Automation list documented
- [ ] Handover pack completed
- [ ] Emergency procedures documented

---

## Ongoing Operations

### Routine Tasks

| Task | Frequency | Responsible |
|------|-----------|-------------|
| Review energy usage | Monthly | Owner |
| Check system health | Daily | Auto-alert |
| Test security system | Monthly | Owner |
| Test pool automation | Weekly | Owner |
| System backup verify | Weekly | Auto |
| Update scenes/modes | As needed | Owner |
| Review automation | Quarterly | Owner |

### Maintenance

```yaml
maintenance:
  # PHM alerts
  phm_alerts:
    enabled: true
    devices:
      - "pool_pump"
      - "heat_pump"
      - "pool_heater"
    notifications: ["owner"]
    
  # Software updates
  updates:
    policy: "security_only"          # No feature updates forced
    schedule: "quarterly"
    require_approval: true
    
  # Backup
  backup:
    schedule: "daily"
    retention_days: 30
    location: "local + optional cloud"
```

### Support

```yaml
support:
  # Self-service
  self_service:
    - "Mobile app"
    - "Web admin"
    - "Voice commands"
    - "User guide"
    
  # Remote support
  remote_support:
    enabled: true
    method: "WireGuard VPN"
    access: "on-demand, owner-initiated"
    
  # On-site support
  on_site:
    available: true
    response_time: "Next business day"
    emergency: "4 hours"
```

---

## Handover

### Handover Pack

See [Handover Pack Template](handover-pack-template.md) for complete checklist.

Key items:

- [ ] System documentation
- [ ] Device schedule
- [ ] Network diagram
- [ ] Scene and mode lists
- [ ] Automation documentation
- [ ] User guides
- [ ] Emergency procedures
- [ ] Support contact information
- [ ] Backup procedures
- [ ] "Doomsday pack" (if requested)

### Owner Training Checklist

- [ ] Basic operation (scenes, modes)
- [ ] Mobile app usage
- [ ] Voice commands
- [ ] Web admin access
- [ ] Troubleshooting basics
- [ ] Emergency procedures
- [ ] Support contact

---

## Related Documents

- [System Overview](../architecture/system-overview.md) — Technical architecture
- [Lighting Domain](../domains/lighting.md) — Residential lighting control
- [Climate Domain](../domains/climate.md) — Residential HVAC control
- [Audio Domain](../domains/audio.md) — Multi-room audio
- [Video Domain](../domains/video.md) — Video distribution
- [Pool Domain](../domains/pool.md) — Pool and spa control
- [Security Domain](../domains/security.md) — Alarm integration
- [Access Control](../integration/access-control.md) — Residential access control
- [CCTV Integration](../integration/cctv.md) — Residential CCTV
- [Voice Pipeline](../intelligence/voice.md) — Voice control
- [Handover Pack Template](handover-pack-template.md) — Customer handover documentation
