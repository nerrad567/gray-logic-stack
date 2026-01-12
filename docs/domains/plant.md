---
title: Plant Domain Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - data-model/entities.md
  - architecture/bridge-interface.md
  - protocols/modbus.md
  - protocols/bacnet.md
---

# Plant Domain Specification

This document defines how Gray Logic manages mechanical plant equipment — pumps, air handling units, chillers, VFDs, and associated sensors/actuators common in plant rooms, pool facilities, and commercial buildings.

---

## Overview

### What is Plant?

The plant domain covers mechanical equipment that:
- Has moving parts (pumps, fans, compressors)
- Requires run-time management (sequencing, staging)
- Benefits from Predictive Health Monitoring (PHM)
- Often communicates via Modbus or BACnet
- May have complex fault/alarm states

### Scope

| Included | Not Included |
|----------|--------------|
| Pumps (all types) | Life safety systems (fire pumps — monitoring only) |
| Air handling units | Certified emergency equipment |
| Chillers and cooling towers | Medical/laboratory equipment |
| Heat pumps and boilers | Industrial process control |
| VFDs and soft starters | — |
| Fans and extract systems | — |
| Water treatment (pool, softeners) | — |
| Generators and UPS | — |

### Key Principles

1. **Safety First**: Gray Logic can enable/disable plant, but physical interlocks remain independent
2. **Graceful Degradation**: Equipment continues in safe mode if communication lost
3. **PHM Foundation**: Plant equipment is primary target for predictive health monitoring
4. **Energy Awareness**: Run-time optimization for efficiency
5. **Alarm Management**: Structured approach to faults and notifications

---

## Device Types

### Pumps

| Type | Use Cases | Typical Control |
|------|-----------|-----------------|
| `pump` (circulation) | Heating circuits, pool circulation, chilled water | On/off, VFD speed |
| `pump` (booster) | Pressure boosting, irrigation | On/off, pressure setpoint |
| `pump` (dosing) | Pool chemistry, water treatment | On/off, flow setpoint |
| `pump` (submersible) | Sump, drainage, rainwater | Level-based on/off |

**State Model:**
```yaml
PumpState:
  running: boolean              # Currently running
  speed_percent: float | null   # 0-100 if VFD controlled
  speed_hz: float | null        # Actual frequency if VFD
  run_hours: integer            # Cumulative run hours
  fault: boolean                # Fault condition active
  fault_code: string | null     # Specific fault identifier
  
  # Optional sensor readings
  flow_rate: float | null       # L/min or m³/h
  inlet_pressure: float | null  # bar
  outlet_pressure: float | null # bar
  differential_pressure: float | null
  power: float | null           # kW
  current: float | null         # Amps
  
  # PHM data
  vibration_level: float | null # mm/s RMS
  bearing_temp: float | null    # °C
```

**Capabilities:**
- `run_stop` — Start/stop pump
- `speed_control` — Set speed (0-100% or Hz)
- `speed_read` — Read actual speed
- `run_hours` — Read cumulative runtime
- `fault_status` — Read fault state
- `flow_read` — Read flow rate
- `pressure_read` — Read pressure
- `power_read` — Read power consumption
- `vibration_read` — Read vibration (PHM)

### Variable Frequency Drives (VFDs)

VFDs control pump and fan motor speed for energy efficiency.

**State Model:**
```yaml
VFDState:
  running: boolean              # Motor running
  speed_percent: float          # Speed setpoint (0-100%)
  speed_hz: float               # Output frequency
  speed_rpm: float | null       # Motor RPM if available
  
  # Electrical
  output_voltage: float         # Volts
  output_current: float         # Amps
  output_power: float           # kW
  dc_bus_voltage: float | null  # DC link voltage
  
  # Status
  fault: boolean
  fault_code: string | null
  warning: boolean
  warning_code: string | null
  
  # Thermal
  heatsink_temp: float | null   # °C
  motor_temp: float | null      # °C (if PTC connected)
  
  # Counters
  run_hours: integer
  energy_kwh: float             # Cumulative energy
```

**Common Modbus Registers (ABB ACS580 example):**
```yaml
vfd_profile:
  manufacturer: "ABB"
  model: "ACS580"
  registers:
    # Status
    status_word:
      address: 2101
      type: "holding"
      format: "uint16"
      bits:
        0: "ready"
        1: "running"
        2: "direction"
        3: "fault"
        4: "warning"
    
    # Readings
    output_frequency:
      address: 2103
      type: "holding"
      format: "uint16"
      scale: 0.1
      unit: "Hz"
    
    output_current:
      address: 2104
      type: "holding"
      format: "uint16"
      scale: 0.1
      unit: "A"
    
    output_power:
      address: 2106
      type: "holding"
      format: "uint16"
      scale: 0.1
      unit: "kW"
    
    # Control
    speed_reference:
      address: 1
      type: "holding"
      format: "uint16"
      scale: 0.01
      unit: "%"
      writeable: true
    
    control_word:
      address: 0
      type: "holding"
      format: "uint16"
      writeable: true
      bits:
        0: "start"
        1: "stop"
        2: "direction"
        7: "fault_reset"
```

### Air Handling Units (AHUs)

AHUs condition and distribute air. Often BACnet-controlled in commercial settings.

**State Model:**
```yaml
AHUState:
  # Status
  running: boolean
  operating_mode: enum          # off | heating | cooling | free_cooling | economy
  fault: boolean
  fault_codes: [string]
  
  # Temperatures (°C)
  supply_air_temp: float
  return_air_temp: float
  outside_air_temp: float
  mixed_air_temp: float | null
  
  # Setpoints
  supply_air_setpoint: float
  
  # Airflow
  supply_air_pressure: float | null  # Pa
  supply_airflow: float | null       # m³/h
  return_airflow: float | null
  
  # Dampers (0-100%)
  outside_air_damper: float
  return_air_damper: float
  exhaust_air_damper: float | null
  
  # Valves (0-100%)
  cooling_valve: float
  heating_valve: float
  humidifier_valve: float | null
  
  # Fan status
  supply_fan:
    running: boolean
    speed_percent: float
    fault: boolean
  return_fan:
    running: boolean
    speed_percent: float
    fault: boolean
  
  # Filter status
  filter_dp: float              # Pa differential pressure
  filter_alarm: boolean
  
  # Safety
  freeze_stat: boolean          # Freeze protection active
  smoke_detected: boolean
  
  # Energy
  power_kw: float | null
  run_hours: integer
```

**Operating Modes:**
```yaml
AHUMode:
  - off           # Unit disabled
  - heating       # Heating coil active
  - cooling       # Cooling coil active
  - free_cooling  # 100% outside air, no mechanical cooling
  - economy       # Mixed air economizer active
  - night_purge   # Pre-cooling with outside air
  - warmup        # Morning warm-up sequence
  - setback       # Reduced ventilation, maintain minimum
```

### Chillers

**State Model:**
```yaml
ChillerState:
  # Status
  running: boolean
  operating_mode: enum          # off | cooling | ice_making
  capacity_percent: float       # 0-100%
  
  # Temperatures (°C)
  leaving_water_temp: float     # LCHWT
  entering_water_temp: float    # ECHWT
  setpoint: float
  
  # Condenser (air-cooled or water-cooled)
  condenser_entering_temp: float
  condenser_leaving_temp: float | null
  
  # Refrigerant
  suction_pressure: float | null   # bar
  discharge_pressure: float | null
  superheat: float | null          # K
  subcooling: float | null         # K
  
  # Compressors
  compressors:
    - id: 1
      running: boolean
      load_percent: float
      run_hours: integer
      fault: boolean
  
  # Alarms
  fault: boolean
  fault_codes: [string]
  high_pressure_alarm: boolean
  low_pressure_alarm: boolean
  flow_alarm: boolean
  
  # Energy
  power_kw: float
  efficiency_kw_ton: float | null  # kW/ton
  run_hours: integer
```

### Boilers and Heat Pumps

**State Model:**
```yaml
BoilerState:
  # Status
  running: boolean
  firing: boolean               # Burner active
  modulation_percent: float     # 0-100%
  
  # Temperatures (°C)
  flow_temp: float              # Water leaving boiler
  return_temp: float            # Water entering boiler
  flow_setpoint: float
  flue_temp: float | null       # Exhaust gas
  
  # Safety
  fault: boolean
  fault_code: string | null
  lockout: boolean              # Requires manual reset
  flame_signal: float | null    # Flame ionization
  
  # Pressure
  system_pressure: float        # bar
  low_pressure_fault: boolean
  
  # Energy
  gas_consumption: float | null # m³/h
  run_hours: integer
  ignition_cycles: integer
```

```yaml
HeatPumpState:
  # Status
  running: boolean
  operating_mode: enum          # off | heating | cooling | dhw | defrost
  
  # Temperatures (°C)
  flow_temp: float
  return_temp: float
  outside_temp: float           # Source temp (air/ground)
  flow_setpoint: float
  
  # Performance
  capacity_percent: float
  cop: float | null             # Coefficient of Performance
  
  # Compressor
  compressor_running: boolean
  compressor_speed: float | null  # Hz if inverter
  compressor_current: float | null
  
  # Defrost (ASHP)
  defrost_active: boolean
  defrost_cycles: integer
  
  # Status
  fault: boolean
  fault_code: string | null
  
  # Energy
  power_kw: float
  run_hours: integer
```

---

## Commands

### Pump Commands

```yaml
# Start pump
command: "run"
parameters:
  speed_percent: 100           # Optional, default 100%

# Stop pump
command: "stop"

# Set speed (VFD)
command: "set_speed"
parameters:
  speed_percent: 75            # 0-100%
  # OR
  speed_hz: 45                 # Direct frequency

# Reset fault
command: "fault_reset"
```

### AHU Commands

```yaml
# Enable/disable
command: "enable"
parameters:
  enabled: true

# Set supply air setpoint
command: "set_supply_temp"
parameters:
  temperature: 18.0

# Set operating mode
command: "set_mode"
parameters:
  mode: "cooling"              # off | heating | cooling | economy

# Override damper position (commissioning)
command: "override_damper"
parameters:
  damper: "outside_air"
  position: 50                 # 0-100%
  duration_min: 30             # Auto-release
```

### Chiller Commands

```yaml
# Enable/disable
command: "enable"
parameters:
  enabled: true

# Set leaving water setpoint
command: "set_setpoint"
parameters:
  temperature: 7.0

# Set capacity limit
command: "set_capacity_limit"
parameters:
  max_percent: 80
```

---

## Sequences of Operation

### Pump Lead/Lag

For redundant pump pairs:

```yaml
PumpLeadLagConfig:
  group_id: "chw-pumps"
  pumps:
    - id: "pump-chw-1"
      role: "lead"             # Currently lead
    - id: "pump-chw-2"
      role: "lag"              # Currently standby
  
  rotation:
    enabled: true
    interval_hours: 168        # Weekly rotation
    last_rotation: "2026-01-05T00:00:00Z"
  
  staging:
    lag_start_threshold: 80    # Start lag at 80% lead capacity
    lag_stop_threshold: 40     # Stop lag below 40% total demand
    stage_delay_min: 5         # Delay between staging
  
  failover:
    auto_failover: true
    failover_on:
      - "fault"
      - "communication_loss"   # After 60s timeout
    notification: true
```

**Lead/Lag Logic:**
```
1. Start lead pump on demand
2. Monitor flow/pressure/speed
3. If lead reaches staging threshold, start lag
4. If demand drops, stop lag (stage_delay after start)
5. On lead fault, immediately start lag
6. After rotation interval, swap lead/lag designation
7. Run hours should equalize over time
```

### AHU Economizer Sequence

```yaml
EconomizerConfig:
  ahu_id: "ahu-01"
  
  # Enable conditions
  enable_conditions:
    outside_air_temp_max: 18     # °C
    outside_air_temp_min: 5      # °C
    outside_humidity_max: 70     # %RH
    
  # Staging
  stages:
    # Stage 1: Minimum outside air
    - name: "minimum_oa"
      condition: "default"
      outside_air_damper: 20
      
    # Stage 2: Economizer (modulating)
    - name: "economy"
      condition: "cooling_demand AND oa_suitable"
      outside_air_damper: "modulating"  # 20-100%
      cooling_valve: 0
      
    # Stage 3: Mechanical cooling
    - name: "mechanical"
      condition: "economy_insufficient"
      outside_air_damper: 20
      cooling_valve: "modulating"       # 0-100%
```

### Chiller Staging

```yaml
ChillerStagingConfig:
  plant_id: "chiller-plant"
  
  chillers:
    - id: "chiller-1"
      capacity_kw: 500
      min_load_percent: 20
      priority: 1
    - id: "chiller-2"
      capacity_kw: 500
      min_load_percent: 20
      priority: 2
  
  staging:
    start_threshold_percent: 85    # Start next chiller
    stop_threshold_percent: 40     # Stop trailing chiller
    stage_delay_min: 10
    
  optimization:
    equal_loading: true            # Balance load across running chillers
    efficiency_tracking: true      # Monitor kW/ton
```

---

## Alarm Management

### Alarm Priorities

| Priority | Name | Response Time | Examples |
|----------|------|---------------|----------|
| 1 | **Critical** | Immediate | Equipment protection trip, safety interlock |
| 2 | **High** | < 15 min | Equipment fault, process deviation |
| 3 | **Medium** | < 1 hour | Maintenance required, sensor fault |
| 4 | **Low** | < 24 hours | Filter change, routine service |
| 5 | **Info** | Logged only | Mode changes, normal transitions |

### Alarm State Machine

```
                    ┌──────────────┐
                    │   NORMAL     │
                    └──────┬───────┘
                           │ condition triggered
                           ▼
                    ┌──────────────┐
                    │   ACTIVE     │◄──────────────┐
                    │ (unacknowledged)             │
                    └──────┬───────┘               │ return to alarm
                           │ acknowledge           │
                           ▼                       │
                    ┌──────────────┐               │
        ┌───────────│ ACKNOWLEDGED │───────────────┤
        │           └──────┬───────┘               │
        │                  │ condition cleared     │
        │                  ▼                       │
        │           ┌──────────────┐               │
        │           │   CLEARED    │───────────────┘
        │           │(needs review)│
        │           └──────┬───────┘
        │                  │ reset
        │                  ▼
        │           ┌──────────────┐
        └──────────►│   NORMAL     │
   (cleared while   └──────────────┘
    acknowledged)
```

### Alarm Configuration

```yaml
AlarmConfig:
  device_id: "pump-chw-1"
  
  alarms:
    - id: "fault"
      name: "Pump Fault"
      priority: 2
      condition:
        property: "fault"
        operator: "eq"
        value: true
      message: "CHW Pump 1 fault - check VFD"
      actions:
        - notify: ["facility_manager", "on_call"]
        - start_lag_pump: true
        
    - id: "high_vibration"
      name: "High Vibration Warning"
      priority: 3
      condition:
        property: "vibration_level"
        operator: "gt"
        value: 4.5                 # mm/s RMS
      delay_sec: 60                # Must persist
      message: "CHW Pump 1 vibration elevated - schedule inspection"
      actions:
        - notify: ["maintenance"]
        
    - id: "run_hours_service"
      name: "Service Due"
      priority: 4
      condition:
        property: "run_hours"
        operator: "gt"
        value: 8760                # Annual service
      message: "CHW Pump 1 service due ({{ run_hours }} hours)"
      reset_on_acknowledge: true   # Reset counter after service
```

### Alarm Shelving

Temporarily suppress alarms during maintenance:

```yaml
AlarmShelve:
  alarm_id: "pump-chw-1.high_vibration"
  shelved_by: "installer_john"
  shelved_at: "2026-01-12T09:00:00Z"
  shelve_until: "2026-01-12T17:00:00Z"
  reason: "Planned maintenance - impeller replacement"
  max_duration_hours: 24          # Auto-unshelve limit
```

---

## Predictive Health Monitoring (PHM)

### Target Equipment

| Equipment | PHM Value | Key Indicators |
|-----------|-----------|----------------|
| Pumps | High | Vibration, bearing temp, power deviation |
| VFDs | Medium | DC bus ripple, heatsink temp, output current |
| AHU Fans | High | Vibration, belt wear, filter DP trend |
| Chillers | High | Refrigerant pressures, compressor current, COP |
| Boilers | Medium | Flue temp trend, ignition cycles, modulation hunting |

### Baseline Learning

```yaml
PHMBaseline:
  device_id: "pump-chw-1"
  parameter: "vibration_level"
  
  # Learning period
  learning_start: "2026-01-01T00:00:00Z"
  learning_end: "2026-01-08T00:00:00Z"
  learning_samples: 10080          # 1-minute samples
  
  # Baseline statistics
  baseline:
    mean: 2.1                      # mm/s RMS
    std_dev: 0.3
    min: 1.5
    max: 3.2
    
  # Operating context
  context:
    speed_percent:
      50: { mean: 1.8, std_dev: 0.2 }
      75: { mean: 2.1, std_dev: 0.25 }
      100: { mean: 2.5, std_dev: 0.3 }
    
  # Alert thresholds (auto-calculated)
  thresholds:
    warning: 3.5                   # baseline.mean + 4*std_dev
    alarm: 4.5                     # baseline.mean + 8*std_dev
    critical: 7.1                  # ISO 10816 limit
```

### Deviation Detection

```yaml
PHMDeviation:
  device_id: "pump-chw-1"
  parameter: "vibration_level"
  
  detection:
    method: "rolling_zscore"
    window_hours: 24
    threshold_sigma: 3             # Alert at 3σ deviation
    
  current_status:
    latest_value: 3.8
    rolling_mean: 2.3
    zscore: 5.0                    # Significant deviation
    trend: "increasing"
    trend_rate: 0.1                # mm/s per week
    
  prediction:
    days_to_warning: 14
    days_to_alarm: 28
    confidence: 0.85
```

### PHM Dashboard Data

```yaml
PHMSummary:
  device_id: "pump-chw-1"
  health_score: 72                 # 0-100
  
  indicators:
    - name: "Vibration"
      status: "warning"
      value: 3.8
      baseline: 2.1
      trend: "increasing"
      
    - name: "Bearing Temperature"
      status: "normal"
      value: 45
      baseline: 42
      trend: "stable"
      
    - name: "Power Consumption"
      status: "normal"
      value: 5.2
      baseline: 5.0
      trend: "stable"
      
  recommendations:
    - priority: "medium"
      message: "Schedule vibration analysis - bearing wear suspected"
      action: "create_work_order"
```

---

## Energy Optimization

### Pump Affinity Laws

VFD speed control follows affinity laws:

```
Flow ∝ Speed
Head ∝ Speed²
Power ∝ Speed³
```

Energy savings at reduced speed are significant:
- 80% speed = 51% power
- 60% speed = 22% power
- 50% speed = 12.5% power

### Optimization Strategies

```yaml
EnergyOptimization:
  type: "pump_pressure_reset"
  
  config:
    pump_id: "pump-chw-1"
    setpoint_type: "differential_pressure"
    
    # Fixed setpoint (baseline)
    fixed_setpoint: 150            # kPa
    
    # Trim-and-respond
    trim_respond:
      enabled: true
      min_setpoint: 80             # kPa
      max_setpoint: 150            # kPa
      trim_interval_min: 5
      trim_amount: 5               # kPa per interval
      respond_threshold: 95        # % valve open anywhere
      respond_amount: 10           # kPa increase
      
  estimated_savings:
    baseline_kwh_year: 45000
    optimized_kwh_year: 28000
    savings_percent: 38
```

### Run-Time Logging

```yaml
RuntimeLog:
  device_id: "pump-chw-1"
  period: "2026-01"
  
  summary:
    total_run_hours: 456
    total_energy_kwh: 2280
    average_speed_percent: 72
    average_power_kw: 5.0
    
  daily:
    - date: "2026-01-01"
      run_hours: 18.5
      energy_kwh: 85
      average_speed: 70
    - date: "2026-01-02"
      run_hours: 22.0
      energy_kwh: 110
      average_speed: 78
```

---

## Protocol Mapping

### Modbus (Residential/Light Commercial)

Most plant equipment uses Modbus RTU or TCP:

```yaml
# Example: Grundfos Magna3 pump with integrated VFD
modbus_profile:
  name: "Grundfos Magna3"
  protocol: "modbus_tcp"
  
  connection:
    default_port: 502
    unit_id: 1
    
  registers:
    actual_speed:
      address: 0x0080
      type: "input"
      format: "uint16"
      scale: 0.1
      unit: "%"
      
    setpoint:
      address: 0x0100
      type: "holding"
      format: "uint16"
      scale: 0.1
      unit: "%"
      writeable: true
      
    operating_mode:
      address: 0x0081
      type: "input"
      format: "uint16"
      enum:
        0: "stopped"
        1: "running"
        2: "fault"
        
    run_hours:
      address: 0x0090
      type: "input"
      format: "uint32"
      unit: "hours"
      
    power:
      address: 0x0084
      type: "input"
      format: "uint16"
      unit: "W"
```

### BACnet (Commercial)

AHUs and chillers often use BACnet — see [BACnet Protocol](../protocols/bacnet.md) for object mapping.

---

## Commissioning Checklist

### Pump Commissioning

- [ ] Verify Modbus/BACnet communication
- [ ] Confirm run/stop control
- [ ] Test fault reset command
- [ ] Verify speed control range
- [ ] Record baseline vibration
- [ ] Record baseline power at 100%
- [ ] Configure run-hour counter reset point
- [ ] Test lead/lag failover
- [ ] Configure alarm thresholds
- [ ] Enable PHM baseline learning

### AHU Commissioning

- [ ] Verify all sensor readings plausible
- [ ] Confirm supply fan start/stop
- [ ] Confirm return fan start/stop
- [ ] Test damper operation (0%, 50%, 100%)
- [ ] Test valve operation (heating, cooling)
- [ ] Verify economizer sequence
- [ ] Test safety interlocks (freeze stat, smoke)
- [ ] Record baseline filter DP
- [ ] Configure setpoints
- [ ] Enable sequence schedules

---

## Related Documents

- [Entities](../data-model/entities.md) — Device type definitions
- [Modbus Protocol](../protocols/modbus.md) — Modbus register mapping
- [BACnet Protocol](../protocols/bacnet.md) — BACnet object mapping
- [PHM Specification](../intelligence/phm.md) — Predictive health monitoring framework
- [Climate Domain](climate.md) — HVAC zone control
- [Energy Model](../architecture/energy-model.md) — Energy monitoring

