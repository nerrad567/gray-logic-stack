---
title: Irrigation Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - data-model/entities.md
  - domains/lighting.md
---

# Irrigation Domain Specification

This document specifies how Gray Logic manages garden irrigation, outdoor watering, and related outdoor automation including garden lighting schedules.

---

## Overview

### Philosophy

Irrigation in Gray Logic balances plant health, water conservation, and convenience:

| Goal | Approach |
|------|----------|
| **Plant health** | Right amount of water at right time |
| **Water conservation** | Weather-aware, soil-aware scheduling |
| **Convenience** | Automated schedules with smart adjustments |
| **Integration** | Coordinate with weather, presence, lighting |

### Design Principles

1. **Weather-aware** — Skip watering when rain is forecast
2. **Time-appropriate** — Water early morning or evening (less evaporation)
3. **Zone-based** — Different plants need different watering
4. **Fail-safe** — Stuck valve detection, flood prevention
5. **Water-conscious** — Track usage, detect leaks

---

## System Components

### Irrigation Controller

The physical controller that operates valves.

```yaml
IrrigationController:
  id: uuid
  name: string                        # "Main Irrigation Controller"
  
  # Hardware
  manufacturer: string                # "rainbird", "hunter", "rachio"
  model: string
  
  # Connection
  connection:
    type: "ip" | "serial" | "modbus"
    host: string | null
    port: integer | null
    
  # Zone count
  max_zones: integer                  # Number of valve outputs
  
  # Capabilities
  capabilities:
    flow_sensing: boolean             # Has flow meter input
    rain_sensor: boolean              # Has rain sensor input
    soil_moisture: boolean            # Has soil sensor inputs
    
  # Current state
  state:
    active_zone: integer | null       # Currently running zone (0 = none)
    rain_delay: boolean               # Rain delay active
    flow_rate_lpm: float | null       # Current flow (if flow sensing)
```

### Irrigation Zone

A group of sprinklers/drippers controlled together.

```yaml
IrrigationZone:
  id: uuid
  controller_id: uuid
  zone_number: integer                # Output number on controller
  name: string                        # "Front Lawn", "Flower Beds"
  
  # Zone characteristics
  config:
    zone_type: enum                   # See zone types below
    plant_type: enum                  # Type of plants
    soil_type: enum                   # Soil characteristics
    sun_exposure: enum                # sun, partial, shade
    slope: enum                       # flat, slight, steep
    
    # Watering parameters
    default_duration_minutes: integer
    flow_rate_lpm: float | null       # Known flow rate
    precipitation_rate_mmh: float | null  # Sprinkler precipitation rate
    
  # Area
  area_sqm: float | null              # Zone area in square meters
  
  # Current state
  state:
    running: boolean
    run_time_remaining_seconds: integer | null
    last_run: timestamp | null
    last_duration_minutes: integer | null
```

**Zone Types:**

| Type | Description | Typical Duration |
|------|-------------|------------------|
| `lawn_spray` | Pop-up spray heads on lawn | 10-20 min |
| `lawn_rotor` | Rotating heads on lawn | 30-45 min |
| `drip` | Drip irrigation for beds | 30-60 min |
| `microspray` | Micro sprayers for beds | 15-30 min |
| `subsurface` | Underground drip | 45-90 min |

**Plant Types:**

| Type | Water Needs |
|------|-------------|
| `grass_cool` | Cool-season grass (fescue, ryegrass) |
| `grass_warm` | Warm-season grass (bermuda, zoysia) |
| `shrubs` | Established shrubs |
| `perennials` | Flower beds, perennials |
| `annuals` | Annual flowers (high water) |
| `vegetables` | Vegetable garden |
| `trees` | Established trees |
| `natives` | Native/drought-tolerant plants |

**Soil Types:**

| Type | Characteristics |
|------|-----------------|
| `clay` | Slow absorption, holds water |
| `loam` | Balanced, ideal |
| `sandy` | Fast draining, needs more water |
| `rocky` | Variable, often fast draining |

---

## Schedules

### Basic Schedule

```yaml
IrrigationSchedule:
  id: uuid
  name: string                        # "Summer Morning Schedule"
  enabled: boolean
  
  # When to run
  timing:
    days: ["mon", "wed", "fri", "sun"]  # Or "odd", "even", "interval:3"
    start_time: "05:30"               # Start time
    
  # What to run
  zones:
    - zone_id: "zone-front-lawn"
      duration_minutes: 20
      
    - zone_id: "zone-back-lawn"
      duration_minutes: 25
      
    - zone_id: "zone-flower-beds"
      duration_minutes: 15
      
  # Run sequentially or with soak cycles
  mode: "sequential" | "cycle_soak"
  
  # For cycle & soak
  cycle_soak:
    cycle_minutes: 10                 # Run for 10 min
    soak_minutes: 30                  # Wait 30 min
    cycles: 2                         # Repeat twice
    
  # Seasonal adjustment
  seasonal_adjust:
    enabled: true
    current_percent: 80               # Currently at 80% of base duration
```

### Smart Scheduling

Weather and condition-aware adjustments:

```yaml
SmartSchedule:
  schedule_id: "schedule-summer"
  
  adjustments:
    # Weather-based
    weather:
      skip_if_rain_forecast_mm: 5     # Skip if >5mm forecast
      skip_if_rained_mm_24h: 10       # Skip if >10mm in last 24h
      reduce_if_humidity_above: 80    # Reduce if humid
      increase_if_temp_above_c: 30    # Increase if hot
      wind_delay_above_kmh: 30        # Delay if windy
      
    # Soil moisture (if sensors available)
    soil:
      skip_if_moisture_above: 60      # Skip if soil moist (%)
      increase_if_moisture_below: 20  # Increase if dry
      
    # Seasonal
    seasonal:
      method: "monthly"               # or "et_based"
      monthly_adjust:
        jan: 0
        feb: 0
        mar: 40
        apr: 60
        may: 80
        jun: 100
        jul: 100
        aug: 100
        sep: 80
        oct: 60
        nov: 40
        dec: 0
```

### ET-Based Scheduling

Evapotranspiration-based watering (advanced):

```yaml
ETSchedule:
  enabled: true
  
  # ET data source
  et_source:
    type: "weather_api"               # or "local_station"
    api: "openweathermap"
    
  # Crop coefficients by zone
  zones:
    - zone_id: "zone-front-lawn"
      crop_coefficient: 0.8           # Lawn Kc
      root_depth_cm: 15
      
    - zone_id: "zone-flower-beds"
      crop_coefficient: 0.5           # Mixed plants Kc
      root_depth_cm: 30
      
  # Calculate water needed
  calculation:
    # Water needed = ET × Kc × Area / Efficiency
    efficiency: 0.75                  # Irrigation efficiency
    precipitation_rate_mmh: 25        # Sprinkler output
```

---

## Commands

### Manual Control

```yaml
# Start a zone manually
- zone_id: "zone-front-lawn"
  command: "start"
  parameters:
    duration_minutes: 15

# Stop a zone
- zone_id: "zone-front-lawn"
  command: "stop"

# Stop all zones
- controller_id: "controller-main"
  command: "stop_all"
```

### Schedule Control

```yaml
# Run a schedule now
- schedule_id: "schedule-summer"
  command: "run_now"

# Skip next scheduled run
- schedule_id: "schedule-summer"
  command: "skip_next"
  parameters:
    reason: "manual_skip"

# Rain delay
- controller_id: "controller-main"
  command: "rain_delay"
  parameters:
    hours: 48
```

### Seasonal Adjustment

```yaml
# Set seasonal adjustment
- controller_id: "controller-main"
  command: "seasonal_adjust"
  parameters:
    percent: 80                       # 80% of normal duration
```

---

## Weather Integration

### Rain Skip

```yaml
RainSkip:
  enabled: true
  
  sources:
    # Hardware rain sensor
    - type: "sensor"
      device_id: "rain-sensor-1"
      
    # Weather forecast
    - type: "forecast"
      provider: "openweathermap"
      threshold_mm: 5
      lookahead_hours: 24
      
    # Recent rainfall
    - type: "history"
      threshold_mm: 10
      lookback_hours: 24
      
  # Any source can trigger skip
  logic: "any"
```

### Freeze Protection

```yaml
FreezeProtection:
  enabled: true
  
  # Skip watering if freezing
  skip_if_temp_below_c: 2
  
  # Look ahead for freeze
  forecast_hours: 12
  
  # Optional: Run briefly to prevent pipe freeze
  anti_freeze_run:
    enabled: false                    # Usually not needed
    trigger_temp_c: -5
    duration_seconds: 30
```

### Wind Delay

```yaml
WindDelay:
  enabled: true
  
  # Delay if wind above threshold
  threshold_kmh: 25
  
  # Check forecast
  check_forecast: true
  
  # Retry window
  retry_within_hours: 4
```

---

## Water Management

### Flow Monitoring

```yaml
FlowMonitor:
  device_id: "flow-meter-main"
  
  # Expected vs actual
  zones:
    - zone_id: "zone-front-lawn"
      expected_flow_lpm: 45
      tolerance_percent: 20
      
  # Alerts
  alerts:
    high_flow:
      threshold_percent: 130          # >130% of expected
      action: "alert"
      
    low_flow:
      threshold_percent: 70           # <70% of expected
      action: "alert"
      
    no_flow:
      threshold_lpm: 2
      action: "stop_and_alert"
      
    flow_when_off:
      threshold_lpm: 5
      action: "alert"                 # Possible leak
```

### Water Usage Tracking

```yaml
WaterUsage:
  period: "month"
  
  summary:
    total_liters: 12500
    total_cubic_m: 12.5
    cost_estimate: 25.00              # Based on water rate
    
  by_zone:
    - zone: "Front Lawn"
      liters: 5200
      percent: 41.6
      
    - zone: "Back Lawn"
      liters: 4800
      percent: 38.4
      
    - zone: "Flower Beds"
      liters: 2500
      percent: 20.0
      
  comparison:
    last_month_liters: 14200
    change_percent: -12.0
    
  efficiency:
    schedule_adherence: 95            # % of scheduled runs completed
    rain_skips: 4                     # Runs skipped due to rain
    water_saved_liters: 3200          # Estimated savings
```

---

## Automation Integration

### Mode Integration

```yaml
modes:
  - id: "away"
    behaviours:
      irrigation:
        enabled: true                 # Keep watering while away
        adjust_percent: 90            # Slightly reduce
        
  - id: "vacation"
    behaviours:
      irrigation:
        enabled: true
        adjust_percent: 80
        skip_manual: true             # Disable manual runs
        
  - id: "winter"
    behaviours:
      irrigation:
        enabled: false                # Winterize - no watering
        blow_out_reminder: true       # Remind to blow out lines
```

### Scene Integration

```yaml
scenes:
  - id: "scene-garden-party"
    name: "Garden Party"
    actions:
      # Turn off any running irrigation
      - domain: "irrigation"
        command: "stop_all"
        
      # Delay any scheduled runs
      - domain: "irrigation"
        command: "rain_delay"
        parameters:
          hours: 6
          
      # Turn on garden lighting
      - domain: "lighting"
        scope: "area-garden"
        command: "scene"
        parameters:
          scene: "party"
```

### Event Triggers

```yaml
triggers:
  # Leak detected → Stop irrigation
  - type: "device_state_changed"
    source:
      device_id: "flow-meter-main"
      condition: "flow_when_idle"
    execute:
      - domain: "irrigation"
        command: "stop_all"
      - domain: "notification"
        command: "send"
        parameters:
          title: "Irrigation Leak Detected"
          message: "Unexpected water flow detected - irrigation stopped"
          priority: "high"

  # Soil moisture low → Start zone
  - type: "sensor_threshold"
    source:
      device_id: "soil-moisture-veg"
      condition: "below"
      value: 25
    conditions:
      - type: "time"
        operator: "between"
        value: ["05:00", "09:00"]
    execute:
      - zone_id: "zone-vegetables"
        command: "start"
        parameters:
          duration_minutes: 20
```

---

## Outdoor Lighting Integration

Irrigation often shares controllers and schedules with garden lighting:

```yaml
OutdoorLighting:
  # Garden lights can be on same controller
  controller_id: "controller-outdoor"
  
  zones:
    - id: "lights-path"
      name: "Path Lights"
      channel: 9                      # If relay-based
      
    - id: "lights-accent"
      name: "Accent Lights"
      channel: 10
      
  schedules:
    - id: "schedule-evening-lights"
      zones: ["lights-path", "lights-accent"]
      trigger:
        type: "sunset"
        offset_minutes: -15           # 15 min before sunset
      duration_until:
        type: "time"
        time: "23:00"
        
    - id: "schedule-morning-lights"
      zones: ["lights-path"]
      trigger:
        type: "time"
        time: "06:00"
      duration_until:
        type: "sunrise"
        offset_minutes: 30
```

---

## MQTT Topics

### Zone State

```yaml
topic: graylogic/irrigation/zone/{zone_id}/state
payload:
  zone_id: "zone-front-lawn"
  timestamp: "2026-01-12T05:30:00Z"
  state:
    running: true
    run_time_remaining_seconds: 720
    flow_rate_lpm: 42.5
```

### Controller State

```yaml
topic: graylogic/irrigation/controller/{controller_id}/state
payload:
  controller_id: "controller-main"
  timestamp: "2026-01-12T05:30:00Z"
  state:
    active_zone: 1
    rain_delay: false
    rain_sensor: "dry"
    flow_rate_lpm: 42.5
```

### Commands

```yaml
topic: graylogic/irrigation/command
payload:
  target: "zone-front-lawn"
  command: "start"
  parameters:
    duration_minutes: 15
  request_id: "req-12345"
```

---

## PHM Integration

### PHM Value for Irrigation

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Valves | ★★★☆☆ | Stuck open/closed, slow to open |
| Flow meter | ★★★☆☆ | Accuracy drift |
| Controller | ★★☆☆☆ | Communication errors |
| Pumps (if used) | ★★★★☆ | Current, pressure, flow |

### PHM Configuration

```yaml
phm_irrigation:
  devices:
    - device_id: "valve-zone-1"
      type: "irrigation_valve"
      parameters:
        - name: "open_time_ms"
          baseline_method: "initial_calibration"
          deviation_threshold_percent: 50
          alert: "Valve slow to open - possible obstruction"
          
        - name: "stuck_detection"
          condition: "flow_when_closed"
          threshold_lpm: 5
          alert: "Valve stuck open - manual inspection required"
          
    - device_id: "pump-irrigation"
      type: "pump"
      parameters:
        - name: "pressure_psi"
          baseline_method: "context_aware"
          context_parameter: "flow_rate"
          deviation_threshold_percent: 20
          alert: "Pump pressure deviation - check for leaks or blockage"
          
        - name: "current_a"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 25
          alert: "Pump current elevated - motor issue"
```

---

## Configuration Examples

### Residential: Basic

```yaml
irrigation:
  controller:
    manufacturer: "rachio"
    model: "3"
    connection:
      type: "api"
      api_key_env: "RACHIO_API_KEY"
      
  zones:
    - id: "zone-front-lawn"
      name: "Front Lawn"
      zone_number: 1
      zone_type: "lawn_spray"
      default_duration_minutes: 15
      
    - id: "zone-back-lawn"
      name: "Back Lawn"
      zone_number: 2
      zone_type: "lawn_rotor"
      default_duration_minutes: 25
      
    - id: "zone-flower-beds"
      name: "Flower Beds"
      zone_number: 3
      zone_type: "drip"
      default_duration_minutes: 30
      
  schedules:
    - id: "schedule-summer"
      name: "Summer Watering"
      timing:
        days: ["mon", "wed", "fri"]
        start_time: "05:30"
      zones:
        - zone_id: "zone-front-lawn"
        - zone_id: "zone-back-lawn"
        - zone_id: "zone-flower-beds"
      smart:
        rain_skip: true
        freeze_skip: true
        seasonal_adjust: true
```

### Residential: Advanced

```yaml
irrigation:
  controller:
    manufacturer: "hunter"
    model: "pro-hc"
    connection:
      type: "ip"
      host: "192.168.1.90"
      
  flow_meter:
    device_id: "flow-meter-main"
    pulse_per_liter: 1.0
    
  weather:
    provider: "openweathermap"
    api_key_env: "OWM_API_KEY"
    
  zones:
    - id: "zone-front-lawn"
      name: "Front Lawn"
      zone_number: 1
      zone_type: "lawn_spray"
      plant_type: "grass_cool"
      soil_type: "clay"
      sun_exposure: "sun"
      area_sqm: 150
      precipitation_rate_mmh: 25
      expected_flow_lpm: 45
      
  smart_schedule:
    enabled: true
    et_based: true
    rain_skip_mm: 5
    freeze_skip_c: 2
```

### Commercial: Large Site

```yaml
irrigation:
  controllers:
    - id: "controller-north"
      manufacturer: "rainbird"
      model: "esp-lxme"
      zones: 24
      connection:
        type: "modbus_tcp"
        host: "192.168.1.91"
        
    - id: "controller-south"
      manufacturer: "rainbird"
      model: "esp-lxme"
      zones: 24
      connection:
        type: "modbus_tcp"
        host: "192.168.1.92"
        
  pump_station:
    device_id: "pump-irrigation-main"
    flow_meter: "flow-meter-main"
    pressure_sensor: "pressure-main"
    
  central_control:
    enabled: true
    flow_management: true
    max_simultaneous_zones: 4
    max_flow_lpm: 200
```

---

## Best Practices

### Do's

1. **Water early morning** — Less evaporation, less disease
2. **Check rain forecast** — Don't water before rain
3. **Monitor flow** — Detect leaks early
4. **Seasonal adjust** — Plants need less water in cool months
5. **Zone appropriately** — Group similar plants together
6. **Deep and infrequent** — Better for root development

### Don'ts

1. **Don't water midday** — High evaporation
2. **Don't overwater** — Wastes water, harms plants
3. **Don't ignore leaks** — Small leaks waste lots of water
4. **Don't forget winterization** — Freeze damage is expensive
5. **Don't water in wind** — Poor distribution

---

## Related Documents

- [Lighting Domain](lighting.md) — Outdoor lighting control
- [Weather Integration](../intelligence/weather.md) — Weather data (to be created)
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring
- [Automation Specification](../automation/automation.md) — Schedule integration
- [Leak Protection](leak-protection.md) — Water leak detection
