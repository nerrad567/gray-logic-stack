# Changelog – Gray Logic Stack

All notable changes to this project will be documented in this file.

This changelog tracks the **spec and business docs** for the Gray Logic Stack, not code releases (yet).

---

## 0.4 – Predictive Health, Doomsday Pack & Golden Handcuffs (2025-12-02)

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
