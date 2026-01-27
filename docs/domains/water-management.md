---
title: Water Management Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - data-model/entities.md
  - domains/leak-protection.md
---

# Water Management Domain Specification

This document specifies how Gray Logic manages water infrastructure including rainwater harvesting, greywater recycling, water treatment, and usage monitoring.

For leak detection and automatic shutoff, see [Leak Protection](leak-protection.md).

---

## Overview

### Philosophy

Water management focuses on sustainability, quality, and efficiency:

| Goal | Approach |
|------|----------|
| **Conservation** | Rainwater harvesting, greywater reuse |
| **Quality** | Treatment, softening, filtration |
| **Monitoring** | Usage tracking, cost awareness |
| **Efficiency** | Optimal source selection |

### Design Principles

1. **Use alternative sources first** — Rainwater/greywater before mains
2. **Quality matching** — Right water quality for each use
3. **Continuous monitoring** — Track levels, quality, usage
4. **Fail-safe** — Mains backup when alternatives unavailable
5. **Health-first** — Never compromise potable water quality

---

## Water Sources

### Source Types

| Source | Quality | Typical Uses |
|--------|---------|--------------|
| **Mains** | Potable | Drinking, cooking, bathing |
| **Rainwater** | Non-potable (treatable) | Irrigation, toilets, laundry |
| **Greywater** | Non-potable | Irrigation, toilets |
| **Borehole** | Variable (test required) | Irrigation, or treated for domestic |
| **Recycled** | Treated greywater | Irrigation, toilets |

### Source Priority

```yaml
SourcePriority:
  # For toilet flushing
  toilet:
    1: "rainwater"
    2: "greywater"
    3: "mains"
    
  # For irrigation
  irrigation:
    1: "rainwater"
    2: "greywater"
    3: "borehole"
    4: "mains"
    
  # For laundry
  laundry:
    1: "rainwater"      # If treated
    2: "mains"
    
  # For potable (no alternatives)
  potable:
    1: "mains"
```

---

## Rainwater Harvesting

### System Components

```yaml
RainwaterSystem:
  id: uuid
  name: string                        # "Main Rainwater System"
  
  # Collection
  collection:
    roof_area_sqm: float              # Collection area
    runoff_coefficient: float         # Typically 0.8-0.9
    
  # Storage
  tank:
    capacity_liters: integer
    material: string                  # "concrete", "plastic", "steel"
    location: string                  # "underground", "above_ground"
    
  # Level monitoring
  level_sensor:
    device_id: uuid
    type: "ultrasonic" | "pressure" | "float"
    
  # Treatment (if installed)
  treatment:
    first_flush_diverter: boolean
    sediment_filter: boolean
    uv_sterilization: boolean
    
  # Pump
  pump:
    device_id: uuid
    flow_rate_lpm: float
    pressure_bar: float
    
  # Current state
  state:
    level_percent: integer
    level_liters: integer
    pump_running: boolean
    treatment_status: "ok" | "service_required"
```

### Rainwater Scheduling

```yaml
RainwaterControl:
  # Automatic source switching
  auto_switching:
    enabled: true
    
    # Use rainwater when available
    use_rainwater_above_percent: 20
    
    # Switch to mains when low
    switch_to_mains_below_percent: 10
    
    # Hysteresis to prevent cycling
    hysteresis_percent: 5
    
  # Pump control
  pump:
    # Run pump to pressurize system
    pressure_tank: true
    target_pressure_bar: 2.5
    
    # Or on-demand
    on_demand: false
    
  # Mains top-up (if connected)
  mains_topup:
    enabled: true
    trigger_below_percent: 15
    fill_to_percent: 30               # Don't overfill - leave room for rain
```

### Rain Forecasting Integration

```yaml
RainForecast:
  # Check forecast before mains top-up
  check_forecast: true
  
  # Skip top-up if rain expected
  skip_topup_if:
    rain_probability_above: 60
    rain_amount_above_mm: 10
    forecast_hours: 48
    
  # Estimate collection
  estimate_collection:
    roof_area_sqm: 150
    runoff_coefficient: 0.85
    # Expected collection = rain_mm × area × coefficient / 1000 = liters
```

---

## Greywater Recycling

### Greywater Sources

| Source | Quality | Volume |
|--------|---------|--------|
| Shower/bath | Low contamination | High |
| Bathroom sinks | Low contamination | Medium |
| Laundry | Detergent residue | High |
| Kitchen sink | High contamination | **Excluded** |
| Dishwasher | High contamination | **Excluded** |

### Greywater System

```yaml
GreywaterSystem:
  id: uuid
  name: string
  
  # Collection sources
  sources:
    - "shower-master"
    - "shower-ensuite"
    - "bath-main"
    - "sink-bathroom-1"
    - "sink-bathroom-2"
    
  # Treatment
  treatment:
    stages:
      - "settling_tank"
      - "sand_filter"
      - "uv_sterilization"
    retention_time_hours: 24          # Max before discharge
    
  # Storage
  tank:
    capacity_liters: 500
    
  # Level monitoring
  level_sensor:
    device_id: uuid
    
  # Distribution
  uses:
    - "toilet_flushing"
    - "irrigation"
    
  # Current state
  state:
    level_percent: integer
    treatment_ok: boolean
    last_discharge: timestamp | null
```

### Greywater Safety

```yaml
GreywaterSafety:
  # Maximum retention time
  max_retention_hours: 24
  
  # Auto-discharge if not used
  auto_discharge:
    after_hours: 24
    to: "sewer"                       # or "irrigation" if safe
    
  # Treatment verification
  treatment:
    uv_lamp_hours_max: 9000
    alert_at_percent: 80
    
  # Never use for
  prohibited_uses:
    - "drinking"
    - "cooking"
    - "bathing"
    - "vegetable_garden"              # Edible crops
```

---

## Water Treatment

### Water Softening

```yaml
WaterSoftener:
  id: uuid
  name: string                        # "Main Water Softener"
  
  # Hardware
  manufacturer: string
  model: string
  
  # Capacity
  capacity_liters: integer            # Between regenerations
  
  # Salt monitoring
  salt:
    tank_capacity_kg: float
    level_sensor: uuid | null
    
  # Regeneration
  regeneration:
    trigger: "volume" | "time" | "hardness"
    volume_liters: integer | null
    interval_days: integer | null
    preferred_time: "02:00"           # Off-peak
    
  # Monitoring
  connection:
    type: "ip" | "modbus" | "dry_contact"
    
  # Current state
  state:
    remaining_capacity_liters: integer
    salt_level_percent: integer
    last_regeneration: timestamp
    next_regeneration: timestamp
    hardness_in_ppm: integer | null
    hardness_out_ppm: integer | null
```

### UV Sterilization

```yaml
UVSterilizer:
  id: uuid
  name: string
  
  # Location in system
  treats: "rainwater" | "borehole" | "mains"
  
  # Hardware
  lamp_life_hours: integer            # Rated life
  
  # Monitoring
  sensors:
    uv_intensity: uuid | null         # UV sensor
    flow_rate: uuid | null
    
  # Current state
  state:
    lamp_hours: integer
    lamp_ok: boolean
    uv_intensity_percent: integer | null
```

### Filtration

```yaml
FiltrationSystem:
  id: uuid
  name: string
  
  stages:
    - type: "sediment"
      micron: 20
      change_interval_months: 6
      
    - type: "carbon"
      micron: 5
      change_interval_months: 6
      
    - type: "reverse_osmosis"
      membrane_life_years: 3
      
  # Monitoring
  pressure_sensors:
    pre_filter: uuid
    post_filter: uuid
    
  # Current state
  state:
    pressure_drop_bar: float
    filter_status: "ok" | "replace_soon" | "replace_now"
    last_service: timestamp
```

---

## Water Monitoring

### Usage Tracking

```yaml
WaterUsage:
  meters:
    - id: "meter-mains"
      location: "Main supply"
      type: "mains"
      
    - id: "meter-rainwater"
      location: "Rainwater pump outlet"
      type: "rainwater"
      
    - id: "meter-irrigation"
      location: "Irrigation supply"
      type: "irrigation"
      
  tracking:
    resolution: "hourly"
    
  current_day:
    mains_liters: 245
    rainwater_liters: 180
    total_liters: 425
    
  current_month:
    mains_liters: 8500
    mains_cost: 25.50
    rainwater_liters: 4200
    savings_liters: 4200
    savings_cost: 12.60
```

### Water Quality Monitoring

```yaml
WaterQuality:
  sensors:
    - id: "quality-mains"
      location: "Mains inlet"
      parameters:
        - "tds"                       # Total dissolved solids
        - "ph"
        - "temperature"
        
    - id: "quality-rainwater"
      location: "Rainwater tank outlet"
      parameters:
        - "turbidity"
        - "ph"
        
  alerts:
    - parameter: "tds"
      threshold_above: 500            # ppm
      alert: "High TDS - check source"
      
    - parameter: "ph"
      threshold_below: 6.5
      threshold_above: 8.5
      alert: "pH out of range"
```

---

## Automation Integration

### Mode Integration

```yaml
modes:
  - id: "away"
    behaviours:
      water:
        rainwater_pump: "off"         # No demand while away
        greywater_discharge: true     # Discharge to prevent stagnation
        
  - id: "vacation"
    behaviours:
      water:
        rainwater_pump: "off"
        greywater_discharge: true
        mains_shutoff: true           # Optional - close mains while away
        
  - id: "eco"
    behaviours:
      water:
        prefer_alternatives: true     # Maximize rainwater/greywater use
```

### Event Triggers

```yaml
triggers:
  # Rainwater tank full → Use for irrigation
  - type: "level_above"
    source:
      device_id: "tank-rainwater"
      threshold_percent: 90
    execute:
      - domain: "irrigation"
        action: "run_extra_cycle"
        parameters:
          source: "rainwater"
          
  # Rainwater low → Alert
  - type: "level_below"
    source:
      device_id: "tank-rainwater"
      threshold_percent: 15
    execute:
      - domain: "notification"
        action: "send"
        parameters:
          message: "Rainwater tank low (${level}%)"
          
  # Salt low → Alert
  - type: "level_below"
    source:
      device_id: "softener-salt"
      threshold_percent: 20
    execute:
      - domain: "notification"
        action: "send"
        parameters:
          message: "Water softener salt low - please refill"
```

---

## MQTT Topics

### Tank State

```yaml
topic: graylogic/water/tank/{tank_id}/state
payload:
  tank_id: "tank-rainwater"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    level_percent: 65
    level_liters: 3250
    temperature_c: 12
```

### Treatment State

```yaml
topic: graylogic/water/treatment/{device_id}/state
payload:
  device_id: "softener-main"
  timestamp: "2026-01-12T14:30:00Z"
  state:
    remaining_capacity_liters: 1500
    salt_level_percent: 45
    regeneration_due: "2026-01-15T02:00:00Z"
```

### Usage

```yaml
topic: graylogic/water/usage/daily
payload:
  date: "2026-01-12"
  mains_liters: 245
  rainwater_liters: 180
  greywater_liters: 50
  total_liters: 475
```

---

## PHM Integration

### PHM Value for Water Equipment

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Pumps | ★★★★☆ | Current, pressure, runtime |
| UV lamps | ★★★★☆ | Hours, intensity |
| Filters | ★★★☆☆ | Pressure drop |
| Softener | ★★★☆☆ | Regeneration frequency |
| Level sensors | ★★☆☆☆ | Accuracy, drift |

### PHM Configuration

```yaml
phm_water:
  devices:
    - device_id: "pump-rainwater"
      type: "pump"
      parameters:
        - name: "current_a"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 25
          alert: "Pump current deviation"
          
        - name: "pressure_bar"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 20
          alert: "Pump pressure deviation"
          
    - device_id: "uv-rainwater"
      type: "uv_sterilizer"
      parameters:
        - name: "lamp_hours"
          threshold: 9000
          alert_at_percent: 80
          alert: "UV lamp approaching end of life"
          
        - name: "intensity_percent"
          threshold_below: 70
          alert: "UV intensity low - lamp may need replacement"
          
    - device_id: "softener-main"
      type: "water_softener"
      parameters:
        - name: "regeneration_frequency"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 30
          alert: "Abnormal regeneration frequency - check settings or water hardness"
```

---

## Configuration Examples

### Residential: Rainwater Only

```yaml
water_management:
  rainwater:
    tank:
      capacity_liters: 5000
      level_sensor: "sensor-tank-level"
      
    pump:
      device_id: "pump-rainwater"
      pressure_bar: 2.5
      
    uses:
      - toilet_flushing: true
      - irrigation: true
      - laundry: false              # Not treated
      
    mains_backup:
      enabled: true
      switch_below_percent: 10
```

### Residential: Full System

```yaml
water_management:
  rainwater:
    tank_liters: 10000
    treatment:
      first_flush: true
      sediment_filter: true
      uv_sterilization: true
    uses: ["toilet", "laundry", "irrigation"]
    
  greywater:
    sources: ["showers", "baths", "bathroom_sinks"]
    treatment: ["settling", "filter", "uv"]
    tank_liters: 500
    uses: ["toilet", "irrigation"]
    
  softener:
    enabled: true
    capacity_liters: 3000
    salt_monitoring: true
    
  monitoring:
    mains_meter: "meter-mains"
    rainwater_meter: "meter-rainwater"
    quality_sensors: ["tds-mains", "ph-rainwater"]
```

### Commercial: Multi-Source

```yaml
water_management:
  sources:
    - id: "mains"
      type: "mains"
      meter: "meter-mains"
      
    - id: "rainwater"
      type: "rainwater"
      tank_liters: 50000
      meter: "meter-rainwater"
      
    - id: "borehole"
      type: "borehole"
      treatment: ["sediment", "uv", "softening"]
      meter: "meter-borehole"
      
  priority:
    irrigation: ["rainwater", "borehole", "mains"]
    toilets: ["rainwater", "mains"]
    cooling_towers: ["borehole", "mains"]
    potable: ["mains"]
    
  monitoring:
    usage_by_area: true
    cost_allocation: true
    quality_logging: true
```

---

## Best Practices

### Do's

1. **Test water quality** — Especially borehole and treated rainwater
2. **Regular maintenance** — UV lamps, filters, softener salt
3. **Monitor levels** — Don't run pumps dry
4. **Discharge greywater** — Don't let it stagnate
5. **Label systems** — Clear marking of non-potable supplies
6. **Comply with regulations** — Local codes for greywater/rainwater

### Don'ts

1. **Don't cross-connect** — Keep potable and non-potable separate
2. **Don't skip treatment** — Required for certain uses
3. **Don't ignore maintenance** — Failing treatment is worse than none
4. **Don't overfill rainwater** — Leave capacity for storms
5. **Don't use untreated greywater on edibles** — Health risk

---

## Related Documents

- [Leak Protection](leak-protection.md) — Leak detection and shutoff
- [Irrigation](irrigation.md) — Garden watering
- [Pool Domain](pool.md) — Pool water management
- [Plant Domain](plant.md) — Pump and treatment equipment
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring
- [Energy Domain](energy.md) — Pump energy optimization
