---
title: Data Model - Entities
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/glossary.md
  - architecture/system-overview.md
---

# Data Model: Entities

This document defines the core data entities used throughout Gray Logic. These definitions inform database schema, API contracts, and configuration files.

---

## Entity Relationship Overview

```
Site
 └── Area (1:N)
      └── Room (1:N)
           ├── Device (1:N)
           ├── AudioZone (0:1)
           └── ClimateZone (0:1)

Device
 ├── belongs to Room (or Area for area-level devices)
 ├── has Capabilities (1:N)
 ├── has State (1:1)
 ├── has HealthData (0:1, for PHM-monitored devices)
 └── has Associations (0:N, for monitoring/control relationships)

DeviceAssociation
 ├── source_device (sensor, relay, smart plug)
 ├── target_device or target_group (equipment being monitored/controlled)
 └── type: monitors | controls | monitors_and_controls

Scene
 ├── scoped to Room, Area, or Site
 └── contains Actions (1:N)

Schedule
 └── triggers Scene or Actions

Mode
 └── affects Automation behaviour site-wide

User
 ├── has Role
 └── has Presence devices
```

---

## Core Entities

### Site

A single Gray Logic installation.

```yaml
Site:
  id: uuid                          # Unique identifier
  name: string                      # Human-readable name
  slug: string                      # URL-safe identifier (e.g., "oak-street")
  
  location:
    address: string                 # Physical address (optional)
    latitude: float                 # GPS latitude
    longitude: float                # GPS longitude
    timezone: string                # IANA timezone (e.g., "Europe/London")
    elevation_m: float              # Elevation in meters (for weather/solar)
  
  modes:
    available: [string]             # List of available modes
    current: string                 # Currently active mode
    
  settings:
    units:
      temperature: "celsius" | "fahrenheit"
      distance: "metric" | "imperial"
    locale: string                  # Language/locale (e.g., "en-GB")
    
  created_at: timestamp
  updated_at: timestamp
```

**Example:**
```yaml
id: "a7b3c9d2-1234-5678-9abc-def012345678"
name: "Oak Street Residence"
slug: "oak-street"
location:
  address: "123 Oak Street, London"
  latitude: 51.5074
  longitude: -0.1278
  timezone: "Europe/London"
  elevation_m: 35
modes:
  available: ["home", "away", "night", "holiday", "entertaining"]
  current: "home"
settings:
  units:
    temperature: "celsius"
    distance: "metric"
  locale: "en-GB"
```

---

### Area

A logical grouping within a site (floor, building, outdoor space).

```yaml
Area:
  id: uuid
  site_id: uuid                     # Reference to parent Site
  name: string                      # Human-readable name
  slug: string                      # URL-safe identifier
  type: enum                        # Area type (see below)
  sort_order: integer               # Display order
  
  created_at: timestamp
  updated_at: timestamp
```

**Area Types:**
- `floor` — A floor within a building (Ground Floor, First Floor)
- `building` — A separate building (Main House, Pool House)
- `wing` — A wing or section of a larger building (North Wing, East Block)
- `zone` — A logical zone spanning multiple rooms (Reception Zone, Executive Suite)
- `outdoor` — External areas (Garden, Driveway, Car Park)
- `utility` — Service areas (Plant Room, Garage, Loading Bay)

**Hierarchy Guidance:**

The Area entity is flexible to support both residential and commercial deployments:

| Deployment | Site | Area | Room |
|------------|------|------|------|
| **Residential** | Property | Floor/Building | Living Room, Bedroom |
| **Office** | Building | Floor or Wing | Open Plan, Meeting Room |
| **Multi-tenant** | Building | Tenant/Floor | Individual spaces |
| **Estate** | Estate | Individual property | Rooms within |

**Residential Example:**
```yaml
id: "b8c4d3e5-2345-6789-abcd-ef0123456789"
site_id: "a7b3c9d2-1234-5678-9abc-def012345678"
name: "Ground Floor"
slug: "ground-floor"
type: "floor"
sort_order: 1
```

**Commercial Example:**
```yaml
id: "b8c4d3e5-2345-6789-abcd-ef0123456790"
site_id: "a7b3c9d2-1234-5678-9abc-def012345679"
name: "North Wing"
slug: "north-wing"
type: "wing"
sort_order: 1
```

---

### Room

A physical space within an area.

```yaml
Room:
  id: uuid
  area_id: uuid                     # Reference to parent Area
  name: string                      # Human-readable name
  slug: string                      # URL-safe identifier
  type: enum                        # Room type (see below)
  sort_order: integer               # Display order
  
  # Optional zone associations
  climate_zone_id: uuid | null      # Climate zone this room belongs to
  audio_zone_id: uuid | null        # Audio zone this room belongs to
  
  # Room-specific settings
  settings:
    default_scene: uuid | null      # Scene to use when room becomes occupied
    vacancy_timeout_min: integer    # Minutes before room considered vacant
    
  created_at: timestamp
  updated_at: timestamp
```

**Room Types:**

*Residential:*
- `living` — Living room, lounge, sitting room
- `bedroom` — Bedroom, guest room
- `bathroom` — Bathroom, en-suite, WC
- `kitchen` — Kitchen, kitchenette
- `dining` — Dining room
- `office` — Home office, study
- `utility` — Utility room, laundry, boot room
- `hallway` — Hallway, corridor, landing
- `stairs` — Stairwell
- `garage` — Garage
- `plant` — Plant room, mechanical room
- `cinema` — Cinema room, media room
- `gym` — Gym, exercise room
- `pool` — Pool room, pool hall
- `wine_cellar` — Wine cellar, cold storage
- `spa` — Spa, sauna, steam room

*Commercial/Office:*
- `open_plan` — Open plan office area
- `meeting_room` — Bookable meeting room
- `boardroom` — Large meeting/board room with AV
- `reception` — Reception, entrance lobby
- `break_room` — Kitchen, break area, canteen
- `hot_desk` — Hot desking zone
- `server_room` — IT/comms/server room
- `storage` — Storage room, archive
- `loading_bay` — Goods in, delivery area
- `washroom` — Commercial WC, washroom
- `corridor` — Corridor (for lighting/HVAC control)
- `workshop` — Workshop, maintenance area
- `changing_room` — Changing room, locker room

*Universal:*
- `other` — Other room types

**Example:**
```yaml
id: "c9d5e4f6-3456-7890-bcde-f01234567890"
area_id: "b8c4d3e5-2345-6789-abcd-ef0123456789"
name: "Living Room"
slug: "living-room"
type: "living"
sort_order: 1
climate_zone_id: "d0e6f5a7-4567-8901-cdef-012345678901"
audio_zone_id: "e1f7a6b8-5678-9012-def0-123456789012"
settings:
  default_scene: null
  vacancy_timeout_min: 15
```

---

### Device

Any controllable or monitorable entity.

```yaml
Device:
  id: uuid
  room_id: uuid | null              # Room containing device (null for area-level)
  area_id: uuid | null              # Area if not room-specific
  
  name: string                      # Human-readable name
  slug: string                      # URL-safe identifier
  
  # Device classification
  type: enum                        # Device type (see below)
  domain: enum                      # Domain (lighting, climate, etc.)
  
  # Protocol information
  protocol: enum                    # Communication protocol
  address: object                   # Protocol-specific address
  gateway_id: uuid | null           # Gateway device (if applicable)
  
  # Capabilities and configuration
  capabilities: [string]            # List of capability identifiers
  config: object                    # Device-specific configuration
  
  # Current state
  state: object                     # Current device state
  state_updated_at: timestamp       # When state was last updated
  
  # Health monitoring
  health:
    status: enum                    # online | offline | degraded | unknown
    last_seen: timestamp            # Last communication
    phm_enabled: boolean            # PHM monitoring active
    phm_baseline: object | null     # Learned baseline data
    
  # Metadata
  manufacturer: string | null
  model: string | null
  firmware_version: string | null
  
  created_at: timestamp
  updated_at: timestamp
```

**Device Types:**

*Lighting:*
- `light_switch` — On/off only
- `light_dimmer` — Dimmable light
- `light_ct` — Colour temperature adjustable
- `light_rgb` — Full colour
- `light_rgbw` — RGB plus white

*Climate:*
- `thermostat` — Temperature control
- `temperature_sensor` — Temperature measurement only
- `humidity_sensor` — Humidity measurement
- `air_quality_sensor` — CO2, VOC, etc.
- `hvac_unit` — HVAC equipment
- `valve_actuator` — Heating/cooling valve

*Blinds:*
- `blind_switch` — Open/close only
- `blind_position` — Position control
- `blind_tilt` — Position plus tilt angle

*Sensors:*
- `motion_sensor` — PIR or mmWave motion
- `presence_sensor` — Occupancy detection
- `door_sensor` — Door open/close
- `window_sensor` — Window open/close
- `leak_sensor` — Water leak detection
- `smoke_sensor` — Smoke detection
- `co_sensor` — Carbon monoxide

*Switches/Controls:*
- `switch` — Generic switch
- `keypad` — Multi-button keypad
- `remote` — Remote control

*Audio:*
- `audio_zone` — Audio zone output
- `audio_source` — Audio input source

*Video:*
- `video_output` — Display/TV
- `video_source` — Video input source
- `video_matrix` — Video matrix switcher

*Security:*
- `alarm_panel` — Alarm system panel
- `camera` — IP camera
- `nvr` — Network video recorder
- `door_lock` — Electric lock
- `door_station` — Video intercom

*Energy:*
- `energy_meter` — Power measurement
- `ct_clamp` — Current transformer

*Plant Equipment:*
- `pump` — Pump (circulation, booster, dosing, pool)
- `boiler` — Boiler, furnace, water heater
- `heat_pump` — Heat pump (air source, ground source)
- `chiller` — Chiller unit
- `cooling_tower` — Cooling tower, dry cooler
- `ahu` — Air handling unit
- `fcu` — Fan coil unit
- `vav_box` — Variable air volume box
- `vfd` — Variable frequency drive
- `fan` — Standalone fan, extract fan
- `compressor` — Compressor unit
- `humidifier` — Humidifier unit
- `dehumidifier` — Dehumidifier unit
- `water_heater` — DHW cylinder, calorifier
- `water_softener` — Water softener, treatment
- `generator` — Backup generator
- `ups` — Uninterruptible power supply

*Plant Sensors & Actuators:*
- `tank_level` — Tank/vessel level sensor
- `flow_meter` — Flow rate sensor
- `pressure_sensor` — Pressure transducer
- `differential_pressure` — Filter DP, duct pressure
- `valve_2way` — 2-way control valve
- `valve_3way` — 3-way mixing/diverting valve
- `damper` — Air damper actuator
- `vibration_sensor` — Vibration monitoring
- `bearing_temp` — Bearing temperature sensor

*Emergency & Safety (Monitoring Only):*
- `emergency_light` — Monitored emergency luminaire (DALI Part 202)
- `exit_sign` — Emergency exit sign
- `fire_input` — Fire alarm system input signal
- `gas_detector` — Gas detection sensor

*Access Control:*
- `card_reader` — Access card/fob reader
- `door_controller` — Access control door controller
- `turnstile` — Turnstile, barrier, speed gate
- `intercom` — Audio/video intercom station

*I/O and Control:*
- `relay_module` — Multi-channel relay output module
- `relay_channel` — Individual relay output channel
- `digital_input` — Digital input (contact closure)
- `digital_output` — Digital output (relay, solid state)
- `analog_input` — Analog input (0-10V, 4-20mA)
- `analog_output` — Analog output (0-10V)
- `io_module` — Combined I/O module

*Monitoring (External):*
- `power_monitor` — Inline power monitor / smart plug
- `ct_clamp` — Current transformer clamp
- `energy_submeter` — Sub-metering device
- `external_temp_sensor` — Standalone temperature sensor
- `vibration_monitor` — External vibration sensor

**Domains:**
- `lighting` — Lights, scenes, emergency lighting monitoring
- `climate` — HVAC, heating, cooling, ventilation
- `blinds` — Blinds, shades, curtains, awnings
- `audio` — Multi-room audio, speakers
- `video` — Displays, video distribution, matrices
- `security` — Alarm panels, sensors, monitoring
- `access` — Door access control, intercoms
- `energy` — Metering, solar, battery, EV charging
- `plant` — Mechanical plant, pumps, VFDs, AHUs
- `irrigation` — Garden irrigation, water management
- `safety` — Emergency lighting, fire inputs (monitoring only)
- `sensor` — Generic sensors, environmental monitoring

**Protocols:**
- `knx` — KNX TP or IP
- `dali` — DALI via gateway
- `modbus_rtu` — Modbus RTU (serial)
- `modbus_tcp` — Modbus TCP (Ethernet)
- `bacnet_ip` — BACnet/IP (commercial HVAC)
- `bacnet_mstp` — BACnet MS/TP (commercial HVAC)
- `mqtt` — Native MQTT device
- `http` — HTTP/REST API
- `sip` — SIP for intercoms
- `rtsp` — RTSP for cameras
- `onvif` — ONVIF for IP cameras
- `ocpp` — OCPP for EV chargers
- `rs232` — Serial RS232
- `rs485` — Serial RS485

**Address Examples:**
```yaml
# KNX device
protocol: "knx"
address:
  group_address: "1/2/3"
  feedback_address: "1/2/4"

# DALI device
protocol: "dali"
address:
  gateway: "dali-gw-01"
  short_address: 15
  group: 3

# Modbus device
protocol: "modbus_tcp"
address:
  host: "192.168.1.100"
  port: 502
  unit_id: 1
  registers:
    status: { address: 100, type: "holding" }
    control: { address: 200, type: "holding" }
```

**Capabilities:**

*Basic Control:*
- `on_off` — Can be turned on/off
- `dim` — Supports dimming (0-100%)
- `color_temp` — Colour temperature control
- `color_rgb` — RGB colour control
- `position` — Position control (blinds, 0-100%)
- `tilt` — Tilt angle control (blinds, 0-100%)

*Climate & Environment:*
- `temperature_read` — Reads temperature
- `temperature_set` — Sets temperature setpoint
- `humidity_read` — Reads humidity
- `pressure_read` — Reads pressure
- `flow_read` — Reads flow rate
- `co2_read` — Reads CO2 level
- `voc_read` — Reads VOC level
- `air_quality_read` — Reads air quality index

*Presence & Occupancy:*
- `motion_detect` — Detects motion
- `presence_detect` — Detects presence/occupancy
- `people_count` — Counts number of people
- `contact_state` — Reports open/close state

*Audio/Video:*
- `volume` — Volume control
- `mute` — Mute control
- `source_select` — Input source selection

*Security & Access:*
- `lock_unlock` — Lock/unlock control
- `arm_disarm` — Arm/disarm control
- `card_access` — Card/credential access events

*Plant & Equipment:*
- `speed_control` — Speed setpoint (0-100% or Hz)
- `speed_read` — Speed feedback
- `run_stop` — Run/stop command
- `run_hours` — Runtime hour counter
- `fault_status` — Fault/alarm indicator
- `enable_disable` — Enable/disable equipment
- `setpoint` — Generic setpoint control
- `valve_position` — Valve position (0-100%)
- `damper_position` — Damper position (0-100%)
- `filter_status` — Filter condition/DP status

*Energy Monitoring:*
- `power_read` — Reads power (W/kW)
- `energy_read` — Reads energy (Wh/kWh)
- `current_read` — Reads current (A)
- `voltage_read` — Reads voltage (V)
- `power_factor_read` — Reads power factor
- `frequency_read` — Reads frequency (Hz)

*Condition Monitoring (PHM):*
- `vibration_read` — Reads vibration level
- `bearing_temp_read` — Reads bearing temperature
- `oil_pressure_read` — Reads oil pressure
- `oil_temp_read` — Reads oil temperature

*Emergency Lighting (DALI Part 202):*
- `emergency_mode` — Emergency mode status
- `emergency_test` — Trigger emergency test
- `battery_status` — Battery health status
- `lamp_status` — Lamp/LED health status

*Booking & Scheduling:*
- `booking_status` — Calendar/booking integration
- `override_request` — Out-of-hours override request

---

### Scene

A predefined collection of device states.

```yaml
Scene:
  id: uuid
  site_id: uuid                     # Always scoped to site
  
  name: string                      # Human-readable name
  slug: string                      # URL-safe identifier
  icon: string | null               # Icon identifier
  color: string | null              # UI colour (hex)
  
  # Scope
  scope:
    type: "site" | "area" | "room"
    id: uuid                        # ID of site/area/room
    
  # Trigger configuration
  triggers:
    manual: boolean                 # Can be manually activated
    voice_phrase: string | null     # Voice activation phrase
    schedule_ids: [uuid]            # Linked schedules
    
  # Actions to perform
  actions: [Action]                 # Ordered list of actions
  
  # Conditions for execution
  conditions: [Condition] | null    # Optional conditions
  
  # Metadata
  category: string | null           # Grouping category
  sort_order: integer
  
  created_at: timestamp
  updated_at: timestamp
```

**Action Definition:**
```yaml
Action:
  # Target (one of)
  device_id: uuid | null            # Specific device
  device_group: string | null       # Device group/tag
  room_id: uuid | null              # All devices in room matching domain
  domain: string | null             # Filter by domain (with room_id)
  
  # Command
  command: string                   # Command to execute
  parameters: object                # Command parameters
  
  # Timing
  delay_ms: integer                 # Delay before execution (default: 0)
  fade_ms: integer                  # Transition duration (default: 0)
  
  # Execution control
  parallel: boolean                 # Execute in parallel with previous (default: false)
```

**Example Scene:**
```yaml
id: "f2a8b7c9-6789-0123-ef01-234567890abc"
name: "Cinema Mode"
slug: "cinema-mode"
icon: "movie"
color: "#6B21A8"
scope:
  type: "room"
  id: "c9d5e4f6-3456-7890-bcde-f01234567890"
triggers:
  manual: true
  voice_phrase: "cinema mode"
  schedule_ids: []
actions:
  - device_group: "living_room_lights"
    command: "dim"
    parameters: { level: 0 }
    delay_ms: 0
    fade_ms: 3000
    parallel: false
    
  - device_id: "blind-living-01"
    command: "position"
    parameters: { position: 0 }
    delay_ms: 0
    fade_ms: 0
    parallel: true
    
  - device_id: "av-receiver-01"
    command: "source"
    parameters: { input: "hdmi2" }
    delay_ms: 3000
    fade_ms: 0
    parallel: false
    
  - device_id: "living-room-speaker"
    command: "volume"
    parameters: { level: 40 }
    delay_ms: 3500
    fade_ms: 1000
    parallel: true
conditions:
  - type: "mode"
    operator: "in"
    value: ["home", "entertaining"]
category: "entertainment"
sort_order: 1
```

---

### Schedule

Time-based automation trigger.

```yaml
Schedule:
  id: uuid
  site_id: uuid
  
  name: string
  enabled: boolean
  
  # Trigger timing
  trigger:
    type: "time" | "sunrise" | "sunset" | "cron"
    value: string                   # Time or offset or cron expression
    days: [string] | null           # Days of week (null = all)
    
  # What to execute
  execute:
    type: "scene" | "actions"
    scene_id: uuid | null           # If type=scene
    actions: [Action] | null        # If type=actions
    
  # Conditions
  conditions: [Condition] | null
  
  # Validity period
  valid_from: date | null
  valid_until: date | null
  
  created_at: timestamp
  updated_at: timestamp
```

**Trigger Examples:**
```yaml
# Fixed time
trigger:
  type: "time"
  value: "07:30"
  days: ["mon", "tue", "wed", "thu", "fri"]

# Relative to sunrise
trigger:
  type: "sunrise"
  value: "-30m"                     # 30 minutes before sunrise
  days: null                        # Every day

# Cron expression
trigger:
  type: "cron"
  value: "0 */2 * * *"              # Every 2 hours
```

---

### Mode

System-wide operational state.

```yaml
Mode:
  id: string                        # Mode identifier (home, away, etc.)
  name: string                      # Display name
  icon: string
  color: string
  
  # Behaviour modifications
  behaviours:
    climate:
      setpoint_offset: float        # Temperature offset from normal
      eco_mode: boolean             # Enable eco/setback mode
    lighting:
      auto_off_enabled: boolean     # Auto-off on vacancy
      auto_off_delay_min: integer   # Delay before auto-off
      max_brightness: integer       # Maximum brightness percentage
    security:
      arm_state: string | null      # Desired arm state (arm_away, arm_stay, disarm)
    audio:
      max_volume: integer | null    # Maximum volume percentage
    notifications:
      priority_only: boolean        # Only critical notifications
      
  # Activation rules
  can_activate:
    roles: [string]                 # Roles that can activate
    require_pin: boolean            # Require PIN to activate
    remote_allowed: boolean         # Can be activated remotely
```

**Standard Modes:**
```yaml
- id: "home"
  name: "Home"
  icon: "home"
  color: "#22C55E"
  behaviours:
    climate:
      setpoint_offset: 0
      eco_mode: false
    lighting:
      auto_off_enabled: true
      auto_off_delay_min: 15
      max_brightness: 100
    security:
      arm_state: "disarm"
    notifications:
      priority_only: false

- id: "away"
  name: "Away"
  icon: "door-open"
  color: "#EAB308"
  behaviours:
    climate:
      setpoint_offset: -3
      eco_mode: true
    lighting:
      auto_off_enabled: true
      auto_off_delay_min: 5
      max_brightness: 50
    security:
      arm_state: "arm_away"
    notifications:
      priority_only: true

- id: "night"
  name: "Night"
  icon: "moon"
  color: "#6366F1"
  behaviours:
    climate:
      setpoint_offset: -2
      eco_mode: false
    lighting:
      auto_off_enabled: true
      auto_off_delay_min: 5
      max_brightness: 30
    security:
      arm_state: "arm_stay"
    audio:
      max_volume: 30
    notifications:
      priority_only: true

- id: "holiday"
  name: "Holiday"
  icon: "plane"
  color: "#F97316"
  behaviours:
    climate:
      setpoint_offset: -5
      eco_mode: true
    lighting:
      presence_simulation: true
    security:
      arm_state: "arm_away"
    notifications:
      priority_only: true
```

---

### User

A person who interacts with the system.

```yaml
User:
  id: uuid
  
  name: string
  email: string | null
  phone: string | null
  
  # Authentication
  auth:
    pin_hash: string | null         # Hashed PIN
    password_hash: string | null    # Hashed password
    api_keys: [ApiKey]              # API keys for integrations
    
  # Authorization
  role: "admin" | "resident" | "guest" | "installer"
  permissions: [string] | null      # Override permissions (null = use role defaults)
  
  # Presence tracking
  presence:
    devices: [PresenceDevice]       # Devices used to track presence
    is_home: boolean                # Currently detected as home
    last_seen: timestamp
    
  # Preferences
  preferences:
    default_temperature: float
    wake_time: time | null
    sleep_time: time | null
    voice_profile_id: uuid | null   # For voice recognition
    
  # Access restrictions
  access:
    rooms: [uuid] | null            # Allowed rooms (null = all)
    valid_from: timestamp | null
    valid_until: timestamp | null
    
  created_at: timestamp
  updated_at: timestamp
```

**PresenceDevice:**
```yaml
PresenceDevice:
  type: "phone_wifi" | "phone_bluetooth" | "beacon" | "vehicle"
  identifier: string                # MAC address, beacon ID, etc.
  name: string                      # "Darren's iPhone"
```

---

### Condition

A prerequisite for automation execution.

```yaml
Condition:
  type: string                      # Condition type
  operator: string                  # Comparison operator
  value: any                        # Value to compare against
  
  # Optional device reference
  device_id: uuid | null
```

**Condition Types:**
```yaml
# Mode condition
- type: "mode"
  operator: "eq" | "neq" | "in"
  value: "home" | ["home", "night"]

# Time condition
- type: "time"
  operator: "between" | "before" | "after"
  value: ["22:00", "06:00"] | "22:00"

# Day of week
- type: "day"
  operator: "in" | "not_in"
  value: ["sat", "sun"]

# Device state
- type: "device_state"
  device_id: "uuid"
  operator: "eq" | "neq" | "gt" | "lt" | "gte" | "lte"
  value: { "level": 50 }

# Presence
- type: "presence"
  operator: "any_home" | "nobody_home" | "user_home"
  value: null | "user-uuid"

# Sun position
- type: "sun"
  operator: "above_horizon" | "below_horizon" | "elevation_gt" | "elevation_lt"
  value: null | 10                  # Degrees above horizon
```

---

### DeviceAssociation

Links two devices together for monitoring, control, or combined purposes. This enables:
- External sensors monitoring equipment (CT clamp → pump)
- Relay modules controlling dumb equipment (relay → pump)
- Smart plugs that do both (power monitor → pump)

```yaml
DeviceAssociation:
  id: uuid
  
  # The device providing the service (sensor, relay, smart plug)
  source_device_id: uuid
  
  # The device being monitored/controlled (pump, heater, circuit)
  target_device_id: uuid             # Single device
  target_group_id: uuid | null       # OR a device group (for circuits)
  
  # Relationship type
  type: enum                         # monitors | controls | monitors_and_controls
  
  # Association details
  config:
    # For monitoring associations
    metrics: [string] | null         # ["power_kw", "energy_kwh", "current_a"]
    
    # For control associations
    control_function: string | null  # "power" | "enable" | "speed"
    
    # Mapping configuration
    source_property: string | null   # Property on source device
    target_property: string | null   # Property to attribute to target
    
    # Scaling (if needed)
    scale: float | null              # Multiply source value
    offset: float | null             # Add to source value
    
  # Metadata
  description: string | null         # "CT clamp monitoring pump power"
  
  enabled: boolean                   # Association active
  created_at: timestamp
  updated_at: timestamp
```

**Association Types:**

| Type | Data Flow | Use Case |
|------|-----------|----------|
| `monitors` | Source → Target | CT clamp readings attributed to pump |
| `controls` | Target → Source | Commands to pump sent to relay |
| `monitors_and_controls` | Both | Smart plug monitors AND controls pump |

**Examples:**

```yaml
# CT clamp monitoring a pump's power consumption
- id: "assoc-ct-pump-1"
  source_device_id: "ct-clamp-plantroom-1"
  target_device_id: "pump-chw-1"
  type: "monitors"
  config:
    metrics: ["power_kw", "energy_kwh", "current_a"]
    source_property: "power"
    target_property: "power_kw"
  description: "CT clamp on CHW pump 1 supply"

# Relay controlling a pump's power
- id: "assoc-relay-pump-1"
  source_device_id: "relay-module-1-ch3"
  target_device_id: "pump-chw-1"
  type: "controls"
  config:
    control_function: "power"
  description: "Relay controlling CHW pump 1"

# Smart plug monitoring AND controlling a dumb heater
- id: "assoc-plug-heater"
  source_device_id: "smart-plug-garage"
  target_device_id: "heater-garage"
  type: "monitors_and_controls"
  config:
    metrics: ["power_kw", "energy_kwh"]
    control_function: "power"
  description: "Smart plug for garage heater"

# CT clamp monitoring a lighting circuit (group)
- id: "assoc-ct-kitchen-lights"
  source_device_id: "ct-clamp-db-ch5"
  target_group_id: "group-kitchen-lights"
  type: "monitors"
  config:
    metrics: ["power_kw", "energy_kwh"]
  description: "Kitchen lighting circuit power"
```

**How Core Resolves Associations:**

1. **Monitoring (data attribution):**
   - When source device reports a reading (e.g., CT clamp reports 5.2kW)
   - Core looks up associations where `source_device_id` matches
   - Core attributes the reading to the target device's metrics
   - PHM and energy tracking see the target device's power consumption

2. **Control (command routing):**
   - When command targets a device (e.g., "turn on pump-chw-1")
   - Core looks up associations where `target_device_id` matches and type includes `controls`
   - Core routes the command to the source device (relay)
   - Source device executes the command (relay closes)

3. **Combined:**
   - Both behaviours active for `monitors_and_controls` type

---

## Supporting Entities

### AudioZone

```yaml
AudioZone:
  id: uuid
  site_id: uuid
  name: string
  
  # Physical mapping
  matrix_zone: integer              # Zone number on audio matrix
  rooms: [uuid]                     # Rooms this zone covers
  
  # Current state
  state:
    power: boolean
    source: integer
    volume: integer                 # 0-100
    mute: boolean
```

### ClimateZone

```yaml
ClimateZone:
  id: uuid
  site_id: uuid
  name: string
  
  # Physical mapping
  rooms: [uuid]                     # Rooms in this zone
  thermostat_id: uuid | null        # Primary thermostat
  sensors: [uuid]                   # Temperature sensors
  actuators: [uuid]                 # Valves, etc.
  
  # Setpoints
  setpoints:
    heating: float
    cooling: float
  
  # Current state
  state:
    current_temp: float
    target_temp: float
    mode: "off" | "heating" | "cooling" | "auto"
    humidity: float | null
```

---

## Schema Files

Formal JSON Schema definitions are maintained in [`docs/data-model/schemas/`](schemas/README.md):

- `common.schema.json` — Shared enums and embedded types
- `site.schema.json` — Site entity
- `area.schema.json` — Area entity
- `room.schema.json` — Room entity
- `device.schema.json` — Device entity (all types, protocols, capabilities)
- `scene.schema.json` — Scene entity
- `schedule.schema.json` — Schedule entity
- `mode.schema.json` — Mode entity
- `condition.schema.json` — Condition entity
- `user.schema.json` — User entity
- `device-association.schema.json` — DeviceAssociation entity
- `audio-zone.schema.json` — AudioZone entity
- `climate-zone.schema.json` — ClimateZone entity

These schemas support runtime validation, documentation, and code generation. See the [schemas README](schemas/README.md) for usage examples.

---

## Related Documents

- [Glossary](../overview/glossary.md) — Term definitions
- [System Overview](../architecture/system-overview.md) — How entities are used
- [Core Internals](../architecture/core-internals.md) — State management and internal architecture
- [Automation Specification](../automation/automation.md) — How scenes, schedules, and modes work
- [API Specification](../interfaces/api.md) — API for entity operations
- [Audio Domain](../domains/audio.md) — Multi-room audio and AudioZone configuration
- [CCTV Integration](../integration/cctv.md) — Camera and NVR configuration
- [Access Control Integration](../integration/access-control.md) — Door, gate, intercom configuration
