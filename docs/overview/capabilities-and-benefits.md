---
title: System Capabilities & Benefits
version: 1.0.0
status: draft
last_updated: 2026-01-18
depends_on:
  - vision.md
  - principles.md
  - ../domains/*.md
---

# Gray Logic System: Capabilities & Benefits

## 1. Executive Summary

Gray Logic is a **future-proof, offline-first building intelligence platform** designed to serve as the "central nervous system" for properties. Unlike proprietary systems (Crestron, Savant) that lock users into specific dealers, or hobbyist platforms (Home Assistant) that lack stability, Gray Logic targets the underserved professional mid-market (£8k–£40k) with an **electrician-first approach**.

The system is built on four non-negotiable pillars:
1.  **Offline-First**: 99%+ of functionality works without an internet connection. The cloud is an optional enhancement, not a dependency.
2.  **No Vendor Lock-In**: Built on open standards (KNX, DALI, Modbus) and open-source software, ensuring owners are never held hostage by a single company or installer.
3.  **Safety-First**: Hardware physical controls always work. Life-safety systems (fire, security) operate independently. Frost protection is enforced at the hardware layer.
4.  **10-Year Horizon**: Designed for a decade of stable operation with version-pinned deployments and proven industrial technologies (Go, SQLite, MQTT).

---

## 2. System Layers & Core Capabilities

The architecture is layered to ensure resilience. Failure in a higher layer never compromises a lower layer.

### Layer 1: Hardware Backbone (The "Spine")
*   **Technology**: KNX, DALI, Modbus.
*   **Capability**: Direct physical control. Wall switches communicate directly with actuators via the bus.
*   **Resilience**: Works even if the Core server, network, and internet are all down. Lights, heating, and blinds remain manually operable.

### Layer 2: Gray Logic Core (The "Brain")
*   **Technology**: Custom Go application, SQLite, MQTT.
*   **Capability**:
    *   **Automation**: Executing scenes, schedules, and logic.
    *   **State Management**: Validating global state using timestamp conflict resolution.
    *   **Context Awareness**: Understanding "Presence", "Modes" (Home, Away, Night), and "Time".
*   **Resilience**: Auto-restarts via systemd. Caches state to survive restarts.

### Layer 3: Feature Modules (The "Senses")
*   **Voice Bridge**: Local, privacy-first voice control (Whisper STT, Piper TTS).
*   **Resilience Engine**: Monitors component health, managing degradation levels (e.g., "Internet Down" vs "Core Down").
*   **PHM (Predictive Health Monitoring)**: Analyzes device metrics (runtime, current, error rates) to predict failures before they occur.

---

## 3. Domain-Specific Capabilities

### 🌡️ Climate Control
*   **Multi-Zone Intelligence**: Independent control of UFH, radiators, and A/C capabilities per room.
*   **Predictive Comfort**: Uses weather forecasts to pre-cool or pre-heat homes (e.g., "It will be hot at 2 PM, start cooling at 11 AM").
*   **Solar Gain Management**: Auto-closes blinds in summer to reduce cooling load; opens them in winter for free heat.
*   **Air Quality**: CO2 and humidity monitoring drives MVHR/ventilation rates automatically (e.g., "Boost ventilation if CO2 > 1000ppm").
*   **Edge Case Handling**:
    *   **Frost Protection**: Hardware-enforced heating activation at 5°C, ensuring pipes never burst even if software fails.
    *   **Window Open**: Instantly cuts heating in a specific zone to save energy.

### 💡 Intelligent Lighting
*   **Circadian Rhythms**: Temperature seamlessly shifts from cool white (morning) to warm glow (evening) to support natural sleep cycles.
*   **Daylight Harvesting**: In commercial/workspace zones, lights auto-dim when natural sunlight is sufficient to maintain lux targets.
*   **Scene Recall**: One touch transforms a room (e.g., "Cinema Mode": lights diff to 5%, blinds close, color temp warms).
*   **Lumen Depreciation Tracking**: PHM tracks LED runtime and alerts facility managers before lights fail or become too dim.

### 🪟 Blinds & Shading
*   **Sun Tracking**: Slats tilt automatically to follow the sun angle, maximizing natural light while blocking direct glare.
*   **Wind & Rain Safety**: Awnings retract immediately upon detecting high wind (>35km/h) or rain, overriding all manual or automated commands.
*   **Privacy Automation**: Bedroom blinds close automatically at sunset or when "Night Mode" is active.
*   **Wake Simulation**: Blinds slowly open over 20 minutes before an alarm for a natural wake-up.

### 🛡️ Security & Access
*   **Panel Integration**: bi-directional sync with professional panels (Texecom, Honeywell). Gray Logic arm state matches the panel.
*   **Voice Security**: Authenticated voice commands (e.g., "Disarm alarm") require a PIN, which is virtually redacted from logs and never stored in plain text.
*   **Hardware Independence**: If Gray Logic fails, the alarm panel, keypads, and sirens continue to function as a certified Grade 2/3 system.
*   **Smart Triggers**: "Away" mode simulates presence by randomizing lights and blinds.

### ⚡ Energy Management
*   **Grid Intelligence**: Smart meter telemetry monitors voltage and frequency for brownout/surge protection.
*   **Load Shedding**: Hierarchical limitation (Comfort > Safety > Cost > Carbon). Automatically sheds "Discretionary" loads (Hot tub, Towel rails) during peak tariff pricing.
*   **Solar/Battery Arbitrage**: Charges batteries when energy is cheap/green; discharges during peak cost windows.
*   **EV Charging**: "Solar-only" charging mode ensures cars run on 100% free sunshine.

### 🗣️ Voice & AI
*   **100% Local**: No audio ever leaves the building. No Amazon/Google listening.
*   **Natural Language**: "It's a bit dark in here" understands intent to turn on lights.
*   **Room Awareness**: "Turn on the lights" knows which room you are in via microphone location.
*   **Intercom & TTS**: Text-to-speech announcements for events (e.g., "Doorbell", "Dinner Ready", "Fire Alarm").

---

## 4. Resilience, Safety & Offline Capabilities

### Truly Offline-First
*   **Zero Cloud Dependency**: Voice, automation, and app control function identically with the internet cable cut.
*   **Timestamp Conflict Resolution**: When connectivity restores, conflicting states are resolved using precise timestamps and "Hardware Authority" rules.
*   **Context-Aware Catch-Up**: If Core restarts (e.g., after a power cut), it intelligently "catches up" miss schedules—but only if safe (e.g., won't turn on lights in an empty house at 3 AM).

### Safety Gates
*   **Hardware Layer**: Direct electrical interlocking.
*   **Software Layer**: "Hard Rules" (e.g., Frost Protection cannot be disabled via UI).
*   **Commissioning Gates**: Installers must physically verify safety interlocks (e.g., E-Stop checks) before the system allows full automation.

---

## 5. Integration & Extensibility

Gray Logic assumes a heterogeneous world ("Honest Disruption"). It doesn't force one brand but bridges best-in-class technologies.

*   **Modular Bridges**: Individual micro-services for KNX, Modbus, DALI, etc. If the "Sonos Bridge" crashes, it does not affect the "Lighting Bridge".
*   **Local API**: Full REST and WebSocket API for 3rd party integration (HomeKit, bespoke dashboards).
*   **Upgrade Path**:
    *   **Starter**: Single-room control (Lighting + Heating).
    *   **Standard**: Full home automation (Energy, Security, Voice).
    *   **Estate**: Multi-building coordination, central plant monitoring, dedicated server specs.

---

## 6. User Benefits

### 🏡 For Homeowners
| Feature | Benefit | Quantifiable Impact |
| :--- | :--- | :--- |
| **Energy Management** | Lower bills without effort | 20–40% reduction in heating/cooling costs |
| **Privacy First** | No creeping surveillance | 0 bytes of voice/video data sent to cloud |
| **10-Year Horizon** | Investment protection | System won't be "obsolete" in 3 years |
| **One App** | Simplified control | Replaces 5-6 fragmented apps (Hue, Ring, Hive, etc.) |
| **Healthy Home** | Better sleep and air | Automation of CO2 levels and Circadian lighting |

### 🛠️ For Installers (Electricians)
| Feature | Benefit | Impact |
| :--- | :--- | :--- |
| **Remote Diagnostics** | Reduced truck rolls | Solve 80% of issues via VPN/SSH without travel |
| **Standardized Config** | Predictable installation | Reduce commissioning time by 50% vs custom coding |
| **Upsell Potential** | Modular capability | Easy to add "Voice" or "Energy" packages later |
| **Documentation** | Transferable maintenance | Any qualified engineer can service the system |

### 🏢 For Businesses / Commercial
| Feature | Benefit | Impact |
| :--- | :--- | :--- |
| **PHM (Predictive Health)** | Prevent downtime | Catch failing pumps/drivers before total failure |
| **Compliance** | Auto-testing | Emergency lighting reports generated automatically |
| **OpEx Reduction** | Automated efficiency | Lighting/HVAC off in empty meeting rooms |
| **Scalability** | Multi-partition security | Granular access control for staff/cleaners |

---

## 7. Potential Limitations & Mitigations

| Limitation | Impact | Mitigation Design |
| :--- | :--- | :--- |
| **Initial Cost** | Higher upfront than DIY (Hue/WiFi) | Positioned as "Infrastructure" (like plumbing) with 10yr+ lifespan to amortize cost. |
| **Installation Skill** | Requires professional electrician | Use "Commissioning Checklists" and strict "Golden Paths" to guide installers. |
| **Hardware Dependency** | Requires specific bus wiring (KNX) | Retrofit wireless bridges (Zigbee/Matter) planned for Year 3, though wired remains preferred. |
| **Complexity** | More complex than a single switch | "Consumer Overlay" UI hides complexity; physical switches always remain as simple fallbacks. |

---

## 8. Conclusion

Gray Logic fundamentally reimagines the "Smart Home" not as a collection of gadgets, but as **infrastructure**. It transforms a property into a responsive, living entity that takes care of its occupants.

By rejecting the "move fast and break things" Silicon Valley model in favor of **industrial stability and openness**, Gray Logic offers the only viable path for properties that need to work reliably for decades. It is the "anti-Alexa": private, robust, and designed to work quietly in the background, ensuring that even when the internet fails, the home keeps running.

This is the central nervous system for the next generation of buildings—built to last.
