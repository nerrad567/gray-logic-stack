---
title: KNX Protocol Reference
version: 1.0.0
status: draft
last_updated: 2026-01-25
---

# KNX Protocol Reference

This document provides a reference for KNX protocol implementation in Gray Logic.

**Sources:**
- [KNX Association - Datapoint Types](https://support.knx.org/hc/en-us/articles/115001133744-Datapoint-Type)
- [KNX Association - Telegram Structure](https://support.knx.org/hc/en-us/sections/115001019689-TP1-Telegrams-Structure)
- [Qt KNX Documentation](https://doc.qt.io/qt-5/qtknx-datapoint-types.html)
- [KnxDevice Library](https://github.com/franckmarini/KnxDevice)

---

## Telegram Structure

### Standard Frame Format

| Byte | Field | Description |
|------|-------|-------------|
| 0 | Control | Frame format, repeat, priority |
| 1-2 | Source Address | Sender individual address |
| 3-4 | Destination Address | Group or individual address |
| 5 | Routing | Address type, hop count, length |
| 6+ | TPDU/APDU | Transport/Application data |
| Last | Checksum | XOR of all bytes |

### Control Field (Byte 0)

| Bits | Field | Values |
|------|-------|--------|
| 7-6 | Frame Format | `10` = Standard, `00` = Extended |
| 5 | Repeat | `0` = Repeat on error, `1` = No repeat |
| 4 | System Broadcast | `0` = System, `1` = Normal |
| 3-2 | Priority | See below |
| 1-0 | Reserved | |

### Priority Values

| Code | Priority | Use Case |
|------|----------|----------|
| `00` | System | Configuration, management |
| `01` | High | Urgent control commands |
| `10` | Alarm | Safety-related |
| `11` | Normal | Default for most telegrams |

### Routing Field (Byte 5)

| Bits | Field | Description |
|------|-------|-------------|
| 7 | Address Type | `0` = Individual, `1` = Group |
| 6-4 | Hop Count | Decremented by routers |
| 3-0 | Payload Length | APDU length - 1 |

---

## Addressing

### Individual Address Format: `A.L.D`

| Part | Bits | Range | Description |
|------|------|-------|-------------|
| Area (A) | 4 | 0-15 | Building area |
| Line (L) | 4 | 0-15 | Line within area |
| Device (D) | 8 | 0-255 | Device on line |

**Binary Layout:** `AAAA.LLLL.DDDDDDDD` (16 bits total)

### Group Address Format: `M/S/G` (3-level)

| Part | Bits | Range | Description |
|------|------|-------|-------------|
| Main (M) | 5 | 0-31 | Main group |
| Sub (S) | 3 | 0-7 | Middle group |
| Group (G) | 8 | 0-255 | Sub group |

**Binary Layout:** `MMMMM.SSS.GGGGGGGG` (16 bits total)

---

## APDU Commands

| Code | Binary | Command | Description |
|------|--------|---------|-------------|
| 0x00 | `0000` | Value Read | Request current value |
| 0x01 | `0001` | Value Response | Reply with value |
| 0x02 | `0010` | Value Write | Set new value |
| 0x0A | `1010` | Memory Write | Write to device memory |

The command code is in the APCI field (bits 0-3 of first APDU byte).

---

## Datapoint Types (DPT)

DPTs define how to encode/decode data values. Format: `DPT X.YYY` where X is main type, YYY is subtype.

### DPT 1.xxx - Boolean (1 bit)

| DPT | Name | 0 | 1 |
|-----|------|---|---|
| 1.001 | Switch | Off | On |
| 1.002 | Bool | False | True |
| 1.003 | Enable | Disable | Enable |
| 1.005 | Alarm | No alarm | Alarm |
| 1.008 | Up/Down | Up | Down |
| 1.009 | Open/Close | Open | Close |

**Encoding:** LSB of single byte. Value `0x00` = 0, Value `0x01` = 1.

### DPT 3.xxx - Controlled Dimming (4 bits)

| Bits | Field | Description |
|------|-------|-------------|
| 3 | Direction | 0 = Decrease, 1 = Increase |
| 2-0 | Step | 0 = Break, 1-7 = Step size |

**Step codes:** `001`=100%, `010`=50%, `011`=25%, `100`=12.5%, `101`=6%, `110`=3%, `111`=1%

### DPT 5.xxx - Unsigned 8-bit (1 byte)

| DPT | Name | Range | Unit |
|-----|------|-------|------|
| 5.001 | Percentage | 0-100 | % |
| 5.003 | Angle | 0-360 | ° |
| 5.004 | Percent_U8 | 0-255 | % (scaled) |
| 5.010 | Value_1_Ucount | 0-255 | counter |

**Encoding:** Direct unsigned byte value.

### DPT 9.xxx - 2-byte Float

| DPT | Name | Range | Unit |
|-----|------|-------|------|
| 9.001 | Temperature | -273 to +670760 | °C |
| 9.002 | Temperature Diff | -670760 to +670760 | K |
| 9.004 | Lux | 0 to 670760 | lux |
| 9.005 | Wind Speed | 0 to 670760 | m/s |
| 9.007 | Humidity | 0 to 670760 | % |

**Encoding:** 
```
Byte 1: SEEEEMMM (S=sign, E=exponent, M=mantissa high)
Byte 2: MMMMMMMM (mantissa low)

Value = (0.01 * M) * 2^E
If S=1: Value = Value - (1 << E) * 20.48
```

### DPT 17.001 - Scene Number (1 byte)

| Range | Description |
|-------|-------------|
| 0-63 | Scene number (displayed as 1-64) |

**Encoding:** Direct byte, bits 0-5 = scene number.

### DPT 18.001 - Scene Control (1 byte)

| Bit 7 | Bits 0-5 | Action |
|-------|----------|--------|
| 0 | Scene (0-63) | Activate scene |
| 1 | Scene (0-63) | Learn/store scene |

### DPT 232.600 - RGB Color (3 bytes)

| Byte | Value | Range |
|------|-------|-------|
| 0 | Red | 0-255 |
| 1 | Green | 0-255 |
| 2 | Blue | 0-255 |

---

## Constants Used in Gray Logic

From `internal/bridges/knx/dpt.go` and `internal/knxd/manager.go`:

```go
// EIB Protocol Constants
const (
    EIB_OPEN_GROUPCON    uint16 = 0x0026  // Open group connection
    EIB_OPEN_T_GROUP     uint16 = 0x0022  // Open T_Group connection
    EIB_OPEN_T_CONNECTION uint16 = 0x0020 // Open point-to-point
    EIB_APDU_PACKET      uint16 = 0x0025  // APDU packet
)

// APCI Commands
const (
    APCI_READ     byte = 0x00  // GroupValue_Read
    APCI_RESPONSE byte = 0x40  // GroupValue_Response
    APCI_WRITE    byte = 0x80  // GroupValue_Write
)
```

---

## Validation Rules

These rules MUST be enforced by Gray Logic:

### Address Validation

| Check | Rule |
|-------|------|
| Individual Area | 0-15 |
| Individual Line | 0-15 |
| Individual Device | 0-255 |
| Group Main | 0-31 |
| Group Sub | 0-7 |
| Group Address | 0-255 |

### DPT Validation

| DPT | Validation |
|-----|------------|
| 1.xxx | Value must be 0 or 1 |
| 5.001 | Value must be 0-100 |
| 5.003 | Value must be 0-360 |
| 9.001 | Temperature must be >= -273.15°C |
| 17.001 | Scene must be 0-63 |
| 18.001 | Scene must be 0-63 |
| 232.600 | RGB values must be 0-255 each |

---

## References

### Official Specification

The complete KNX specification is available from [MyKNX Portal](https://my.knx.org/en/shop/knx-specifications). Key volumes:

| Volume | Content |
|--------|---------|
| 3/7/2 | Datapoint Types |
| 3/3/7 | Application Layer |
| 3/5/1 | Network Layer |

### ISO Standard

KNX is standardized as:
- ISO/IEC 14543-3 (Home Electronic Systems)
- EN 50090 (European building automation)
- ANSI/ASHRAE 135 (BACnet interoperability)

---

## TODO: Missing Information

The following needs to be added from official KNX specs:

- [ ] Complete DPT list (DPT 2.xxx, 4.xxx, 6.xxx, 7.xxx, 8.xxx, etc.)
- [ ] Extended frame format details
- [ ] Device descriptor read/response format
- [ ] Programming mode specifications
- [ ] KNX Secure protocol details
- [ ] cEMI frame format for KNX/IP
