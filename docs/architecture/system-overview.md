---
title: System Overview
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - overview/vision.md
  - overview/principles.md
---

# System Overview

This document describes the high-level architecture of Gray Logic — the components, how they interact, and how data flows through the system.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              USER INTERFACES                                     │
├───────────────┬───────────────┬───────────────┬───────────────┬─────────────────┤
│  Wall Panels  │  Mobile App   │  Voice Input  │  Web Admin    │  Remote Access  │
│  (per room)   │  (iOS/Android)│  (room mics)  │  (browser)    │  (VPN only)     │
└───────┬───────┴───────┬───────┴───────┬───────┴───────┬───────┴────────┬────────┘
        │               │               │               │                │
        └───────────────┴───────────────┴───────┬───────┴────────────────┘
                                                │
                                    REST API + WebSocket
                                                │
┌───────────────────────────────────────────────▼──────────────────────────────────┐
│                              GRAY LOGIC CORE                                      │
│                                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                         INTELLIGENCE LAYER                                   │ │
│  │  ┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐  │ │
│  │  │   AI Engine     │  Voice / NLU    │    Presence     │   Learning &    │  │ │
│  │  │   (Local LLM)   │   Processing    │   Detection     │   Prediction    │  │ │
│  │  └─────────────────┴─────────────────┴─────────────────┴─────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                         AUTOMATION LAYER                                     │ │
│  │  ┌───────────┬───────────┬───────────┬───────────┬───────────────────────┐  │ │
│  │  │  Scene    │ Scheduler │   Mode    │   Event   │  Conditional Logic    │  │ │
│  │  │  Engine   │           │  Manager  │   Router  │       Engine          │  │ │
│  │  └───────────┴───────────┴───────────┴───────────┴───────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                           DEVICE LAYER                                       │ │
│  │  ┌───────────┬───────────┬───────────┬───────────┬───────────────────────┐  │ │
│  │  │  Device   │   State   │  Command  │ Discovery │   Health Monitor      │  │ │
│  │  │ Registry  │  Manager  │ Processor │  Service  │       (PHM)           │  │ │
│  │  └───────────┴───────────┴───────────┴───────────┴───────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                       INFRASTRUCTURE LAYER                                   │ │
│  │  ┌───────────┬───────────┬───────────┬───────────┬───────────────────────┐  │ │
│  │  │    API    │ WebSocket │  Database │  Message  │   Security & Auth     │  │ │
│  │  │   Server  │   Server  │ (SQLite)  │ Bus(MQTT) │                       │  │ │
│  │  └───────────┴───────────┴───────────┴───────────┴───────────────────────┘  │ │
│  └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│                              Single Go Binary (~30MB RAM)                         │
└───────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                 Internal MQTT Bus
                                        │
┌───────────────────────────────────────▼──────────────────────────────────────────┐
│                            PROTOCOL BRIDGES                                       │
├─────────┬─────────┬─────────┬─────────┬─────────┬─────────┬─────────┬────────────┤
│   KNX   │  DALI   │ Modbus  │  Audio  │  Video  │Security │  CCTV   │   Voice    │
│ Bridge  │ Bridge  │ Bridge  │ Matrix  │ Matrix  │  Panel  │  /NVR   │   Input    │
└────┬────┴────┬────┴────┬────┴────┬────┴────┬────┴────┬────┴────┬────┴─────┬─────┘
     │         │         │         │         │         │         │          │
     ▼         ▼         ▼         ▼         ▼         ▼         ▼          ▼
  KNX Bus   DALI Bus  Modbus   Audio     HDMI     Texecom   Uniview    Room
  (TP/IP)  (Gateway)  TCP/RTU  Matrix   Matrix   /Galaxy     NVR       Mics
```

---

## Component Descriptions

### User Interfaces

All user interfaces communicate with Gray Logic Core via REST API and WebSocket connections. They are thin clients — all logic lives in the Core.

| Interface | Technology | Purpose |
|-----------|------------|---------|
| **Wall Panels** | Flutter (Android/Linux) | Per-room control, always on, mounted |
| **Mobile App** | Flutter (iOS/Android) | Away-from-home control, notifications |
| **Voice Input** | Room microphone arrays | Hands-free control via local AI |
| **Web Admin** | React/Svelte | Commissioning, configuration, diagnostics |
| **Remote Access** | Via WireGuard VPN | Identical to local access, encrypted |

### Gray Logic Core

The Core is a single Go binary containing all automation logic. It runs on the on-site server and manages everything.

#### Infrastructure Layer

| Component | Responsibility |
|-----------|----------------|
| **API Server** | REST endpoints for UI operations |
| **WebSocket Server** | Real-time state updates to UIs |
| **Database (SQLite)** | Device registry, configuration, state persistence |
| **Message Bus (MQTT)** | Internal communication with bridges |
| **Security & Auth** | Authentication, authorization, session management |

#### Device Layer

| Component | Responsibility |
|-----------|----------------|
| **Device Registry** | Catalog of all devices, their capabilities, addresses |
| **State Manager** | Current state of all devices, persistence, history |
| **Command Processor** | Validates and routes commands to bridges |
| **Discovery Service** | Finds new devices on supported protocols |
| **Health Monitor (PHM)** | Predictive health monitoring, baseline learning |

#### Automation Layer

| Component | Responsibility |
|-----------|----------------|
| **Scene Engine** | Executes scenes (multi-device, timed, conditional) |
| **Scheduler** | Time-based triggers (clock, astronomical, calendar) |
| **Mode Manager** | System modes (Home/Away/Night/Holiday) |
| **Event Router** | Routes device events to appropriate handlers |
| **Conditional Logic Engine** | Evaluates complex conditions for automation |

#### Intelligence Layer

| Component | Responsibility |
|-----------|----------------|
| **AI Engine** | Local LLM for complex queries and pattern detection |
| **Voice/NLU Processing** | Speech-to-text, intent extraction, text-to-speech |
| **Presence Detection** | Occupancy tracking, arrival/departure detection |
| **Learning & Prediction** | Pattern learning, behaviour prediction, PHM AI |

### Protocol Bridges

Bridges are separate processes that handle protocol-specific communication. They connect to the Core via MQTT.

| Bridge | Protocol | Hardware |
|--------|----------|----------|
| **KNX Bridge** | KNX TP / KNX IP | Via knxd daemon |
| **DALI Bridge** | DALI-2 | Via gateway (e.g., Tridonic) |
| **Modbus Bridge** | Modbus RTU/TCP | Direct or via converter |
| **Audio Matrix** | RS232/IP | HTD, Russound, AudioControl |
| **Video Matrix** | RS232/IP | Atlona, WyreStorm, etc. |
| **Security Panel** | Varies | Texecom, Galaxy, Pyronix |
| **CCTV/NVR** | RTSP/ONVIF | Uniview, Hikvision, etc. |
| **Voice Input** | Audio stream | Room microphone arrays |

---

## Data Flow

### Command Flow (User → Device)

```
User taps "Cinema Mode" on wall panel
            │
            ▼
    Wall Panel sends POST /api/scenes/cinema/activate
            │
            ▼
    API Server validates request, checks authorization
            │
            ▼
    Scene Engine loads scene definition
            │
            ▼
    Scene Engine publishes commands to MQTT:
        - graylogic/command/knx/1.2.3 → {"action": "dim", "level": 5}
        - graylogic/command/knx/1.2.4 → {"action": "off"}
        - graylogic/command/audio/zone3 → {"action": "source", "input": 2}
            │
            ▼
    Bridges receive commands, translate to protocol:
        - KNX Bridge → KNX telegram
        - Audio Bridge → RS232 command
            │
            ▼
    Physical devices respond
            │
            ▼
    Devices report new state (KNX status telegram, etc.)
            │
            ▼
    Bridges publish state to MQTT:
        - graylogic/state/knx/1.2.3 → {"level": 5}
            │
            ▼
    State Manager updates database
            │
            ▼
    WebSocket Server broadcasts state change to all UIs
            │
            ▼
    Wall panels, mobile apps update display
```

### Event Flow (Device → Automation)

```
Motion sensor detects presence
            │
            ▼
    KNX actuator sends status telegram
            │
            ▼
    KNX Bridge publishes to MQTT:
        - graylogic/state/knx/1.3.1 → {"motion": true}
            │
            ▼
    State Manager updates device state
            │
            ▼
    Event Router receives state change event
            │
            ▼
    Event Router checks for matching triggers:
        - Scene triggers
        - Automation rules
        - PHM inputs
            │
            ▼
    Matching automation found:
        "If motion in hallway AND mode=Night AND time=22:00-06:00
         THEN activate Pathway Lights scene"
            │
            ▼
    Conditional Logic Engine evaluates conditions
            │
            ▼
    Conditions met → Scene Engine activates scene
```

### Voice Flow (Speech → Action)

```
User says "Hey Gray, turn on the kitchen lights"
            │
            ▼
    Room microphone detects wake word locally
            │
            ▼
    Audio stream sent to Voice Bridge
            │
            ▼
    Voice Bridge runs Whisper (local speech-to-text):
        "turn on the kitchen lights"
            │
            ▼
    NLU Engine extracts intent:
        {
            "intent": "control_device",
            "domain": "lighting",
            "room": "kitchen",
            "action": "on"
        }
            │
            ▼
    Command Processor resolves "kitchen lights":
        - Device Registry lookup → [device_id_1, device_id_2]
            │
            ▼
    Commands sent via normal command flow
            │
            ▼
    TTS Engine generates response:
        "Kitchen lights are now on"
            │
            ▼
    Audio played through room speaker
```

---

## Network Architecture

### Physical Network

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ON-SITE NETWORK                                 │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ CONTROL VLAN (192.168.1.0/24)                                        │    │
│  │   • Gray Logic Server                                                │    │
│  │   • KNX/IP Interface                                                 │    │
│  │   • DALI Gateways                                                    │    │
│  │   • Wall Panels                                                      │    │
│  │   • Audio/Video Matrix                                               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ CCTV VLAN (192.168.2.0/24)                                           │    │
│  │   • NVR                                                              │    │
│  │   • IP Cameras                                                       │    │
│  │   • Door Stations (video)                                            │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ USER VLAN (192.168.10.0/24)                                          │    │
│  │   • Resident devices (phones, laptops)                               │    │
│  │   • Guest devices                                                    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│                              ┌─────────────┐                                │
│                              │   Router    │                                │
│                              │  Firewall   │                                │
│                              └──────┬──────┘                                │
│                                     │                                       │
└─────────────────────────────────────┼───────────────────────────────────────┘
                                      │
                                 Internet
                                      │
                              ┌───────▼───────┐
                              │  VPS (Remote  │
                              │   Optional)   │
                              └───────────────┘
```

### Firewall Rules (Summary)

| From | To | Allowed |
|------|------|---------|
| Control VLAN | Internet | Blocked (except NTP, optional weather API) |
| CCTV VLAN | Internet | Blocked |
| User VLAN | Control VLAN | Port 443 only (UI access) |
| User VLAN | CCTV VLAN | Blocked (access via Gray Logic) |
| Gray Logic Server | CCTV VLAN | RTSP, ONVIF ports |
| WireGuard VPN | Control VLAN | Full access (authenticated users) |

---

## Hardware Requirements

### Gray Logic Server

| Spec | Minimum | Recommended |
|------|---------|-------------|
| **CPU** | x86_64 or ARM64, 4 cores | Intel i5/i7 or AMD Ryzen |
| **RAM** | 4GB | 8GB |
| **Storage** | 64GB SSD | 256GB NVMe |
| **Network** | 1GbE | 2.5GbE or dual 1GbE |
| **Form Factor** | NUC-style or 1U rack | Industrial fanless preferred |

**Example Hardware:**
- Intel NUC (various generations)
- Lenovo ThinkCentre Tiny
- HP ProDesk Mini
- Protectli Vault (fanless)
- Custom industrial PC

### Wall Panels

| Spec | Requirement |
|------|-------------|
| **Display** | 7-10" touchscreen, IPS |
| **Resolution** | 1280x800 minimum |
| **Connectivity** | Ethernet (PoE preferred) or WiFi |
| **Mounting** | Flush or surface mount options |
| **Power** | PoE or 12/24V DC |

**Example Hardware:**
- Industrial Android panels (Akuvox, 2N)
- Raspberry Pi CM4 + display
- Custom Android tablets in enclosures

### Voice Hardware

| Component | Purpose |
|-----------|---------|
| **Microphone Array** | Far-field audio capture (ReSpeaker, etc.) |
| **Speaker** | Response playback (can use audio system) |
| **Processing** | On Gray Logic Server or dedicated device |

---

## Technology Stack Summary

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Core** | Go | Single binary, no runtime, cross-compiles, 10-year stability |
| **Database** | SQLite | Embedded, zero maintenance, reliable |
| **Time-Series** | InfluxDB | PHM data, energy monitoring |
| **Message Bus** | MQTT (Mosquitto) | Simple, proven, debuggable |
| **API** | REST + WebSocket | Universal client support |
| **Wall Panel UI** | Flutter | Cross-platform, native performance |
| **Mobile App** | Flutter | Same codebase as panels |
| **Web Admin** | Svelte/React | Modern, maintainable |
| **Voice STT** | Whisper | Local, accurate, open |
| **Voice TTS** | Piper | Local, natural sounding |
| **Local LLM** | Llama/Phi | On-device intelligence |

---

## Related Documents

- [Vision](../overview/vision.md) — What Gray Logic is and why
- [Principles](../overview/principles.md) — Hard rules and design principles
- [Entities](../data-model/entities.md) — Data model definitions
- [API Specification](../interfaces/api-rest.md) — REST API details
- [Development Strategy](../development/DEVELOPMENT-STRATEGY.md) — Overall development approach and phased plan
- [Coding Standards](../development/CODING-STANDARDS.md) — How to write and document code
- [Security Checklist](../development/SECURITY-CHECKLIST.md) — Security verification and release gates
- Protocol bridge specifications in [protocols/](../protocols/)
- Integration specifications in [integration/](../integration/) — Access Control, CCTV, Fire Alarm
