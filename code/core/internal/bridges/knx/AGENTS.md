# AI Agent Guidance — KNX Bridge

> **MANDATORY: Read [`docs/protocols/knx-reference.md`](../../../../../docs/protocols/knx-reference.md) before ANY work on this package.**
>
> The KNX reference contains the official protocol specification extracted from KNX Standard v3.0.0.

---

## Prerequisites (Read In Order)

1. **[docs/protocols/knx-reference.md](../../../../../docs/protocols/knx-reference.md)** — KNX protocol specification (MANDATORY)
2. **[docs/CONSTRAINTS.md](../../../../../docs/CONSTRAINTS.md)** — Section 8.1 KNX rules
3. **[code/core/AGENTS.md](../../AGENTS.md)** — Go coding standards

---

## Package Purpose

This package implements the KNX protocol bridge for Gray Logic:

- **knxd Manager** (`manager.go`) — Subprocess lifecycle for knxd daemon
- **DPT Encoding** (`dpt.go`) — Datapoint Type encode/decode per KNX spec
- **Telegram Handling** — Parse/construct KNX telegrams via EIB protocol
- **MQTT Integration** — Translate between MQTT commands and KNX bus

---

## Specification Compliance Checklist

Before modifying ANY code in this package, verify against the spec:

### DPT Implementation (`dpt.go`)

| DPT | Spec Section | Key Points |
|-----|--------------|------------|
| 1.xxx | knx-reference.md §DPT 1.xxx | 1 bit, value 0 or 1 only |
| 3.xxx | knx-reference.md §DPT 3.xxx | 4 bits: direction (1 bit) + step code (3 bits) |
| 5.xxx | knx-reference.md §DPT 5.xxx | 8-bit unsigned, 5.001 scaled 0-100% |
| 9.xxx | knx-reference.md §DPT 9.xxx | 2-byte KNX float: `Value = (0.01 × M) × 2^E` |
| 232.600 | knx-reference.md §DPT 232.600 | 3 bytes RGB |

**Critical:** DPT 9.xxx uses KNX-specific float encoding, NOT IEEE 754!

```go
// CORRECT - KNX 2-byte float
// Byte 0: MEEE EMMM (sign, exponent, mantissa high)
// Byte 1: MMMM MMMM (mantissa low)
// Value = (0.01 × M) × 2^E

// WRONG - This is NOT how DPT 9 works
// float32 bits packed into 2 bytes
```

### Address Validation

From spec §Addressing:

```go
// Individual Address: A.L.D (16 bits)
// Area:   bits 15-12 (0-15)
// Line:   bits 11-8  (0-15)  
// Device: bits 7-0   (0-255)

// Group Address 3-level: M/S/G (16 bits)
// Main:  bits 15-11 (0-31)
// Sub:   bits 10-8  (0-7)
// Group: bits 7-0   (0-255)
```

### APCI Commands

From spec §Application Layer:

| Command | APCI Code | Use |
|---------|-----------|-----|
| GroupValue_Read | 0x0000 | Request value |
| GroupValue_Response | 0x0040 | Reply with value |
| GroupValue_Write | 0x0080 | Send new value |

### Invalid Data Handling

From spec:
- DPT 9.xxx: Value `0x7FFF` indicates invalid data — MUST handle
- DPT 5.006 (Tariff): Value `255` is reserved — MUST reject

---

## Testing Against Specification

### Unit Tests Must Verify

1. **Encoding correctness** — Test vectors from spec
2. **Decoding correctness** — Round-trip encode/decode
3. **Range validation** — Reject out-of-spec values
4. **Invalid data handling** — Proper error for `0x7FFF` etc.

### Example Test Vectors (from KNX Standard)

```go
// DPT 9.001 Temperature test vectors
var dpt9TestVectors = []struct {
    value float64
    bytes [2]byte
}{
    {0.0, [2]byte{0x00, 0x00}},
    {0.01, [2]byte{0x00, 0x01}},    // Minimum positive
    {20.48, [2]byte{0x0C, 0x00}},   // E=1, M=1024
    {-273.0, [2]byte{0xA9, 0xB6}},  // Absolute zero (edge case)
    // 0x7FFF = invalid data, should error
}
```

---

## Common Mistakes to Avoid

### 1. Wrong Float Encoding

```go
// WRONG: Using IEEE 754
func encodeDPT9(v float64) []byte {
    bits := math.Float32bits(float32(v))
    return []byte{byte(bits >> 8), byte(bits)}  // WRONG!
}

// CORRECT: Using KNX 2-byte float formula
func encodeDPT9(v float64) []byte {
    // See docs/protocols/knx-reference.md §DPT 9.xxx
    // Value = (0.01 × M) × 2^E
    // Find E and M that represent v
}
```

### 2. Ignoring Address Validation

```go
// WRONG: No validation
func sendToGroup(addr string, data []byte) error {
    ga := parseGroupAddress(addr)  // Could be invalid!
    return send(ga, data)
}

// CORRECT: Validate against spec
func sendToGroup(addr string, data []byte) error {
    ga, err := ParseGroupAddress(addr)
    if err != nil {
        return fmt.Errorf("invalid group address %q: %w", addr, err)
    }
    // Spec: Main 0-31, Sub 0-7, Group 0-255
    if ga.Main > 31 || ga.Sub > 7 {
        return fmt.Errorf("group address out of range: %s", addr)
    }
    return send(ga, data)
}
```

### 3. Not Handling Invalid DPT Values

```go
// WRONG: Assuming all values are valid
func decodeDPT9(data []byte) float64 {
    // decode and return
}

// CORRECT: Check for invalid marker
func decodeDPT9(data []byte) (float64, error) {
    if data[0] == 0x7F && data[1] == 0xFF {
        return 0, ErrInvalidData  // Spec: 0x7FFF = invalid
    }
    // decode and return
}
```

---

## Files in This Package

| File | Purpose | Spec Sections |
|------|---------|---------------|
| `manager.go` | knxd subprocess lifecycle | — |
| `manager_test.go` | Manager tests | — |
| `dpt.go` | DPT encoding/decoding | §DPT 1-232 |
| `dpt_test.go` | DPT tests with spec vectors | §DPT 1-232 |
| `address.go` | Address parsing/validation | §Addressing |
| `telegram.go` | Telegram construction | §Telegram Structure, §APCI |

---

## Related Documentation

- **[docs/protocols/knx-reference.md](../../../../../docs/protocols/knx-reference.md)** — Full KNX specification reference
- **[docs/protocols/mqtt.md](../../../../../docs/protocols/mqtt.md)** — MQTT topic structure
- **[docs/specs/knx/](../../../../../docs/specs/knx/)** — Original KNX Standard PDFs (not in repo)

---

## When in Doubt

1. Check `docs/protocols/knx-reference.md` first
2. If not covered, check the original PDF in `docs/specs/knx/`
3. Key documents:
   - `03_07_02 Datapoint Types v02.02.01 AS.pdf` — DPT encoding
   - `03_03_07 Application Layer v02.01.01 AS.pdf` — APCI commands
   - `03_08_04 Tunnelling v01.07.01 AS.pdf` — KNXnet/IP

**Never guess protocol behaviour. Always verify against specification.**
