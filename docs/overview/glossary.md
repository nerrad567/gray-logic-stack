---
title: Gray Logic Glossary
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on: []
---

# Gray Logic Glossary

Standard terminology used throughout Gray Logic documentation and code.

---

## Core Concepts

### Gray Logic
The company, brand, and product ecosystem. Encompasses the software, hardware recommendations, installation methodology, and support services.

### Gray Logic Core
The central software component — a custom Go-based service that manages all automation logic, device state, and user interfaces. This is the "brain" of a Gray Logic installation.

### Site
A single Gray Logic installation. Typically one property (house, building, or facility). Each site has one Gray Logic Core instance.

### Area
A logical grouping within a site. Examples: "Ground Floor", "First Floor", "Pool House", "Garden". Areas contain rooms.

### Room
A physical space within an area. Examples: "Living Room", "Master Bedroom", "Kitchen". Rooms contain devices and define audio/climate zones.

### Device
Any controllable or monitorable entity. Lights, switches, sensors, blinds, thermostats, audio zones, cameras, etc.

### Protocol Bridge
A component that translates between Gray Logic Core's internal representation and a specific physical protocol (KNX, DALI, Modbus, etc.).

---

## Automation

### Scene
A predefined collection of device states that can be activated together. Examples: "Cinema Mode", "Good Night", "Away". Scenes can include timing, sequences, and conditions.

### Mode
A system-wide state that affects automation behaviour. Standard modes: Home, Away, Night, Holiday. Modes influence how other automation responds.

### Schedule
A time-based trigger for automation. Can be clock time, sunrise/sunset relative, or calendar-based.

### Event
Something that happens in the system. Device state changes, button presses, sensor triggers, mode changes. Events can trigger automation.

### Trigger
A condition that initiates automation. Can be: manual (button press), scheduled (time-based), event-driven (sensor), or compound (multiple conditions).

### Action
A single operation performed by automation. Examples: "set light to 50%", "close blind", "play audio in kitchen".

### Condition
A prerequisite for automation to execute. Examples: "only if mode is Home", "only between 6am and 10pm", "only if room is occupied".

---

## Device Types

### Actuator
A device that performs physical actions. Examples: light dimmer, relay, blind motor, valve.

### Sensor
A device that measures or detects. Examples: temperature sensor, motion detector, door contact, light level sensor.

### Controller
A device that provides user input. Examples: keypad, wall switch, touchscreen, remote control.

### Gateway
A device that bridges protocols. Examples: KNX/IP interface, DALI gateway, Modbus/TCP converter.

---

## Protocols

### KNX
International standard for building automation (ISO/IEC 14543-3). Used for switches, dimmers, sensors, blinds, HVAC. Supports both bus cable (twisted pair) and IP.

### DALI
Digital Addressable Lighting Interface (IEC 62386). Protocol for lighting control, primarily used for dimming and colour control of LED drivers and ballasts.

### Modbus
Serial communication protocol common in industrial applications. Used for plant equipment, energy meters, and HVAC systems. Variants: Modbus RTU (serial), Modbus TCP (Ethernet).

### MQTT
Lightweight message protocol. Used internally within Gray Logic for communication between Core and bridges. Topic-based publish/subscribe model.

### SIP
Session Initiation Protocol. Standard for VoIP communications. Used for door stations, intercoms, and internal communication.

### RTSP
Real Time Streaming Protocol. Used for live video streams from cameras and NVRs.

### ONVIF
Open standard for IP-based security devices. Provides discovery, configuration, and streaming interfaces for cameras and NVRs.

---

## Intelligence

### PHM (Predictive Health Monitoring)
System for detecting equipment problems before failure. Uses statistical analysis of operational data to identify deviations from normal behaviour.

### Baseline
The learned "normal" behaviour of a monitored asset. Calculated from historical data (e.g., 7-day rolling average).

### Deviation
When current behaviour differs significantly from baseline. Sustained deviation triggers PHM alerts.

### Local AI
On-device artificial intelligence processing. Includes voice recognition, natural language understanding, and pattern detection. No cloud dependency.

### NLU (Natural Language Understanding)
The ability to interpret human language commands. Converts "turn on the kitchen lights" into executable actions.

### Wake Word
A specific phrase that activates voice command listening. Default: "Hey Gray". Processed locally, always.

---

## Resilience

### Offline-First
Design principle where full functionality is available without internet. Remote features enhance but never enable core operation.

### Graceful Degradation
System behaviour when components fail. Higher-level features fail first; manual control never fails.

### Golden Backup
Complete snapshot of system configuration at commissioning. Used for factory reset or disaster recovery.

### Operational Backup
Periodic snapshot of running configuration including user customizations. Used for rollback after changes.

### Satellite Weather
Direct reception of NOAA/EUMETSAT satellite imagery. Provides weather data without internet dependency.

### LoRa Mesh
Long-range, low-power radio communication. Used for building-to-building communication without LAN/internet.

---

## Security

### Role
A set of permissions assigned to users. Standard roles: Admin, Resident, Guest, Installer.

### Authentication
Verifying identity. Methods: PIN, password, certificate, biometric.

### Authorization
Verifying permission. Determines what an authenticated user can do.

### Audit Log
Record of security-relevant actions. Who did what, when, from where.

### VPN
Virtual Private Network. Remote access to site uses WireGuard VPN exclusively.

### Sensitive Action
An action requiring additional confirmation. Examples: arming/disarming security, unlocking doors remotely, changing modes.

---

## Business

### Support Tier
Level of ongoing support service. Tiers: Core Support, Enhanced Support, Premium Support.

### Commissioning
The process of installing, configuring, and handing over a Gray Logic system.

### Handover Pack
Complete documentation provided to customer at commissioning. Includes credentials, diagrams, runbooks, and "doomsday" recovery information.

### Doomsday Pack
Sealed documentation enabling system recovery or handover if Gray Logic becomes unavailable. Contains full credentials and instructions.

---

## Abbreviations

| Abbrev | Meaning |
|--------|---------|
| AHU | Air Handling Unit |
| API | Application Programming Interface |
| CEC | Consumer Electronics Control (HDMI) |
| CT | Current Transformer (for energy monitoring) |
| DHW | Domestic Hot Water |
| HVAC | Heating, Ventilation, and Air Conditioning |
| LAN | Local Area Network |
| LED | Light Emitting Diode |
| LLM | Large Language Model |
| MVHR | Mechanical Ventilation with Heat Recovery |
| NVR | Network Video Recorder |
| PIR | Passive Infrared (motion sensor) |
| POE | Power Over Ethernet |
| REST | Representational State Transfer |
| SBC | Single Board Computer |
| SQL | Structured Query Language |
| TLS | Transport Layer Security |
| UI | User Interface |
| UPS | Uninterruptible Power Supply |
| VFD | Variable Frequency Drive |
| VLAN | Virtual Local Area Network |
| WS | WebSocket |

---

## Related Documents

- [Vision](vision.md) — What Gray Logic is and why
- [Principles](principles.md) — Hard rules and design principles
- [System Overview](../architecture/system-overview.md) — Technical architecture
