---
title: Modbus Protocol Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - architecture/bridge-interface.md
---

# Modbus Protocol Specification

This document specifies how Gray Logic integrates with Modbus devices for plant equipment, energy monitoring, HVAC systems, and industrial equipment.

---

## Overview

### What is Modbus?

Modbus is a serial communication protocol published by Modicon in 1979. It has become a de facto standard for industrial communication:

- **Simple** — Easy to implement, well-documented
- **Universal** — Supported by virtually all industrial equipment
- **Variants** — RTU (serial), TCP (Ethernet), ASCII (text)
- **Open** — No licensing, public specification

### Why Modbus for Gray Logic?

| Requirement | Modbus Capability |
|-------------|-------------------|
| **Plant equipment** | Heat pumps, boilers, VFDs, pool equipment |
| **Energy monitoring** | Power meters, CT clamps, solar inverters |
| **HVAC** | Commercial equipment, BMS integration |
| **Open standard** | Widely implemented, vendor-neutral |
| **Multi-decade stability** | Unchanged protocol, long-term support |

### Modbus Variants

| Variant | Medium | Speed | Format | Use Case |
|---------|--------|-------|--------|----------|
| **RTU** | RS-485/232 | 9600-115200 | Binary | Serial equipment |
| **TCP** | Ethernet | 10/100 Mbps | Binary | Network equipment |
| **ASCII** | RS-485/232 | 300-19200 | ASCII | Legacy equipment |

**Gray Logic supports**: Modbus RTU and Modbus TCP (not ASCII).

---

## Modbus Fundamentals

### Architecture

```
                                      Modbus Network
                                            │
     ┌──────────────┬───────────────────────┼───────────────────────┐
     │              │                       │                       │
┌────▼────┐   ┌─────▼─────┐           ┌─────▼─────┐           ┌─────▼─────┐
│ Master  │   │  Slave 1  │           │  Slave 2  │           │  Slave N  │
│(Client) │   │ Unit ID 1 │           │ Unit ID 2 │           │ Unit ID N │
└─────────┘   └───────────┘           └───────────┘           └───────────┘
Gray Logic        Meter                Heat Pump                  VFD
```

- **Master/Client**: Initiates requests (Gray Logic Modbus Bridge)
- **Slave/Server**: Responds to requests (devices)
- **Unit ID**: Device address (1-247, 0=broadcast)

### Register Types

Modbus defines four types of data:

| Type | Address Range | Access | Size | Description |
|------|---------------|--------|------|-------------|
| **Coils** | 00001-09999 | Read/Write | 1 bit | Digital outputs |
| **Discrete Inputs** | 10001-19999 | Read | 1 bit | Digital inputs |
| **Input Registers** | 30001-39999 | Read | 16 bit | Analog inputs |
| **Holding Registers** | 40001-49999 | Read/Write | 16 bit | Parameters |

**Note**: Address ranges are conventional. Actual addresses are 0-based in protocol.

### Function Codes

| Code | Name | Operation |
|------|------|-----------|
| 01 | Read Coils | Read 1-2000 coils |
| 02 | Read Discrete Inputs | Read 1-2000 inputs |
| 03 | Read Holding Registers | Read 1-125 registers |
| 04 | Read Input Registers | Read 1-125 registers |
| 05 | Write Single Coil | Write 1 coil |
| 06 | Write Single Register | Write 1 register |
| 15 | Write Multiple Coils | Write 1-1968 coils |
| 16 | Write Multiple Registers | Write 1-123 registers |
| 23 | Read/Write Multiple Registers | Read + Write |

### Data Types

Modbus registers are 16-bit. Larger types span multiple registers:

| Data Type | Size | Registers | Byte Order |
|-----------|------|-----------|------------|
| **UINT16** | 16 bit | 1 | - |
| **INT16** | 16 bit | 1 | - |
| **UINT32** | 32 bit | 2 | Big or Little Endian |
| **INT32** | 32 bit | 2 | Big or Little Endian |
| **FLOAT32** | 32 bit | 2 | IEEE 754 |
| **UINT64** | 64 bit | 4 | Big or Little Endian |
| **FLOAT64** | 64 bit | 4 | IEEE 754 |
| **String** | Variable | N | ASCII |

**Byte Order Variations** (CRITICAL):

| Order | Name | Register Order | Byte Order |
|-------|------|----------------|------------|
| ABCD | Big Endian | High-Low | High-Low |
| DCBA | Little Endian | Low-High | Low-High |
| BADC | Mid-Big Endian | High-Low | Low-High |
| CDAB | Mid-Little Endian | Low-High | High-Low |

**Device documentation must be consulted** for correct byte order.

---

## Modbus RTU

### Physical Layer

| Parameter | Typical Value |
|-----------|---------------|
| **Standard** | TIA/EIA-485 (RS-485) |
| **Wiring** | 2-wire (A, B) + GND, or 4-wire |
| **Baud Rate** | 9600, 19200, 38400, 115200 |
| **Data Bits** | 8 |
| **Parity** | None, Even, or Odd |
| **Stop Bits** | 1 or 2 |
| **Termination** | 120Ω at each end |

### Frame Format

```
┌────────┬──────────────┬─────────────────┬──────────┐
│ Slave  │   Function   │      Data       │   CRC    │
│Address │    Code      │   (variable)    │  16-bit  │
│ 1 byte │   1 byte     │   0-252 bytes   │  2 bytes │
└────────┴──────────────┴─────────────────┴──────────┘
```

### Timing

- **Inter-character timeout**: 1.5 character times
- **Inter-frame delay**: 3.5 character times
- **Response timeout**: 100-500ms (device-dependent)

### RS-485 Adapters

| Type | Interface | Notes |
|------|-----------|-------|
| **USB-RS485** | USB | Most flexible |
| **Ethernet-RS485** | TCP | Remote access |
| **Embedded** | GPIO | SBC with UART |

**Recommended**: FTDI-based USB adapters for reliability.

---

## Modbus TCP

### Transport

| Parameter | Value |
|-----------|-------|
| **Port** | 502 (default) |
| **Protocol** | TCP/IP |
| **Connections** | Multiple clients supported |

### Frame Format

```
┌──────────────────────────────────────────────────────────────────┐
│                       MBAP Header                                 │
├──────────────┬──────────────┬──────────────┬──────────────────────┤
│ Transaction  │  Protocol    │    Length    │    Unit ID           │
│    ID        │     ID       │              │                      │
│   2 bytes    │   2 bytes    │   2 bytes    │    1 byte            │
│   (seq #)    │   (0x0000)   │  (bytes after)│  (Slave addr)       │
└──────────────┴──────────────┴──────────────┴──────────────────────┘
                                      │
                           ┌──────────▼──────────────────┐
                           │        PDU                  │
                           ├─────────────┬───────────────┤
                           │  Function   │    Data       │
                           │    Code     │               │
                           │   1 byte    │  0-252 bytes  │
                           └─────────────┴───────────────┘
```

### Unit ID

- **Unit ID 255**: For gateways, access serial devices behind TCP gateway
- **Unit ID 0-247**: Direct device addressing
- **Unit ID 0**: Broadcast (rarely used in TCP)

---

## Modbus Bridge Specification

### Responsibilities

| Direction | Source | Destination | Action |
|-----------|--------|-------------|--------|
| **Polling** | Bridge | Devices | Periodically read registers |
| **Commands** | MQTT | Devices | Write registers on command |
| **State** | Devices | MQTT | Publish register values |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      MODBUS BRIDGE                               │
│                                                                  │
│  ┌────────────────┐     ┌──────────────────┐     ┌────────────┐ │
│  │ Modbus Client  │────▶│   Poll Manager   │────▶│   MQTT     │ │
│  │ (RTU or TCP)   │◀────│   & Translator   │◀────│  Client    │ │
│  └────────────────┘     └──────────────────┘     └────────────┘ │
│          │                       │                      │       │
│          │              ┌────────▼────────┐             │       │
│          │              │  Device         │             │       │
│          │              │  Profiles       │             │       │
│          │              │  (Register Maps)│             │       │
│          │              └─────────────────┘             │       │
│          ▼                       ▼                      ▼       │
│   RS-485 Bus or            Configuration          MQTT Broker   │
│   TCP Network                  DB                               │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

```yaml
# modbus-bridge-config.yaml
bridge:
  id: "modbus-bridge-01"
  mqtt:
    broker: "tcp://localhost:1883"
    client_id: "modbus-bridge-01"
    topic_prefix: "graylogic"

# RTU Connection
connections:
  - id: "rtu-plant"
    type: "rtu"
    port: "/dev/ttyUSB0"
    baudrate: 9600
    parity: "N"
    stopbits: 1
    timeout_ms: 500

  - id: "tcp-meters"
    type: "tcp"
    host: "192.168.1.120"
    port: 502
    timeout_ms: 1000

devices:
  - device_id: "meter-main"
    connection: "tcp-meters"
    unit_id: 1
    type: "energy_meter"
    profile: "eastron-sdm630"
    poll_interval_ms: 5000

  - device_id: "heat-pump-01"
    connection: "rtu-plant"
    unit_id: 10
    type: "heat_pump"
    profile: "custom"
    poll_interval_ms: 10000
    registers:
      # Read registers
      - name: "outdoor_temp"
        address: 100
        type: "input"
        datatype: "int16"
        scale: 0.1
        unit: "°C"
        
      - name: "water_temp"
        address: 101
        type: "input"
        datatype: "int16"
        scale: 0.1
        unit: "°C"
        
      - name: "power"
        address: 110
        type: "input"
        datatype: "uint32"
        byte_order: "ABCD"
        unit: "W"
        
      - name: "cop"
        address: 120
        type: "input"
        datatype: "float32"
        byte_order: "ABCD"
        
      - name: "status"
        address: 200
        type: "input"
        datatype: "uint16"
        
      # Write registers
      - name: "setpoint"
        address: 300
        type: "holding"
        datatype: "int16"
        scale: 0.1
        unit: "°C"
        writable: true
        
      - name: "mode"
        address: 301
        type: "holding"
        datatype: "uint16"
        writable: true
        values:
          0: "off"
          1: "heating"
          2: "cooling"
          3: "auto"

  - device_id: "pool-pump-01"
    connection: "rtu-plant"
    unit_id: 20
    type: "pump"
    profile: "custom"
    poll_interval_ms: 5000
    registers:
      - name: "running"
        address: 0
        type: "coil"
        writable: true
        
      - name: "speed"
        address: 100
        type: "holding"
        datatype: "uint16"
        unit: "%"
        writable: true
        
      - name: "current"
        address: 200
        type: "input"
        datatype: "float32"
        byte_order: "ABCD"
        unit: "A"
        
      - name: "runtime_hours"
        address: 210
        type: "input"
        datatype: "uint32"
        byte_order: "ABCD"
        unit: "h"
```

### Device Profiles

Pre-defined profiles for common equipment:

```yaml
# profiles/eastron-sdm630.yaml
name: "Eastron SDM630 Energy Meter"
manufacturer: "Eastron"
model: "SDM630"
protocol: "modbus_tcp"
unit_id: 1
byte_order: "ABCD"

registers:
  - name: "voltage_l1"
    address: 0
    type: "input"
    datatype: "float32"
    unit: "V"
    
  - name: "voltage_l2"
    address: 2
    type: "input"
    datatype: "float32"
    unit: "V"
    
  - name: "voltage_l3"
    address: 4
    type: "input"
    datatype: "float32"
    unit: "V"
    
  - name: "current_l1"
    address: 6
    type: "input"
    datatype: "float32"
    unit: "A"
    
  - name: "current_l2"
    address: 8
    type: "input"
    datatype: "float32"
    unit: "A"
    
  - name: "current_l3"
    address: 10
    type: "input"
    datatype: "float32"
    unit: "A"
    
  - name: "power_total"
    address: 52
    type: "input"
    datatype: "float32"
    unit: "W"
    
  - name: "power_l1"
    address: 12
    type: "input"
    datatype: "float32"
    unit: "W"
    
  - name: "power_l2"
    address: 14
    type: "input"
    datatype: "float32"
    unit: "W"
    
  - name: "power_l3"
    address: 16
    type: "input"
    datatype: "float32"
    unit: "W"
    
  - name: "import_energy"
    address: 72
    type: "input"
    datatype: "float32"
    unit: "kWh"
    
  - name: "export_energy"
    address: 74
    type: "input"
    datatype: "float32"
    unit: "kWh"
    
  - name: "frequency"
    address: 70
    type: "input"
    datatype: "float32"
    unit: "Hz"
    
  - name: "power_factor"
    address: 62
    type: "input"
    datatype: "float32"
```

### MQTT Topics

**State updates (periodic polling):**
```
graylogic/state/modbus/{address}
```

**Commands (write to registers):**
```
graylogic/command/modbus/{address}
```

### Message Formats

**State update (polled values):**

```yaml
topic: graylogic/state/modbus/meter-main
payload:
  device_id: "meter-main"
  timestamp: "2026-01-12T14:30:00Z"
  source: "modbus"
  state:
    voltage_l1: 239.5
    voltage_l2: 240.1
    voltage_l3: 238.8
    current_l1: 12.3
    current_l2: 8.7
    current_l3: 15.2
    power_total: 8450
    power_l1: 2945
    power_l2: 2087
    power_l3: 3418
    import_energy: 45678.9
    export_energy: 12345.6
    frequency: 50.02
    power_factor: 0.95
  raw:
    # Original register values for debugging
    - address: 0
      value: [0x43, 0x6F, 0x80, 0x00]  # voltage_l1
```

**Command message:**

```yaml
topic: graylogic/command/modbus/heat-pump-01
payload:
  device_id: "heat-pump-01"
  command: "set"
  parameters:
    setpoint: 45.0
    mode: "heating"
  request_id: "req-99999"
```

**Command response:**

```yaml
topic: graylogic/response/modbus/req-99999
payload:
  request_id: "req-99999"
  device_id: "heat-pump-01"
  success: true
  timestamp: "2026-01-12T14:30:01Z"
```

---

## Common Device Profiles

### Energy Meters

| Manufacturer | Model | Features |
|--------------|-------|----------|
| Eastron | SDM120/230/630 | Single/three-phase, budget |
| Schneider | PM5xxx | Professional, extensive data |
| ABB | B2x series | DIN rail, reliable |
| Carlo Gavazzi | EM series | Industrial, accurate |

### Solar Inverters

| Manufacturer | Models | Protocol |
|--------------|--------|----------|
| SolarEdge | All | Modbus TCP (SunSpec) |
| Fronius | All | Modbus TCP (SunSpec) |
| SMA | All | Modbus TCP |
| Huawei | SUN2000 | Modbus TCP |
| GoodWe | All | Modbus TCP |

### Battery Storage

| Manufacturer | Models | Protocol |
|--------------|--------|----------|
| BYD | All | Modbus TCP (via inverter) |
| Pylontech | All | Modbus RTU/TCP |
| Tesla | Powerwall | Modbus TCP (via gateway) |

### Heat Pumps

| Manufacturer | Models | Protocol | Notes |
|--------------|--------|----------|-------|
| Vaillant | aroTHERM | Modbus TCP | Via eBus gateway |
| Daikin | Altherma | Modbus RTU | Optional module |
| Mitsubishi | Ecodan | Modbus RTU | FTC controller |
| NIBE | F-series | Modbus TCP | Native |
| Samsung | EHS | Modbus RTU | NASA interface |

### VFDs (Variable Frequency Drives)

| Manufacturer | Series | Protocol |
|--------------|--------|----------|
| ABB | ACS series | Modbus RTU |
| Siemens | SINAMICS | Modbus RTU/TCP |
| Danfoss | VLT | Modbus RTU/TCP |
| Schneider | Altivar | Modbus RTU/TCP |

### Pool Equipment

| Manufacturer | Equipment | Protocol |
|--------------|-----------|----------|
| Pentair | IntelliFlo | Modbus RTU |
| Hayward | VS pumps | Modbus RTU |
| Zodiac | Various | Modbus RTU |

---

## Polling Strategy

### Poll Intervals

| Data Type | Interval | Rationale |
|-----------|----------|-----------|
| **Power/Current** | 5s | Fast-changing, display updates |
| **Temperature** | 30s | Slow-changing |
| **Energy totals** | 60s | Cumulative, rarely urgent |
| **Status/Faults** | 10s | Operational awareness |
| **Configuration** | On demand | Rarely changes |

### Optimizing Bus Traffic

1. **Read contiguous registers** — Single request vs. multiple
2. **Stagger device polling** — Avoid bus congestion
3. **Change-based reporting** — Only publish on significant change
4. **Coalesce writes** — Batch multiple register writes

### Example Polling Schedule

```yaml
polling_groups:
  - name: "fast_power"
    devices: ["meter-main", "meter-solar"]
    interval_ms: 5000
    registers: ["power_*", "current_*"]
    
  - name: "medium_temps"
    devices: ["heat-pump-01", "heat-pump-02"]
    interval_ms: 30000
    registers: ["*_temp", "setpoint"]
    
  - name: "slow_energy"
    devices: ["meter-main", "meter-solar"]
    interval_ms: 60000
    registers: ["*_energy"]
    
  - name: "status"
    devices: ["*"]
    interval_ms: 10000
    registers: ["status", "fault*", "alarm*"]
```

---

## Error Handling

### Exception Codes

| Code | Name | Meaning |
|------|------|---------|
| 01 | Illegal Function | Function not supported |
| 02 | Illegal Data Address | Address out of range |
| 03 | Illegal Data Value | Value out of range |
| 04 | Slave Device Failure | Device internal error |
| 05 | Acknowledge | Request accepted, processing |
| 06 | Slave Device Busy | Device busy, retry |
| 08 | Memory Parity Error | Memory error |
| 0A | Gateway Path Unavailable | Gateway routing error |
| 0B | Gateway Target Failed | Target device unresponsive |

### Retry Strategy

```yaml
retry:
  max_attempts: 3
  initial_delay_ms: 100
  backoff_multiplier: 2
  max_delay_ms: 5000
  
  # Exception-specific handling
  exceptions:
    busy: 
      delay_ms: 1000
      max_attempts: 5
    gateway_target_failed:
      delay_ms: 5000
      max_attempts: 2
```

### Device Health

| Condition | Detection | Action |
|-----------|-----------|--------|
| No response | Timeout | Mark offline, alert |
| CRC error | Validation | Retry, log error rate |
| Exception | Response code | Handle per exception |
| Stale data | Timestamp | Alert if > threshold |

---

## PHM Integration

Modbus provides rich data for Predictive Health Monitoring:

### Energy Analysis

- **Power baseline** — Learn normal consumption patterns
- **Power factor** — Detect motor degradation
- **THD** — Harmonic distortion indicates problems

### Equipment Monitoring

- **Runtime hours** — Maintenance scheduling
- **Start counts** — Cycle tracking
- **Error counters** — Fault trending
- **Temperature rise** — Degradation indicator

### Alerting Rules

```yaml
phm_rules:
  - device_type: "pump"
    metric: "current"
    condition: "deviation_from_baseline"
    threshold_percent: 20
    duration_minutes: 60
    severity: "warning"
    message: "Pump current deviation may indicate bearing wear"
    
  - device_type: "energy_meter"
    metric: "power_factor"
    condition: "below"
    threshold: 0.85
    duration_minutes: 30
    severity: "warning"
    message: "Low power factor may indicate motor issues"
```

---

## Security Considerations

### Network Security

- Modbus TCP devices on Control VLAN only
- No direct internet access
- Firewall rules restrict access to bridge only

### Protocol Limitations

- **No authentication** — Modbus has no built-in security
- **No encryption** — All traffic is plaintext
- **Trust network security** — Rely on VLAN isolation

### Secure Alternatives

| Alternative | Description |
|-------------|-------------|
| **Modbus/TCP over TLS** | Vendor-specific implementations |
| **VPN** | Tunnel Modbus over encrypted VPN |
| **Secure gateway** | Bridge to secure protocol |

---

## Commissioning Checklist

### RTU Setup

- [ ] RS-485 wiring correct (A/B polarity)
- [ ] Termination resistors installed
- [ ] Baud rate matches all devices
- [ ] Parity/stop bits consistent
- [ ] Unit IDs unique per bus
- [ ] USB adapter recognized (`/dev/ttyUSB0`)

### TCP Setup

- [ ] Devices have static IPs
- [ ] Devices on Control VLAN
- [ ] Port 502 accessible
- [ ] Firewall rules configured

### Device Configuration

- [ ] Register map documented
- [ ] Byte order verified
- [ ] Scale factors confirmed
- [ ] Write registers tested
- [ ] Polling interval appropriate

### Gray Logic Integration

- [ ] Bridge configuration created
- [ ] Devices mapped correctly
- [ ] Initial polling successful
- [ ] Values match device display
- [ ] Commands verified

---

## Troubleshooting

### Common Issues

| Symptom | Possible Cause | Solution |
|---------|---------------|----------|
| No response | Wrong unit ID | Verify device address |
| CRC errors | Wiring issues | Check RS-485 connections |
| Wrong values | Byte order | Try different byte orders |
| Timeouts | Slow device | Increase timeout |
| Intermittent | Bus conflicts | Stagger polling |

### Diagnostic Tools

```bash
# Linux: mbpoll (Modbus CLI)
sudo apt install mbpoll

# Read holding registers 0-9 from unit 1
mbpoll -a 1 -r 0 -c 10 /dev/ttyUSB0

# Read input registers via TCP
mbpoll -a 1 -r 0 -c 10 -t 3 192.168.1.120

# Write single register
mbpoll -a 1 -r 300 -t 4 192.168.1.120 -- 450

# Continuous monitoring
mbpoll -a 1 -r 0 -c 10 -l 1000 /dev/ttyUSB0
```

---

## Related Documents

- [KNX Protocol Specification](knx.md) — Complementary protocol
- [MQTT Protocol Specification](mqtt.md) — Internal message bus
- [Energy Domain Specification](../domains/energy.md) — Energy features
- [Plant Domain Specification](../domains/plant.md) — Equipment monitoring
- [Bridge Interface Specification](../architecture/bridge-interface.md) — Bridge contract

