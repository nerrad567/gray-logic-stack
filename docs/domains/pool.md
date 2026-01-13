---
title: Pool Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/principles.md
  - architecture/core-internals.md
  - data-model/entities.md
  - domains/plant.md
---

# Pool Domain Specification

This document specifies how Gray Logic manages swimming pools, spas, and water features, with particular emphasis on **safety-critical** chemical dosing systems.

For pool pumps and heating equipment, see [Plant Domain](plant.md).

---

## Overview

### Philosophy

Pool automation balances convenience, safety, and water quality:

| Goal | Approach |
|------|----------|
| **Water quality** | Continuous monitoring, automated dosing |
| **Safety** | Strict limits, interlocks, manual overrides |
| **Energy efficiency** | Smart pump scheduling, solar heating priority |
| **Convenience** | Automated maintenance, remote monitoring |

### Safety Classification

| Function | Safety Level | Automation |
|----------|--------------|------------|
| Water quality monitoring | Standard | ✅ Full |
| Pump/filter scheduling | Standard | ✅ Full |
| Pool cover control | Standard | ✅ Full |
| Heating control | Standard | ✅ Full |
| Chemical dosing | **CRITICAL** | ⚠️ With safeguards |
| Water features | Standard | ✅ Full |

### Critical Safety Rules

**Chemical dosing is SAFETY-CRITICAL:**

1. **Never dose chlorine and acid simultaneously** — Toxic gas risk
2. **Maximum dose limits** — Prevent overdose
3. **Flow verification** — Only dose when water is circulating
4. **Sensor validation** — Don't trust single reading
5. **Manual override** — Always possible to stop dosing
6. **Fail-safe** — Power loss = dosing stops

---

## Water Quality Monitoring

### Key Parameters

| Parameter | Ideal Range | Test Frequency |
|-----------|-------------|----------------|
| **pH** | 7.2 - 7.6 | Continuous |
| **Free Chlorine** | 1 - 3 ppm | Continuous |
| **ORP** | 650 - 750 mV | Continuous |
| **Temperature** | 26 - 30°C | Continuous |
| **TDS** | < 1500 ppm | Weekly |
| **Alkalinity** | 80 - 120 ppm | Weekly |
| **Cyanuric Acid** | 30 - 50 ppm | Monthly |
| **Calcium Hardness** | 200 - 400 ppm | Monthly |

### Sensor Configuration

```yaml
PoolSensors:
  pool_id: uuid
  
  # Continuous sensors
  continuous:
    - id: "sensor-ph"
      type: "ph"
      location: "return_line"
      calibration:
        last_calibration: timestamp
        next_calibration: timestamp
        offset: float
        
    - id: "sensor-orp"
      type: "orp"
      location: "return_line"
      
    - id: "sensor-temp"
      type: "temperature"
      location: "return_line"
      
    - id: "sensor-flow"
      type: "flow"
      location: "return_line"
      
  # Validation
  validation:
    # Cross-check sensors
    ph_orp_correlation: true
    
    # Ignore readings when flow is low
    min_flow_for_reading_lpm: 50
    
    # Sensor fault detection
    stuck_reading_minutes: 30
    reading_range_check: true
```

### Water Quality State

```yaml
WaterQualityState:
  pool_id: uuid
  timestamp: timestamp
  
  readings:
    ph: 7.4
    orp_mv: 720
    free_chlorine_ppm: 2.1           # Calculated from ORP/temp
    temperature_c: 28.5
    flow_lpm: 180
    
  status:
    overall: "good" | "attention" | "action_required"
    
    parameters:
      - name: "pH"
        value: 7.4
        status: "ok"
        in_range: true
        
      - name: "Chlorine"
        value: 2.1
        status: "ok"
        in_range: true
        
  last_dose:
    chlorine: timestamp | null
    acid: timestamp | null
```

---

## Chemical Dosing

### Dosing System

```yaml
DosingSystem:
  pool_id: uuid
  
  # Chlorine dosing
  chlorine:
    type: "liquid" | "salt_chlorinator" | "tablet"
    pump_id: uuid | null              # Peristaltic pump
    tank_level_sensor: uuid | null
    
    # For salt chlorinators
    chlorinator:
      device_id: uuid
      cell_hours: integer
      
  # Acid dosing
  acid:
    type: "muriatic" | "co2"
    pump_id: uuid
    tank_level_sensor: uuid | null
    
  # Control mode
  mode: "automatic" | "manual" | "disabled"
  
  # Current state
  state:
    chlorine_dosing: boolean
    acid_dosing: boolean
    chlorine_tank_percent: integer | null
    acid_tank_percent: integer | null
```

### Dosing Safety System

```yaml
DosingSafety:
  # CRITICAL: Never dose simultaneously
  interlock:
    chlorine_acid_lockout: true
    minimum_gap_seconds: 300          # 5 min between different chemicals
    
  # Flow verification
  flow_interlock:
    enabled: true
    minimum_flow_lpm: 50
    flow_sensor_id: "sensor-flow"
    
  # Maximum dose limits
  limits:
    chlorine:
      max_dose_ml: 500                # Per dosing cycle
      max_per_hour_ml: 1000
      max_per_day_ml: 5000
      
    acid:
      max_dose_ml: 300
      max_per_hour_ml: 600
      max_per_day_ml: 3000
      
  # Sensor validation
  sensor_validation:
    # Require stable reading before dosing
    stable_reading_minutes: 5
    max_deviation: 0.2                # pH units
    
    # Cross-check ORP and calculated chlorine
    orp_chlorine_correlation: true
    
  # Manual confirmation for large doses
  confirmation_required:
    chlorine_above_ml: 300
    acid_above_ml: 200
    
  # Emergency stop
  emergency_stop:
    button_device_id: uuid | null
    app_accessible: true
```

### Dosing Logic

```yaml
DosingControl:
  chlorine:
    # Target
    target_orp_mv: 700
    target_chlorine_ppm: 2.0
    
    # Control
    control_method: "orp"             # or "chlorine_sensor"
    
    # When to dose
    dose_when:
      orp_below_mv: 680
      # AND
      flow_above_lpm: 50
      # AND
      ph_in_range: [7.0, 7.8]
      # AND
      last_acid_dose_minutes_ago: 5
      
    # Dose calculation
    dosing:
      method: "proportional"
      ml_per_mv_deficit: 10           # 10ml per mV below setpoint
      min_dose_ml: 50
      max_dose_ml: 500
      wait_after_dose_minutes: 30     # Wait for effect
      
  acid:
    # Target
    target_ph: 7.4
    
    # When to dose
    dose_when:
      ph_above: 7.6
      # AND
      flow_above_lpm: 50
      # AND
      last_chlorine_dose_minutes_ago: 5
      
    # Dose calculation
    dosing:
      method: "proportional"
      ml_per_ph_excess: 100           # 100ml per 0.1 pH above setpoint
      min_dose_ml: 30
      max_dose_ml: 300
      wait_after_dose_minutes: 30
```

### Dosing Interlocks

```yaml
DosingInterlocks:
  # All must be satisfied before dosing
  preconditions:
    - name: "Pump running"
      condition: "pump_state == 'running'"
      
    - name: "Flow adequate"
      condition: "flow_lpm >= 50"
      
    - name: "No other chemical dosing"
      condition: "other_dosing == false"
      
    - name: "Chemical gap satisfied"
      condition: "time_since_other_dose >= 300"
      
    - name: "Sensors valid"
      condition: "sensors_valid == true"
      
    - name: "Not in maintenance mode"
      condition: "maintenance_mode == false"
      
  # Any triggers emergency stop
  emergency_stop_triggers:
    - "flow_sensor_fault"
    - "ph_sensor_fault"
    - "orp_sensor_fault"
    - "pump_stopped_unexpectedly"
    - "manual_stop_pressed"
    - "dose_limit_exceeded"
```

---

## Pool Equipment

### Pump & Filter

See [Plant Domain](plant.md) for detailed pump specifications. Integration:

```yaml
PoolPump:
  # Reference to plant equipment
  pump_device_id: "pump-pool-main"
  
  # Pool-specific scheduling
  schedules:
    - name: "Summer Daytime"
      season: ["jun", "jul", "aug"]
      time: "08:00-20:00"
      speed_percent: 75
      
    - name: "Summer Night"
      season: ["jun", "jul", "aug"]
      time: "20:00-08:00"
      speed_percent: 40
      
    - name: "Winter"
      season: ["nov", "dec", "jan", "feb"]
      time: "10:00-16:00"
      speed_percent: 50
      
  # Turnover calculation
  turnover:
    pool_volume_liters: 50000
    target_turnovers_per_day: 2
    calculated_runtime_hours: 8
```

### Pool Heating

```yaml
PoolHeating:
  # Heat sources (in priority order)
  sources:
    - type: "solar"
      device_id: "solar-pool"
      priority: 1
      
    - type: "heat_pump"
      device_id: "heatpump-pool"
      priority: 2
      
    - type: "gas"
      device_id: "boiler-pool"
      priority: 3
      
  # Control
  control:
    target_temperature_c: 28
    hysteresis_c: 0.5
    
    # Solar priority
    solar_priority:
      enabled: true
      use_solar_if_available: true
      backup_if_solar_insufficient: true
      
    # Cover-based heating
    heat_with_cover_on: true          # More efficient
    
  # Schedules
  schedules:
    - name: "Party Mode"
      trigger: "scene-pool-party"
      target_c: 30
      preheat_hours: 4
```

### Pool Cover

```yaml
PoolCover:
  id: uuid
  name: string                        # "Main Pool Cover"
  
  # Type
  type: "safety" | "thermal" | "automatic" | "solar"
  
  # Control
  control:
    type: "motorized" | "manual"
    device_id: uuid | null            # Motor controller
    
  # Safety features
  safety:
    # Interlock with pool pump
    pump_interlock: true              # Stop pump if cover stuck
    
    # Sensor for cover position
    position_sensors:
      open: uuid
      closed: uuid
      
  # Configuration
  config:
    travel_time_seconds: 120
    auto_close_after_minutes: 60 | null
    
  # Current state
  state:
    position: "open" | "closed" | "moving" | "error"
    last_opened: timestamp
    last_closed: timestamp
```

### Water Features

```yaml
WaterFeatures:
  features:
    - id: "feature-waterfall"
      name: "Waterfall"
      type: "waterfall"
      pump_id: "pump-waterfall"
      
    - id: "feature-fountain"
      name: "Fountain"
      type: "fountain"
      pump_id: "pump-fountain"
      
    - id: "feature-jets"
      name: "Massage Jets"
      type: "spa_jets"
      pump_id: "pump-jets"
      blower_id: "blower-jets" | null
      
  # Scheduling
  schedules:
    - feature: "feature-waterfall"
      enabled_hours: ["10:00", "18:00"]
      enabled_modes: ["home", "entertaining"]
      
  # Integration
  scene_integration:
    - scene: "pool-party"
      enable: ["feature-waterfall", "feature-fountain"]
```

---

## Automation Integration

### Mode Integration

```yaml
modes:
  - id: "home"
    behaviours:
      pool:
        dosing: "automatic"
        cover: "open"                 # During day
        heating: true
        
  - id: "away"
    behaviours:
      pool:
        dosing: "automatic"           # Keep chemistry right
        cover: "closed"               # Safety + heat retention
        heating: "economy"            # Lower setpoint
        features: "off"
        
  - id: "vacation"
    behaviours:
      pool:
        dosing: "automatic"
        cover: "closed"
        heating: "minimum"            # Frost protection only
        pump_schedule: "economy"
        
  - id: "winter"
    behaviours:
      pool:
        winterized: true              # Different operating mode
```

### Scene Integration

```yaml
scenes:
  - id: "scene-pool-party"
    name: "Pool Party"
    actions:
      # Open cover
      - device_id: "cover-pool"
        command: "open"
        
      # Start features
      - device_id: "feature-waterfall"
        command: "on"
      - device_id: "feature-fountain"
        command: "on"
        
      # Pool lighting
      - domain: "lighting"
        scope: "area-pool"
        command: "scene"
        parameters:
          scene: "party"
          
      # Increase heating
      - domain: "pool"
        command: "set_temperature"
        parameters:
          target_c: 30
          
  - id: "scene-pool-close"
    name: "Close Pool"
    actions:
      - device_id: "cover-pool"
        command: "close"
      - device_id: "feature-waterfall"
        command: "off"
      - device_id: "feature-fountain"
        command: "off"
```

### Event Triggers

```yaml
triggers:
  # Water quality alert
  - type: "pool_water_quality"
    condition:
      parameter: "ph"
      operator: "outside_range"
      range: [7.0, 7.8]
      duration_minutes: 30
    execute:
      - domain: "notification"
        action: "send"
        parameters:
          title: "Pool pH Alert"
          message: "Pool pH is ${ph} - outside safe range"
          priority: "high"
          
  # Chemical tank low
  - type: "level_below"
    source:
      device_id: "tank-chlorine"
      threshold_percent: 20
    execute:
      - domain: "notification"
        action: "send"
        parameters:
          message: "Chlorine tank low (${level}%) - please refill"
          
  # Cover left open at night
  - type: "time"
    time: "22:00"
    conditions:
      - type: "device_state"
        device_id: "cover-pool"
        state: "open"
    execute:
      - domain: "notification"
        action: "send"
        parameters:
          message: "Pool cover is still open"
          actions:
            - label: "Close Cover"
              action: "close_pool_cover"
```

---

## Spa/Hot Tub

### Spa Configuration

```yaml
Spa:
  id: uuid
  name: string                        # "Main Spa"
  
  # Equipment
  equipment:
    pump_id: uuid
    heater_id: uuid
    blower_id: uuid | null
    jets: [uuid]
    
  # Water quality (typically combined with pool or separate)
  chemistry:
    separate_system: boolean
    dosing_system_id: uuid | null
    
  # Configuration
  config:
    volume_liters: 1500
    target_temperature_c: 38
    max_temperature_c: 40
    
  # Modes
  modes:
    ready:
      maintain_temperature: true
      filter_schedule: "4h_twice_daily"
      
    eco:
      maintain_temperature: false
      heat_on_demand: true
      preheat_time_minutes: 45
      
    off:
      frost_protection_only: true
```

### Spa Automation

```yaml
SpaAutomation:
  # Ready mode scheduling
  ready_schedule:
    - days: ["fri", "sat", "sun"]
      time: "16:00"
      duration_hours: 6
      
  # On-demand heating
  on_demand:
    trigger: "voice_command"          # "Hey Gray, heat the spa"
    or_trigger: "app_button"
    preheat_time_minutes: 45
    ready_notification: true
    
  # Auto-off
  auto_off:
    after_use_minutes: 60
    max_runtime_hours: 4
```

---

## MQTT Topics

### Water Quality

```yaml
topic: graylogic/pool/{pool_id}/water_quality
payload:
  pool_id: "pool-main"
  timestamp: "2026-01-12T14:30:00Z"
  readings:
    ph: 7.4
    orp_mv: 720
    temperature_c: 28.5
    flow_lpm: 180
  status: "good"
```

### Dosing State

```yaml
topic: graylogic/pool/{pool_id}/dosing
payload:
  pool_id: "pool-main"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    mode: "automatic"
    chlorine_dosing: false
    acid_dosing: false
    last_chlorine_dose: "2026-01-12T12:30:00Z"
    last_acid_dose: "2026-01-12T13:15:00Z"
  tanks:
    chlorine_percent: 65
    acid_percent: 45
```

### Commands

```yaml
# Manual dose (requires confirmation)
topic: graylogic/pool/command
payload:
  pool_id: "pool-main"
  command: "manual_dose"
  parameters:
    chemical: "chlorine"
    amount_ml: 200
    user_pin: "1234"
  request_id: "req-12345"

# Emergency stop
topic: graylogic/pool/command
payload:
  pool_id: "pool-main"
  command: "emergency_stop"
  request_id: "req-12346"
```

---

## PHM Integration

### PHM Value for Pool Equipment

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Pool pump | ★★★★★ | Current, vibration, runtime |
| Chlorinator cell | ★★★★☆ | Cell hours, output efficiency |
| Dosing pumps | ★★★★☆ | Strokes, prime, tube wear |
| pH/ORP sensors | ★★★★☆ | Drift, calibration age |
| Heat pump | ★★★★☆ | COP, defrost cycles |

### PHM Configuration

```yaml
phm_pool:
  devices:
    - device_id: "pump-pool-main"
      type: "pool_pump"
      parameters:
        - name: "current_a"
          baseline_method: "context_aware"
          context_parameter: "speed_percent"
          deviation_threshold_percent: 20
          alert: "Pump current deviation - check impeller/motor"
          
        - name: "strainer_dp_bar"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 50
          alert: "Strainer pressure drop high - clean basket"
          
    - device_id: "chlorinator-cell"
      type: "salt_chlorinator"
      parameters:
        - name: "cell_hours"
          threshold: 10000
          alert_at_percent: 80
          alert: "Chlorinator cell approaching end of life"
          
        - name: "output_efficiency"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 30
          alert: "Cell efficiency declining - inspect/clean cell"
          
    - device_id: "sensor-ph"
      type: "ph_sensor"
      parameters:
        - name: "calibration_age_days"
          threshold: 30
          alert: "pH sensor calibration due"
          
        - name: "reading_stability"
          baseline_method: "rolling_std_dev"
          deviation_threshold: 0.1
          alert: "pH sensor unstable - may need replacement"
```

---

## Maintenance

### Scheduled Maintenance

```yaml
MaintenanceSchedule:
  tasks:
    - task: "Backwash filter"
      frequency: "weekly"
      trigger:
        type: "filter_dp"
        threshold_bar: 0.8
      notification: true
      
    - task: "Clean strainer basket"
      frequency: "weekly"
      notification: true
      
    - task: "Calibrate pH sensor"
      frequency: "monthly"
      notification: true
      procedure: "docs/pool-calibration.md"
      
    - task: "Calibrate ORP sensor"
      frequency: "monthly"
      notification: true
      
    - task: "Inspect chlorinator cell"
      frequency: "quarterly"
      notification: true
      
    - task: "Full water test"
      frequency: "monthly"
      notification: true
      parameters: ["alkalinity", "calcium", "cya", "tds"]
```

### Winterization

```yaml
Winterization:
  # Pre-winter checklist
  checklist:
    - "Lower water level below skimmer"
    - "Drain pump and filter"
    - "Blow out lines"
    - "Add winterizing chemicals"
    - "Install cover"
    
  # System state during winter
  winter_mode:
    pump_schedule: "off"
    dosing: "disabled"
    heating: "frost_protection"
    cover: "closed"
    
    # Frost protection
    frost_protection:
      enabled: true
      trigger_temp_c: 2
      run_pump_minutes: 15
```

---

## Configuration Examples

### Residential: Basic

```yaml
pool:
  id: "pool-main"
  name: "Swimming Pool"
  volume_liters: 50000
  
  equipment:
    pump: "pump-pool-main"
    filter: "filter-sand"
    heater: "heatpump-pool"
    
  chemistry:
    type: "salt_chlorinator"
    chlorinator: "chlorinator-main"
    acid_dosing: false                # Manual acid addition
    
  sensors:
    - type: "ph"
    - type: "orp"
    - type: "temperature"
    
  cover:
    type: "thermal"
    motorized: true
```

### Residential: Full Automation

```yaml
pool:
  id: "pool-main"
  volume_liters: 60000
  
  equipment:
    pump: "pump-pool-vsd"             # Variable speed
    filter: "filter-de"
    heater:
      primary: "solar-pool"
      backup: "heatpump-pool"
    cover: "cover-automatic"
    
  chemistry:
    monitoring:
      - type: "ph"
      - type: "orp"
      - type: "temperature"
      - type: "flow"
      
    dosing:
      chlorine:
        type: "liquid"
        pump: "pump-dose-chlorine"
        tank: "tank-chlorine"
        
      acid:
        type: "muriatic"
        pump: "pump-dose-acid"
        tank: "tank-acid"
        
    safety:
      interlock: true
      flow_verification: true
      max_doses:
        chlorine_ml_per_day: 3000
        acid_ml_per_day: 2000
        
  features:
    - id: "waterfall"
      pump: "pump-waterfall"
    - id: "spa"
      pump: "pump-spa-jets"
      heater: "heater-spa"
      
  automation:
    schedules: true
    solar_priority: true
    cover_automation: true
```

---

## Best Practices

### Do's

1. **Calibrate sensors regularly** — Monthly for pH/ORP
2. **Monitor chemical levels** — Never run dry
3. **Test water weekly** — Verify sensor readings
4. **Maintain interlocks** — Safety systems must work
5. **Document procedures** — Calibration, winterization
6. **Keep spare sensors** — Quick replacement

### Don'ts

1. **Don't disable interlocks** — Even temporarily
2. **Don't ignore drift** — Calibrate or replace sensors
3. **Don't overdose** — Respect limits
4. **Don't neglect maintenance** — Problems compound
5. **Don't bypass safety** — It exists for good reason

---

## Related Documents

- [Plant Domain](plant.md) — Pumps, heaters, equipment
- [Leak Protection](leak-protection.md) — Pool area leak detection
- [Energy Domain](energy.md) — Pump/heater optimization
- [Lighting Domain](lighting.md) — Pool and underwater lighting
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring
- [Automation Specification](../automation/automation.md) — Scene and mode integration
