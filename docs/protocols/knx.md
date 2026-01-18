---
title: KNX Protocol Specification
version: 1.0.0
status: active
implementation_status: specified
last_updated: 2026-01-17
depends_on:
  - architecture/system-overview.md
  - architecture/bridge-interface.md
---

# KNX Protocol Specification

This document specifies how Gray Logic integrates with KNX building automation systems. KNX is the primary field-layer protocol for switches, dimmers, sensors, blinds, and HVAC actuators.

---

## Overview

### What is KNX?

KNX is an international standard (ISO/IEC 14543-3) for building automation. It provides:

- **Decentralized intelligence** — Each device has its own processor
- **Bus-level communication** — Devices communicate directly without central controller
- **30+ year product availability** — Mature ecosystem, long-term support
- **Multi-manufacturer interoperability** — 500+ manufacturers, 8000+ certified products

### Why KNX for Gray Logic?

| Requirement | KNX Capability |
|-------------|----------------|
| **Physical controls always work** | Devices respond directly to bus commands, no Gray Logic dependency |
| **Open standard** | ISO/IEC 14543-3, no vendor lock-in |
| **10-year deployment** | Products available and supported for decades |
| **Professional quality** | Designed for commercial/high-end residential |

### Gray Logic's Role

Gray Logic **observes and enhances** KNX, never replaces it:

```
┌─────────────────────────────────────────────────────────────────┐
│                         KNX BUS                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │ Switch  │──│ Dimmer  │──│ Sensor  │──│ Blind   │──── ...    │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘            │
│       │            │            │            │                  │
│       └────────────┴────────────┴────────────┘                  │
│                           │                                      │
│                    Direct bus pairing                            │
│                    (works without Gray Logic)                    │
│                           │                                      │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                     ┌──────▼──────┐
                     │  KNX/IP     │
                     │  Interface  │
                     └──────┬──────┘
                            │
                     ┌──────▼──────┐
                     │   knxd      │
                     │  (daemon)   │
                     └──────┬──────┘
                            │
                     ┌──────▼──────┐
                     │ KNX Bridge  │
                     │(Gray Logic) │
                     └──────┬──────┘
                            │
                         MQTT
                            │
                     ┌──────▼──────┐
                     │ Gray Logic  │
                     │    Core     │
                     └─────────────┘
```

---

## KNX Fundamentals

### Physical Media

|       Medium        | Code |          Description          |           Use Case          |
|---------------------|------|-------------------------------|-----------------------------|
|   **Twisted Pair**  |  TP  | Bus cable, 9600 baud, powered | Primary installation medium |
|        **IP**       |  IP  |   Ethernet/WiFi, high speed   |  Backbone, remote access    |
| **Radio Frequency** |  RF  |       868 MHz wireless        |  Retrofit, battery devices  |
|    **Powerline**    |  PL  |       Over mains wiring       |      Retrofit (rare)        |

**Gray Logic requirement**: KNX/IP interface for connection to TP bus. Recommended: Weinzierl KNX IP Interface 731/732 or equivalent.

### Addressing

#### Physical Addresses

Format: `Area.Line.Device` (e.g., `1.2.15`)

- **Area**: 1-15 (building section)
- **Line**: 1-15 (bus segment)
- **Device**: 1-255 (individual device)

Physical addresses are assigned during ETS commissioning and identify specific hardware.

#### Group Addresses

Format varies by style:
- **3-level** (recommended): `Main/Middle/Sub` (e.g., `1/2/3`)
- **2-level**: `Main/Sub` (e.g., `1/23`)
- **Free**: Single number (e.g., `1234`)

Gray Logic uses **3-level addressing** as the standard format.

**Group Address Ranges** (recommended structure):

| Main Group |      Purpose       |             Example                |
|------------|--------------------|------------------------------------|
|     0      | System/diagnostics |         0/0/1 = Heartbeat          |
|     1      | Lighting switching |   1/0/1 = Living room main light   |
|     2      |  Lighting dimming  | 2/0/1 = Living room main dim value |
|     3      |   Blinds/shading   | 3/0/1 = Living room blind position |
|     4      |        HVAC        |  4/0/1 = Living room temperature   |
|     5      |      Sensors       |     5/0/1 = Living room motion     |
|     6      |  Status feedback   |  6/0/1 = Living room light status  |
|    7-14    |   Domain-specific  |             As needed              |
|     15     |        Scenes      |     15/0/1 = Scene recall          |

### Datapoint Types (DPT)

KNX defines standardized data formats. Gray Logic must handle these correctly:

#### Commonly Used DPTs

| DPT | Name | Size | Values | Use |
|-----|------|------|--------|-----|
| **1.001** | Switch | 1 bit | 0=Off, 1=On | Switching |
| **1.002** | Bool | 1 bit | 0=False, 1=True | Binary state |
| **1.003** | Enable | 1 bit | 0=Disable, 1=Enable | Enable/disable |
| **1.007** | Step | 1 bit | 0=Decrease, 1=Increase | Dimming step |
| **1.008** | UpDown | 1 bit | 0=Up, 1=Down | Blind direction |
| **1.009** | OpenClose | 1 bit | 0=Open, 1=Close | Contacts |
| **1.010** | Start | 1 bit | 0=Stop, 1=Start | Motor control |
| **1.017** | Trigger | 1 bit | 1=Trigger | Pulse trigger |
| **2.001** | Switch Control | 2 bit | Control + value | Forced switching |
| **3.007** | Dimming Control | 4 bit | Direction + steps | Relative dimming |
| **3.008** | Blind Control | 4 bit | Direction + steps | Relative blind |
| **5.001** | Percentage | 1 byte | 0-100% | Dim level, position |
| **5.003** | Angle | 1 byte | 0-360° | Blind tilt |
| **5.004** | Percentage U8 | 1 byte | 0-255 | Raw percentage |
| **5.010** | Counter Pulses | 1 byte | 0-255 | Pulse count |
| **6.001** | Percent V8 | 1 byte | -128 to 127% | Signed percentage |
| **7.001** | Pulses | 2 bytes | 0-65535 | Counter |
| **7.012** | Current mA | 2 bytes | 0-65535 mA | Current measurement |
| **8.001** | Pulses Diff | 2 bytes | -32768 to 32767 | Signed counter |
| **9.001** | Temperature | 2 bytes | -273 to 670760 °C | Temperature |
| **9.002** | Temp Diff | 2 bytes | -670760 to 670760 K | Temperature difference |
| **9.004** | Lux | 2 bytes | 0 to 670760 lux | Light level |
| **9.005** | Speed m/s | 2 bytes | 0 to 670760 m/s | Wind speed |
| **9.006** | Pressure Pa | 2 bytes | 0 to 670760 Pa | Air pressure |
| **9.007** | Humidity | 2 bytes | 0 to 670760 % | Relative humidity |
| **9.008** | Air Quality ppm | 2 bytes | 0 to 670760 ppm | CO2, VOC |
| **9.020** | Voltage mV | 2 bytes | -670760 to 670760 mV | Voltage |
| **9.021** | Current mA | 2 bytes | -670760 to 670760 mA | Current |
| **9.024** | Power kW | 2 bytes | -670760 to 670760 kW | Power |
| **10.001** | Time of Day | 3 bytes | Day, Hour, Min, Sec | Time |
| **11.001** | Date | 3 bytes | Day, Month, Year | Date |
| **12.001** | Counter 4 | 4 bytes | 0 to 4294967295 | Large counter |
| **13.001** | Counter 4 Signed | 4 bytes | ±2147483647 | Signed counter |
| **13.010** | Active Energy Wh | 4 bytes | ±2147483647 Wh | Energy meter |
| **13.013** | Active Energy kWh | 4 bytes | ±2147483647 kWh | Energy meter |
| **14.*** | 4-byte Float | 4 bytes | IEEE 754 float | Various measurements |
| **14.056** | Power W | 4 bytes | Float | Power measurement |
| **16.000** | ASCII String | 14 bytes | Text | Display text |
| **16.001** | Latin1 String | 14 bytes | Text | Display text |
| **17.001** | Scene Number | 1 byte | 0-63 | Scene recall |
| **18.001** | Scene Control | 1 byte | Scene + learn bit | Scene recall/learn |
| **20.102** | HVAC Mode | 1 byte | 0-4 | Comfort/Standby/Night/Frost/Auto |
| **232.600** | RGB Colour | 3 bytes | R, G, B | RGB value |

### Flags

Each group object has communication flags:

| Flag | Meaning | Typical Use |
|------|---------|-------------|
| **C** | Communication | Enable bus communication |
| **R** | Read | Object can be read |
| **W** | Write | Object can be written |
| **T** | Transmit | Object sends on change |
| **U** | Update | Object updates on read |
| **I** | Init | Read value at startup |

Gray Logic primarily uses:
- **Write** to send commands
- **Read** to request current state
- **Listens** to all telegrams with matching group addresses

---

## knxd Integration

### What is knxd?

[knxd](https://github.com/knxd/knxd) is an open-source KNX daemon that provides:

- Connection to KNX/IP interfaces and routers
- Connection to USB and serial interfaces
- Bus monitoring and logging
- Client API for applications

### Installation

```bash
# Debian/Ubuntu
sudo apt install knxd knxd-tools

# Build from source (for latest features)
git clone https://github.com/knxd/knxd.git
cd knxd
./bootstrap.sh
./configure --enable-eibnetipserver
make
sudo make install
```

### Configuration

**`/etc/knxd.ini`** (recommended configuration):

```ini
[main]
name = knxd
addr = 0.0.1
client-addrs = 0.0.2:10
connections = server,A.tcp

[A.tcp]
# KNX/IP Interface connection
driver = ipt
ip-address = 192.168.1.100
dest-port = 3671
nat = false

[server]
# Server for local clients (Gray Logic KNX Bridge)
server = ets_router
name = knxd_server
router = tunnel
tunnel = server
discover = true
```

### knxd Client Connection

Gray Logic KNX Bridge connects via:

1. **Unix socket**: `/run/knxd` (preferred for local)
2. **TCP**: `localhost:6720` (for remote or containerized)
3. **EIBnet/IP tunneling**: Direct to knxd's EIBnet/IP server

---

## KNX Bridge Specification

### Responsibilities

The KNX Bridge translates between KNX telegrams and Gray Logic's internal MQTT messages.

| Direction | Source | Destination | Action |
|-----------|--------|-------------|--------|
| **Inbound** | KNX bus | MQTT | Convert telegram → state update message |
| **Outbound** | MQTT | KNX bus | Convert command → telegram |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        KNX BRIDGE                                │
│                                                                  │
│  ┌────────────────┐     ┌──────────────────┐     ┌────────────┐ │
│  │  knxd Client   │────▶│  Message Router  │────▶│   MQTT     │ │
│  │  (connection)  │◀────│  & Translator    │◀────│  Client    │ │
│  └────────────────┘     └──────────────────┘     └────────────┘ │
│          │                       │                      │       │
│          │              ┌────────▼────────┐             │       │
│          │              │  Address Map    │             │       │
│          │              │  (GA → Device)  │             │       │
│          │              └─────────────────┘             │       │
│          │                       │                      │       │
│          ▼                       ▼                      ▼       │
│     KNX/IP Interface    Configuration DB         MQTT Broker    │
└─────────────────────────────────────────────────────────────────┘
```

### Configuration

The bridge requires a mapping of KNX group addresses to Gray Logic devices:

```yaml
# knx-bridge-config.yaml
bridge:
  id: "knx-bridge-01"
  knxd:
    connection: "unix:///run/knxd"
    # Or: connection: "tcp://localhost:6720"
  mqtt:
    broker: "tcp://localhost:1883"
    client_id: "knx-bridge-01"
    topic_prefix: "graylogic"

# Group address mappings
devices:
  - device_id: "light-living-main"
    type: "light_dimmer"
    addresses:
      switch:
        ga: "1/0/1"
        dpt: "1.001"
        flags: ["write"]
      switch_status:
        ga: "6/0/1"
        dpt: "1.001"
        flags: ["read", "transmit"]
      brightness:
        ga: "2/0/1"
        dpt: "5.001"
        flags: ["write"]
      brightness_status:
        ga: "6/0/2"
        dpt: "5.001"
        flags: ["read", "transmit"]

  - device_id: "blind-living-01"
    type: "blind_position"
    addresses:
      up_down:
        ga: "3/0/1"
        dpt: "1.008"
        flags: ["write"]
      stop:
        ga: "3/0/2"
        dpt: "1.017"
        flags: ["write"]
      position:
        ga: "3/0/3"
        dpt: "5.001"
        flags: ["write"]
      position_status:
        ga: "6/0/10"
        dpt: "5.001"
        flags: ["read", "transmit"]
      tilt:
        ga: "3/0/4"
        dpt: "5.001"
        flags: ["write"]
      tilt_status:
        ga: "6/0/11"
        dpt: "5.001"
        flags: ["read", "transmit"]

  - device_id: "sensor-living-temp"
    type: "temperature_sensor"
    addresses:
      temperature:
        ga: "5/0/1"
        dpt: "9.001"
        flags: ["read", "transmit"]

  - device_id: "sensor-living-motion"
    type: "motion_sensor"
    addresses:
      motion:
        ga: "5/0/2"
        dpt: "1.002"
        flags: ["read", "transmit"]
```

### MQTT Topic Structure

See [MQTT Protocol Specification](mqtt.md) for full topic structure. KNX Bridge uses:

**State updates (KNX → Core):**
```
graylogic/bridge/knx-bridge-01/state/{device_id}
```

**Commands (Core → KNX):**
```
graylogic/bridge/knx-bridge-01/command/{device_id}
```

**Bridge status:**
```
graylogic/bridge/knx-bridge-01/status
```

### Message Formats

**State update message (KNX → MQTT):**

```yaml
topic: graylogic/bridge/knx-bridge-01/state/light-living-main
payload:
  device_id: "light-living-main"
  timestamp: "2026-01-12T14:30:00Z"
  source: "knx"
  state:
    on: true
    brightness: 75
  raw:
    - ga: "6/0/1"
      dpt: "1.001"
      value: 1
    - ga: "6/0/2"
      dpt: "5.001"
      value: 75
```

**Command message (MQTT → KNX):**

```yaml
topic: graylogic/bridge/knx-bridge-01/command/light-living-main
payload:
  device_id: "light-living-main"
  command: "set"
  parameters:
    on: true
    brightness: 50
    transition_ms: 1000
  request_id: "req-12345"
```

**Bridge translates to KNX:**
1. Send to GA `2/0/1` value `50` (DPT 5.001) — brightness
2. Send to GA `1/0/1` value `1` (DPT 1.001) — switch on (if was off)

### Startup Behavior

1. Connect to knxd
2. Connect to MQTT broker
3. Publish bridge status: `online`
4. Subscribe to command topics
5. Request current state from all status GAs (read requests)
6. Begin listening for bus telegrams

### Error Handling

| Error | Action |
|-------|--------|
| knxd connection lost | Retry with exponential backoff, publish `degraded` status |
| MQTT connection lost | Queue messages locally, retry connection |
| Invalid GA in command | Log error, publish NACK to response topic |
| DPT encoding error | Log error, skip message |
| Telegram timeout | Log warning, mark request unconfirmed |

### Unconfirmed State behavior

If a command times out (no validation from a status GA), the UI displays a "Partial/Unconfirmed" indicator (e.g., a hollow icon or spinner). This persists until:
1. A status telegram is received (confirming state).
2. A subsequent poll or read request clarifies the state.
3. The user initiates a new command.

This ensures the user never sees a "Success" state that is not actually backed by device feedback.

### Health Reporting

Bridge publishes health status every 30 seconds:

```yaml
topic: graylogic/bridge/knx-bridge-01/health
payload:
  bridge_id: "knx-bridge-01"
  status: "online"  # online | degraded | offline
  timestamp: "2026-01-12T14:30:00Z"
  metrics:
    knxd_connected: true
    mqtt_connected: true
    telegrams_rx_total: 15432
    telegrams_tx_total: 8721
    errors_last_hour: 0
    uptime_seconds: 86400
  last_telegram:
    direction: "rx"
    ga: "5/0/1"
    timestamp: "2026-01-12T14:29:55Z"
```

---

## Device Type Mappings

### Lighting

| Gray Logic Type | KNX Objects Required | DPTs |
|-----------------|---------------------|------|
| `light_switch` | Switch, Switch Status | 1.001 |
| `light_dimmer` | Switch, Brightness, Status × 2 | 1.001, 5.001 |
| `light_ct` | Switch, Brightness, Color Temp, Status × 3 | 1.001, 5.001, 7.600 |
| `light_rgb` | Switch, Brightness, RGB, Status × 3 | 1.001, 5.001, 232.600 |
| `light_rgbw` | Switch, Brightness, RGBW, Status × 4 | 1.001, 5.001, 251.600 |

### Blinds

| Gray Logic Type | KNX Objects Required | DPTs |
|-----------------|---------------------|------|
| `blind_switch` | Up/Down, Stop | 1.008, 1.017 |
| `blind_position` | Up/Down, Stop, Position, Position Status | 1.008, 1.017, 5.001 |
| `blind_tilt` | All above + Tilt, Tilt Status | + 5.001 |

### Climate

| Gray Logic Type | KNX Objects Required | DPTs |
|-----------------|---------------------|------|
| `temperature_sensor` | Temperature | 9.001 |
| `humidity_sensor` | Humidity | 9.007 |
| `thermostat` | Temp, Setpoint, Mode, Valve | 9.001, 9.001, 20.102, 5.001 |
| `valve_actuator` | Position, Status | 5.001 |

### Sensors

| Gray Logic Type | KNX Objects Required | DPTs |
|-----------------|---------------------|------|
| `motion_sensor` | Motion | 1.002 |
| `presence_sensor` | Presence | 1.002 |
| `door_sensor` | Contact | 1.009 |
| `window_sensor` | Contact | 1.009 |
| `brightness_sensor` | Lux | 9.004 |
| `wind_sensor` | Speed | 9.005 |
| `rain_sensor` | Rain | 1.002 |

### Controls

| Gray Logic Type | KNX Objects Required | DPTs |
|-----------------|---------------------|------|
| `switch` | Switch | 1.001 |
| `keypad` | Multiple switch objects | 1.001 × n |
| `scene_controller` | Scene | 17.001 or 18.001 |

---

## ETS Integration

### Project Export

Gray Logic supports importing device configurations from ETS project exports:

1. **ETS XML Export** — Full project export (OPC format)
2. **ESF (ETS Symbol File)** — Group address list with names
3. **CSV Export** — Simple GA list

**Preferred**: ESF or structured CSV for initial import, with manual review.

### Address Naming Convention

Recommend structured naming in ETS for auto-mapping:

```
{Area}/{Room}/{Device}/{Function}

Examples:
Ground Floor/Living Room/Main Light/Switch
Ground Floor/Living Room/Main Light/Brightness
Ground Floor/Living Room/Main Light/Status
Ground Floor/Living Room/Blind 1/Position
```

### Programming Approach

1. **ETS handles device parameters** — All device configuration in ETS
2. **ETS handles direct links** — Switch-to-actuator links for "physical always works"
3. **Gray Logic monitors all** — Subscribes to all relevant GAs
4. **Gray Logic commands via GAs** — Sends commands to same GAs as physical devices

---

## Commissioning Checklist

### Prerequisites

- [ ] KNX installation completed and tested in ETS
- [ ] KNX/IP interface installed and accessible
- [ ] All devices have group addresses assigned
- [ ] Direct links programmed (switches → actuators)
- [ ] Status feedback addresses configured on actuators

### Gray Logic Integration

- [ ] knxd installed and configured
- [ ] knxd can connect to KNX/IP interface
- [ ] KNX Bridge configuration created
- [ ] Group addresses mapped to Gray Logic devices
- [ ] All status GAs verified readable
- [ ] All command GAs verified writable
- [ ] Bus monitor confirms telegram flow

### Testing

- [ ] Physical switch operates light (without Gray Logic)
- [ ] Gray Logic sees status change
- [ ] Gray Logic command operates light
- [ ] Status feedback received by Gray Logic
- [ ] Scene recall works from keypad
- [ ] Scene recall works from Gray Logic

---

## Troubleshooting

### Common Issues

| Symptom | Possible Cause | Solution |
|---------|---------------|----------|
| No telegrams received | knxd not connected | Check `/var/log/knxd.log` |
| Commands not working | Wrong GA or DPT | Verify ETS configuration |
| Status not updating | No status GA configured | Add feedback objects in ETS |
| Intermittent connection | Network issues | Check IP interface, switches |
| Slow response | Bus overload | Reduce telegram frequency |

### Diagnostic Commands

```bash
# Check knxd status
systemctl status knxd

# Monitor bus traffic
knxtool busmonitor1 ip:localhost

# Send test telegram
knxtool groupwrite ip:localhost 1/0/1 1

# Read group address
knxtool groupread ip:localhost 6/0/1

# Check knxd connections
knxtool info ip:localhost
```

---

## Security Considerations

### Bus Security

- KNX Secure (Data Secure, IP Secure) supported by modern devices
- Recommend KNX IP Secure for backbone connections
- Gray Logic stores keys securely if KNX Secure is used

### Network Security

- KNX/IP interface should be on Control VLAN only
- No direct internet access to KNX infrastructure
- knxd accessible only from Gray Logic server

### Physical Security

- KNX bus cable in secure locations
- No exposed bus connectors in public areas
- BCU keys stored securely

---

## Hardware Recommendations

### KNX/IP Interfaces

| Manufacturer | Model | Features | Notes |
|--------------|-------|----------|-------|
| Weinzierl | KNX IP Interface 731 | Tunneling, routing | Reliable, well-supported |
| Weinzierl | KNX IP BAOS 774 | + Object server | Object access without GA |
| MDT | IP Interface | Tunneling | Good value |
| ABB | IPR/S 3.5.1 | Routing, tunneling | Enterprise-grade |
| Siemens | N 148/22 | Tunneling | Enterprise-grade |

### Actuators (Recommended)

| Category | Recommended Brands |
|----------|-------------------|
| Switching | ABB, MDT, Theben |
| Dimming | MDT, ABB, Gira |
| Blinds | MDT, ABB, Warema |
| HVAC | ABB, Siemens, Theben |
| Sensors | Theben, B.E.G., Steinel |

---

## Related Documents

- [MQTT Protocol Specification](mqtt.md) — Internal message bus
- [DALI Protocol Specification](dali.md) — Alternative lighting protocol
- [Bridge Interface Specification](../architecture/bridge-interface.md) — Bridge contract
- [Lighting Domain Specification](../domains/lighting.md) — Lighting features

