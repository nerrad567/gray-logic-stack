---
title: DALI Protocol Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - architecture/bridge-interface.md
  - protocols/knx.md
---

# DALI Protocol Specification

This document specifies how Gray Logic integrates with DALI (Digital Addressable Lighting Interface) for professional lighting control.

---

## Overview

### What is DALI?

DALI is an international standard (IEC 62386) specifically designed for lighting control:

- **Addressable** — Each device has a unique address (0-63)
- **Two-way communication** — Query device state, detect failures
- **Standardized commands** — Consistent across manufacturers
- **Scene storage** — Devices store scenes locally
- **Groups** — Up to 16 groups for simultaneous control

### DALI vs DALI-2

| Feature | DALI | DALI-2 |
|---------|------|--------|
| Standard | IEC 62386-1/2 | IEC 62386-1xx series |
| Interoperability | Basic | Certification required |
| Device types | Drivers only | Drivers, sensors, buttons |
| Diagnostics | Limited | Extended (Part 103) |
| Color control | Vendor-specific | Standardized (Parts 209, 102) |
| Certification | Optional | Mandatory for logo |

**Gray Logic targets DALI-2** for best interoperability, with backwards compatibility for legacy DALI-1 devices.

### Why DALI for Gray Logic?

| Requirement | DALI Capability |
|-------------|-----------------|
| **Superior dimming** | Logarithmic curve, flicker-free, precise |
| **Device monitoring** | Lamp failure detection, runtime hours |
| **Open standard** | IEC 62386, multi-vendor |
| **Professional quality** | Commercial and high-end residential |
| **KNX complementary** | KNX for switching, DALI for lighting control |

### DALI and KNX Integration

Typical high-end installation uses both:

```
┌─────────────────────────────────────────────────────────────────┐
│                         KNX BUS                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐ │
│  │ Switch  │──│ Sensor  │──│ Blind   │──│  DALI Gateway       │ │
│  └─────────┘  └─────────┘  └─────────┘  │  (KNX/DALI)         │ │
│                                          └──────────┬──────────┘ │
└─────────────────────────────────────────────────────┼───────────┘
                                                      │
                                               ┌──────▼──────┐
                                               │  DALI BUS   │
                                               │  (2-wire)   │
                                               └──────┬──────┘
                                                      │
                               ┌──────────┬───────────┼───────────┐
                               │          │           │           │
                          ┌────▼────┐ ┌───▼───┐ ┌────▼────┐ ┌────▼────┐
                          │Driver 1 │ │Driver 2│ │Driver 3 │ │Sensor   │
                          │(addr 0) │ │(addr 1)│ │(addr 2) │ │(addr 60)│
                          └─────────┘ └────────┘ └─────────┘ └─────────┘
```

For Gray Logic direct DALI control (without KNX):

```
┌─────────────────────────────────────────────────────────────────┐
│                     GRAY LOGIC SERVER                            │
│  ┌─────────────────┐                                            │
│  │   DALI Bridge   │                                            │
│  └────────┬────────┘                                            │
└───────────┼─────────────────────────────────────────────────────┘
            │
     ┌──────▼──────┐
     │ DALI/IP     │  (or DALI/USB, DALI/RS485)
     │ Gateway     │
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │  DALI BUS   │
     └──────┬──────┘
            │
    ┌───────┼───────┬───────────┐
    │       │       │           │
 ┌──▼──┐ ┌──▼──┐ ┌──▼──┐    ┌──▼──┐
 │ D0  │ │ D1  │ │ D2  │....│ D63 │
 └─────┘ └─────┘ └─────┘    └─────┘
```

---

## DALI Fundamentals

### Physical Layer

| Parameter | Value |
|-----------|-------|
| **Wiring** | 2-wire, polarity independent |
| **Voltage** | 16V DC (9.5V-22.5V) |
| **Bus Power** | Up to 250mA |
| **Topology** | Bus, star, tree, or mixed |
| **Max Length** | 300m (voltage drop dependent) |
| **Max Devices** | 64 per bus |
| **Max Groups** | 16 per bus |
| **Max Scenes** | 16 per device |
| **Data Rate** | 1200 baud |

### Addressing

| Address Type | Range | Use |
|--------------|-------|-----|
| **Short Address** | 0-63 | Individual device |
| **Group Address** | 0-15 | Group of devices |
| **Broadcast** | 255 | All devices |

### Device Types (IEC 62386-2xx series)

| Part | Type | Description |
|------|------|-------------|
| 201 | 0 | Fluorescent lamps |
| 202 | 1 | Self-contained emergency lighting |
| 203 | 2 | Discharge lamps (HID) |
| 204 | 3 | Low-voltage halogen |
| 205 | 4 | Supply voltage regulator for incandescent |
| 206 | 5 | Conversion from digital to DC voltage |
| 207 | 6 | LED modules |
| 208 | 7 | Switching function (relay) |
| 209 | 8 | Color control (Tc, RGBWAF) |

### Control Gear Features (Part 207 - LED)

| Feature | Description |
|---------|-------------|
| **Arc Power** | Actual power level (0-254) |
| **Fade Time** | Transition duration |
| **Fade Rate** | Transition speed |
| **Power On Level** | Level on power-up |
| **System Failure Level** | Level if communication lost |
| **Min/Max Level** | Operating range limits |
| **Physical Minimum** | Hardware minimum |
| **Group Membership** | 16 groups, bitmap |
| **Scene Levels** | 16 stored scenes |

### Color Control (Part 209 - Device Type 8)

| Sub-type | Features | Variables |
|----------|----------|-----------|
| **Tc** | Tunable white | Color temperature (mirek) |
| **xy** | CIE 1931 | x, y coordinates |
| **Primary N** | Direct primary control | Per-primary levels |
| **RGBWAF** | RGB + extras | R, G, B, W, A, F channels |

---

## DALI Commands

### Arc Power Commands

| Command | Code | Description |
|---------|------|-------------|
| `OFF` | 0x00 | Immediate off |
| `UP` | 0x01 | Increase by fade rate |
| `DOWN` | 0x02 | Decrease by fade rate |
| `STEP UP` | 0x03 | Increase one step |
| `STEP DOWN` | 0x04 | Decrease one step |
| `RECALL MAX LEVEL` | 0x05 | Go to max level |
| `RECALL MIN LEVEL` | 0x06 | Go to min level |
| `STEP DOWN AND OFF` | 0x07 | Step down or off if min |
| `ON AND STEP UP` | 0x08 | On and step up |
| `GO TO LAST ACTIVE LEVEL` | 0x0A | Recall last non-zero |
| `DAPC` (level) | 0xFE + level | Direct arc power control |
| `GO TO SCENE` (0-15) | 0x10-0x1F | Recall scene |

### Configuration Commands

| Command | Code | Description |
|---------|------|-------------|
| `RESET` | 0x20 | Factory reset |
| `STORE DTR AS MAX` | 0x2A | Set max level |
| `STORE DTR AS MIN` | 0x2B | Set min level |
| `STORE DTR AS SYSTEM FAILURE` | 0x2C | Set failure level |
| `STORE DTR AS POWER ON` | 0x2D | Set power-on level |
| `STORE DTR AS FADE TIME` | 0x2E | Set fade time |
| `STORE DTR AS FADE RATE` | 0x2F | Set fade rate |
| `STORE DTR AS SCENE` (0-15) | 0x40-0x4F | Store scene level |
| `ADD TO GROUP` (0-15) | 0x60-0x6F | Add to group |
| `REMOVE FROM GROUP` (0-15) | 0x70-0x7F | Remove from group |

### Query Commands

| Command | Code | Response |
|---------|------|----------|
| `QUERY STATUS` | 0x90 | Status byte |
| `QUERY CONTROL GEAR PRESENT` | 0x91 | Yes (0xFF) |
| `QUERY LAMP FAILURE` | 0x92 | Yes/No |
| `QUERY LAMP POWER ON` | 0x93 | Yes/No |
| `QUERY ACTUAL LEVEL` | 0xA0 | Current level |
| `QUERY MAX LEVEL` | 0xA1 | Max level |
| `QUERY MIN LEVEL` | 0xA2 | Min level |
| `QUERY POWER ON LEVEL` | 0xA3 | Power-on level |
| `QUERY SYSTEM FAILURE LEVEL` | 0xA4 | Failure level |
| `QUERY FADE TIME/RATE` | 0xA5 | Fade settings |
| `QUERY SCENE LEVEL` (0-15) | 0xB0-0xBF | Stored scene level |
| `QUERY GROUPS 0-7` | 0xC0 | Group membership bitmap |
| `QUERY GROUPS 8-15` | 0xC1 | Group membership bitmap |
| `QUERY DEVICE TYPE` | 0x99 | Device type (0-8) |

### Color Commands (Device Type 8)

| Command | Description |
|---------|-------------|
| `SET TEMPORARY COLOUR TEMPERATURE TC` | Set Tc (mirek in DTR) |
| `ACTIVATE` | Apply temporary color |
| `SET TEMPORARY X COORDINATE` | Set CIE x |
| `SET TEMPORARY Y COORDINATE` | Set CIE y |
| `SET TEMPORARY PRIMARY N DIMLEVEL` | Set primary level |
| `SET TEMPORARY RGB DIMLEVEL` | Set RGB levels |

---

## DALI Gateway Requirements

### Gateway Types

| Type | Interface | Use Case |
|------|-----------|----------|
| **DALI/IP** | Ethernet | Primary for Gray Logic |
| **DALI/KNX** | KNX bus | When KNX is primary |
| **DALI/USB** | USB | Development, small systems |
| **DALI/RS485** | RS-485 | Legacy, industrial |

### Recommended Gateways

| Manufacturer | Model | Channels | Interface | Notes |
|--------------|-------|----------|-----------|-------|
| Tridonic | DALI Gateway 3-IP | 3 buses | IP, REST API | Excellent API |
| Helvar | Imagine Router 920 | 4 buses | IP, proprietary | Enterprise |
| Lunatone | DALI Gateway 2 | 2 buses | IP, Modbus TCP | Good Modbus |
| Wago | 753-647 | 2 buses | Modbus TCP | PLC integration |
| Zencontrol | ZC-DP | 1 bus | IP, MQTT | MQTT native |

### Gateway Features Required

| Feature | Requirement |
|---------|-------------|
| **Multi-bus** | 2+ DALI buses per gateway |
| **IP connectivity** | Ethernet, static IP |
| **API access** | REST, Modbus TCP, or MQTT |
| **Addressing** | Commission addresses remotely |
| **Group management** | Create/modify groups via API |
| **Scene storage** | Store scenes via API |
| **Status queries** | Query device status |
| **Event reporting** | Push state changes |
| **Diagnostics** | Lamp failure, error reporting |

---

## DALI Bridge Specification

### Responsibilities

| Direction | Source | Destination | Action |
|-----------|--------|-------------|--------|
| **Inbound** | DALI gateway | MQTT | Convert status → state message |
| **Outbound** | MQTT | DALI gateway | Convert command → DALI command |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       DALI BRIDGE                                │
│                                                                  │
│  ┌────────────────┐     ┌──────────────────┐     ┌────────────┐ │
│  │ Gateway Client │────▶│  Message Router  │────▶│   MQTT     │ │
│  │ (Modbus/REST)  │◀────│  & Translator    │◀────│  Client    │ │
│  └────────────────┘     └──────────────────┘     └────────────┘ │
│          │                       │                      │       │
│          │              ┌────────▼────────┐             │       │
│          │              │ Address Map     │             │       │
│          │              │ (Bus.Addr →     │             │       │
│          │              │  Device)        │             │       │
│          │              └─────────────────┘             │       │
│          ▼                       ▼                      ▼       │
│    DALI Gateway         Configuration DB         MQTT Broker    │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

```yaml
# dali-bridge-config.yaml
bridge:
  id: "dali-bridge-01"
  mqtt:
    broker: "tcp://localhost:1883"
    client_id: "dali-bridge-01"
    topic_prefix: "graylogic"

gateways:
  - id: "dali-gw-01"
    type: "tridonic"
    host: "192.168.1.110"
    port: 502  # Modbus TCP
    buses:
      - bus: 1
        name: "Ground Floor Lighting"
      - bus: 2
        name: "First Floor Lighting"

devices:
  - device_id: "light-kitchen-island"
    gateway: "dali-gw-01"
    bus: 1
    address: 0
    type: "light_dimmer"
    device_type: 6  # LED
    groups: [0, 5]
    features:
      dimming: true
      color_temp: false

  - device_id: "light-living-ceiling"
    gateway: "dali-gw-01"
    bus: 1
    address: 1
    type: "light_ct"
    device_type: 8  # Color control
    groups: [1]
    features:
      dimming: true
      color_temp: true
      color_temp_range:
        min_mirek: 250  # 4000K
        max_mirek: 400  # 2500K

  - device_id: "light-living-rgb-accent"
    gateway: "dali-gw-01"
    bus: 1
    address: 2
    type: "light_rgb"
    device_type: 8
    features:
      dimming: true
      rgb: true

# Group mappings (DALI groups to Gray Logic)
groups:
  - group_id: "group-kitchen-all"
    gateway: "dali-gw-01"
    bus: 1
    dali_group: 0
    devices: ["light-kitchen-island", "light-kitchen-counter", "light-kitchen-pendant"]

  - group_id: "group-living-all"
    gateway: "dali-gw-01"
    bus: 1
    dali_group: 1
    devices: ["light-living-ceiling", "light-living-rgb-accent", "light-living-floor"]
```

### MQTT Topics

**State updates (DALI → Core):**
```
graylogic/bridge/dali-bridge-01/state/{device_id}
```

**Commands (Core → DALI):**
```
graylogic/bridge/dali-bridge-01/command/{device_id}
```

**Group commands:**
```
graylogic/bridge/dali-bridge-01/command/group/{group_id}
```

### Message Formats

**State update:**

```yaml
topic: graylogic/bridge/dali-bridge-01/state/light-living-ceiling
payload:
  device_id: "light-living-ceiling"
  timestamp: "2026-01-12T14:30:00Z"
  source: "dali"
  state:
    on: true
    brightness: 80
    color_temp_mirek: 300  # 3333K
  diagnostics:
    lamp_failure: false
    power_on: true
    actual_level: 204  # Raw DALI level (0-254)
```

**Command message:**

```yaml
topic: graylogic/bridge/dali-bridge-01/command/light-living-ceiling
payload:
  device_id: "light-living-ceiling"
  command: "set"
  parameters:
    on: true
    brightness: 50
    color_temp_kelvin: 3000  # Converted to mirek
    transition_ms: 2000  # Converted to fade time
  request_id: "req-67890"
```

**Scene recall command:**

```yaml
topic: graylogic/bridge/dali-bridge-01/command/group/group-living-all
payload:
  group_id: "group-living-all"
  command: "scene"
  parameters:
    scene: 3  # DALI scene 0-15
  request_id: "req-67891"
```

### Level Conversion

DALI uses logarithmic levels for perceptual linearity:

| DALI Level | Approx % | Actual Light Output |
|------------|----------|---------------------|
| 254 | 100% | 100% |
| 229 | 90% | 50% |
| 204 | 80% | 25% |
| 178 | 70% | 12.5% |
| 153 | 60% | 6.25% |
| 1 | 0.4% | 0.1% |
| 0 | 0% | Off |

**Conversion formula:**

```
DALI level = 10 * ln(percent / 100) / ln(10) * 253 / 3 + 1
percent = 10 ^ (3 * (DALI level - 1) / 253) * 100
```

The bridge handles conversion so Gray Logic Core uses percentages (0-100%).

### Fade Time Mapping

DALI fade times are discrete values:

| Fade Time Code | Duration |
|----------------|----------|
| 0 | 0 (instant) |
| 1 | 0.7s |
| 2 | 1.0s |
| 3 | 1.4s |
| 4 | 2.0s |
| 5 | 2.8s |
| 6 | 4.0s |
| 7 | 5.7s |
| 8 | 8.0s |
| 9 | 11.3s |
| 10 | 16.0s |
| 11 | 22.6s |
| 12 | 32.0s |
| 13 | 45.3s |
| 14 | 64.0s |
| 15 | 90.5s |

Bridge maps requested transition time to nearest DALI fade time.

### Color Temperature Conversion

DALI uses mirek (micro reciprocal degrees):

```
mirek = 1,000,000 / kelvin
kelvin = 1,000,000 / mirek
```

| Kelvin | Mirek | Description |
|--------|-------|-------------|
| 6500K | 154 | Daylight |
| 5000K | 200 | Cool white |
| 4000K | 250 | Neutral white |
| 3000K | 333 | Warm white |
| 2700K | 370 | Warm |
| 2200K | 455 | Very warm |

---

## Commissioning

### Initial Setup

1. **Physical installation** — Wire DALI bus, connect drivers
2. **Address assignment** — Use gateway or commissioning tool
3. **Group assignment** — Assign drivers to groups (0-15)
4. **Scene storage** — Store scene levels in drivers
5. **Gateway configuration** — Configure IP, buses
6. **Bridge configuration** — Map addresses to devices

### Addressing Methods

| Method | Tool | Notes |
|--------|------|-------|
| **Random commissioning** | Gateway | Assign available addresses |
| **Physical selection** | Gateway + button | Press button on driver to identify |
| **Sequential** | Commissioning tool | Walk around, assign in order |
| **Import from design** | Gateway API | Bulk configuration |

### Gray Logic Commissioning Workflow

1. Gateway discovers devices on bus
2. Export device list (short addresses)
3. Map to Gray Logic devices in configuration
4. Verify communication with each device
5. Configure groups and scenes
6. Test all functions

---

## Health Monitoring

### Device Diagnostics

| Query | Purpose | Frequency |
|-------|---------|-----------|
| `QUERY STATUS` | Overall status byte | Every 60s |
| `QUERY LAMP FAILURE` | Lamp/driver failure | Every 60s |
| `QUERY ACTUAL LEVEL` | Current output | On demand |
| `QUERY CONTROL GEAR PRESENT` | Device alive | On startup |

### Status Byte Interpretation

| Bit | Meaning |
|-----|---------|
| 0 | Control gear failure |
| 1 | Lamp failure |
| 2 | Lamp on (arc power > 0) |
| 3 | Limit error |
| 4 | Fade running |
| 5 | Reset state |
| 6 | Missing short address |
| 7 | Power failure (< 200ms) |

### PHM Integration

DALI provides rich diagnostic data for Predictive Health Monitoring:

- **Lamp failure detection** — Immediate alert
- **Runtime tracking** — LED lifespan prediction
- **Error rate tracking** — Communication reliability
- **Power consumption** — Baseline deviation

---

## Error Handling

| Error | Detection | Action |
|-------|-----------|--------|
| No response | Timeout | Retry 3×, mark offline |
| Lamp failure | Status query | Alert, log, continue |
| Wrong response | Validation | Log, ignore |
| Gateway offline | Connection lost | Retry, publish degraded |
| Bus short circuit | Gateway error | Alert, stop commands |

---

## Security Considerations

- DALI has no built-in security
- Gateway should be on Control VLAN only
- No direct internet access to DALI infrastructure
- API access controlled by gateway authentication

---

## Device Type Mappings

### Lighting Devices

| Gray Logic Type | DALI Device Type | Features |
|-----------------|-----------------|----------|
| `light_switch` | 7 (Relay) | On/off only |
| `light_dimmer` | 6 (LED) | 0-100% brightness |
| `light_ct` | 8 (Color, Tc) | Brightness + CCT |
| `light_rgb` | 8 (Color, RGBWAF) | RGB control |
| `light_rgbw` | 8 (Color, RGBWAF) | RGBW control |

### Sensor Devices (DALI-2)

| Gray Logic Type | DALI Part | Notes |
|-----------------|-----------|-------|
| `motion_sensor` | 303 | Occupancy sensor |
| `brightness_sensor` | 304 | Light sensor |

---

## Related Documents

- [KNX Protocol Specification](knx.md) — Complementary protocol
- [MQTT Protocol Specification](mqtt.md) — Internal message bus
- [Lighting Domain Specification](../domains/lighting.md) — Lighting features
- [Bridge Interface Specification](../architecture/bridge-interface.md) — Bridge contract

