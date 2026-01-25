---
title: KNX Protocol Reference
version: 2.0.0
status: approved
last_updated: 2026-01-25
spec_source: KNX Standard v3.0.0 (2021)
---

# KNX Protocol Reference

This document provides a comprehensive reference for KNX protocol implementation in Gray Logic, extracted from the official **KNX Standard v3.0.0** (2021).

**Normative References:**
- `03_07_02 Datapoint Types v02.02.01 AS.pdf` - Datapoint encoding/decoding
- `03_03_07 Application Layer v02.01.01 AS.pdf` - APCI commands, telegram structure
- `03_08_04 Tunnelling v01.07.01 AS.pdf` - KNXnet/IP tunneling protocol
- `03_02_06 Communication Medium KNX IP v01.01.02 AS.pdf` - KNX IP routing
- `03_08_02 Core v01.06.02 AS.pdf` - KNXnet/IP core services

The complete KNX Standard library is available in `docs/specs/knx/` (not distributed - copyrighted material).

---

## Table of Contents

1. [Telegram Structure](#telegram-structure)
2. [Addressing](#addressing)
3. [Application Layer (APCI)](#application-layer-apci)
4. [Datapoint Types (DPT)](#datapoint-types-dpt)
5. [KNXnet/IP Protocol](#knxnetip-protocol)
6. [cEMI Frame Format](#cemi-frame-format)
7. [Validation Rules](#validation-rules)
8. [Constants Reference](#constants-reference)

---

## Telegram Structure

> **Source:** KNX Standard 03_03_07 Application Layer v02.01.01, Section 2

### APDU Structure

The Application Protocol Data Unit (APDU) is the payload of a KNX telegram:

```
octet 6    octet 7         octet 8 ...     octet 21
┌─────────┬───────────────┬───────────────────────────┐
│ TPCI    │ APCI (4-10b)  │ Data (0-14 bytes)         │
└─────────┴───────────────┴───────────────────────────┘
```

- **TPCI**: Transport Control Field (handled by Transport Layer)
- **APCI**: Application Control Field (4 or 10 bits depending on service)
- **Data**: Application data, DPT-encoded

### Standard Frame Format (TP1)

| Byte | Field | Description |
|------|-------|-------------|
| 0 | Control Field 1 | Frame format, repeat, priority |
| 1-2 | Source Address | Sender individual address (16 bits) |
| 3-4 | Destination Address | Group or individual address (16 bits) |
| 5 | Control Field 2 / Routing | Address type, hop count, APDU length |
| 6+ | TPDU/APDU | Transport/Application data |
| Last | Checksum | XOR of all preceding bytes |

### Control Field 1 (Byte 0)

| Bits | Field | Values |
|------|-------|--------|
| 7-6 | Frame Format | `10` = Standard frame, `00` = Extended frame |
| 5 | Repeat Flag | `0` = Repeat on error, `1` = Do not repeat |
| 4 | System Broadcast | `0` = System broadcast, `1` = Normal broadcast |
| 3-2 | Priority | See Priority Values |
| 1-0 | Reserved | Must be `00` |

### Priority Values

| Code | Priority | Description | Use Case |
|------|----------|-------------|----------|
| `00` | System | Highest priority | Configuration, device management |
| `01` | High | Urgent | Urgent control commands |
| `10` | Alarm | Safety-critical | Fire, security alarms |
| `11` | Normal | Default | Regular automation telegrams |

### Control Field 2 / Routing (Byte 5)

| Bits | Field | Description |
|------|-------|-------------|
| 7 | Address Type (AT) | `0` = Individual address, `1` = Group address |
| 6-4 | Hop Count | Routing counter, decremented by line couplers |
| 3-0 | APDU Length | Length of APDU minus 1 (0-15 = 1-16 bytes) |

---

## Addressing

> **Source:** KNX Standard 03_01_01 Architecture v03.00.02

### Individual Address Format: `A.L.D`

Individual addresses uniquely identify each device on the KNX bus.

| Part | Bits | Range | Description |
|------|------|-------|-------------|
| Area (A) | 4 (15-12) | 0-15 | Building area or backbone segment |
| Line (L) | 4 (11-8) | 0-15 | Line within area |
| Device (D) | 8 (7-0) | 0-255 | Device on line (0 = line coupler) |

**Binary Layout:** 
```
Byte 0 (MSB): AAAA LLLL
Byte 1 (LSB): DDDD DDDD
```

**Special Addresses:**
- `0.0.0` - Reserved (invalid)
- `x.0.0` - Backbone coupler for area x
- `x.y.0` - Line coupler for line y in area x

### Group Address Formats

Group addresses are used for multicast communication between devices.

#### 3-Level Format: `M/S/G` (Recommended)

| Part | Bits | Range | Description |
|------|------|-------|-------------|
| Main (M) | 5 (15-11) | 0-31 | Main group (function category) |
| Sub (S) | 3 (10-8) | 0-7 | Middle group |
| Group (G) | 8 (7-0) | 0-255 | Sub group |

**Binary Layout:**
```
Byte 0 (MSB): MMMM MSSS
Byte 1 (LSB): GGGG GGGG
```

#### 2-Level Format: `M/G`

| Part | Bits | Range | Description |
|------|------|-------|-------------|
| Main (M) | 5 (15-11) | 0-31 | Main group |
| Group (G) | 11 (10-0) | 0-2047 | Sub group |

#### Free Format (16-bit)

Full 16-bit address space: 0-65535

**Special Group Addresses:**
- `0/0/0` (0x0000) - Broadcast address (all devices)

---

## Application Layer (APCI)

> **Source:** KNX Standard 03_03_07 Application Layer v02.01.01, Table 1

### APCI Encoding

The Application Control field identifies the service type. It is encoded in the first two bytes of the APDU.

```
Byte 0:  APCI[9:6] (upper 4 bits of APCI or APCI + data)
Byte 1:  APCI[5:0] / Data (lower 6 bits or small data values)
```

### Group Communication Services (T_Data_Group)

| APCI Binary | Hex | Service | Description |
|-------------|-----|---------|-------------|
| `00 0000 0000` | 0x000 | A_GroupValue_Read | Request current value from group |
| `00 01xx xxxx` | 0x040 | A_GroupValue_Response | Reply with current value |
| `00 10xx xxxx` | 0x080 | A_GroupValue_Write | Send new value to group |

**Note:** For small payloads (≤6 bits), the data is embedded in the APCI field itself (bits 5-0).

### Broadcast Services (T_Data_Broadcast)

| APCI Binary | Hex | Service | Description |
|-------------|-----|---------|-------------|
| `00 1100 0000` | 0x0C0 | A_IndividualAddress_Write | Assign individual address |
| `01 0000 0000` | 0x100 | A_IndividualAddress_Read | Query individual address |
| `01 0100 0000` | 0x140 | A_IndividualAddress_Response | Response with address |

### Point-to-Point Services (T_Data_Connected)

| APCI Binary | Hex | Service | Description |
|-------------|-----|---------|-------------|
| `01 10xx xxxx` | 0x180 | A_ADC_Read | Read A/D converter |
| `01 11xx xxxx` | 0x1C0 | A_ADC_Response | A/D converter response |
| `10 00nn nnnn` | 0x200 | A_Memory_Read | Read device memory |
| `10 01nn nnnn` | 0x240 | A_Memory_Response | Memory read response |
| `10 10nn nnnn` | 0x280 | A_Memory_Write | Write device memory |
| `11 0000 0000` | 0x300 | A_DeviceDescriptor_Read | Read device descriptor |
| `11 0100 0000` | 0x340 | A_DeviceDescriptor_Response | Device descriptor response |
| `11 1000 0000` | 0x380 | A_Restart | Restart device |

### Property Services (Extended APCI)

| APCI (10-bit) | Service | Description |
|---------------|---------|-------------|
| `11 1101 0101` | A_PropertyValue_Read | Read interface object property |
| `11 1101 0110` | A_PropertyValue_Response | Property value response |
| `11 1101 0111` | A_PropertyValue_Write | Write interface object property |
| `11 1101 1000` | A_PropertyDescription_Read | Read property description |
| `11 1101 1001` | A_PropertyDescription_Response | Property description response |

---

## Datapoint Types (DPT)

> **Source:** KNX Standard 03_07_02 Datapoint Types v02.02.01 AS

Datapoint Types define the encoding and semantics of data values. Format: `DPT X.YYY`
- **X**: Main type number (defines encoding format)
- **YYY**: Subtype number (defines interpretation/unit)

### Format Notation

| Symbol | Meaning |
|--------|---------|
| B | Boolean (1 bit) |
| U | Unsigned integer |
| V | Signed integer (two's complement) |
| F | Floating point |
| A | ASCII character |
| N | Enumeration (nibble) |
| r | Reserved bits |

### DPT 1.xxx - Boolean (1 bit, B1)

**Format:** 1 bit
**PDT:** PDT_BINARY_INFORMATION

| DPT | Name | Encoding: 0 | Encoding: 1 | Use |
|-----|------|-------------|-------------|-----|
| 1.001 | DPT_Switch | Off | On | G |
| 1.002 | DPT_Bool | False | True | G |
| 1.003 | DPT_Enable | Disable | Enable | G |
| 1.004 | DPT_Ramp | No ramp | Ramp | FB |
| 1.005 | DPT_Alarm | No alarm | Alarm | FB |
| 1.006 | DPT_BinaryValue | Low | High | FB |
| 1.007 | DPT_Step | Decrease | Increase | FB |
| 1.008 | DPT_UpDown | Up | Down | G |
| 1.009 | DPT_OpenClose | Open | Close | G |
| 1.010 | DPT_Start | Stop | Start | G |
| 1.011 | DPT_State | Inactive | Active | FB |
| 1.012 | DPT_Invert | Not inverted | Inverted | FB |
| 1.015 | DPT_Reset | No action | Reset (trigger) | G |
| 1.016 | DPT_Ack | No action | Acknowledge | G |
| 1.017 | DPT_Trigger | Trigger | Trigger | G |
| 1.018 | DPT_Occupancy | Not occupied | Occupied | G |
| 1.019 | DPT_Window_Door | Closed | Open | G |
| 1.024 | DPT_DayNight | Day | Night | G |

**Encoding:** Single byte, LSB contains value. `0x00` = 0, `0x01` = 1.

**Note:** For DPT_Trigger (1.017), both values 0 and 1 have the same effect.

### DPT 2.xxx - Boolean Control (2 bits, B2)

**Format:** 2 bits (c, v)
**PDT:** PDT_GENERIC_01

```
Bits:  c v
       │ └── Value (according to DPT 1.xxx)
       └──── Control (0=no control, 1=control)
```

| c | v | Meaning |
|---|---|---------|
| 0 | 0 | No control |
| 0 | 1 | No control |
| 1 | 0 | Control, function value 0 |
| 1 | 1 | Control, function value 1 |

| DPT | Name |
|-----|------|
| 2.001 | DPT_Switch_Control |
| 2.002 | DPT_Bool_Control |
| 2.008 | DPT_Direction1_Control |

### DPT 3.xxx - Controlled Dimming/Blinds (4 bits, B1U3)

**Format:** 4 bits
**PDT:** PDT_GENERIC_01

```
Bits:  c SSS
       │ └┴┴── StepCode (0-7)
       └────── Control direction
```

| DPT | Name | c=0 | c=1 |
|-----|------|-----|-----|
| 3.007 | DPT_Control_Dimming | Decrease | Increase |
| 3.008 | DPT_Control_Blinds | Up | Down |

**StepCode values:**
| Code | Intervals | Step Size |
|------|-----------|-----------|
| 000 | Break | Stop movement |
| 001 | 1 | 100% |
| 010 | 2 | 50% |
| 011 | 4 | 25% |
| 100 | 8 | 12.5% |
| 101 | 16 | 6.25% |
| 110 | 32 | 3.125% |
| 111 | 64 | 1.5625% |

**Formula:** Number of intervals = 2^(stepcode-1) for stepcode > 0

### DPT 4.xxx - Character (8 bits, A8)

**Format:** 8 bits (1 byte)

| DPT | Name | Range | Encoding |
|-----|------|-------|----------|
| 4.001 | DPT_Char_ASCII | 0-127 | 7-bit ASCII (MSB=0) |
| 4.002 | DPT_Char_8859_1 | 0-255 | ISO 8859-1 (Latin-1) |

### DPT 5.xxx - Unsigned 8-bit (U8)

**Format:** 8 bits unsigned
**PDT:** PDT_UNSIGNED_CHAR

| DPT | Name | Range | Unit | Resolution |
|-----|------|-------|------|------------|
| 5.001 | DPT_Scaling | 0-100 | % | ≈0.4% |
| 5.003 | DPT_Angle | 0-360 | ° | ≈1.4° |
| 5.004 | DPT_Percent_U8 | 0-255 | % | 1% |
| 5.005 | DPT_DecimalFactor | 0-255 | ratio | 1 |
| 5.006 | DPT_Tariff | 0-254 | - | 1 |
| 5.010 | DPT_Value_1_Ucount | 0-255 | pulses | 1 |

**DPT_Scaling (5.001) vs DPT_Percent_U8 (5.004):**
| Value | 5.001 (scaled) | 5.004 (linear) |
|-------|----------------|----------------|
| 50% | 0x80 (128) | 0x32 (50) |
| 100% | 0xFF (255) | 0x64 (100) |

### DPT 6.xxx - Signed 8-bit (V8)

**Format:** 8 bits signed (two's complement)
**PDT:** PDT_CHAR

| DPT | Name | Range | Unit | Resolution |
|-----|------|-------|------|------------|
| 6.001 | DPT_Percent_V8 | -128 to 127 | % | 1% |
| 6.010 | DPT_Value_1_Count | -128 to 127 | pulses | 1 |

### DPT 7.xxx - Unsigned 16-bit (U16)

**Format:** 16 bits unsigned (MSB first)
**PDT:** PDT_UNSIGNED_INT

| DPT | Name | Range | Unit | Resolution |
|-----|------|-------|------|------------|
| 7.001 | DPT_Value_2_Ucount | 0-65535 | pulses | 1 |
| 7.002 | DPT_TimePeriodMsec | 0-65535 | ms | 1 ms |
| 7.003 | DPT_TimePeriod10Msec | 0-655.35 | s | 10 ms |
| 7.005 | DPT_TimePeriodSec | 0-65535 | s | 1 s |
| 7.006 | DPT_TimePeriodMin | 0-65535 | min | 1 min |
| 7.007 | DPT_TimePeriodHrs | 0-65535 | h | 1 h |
| 7.011 | DPT_Length_mm | 0-65535 | mm | 1 mm |
| 7.012 | DPT_UElCurrentmA | 0-65535 | mA | 1 mA |
| 7.013 | DPT_Brightness | 0-65535 | lux | 1 lux |
| 7.600 | DPT_Absolute_Colour_Temperature | 0-65535 | K | 1 K |

### DPT 8.xxx - Signed 16-bit (V16)

**Format:** 16 bits signed (two's complement, MSB first)
**PDT:** PDT_INT

| DPT | Name | Range | Unit | Resolution |
|-----|------|-------|------|------------|
| 8.001 | DPT_Value_2_Count | -32768 to 32767 | pulses | 1 |
| 8.002 | DPT_DeltaTimeMsec | -32768 to 32767 | ms | 1 ms |
| 8.005 | DPT_DeltaTimeSec | -32768 to 32767 | s | 1 s |
| 8.010 | DPT_Percent_V16 | -327.68 to 327.67 | % | 0.01% |
| 8.011 | DPT_Rotation_Angle | -32768 to 32767 | ° | 1° |
| 8.012 | DPT_Length_m | -32768 to 32767 | m | 1 m |

**Note:** For 8.010, value 0x7FFF (32767) indicates invalid data.

### DPT 9.xxx - 2-byte Float (F16)

**Format:** 16 bits KNX float
**PDT:** PDT_KNX_FLOAT

```
Byte 0 (MSB):  M E E E E M M M
Byte 1 (LSB):  M M M M M M M M
               │ └┴┴┴┘ └┴┴┴┴┴┴┴── Mantissa (11 bits, signed)
               └──────────────── Exponent (4 bits, 0-15)
```

**Encoding formula:**
```
FloatValue = (0.01 × M) × 2^E

Where:
  E = Exponent [0..15]
  M = Mantissa [-2048..2047] (two's complement)
```

**Invalid value:** `0x7FFF` indicates invalid data for all DPT 9.xxx

| DPT | Name | Range | Unit | Resolution |
|-----|------|-------|------|------------|
| 9.001 | DPT_Value_Temp | -273 to 670760 | °C | 0.01°C |
| 9.002 | DPT_Value_Tempd | -670760 to 670760 | K | 0.01K |
| 9.003 | DPT_Value_Tempa | -670760 to 670760 | K/h | 0.01 K/h |
| 9.004 | DPT_Value_Lux | 0 to 670760 | lux | 0.01 lux |
| 9.005 | DPT_Value_Wsp | 0 to 670760 | m/s | 0.01 m/s |
| 9.006 | DPT_Value_Pres | 0 to 670760 | Pa | 0.01 Pa |
| 9.007 | DPT_Value_Humidity | 0 to 670760 | % | 0.01% |
| 9.008 | DPT_Value_AirQuality | 0 to 670760 | ppm | 0.01 ppm |
| 9.009 | DPT_Value_AirFlow | -670760 to 670760 | m³/h | 0.01 m³/h |
| 9.020 | DPT_Value_Volt | -670760 to 670760 | mV | 0.01 mV |
| 9.021 | DPT_Value_Curr | -670760 to 670760 | mA | 0.01 mA |
| 9.024 | DPT_Power | -670760 to 670760 | kW | 0.01 kW |
| 9.027 | DPT_Value_Temp_F | -459.6 to 670760 | °F | 0.01°F |
| 9.028 | DPT_Value_Wsp_kmh | 0 to 670760 | km/h | 0.01 km/h |

### DPT 10.001 - Time of Day (N3U5r2U6r2U6)

**Format:** 3 bytes
**PDT:** PDT_TIME

```
Byte 0:  DDD HHHHH    (Day 0-7, Hour 0-23)
Byte 1:  00 MMMMMM    (Minutes 0-59)
Byte 2:  00 SSSSSS    (Seconds 0-59)
```

| Day Value | Meaning |
|-----------|---------|
| 0 | No day specified |
| 1 | Monday |
| 2 | Tuesday |
| 3 | Wednesday |
| 4 | Thursday |
| 5 | Friday |
| 6 | Saturday |
| 7 | Sunday |

### DPT 11.001 - Date (r3U5r4U4r1U7)

**Format:** 3 bytes
**PDT:** PDT_DATE

```
Byte 0:  000 DDDDD    (Day 1-31)
Byte 1:  0000 MMMM    (Month 1-12)
Byte 2:  0 YYYYYYY    (Year 0-99)
```

**Century encoding:**
- Year ≥ 90: 19xx (1990-1999)
- Year < 90: 20xx (2000-2089)

### DPT 12.xxx - Unsigned 32-bit (U32)

**Format:** 32 bits unsigned (MSB first)
**PDT:** PDT_UNSIGNED_LONG

| DPT | Name | Range | Unit |
|-----|------|-------|------|
| 12.001 | DPT_Value_4_Ucount | 0 to 4294967295 | pulses |
| 12.100 | DPT_LongTimePeriod_Sec | 0 to 4294967295 | s |
| 12.101 | DPT_LongTimePeriod_Min | 0 to 4294967295 | min |
| 12.102 | DPT_LongTimePeriod_Hrs | 0 to 4294967295 | h |

### DPT 13.xxx - Signed 32-bit (V32)

**Format:** 32 bits signed (two's complement, MSB first)
**PDT:** PDT_LONG

| DPT | Name | Range | Unit |
|-----|------|-------|------|
| 13.001 | DPT_Value_4_Count | -2147483648 to 2147483647 | pulses |
| 13.002 | DPT_FlowRate_m3/h | -2147483648 to 2147483647 | 0.0001 m³/h |
| 13.010 | DPT_ActiveEnergy | -2147483648 to 2147483647 | Wh |
| 13.011 | DPT_ApparantEnergy | -2147483648 to 2147483647 | VAh |
| 13.012 | DPT_ReactiveEnergy | -2147483648 to 2147483647 | VARh |
| 13.013 | DPT_ActiveEnergy_kWh | -2147483648 to 2147483647 | kWh |
| 13.016 | DPT_ActiveEnergy_MWh | -2147483648 to 2147483647 | MWh |
| 13.100 | DPT_LongDeltaTimeSec | -2147483648 to 2147483647 | s |

### DPT 14.xxx - 4-byte Float (F32)

**Format:** 32 bits IEEE 754 single precision
**PDT:** PDT_FLOAT

| DPT | Name | Unit |
|-----|------|------|
| 14.000 | DPT_Value_Acceleration | m/s² |
| 14.007 | DPT_Value_AngleDeg | ° |
| 14.019 | DPT_Value_Electric_Current | A |
| 14.027 | DPT_Value_Electric_Potential | V |
| 14.033 | DPT_Value_Frequency | Hz |
| 14.039 | DPT_Value_Length | m |
| 14.056 | DPT_Value_Power | W |
| 14.057 | DPT_Value_Power_Factor | - (dimensionless) |
| 14.065 | DPT_Value_Speed | m/s |
| 14.068 | DPT_Value_Common_Temperature | °C |
| 14.076 | DPT_Value_Volume | m³ |
| 14.077 | DPT_Value_Volume_Flux | m³/s |
| 14.080 | DPT_Value_ApparentPower | VA |

### DPT 16.xxx - String (A[14])

**Format:** 14 bytes (14 characters)
**PDT:** PDT_GENERIC_14

| DPT | Name | Encoding |
|-----|------|----------|
| 16.000 | DPT_String_ASCII | 7-bit ASCII |
| 16.001 | DPT_String_8859_1 | ISO 8859-1 |

**Padding:** Unused characters must be `0x00` (NUL).

### DPT 17.001 - Scene Number (U8)

**Format:** 8 bits (only bits 0-5 used)
**Range:** 0-63

**Note:** Scene numbers are typically displayed as 1-64 (with +1 offset from encoded value).

### DPT 18.001 - Scene Control (B1U7)

**Format:** 8 bits

```
Bit 7:  L          (0=Activate, 1=Learn/Store)
Bits 6: r          (reserved)
Bits 5-0: SSSSSS   (Scene number 0-63)
```

| Bit 7 | Action |
|-------|--------|
| 0 | Activate scene |
| 1 | Learn/store current values to scene |

### DPT 19.001 - Date and Time (U3U5r2U6r2U6r4U4r7U7N3N5)

**Format:** 8 bytes
**PDT:** PDT_DATE_TIME

```
Byte 0: Year (0-255, offset from 1900)
Byte 1: 0000 MMMM (Month 1-12)
Byte 2: 000 DDDDD (Day 1-31)
Byte 3: DDD HHHHH (Day of week, Hour 0-23)
Byte 4: 00 MMMMMM (Minutes 0-59)
Byte 5: 00 SSSSSS (Seconds 0-59)
Byte 6: Flags
Byte 7: Quality flags
```

### DPT 20.xxx - Enumeration (N8)

**Format:** 8 bits enumeration
**PDT:** PDT_ENUM8

| DPT | Name | Values |
|-----|------|--------|
| 20.001 | DPT_SCLOMode | 0=Autonomous, 1=Slave |
| 20.002 | DPT_BuildingMode | 0=Building in use, 1=Building not used, 2=Building protection |
| 20.003 | DPT_OccMode | 0=Occupied, 1=Standby, 2=Not occupied |
| 20.102 | DPT_HVACMode | 0=Auto, 1=Comfort, 2=Standby, 3=Economy, 4=Building protection |
| 20.105 | DPT_HVACContrMode | 0=Auto, 1=Heat, 2=Morning warmup, 3=Cool, 4=Night purge, 5=Precool, 6=Off, 7=Test, 8=Emergency heat, 9=Fan only, 10=Free cool, 11=Ice, 12=Max heat, 13=Economic, 14=Dehumidify, 15=Calibrate, 16=Emergency cool, 17=Emergency steam, 18=Max cool, 19=Hv load, 20=No load |

### DPT 232.600 - RGB Color (U8U8U8)

**Format:** 3 bytes
**PDT:** PDT_GENERIC_03

```
Byte 0: Red   (0-255)
Byte 1: Green (0-255)
Byte 2: Blue  (0-255)
```

### DPT 251.600 - RGBW Color (U8U8U8U8r4B4)

**Format:** 6 bytes

```
Bytes 0-3: R, G, B, W (0-255 each)
Byte 4:    0000 VVVV (validity bits: R, G, B, W)
Byte 5:    Reserved
```

---

## KNXnet/IP Protocol

> **Source:** KNX Standard 03_08_04 Tunnelling v01.07.01, 03_02_06 KNX IP v01.01.02

### Protocol Overview

KNXnet/IP enables KNX communication over IP networks using:
- **Port:** 3671 (UDP and TCP)
- **Multicast Address:** 224.0.23.12 (system setup)
- **Protocol Version:** 0x10 (1.0)

### KNXnet/IP Header

All KNXnet/IP frames begin with a 6-byte header:

```
Byte 0:    Header Length (0x06)
Byte 1:    Protocol Version (0x10)
Bytes 2-3: Service Type Identifier (MSB first)
Bytes 4-5: Total Length (MSB first)
```

### Service Type Identifiers

| Code | Service | Description |
|------|---------|-------------|
| 0x0201 | SEARCH_REQUEST | Discover KNXnet/IP servers |
| 0x0202 | SEARCH_RESPONSE | Server discovery response |
| 0x0203 | DESCRIPTION_REQUEST | Request server description |
| 0x0204 | DESCRIPTION_RESPONSE | Server description |
| 0x0205 | CONNECT_REQUEST | Open connection |
| 0x0206 | CONNECT_RESPONSE | Connection response |
| 0x0207 | CONNECTIONSTATE_REQUEST | Check connection |
| 0x0208 | CONNECTIONSTATE_RESPONSE | Connection state |
| 0x0209 | DISCONNECT_REQUEST | Close connection |
| 0x020A | DISCONNECT_RESPONSE | Disconnect confirmation |
| 0x0310 | DEVICE_CONFIGURATION_REQUEST | Device management |
| 0x0311 | DEVICE_CONFIGURATION_ACK | Device management ack |
| 0x0420 | TUNNELLING_REQUEST | Tunnel KNX telegram |
| 0x0421 | TUNNELLING_ACK | Tunnel acknowledgment |
| 0x0530 | ROUTING_INDICATION | Multicast routing |
| 0x0531 | ROUTING_LOST_MESSAGE | Buffer overflow |
| 0x0532 | ROUTING_BUSY | Flow control |
| 0x0533 | ROUTING_SYSTEM_BROADCAST | System broadcast |

### Tunnelling Connection

1. Client sends CONNECT_REQUEST with connection type TUNNEL_CONNECTION (0x04)
2. Server responds with CONNECT_RESPONSE including:
   - Channel ID (1 byte)
   - Assigned Individual Address (2 bytes)
3. Telegrams exchanged via TUNNELLING_REQUEST/ACK
4. Each party maintains sequence counter (0-255, wrapping)
5. Timeout: 1 second for ACK (TUNNELLING_REQUEST_TIMEOUT)

### TUNNELLING_REQUEST Frame

```
Header (6 bytes):
  0x06 0x10 0x04 0x20 [Total Length]

Connection Header (4 bytes):
  [Structure Length=0x04] [Channel ID] [Sequence Counter] [Status=0x00]

cEMI Frame (variable):
  [Message Code] [Additional Info Length] [Additional Info...]
  [Control Field 1] [Control Field 2]
  [Source Address High] [Source Address Low]
  [Dest Address High] [Dest Address Low]
  [NPDU Length] [TPCI/APCI] [Data...]
```

### Routing (Multicast)

For KNX IP routing:
- Maximum 50 telegrams/second per device
- Minimum 5ms pause between transmissions
- Devices must handle ≥1000 telegrams/second receive rate
- ROUTING_BUSY used for flow control

### Individual Address Assignment

For tunneling connections, each connection receives a unique Individual Address:
- Server's own address or
- Additional addresses from PID_ADDITIONAL_INDIVIDUAL_ADDRESSES

---

## cEMI Frame Format

> **Source:** KNX Standard 03_06_03 EMI_IMI v01.04.02

### cEMI Message Codes

| Code | Name | Description |
|------|------|-------------|
| 0x11 | L_Data.req | Data request from client |
| 0x29 | L_Data.ind | Data indication to client |
| 0x2E | L_Data.con | Data confirmation |
| 0x10 | L_Raw.req | Raw frame request |
| 0x2D | L_Raw.ind | Raw frame indication |
| 0x2F | L_Raw.con | Raw frame confirmation |
| 0x2B | L_Busmon.ind | Bus monitor indication |
| 0xF1 | M_Reset.req | Reset request |

### L_Data Frame Structure

```
Byte 0:    Message Code (0x11, 0x29, or 0x2E)
Byte 1:    Additional Info Length (0 if none)
Bytes 2-n: Additional Info (if any)
Byte n+1:  Control Field 1
Byte n+2:  Control Field 2
Bytes n+3,n+4: Source Address
Bytes n+5,n+6: Destination Address
Byte n+7:  NPDU Length (APDU length - 1)
Bytes n+8+: TPCI/APCI + Data
```

### Control Field 1 (cEMI)

| Bits | Field | Description |
|------|-------|-------------|
| 7 | Frame Type | 1=Standard, 0=Extended |
| 6 | Reserved | 0 |
| 5 | Repeat Flag | 0=Repeat, 1=No repeat |
| 4 | System Broadcast | 0=System, 1=Normal |
| 3-2 | Priority | 00=System, 01=High, 10=Alarm, 11=Normal |
| 1 | Ack Request | 0=No ack, 1=Ack requested |
| 0 | Confirm Flag | 0=No error, 1=Error |

### Control Field 2 (cEMI)

| Bits | Field | Description |
|------|-------|-------------|
| 7 | Address Type | 0=Individual, 1=Group |
| 6-4 | Hop Count | 0-7 (decremented by routers) |
| 3-0 | Extended Frame Format | 0000 for standard |

---

## Validation Rules

These validation rules MUST be enforced by Gray Logic implementations:

### Address Validation

| Field | Valid Range | Notes |
|-------|-------------|-------|
| Individual Area | 0-15 | 0 valid for backbone couplers |
| Individual Line | 0-15 | 0 valid for line couplers |
| Individual Device | 0-255 | 0 = coupler address |
| Group Main (3-level) | 0-31 | |
| Group Sub (3-level) | 0-7 | |
| Group Address | 0-255 | |
| Group (2-level) | 0-2047 | |

### DPT Range Validation

| DPT | Valid Range | Invalid Value |
|-----|-------------|---------------|
| 1.xxx | 0 or 1 | Any other |
| 3.xxx | c: 0-1, step: 0-7 | Out of range |
| 5.001 | 0-255 (represents 0-100%) | N/A |
| 5.003 | 0-255 (represents 0-360°) | N/A |
| 5.006 | 0-254 | 255 reserved |
| 9.xxx | See encoding | 0x7FFF = invalid |
| 9.001 | ≥ -273.15°C | Below absolute zero |
| 10.001 | Day 0-7, H 0-23, M 0-59, S 0-59 | Out of range |
| 11.001 | D 1-31, M 1-12, Y 0-99 | Out of range |
| 17.001 | 0-63 | > 63 |
| 18.001 | Scene 0-63 | Scene > 63 |
| 232.600 | R,G,B each 0-255 | N/A |

### Protocol Validation

| Check | Rule |
|-------|------|
| Frame length | Must match length field |
| Checksum | XOR of all bytes = 0 (TP1) |
| APCI | Must be valid service code |
| Hop count | 0-7 |
| Sequence counter | 0-255 (tunneling) |

---

## Constants Reference

### EIB/KNX Protocol Constants

```go
// EIB Socket Protocol (used by knxd/eibd)
const (
    EIB_OPEN_GROUPCON     uint16 = 0x0026  // Open group connection
    EIB_OPEN_T_GROUP      uint16 = 0x0022  // Open T_Group connection  
    EIB_OPEN_T_CONNECTION uint16 = 0x0020  // Open point-to-point
    EIB_OPEN_T_INDIVIDUAL uint16 = 0x0049  // Open individual
    EIB_OPEN_T_TPDU       uint16 = 0x0021  // Open T_TPDU
    EIB_APDU_PACKET       uint16 = 0x0025  // APDU packet
    EIB_GROUP_PACKET      uint16 = 0x0027  // Group packet
    EIB_RESET_CONNECTION  uint16 = 0x0004  // Reset connection
    EIB_CACHE_ENABLE      uint16 = 0x0070  // Enable cache
    EIB_CACHE_DISABLE     uint16 = 0x0071  // Disable cache
    EIB_CACHE_READ        uint16 = 0x0074  // Read from cache
)

// APCI Commands (in APDU)
const (
    APCI_GROUP_READ      uint16 = 0x0000  // A_GroupValue_Read
    APCI_GROUP_RESPONSE  uint16 = 0x0040  // A_GroupValue_Response
    APCI_GROUP_WRITE     uint16 = 0x0080  // A_GroupValue_Write
    APCI_INDIVIDUAL_WRITE uint16 = 0x00C0 // A_IndividualAddress_Write
    APCI_INDIVIDUAL_READ uint16 = 0x0100  // A_IndividualAddress_Read
    APCI_INDIVIDUAL_RESP uint16 = 0x0140  // A_IndividualAddress_Response
    APCI_ADC_READ        uint16 = 0x0180  // A_ADC_Read
    APCI_ADC_RESPONSE    uint16 = 0x01C0  // A_ADC_Response
    APCI_MEMORY_READ     uint16 = 0x0200  // A_Memory_Read
    APCI_MEMORY_RESPONSE uint16 = 0x0240  // A_Memory_Response
    APCI_MEMORY_WRITE    uint16 = 0x0280  // A_Memory_Write
    APCI_DEVICE_DESC_READ uint16 = 0x0300 // A_DeviceDescriptor_Read
    APCI_DEVICE_DESC_RESP uint16 = 0x0340 // A_DeviceDescriptor_Response
    APCI_RESTART         uint16 = 0x0380  // A_Restart
)

// KNXnet/IP Constants
const (
    KNXNETIP_VERSION     byte   = 0x10    // Protocol version 1.0
    KNXNETIP_HEADER_SIZE byte   = 0x06    // Header size
    KNXNETIP_PORT        uint16 = 3671    // Default port
    
    // Connection Types
    DEVICE_MANAGEMENT_CONNECTION byte = 0x03
    TUNNEL_CONNECTION           byte = 0x04
    REMLOG_CONNECTION           byte = 0x06
    REMCONF_CONNECTION          byte = 0x07
    OBJSVR_CONNECTION           byte = 0x08
    
    // Tunnelling Layer
    TUNNEL_LINKLAYER    byte = 0x02
    TUNNEL_RAW          byte = 0x04
    TUNNEL_BUSMONITOR   byte = 0x80
)

// cEMI Message Codes
const (
    CEMI_L_DATA_REQ    byte = 0x11  // L_Data.req
    CEMI_L_DATA_IND    byte = 0x29  // L_Data.ind
    CEMI_L_DATA_CON    byte = 0x2E  // L_Data.con
    CEMI_L_RAW_REQ     byte = 0x10  // L_Raw.req
    CEMI_L_RAW_IND     byte = 0x2D  // L_Raw.ind
    CEMI_L_RAW_CON     byte = 0x2F  // L_Raw.con
    CEMI_L_BUSMON_IND  byte = 0x2B  // L_Busmon.ind
    CEMI_M_RESET_REQ   byte = 0xF1  // M_Reset.req
)

// Priority Values
const (
    PRIORITY_SYSTEM byte = 0x00  // System (highest)
    PRIORITY_HIGH   byte = 0x01  // High/Urgent
    PRIORITY_ALARM  byte = 0x02  // Alarm
    PRIORITY_NORMAL byte = 0x03  // Normal (default)
)

// Timing Constants
const (
    TUNNELLING_REQUEST_TIMEOUT = 1 * time.Second
    HEARTBEAT_REQUEST_TIMEOUT  = 10 * time.Second
    CONNECTION_STATE_TIMEOUT   = 60 * time.Second
    ROUTING_BUSY_WAIT_TIME_MS  = 100  // milliseconds
)
```

---

## Encoding/Decoding Examples

### DPT 9 (2-byte Float) Encoding

```go
// Encode float to DPT9
func EncodeDPT9(value float64) [2]byte {
    // Find appropriate exponent
    var mantissa int
    var exponent int
    
    v := value * 100  // Scale by 0.01
    
    for exponent = 0; exponent <= 15; exponent++ {
        mantissa = int(v / math.Pow(2, float64(exponent)))
        if mantissa >= -2048 && mantissa <= 2047 {
            break
        }
    }
    
    // Encode: MEEE EMMM MMMM MMMM
    var result [2]byte
    if mantissa < 0 {
        mantissa = mantissa & 0x07FF  // Keep 11 bits
        mantissa |= 0x8000            // Set sign bit
    }
    result[0] = byte((mantissa >> 8) & 0x07) | byte((exponent << 3) & 0x78)
    if value < 0 {
        result[0] |= 0x80  // Sign bit
    }
    result[1] = byte(mantissa & 0xFF)
    
    return result
}

// Decode DPT9 to float
func DecodeDPT9(data [2]byte) float64 {
    // Extract fields
    sign := (data[0] & 0x80) != 0
    exponent := int((data[0] >> 3) & 0x0F)
    mantissa := int((data[0] & 0x07)) << 8 | int(data[1])
    
    // Handle sign (two's complement for mantissa)
    if sign {
        mantissa = mantissa - 2048
    }
    
    // Calculate value
    return 0.01 * float64(mantissa) * math.Pow(2, float64(exponent))
}
```

### Group Address Parsing

```go
// Parse 3-level group address "M/S/G"
func ParseGroupAddress3Level(addr string) (uint16, error) {
    parts := strings.Split(addr, "/")
    if len(parts) != 3 {
        return 0, errors.New("invalid 3-level address format")
    }
    
    main, _ := strconv.Atoi(parts[0])  // 0-31
    sub, _ := strconv.Atoi(parts[1])   // 0-7
    group, _ := strconv.Atoi(parts[2]) // 0-255
    
    if main < 0 || main > 31 || sub < 0 || sub > 7 || group < 0 || group > 255 {
        return 0, errors.New("address component out of range")
    }
    
    return uint16((main << 11) | (sub << 8) | group), nil
}

// Format group address as 3-level string
func FormatGroupAddress3Level(addr uint16) string {
    main := (addr >> 11) & 0x1F
    sub := (addr >> 8) & 0x07
    group := addr & 0xFF
    return fmt.Sprintf("%d/%d/%d", main, sub, group)
}
```

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-25 | Initial version from web sources |
| 2.0.0 | 2026-01-25 | Complete rewrite from KNX Standard v3.0.0 PDFs |

---

*This document is derived from the KNX Standard v3.0.0 © KNX Association. The KNX Standard is copyrighted material and the full specifications should be obtained from [MyKNX Portal](https://my.knx.org/).*
