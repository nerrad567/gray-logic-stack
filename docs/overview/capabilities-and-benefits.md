# Gray Logic Stack: Capabilities & Benefits Deep Dive

**Status**: Draft
**Date**: January 2026
**Version**: 1.0 (Architecture Phase)

---

## 1. Executive Summary

Gray Logic is not a "smart home" system; it is a **Central Nervous System** for properties. Unlike consumer gadgets (Alexa, Hue) that rely on cloud tethers and proprietary hubs, Gray Logic is an infrastructure-grade platform designed to last as long as the building itself (20+ years).

### Core Philosophy
*   **Offline-First & Sovereign**: 99.9% of functionality works without an internet connection. Your house is not bricked when AWS goes down.
*   **Zero Vendor Lock-in**: Built on open industrial standards (KNX, DALI, Modbus, MQTT). If Gray Logic disappears tomorrow, your physical switches and lights still work.
*   **Safety-First**: Life-safety measures (fire, emergency stops) are hardware-coupled and independent of software state.
*   **10-Year Horizon**: The codebase is designed for stability (Go, SQLite, minimal dependencies) to run untouched for a decade.

This system bridges the gap between massive commercial BMS (Building Management Systems) and modern UX, providing the power of the former with the elegance of the latter.

---

## 2. Core Capabilities

The system architecture consists of three distinct layers working in unison.

### A. Hardware Backbone (The "Body")
*   **Protocol Agnostic**: Connects natively to industrial hardware via standard bridges (KNX, DALI, Modbus).
*   **Bridge Interface**: A standardized MQTT contract ensures any new hardware protocol can be added by writing a simple translator, without touching the Core.
*   **Physical Resilience**: Hardware autonomy is strictly enforced. A "Smart Switch" is just a standard KNX switch that *reports* to the core but *actuates* directly via the bus.

### B. Gray Logic Core (The "Brain")
*   **Single Binary Simplicity**: Written in Go for memory safety and zero-dependency deployment.
*   **State Management**: Uses a "Twin" model—maintaining the desired state and reporting the actual state, continuously reconciling the two.
*   **Automation Engine**:
    *   **Scene Engine**: Orchestrates complex moods (e.g., "Cinema Mode": lights dim, blinds close, thermostat adjusts, phone notifications mute).
    *   **Scene Conflict Resolution**: Implements strict priority rules (Safety > Manual Override > Scene Priority > Last Write) to prevent "ghost" switching.
    *   **Logic Engine**: Supports complex boolean logic, sun position tracking, and weather-dependent triggers.
*   **Local AI Integration**:
    *   **Voice**: Uses **Whisper** (STT) and **Piper** (TTS) running locally. No audio ever leaves the LAN.
    *   **NLU**: Local Large Language Model (Llama-based) for intent understanding, preserving total privacy.

### C. Feature Modules (The "Senses")
*   **Presence Engine**: Fuses data from PIR, mmWave, and WiFi tracking to know exactly *who* is *where*, enabling "follow-me" lighting and climate.
*   **Prognostics (PHM)**: Monitors device health (voltage fluctuations, response latency) to predict failures before they happen (e.g., "Boiler pump showing vibration patterns of imminent failure").

---

## 3. Domain-Specific Features

### 🌡️ Climate
*   **Zoning**: Granular control of every room's micro-climate.
*   **Frost Protection**: **Commissioning Gate**: Hardware-independent frost thermostats are mandatory. Use of software-only frost protection is explicitly banned for safety.
*   **Passive Optimization**: Uses blind positions and sun azimuth to passively heat rooms in winter or cool them in summer before engaging HVAC.

### 💡 Lighting & Blinds
*   **Circadian Rhythm**: Lighting temperature matches the sun's natural cycle (cool blue at noon, warm amber at night) to support biological health.
*   **DALI Integration**: Full bi-directional feedback from drivers (lamp failure detection, exact power usage).
*   **Glare Control**: Blinds automatically adjust angle based on exact sun position geometry to let in light while blocking direct glare.

### ⚡ Energy
*   **Real-Time Cost Integration**: Integrates directly with smart meters (e.g., via P1 port) to read tariffs like **Octopus Agile**.
*   **Cost-Aware Logic**: Automation decisions are financially aware.
    *   *Example*: "It is 4 PM. Energy is expensive (£0.40/kWh). Pre-heat functionality is disabled. Dishwasher start delayed until 2 AM (£0.07/kWh)."
*   **Load Shedding**: Automatic shedding of non-critical loads (pool pumps, EV chargers) if grid limits or cost thresholds are exceeded.

### 🧺 Appliance Orchestration (White Goods)
*   **Local-First, Cloud-Optional**: We prioritize **Matter** and **SG Ready** for 20-year stability but support **Cloud API** integrations (Samsung, LG) for enhanced convenience and monitoring where local options are unavailable.
*   **Spec-Grade Integration**:
    *   *Gold Standard*: Miele (SG Ready), Matter-compliant devices.
    *   *Silver Standard*: Cloud-connected appliances integrated for energy data (aware of API risks).
    *   *Approach*: We provide a "Certified Hardware List" (e.g., Miele with SG Ready, Matter-compliant devices) to ensure 20-year compatibility.

### 🛡️ Security
*   **Privacy-First Video**: NVR runs locally. Object detection (person, car) runs on local hardware. No video streams to cloud servers.
*   **Voice PIN Security**: Sensitive actions (disarming alarm) via voice require a spoken PIN. This PIN is **never verified via cloud AI**, never logged in transcripts, and the audio buffer is wiped immediately after local verification.
*   **Plaintext Mitigation**: PINs are hashed before transmission over MQTT/Internal bus.

### 🏥 Health (PHM)
*   **Device Health Tiers**: Not just "Online/Offline".
    *   **Healthy**: Normal operation.
    *   **Degraded**: Functional but showing warning signs (e.g., 5% packet loss, 200ms latency spike).
    *   **Failed**: Non-functional.
    *   **Maintenance Required**: Predicted failure window approached.

---

## 4. Resilience & Safety Features

### Offline Capability
*   **The "Internet Cable Test"**: The system passes the commissioning test only if *all* functionality (voice, scenes, automation) works with the WAN cable unplugged.
*   **NTP & Time Trust**: Bridges must reference a reliable local NTP source (Core acts as Stratum 2 server). Timestamps from untrusted bridges >60s skewed are rejected to prevent state poisoning.

### Failure Modes & Graceful Degradation
*   **MQTT Failure**: If the message bus dies, physical wall switches *still control lights*. The "Brain" is severed, but the "Reflexes" (hardware links) remain intact.
*   **Core Failure**: Bridges enter "Autonomous Mode", reverting to simple local logic until the Core recovers.

### Data Safety
*   **Additive-Only Migrations**: Database schema changes never destroy data; they only append.
*   **Backup-Before-Write**: A verified cryptographic snapshot of the database is taken automatically before any migration is attempted.

---

## 5. User Benefits

### For Homeowners
| Benefit | Description | Real-World Impact |
| :--- | :--- | :--- |
| **Data Privacy** | Zero data mining. Your conversations and video stay in your house. | "Hey Gray" puts *you* in control, not a tech giant's ad algorithm. |
| **Energy Savings** | 20–40% reduction via intelligent HVAC and load shifting. | Automated blind control + Octopus Agile integration can save £500+/year. |
| **Reliability** | Industrial hardware lasts 20 years, not 2. | No "server down" messages when trying to turn on the kitchen light. |
| **Future Proofing** | Open standards mean you can mix brands (ABB, Schneider, MDT). | You are not stuck buying $80 bulbs from a single bankrupt startup. |

### For Installers & Integrators
*   **Revenue Model**: Shift from "install and leave" to Service Level Agreements (SLA) for remote monitoring and PHM alerts.
*   **Debugging**: Detailed structure logs and health metrics make diagnosing "why did the light turn on?" trivial compared to black-box hubs.
*   **Training**: Standardized checklists and "Commissioning Gates" ensure junior techs can deploy quality systems.

### For Developers (Commercial/Estates)
*   **Compliance**: Automated emergency light testing and reporting.
*   **Scalability**: The same code runs a 1-bed flat or a 50-room estate; just add more bridges.

---

## 6. Integration & Extensibility

### "Lego for Professionals"
The system is designed to be modular.
*   **Start Small**: Deploy a Core + 1 Lighting Bridge.
*   **Expand Later**: Add Climate, Security, and Voice bridges years later. The Core automatically discovers new capabilities.

### The Truth About Retrofits
*   **Honesty**: Gray Logic is optimized for *new builds* or *deep renovations*. It relies on wired buses (KNX/Cat6).
*   **Wireless Support**: While possible via gateways (Zigbee/Matter), wireless is treated as a second-class citizen (lower reliability tier) compared to wire.

---

## 7. Potential Drawbacks & Mitigations

| Drawback | Reality | Mitigation |
| :--- | :--- | :--- |
| **Initial Cost** | 2x-3x higher than generic "smart bulbs". | Offset by 20-year lifespan and property value increase. |
| **Complexity** | Requires qualified electricians/integrators. | Not a DIY weekend project. Ensures safety and code compliance. |
| **Hardware Space** | Requires a dedicated plant room/rack. | Necessary for decentralized, reliable hardware. |

---

## 8. Conclusion

The Gray Logic Stack represents a fundamental shift in how we view building intelligence. It rejects the disposable, cloud-dependent "gadget" culture in favor of **permanence, privacy, and physics**.

By treating the home as a mission-critical environment—akin to a hospital or data center—Gray Logic transforms a property from a collection of dumb bricks into a **sentient, autonomous asset** that takes care of its occupants, protects its data, and optimizes its own consumption for decades to come.

This is not just a smart home. It is a **Future Home**.
