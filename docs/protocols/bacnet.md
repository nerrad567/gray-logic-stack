---
title: BACnet Protocol Integration
version: 0.1.0
status: roadmap
last_updated: 2026-01-12
depends_on:
  - architecture/bridge-interface.md
  - data-model/entities.md
roadmap: Year 2
---

# BACnet Protocol Integration

> **Status**: Year 2 Roadmap — This document outlines planned BACnet integration for commercial HVAC systems. Implementation will begin after core protocols (KNX, DALI, Modbus) are stable.

---

## Overview

BACnet (Building Automation and Control Networks) is the dominant protocol for commercial building automation, particularly HVAC systems. Gray Logic will integrate BACnet to support:

- Commercial office HVAC systems
- Large residential plant rooms with BACnet-native equipment
- Integration with existing BMS (Building Management Systems)
- Air handling units, chillers, and packaged rooftop units

### Why BACnet?

| Scenario | Primary Protocol | BACnet Role |
|----------|------------------|-------------|
| **Residential** | KNX + Modbus | Not typically needed |
| **Light Commercial** | KNX + Modbus | Optional for packaged HVAC |
| **Commercial Office** | KNX (lighting) + BACnet (HVAC) | Primary for HVAC |
| **BMS Integration** | Existing BMS | Gray Logic as supervisor |

### Relationship to Modbus

Many devices support both Modbus and BACnet. Selection criteria:

- **Use Modbus when**: Device is standalone, simple register-based control, residential/light commercial
- **Use BACnet when**: Device is part of larger BACnet network, commercial HVAC, need COV subscriptions, integration with existing BMS

---

## Protocol Variants

### BACnet/IP

Primary variant for Gray Logic integration.

```
Gray Logic Core
      │
      ▼ UDP/IP (port 47808)
┌─────────────────────────┐
│     BACnet/IP Network   │
├─────────────────────────┤
│  AHU-1    Chiller-1     │
│  VAV-01   VAV-02        │
│  FCU-01   FCU-02        │
└─────────────────────────┘
```

**Characteristics:**
- Native IP communication
- UDP port 47808 (0xBAC0)
- Broadcast for discovery (Who-Is/I-Am)
- Direct device communication
- COV (Change of Value) subscriptions

### BACnet MS/TP

Serial variant requiring a gateway.

```
Gray Logic Core
      │
      ▼ BACnet/IP
┌─────────────────────────┐
│   BACnet/IP to MS/TP    │
│       Router/Gateway    │
└──────────┬──────────────┘
           │ RS-485
           ▼
┌─────────────────────────┐
│    BACnet MS/TP Bus     │
├─────────────────────────┤
│  Thermostat  VAV Box    │
│  Sensor      Actuator   │
└─────────────────────────┘
```

**Characteristics:**
- RS-485 physical layer
- Token-passing protocol
- Common for field devices
- Requires IP router for Gray Logic

---

## BACnet Object Model

BACnet uses an object-oriented model. Key object types for Gray Logic:

### Device Object

Every BACnet device has exactly one Device object.

```yaml
Device:
  object_identifier: device,100
  object_name: "AHU-1"
  vendor_identifier: 123
  model_name: "Air Handler Model X"
  firmware_revision: "2.1.0"
  protocol_version: 1
  protocol_revision: 14
```

### Analog Objects

| Object Type | Use Case | Properties |
|-------------|----------|------------|
| `analog-input` | Sensor reading | present_value, units, out_of_service |
| `analog-output` | Setpoint, actuator position | present_value, priority_array |
| `analog-value` | Configuration, calculated | present_value, units |

### Binary Objects

| Object Type | Use Case | Properties |
|-------------|----------|------------|
| `binary-input` | Status, alarm | present_value, polarity |
| `binary-output` | On/off command | present_value, priority_array |
| `binary-value` | Configuration | present_value |

### Multi-State Objects

| Object Type | Use Case | Properties |
|-------------|----------|------------|
| `multi-state-input` | Mode status | present_value, state_text |
| `multi-state-output` | Mode command | present_value, number_of_states |
| `multi-state-value` | Configuration | present_value |

### Schedule and Calendar Objects

| Object Type | Use Case |
|-------------|----------|
| `schedule` | Time-based setpoint changes |
| `calendar` | Holiday/exception definitions |

---

## Gray Logic BACnet Bridge

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    GRAY LOGIC CORE                       │
│                                                          │
│  Device Registry ◄──── BACnet devices as Gray Logic     │
│                        Device entities                   │
│  State Manager ◄────── Present values mapped to state   │
│                                                          │
└────────────────────────────┬────────────────────────────┘
                             │ MQTT
                             ▼
┌─────────────────────────────────────────────────────────┐
│                    BACNET BRIDGE                         │
│                                                          │
│  ┌─────────────────┐  ┌─────────────────┐               │
│  │ Device Discovery│  │ Object Browser  │               │
│  │   (Who-Is/I-Am) │  │  (Read Property)│               │
│  └─────────────────┘  └─────────────────┘               │
│                                                          │
│  ┌─────────────────┐  ┌─────────────────┐               │
│  │  COV Subscriber │  │ Command Writer  │               │
│  │ (state updates) │  │(WriteProperty)  │               │
│  └─────────────────┘  └─────────────────┘               │
│                                                          │
│  ┌─────────────────────────────────────────┐            │
│  │            BACnet Stack                  │            │
│  │  (bacnet-stack or similar open source)  │            │
│  └─────────────────────────────────────────┘            │
│                                                          │
└────────────────────────────┬────────────────────────────┘
                             │ UDP/47808
                             ▼
                      BACnet/IP Network
```

### MQTT Topic Structure

Following the standard bridge interface:

```
# State updates (Bridge → Core)
graylogic/state/bacnet/{device_instance}/{object_type}/{object_instance}

# Commands (Core → Bridge)
graylogic/command/bacnet/{device_instance}/{object_type}/{object_instance}

# Discovery
graylogic/discovery/bacnet/device/{device_instance}

# Bridge status
graylogic/bridge/bacnet/status
```

### Example Messages

**State Update (COV notification):**
```json
{
  "timestamp": "2026-01-12T14:30:00Z",
  "object": "analog-input,1",
  "present_value": 21.5,
  "units": "degrees-celsius",
  "status_flags": {
    "in_alarm": false,
    "fault": false,
    "overridden": false,
    "out_of_service": false
  }
}
```

**Command (Write setpoint):**
```json
{
  "timestamp": "2026-01-12T14:30:05Z",
  "object": "analog-value,1",
  "present_value": 22.0,
  "priority": 8
}
```

---

## Device Profiles

### Air Handling Unit (AHU)

Typical BACnet objects for an AHU:

| Object | Type | Description |
|--------|------|-------------|
| `AI,1` | analog-input | Supply air temperature |
| `AI,2` | analog-input | Return air temperature |
| `AI,3` | analog-input | Outside air temperature |
| `AI,4` | analog-input | Mixed air temperature |
| `AI,5` | analog-input | Supply air pressure |
| `AI,6` | analog-input | Filter differential pressure |
| `AO,1` | analog-output | Cooling valve position |
| `AO,2` | analog-output | Heating valve position |
| `AO,3` | analog-output | Supply fan speed |
| `AO,4` | analog-output | Return fan speed |
| `AO,5` | analog-output | Outside air damper |
| `BI,1` | binary-input | Supply fan status |
| `BI,2` | binary-input | Return fan status |
| `BI,3` | binary-input | Filter alarm |
| `BI,4` | binary-input | Freeze stat |
| `BO,1` | binary-output | Unit enable |
| `MSI,1` | multi-state-input | Operating mode status |
| `MSO,1` | multi-state-output | Operating mode command |
| `AV,1` | analog-value | Supply air setpoint |
| `AV,2` | analog-value | Economizer setpoint |

### Variable Air Volume (VAV) Box

| Object | Type | Description |
|--------|------|-------------|
| `AI,1` | analog-input | Zone temperature |
| `AI,2` | analog-input | Discharge air temperature |
| `AI,3` | analog-input | Airflow |
| `AO,1` | analog-output | Damper position |
| `AO,2` | analog-output | Reheat valve |
| `AV,1` | analog-value | Cooling setpoint |
| `AV,2` | analog-value | Heating setpoint |
| `AV,3` | analog-value | Airflow setpoint |
| `BI,1` | binary-input | Occupancy |
| `MSI,1` | multi-state-input | Operating mode |

### Chiller

| Object | Type | Description |
|--------|------|-------------|
| `AI,1` | analog-input | Leaving water temperature |
| `AI,2` | analog-input | Entering water temperature |
| `AI,3` | analog-input | Condenser water temperature |
| `AI,4` | analog-input | Refrigerant pressure |
| `AI,5` | analog-input | Power consumption |
| `AO,1` | analog-output | Capacity setpoint |
| `BI,1` | binary-input | Compressor 1 status |
| `BI,2` | binary-input | Compressor 2 status |
| `BI,3` | binary-input | Chiller alarm |
| `BO,1` | binary-output | Chiller enable |
| `AV,1` | analog-value | Leaving water setpoint |
| `MSI,1` | multi-state-input | Operating mode |

---

## Priority Array

BACnet uses a 16-level priority array for commandable objects:

| Priority | Use |
|----------|-----|
| 1 | Manual-Life Safety |
| 2 | Automatic-Life Safety |
| 3 | Available |
| 4 | Available |
| 5 | Critical Equipment Control |
| 6 | Minimum On/Off |
| 7 | Available |
| 8 | **Manual Operator (Gray Logic UI)** |
| 9 | Available |
| 10 | Available |
| 11 | Available |
| 12 | Available |
| 13 | Available |
| 14 | Available |
| 15 | Available |
| 16 | **Default/Scheduling** |

Gray Logic typically writes at:
- **Priority 8**: Manual commands from UI
- **Priority 16**: Scheduled/automated commands

---

## Discovery and Commissioning

### Device Discovery

```
1. Bridge sends Who-Is broadcast
2. Devices respond with I-Am
3. Bridge reads Device object properties
4. Bridge reads object list
5. Bridge reads key properties of each object
6. Gray Logic creates Device entities
```

### Object Mapping

Commissioning tool maps BACnet objects to Gray Logic capabilities:

```yaml
device_id: "ahu-01"
name: "Air Handling Unit 1"
protocol: "bacnet_ip"
address:
  device_instance: 100
  ip: "192.168.1.50"

capabilities:
  temperature_read:
    object: "analog-input,1"
    property: "present_value"
    units: "degrees-celsius"
  
  setpoint:
    object: "analog-value,1"
    property: "present_value"
    writeable: true
    priority: 8
  
  run_stop:
    object: "binary-output,1"
    property: "present_value"
    writeable: true
    priority: 8
  
  fault_status:
    object: "binary-input,3"
    property: "present_value"
    invert: false
```

---

## COV Subscriptions

Change of Value (COV) provides efficient state updates without polling.

### Subscription Types

| Type | Use Case |
|------|----------|
| **COV** | Standard COV with increment |
| **COV-Property** | Subscribe to specific property |
| **Unconfirmed COV** | No acknowledgment required |

### Gray Logic COV Strategy

1. Subscribe to critical values (temperatures, setpoints, alarms)
2. Poll less critical values periodically
3. Re-subscribe before expiry (typical lifetime: 300s)
4. Handle subscription failures gracefully

---

## Minimum Viable Implementation (Year 2)

The following defines the smallest useful BACnet integration:

### Scope
- **BACnet/IP only** — No MS/TP router support initially
- **Client mode only** — Gray Logic as BACnet client, not server
- **Manual commissioning** — No auto-discovery in initial release

### Required Capabilities

```yaml
minimum_viable:
  # Discovery
  discovery:
    method: "manual"                # Enter device IP + instance manually
    auto_who_is: false              # Year 2+ enhancement

  # Read support (required)
  read:
    services:
      - "ReadProperty"              # Single property read
      - "ReadPropertyMultiple"      # Batch reads (performance)
    object_types:
      - "analog-input"              # AI - sensors
      - "analog-output"             # AO - actuator positions
      - "analog-value"              # AV - setpoints, calculated
      - "binary-input"              # BI - status, alarms
      - "binary-output"             # BO - on/off commands
    properties:
      - "present_value"             # Current value
      - "object_name"               # Human-readable name
      - "units"                     # Engineering units
      - "status_flags"              # Alarm/fault/override/out-of-service

  # Write support (required)
  write:
    services:
      - "WriteProperty"             # Single property write
    priority: 8                     # Manual operator priority
    object_types:
      - "analog-output"
      - "analog-value"
      - "binary-output"

  # Polling (required - COV is Year 2+)
  polling:
    default_interval_seconds: 30    # Read all monitored objects
    fast_poll_seconds: 5            # For critical values
    slow_poll_seconds: 300          # For rarely-changing values

  # NOT in minimum viable (Year 2+ enhancements)
  deferred:
    - "COV subscriptions"           # Requires subscription management
    - "Who-Is discovery"            # Requires broadcast handling
    - "Alarm/Event handling"        # Complex, defer
    - "Trend log access"            # Nice-to-have
    - "Schedule objects"            # Gray Logic has its own scheduler
    - "BACnet server mode"          # Exposing data to other BMS
```

### Example: Single VAV Box Integration

Minimum viable supports this use case:

```yaml
# /etc/graylogic/bacnet/devices/vav-01.yaml
device_id: "vav-meeting-room"
name: "VAV Box - Meeting Room"
protocol: "bacnet_ip"

connection:
  ip: "192.168.10.50"
  port: 47808
  device_instance: 1001

objects:
  # Read-only sensors
  - capability: "temperature_read"
    object_type: "analog-input"
    instance: 1
    poll_interval: 30

  - capability: "airflow_read"
    object_type: "analog-input"
    instance: 3
    poll_interval: 30

  # Writable setpoints
  - capability: "setpoint"
    object_type: "analog-value"
    instance: 1
    writeable: true
    write_priority: 8

  # On/off control
  - capability: "occupancy_override"
    object_type: "binary-value"
    instance: 1
    writeable: true
```

### Success Criteria for MVP

1. Can manually add a BACnet/IP device by IP address
2. Can read temperature from analog-input
3. Can write setpoint to analog-value
4. State updates appear in Gray Logic UI within poll interval
5. Commands from Gray Logic change device state

---

## Implementation Roadmap

### Phase 1: Read-Only Integration
- Device discovery (Who-Is/I-Am)
- Read Property for polling
- Basic object type support (AI, AO, AV, BI, BO, BV)
- State mapping to Gray Logic

### Phase 2: Write Support
- WriteProperty for setpoints
- Priority array handling
- Multi-state objects
- Schedule reading

### Phase 3: Advanced Features
- COV subscriptions
- Alarm and event handling
- Trend log access
- Schedule writing

### Phase 4: BMS Supervisor Mode
- Act as BACnet client to existing BMS
- Expose Gray Logic data as BACnet server
- BACnet routing between networks

---

## Open Source Options

Potential BACnet stack libraries:

| Library | Language | License | Notes |
|---------|----------|---------|-------|
| bacnet-stack | C | GPL-2.0 | Mature, comprehensive |
| BACpypes | Python | MIT | Good for prototyping |
| bacnet-go | Go | MIT | Native Go, less mature |
| node-bacstack | Node.js | MIT | JavaScript/TypeScript |

Recommended: Evaluate `bacnet-go` first for native integration, fall back to `bacnet-stack` via CGo if needed.

---

## Security Considerations

### BACnet/SC (Secure Connect)

BACnet/SC adds TLS-based security. Consider for:
- Multi-tenant buildings
- Integration over untrusted networks
- Compliance requirements

### Network Segmentation

```
┌─────────────────────────────────────┐
│ CONTROL VLAN                        │
│  • Gray Logic Server                │
│  • BACnet Bridge                    │
└───────────────┬─────────────────────┘
                │ Firewall (UDP 47808)
                ▼
┌─────────────────────────────────────┐
│ BACnet VLAN                         │
│  • AHUs, Chillers, VAVs             │
│  • BACnet/IP Router                 │
└─────────────────────────────────────┘
```

---

## Related Documents

- [Bridge Interface](../architecture/bridge-interface.md) — Standard bridge MQTT contract
- [Modbus Protocol](modbus.md) — Alternative for simpler devices
- [Climate Domain](../domains/climate.md) — HVAC control logic
- [Plant Domain](../domains/plant.md) — Plant room equipment management
- [Entities](../data-model/entities.md) — Device type definitions

---

## Appendix: BACnet Object Type Reference

| ID | Object Type | Abbreviation |
|----|-------------|--------------|
| 0 | analog-input | AI |
| 1 | analog-output | AO |
| 2 | analog-value | AV |
| 3 | binary-input | BI |
| 4 | binary-output | BO |
| 5 | binary-value | BV |
| 8 | device | DEV |
| 13 | multi-state-input | MSI |
| 14 | multi-state-output | MSO |
| 17 | schedule | SCHED |
| 6 | calendar | CAL |
| 19 | multi-state-value | MSV |
| 10 | file | FILE |
| 20 | trend-log | TL |

