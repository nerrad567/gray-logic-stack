---
title: System Overview
version: 1.0.0
status: active
implementation_status: specified
last_updated: 2026-01-17
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
| **Web Admin** | Svelte | Commissioning, configuration, diagnostics |
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
| **DALI Bridge** | DALI-2 | Via any DALI gateway (protocol-agnostic) |
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

| Spec | Minimum (No AI) | Recommended (With AI) | High Performance |
|------|-----------------|-----------------------|------------------|
| **CPU** | x86_64 or ARM64, 4 cores | Intel i5/i7 (12th+ gen) or AMD Ryzen 5000+ | Intel i7/i9 or AMD Ryzen 7/9 |
| **RAM** | 4GB | 16GB | 32GB+ |
| **Storage** | 64GB SSD | 512GB NVMe | 1TB NVMe |
| **Network** | 1GbE | 2.5GbE | Dual 2.5GbE / 10GbE |
| **AI Accelerator** | N/A | Google Coral TPU or NPU | NVIDIA Jetson / Discrete GPU |
| **Form Factor** | Raspberry Pi 4 / CM4 | Intel NUC / Mini PC | Small Form Factor Server |

**Key Considerations:**
- **AI Workloads:** Local LLM and Whisper STT require significant compute. Dedicated accelerators (Coral, NPU) are strongly recommended to offload CPU.
- **Reliability:** Industrial fanless PCs are preferred for the "Set and Forget" 10-year goal.
- **Storage:** High-quality NVMe drives are essential for database reliability and speed.

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

## AI & Resource Isolation

To ensure the "Hard Rules" (particularly system stability) are never compromised by AI workloads:

### 1. Hardware Acceleration Strategy
- **Primary:** Leverage dedicated NPUs (Neural Processing Units) or TPUs (Tensor Processing Units) where available (e.g., Apple Silicon NPU, Intel NPU, Google Coral).
- **Secondary:** Use discrete or integrated GPUs (NVIDIA CUDA, Intel Arc, AMD ROCm).
- **Fallback:** CPU inference (only if robust core isolation is possible).

### 2. Process Isolation (cgroups)
The system uses strict resource limits to prioritize Automation over Intelligence.

| Layer | Priority (Nice) | CPU Limit | Memory Limit | OOM Score |
|-------|-----------------|-----------|--------------|-----------|
| **Core (Automation)** | -10 (High) | Unrestricted | Unrestricted | -1000 (Never Kill) |
| **Bridges (KNX/DALI)** | -5 (High) | Unrestricted | 512MB | -500 |
| **Database (SQLite)** | -5 (High) | Unrestricted | Unrestricted | -900 |
| **Voice/AI Engine** | +10 (Low) | Max 60% | Max 8GB | +1000 (Kill First) |
| **Non-Critical UI** | 0 (Normal) | Max 20% | 1GB | 0 |

**Result:** If the AI model spikes CPU usage or leaks memory, the OS throttles or kills the AI process *only*. The light switches (Core + Bridge) continue to function without jitter.

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
| **Web Admin** | Svelte | Simpler, smaller bundle, maintainable |
| **Voice STT** | Whisper | Local, accurate, open |
| **Voice TTS** | Piper | Local, natural sounding |
| **Local LLM** | Llama/Phi | On-device intelligence |

---

## Multi-Site Architecture

### Current Scope: Single Site

Gray Logic v1.0 is designed for **single-site deployments**:

- **One Core instance per site**
- Sites operate completely independently
- Own database, configuration, user accounts, backup
- No inter-site dependencies

**Rationale:**
- Simplifies offline operation
- Clear security boundary
- Easier to troubleshoot

### Multi-Property Owners

For customers with multiple properties, each site has its own Gray Logic installation:

```
┌──────────────────┐    ┌──────────────────┐
│   Main House     │    │   Holiday Home   │
│  ┌────────────┐  │    │  ┌────────────┐  │
│  │ Gray Logic │  │    │  │ Gray Logic │  │
│  └────────────┘  │    │  └────────────┘  │
└────────┬─────────┘    └────────┬─────────┘
         │                       │
         └───────────┬───────────┘
                     │
             ┌───────▼───────┐
             │  WireGuard    │
             │     VPN       │
             └───────┬───────┘
                     │
             ┌───────▼───────┐
             │ Customer's    │
             │  Phone App    │
             └───────────────┘
```

The mobile app can connect to multiple sites via separate VPN configurations.

### Future: Centralized Management (Year 5+)

Optional lightweight aggregation for multi-site owners:
- View-only dashboard across sites
- Alert aggregation
- Single login to multiple sites

Sites remain fully independent — no cross-site automation or shared state.

---

## Capacity Planning

### Server Sizing

| Scale | Devices | CPU | RAM | Storage | Example Hardware |
|-------|---------|-----|-----|---------|-----------------|
| **Minimum** | <100 | 2 cores | 2 GB | 32 GB SSD | Raspberry Pi 4 |
| **Recommended** | 100-300 | 4 cores | 4 GB | 64 GB SSD | Intel NUC, Mini PC |
| **Large** | 300-1000 | 8 cores | 8 GB | 128 GB SSD | Small server |

**Voice processing adds:** +2 GB RAM, +2 cores (or GPU recommended)

### Resource Estimates

| Resource | Estimate |
|----------|----------|
| RAM per device | ~10 KB |
| InfluxDB per device per day | ~0.5 MB |
| Per 100 devices per year (InfluxDB) | ~18 GB |
| SQLite per device per year | ~1 KB |

**Example: 200 devices**
- Core RAM: ~32 MB
- Storage per year: ~40 GB (mostly InfluxDB)

### Scaling Limits

| Metric | Target | Tested Max | Bottleneck |
|--------|--------|------------|------------|
| Devices | 500 | 1000 | MQTT message rate |
| Concurrent UI clients | 10 | 20 | WebSocket connections |
| Commands/second | 100 | 200 | Bridge throughput |

For deployments exceeding 500 devices, consider splitting into zones.

---

## Related Documents

- [Vision](../overview/vision.md) — What Gray Logic is and why
- [Principles](../overview/principles.md) — Hard rules and design principles
- [Entities](../data-model/entities.md) — Data model definitions
- [API Specification](../interfaces/api.md) — REST API details
- [Development Strategy](../development/DEVELOPMENT-STRATEGY.md) — Build approach and milestones
- [Security Checklist](../development/SECURITY-CHECKLIST.md) — Security verification gates
- [Monitoring](../operations/monitoring.md) — System monitoring and alerting
- [Updates](../operations/updates.md) — Upgrade strategy
- Protocol bridge specifications in [protocols/](../protocols/)
- Integration specifications in [integration/](../integration/) — Access Control, CCTV, Fire Alarm
