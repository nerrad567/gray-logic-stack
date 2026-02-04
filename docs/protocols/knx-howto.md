---
title: KNX Practical How-To Guide
version: 1.0.0
status: active
last_updated: 2026-01-27
depends_on:
  - protocols/knx.md
  - protocols/knx-reference.md
---

# KNX Practical How-To Guide

A hands-on guide for working with KNX — what everything means, best practices, and how to create and configure devices.

---

## Understanding KNX Basics

### The Bus Concept

KNX is a **bus system** — all devices share a single communication line (twisted pair cable). Think of it like a party line where everyone can hear everyone:

```
                         KNX Bus Cable (Green)
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
        │           │           │           │           │
    ┌───┴───┐   ┌───┴───┐   ┌───┴───┐   ┌───┴───┐   ┌───┴───┐
    │Power  │   │Switch │   │Dimmer │   │Blind  │   │Sensor │
    │Supply │   │Input  │   │Actuator   │Actuator   │       │
    └───────┘   └───────┘   └───────┘   └───────┘   └───────┘
```

Every device on the bus can:
- **Send** telegrams (messages) to the bus
- **Receive** telegrams from the bus
- **React** to telegrams addressed to it

### Physical vs Group Addresses

This is the most important concept to understand:

#### Physical Address (Hardware ID)

- Format: `Area.Line.Device` (e.g., `1.2.15`)
- **Unique per device** — like a MAC address
- Assigned during ETS programming
- Used for: diagnostics, programming, identifying hardware
- **You rarely use this day-to-day**

Example: `1.2.15` means Area 1, Line 2, Device 15

#### Group Address (Function ID)

- Format: `Main/Middle/Sub` (e.g., `1/0/5`)
- **Shared between devices** — this is how devices talk to each other
- One device can listen to many group addresses
- Many devices can listen to the same group address
- **This is what you work with 90% of the time**

Example: `1/0/5` might be "Living Room Main Light Switch"

### How Devices Communicate

**Scenario**: Wall switch controls a dimmer

1. You press the wall switch
2. Switch sends telegram to group address `1/0/5` with value `1` (ON)
3. Dimmer is configured to listen to `1/0/5`
4. Dimmer receives telegram, turns on the light
5. Dimmer sends status to `1/0/105` with value `1` (ON)
6. Switch LED (if any) and Gray Logic see the status update

```
Wall Switch                              Dimmer Actuator
    │                                          │
    │──── GA 1/0/5 = ON ─────────────────────▶│
    │                                          │ Light turns on
    │◀─── GA 1/0/105 = ON (status) ───────────│
    │                                          │
```

**Key insight**: The switch doesn't "know" about the dimmer. It just sends to a group address. Any device listening to that address will react.

---

## Group Address Structure (Best Practice)

### 3-Level Addressing

Format: `Main/Middle/Sub`

| Level | Range | Purpose |
|-------|-------|---------|
| Main | 0-31 | Function category |
| Middle | 0-7 | Floor or zone |
| Sub | 0-255 | Individual function |

### Recommended Main Groups

| Main | Category | Description |
|------|----------|-------------|
| **0** | System | Heartbeats, diagnostics, time, date |
| **1** | Switching | On/off commands for lights |
| **2** | Dimming | Brightness values (0-100%) |
| **3** | Blinds | Position, tilt, up/down |
| **4** | HVAC | Temperature, setpoints, modes |
| **5** | Sensors | Motion, brightness, temperature readings |
| **6** | Status | Feedback from actuators |
| **7** | Scenes | Scene recall and storage |
| **8-14** | Reserved | Future use / project-specific |
| **15** | Special | Central functions, all-off, panic |

### Example Address Scheme

For a house with Ground Floor (0) and First Floor (1):

```
Switching (Main 1):
  1/0/1   Ground Floor / Living Room / Main Light
  1/0/2   Ground Floor / Living Room / Wall Light
  1/0/3   Ground Floor / Kitchen / Ceiling Light
  1/1/1   First Floor / Master Bedroom / Main Light
  1/1/2   First Floor / Master Bedroom / Bedside Left

Dimming (Main 2):
  2/0/1   Ground Floor / Living Room / Main Light Brightness
  2/0/2   Ground Floor / Living Room / Wall Light Brightness
  
Status (Main 6):
  6/0/1   Ground Floor / Living Room / Main Light Status
  6/0/2   Ground Floor / Living Room / Wall Light Status
  6/0/101 Ground Floor / Living Room / Main Light Brightness Status
```

### Naming Convention

Use consistent, descriptive names in ETS:

```
{Floor}/{Room}/{Device}/{Function}

Examples:
GF/Living/MainLight/Switch
GF/Living/MainLight/Brightness
GF/Living/MainLight/Status
GF/Living/Blind1/Position
GF/Living/Blind1/PositionStatus
FF/Master/BedsideL/Switch
```

---

## Datapoint Types (DPTs) Explained

DPTs define **how data is encoded** in a telegram. Using the wrong DPT means the receiving device won't understand the message.

### Most Common DPTs

#### Switching (1-bit)

| DPT | Name | Values | Use |
|-----|------|--------|-----|
| **1.001** | Switch | 0=Off, 1=On | Light switching |
| **1.002** | Bool | 0=False, 1=True | Binary states |
| **1.008** | Up/Down | 0=Up, 1=Down | Blind direction |
| **1.009** | Open/Close | 0=Open, 1=Closed | Door/window contacts |
| **1.017** | Trigger | 1=Trigger | Pulse (blind stop) |

#### Dimming (4-bit and 1-byte)

| DPT | Name | Values | Use |
|-----|------|--------|-----|
| **3.007** | Dimming Control | 4-bit: direction + steps | Relative dimming |
| **5.001** | Percentage | 0-100% (1 byte) | Absolute brightness |
| **5.003** | Angle | 0-360° | Blind tilt angle |

#### Temperature (2-byte float)

| DPT | Name | Range | Use |
|-----|------|-------|-----|
| **9.001** | Temperature | -273 to 670760 °C | Temperature reading |
| **9.004** | Lux | 0 to 670760 lux | Brightness sensor |
| **9.007** | Humidity | 0-100% | Humidity reading |

#### Scenes

| DPT | Name | Values | Use |
|-----|------|--------|-----|
| **17.001** | Scene Number | 0-63 | Recall scene |
| **18.001** | Scene Control | 0-63 + learn bit | Recall or learn scene |

### DPT Cheat Sheet

```
Need to turn something on/off?     → DPT 1.001 (Switch)
Need to set a percentage?          → DPT 5.001 (Percentage)
Need to read a temperature?        → DPT 9.001 (Temperature)
Need to move a blind?              → DPT 1.008 (Up/Down)
Need to set blind position?        → DPT 5.001 (Percentage)
Need to recall a scene?            → DPT 17.001 (Scene Number)
```

---

## Communication Flags Explained

Every group object has flags that control its behaviour:

| Flag | Name | Meaning |
|------|------|---------|
| **C** | Communication | Object can communicate on the bus (must be ON) |
| **R** | Read | Object can be read (responds to read requests) |
| **W** | Write | Object can be written (accepts commands) |
| **T** | Transmit | Object sends value when it changes |
| **U** | Update | Object updates its value when it receives a response |
| **I** | ReadOnInit | Object sends a read request at startup |

### Flag Combinations by Object Type

#### Command Object (e.g., Switch Input)

```
C R W T U I
✓ - - ✓ - -     "Send only"
```
- **C**: Can communicate
- **T**: Transmits when button pressed
- No R/W: Doesn't accept commands

#### Actuator Input (e.g., Dimmer receiving commands)

```
C R W T U I
✓ - ✓ - ✓ -     "Write + Update"
```
- **C**: Can communicate
- **W**: Accepts write commands
- **U**: Updates internal value on response

#### Status Object (e.g., Dimmer status feedback)

```
C R W T U I
✓ ✓ - ✓ - -     "Read + Transmit"
```
- **C**: Can communicate
- **R**: Responds to read requests
- **T**: Transmits when value changes

#### Startup Read (e.g., Get initial state)

```
C R W T U I
✓ - - - ✓ ✓     "Init reader"
```
- **U**: Updates on response
- **I**: Sends read request at startup

### Visual Flag Reference

```
SWITCH (wall button):
  ┌─────────────────┐
  │ GA: 1/0/1       │
  │ DPT: 1.001      │
  │ Flags: C--T--   │  ← Transmits when pressed
  └─────────────────┘

DIMMER (receiving commands):
  ┌─────────────────┐
  │ GA: 1/0/1       │
  │ DPT: 1.001      │
  │ Flags: C-WTU-   │  ← Writes to output, transmits status
  └─────────────────┘
  
STATUS FEEDBACK:
  ┌─────────────────┐
  │ GA: 6/0/1       │
  │ DPT: 1.001      │
  │ Flags: CR-T--   │  ← Can be read, transmits on change
  └─────────────────┘
```

---

## Creating Devices in ETS

### Step 1: Add Device to Project

1. Open ETS project
2. Go to **Catalog** panel
3. Search for your device (e.g., "MDT AKD-0424")
4. Drag device to the appropriate line in your topology
5. Assign physical address (e.g., `1.1.5`)

### Step 2: Configure Device Parameters

Each device has parameters that define its behaviour:

**Example: MDT Dimmer Actuator**

| Parameter | Setting | Why |
|-----------|---------|-----|
| Dimming curve | LED optimised | Smooth LED dimming |
| Minimum brightness | 5% | Prevent LED flicker |
| Switch-on brightness | Last value | Resume previous level |
| Status sending | On change | Send feedback when state changes |
| Staircase timer | Disabled | Not using auto-off |

### Step 3: Create Group Addresses

1. Go to **Group Addresses** panel
2. Create main group (e.g., "1 Switching")
3. Create middle group (e.g., "0 Ground Floor")
4. Create sub-address (e.g., "1 Living Room Main")
5. Full address: `1/0/1`

### Step 4: Link Objects to Group Addresses

1. Open device configuration
2. Find communication object (e.g., "Switch Input")
3. Drag group address to object, OR
4. Right-click object → Link → Select GA

**Tip**: Link multiple devices to the same GA to create relationships:

```
Wall Switch → GA 1/0/1 → Dimmer (all linked to same GA)
```

### Step 5: Configure Flags

ETS usually sets sensible defaults, but verify:

**For switch input (wall button)**:
- Flags: `C--T--` (Communicate, Transmit)

**For actuator receiving commands**:
- Flags: `C-W-U-` (Communicate, Write, Update)

**For status feedback**:
- Flags: `CR-T--` (Communicate, Read, Transmit)

### Step 6: Download to Devices

1. Select devices to program
2. **Bus** → **Download** → **Download Application**
3. Wait for confirmation
4. Test functionality

---

## Common Device Configurations

### Simple Light Switch

```
┌─────────────────────────────────────────────────────────┐
│ WALL SWITCH (e.g., MDT BE-TA55P4.01)                   │
├─────────────────────────────────────────────────────────┤
│ Object: Channel A - Switching                           │
│ GA: 1/0/1 "GF/Living/MainLight/Switch"                 │
│ DPT: 1.001                                              │
│ Flags: C--T--                                           │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│ SWITCH ACTUATOR (e.g., MDT AKS-1216.03)                │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Switch                              │
│ GA: 1/0/1 "GF/Living/MainLight/Switch"                 │
│ DPT: 1.001                                              │
│ Flags: C-W-U-                                           │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Status                              │
│ GA: 6/0/1 "GF/Living/MainLight/Status"                 │
│ DPT: 1.001                                              │
│ Flags: CR-T--                                           │
└─────────────────────────────────────────────────────────┘
```

### Dimmable Light

```
┌─────────────────────────────────────────────────────────┐
│ PUSH BUTTON (e.g., MDT BE-TA55P4.01)                   │
├─────────────────────────────────────────────────────────┤
│ Object: Channel A - Switching (short press)             │
│ GA: 1/0/1 "GF/Living/MainLight/Switch"                 │
│ DPT: 1.001, Flags: C--T--                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel A - Dimming (long press)                │
│ GA: 2/0/1 "GF/Living/MainLight/RelDim"                 │
│ DPT: 3.007, Flags: C--T--                              │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│ DIMMER ACTUATOR (e.g., MDT AKD-0424V.02)               │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Switch                              │
│ GA: 1/0/1 "GF/Living/MainLight/Switch"                 │
│ DPT: 1.001, Flags: C-W-U-                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Relative Dimming                    │
│ GA: 2/0/1 "GF/Living/MainLight/RelDim"                 │
│ DPT: 3.007, Flags: C-W---                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Brightness Value                    │
│ GA: 2/0/101 "GF/Living/MainLight/Brightness"           │
│ DPT: 5.001, Flags: C-W-U-                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Switch Status                       │
│ GA: 6/0/1 "GF/Living/MainLight/SwitchStatus"           │
│ DPT: 1.001, Flags: CR-T--                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Brightness Status                   │
│ GA: 6/0/101 "GF/Living/MainLight/BrightnessStatus"     │
│ DPT: 5.001, Flags: CR-T--                              │
└─────────────────────────────────────────────────────────┘
```

### Blind/Shutter

```
┌─────────────────────────────────────────────────────────┐
│ PUSH BUTTON                                             │
├─────────────────────────────────────────────────────────┤
│ Object: Up/Down (short press)                           │
│ GA: 3/0/1 "GF/Living/Blind1/UpDown"                    │
│ DPT: 1.008, Flags: C--T--                              │
├─────────────────────────────────────────────────────────┤
│ Object: Stop (long press or second press)               │
│ GA: 3/0/2 "GF/Living/Blind1/Stop"                      │
│ DPT: 1.017, Flags: C--T--                              │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│ BLIND ACTUATOR (e.g., MDT JAL-0410.02)                 │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Up/Down                             │
│ GA: 3/0/1 "GF/Living/Blind1/UpDown"                    │
│ DPT: 1.008, Flags: C-W---                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Stop/Step                           │
│ GA: 3/0/2 "GF/Living/Blind1/Stop"                      │
│ DPT: 1.017, Flags: C-W---                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Position                            │
│ GA: 3/0/101 "GF/Living/Blind1/Position"                │
│ DPT: 5.001, Flags: C-W-U-                              │
├─────────────────────────────────────────────────────────┤
│ Object: Channel 1 - Position Status                     │
│ GA: 6/0/30 "GF/Living/Blind1/PositionStatus"           │
│ DPT: 5.001, Flags: CR-T--                              │
└─────────────────────────────────────────────────────────┘
```

---

## Best Practices

### 1. Always Configure Status Feedback

**Bad**: Command GA only
```
Switch → GA 1/0/1 → Dimmer
```
Gray Logic has no idea if the light actually turned on.

**Good**: Command + Status GAs
```
Switch → GA 1/0/1 → Dimmer
                  ← GA 6/0/1 (status feedback)
```
Gray Logic sees confirmation of actual state.

### 2. Use Separate Command and Status GAs

**Bad**: Same GA for command and status
```
GA 1/0/1: Used for both sending commands AND reporting status
```
Creates bus traffic loops and confusing behaviour.

**Good**: Separate GAs
```
GA 1/0/1: Commands (write only)
GA 6/0/1: Status (read/transmit only)
```

### 3. Consistent Addressing Scheme

**Bad**: Random addressing
```
1/0/1   Kitchen light
5/2/17  Living room light
0/7/3   Bedroom light
```

**Good**: Logical structure
```
1/0/1   GF/Kitchen/MainLight/Switch
1/0/2   GF/Living/MainLight/Switch
1/1/1   FF/Master/MainLight/Switch
```

### 4. Document Everything in ETS

- Use full descriptive names for GAs
- Add comments to devices
- Create a consistent naming convention
- Export GA list for reference

### 5. Test Physical First

Before integrating with Gray Logic:
1. Verify switch controls actuator directly
2. Verify status feedback is sent
3. Use ETS bus monitor to confirm telegrams
4. Only then add Gray Logic integration

### 6. ReadOnInit for Critical Status

For devices where Gray Logic needs to know the state at startup:
- Add **I** (ReadOnInit) flag to status objects
- Or configure Gray Logic to poll status GAs on startup

---

## Adding Devices to Gray Logic

### 1. Device Must Exist in ETS First

Gray Logic doesn't configure KNX devices — ETS does. Gray Logic only:
- Listens to telegrams
- Sends commands to existing GAs
- Maps GAs to logical devices

### 2. Create Device Mapping

In `knx-bridge.yaml` or via the API:

```yaml
devices:
  - device_id: "light-living-main"
    name: "Living Room Main Light"
    type: "light_dimmer"
    room: "living_room"
    individual_address: "1.1.10"
    functions:
      switch:
        ga: "1/0/1"
        dpt: "1.001"
        flags: ["write"]
      switch_status:
        ga: "6/0/1"
        dpt: "1.001"
        flags: ["read", "transmit"]
      brightness:
        ga: "2/0/101"
        dpt: "5.001"
        flags: ["write"]
      brightness_status:
        ga: "6/0/101"
        dpt: "5.001"
        flags: ["read", "transmit"]
```

### 3. Device Types and Required Addresses

| Type | Required Addresses | Optional |
|------|-------------------|----------|
| `light_switch` | switch, switch_status | — |
| `light_dimmer` | switch, switch_status, brightness, brightness_status | — |
| `blind_position` | up_down, stop, position_status | position, tilt, tilt_status |
| `temperature_sensor` | temperature | — |
| `motion_sensor` | motion | — |
| `thermostat` | temperature, setpoint, setpoint_status | mode, valve_status |

### 4. Verify in Gray Logic

After adding device:
1. Check device appears in device list
2. Operate physical switch — status should update
3. Send command from Gray Logic — actuator should respond
4. Verify status feedback received

---

## Troubleshooting

### Device Not Responding to Commands

1. **Check GA is correct** — typos are common
2. **Check DPT matches** — wrong DPT = garbled data
3. **Check flags** — actuator needs **W** flag on command object
4. **Check ETS download** — did you download after changes?
5. **Check physical wiring** — is device on the bus?

### Status Not Updating

1. **Check status GA exists** — did you configure feedback?
2. **Check flags** — status object needs **T** (Transmit) flag
3. **Check actuator parameter** — "send status on change" enabled?
4. **Check Gray Logic subscription** — is it listening to that GA?

### Commands Work, But Status Delayed

1. **Actuator parameter** — check "status sending" timing
2. **Bus traffic** — heavy traffic can delay status
3. **Read vs Transmit** — if only **R** flag, Gray Logic must poll

### Intermittent Communication

1. **Bus power** — check power supply capacity
2. **Topology** — too many devices on one line?
3. **Cable quality** — proper KNX cable (green, twisted pair)?
4. **Termination** — line properly terminated?

### Debug with Bus Monitor

```bash
# Watch all bus traffic
knxtool busmonitor1 ip:localhost

# Example output:
# from 1.1.5 to 1/0/1 hops: 06 T_Data_Group A_GroupValue_Write small 01
# (Device 1.1.5 sent ON to GA 1/0/1)
```

---

## Quick Reference Card

### Address Format

```
Physical: Area.Line.Device (1.2.15)
Group:    Main/Middle/Sub  (1/0/5)
```

### Common DPTs

```
1.001  Switch      (1 bit)   0=Off, 1=On
1.008  Up/Down     (1 bit)   0=Up, 1=Down  
1.017  Trigger     (1 bit)   1=Trigger
3.007  Dim Control (4 bit)   Direction + steps
5.001  Percentage  (1 byte)  0-100%
9.001  Temperature (2 byte)  °C float
17.001 Scene       (1 byte)  0-63
```

### Flags

```
C = Communication (always on)
R = Read (can be polled)
W = Write (accepts commands)
T = Transmit (sends on change)
U = Update (updates from bus)
I = Init (reads at startup)
```

### Typical Flag Combinations

```
Wall Switch:     C--T--  (transmit only)
Actuator Input:  C-W-U-  (write + update)
Status Output:   CR-T--  (read + transmit)
Sensor:          CR-T--  (read + transmit)
```

---

## Related Documents

- [KNX Protocol Specification](knx.md) — Technical protocol details
- [KNX Reference](knx-reference.md) — Full DPT tables, spec details
- [MQTT Protocol](mqtt.md) — Internal message bus for Gray Logic
