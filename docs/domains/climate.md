---
title: Climate Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - data-model/entities.md
  - protocols/knx.md
  - protocols/modbus.md
---

# Climate Domain Specification

This document specifies all climate control features in Gray Logic, including heating, cooling, ventilation, humidity, and air quality management.

---

## Overview

Climate control is critical for comfort, energy efficiency, and building health. Gray Logic provides comprehensive HVAC management while ensuring **safe fallback operation**.

### Design Principles

1. **Fail safe** — System defaults to safe temperatures if automation fails
2. **Frost protection** — Cannot be disabled by software
3. **Energy efficiency** — Weather-predictive, occupancy-aware
4. **Zone control** — Independent temperature per zone
5. **Equipment protection** — Respect equipment constraints

### Device Types

| Type | Capabilities | Typical Hardware |
|------|-------------|------------------|
| `temperature_sensor` | Temperature reading | KNX sensor, wireless probe |
| `humidity_sensor` | Humidity reading | KNX/Modbus sensor |
| `thermostat` | Temp reading, setpoint, mode | Wall thermostat |
| `valve_actuator` | Position control | Radiator valve, UFH actuator |
| `hvac_unit` | Full HVAC control | Heat pump, AC unit, boiler |
| `air_quality_sensor` | CO2, VOC, PM2.5 | Air quality monitor |
| `ventilation_unit` | MVHR, extract fans | HRV unit, extractor |

---

## Feature Matrix

### Core Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Temperature Setpoint** | Must-have | Set target temperature per zone |
| **Mode Selection** | Must-have | Heat/Cool/Auto/Off |
| **Multi-Zone Control** | Must-have | Independent room temperatures |
| **Scheduling** | Must-have | Time-based temperature programs |
| **Status Feedback** | Must-have | Current temp, valve position, running |
| **Frost Protection** | Must-have | Minimum temperature guarantee |
| **Manual Override** | Must-have | Temporary adjustment with timeout |

### Intelligent Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Occupancy-Based** | Should-have | Setback when unoccupied |
| **Geofencing** | Should-have | Pre-heat/cool on approach |
| **Weather-Predictive** | Should-have | Use forecast to optimize |
| **Solar Gain Awareness** | Should-have | Coordinate with blinds |
| **Adaptive Start** | Should-have | Learn heat-up time |
| **Holiday Mode** | Must-have | Extended setback |
| **Boost Mode** | Should-have | Temporary high output |

### Advanced Features

| Feature | Priority | Description |
|---------|----------|-------------|
| **Heat Pump Optimization** | Should-have | Maximize COP |
| **Demand Response** | Nice-to-have | Grid signal integration |
| **Load Shifting** | Nice-to-have | Time-of-use tariff optimization |
| **Floor Heating Control** | Must-have | UFH manifold control |
| **MVHR Integration** | Should-have | Ventilation heat recovery |
| **CO2-Based Ventilation** | Should-have | Demand-controlled ventilation |
| **Humidity Control** | Should-have | Humidifier/dehumidifier |

---

## State Model

### Climate Zone State

```yaml
ClimateZoneState:
  # Temperature
  current_temp: float               # Current temperature (°C)
  target_temp: float                # Setpoint (°C)
  
  # Mode
  mode: enum                        # off | heat | cool | auto | fan_only | dry
  preset: enum                      # none | home | away | boost | sleep | comfort | eco
  
  # Status
  hvac_action: enum                 # off | heating | cooling | idle | drying | fan
  
  # Humidity (if available)
  current_humidity: float           # Current RH (%)
  target_humidity: float            # Target RH (%) - optional
  
  # Valve/actuator positions
  valve_position: float             # 0-100%
  
  # Override status
  override_active: boolean
  override_expires: timestamp
```

### HVAC Equipment State

```yaml
HVACState:
  # Operating state
  running: boolean
  mode: enum                        # off | heat | cool | auto
  fan_speed: enum                   # auto | low | medium | high
  
  # Temperatures
  supply_temp: float
  return_temp: float
  outdoor_temp: float
  
  # Performance (heat pump)
  cop: float                        # Coefficient of performance
  power_consumption: float          # Watts
  
  # Status
  defrost_active: boolean
  error_code: integer | null
  filter_status: enum               # ok | warning | replace
  
  # Runtime
  compressor_hours: integer
  heating_hours: integer
  cooling_hours: integer
```

### Ventilation State

```yaml
VentilationState:
  # Operating mode
  mode: enum                        # off | low | medium | high | auto | boost
  
  # Speed (0-100% or discrete levels)
  supply_speed: integer
  extract_speed: integer
  
  # Temperatures
  supply_temp: float
  extract_temp: float
  outdoor_temp: float
  exhaust_temp: float
  
  # Heat recovery
  heat_recovery_efficiency: float   # Percent
  bypass_active: boolean            # Summer bypass
  
  # Air quality
  co2_level: integer                # ppm
  humidity: float                   # RH %
  
  # Filter
  filter_days_remaining: integer
```

---

## Zone Configuration

### Zone Definition

```yaml
climate_zones:
  - id: "zone-living"
    name: "Living Area"
    type: "heating_cooling"
    
    # Rooms in this zone
    rooms:
      - "room-living"
      - "room-dining"
      - "room-kitchen"
      
    # Sensors (average or primary)
    temperature_sensors:
      - sensor_id: "sensor-living-temp"
        weight: 1.0
      - sensor_id: "sensor-kitchen-temp"
        weight: 0.5
    sensor_mode: "weighted_average"  # average | primary | weighted_average
    primary_sensor: "sensor-living-temp"
    
    # Actuators
    actuators:
      - device_id: "valve-ufh-living"
        type: "valve"
      - device_id: "valve-ufh-dining"
        type: "valve"
      - device_id: "valve-ufh-kitchen"
        type: "valve"
        
    # Control parameters
    control:
      type: "pid"                   # pid | on_off | proportional
      kp: 1.0
      ki: 0.1
      kd: 0.05
      deadband: 0.5                 # °C
      min_on_time: 300              # seconds
      min_off_time: 180             # seconds
      
    # Setpoints
    setpoints:
      default_heat: 21.0
      default_cool: 24.0
      frost_protect: 8.0
      max_heat: 25.0
      min_cool: 18.0
      
    # Schedule reference
    schedule_id: "schedule-living-climate"
```

### Underfloor Heating Configuration

```yaml
floor_heating:
  zone_id: "zone-living"
  
  # Manifold
  manifold:
    pump_device_id: "pump-ufh-gf"
    mixing_valve_id: "valve-ufh-mixing"
    
  # Loops
  loops:
    - id: "ufh-living-1"
      actuator_id: "valve-ufh-living"
      area_m2: 25
      
    - id: "ufh-dining-1"
      actuator_id: "valve-ufh-dining"
      area_m2: 18
      
  # Water temperature limits
  limits:
    max_flow_temp: 45              # °C - floor protection
    min_flow_temp: 25              # °C
    
  # Pump control
  pump:
    run_on_time: 300               # seconds after valves close
    min_valves_open: 1             # Keep pump off if < N valves
    
  # Floor type factors
  floor_response_time: 60          # minutes (slow response)
```

### Radiator Zone Configuration

```yaml
radiator_zone:
  zone_id: "zone-bedroom"
  
  radiators:
    - device_id: "trv-bedroom-1"
      type: "trv"                  # Thermostatic radiator valve
      max_flow_temp: 70
      
    - device_id: "trv-bedroom-2"
      type: "trv"
      max_flow_temp: 70
      
  # Faster response than UFH
  response_time: 20                # minutes
```

---

## Commands

### Zone Commands

| Command | Parameters | Description |
|---------|------------|-------------|
| `set_temperature` | `temperature`, `mode?` | Set target temperature |
| `set_mode` | `mode` | Change operating mode |
| `set_preset` | `preset` | Apply preset (home/away/boost) |
| `boost` | `duration_minutes`, `temperature?` | Temporary boost |
| `override` | `temperature`, `duration_minutes` | Temporary override |
| `cancel_override` | - | Return to schedule |

### Command Examples

**Set temperature:**
```yaml
target:
  type: "climate_zone"
  zone_id: "zone-living"
command: "set_temperature"
parameters:
  temperature: 22.0
  mode: "heat"
```

**Boost heating:**
```yaml
target:
  type: "climate_zone"
  zone_id: "zone-bathroom"
command: "boost"
parameters:
  duration_minutes: 30
  temperature: 24.0
```

**Apply preset:**
```yaml
target:
  type: "climate_zone"
  zone_id: "zone-living"
command: "set_preset"
parameters:
  preset: "away"
```

---

## Scheduling

### Schedule Definition

```yaml
climate_schedules:
  - id: "schedule-living-climate"
    name: "Living Area Schedule"
    zone_id: "zone-living"
    
    weekday:
      - time: "06:00"
        heat_setpoint: 20.0
        cool_setpoint: 25.0
        
      - time: "08:30"
        heat_setpoint: 18.0        # Away during work hours
        cool_setpoint: 27.0
        
      - time: "17:00"
        heat_setpoint: 21.0        # Return from work
        cool_setpoint: 24.0
        
      - time: "22:00"
        heat_setpoint: 18.0        # Night setback
        cool_setpoint: 26.0
        
    weekend:
      - time: "08:00"
        heat_setpoint: 21.0
        cool_setpoint: 24.0
        
      - time: "23:00"
        heat_setpoint: 18.0
        cool_setpoint: 26.0
        
    # Override for holidays
    holiday:
      heat_setpoint: 15.0
      cool_setpoint: 30.0
```

### Adaptive Start

Learn building thermal characteristics to achieve target temperature on time:

```yaml
adaptive_start:
  zone_id: "zone-living"
  enabled: true
  
  # Learning parameters
  learning:
    min_samples: 7                 # Days of data before confident
    outdoor_temp_bins: 5           # Temperature ranges to learn
    
  # Calculated values (by system)
  learned_parameters:
    heat_rate_per_degree:          # Minutes per °C rise
      outdoor_below_0: 45
      outdoor_0_to_10: 35
      outdoor_10_to_20: 25
      outdoor_above_20: 20
    cool_rate_per_degree: 15       # Minutes per °C drop
    
  # Limits
  max_preheat_hours: 3
  max_precool_hours: 2
```

---

## Weather-Predictive Control

### Integration with Weather Data

```yaml
weather_predictive:
  enabled: true
  
  # Weather data source
  source: "local_cache"            # Uses cached forecast
  
  # Pre-heating/cooling based on forecast
  preheat:
    enabled: true
    forecast_hours: 6              # Look ahead
    outdoor_threshold: 5           # Start pre-heat if forecast < 5°C
    
  precool:
    enabled: true
    forecast_hours: 6
    outdoor_threshold: 25          # Start pre-cool if forecast > 25°C
    
  # Solar gain prediction
  solar:
    enabled: true
    coordinate_with_blinds: true
    reduce_heating_on_sunny: true
```

### Algorithm

1. Fetch weather forecast (cached locally)
2. Predict indoor temperature trajectory
3. Calculate optimal start time for heating/cooling
4. Coordinate with blinds for solar gain management
5. Adjust setpoints based on predicted outdoor conditions

---

## Occupancy Integration

### Presence-Based Control

```yaml
occupancy_control:
  zone_id: "zone-living"
  enabled: true
  
  # Setback when unoccupied
  unoccupied:
    heat_setback: 3.0              # Reduce by 3°C
    cool_setback: 3.0              # Increase by 3°C
    delay_minutes: 30              # Wait before setback
    
  # Recovery when occupied
  occupied:
    restore_immediately: true
    boost_on_return: false
    
  # Detection sources
  detection:
    - type: "motion_sensor"
      devices: ["sensor-living-motion", "sensor-kitchen-motion"]
      
    - type: "door_sensor"
      devices: ["sensor-front-door"]
      trigger: "arrival"
      
    - type: "geofence"
      users: ["user-darren", "user-partner"]
      radius_meters: 500
```

### Geofencing

```yaml
geofencing:
  enabled: true
  
  zones:
    - name: "home_area"
      radius_meters: 500
      center:
        latitude: 51.5074
        longitude: -0.1278
        
    - name: "approach"
      radius_meters: 5000
      
  actions:
    leaving_home_area:
      - set_preset: "away"
      
    entering_approach:
      - cancel_preset: "away"       # Start pre-heating
      
    entering_home_area:
      - set_preset: "home"
```

---

## Equipment Integration

### Heat Pump Control

```yaml
heat_pump:
  device_id: "heatpump-main"
  type: "air_source"
  manufacturer: "Vaillant"
  model: "aroTHERM plus"
  
  # Communication
  protocol: "modbus_tcp"
  address:
    host: "192.168.1.50"
    port: 502
    unit_id: 1
    
  # Operating modes
  modes:
    - heating
    - cooling
    - dhw                          # Domestic hot water
    - auto
    
  # Efficiency optimization
  optimization:
    enabled: true
    prefer_low_output: true        # Better COP at part load
    weather_compensation: true     # Adjust flow temp to outdoor
    
  # Weather compensation curve
  compensation:
    outdoor_20: 25                 # Flow temp at 20°C outdoor
    outdoor_0: 40                  # Flow temp at 0°C outdoor
    outdoor_minus10: 50            # Flow temp at -10°C outdoor
    
  # DHW priority
  dhw:
    priority: true                 # DHW takes priority over heating
    target_temp: 50
    legionella_cycle:
      enabled: true
      day: "sunday"
      time: "02:00"
      temp: 60
      duration_minutes: 30
      
  # Limits
  limits:
    min_outdoor_temp: -20          # Operating limit
    max_flow_temp: 55
    min_runtime_minutes: 10
    min_off_time_minutes: 5
```

### Boiler Control

```yaml
boiler:
  device_id: "boiler-main"
  type: "condensing_gas"
  
  protocol: "opentherm"            # Or modbus
  
  # Control
  control:
    modulating: true               # Modulating vs on/off
    target_flow_temp: 50
    weather_compensation: true
    
  # DHW
  dhw:
    type: "combi"                  # combi | cylinder
    priority: true
    target_temp: 48
    
  # Safety
  safety:
    max_flow_temp: 80
    frost_protect_temp: 5
    error_retry_minutes: 30
```

### MVHR Integration

```yaml
mvhr:
  device_id: "mvhr-main"
  manufacturer: "Zehnder"
  model: "ComfoAir Q"
  
  protocol: "modbus_rtu"
  address:
    port: "/dev/ttyUSB1"
    baudrate: 9600
    unit_id: 1
    
  # Speed control
  speeds:
    low: 30
    medium: 50
    high: 80
    boost: 100
    
  # Automatic control
  auto_control:
    enabled: true
    
    # CO2-based
    co2:
      sensor_id: "sensor-living-co2"
      threshold_low: 600           # Below = can reduce speed
      threshold_high: 1000         # Above = increase speed
      threshold_critical: 1500     # Boost
      
    # Humidity-based
    humidity:
      sensors:
        - "sensor-bathroom-humidity"
        - "sensor-kitchen-humidity"
      threshold_high: 70           # Increase extract
      threshold_critical: 85       # Boost
      
    # Boost triggers
    boost:
      duration_minutes: 30
      triggers:
        - type: "humidity_spike"
          threshold: 10            # % increase in 5 minutes
        - type: "manual_button"
          device_id: "switch-bathroom-boost"
          
  # Summer bypass
  bypass:
    enabled: true
    outdoor_temp_min: 18           # Use bypass above this
    indoor_temp_max: 24            # If indoor > target
```

---

## Air Quality Management

### CO2 Monitoring

```yaml
air_quality:
  zone_id: "zone-living"
  
  sensors:
    - device_id: "sensor-living-co2"
      type: "co2"
      
    - device_id: "sensor-living-voc"
      type: "voc"
      
  thresholds:
    co2:
      good: 600
      moderate: 1000
      poor: 1500
      action: 2000
      
    voc:
      good: 150
      moderate: 300
      poor: 500
      
  actions:
    co2_above_moderate:
      - increase_ventilation: "medium"
      
    co2_above_poor:
      - increase_ventilation: "high"
      - notify_user: true
      
    co2_above_action:
      - increase_ventilation: "boost"
      - notify_user: true
      - alert_severity: "warning"
```

---

## Humidity Control

### Dehumidification

```yaml
humidity_control:
  zone_id: "zone-living"
  
  sensor_id: "sensor-living-humidity"
  
  # Targets
  target:
    winter: 45                     # RH %
    summer: 55
    
  # Control
  dehumidification:
    device_id: "dehumidifier-living"
    threshold_high: 65
    threshold_low: 45
    hysteresis: 5
    
  # Use HVAC dry mode if available
  hvac_dry_mode:
    enabled: true
    threshold: 70
```

---

## Mode Integration

### Climate Behavior by Mode

```yaml
mode_behaviors:
  home:
    use_schedule: true
    occupancy_control: true
    
  away:
    heat_setpoint: 15.0
    cool_setpoint: 28.0
    ventilation: "low"
    
  night:
    heat_setpoint: 18.0
    cool_setpoint: 25.0
    ventilation: "low"
    
  holiday:
    heat_setpoint: 10.0
    cool_setpoint: 30.0
    frost_protect_only: true
    ventilation: "minimum"
    dhw: "off"
```

---

## Safety Rules

### Frost Protection

```yaml
frost_protection:
  enabled: true
  cannot_disable: true             # HARD RULE
  
  threshold_temp: 5.0              # °C
  activation_temp: 3.0             # Turn on heating
  target_temp: 8.0                 # Heat to this
  
  # Which equipment to use
  equipment:
    - device_id: "heatpump-main"
    - device_id: "boiler-backup"
    
  # Notify on activation
  notify: true
```

### Equipment Protection

```yaml
equipment_protection:
  # Minimum runtime to protect compressors
  compressor:
    min_on_time: 180               # seconds
    min_off_time: 180              # seconds
    
  # Pump anti-seize
  pump_exercise:
    enabled: true
    interval_days: 7
    run_seconds: 30
    
  # Valve exercise
  valve_exercise:
    enabled: true
    interval_days: 7
    cycle: true                    # Full open/close cycle
```

---

## PHM Integration

### Equipment Monitoring

```yaml
phm_climate:
  devices:
    - device_id: "heatpump-main"
      metrics:
        - name: "cop"
          baseline_method: "seasonal_average"
          deviation_threshold: 0.5
          alert: "Heat pump efficiency degradation"
          
        - name: "compressor_current"
          baseline_method: "rolling_average_7d"
          deviation_threshold_percent: 20
          alert: "Compressor current deviation"
          
        - name: "defrost_frequency"
          baseline_method: "count_per_day"
          threshold: 10
          alert: "Excessive defrost cycles"
          
    - device_id: "valve-ufh-living"
      metrics:
        - name: "actuator_runtime"
          threshold_hours: 50000
          alert: "Valve actuator approaching end of life"
```

---

## Voice Integration

```yaml
voice_intents:
  - intent: "climate.set_temperature"
    examples:
      - "set the temperature to ${temp} degrees"
      - "make it ${temp} degrees"
      - "turn the heating up"
      - "it's cold in here"
    action:
      command: "set_temperature"
      parameters:
        temperature: "${temp}"      # Or infer adjustment
      target: "${room}"
      
  - intent: "climate.boost"
    examples:
      - "boost the heating"
      - "warm up the bathroom"
    action:
      command: "boost"
      parameters:
        duration_minutes: 30
      target: "${room}"
```

---

## Commercial HVAC

This section covers climate control patterns specific to commercial and office environments.

### VAV Zone Control

Variable Air Volume (VAV) systems are common in commercial buildings. Each VAV box controls airflow to a zone.

```yaml
vav_zone:
  id: "zone-meeting-room-1"
  name: "Meeting Room 1"
  type: "vav"
  
  # VAV box
  vav_device_id: "vav-mr-01"
  
  # Zone sensors
  sensors:
    temperature: "sensor-mr1-temp"
    co2: "sensor-mr1-co2"           # For demand-controlled ventilation
    occupancy: "sensor-mr1-pir"
  
  # Setpoints by occupancy state
  setpoints:
    occupied:
      heating: 21.0
      cooling: 23.0
      min_airflow_percent: 30       # Minimum ventilation
      max_airflow_percent: 100
    unoccupied:
      heating: 16.0
      cooling: 28.0
      min_airflow_percent: 10       # Reduced when empty
      max_airflow_percent: 30
    standby:
      heating: 18.0
      cooling: 26.0
      min_airflow_percent: 15
      max_airflow_percent: 50
  
  # CO2-based ventilation override
  demand_ventilation:
    enabled: true
    co2_setpoint: 800               # ppm target
    co2_max: 1000                   # ppm - increase airflow
    min_airflow_at_setpoint: 30
    max_airflow_at_max: 100
  
  # Reheat (if equipped)
  reheat:
    enabled: true
    type: "hot_water"               # hot_water | electric
    valve_id: "valve-vav-mr1-reheat"
```

**VAV State Model:**
```yaml
VAVState:
  # Airflow
  airflow_setpoint: float           # CFM or %
  actual_airflow: float
  damper_position: float            # 0-100%
  
  # Temperature
  zone_temp: float
  discharge_temp: float
  
  # Heating
  reheat_active: boolean
  reheat_valve_position: float
  
  # Status
  mode: enum                        # heating | cooling | deadband | off
  occupancy: enum                   # occupied | unoccupied | standby
```

### Fan Coil Unit Zones

FCUs provide local heating/cooling in commercial buildings.

```yaml
fcu_zone:
  id: "zone-office-north"
  name: "North Office"
  type: "fcu"
  
  # Fan coil unit
  fcu_device_id: "fcu-office-n-01"
  
  # Configuration
  config:
    cooling_type: "chilled_water"   # chilled_water | dx
    heating_type: "hot_water"       # hot_water | electric | none
    fan_speeds: ["low", "medium", "high", "auto"]
    
  # Control
  setpoints:
    heating: 21.0
    cooling: 24.0
    deadband: 1.0
    
  # Fan speed selection
  fan_control:
    mode: "auto"                    # auto | manual
    speed_at_small_load: "low"
    speed_at_large_load: "high"
```

### Occupancy Scheduling

Commercial buildings typically follow occupancy schedules with pre-conditioning.

```yaml
occupancy_schedule:
  id: "schedule-office-standard"
  name: "Standard Office Hours"
  
  default_state: "unoccupied"
  
  # Weekly schedule
  weekly:
    monday:
      - start: "07:00"
        end: "19:00"
        state: "occupied"
    tuesday:
      - start: "07:00"
        end: "19:00"
        state: "occupied"
    # ... similar for wed, thu, fri
    saturday:
      - start: "09:00"
        end: "13:00"
        state: "occupied"
    sunday: []                      # Unoccupied all day
  
  # Holidays (override weekly)
  holidays:
    - date: "2026-12-25"
      state: "unoccupied"
    - date_range:
        start: "2026-12-24"
        end: "2026-01-02"
      state: "unoccupied"
  
  # Pre-conditioning
  pre_condition:
    enabled: true
    lead_time_minutes: 60           # Start conditioning 1hr before
    adaptive: true                  # Learn optimal lead time
    
  # Optimum start
  optimum_start:
    enabled: true
    target_temp_achieved_by: "occupied_start"
    learning_enabled: true
    max_lead_time_minutes: 120
    
  # Post-occupancy
  post_occupancy:
    delay_minutes: 30               # Run for 30 min after scheduled end
```

### Meeting Room Booking Integration

Meeting rooms can integrate with booking systems for demand-based conditioning.

```yaml
booking_integration:
  zone_id: "zone-meeting-room-1"
  
  # Booking system
  source:
    type: "calendar"                # calendar | webhook | polling
    provider: "microsoft_365"       # Or Google, Exchange, etc.
    room_email: "meetingroom1@company.com"
    
  # Pre-meeting conditioning
  pre_meeting:
    lead_time_minutes: 15           # Condition before meeting
    setpoints:
      heating: 21.0
      cooling: 23.0
    fan_speed: "medium"
    lighting_scene: "meeting-default"
    
  # During meeting
  occupied:
    co2_target: 800
    min_airflow_percent: 50
    
  # Post-meeting (if no next booking)
  post_meeting:
    setback_delay_minutes: 10
    target_state: "unoccupied"
    
  # No booking behavior
  unbooked:
    state: "standby"
    setpoints:
      heating: 18.0
      cooling: 26.0
```

### Out-of-Hours Override

Allow users to request conditioning outside scheduled hours.

```yaml
override_request:
  # Physical button/keypad
  trigger:
    type: "button"
    device_id: "keypad-reception"
    button: 4
    
  # Or via app/UI
  # Or via voice: "extend the heating"
  
  # Override behavior
  override:
    duration_minutes: 120           # Maximum 2 hours
    zone_scope: "area"              # zone | area | site
    area_id: "area-ground-floor"
    
    setpoints:
      heating: 21.0
      cooling: 23.0
      
    # Notification
    notify:
      - facility_manager
      
    # Logging (for billing/audit)
    log:
      user_id: true
      timestamp: true
      duration_actual: true
```

### Night Setback and Purge

```yaml
night_operation:
  # Setback during unoccupied
  setback:
    heating_setpoint: 12.0          # Frost protection++
    cooling_setpoint: 30.0          # Off essentially
    ventilation: "minimum"          # Maintain some fresh air
    
  # Night purge (free cooling)
  night_purge:
    enabled: true
    conditions:
      outside_temp_max: 20          # °C
      outside_temp_min: 12          # °C
      inside_temp_min: 24           # Only if building is warm
      humidity_max: 70              # %RH
    start_time: "02:00"
    end_time: "06:00"
    target_temp: 20                 # Pre-cool building
    
  # Morning warm-up
  morning_warmup:
    enabled: true
    start_time: "adaptive"          # Or fixed time
    end_time: "occupied_start"
    all_zones_to_setpoint: true
    stagger_ahu_starts: true        # Prevent demand spike
    stagger_interval_minutes: 5
```

### Commercial Commissioning Checklist

- [ ] All VAV boxes responding and dampers operational
- [ ] Temperature sensors calibrated (within 0.5°C)
- [ ] CO2 sensors calibrated
- [ ] Occupancy sensors detecting correctly
- [ ] AHU schedules programmed
- [ ] Economizer sequence verified
- [ ] Optimum start learning enabled
- [ ] Night setback programmed
- [ ] Out-of-hours override configured
- [ ] Booking system integration tested (if applicable)
- [ ] BMS trending enabled for commissioning verification

---

## Related Documents

- [KNX Protocol Specification](../protocols/knx.md) — KNX HVAC integration
- [Modbus Protocol Specification](../protocols/modbus.md) — Heat pump integration
- [BACnet Protocol Specification](../protocols/bacnet.md) — Commercial HVAC (Year 2)
- [PHM Specification](../intelligence/phm.md) — Predictive health monitoring framework
- [Plant Domain Specification](plant.md) — AHU and chiller control
- [Blinds Domain Specification](blinds.md) — Solar gain coordination
- [Data Model: Entities](../data-model/entities.md) — ClimateZone entity

