# Changelog – Gray Logic

All notable changes to this project will be documented in this file.

---

## 1.0.2 – AI Assistant Context (2026-01-17)

**Tooling & Guidance**

Added specialized guidance for AI assistants to ensure alignment with project principles and coding standards.

**New Documents**

- `GEMINI.md`: Project-specific guidance for the Gemini CLI agent, including architecture overview, philosophy, coding standards, and interaction rules.

---

## 1.0.1 – Plant Room & Commercial Expansion (2026-01-12)

**Documentation Expansion**

Expanded Gray Logic to properly support plant room environments and commercial/office deployments.

**Updated Documents**

- `docs/data-model/entities.md`:
  - Added Area types: `wing`, `zone` for commercial deployments
  - Added hierarchy guidance table (Residential vs Office vs Multi-tenant)
  - Added Room types: `open_plan`, `meeting_room`, `boardroom`, `reception`, `break_room`, `hot_desk`, `server_room`, `storage`, `loading_bay`, `washroom`, `corridor`, `workshop`, `changing_room`, `wine_cellar`, `spa`
  - Expanded Plant Equipment device types: `chiller`, `cooling_tower`, `ahu`, `fcu`, `vav_box`, `vfd`, `fan`, `compressor`, `humidifier`, `dehumidifier`, `water_heater`, `water_softener`, `generator`, `ups`
  - Added Plant Sensors & Actuators: `flow_meter`, `pressure_sensor`, `differential_pressure`, `valve_2way`, `valve_3way`, `damper`, `vibration_sensor`, `bearing_temp`
  - Added Emergency & Safety (monitoring only): `emergency_light`, `exit_sign`, `fire_input`, `gas_detector`
  - Added Access Control: `card_reader`, `door_controller`, `turnstile`, `intercom`
  - Added Protocols: `bacnet_ip`, `bacnet_mstp`, `onvif`, `ocpp`
  - Added Domain: `safety`
  - Expanded Capabilities with categories: Basic Control, Climate & Environment, Presence & Occupancy, Audio/Video, Security & Access, Plant & Equipment, Energy Monitoring, Condition Monitoring (PHM), Emergency Lighting, Booking & Scheduling

- `docs/domains/climate.md`:
  - Added Commercial HVAC section
  - VAV zone control with state model
  - Fan coil unit zones
  - Occupancy scheduling with pre-conditioning and optimum start
  - Meeting room booking integration
  - Out-of-hours override requests
  - Night setback and purge sequences
  - Commercial commissioning checklist

- `docs/domains/lighting.md`:
  - Added Commercial Lighting section
  - Emergency lighting monitoring (DALI Part 202) with state model
  - Function and duration test scheduling
  - Compliance reporting for emergency lighting
  - Daylight harvesting for commercial spaces
  - Occupancy-based lighting (commercial patterns)
  - Corridor and circulation lighting
  - Meeting room lighting with AV integration
  - Personal control and task lighting
  - Commercial commissioning checklist

**New Documents**

- `docs/protocols/bacnet.md` — Year 2 roadmap placeholder for BACnet/IP and MS/TP integration with commercial HVAC systems (AHUs, chillers, VAVs)

- `docs/domains/plant.md` — Comprehensive plant domain specification covering:
  - Pumps, VFDs, AHUs, chillers, boilers, heat pumps
  - State models for all equipment types
  - Commands and sequences of operation
  - Lead/lag, economizer, staging sequences
  - Alarm management (priorities, state machine, shelving)
  - Predictive Health Monitoring (PHM) for rotating equipment
  - Energy optimization strategies
  - Modbus and BACnet protocol mapping

- `docs/integration/fire-alarm.md` — Fire alarm system integration (monitoring only):
  - Critical safety rules (observe, never control)
  - Integration architecture via auxiliary contacts
  - Signal types and wiring
  - Automation responses (lights, blinds, notifications)
  - UI display and wall panel behavior
  - Audit logging and compliance reporting
  - Testing and commissioning procedures

- `docs/integration/access-control.md` — Access control integration:
  - Integration levels (monitoring, triggering, control)
  - Door controllers, card readers, intercoms, gates
  - State models and access events
  - **Residential Access Control** (new):
    - Video intercom integration (2N, Doorbird, Akuvox)
    - Keypad entry with scheduled cleaner/tradesperson access
    - Smart lock integration (auto-lock, battery monitoring)
    - Garage door control with safety sensors and auto-close
    - Driveway gate automation with ANPR option
    - Guest/temporary access code management
    - Holiday mode access restrictions
    - Pool gate safety compliance (monitoring only)
  - Welcome home and visitor automation
  - Departure/arrival presence detection
  - Emergency egress coordination
  - Remote unlock security requirements
  - SIP intercom integration
  - **Commercial Access Control**:
    - Turnstiles and speed gates
    - Visitor management integration

- `docs/integration/cctv.md` — CCTV and video surveillance integration:
  - **Residential CCTV**:
    - Typical camera placement (front door, driveway, garden, garage)
    - Doorbell/intercom camera integration with wall panels
    - Motion detection automation triggers
    - Package delivery detection
    - Vehicle approaching notifications
    - Privacy controls per camera
  - **Commercial CCTV**:
    - Camera groups and multi-site layouts
    - Analytics integration (line crossing, people counting, ANPR)
    - Access control linking (record on denied access)
    - Video wall configuration
  - Stream management (main/sub stream selection)
  - MQTT event topics and payloads
  - Privacy zones and audit logging
  - Supported hardware (Uniview, Hikvision, Dahua, Axis)

- `docs/deployment/office-commercial.md` — Commercial deployment guide:
  - Commercial vs residential comparison
  - Network architecture and IT integration
  - Occupancy schedules and holiday calendars
  - Open plan and meeting room lighting
  - VAV system configuration
  - Meeting room booking (M365, Google Workspace)
  - Tenant energy billing
  - Active Directory integration
  - Role-based permissions
  - Commissioning checklist
  - Ongoing operations procedures

---

## 1.0.0 – Architecture Pivot to Custom Gray Logic Core (2026-01-12)

**Major Architectural Change**

This release marks a significant pivot from the openHAB-based approach (v0.4 and earlier) to a **custom-built Gray Logic Core** in Go. This decision was made after careful analysis of the project goals:

- **Multi-decade deployment stability** — Control over the entire stack, no dependency on third-party project decisions
- **True offline operation** — Leaner runtime, faster startup, lower resource usage
- **Custom UI** — Wall panels and mobile apps built to our specifications, not constrained by openHAB UI
- **Native AI integration** — Local voice control without Alexa/Google dependency
- **Full feature parity with Crestron/Savant/Loxone** — Scene complexity, multi-room audio, video distribution

**What Changed**

- **Archived** all v0.4 documentation to `docs/archive/v0.4-openhab-era.zip`
- **Archived** Docker Compose stack to `code/archive/v0.4-openhab-era.zip`
- **Created** new modular documentation structure:
  - `docs/overview/` — Vision, principles, glossary
  - `docs/architecture/` — System design
  - `docs/data-model/` — Entity definitions and schemas
  - `docs/domains/` — Per-domain specifications (to be created)
  - `docs/automation/` — Scenes, schedules, modes (to be created)
  - `docs/intelligence/` — Voice, PHM, AI (to be created)
  - `docs/resilience/` — Offline operation, satellite weather (to be created)
  - `docs/protocols/` — Protocol bridges (to be created)
  - `docs/interfaces/` — API and UI specs (to be created)
  - `docs/deployment/` — Installation and commissioning (to be created)
  - `docs/business/` — Business case and pricing (to be created)

**New Foundation Documents**

- `docs/overview/vision.md` — What Gray Logic is and why we're building it
- `docs/overview/principles.md` — Hard rules that can never be broken
- `docs/overview/glossary.md` — Standard terminology
- `docs/architecture/system-overview.md` — Complete system architecture
- `docs/data-model/entities.md` — Core data model (Site, Area, Room, Device, Scene, etc.)

**Technology Decisions**

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Core | Go | Single binary, no runtime, cross-compiles, multi-decade stability |
| Database | SQLite | Embedded, zero maintenance |
| Time-Series | InfluxDB | PHM data, energy monitoring |
| Message Bus | MQTT | Simple, proven, debuggable |
| Wall Panel/Mobile | Flutter | Cross-platform native |
| Voice STT | Whisper | Local, accurate, open |
| Voice TTS | Piper | Local, natural |
| Local AI | Llama/Phi | On-device intelligence |

**Principles Carried Forward**

All core principles from v0.4 remain valid:
- Offline-first (99%+ without internet)
- Physical controls always work
- Life safety independent
- Open standards at field layer (KNX, DALI, Modbus)
- No vendor lock-in
- Customer owns their system

**What's Next**

This is a 5-year part-time project:
- Year 1: Core + KNX + lighting in own home
- Year 2: Full scenes, modes, blinds, climate
- Year 3: Audio, video, security, CCTV
- Year 4: Voice control, PHM, AI
- Year 5: Commissioning tools, first customer

---

## 0.4 – Predictive Health, Doomsday Pack & Golden Handcuffs (2025-12-02)

> **Note**: This version used openHAB + Node-RED. Documentation archived to `docs/archive/v0.4-openhab-era.zip`

**Docs**

- Updated `docs/gray-logic-stack.md` to **Working Draft v0.4**:

  - Refined the **Design Principles** and module structure to explicitly support:
    - **Predictive Health Monitoring (PHM)** for suitable plant (pumps, boilers/heat pumps, AHUs, pool kit).
    - A clearer split between:
      - Short/medium-term **local logging** on-site.
      - Long-term **trend retention** as a remote premium bonus.
  - Added a dedicated **Plant & PHM** section (within modules and principles) covering:
    - The idea of a plant “**heartbeat**” (current, temp, run hours, vibration, etc.).
    - Using rolling averages and deviation thresholds as **early warning**, not magic:
      - e.g. “If pump current or temp deviates from its 7-day average by ≥20% for >2h, raise a ‘maintenance warning’, not a full fault.”
    - Asset categories where PHM makes sense (pumps, AHUs, boilers/heat pumps, etc.).
    - Examples of how PHM logic is split between:
      - openHAB (device states, items, basic rules).
      - Node-RED (cross-system logic and PHM flows).
  - Tightened the **Security & CCTV** section to:
    - Reiterate that cloud-only CCTV/doorbells (Ring/Nest-style) are **out of scope** for core logic.
    - Explicitly list examples of **integrator-grade** doorbells (Amcrest, DoorBird, Uniview) as _illustrative_ of the required capabilities (local RTSP/ONVIF, predictable behaviour).
  - Reinforced the **Consumer Overlay** rules:
    - Overlay devices remain non-critical and best-effort.
    - Overlay logic cannot become a dependency for plant, core lighting safety, or security.
  - Clarified that the stack is designed to avoid **“Golden Handcuffs”**:
    - Open standards at the field layer.
    - Documented configs and runbooks.
    - A clear handover path if someone else needs to take over.

- Updated `docs/business-case.md` to **Business Case v0.4**:

  - Extended the **Problem Statement** to include:
    - Demand for **BMS-like early warning** on small sites without full BMS cost.
    - The “Golden Handcuffs” problem of proprietary platforms (high cost to leave an ecosystem).
  - Strengthened the **Solution Overview**:
    - Added Predictive Health Monitoring (PHM) as a headline capability:
      - The system “learns” plant behaviour and flags deviations as early warnings.
    - Made the **internal vs external** split more explicit around data:
      - On-site: short/medium history and PHM rules that still work offline.
      - Remote: long-term retention, pretty dashboards, multi-year comparisons.
  - Reframed the **Value Proposition**:
    - For clients: emphasised **early warning** and evidence-based maintenance,
      not promises of zero failures.
    - For the business: PHM is a signed, billable differentiator that justifies Enhanced/Premium tiers.
  - Reworked **Support Tiers** to map directly to PHM capability:

    - **Core Support**:

      - Safe, documented system with basic binary monitoring.
      - Offline-first, no VPS required.

    - **Enhanced Support**:

      - Adds PHM alerts using trend deviation logic.
      - Short/medium-term trend views per site.
      - Clear “we spot issues earlier” story.

    - **Premium Support**:
      - Adds deeper VFD/Modbus data, multi-year history and reporting.
      - Multi-site or estate-level overviews where relevant.

  - Added a new **Risk & Mitigation** subsection for PHM:

    - Risk of over-promising.
    - Mitigation: honest “early warning, not magic” messaging and tunable rules.

  - Added a new **Trust / One-man band risk** section:

    - Introduced the **Doomsday / Handover Package** concept:
      - “God mode” exports (root access, KNX project, configs, Docker compose).
      - Printed **Safety Breaker** instructions to gracefully shut down the stack and revert to physical controls.
      - A “Yellow Pages” list of alternative integrators.
      - A “Dead Man’s Switch” clause allowing clients to open the sealed pack if Gray Logic is unresponsive for an agreed period.
    - Framed this as a positive trust signal and a counter to proprietary **Golden Handcuffs**.

  - Updated **Success Criteria**:
    - Success now explicitly includes:
      - At least one **PHM pattern** that has proven useful in the real world (e.g. pool pump early warning).
      - Clients seeing real value from offline-first behaviour **and** early warning on plant.

---

## 0.3 – Offline-first model, internal vs external (2025-12-02)

**Docs**

- Updated `docs/gray-logic-stack.md` to **Working Draft v0.3**:

  - Clarified the distinction between **Gray Logic** (company/brand) and the **Gray Logic Stack** (product/architecture).
  - Added an explicit **internal (on-site) vs external (remote)** model:
    - Internal = the on-site Linux/Docker node that must keep running if the internet/VPN is down.
    - External = optional remote services on a VPS, reachable via WireGuard, treated as non-critical bonuses.
  - Stated a clear **offline-first target**:
    - At least **99% of everyday functionality** (lighting, scenes, schedules, modes, local dashboards, plant logic) must work with **no internet connection**.
  - Defined a **Consumer Overlay**:
    - Segregated, non-critical module for consumer-grade IoT (Hue, random smart plugs, etc.).
    - Overlay logic is not allowed to become a dependency for core lighting safety, life safety, or plant operation.
  - Expanded **disaster recovery and rebuild**:
    - Config in Docker volumes/bind mounts.
    - Version-controlled `docker-compose.yml`.
    - Simple host rebuild process described in the spec.

- Updated `docs/business-case.md` to reflect the same model:
  - Internal vs external split baked into the solution overview and value proposition.
  - Support tiers aligned with on-site vs remote:
    - Core = on-site focus and basic support.
    - Enhanced = adds remote monitoring and alerts.
    - Premium = adds remote changes/updates and richer reporting.
  - Success criteria updated to include “clients actually experience the offline reliability benefit”.

---

## 0.2 – Spec + business case foundation

- `docs/gray-logic-stack.md`:

  - First proper written spec for the **Gray Logic Stack**:
    - Goals, principles, hard rules, and module breakdown.
    - Target domains: high-end homes, pools/leisure, small mixed-use/light commercial.
    - “Mini-BMS” positioning vs proprietary smart home platforms and full BMS/SCADA.
  - Described key functional modules:
    - Core (Traefik, dashboard, metrics).
    - Environment monitoring.
    - Lighting & scenes.
    - Media / cinema.
    - Security, alarms, and CCTV.
  - Added a roadmap from v0.1 → v1.0.

- `docs/business-case.md`:
  - Defined the **technical and business problem** as I see it.
  - Identified target customers and stakeholders.
  - Outlined a **project + recurring revenue** model.
  - Wrote down early success criteria and explicit “permission to walk away” conditions.

---

## 0.1 – Initial repo structure

- Created the initial repo layout:
  - `docs/` for specs and business case.
  - `code/` for future Docker Compose and config.
  - `notes/` for brainstorming and meeting notes.
- Dropped in early drafts of the Gray Logic Stack concept and basic architecture ideas.
