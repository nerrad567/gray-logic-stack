---
title: Energy Model Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/core-internals.md
  - protocols/modbus.md
  - protocols/mqtt.md
---

# Energy Model Specification

This document specifies how Gray Logic models, monitors, and manages energy flows within a building. The model supports bidirectional flows from the start, enabling future V2G (Vehicle-to-Grid) and V2H (Vehicle-to-Home) capabilities.

---

## Overview

Modern buildings are evolving from simple energy consumers to active participants in the energy ecosystem. Gray Logic models energy as bidirectional flows between nodes, supporting:

- Solar PV generation
- Battery storage
- EV charging and discharging
- Grid import and export
- Load management

### Design Principles

1. **Bidirectional from start** — Model flows in both directions
2. **Node-based architecture** — All sources and loads are nodes
3. **Real-time visibility** — Current power and accumulated energy
4. **Historical analysis** — Time-series for optimization
5. **Protocol agnostic** — Works with any meter/inverter via Modbus

---

## Energy Flow Model

### Conceptual Model

```
                                    ┌─────────────┐
                                    │    GRID     │
                                    │   (import/  │
                                    │   export)   │
                                    └──────┬──────┘
                                           │
                                           │ bidirectional
                                           │
                              ┌────────────┴────────────┐
                              │      MAIN METER         │
                              │   (grid connection)     │
                              └────────────┬────────────┘
                                           │
            ┌──────────────────────────────┼──────────────────────────────┐
            │                              │                              │
            │                              │                              │
     ┌──────┴──────┐              ┌────────┴────────┐             ┌───────┴───────┐
     │   SOLAR     │              │     BATTERY     │             │      EV       │
     │ (generate)  │              │ (charge/disch)  │             │ (charge/V2H)  │
     └──────┬──────┘              └────────┬────────┘             └───────┬───────┘
            │                              │                              │
            │                              │                              │
            └──────────────────────────────┼──────────────────────────────┘
                                           │
                              ┌────────────┴────────────┐
                              │         LOADS           │
                              │  (circuits, devices)    │
                              └─────────────────────────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    │                      │                      │
             ┌──────┴──────┐       ┌───────┴───────┐       ┌──────┴──────┐
             │   HVAC      │       │   LIGHTING    │       │    OTHER    │
             │   loads     │       │    loads      │       │    loads    │
             └─────────────┘       └───────────────┘       └─────────────┘
```

### Node Types

| Node Type | Direction | Examples |
|-----------|-----------|----------|
| `grid` | Bidirectional | Main meter, grid connection |
| `solar` | Generation only | PV inverter, solar array |
| `battery` | Bidirectional | Home battery, PowerWall |
| `ev` | Bidirectional | EV charger with V2H |
| `load` | Consumption only | Circuits, devices |
| `meter` | Measurement only | CT clamps, sub-meters |

---

## Data Model

### Energy Node

```yaml
EnergyNode:
  id: uuid
  name: string
  type: enum                        # grid | solar | battery | ev | load | meter
  
  # Physical connection
  protocol: enum                    # modbus_tcp | modbus_rtu | mqtt
  address: object                   # Protocol-specific address
  
  # Node properties
  properties:
    direction: enum                 # import | export | bidirectional
    max_power_w: integer            # Maximum power rating
    phases: integer                 # 1 or 3
    
  # Current state
  state:
    power_w: float                  # Current power (+ = export/generation, - = import/consumption)
    energy_today_wh: float          # Energy today
    energy_total_wh: float          # Lifetime energy
    voltage_v: float[]              # Per-phase voltage
    current_a: float[]              # Per-phase current
    power_factor: float             # Power factor
    frequency_hz: float             # Grid frequency
    
  # For batteries/EVs
  storage:
    capacity_wh: integer            # Total capacity
    soc_percent: float              # State of charge
    charging: boolean               # Currently charging
    discharging: boolean            # Currently discharging
    
  created_at: timestamp
  updated_at: timestamp
```

### Energy Flow

```yaml
EnergyFlow:
  id: uuid
  timestamp: timestamp
  interval_seconds: integer         # Measurement interval (e.g., 60)
  
  # Node measurements
  nodes:
    - node_id: "grid-main"
      power_w: -2500                # Importing 2.5kW
      energy_wh: 41.67              # Energy in interval
      
    - node_id: "solar-roof"
      power_w: 4500                 # Generating 4.5kW
      energy_wh: 75.0
      
    - node_id: "battery-main"
      power_w: 1500                 # Charging at 1.5kW
      energy_wh: 25.0
      soc_percent: 65.0
      
    - node_id: "ev-charger"
      power_w: 0
      energy_wh: 0
      connected: false
      
    - node_id: "load-hvac"
      power_w: -1200                # Consuming 1.2kW
      energy_wh: 20.0
```

### Sign Convention

**Power (W):**
- Positive (+) = Generation / Export / Discharge
- Negative (-) = Consumption / Import / Charge

This follows the "export positive" convention common in energy management.

| Scenario | Power Sign |
|----------|------------|
| Solar generating | +4500 W |
| Importing from grid | -2500 W |
| Battery charging | -1500 W |
| Battery discharging | +1500 W |
| EV charging | -7000 W |
| EV V2H (discharging) | +5000 W |
| Load consuming | -1200 W |

---

## Device Configuration

### Grid Connection (Main Meter)

```yaml
energy_nodes:
  - id: "grid-main"
    name: "Grid Connection"
    type: "grid"
    
    protocol: "modbus_tcp"
    address:
      host: "192.168.1.100"
      port: 502
      unit_id: 1
      
    device_profile: "eastron_sdm630"
    
    properties:
      direction: "bidirectional"
      max_power_w: 27600            # 40A x 230V x 3 phases
      phases: 3
      
    registers:
      # Eastron SDM630 register map
      voltage_l1: { address: 0, type: "float32", unit: "V" }
      voltage_l2: { address: 2, type: "float32", unit: "V" }
      voltage_l3: { address: 4, type: "float32", unit: "V" }
      current_l1: { address: 6, type: "float32", unit: "A" }
      current_l2: { address: 8, type: "float32", unit: "A" }
      current_l3: { address: 10, type: "float32", unit: "A" }
      power_total: { address: 52, type: "float32", unit: "W" }
      energy_import: { address: 72, type: "float32", unit: "kWh" }
      energy_export: { address: 74, type: "float32", unit: "kWh" }
      frequency: { address: 70, type: "float32", unit: "Hz" }
```

### Solar Inverter

```yaml
energy_nodes:
  - id: "solar-roof"
    name: "Roof Solar"
    type: "solar"
    
    protocol: "modbus_tcp"
    address:
      host: "192.168.1.101"
      port: 502
      unit_id: 1
      
    device_profile: "solaredge_se5000h"
    
    properties:
      direction: "export"           # Generation only
      max_power_w: 5000             # 5kW inverter
      phases: 1
      
    # SolarEdge-specific registers
    registers:
      ac_power: { address: 40083, type: "int16", scale: 40084, unit: "W" }
      ac_energy_wh: { address: 40093, type: "uint32", unit: "Wh" }
      dc_power: { address: 40100, type: "float32", unit: "W" }
      temperature: { address: 40103, type: "int16", scale: 40106, unit: "C" }
      status: { address: 40107, type: "uint16" }
```

### Battery System

```yaml
energy_nodes:
  - id: "battery-main"
    name: "Home Battery"
    type: "battery"
    
    protocol: "modbus_tcp"
    address:
      host: "192.168.1.102"
      port: 502
      unit_id: 1
      
    device_profile: "tesla_powerwall"
    
    properties:
      direction: "bidirectional"
      max_power_w: 5000
      phases: 1
      
    storage:
      capacity_wh: 13500            # 13.5 kWh
      min_soc_percent: 10           # Reserve
      max_soc_percent: 100
      
    registers:
      power: { address: 1000, type: "int32", unit: "W" }
      soc: { address: 1010, type: "uint16", unit: "%" }
      state: { address: 1020, type: "uint16" }
```

### EV Charger (OCPP)

```yaml
energy_nodes:
  - id: "ev-charger-1"
    name: "Garage EV Charger"
    type: "ev"
    
    protocol: "ocpp"
    address:
      chargepoint_id: "CP001"
      
    properties:
      direction: "bidirectional"    # V2H capable
      max_power_w: 22000            # 22kW
      phases: 3
      
    capabilities:
      - "smart_charging"
      - "v2g"                       # Vehicle-to-grid
      - "v2h"                       # Vehicle-to-home
      
    # OCPP configuration
    ocpp:
      version: "2.0.1"
      authorization: true
      meter_values_interval: 60
```

### Sub-Meter / CT Clamp

```yaml
energy_nodes:
  - id: "load-hvac"
    name: "HVAC Circuit"
    type: "load"
    
    protocol: "modbus_rtu"
    address:
      port: "/dev/ttyUSB0"
      baudrate: 9600
      unit_id: 2
      
    device_profile: "shelly_pro_3em"
    
    properties:
      direction: "import"           # Consumption only
      max_power_w: 6000
      phases: 1
      parent_node: "grid-main"      # This is behind the main meter
```

### Device-Level Energy Attribution

For equipment that doesn't have built-in energy monitoring, external sensors can be associated with devices to track per-device consumption. This uses the **Device Association** mechanism.

```
┌───────────────────┐        ┌───────────────────┐
│  CT Clamp         │───────→│  Pump             │
│  (measures power) │monitors│  (consumes power) │
└───────────────────┘        └───────────────────┘
         │                            │
         ▼                            ▼
    Energy Model sees:           Device shows:
    "pump-chw-1: 5.2 kW"        "Power: 5.2 kW"
```

**Configuration:**

```yaml
# 1. Define the monitoring device
devices:
  - id: "ct-pump-chw-1"
    type: "ct_clamp"
    protocol: "modbus_tcp"
    address:
      host: "192.168.1.100"
      unit_id: 3

# 2. Create association to attribute readings
associations:
  - source_device_id: "ct-pump-chw-1"
    target_device_id: "pump-chw-1"
    type: "monitors"
    config:
      metrics: ["power_kw", "energy_kwh"]
```

**How attribution works:**

1. CT clamp reports power/energy via Modbus bridge
2. Association Resolver attributes readings to `pump-chw-1`
3. Energy Model tracks `pump-chw-1` consumption
4. Reports show per-device energy breakdown

**Use cases:**

| Scenario | Configuration | Benefit |
|----------|---------------|---------|
| Track pump energy | CT clamp → pump | PHM + energy cost allocation |
| Monitor lighting circuit | DIN meter → light group | Circuit-level reporting |
| Smart plug on heater | Smart plug → heater | Per-device tracking + control |
| HVAC zone consumption | Sub-meter → HVAC zone | Zone-level cost allocation |

**Reporting with attribution:**

```yaml
# Energy breakdown by device (using associations)
energy_report:
  period: "2026-01"
  by_device:
    - device_id: "pump-chw-1"
      device_name: "Chilled Water Pump 1"
      energy_kwh: 245.6
      cost: 36.84
      source: "ct-pump-chw-1"       # Via association
      
    - device_id: "heater-garage"
      device_name: "Garage Heater"
      energy_kwh: 89.2
      cost: 13.38
      source: "smart-plug-garage"   # Via association
      
    - device_id: "heatpump-main"
      device_name: "Heat Pump"
      energy_kwh: 312.4
      cost: 46.86
      source: "native"              # Built-in monitoring
```

See [Data Model: DeviceAssociation](../data-model/entities.md#deviceassociation) for full specification.

---

## Energy Balance Calculations

### Real-Time Balance

At any moment:

```
Grid Power = Solar Power + Battery Power + EV Power - Total Load Power

Where:
- Grid Power < 0 = Importing from grid
- Grid Power > 0 = Exporting to grid
- Battery Power < 0 = Charging
- Battery Power > 0 = Discharging
```

### Self-Consumption Calculation

```go
type EnergyBalance struct {
    Timestamp      time.Time
    
    // Generation
    SolarPower     float64  // W
    
    // Storage
    BatteryPower   float64  // W (+discharge, -charge)
    BatterySoC     float64  // %
    EVPower        float64  // W (+V2H, -charging)
    
    // Grid
    GridPower      float64  // W (+export, -import)
    
    // Loads
    TotalLoad      float64  // W
    
    // Calculated
    SelfConsumption float64 // % of solar used directly
    Autarky        float64  // % of load met by own generation
}

func CalculateBalance(nodes []EnergyNode) EnergyBalance {
    balance := EnergyBalance{Timestamp: time.Now()}
    
    for _, node := range nodes {
        switch node.Type {
        case "solar":
            balance.SolarPower += node.State.PowerW
        case "battery":
            balance.BatteryPower += node.State.PowerW
            balance.BatterySoC = node.State.Storage.SoCPercent
        case "ev":
            balance.EVPower += node.State.PowerW
        case "grid":
            balance.GridPower += node.State.PowerW
        case "load":
            balance.TotalLoad += abs(node.State.PowerW)
        }
    }
    
    // Self-consumption = (Solar - Export) / Solar
    if balance.SolarPower > 0 {
        exported := max(0, balance.GridPower)
        balance.SelfConsumption = (balance.SolarPower - exported) / balance.SolarPower * 100
    }
    
    // Autarky = (Load - Import) / Load
    if balance.TotalLoad > 0 {
        imported := abs(min(0, balance.GridPower))
        balance.Autarky = (balance.TotalLoad - imported) / balance.TotalLoad * 100
    }
    
    return balance
}
```

---

## Time-of-Use Optimization

### Tariff Configuration

```yaml
tariffs:
  - id: "tariff-octopus-agile"
    name: "Octopus Agile"
    type: "dynamic"
    
    # Dynamic pricing (fetched from API)
    api:
      provider: "octopus"
      product_code: "AGILE-FLEX-22-11-25"
      
    # Import and export rates
    rates:
      import:
        source: "api"
        unit: "GBP/kWh"
        
      export:
        source: "api"
        unit: "GBP/kWh"
        
  - id: "tariff-economy-7"
    name: "Economy 7"
    type: "time_of_use"
    
    # Fixed time-of-use rates
    rates:
      import:
        peak:
          hours: ["07:00-00:00"]
          rate: 0.35                # £/kWh
        off_peak:
          hours: ["00:00-07:00"]
          rate: 0.15
          
      export:
        rate: 0.15                  # Fixed export rate
```

### Battery Optimization Strategy

```yaml
battery_optimization:
  enabled: true
  battery_id: "battery-main"
  tariff_id: "tariff-octopus-agile"
  
  strategies:
    # Charge from grid during cheap periods
    charge_from_grid:
      enabled: true
      max_import_rate: 0.10         # Only charge if rate < £0.10/kWh
      target_soc: 80                # Charge to 80%
      
    # Discharge to avoid expensive grid import
    discharge_to_home:
      enabled: true
      min_export_rate: 0.20         # Discharge if import > £0.20/kWh
      min_soc: 20                   # Don't discharge below 20%
      
    # Export to grid during high prices
    grid_export:
      enabled: true
      min_export_rate: 0.30         # Export if rate > £0.30/kWh
      min_soc: 30                   # Keep 30% reserve
      
  # Solar priority (always)
  solar_priority:
    - "self_consumption"            # Use solar directly
    - "battery_charge"              # Then charge battery
    - "grid_export"                 # Then export
```

---

## EV Charging Integration

### Smart Charging

```yaml
ev_charging:
  charger_id: "ev-charger-1"
  
  # Charging modes
  modes:
    immediate:
      description: "Charge as fast as possible"
      max_power: 22000
      
    scheduled:
      description: "Charge by departure time"
      departure_time: "07:30"
      required_soc: 80
      prefer_cheap: true
      
    solar_only:
      description: "Only charge from solar"
      min_solar_surplus: 1400       # Minimum surplus to start
      
    smart:
      description: "Optimize for cost and solar"
      departure_time: "07:30"
      required_soc: 80
      prefer_solar: true
      max_import_rate: 0.15         # Tariff threshold
      
  # V2H configuration
  v2h:
    enabled: true
    min_vehicle_soc: 30             # Keep 30% for driving
    max_discharge_power: 5000       # Limit V2H power
    
    triggers:
      - type: "grid_outage"
        action: "backup_power"
        
      - type: "high_tariff"
        threshold: 0.40             # Rate > £0.40/kWh
        action: "power_home"
```

### OCPP Integration

```yaml
ocpp:
  # Gray Logic as CSMS (Charging Station Management System)
  server:
    listen_address: "0.0.0.0:9000"
    protocol: "ocpp2.0.1"
    
  charge_points:
    - id: "CP001"
      name: "Garage Charger"
      location: "garage"
      
      # Authorization
      auth:
        mode: "local_list"          # Or "central" for remote auth
        local_list:
          - id_token: "USER001"
            type: "ISO14443"
            
      # Smart charging profile
      charging_profile:
        type: "TxDefaultProfile"
        stack_level: 1
        schedule:
          - start: "00:00"
            limit_w: 22000
          - start: "07:00"
            limit_w: 7000           # Limit during peak hours
          - start: "22:00"
            limit_w: 22000
```

---

## Grid Services (Future)

### Demand Response

```yaml
demand_response:
  enabled: false                    # Future feature
  
  # Grid signal sources
  signals:
    - provider: "national_grid"
      signal: "demand_flexibility"
      
  # Assets that can respond
  flex_assets:
    - id: "battery-main"
      type: "battery"
      response_time_s: 30
      
    - id: "ev-charger-1"
      type: "ev"
      response_time_s: 60
      
    - id: "load-hvac"
      type: "load"
      response_time_s: 300
      sheddable: true
```

### Frequency Response (Future)

```yaml
frequency_response:
  enabled: false                    # Future feature / requires certification
  
  # Only for battery systems with fast response
  assets:
    - id: "battery-main"
      response_time_ms: 1000
      min_capacity_kw: 5
```

---

## Monitoring & Alerts

### Energy Dashboard Metrics

```yaml
dashboard_metrics:
  # Real-time
  real_time:
    - metric: "grid_power"
      unit: "W"
      
    - metric: "solar_power"
      unit: "W"
      
    - metric: "battery_power"
      unit: "W"
      
    - metric: "battery_soc"
      unit: "%"
      
    - metric: "ev_power"
      unit: "W"
      
    - metric: "total_load"
      unit: "W"
      
  # Daily summaries
  daily:
    - metric: "energy_imported"
      unit: "kWh"
      
    - metric: "energy_exported"
      unit: "kWh"
      
    - metric: "solar_generated"
      unit: "kWh"
      
    - metric: "self_consumption"
      unit: "%"
      
    - metric: "cost_today"
      unit: "GBP"
      
    - metric: "savings_today"
      unit: "GBP"
```

### Energy Alerts

```yaml
energy_alerts:
  - name: "High Grid Import"
    condition:
      metric: "grid_power"
      operator: "lt"                # Less than (importing)
      threshold: -10000             # > 10kW import
    severity: "warning"
    
  - name: "Battery Low"
    condition:
      metric: "battery_soc"
      operator: "lt"
      threshold: 15
    severity: "warning"
    
  - name: "Solar Production Low"
    condition:
      metric: "solar_power"
      operator: "lt"
      threshold: 100
      when: "daytime"
    severity: "info"
    
  - name: "Grid Export"
    condition:
      metric: "grid_power"
      operator: "gt"
      threshold: 0
    severity: "info"
    log_only: true
```

---

## InfluxDB Storage

### Time-Series Schema

```
Measurement: energy_node
Tags:
  - node_id
  - node_type
  - phase (for multi-phase)
  
Fields:
  - power (float, W)
  - voltage (float, V)
  - current (float, A)
  - power_factor (float)
  - frequency (float, Hz)
  - energy_total (float, Wh)
  - soc (float, %) [for storage]

Example:
energy_node,node_id=grid-main,node_type=grid power=-2500,voltage_l1=233.5,voltage_l2=234.1,voltage_l3=232.8 1736674200000000000
```

### Retention Policies

```
# High resolution for recent data
CREATE RETENTION POLICY "realtime" ON "graylogic" DURATION 7d REPLICATION 1 DEFAULT

# Downsampled for historical
CREATE RETENTION POLICY "historical" ON "graylogic" DURATION 365d REPLICATION 1

# Continuous query for downsampling
CREATE CONTINUOUS QUERY "downsample_energy" ON "graylogic"
BEGIN
  SELECT mean(power) as power_avg,
         max(power) as power_max,
         sum(power) / 60 as energy_wh
  INTO "historical"."energy_node_hourly"
  FROM "realtime"."energy_node"
  GROUP BY time(1h), node_id, node_type
END
```

---

## State Messages (MQTT)

### Energy Node State

```json
{
  "node_id": "solar-roof",
  "timestamp": "2026-01-12T12:30:00Z",
  "type": "solar",
  "state": {
    "power_w": 4523.5,
    "energy_today_wh": 18450,
    "energy_total_wh": 4523890,
    "voltage_v": [234.2],
    "current_a": [19.3],
    "power_factor": 0.99,
    "temperature_c": 45.2,
    "status": "generating"
  }
}
```

**Topic:** `graylogic/state/energy/{node_id}`

### Energy Balance Summary

```json
{
  "timestamp": "2026-01-12T12:30:00Z",
  "balance": {
    "solar_w": 4523,
    "battery_w": 1500,
    "battery_soc": 65,
    "ev_w": 0,
    "grid_w": -1023,
    "load_w": 4000,
    "self_consumption_pct": 77.4,
    "autarky_pct": 74.4
  }
}
```

**Topic:** `graylogic/state/energy/balance`

---

## Related Documents

- [Data Model: Entities](../data-model/entities.md) — DeviceAssociation for energy attribution
- [Modbus Protocol Specification](../protocols/modbus.md) — Meter/inverter communication
- [Core Internals Architecture](core-internals.md) — Energy service integration
- [PHM Specification](../intelligence/phm.md) — Equipment health monitoring using energy data
- [Climate Domain Specification](../domains/climate.md) — HVAC load management
- [OCPP Protocol Specification](../protocols/ocpp.md) — EV charging protocol

