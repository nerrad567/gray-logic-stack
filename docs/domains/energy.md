---
title: Energy Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/energy-model.md
  - architecture/core-internals.md
  - data-model/entities.md
  - protocols/modbus.md
---

# Energy Domain Specification

This document specifies how Gray Logic presents energy management to users, integrates with automation, and provides insights into energy consumption, generation, and costs.

For the technical energy flow model, metering configuration, and protocol details, see [Energy Model](../architecture/energy-model.md).

---

## Overview

### Philosophy

Energy management in Gray Logic focuses on:

| Goal | Approach |
|------|----------|
| **Visibility** | Know where energy goes, in real-time and historically |
| **Optimization** | Use energy when it's cheapest or greenest |
| **Control** | Manage loads, storage, and generation intelligently |
| **Automation** | Energy-aware scenes, modes, and schedules |

### Design Principles

1. **Measure first** — You can't manage what you don't measure
2. **User control** — Automation suggests, user decides
3. **Cost-aware** — Factor in tariffs, time-of-use, and demand charges
4. **Green-aware** — Prefer self-generated and low-carbon energy
5. **Comfort preserved** — Energy savings shouldn't compromise comfort unless requested

### Energy Hierarchy

```
                    ┌─────────────────┐
                    │  User Comfort   │  ← Highest priority
                    ├─────────────────┤
                    │  Safety/Health  │
                    ├─────────────────┤
                    │  Cost Savings   │
                    ├─────────────────┤
                    │  Carbon Reduction│  ← Lowest priority
                    └─────────────────┘
```

Users can adjust this hierarchy via Energy Mode settings.

---

## Energy Sources & Loads

### Source Types

| Type | Direction | Examples |
|------|-----------|----------|
| **Grid** | Import/Export | Mains connection |
| **Solar** | Generation | Roof PV, solar thermal |
| **Battery** | Charge/Discharge | Home battery, PowerWall |
| **EV** | Charge (V2G: discharge) | EV charger, vehicle battery |
| **Generator** | Generation | Backup generator |
| **Wind** | Generation | Small wind turbine |

### Load Categories

| Category | Priority | Examples | Deferrable |
|----------|----------|----------|------------|
| **Critical** | 1 | Medical equipment, security | No |
| **Essential** | 2 | Refrigeration, heating (min) | No |
| **Comfort** | 3 | HVAC, hot water, lighting | Partially |
| **Flexible** | 4 | EV charging, pool pump, laundry | Yes |
| **Discretionary** | 5 | Entertainment, spa, sauna | Yes |

---

## User Interface

### Energy Dashboard

Real-time overview of energy flows:

```yaml
EnergyDashboard:
  timestamp: "2026-01-12T14:30:00Z"
  
  # Current power flows (watts)
  flows:
    grid:
      power_w: -1200                  # Negative = exporting
      direction: "export"
    solar:
      power_w: 4500
      generation_today_kwh: 28.5
    battery:
      power_w: 800                    # Positive = charging
      soc_percent: 65
      direction: "charging"
    home:
      power_w: 2500                   # Total consumption
      
  # Today's summary
  today:
    consumed_kwh: 18.4
    generated_kwh: 28.5
    imported_kwh: 2.1
    exported_kwh: 12.2
    self_consumption_percent: 92
    cost_estimate: 3.42
    
  # Current tariff
  tariff:
    name: "Octopus Agile"
    current_rate_kwh: 0.15
    next_rate_kwh: 0.28
    next_change: "16:00"
    period: "off-peak"
```

### Consumption Breakdown

```yaml
ConsumptionBreakdown:
  period: "today"
  total_kwh: 18.4
  
  by_category:
    - category: "HVAC"
      kwh: 8.2
      percent: 44.6
      cost: 1.23
      
    - category: "Hot Water"
      kwh: 3.5
      percent: 19.0
      cost: 0.53
      
    - category: "Lighting"
      kwh: 1.8
      percent: 9.8
      cost: 0.27
      
    - category: "EV Charging"
      kwh: 2.4
      percent: 13.0
      cost: 0.36
      
    - category: "Other"
      kwh: 2.5
      percent: 13.6
      cost: 0.38
      
  by_room:
    - room: "Living Room"
      kwh: 4.2
    - room: "Kitchen"
      kwh: 3.8
    - room: "Master Bedroom"
      kwh: 1.2
```

### Historical Analysis

```yaml
EnergyHistory:
  period: "month"
  start: "2026-01-01"
  end: "2026-01-31"
  
  summary:
    consumed_kwh: 485
    generated_kwh: 320
    imported_kwh: 210
    exported_kwh: 45
    net_kwh: 165                      # Net import
    total_cost: 72.50
    
  comparison:
    previous_period_kwh: 520
    change_percent: -6.7
    
  daily_breakdown:
    - date: "2026-01-01"
      consumed: 15.2
      generated: 8.5
      cost: 2.85
    # ... more days
    
  trends:
    - insight: "Heating accounts for 45% of consumption"
    - insight: "Weekend usage 20% higher than weekdays"
    - insight: "Best solar day: Jan 15 (32 kWh)"
```

---

## Tariff Management

### Tariff Configuration

```yaml
Tariff:
  id: "tariff-main"
  name: "Octopus Agile"
  provider: "Octopus Energy"
  type: "time_of_use"                 # flat, time_of_use, agile, demand
  
  # For time-of-use tariffs
  rates:
    - name: "Off-Peak"
      rate_kwh: 0.075
      periods:
        - days: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
          start: "00:30"
          end: "04:30"
          
    - name: "Peak"
      rate_kwh: 0.35
      periods:
        - days: ["mon", "tue", "wed", "thu", "fri"]
          start: "16:00"
          end: "19:00"
          
    - name: "Standard"
      rate_kwh: 0.22
      periods:
        - default: true               # All other times
        
  # Standing charge
  standing_charge_day: 0.42
  
  # Export rate
  export:
    type: "fixed"                     # fixed, agile, seg
    rate_kwh: 0.15
```

### Agile Tariff Integration

For variable-rate tariffs like Octopus Agile:

```yaml
AgileTariff:
  provider: "octopus"
  api:
    product_code: "AGILE-FLEX-22-11-25"
    tariff_code: "E-1R-AGILE-FLEX-22-11-25-A"
    region: "A"                       # DNO region
    
  # Fetched rates (30-min periods)
  rates:
    - start: "2026-01-12T00:00:00Z"
      end: "2026-01-12T00:30:00Z"
      rate_kwh: 0.12
    - start: "2026-01-12T00:30:00Z"
      end: "2026-01-12T01:00:00Z"
      rate_kwh: 0.08
    # ... 48 periods per day
    
  # Optimization windows
  cheapest_periods:
    - start: "02:30"
      end: "05:00"
      average_rate: 0.07
  most_expensive_periods:
    - start: "17:00"
      end: "19:00"
      average_rate: 0.38
```

---

## Load Management

### Load Prioritization

When grid capacity is limited or prices are high:

```yaml
LoadPriority:
  # Priority 1: Never shed
  critical:
    - device_id: "fridge-kitchen"
    - device_id: "freezer-garage"
    - device_id: "medical-cpap"
    - device_id: "alarm-panel"
    
  # Priority 2: Shed only in emergency
  essential:
    - device_id: "heatpump-main"
      min_setpoint: 16                # Reduce but don't turn off
    - device_id: "hot-water-cylinder"
      
  # Priority 3: Shed during peak pricing
  comfort:
    - device_id: "hvac-zone-living"
      allow_setback: 2                # Allow 2°C setback
    - device_id: "lighting-garden"
      
  # Priority 4: Defer to cheap periods
  flexible:
    - device_id: "ev-charger"
      defer_until: "cheapest_4h"
    - device_id: "pool-pump"
      run_window: ["00:00", "06:00"]
    - device_id: "dishwasher"
      delay_start: true
      
  # Priority 5: Shed first
  discretionary:
    - device_id: "hot-tub"
    - device_id: "sauna"
    - device_id: "towel-rail-bathroom"
```

### Demand Response

Respond to grid signals or price spikes:

```yaml
DemandResponse:
  enabled: true
  
  triggers:
    # Price-based
    - condition: "price_above"
      threshold_kwh: 0.30
      action: "shed_discretionary"
      
    - condition: "price_above"
      threshold_kwh: 0.50
      action: "shed_comfort"
      notify: true
      
    # Grid signal (future)
    - condition: "grid_signal"
      signal: "reduce_demand"
      action: "shed_flexible"
      
  # Override protection
  override:
    allow_user_override: true
    override_duration_max_hours: 4
```

### Smart Scheduling

Schedule flexible loads for optimal times:

```yaml
SmartSchedule:
  device_id: "ev-charger"
  
  goal:
    type: "charge_by"
    target_soc: 80
    deadline: "07:00"
    
  constraints:
    min_rate_a: 6                     # Minimum 6A (1.4kW)
    max_rate_a: 32                    # Maximum 32A (7.4kW)
    
  optimization:
    strategy: "cheapest"              # cheapest, greenest, balanced
    use_solar_forecast: true
    use_battery: false                # Don't discharge battery for EV
    
  result:
    scheduled_periods:
      - start: "01:00"
        end: "05:00"
        rate_a: 32
        estimated_cost: 2.45
        estimated_kwh: 29.6
```

---

## Solar & Battery Optimization

### Self-Consumption Strategy

Maximize use of self-generated energy:

```yaml
SelfConsumptionStrategy:
  enabled: true
  
  priorities:
    1: "power_home_loads"             # Use solar for current consumption
    2: "charge_battery"               # Store excess in battery
    3: "charge_ev"                    # Charge EV if plugged in
    4: "heat_water"                   # Divert to hot water
    5: "export_grid"                  # Export remainder
    
  battery:
    reserve_percent: 20               # Keep 20% for evening
    charge_from_grid: false           # Only charge from solar
    winter_charge_from_grid: true     # Allow grid charging in winter
    
  ev:
    solar_only_mode: true             # Only charge from excess solar
    min_solar_w: 1400                 # Minimum solar before EV charging
```

### Battery Arbitrage

Buy cheap, use during expensive periods:

```yaml
BatteryArbitrage:
  enabled: true
  
  strategy:
    charge_periods:
      - time: "02:00-05:00"
        max_rate_w: 5000
        condition: "price_below"
        threshold_kwh: 0.10
        
    discharge_periods:
      - time: "16:00-19:00"
        condition: "price_above"
        threshold_kwh: 0.25
        reserve_percent: 20           # Don't discharge below 20%
        
  constraints:
    max_cycles_per_day: 1             # Preserve battery life
    prefer_solar_charge: true         # Solar priority over grid
```

### Solar Diverter Control

Divert excess solar to immersion heater:

```yaml
SolarDiverter:
  device_id: "immersion-diverter"
  
  config:
    target_device: "immersion-heater"
    threshold_w: 500                  # Start diverting at 500W export
    max_power_w: 3000                 # Immersion max power
    
  conditions:
    - tank_temperature_below: 60      # Only if tank needs heating
    - export_power_above: 500         # Only if actually exporting
    
  priority: 4                         # After battery, before grid export
```

---

## Automation Integration

### Energy Modes

```yaml
EnergyModes:
  - id: "normal"
    name: "Normal"
    description: "Balanced comfort and cost"
    settings:
      comfort_priority: 0.6
      cost_priority: 0.4
      
  - id: "eco"
    name: "Eco"
    description: "Maximize savings"
    settings:
      comfort_priority: 0.3
      cost_priority: 0.7
      hvac_setback: 2                 # °C setback from normal
      lighting_max_percent: 80
      hot_water_setpoint: 50          # Lower hot water temp
      
  - id: "comfort"
    name: "Comfort"
    description: "Prioritize comfort"
    settings:
      comfort_priority: 0.9
      cost_priority: 0.1
      
  - id: "away"
    name: "Away"
    description: "Minimal energy use"
    settings:
      hvac_setback: 5
      hot_water_off: true
      lighting_off: true
      shed_discretionary: true
```

### Mode Integration

Energy behavior tied to Gray Logic modes:

```yaml
modes:
  - id: "home"
    behaviours:
      energy:
        mode: "normal"
        
  - id: "away"
    behaviours:
      energy:
        mode: "away"
        ev_charging: "pause"          # Don't charge while away
        
  - id: "night"
    behaviours:
      energy:
        mode: "eco"
        ev_charging: "smart"          # Charge overnight
        battery: "charge"             # Charge battery off-peak
        
  - id: "vacation"
    behaviours:
      energy:
        mode: "away"
        hot_water: "off"
        hvac: "frost_protection"
```

### Energy-Aware Scenes

```yaml
scenes:
  - id: "scene-movie-night"
    name: "Movie Night"
    actions:
      # Normal scene actions
      - domain: "lighting"
        action: "dim"
        parameters:
          level: 20
      - domain: "blinds"
        action: "close"
        
      # Energy-aware actions
      - domain: "energy"
        action: "set_mode"
        parameters:
          mode: "comfort"             # Prioritize comfort during movie
          duration_hours: 3           # Revert after 3 hours

  - id: "scene-leaving-home"
    name: "Leaving Home"
    actions:
      - domain: "energy"
        action: "set_mode"
        parameters:
          mode: "away"
      - domain: "energy"
        action: "shed_loads"
        parameters:
          categories: ["discretionary", "flexible"]
```

### Event-Driven Energy

```yaml
triggers:
  # Price spike → Reduce consumption
  - type: "energy_price_change"
    condition:
      price_kwh_above: 0.35
    execute:
      - domain: "energy"
        action: "shed_loads"
        parameters:
          categories: ["discretionary"]
      - domain: "notification"
        action: "send"
        parameters:
          message: "High electricity price - reducing non-essential loads"

  # Solar generation high → Start flexible loads
  - type: "solar_generation"
    condition:
      export_w_above: 2000
      duration_minutes: 10
    execute:
      - device_id: "ev-charger"
        action: "start_solar_charging"
      - device_id: "pool-pump"
        action: "on"

  # Battery low + no solar → Alert
  - type: "battery_state"
    condition:
      soc_below: 20
      solar_w_below: 100
    execute:
      - domain: "notification"
        action: "send"
        parameters:
          message: "Battery low, switching to grid power"
```

---

## Reporting & Insights

### Daily Report

```yaml
DailyEnergyReport:
  date: "2026-01-12"
  
  summary:
    consumed_kwh: 18.4
    generated_kwh: 28.5
    self_consumption_percent: 92
    total_cost: 3.42
    co2_kg: 2.1
    
  highlights:
    - "Generated 155% of consumption"
    - "Exported 12.2 kWh to grid"
    - "EV charged 100% from solar"
    
  comparison:
    yesterday: { consumed: 19.2, cost: 4.15 }
    last_week_avg: { consumed: 17.8, cost: 3.85 }
    
  recommendations:
    - "Consider running dishwasher at 2pm when solar peaks"
    - "Hot water heated to 65°C - could reduce to 60°C"
```

### Monthly Report

```yaml
MonthlyEnergyReport:
  month: "2026-01"
  
  summary:
    consumed_kwh: 485
    generated_kwh: 320
    imported_kwh: 210
    exported_kwh: 45
    total_cost: 72.50
    total_export_income: 6.75
    net_cost: 65.75
    
  by_category:
    hvac: { kwh: 218, cost: 32.70, percent: 45 }
    hot_water: { kwh: 87, cost: 13.05, percent: 18 }
    ev_charging: { kwh: 72, cost: 5.40, percent: 15 }
    lighting: { kwh: 34, cost: 5.10, percent: 7 }
    other: { kwh: 74, cost: 11.10, percent: 15 }
    
  trends:
    consumption_vs_last_month: -8.2   # % change
    cost_vs_last_month: -12.5
    self_consumption_trend: "improving"
    
  goals:
    target_kwh: 500
    actual_kwh: 485
    status: "on_track"
```

### Carbon Tracking

```yaml
CarbonReport:
  period: "month"
  
  emissions:
    total_kg_co2: 62.5
    grid_kg_co2: 84.0                 # Would have been without solar
    avoided_kg_co2: 21.5              # Saved by solar
    
  grid_intensity:
    average_g_co2_kwh: 200
    when_imported_g_co2_kwh: 180      # Imported during cleaner periods
    
  comparison:
    national_average_kg: 125
    your_emissions_kg: 62.5
    better_than_average_percent: 50
```

---

## EV Charging

### Charging Modes

```yaml
EVChargingModes:
  - id: "immediate"
    name: "Charge Now"
    description: "Start charging immediately at max rate"
    
  - id: "scheduled"
    name: "Scheduled"
    description: "Charge during specified time window"
    parameters:
      start_time: "23:00"
      end_time: "07:00"
      
  - id: "smart"
    name: "Smart"
    description: "Optimize for cheapest/greenest charging"
    parameters:
      ready_by: "07:00"
      target_soc: 80
      strategy: "cheapest"            # cheapest, greenest, balanced
      
  - id: "solar"
    name: "Solar Only"
    description: "Only charge from excess solar"
    parameters:
      min_solar_w: 1400
      
  - id: "paused"
    name: "Paused"
    description: "Don't charge until mode changed"
```

### Charging Session

```yaml
ChargingSession:
  charger_id: "ev-charger-garage"
  vehicle_id: "tesla-model-3"
  
  status: "charging"
  
  session:
    started_at: "2026-01-12T01:00:00Z"
    energy_kwh: 15.4
    duration_minutes: 120
    average_power_kw: 7.2
    cost_so_far: 1.15
    
  vehicle:
    soc_percent: 65
    target_soc_percent: 80
    range_km: 280
    
  schedule:
    mode: "smart"
    ready_by: "07:00"
    estimated_completion: "05:30"
    remaining_kwh: 12.0
    estimated_cost: 0.90
```

---

## Commercial Energy

### Demand Management

Commercial tariffs often include demand charges:

```yaml
DemandManagement:
  tariff:
    demand_charge_kw: 12.50           # £/kW/month
    measurement_period_minutes: 30
    
  current:
    peak_demand_kw: 45.2
    peak_time: "2026-01-12T09:15:00Z"
    
  limits:
    target_max_kw: 50
    absolute_max_kw: 60
    
  alerts:
    - threshold_kw: 45
      action: "warn"
    - threshold_kw: 50
      action: "shed_flexible"
    - threshold_kw: 55
      action: "shed_comfort"
```

### Multi-Tenant Allocation

For buildings with multiple tenants:

```yaml
TenantAllocation:
  building_id: "building-main"
  
  common_areas:
    meters: ["meter-common-1", "meter-common-2"]
    allocation: "by_floor_area"       # or "equal", "by_headcount"
    
  tenants:
    - tenant_id: "tenant-acme"
      name: "Acme Corp"
      floor_area_sqm: 500
      meters: ["meter-tenant-acme"]
      
    - tenant_id: "tenant-globex"
      name: "Globex Inc"
      floor_area_sqm: 300
      meters: ["meter-tenant-globex"]
      
  billing:
    common_area_split:
      tenant-acme: 62.5               # % of common area cost
      tenant-globex: 37.5
```

### Energy Compliance

For reporting requirements:

```yaml
ComplianceReporting:
  # UK ESOS (Energy Savings Opportunity Scheme)
  esos:
    enabled: true
    qualification_threshold_kwh: 6000000
    
  # Display Energy Certificate
  dec:
    enabled: true
    building_type: "office"
    floor_area_sqm: 2500
    
  # NABERS (Australia)
  nabers:
    enabled: false
    
  exports:
    - format: "csv"
      schedule: "monthly"
      destination: "reports/energy/"
    - format: "api"
      endpoint: "https://compliance.example.com/submit"
```

---

## PHM Integration

### PHM Value for Energy Equipment

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Solar inverter | ★★★★☆ | Efficiency degradation, error codes |
| Battery | ★★★★★ | Capacity fade, cycle count, temperature |
| Heat pump | ★★★★☆ | COP trending, defrost frequency |
| EV charger | ★★★☆☆ | Session failures, power deviation |
| Energy meters | ★★☆☆☆ | Communication errors |

### PHM Configuration

```yaml
phm_energy:
  devices:
    - device_id: "solar-inverter-1"
      type: "solar_inverter"
      parameters:
        - name: "efficiency"
          calculation: "ac_power / dc_power"
          baseline_method: "seasonal_context"
          context_parameter: "irradiance"
          deviation_threshold_percent: 10
          alert: "Inverter efficiency degradation"
          
        - name: "error_count"
          baseline_method: "daily_count"
          threshold_per_day: 5
          alert: "Excessive inverter errors"
          
    - device_id: "battery-main"
      type: "home_battery"
      parameters:
        - name: "capacity_kwh"
          baseline_method: "initial_calibration"
          deviation_threshold_percent: 20
          alert: "Battery capacity degradation"
          
        - name: "round_trip_efficiency"
          calculation: "energy_out / energy_in"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 10
          alert: "Battery efficiency declining"
          
        - name: "temperature_c"
          threshold_high: 45
          threshold_critical: 55
          alert: "Battery temperature elevated"
          
    - device_id: "ev-charger-garage"
      type: "ev_charger"
      parameters:
        - name: "session_failure_rate"
          calculation: "failed_sessions / total_sessions"
          baseline_method: "rolling_mean"
          deviation_threshold_percent: 200
          alert: "EV charger session failures increasing"
```

---

## MQTT Topics

### Energy State

```yaml
# Overall energy state
topic: graylogic/energy/state
payload:
  timestamp: "2026-01-12T14:30:00Z"
  flows:
    grid_w: -1200
    solar_w: 4500
    battery_w: 800
    home_w: 2500
  battery:
    soc_percent: 65
    state: "charging"
  tariff:
    current_rate_kwh: 0.15
    period: "off-peak"

# Device-specific
topic: graylogic/energy/device/{device_id}/state
payload:
  device_id: "solar-inverter-1"
  timestamp: "2026-01-12T14:30:00Z"
  power_w: 4500
  energy_today_kwh: 28.5
  status: "generating"
```

### Commands

```yaml
# Set EV charging mode
topic: graylogic/energy/command
payload:
  command: "set_ev_mode"
  parameters:
    charger_id: "ev-charger-garage"
    mode: "smart"
    ready_by: "07:00"
    target_soc: 80

# Override load shedding
topic: graylogic/energy/command
payload:
  command: "override_shedding"
  parameters:
    device_id: "hot-tub"
    duration_hours: 2
```

---

## Configuration Examples

### Residential: Solar + Battery

```yaml
energy:
  sources:
    - id: "grid-main"
      type: "grid"
      tariff: "octopus-agile"
      
    - id: "solar-roof"
      type: "solar"
      capacity_kw: 6.5
      
    - id: "battery-main"
      type: "battery"
      capacity_kwh: 13.5
      
  loads:
    critical: ["fridge", "freezer", "alarm"]
    flexible: ["ev-charger", "pool-pump", "immersion"]
    
  optimization:
    self_consumption: true
    battery_arbitrage: true
    ev_smart_charging: true
    
  modes:
    default: "normal"
```

### Residential: EV Only

```yaml
energy:
  sources:
    - id: "grid-main"
      type: "grid"
      tariff: "octopus-go"            # EV tariff with off-peak
      
  ev_chargers:
    - id: "ev-charger-driveway"
      default_mode: "smart"
      ready_by: "07:00"
      off_peak_window: ["00:30", "04:30"]
```

### Commercial: Demand Management

```yaml
energy:
  sources:
    - id: "grid-main"
      type: "grid"
      tariff:
        type: "commercial"
        unit_rate_kwh: 0.18
        demand_charge_kw: 15.00
        
    - id: "solar-roof"
      type: "solar"
      capacity_kw: 50
      
  demand_management:
    enabled: true
    target_max_kw: 100
    alerts: [80, 90, 95]              # % of target
    
  schedules:
    hvac_precool:
      enabled: true
      start: "06:00"                  # Before demand window
      
  reporting:
    monthly_report: true
    tenant_allocation: true
```

---

## Best Practices

### Do's

1. **Install sub-metering** — More data = better optimization
2. **Configure tariff correctly** — Wrong tariff = wrong decisions
3. **Set realistic goals** — Battery can't power everything forever
4. **Review reports regularly** — Catch issues early
5. **Use smart charging** — EV is biggest flexible load
6. **Maintain equipment** — Dirty solar panels = lost generation

### Don'ts

1. **Don't over-optimize** — Comfort matters
2. **Don't ignore maintenance** — Degraded equipment wastes energy
3. **Don't forget backup** — What happens when grid fails?
4. **Don't trust forecasts blindly** — Weather is unpredictable
5. **Don't cycle battery excessively** — Reduces lifespan

---

## Related Documents

- [Energy Model](../architecture/energy-model.md) — Technical energy flow model
- [Modbus Protocol](../protocols/modbus.md) — Meter/inverter communication
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring
- [Climate Domain](climate.md) — HVAC energy coordination
- [Automation Specification](../automation/automation.md) — Energy-aware automation
- [Data Model: Entities](../data-model/entities.md) — Energy device types
