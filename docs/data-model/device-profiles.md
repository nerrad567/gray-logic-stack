---
title: Device Profiles
version: 1.0.0
status: active
last_updated: 2026-01-17
depends_on:
  - data-model/entities.md
  - architecture/bridge-interface.md
---

# Device Profiles

This document defines the "Device Profile" (or Driver) abstraction, which bridges the gap between raw protocol data (like Modbus registers) and the high-level Gray Logic `Device` entity.

---

## The Problem

Complex plant equipment (Heat Pumps, Solar Inverters, Chillers, AHUs) often expose dozens or hundreds of raw data points via protocols like Modbus or BACnet.

Manually mapping every register for every device instance is:
1.  **Error-prone:** Typos in register addresses cause dangerous behavior.
2.  **Unmaintainable:** Replacing a unit with a slightly different model requires re-entering 50+ mappings.
3.  **Inconsistent:** One integrator might map "Flow Temp" to `temp_flow`, another to `flow_temperature`.

## The Solution: Device Profiles

A **Device Profile** is a standardized, reusable template that maps a specific hardware model's protocol interface to Gray Logic's standardized Capabilities.

**Concept:**
```
[ Physical Device ]  <-- (Protocol) -->  [ Device Profile ]  <-- (Normalization) -->  [ Gray Logic Device ]
   Grundfos Magna3                        Mappings:                                     Type: pump
   Modbus Address 1                       Reg 100 -> speed_read                         Capability: speed_read
                                          Reg 200 -> speed_set
```

## Profile Structure

A profile is a YAML or JSON file containing:
1.  **Metadata:** Manufacturer, Model, Version.
2.  **Protocol Config:** Default connection settings (baud rate, parity).
3.  **Mappings:** Rules for translating protocol data points to Capability properties.

### Example: Modbus Profile (Grundfos Magna3)

```yaml
id: "profile-grundfos-magna3"
manufacturer: "Grundfos"
model: "Magna3"
protocol: "modbus_rtu"
description: "Circulator pump with Modbus module"

# Default settings (can be overridden per device)
defaults:
  baud_rate: 19200
  parity: "even"
  stop_bits: 1

# Capabilities this profile supports
capabilities:
  - "run_stop"
  - "speed_control"
  - "flow_read"
  - "power_read"

# Register Mappings
mappings:
  # Control
  - capability: "run_stop"
    property: "running"
    direction: "read_write"
    register:
      type: "coil"
      address: 100
      
  - capability: "speed_control"
    property: "speed_percent"
    direction: "read_write"
    register:
      type: "holding"
      address: 200
    transform:
      scale: 0.01          # Raw 5000 -> 50.00%
      
  # Monitoring
  - capability: "flow_read"
    property: "flow_rate"
    direction: "read_only"
    register:
      type: "input"
      address: 300
    transform:
      scale: 0.1           # Raw 150 -> 15.0 m3/h
      
  - capability: "power_read"
    property: "power_w"
    direction: "read_only"
    register:
      type: "input"
      address: 302
```

## How It Works

1.  **Selection:** During commissioning, the user selects the "Grundfos Magna3" profile for a new device.
2.  **Instantiation:** The system creates a `Device` entity.
    - `protocol` is set to `modbus_rtu`.
    - `capabilities` are populated from the profile.
    - `config.profile_id` links to the profile.
3.  **Runtime:**
    - The **Modbus Bridge** loads the profile.
    - It polls the registers defined in `mappings`.
    - It applies `transform` rules (scaling, offset, bitmasks).
    - It publishes normalized state to MQTT: `{"flow_rate": 15.0}`.
    - It accepts normalized commands `{"speed_percent": 50}` and writes to register `200`.

## Profile Repository

Profiles are stored in a version-controlled repository (Git) distributed with the Core.

Structure:
```
profiles/
  ├── modbus/
  │   ├── grundfos/
  │   │   └── magna3.yaml
  │   └── daikin/
  │       └── vrv_interface.yaml
  ├── knx/
  │   └── abb/
  │       └── sua_4_16_1.yaml  # Often handled via ETS import, but useful for complex gateways
  └── bacnet/
      └── siemens/
          └── pxc.yaml
```

## User Customization

Users can extend or override profiles:
1.  **User Profiles:** Stored in `/etc/graylogic/profiles/user/`.
2.  **Overrides:** A device instance can override specific mappings in its own `config` JSON (e.g., if a specific installation uses a non-standard register).

## Benefits

1.  **Plug-and-Play:** Adding a complex boiler is as simple as "Select Model -> Enter IP Address".
2.  **Consistency:** All "Magna3" pumps in the system behave identically.
3.  **Community:** Users can share profiles for new hardware.
4.  **Hardware Agnostic:** Core logic (Rules, Scenes) interacts with the `speed_percent` capability, never knowing or caring that it's a Modbus register `200` scaled by `0.01`.
