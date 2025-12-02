# Changelog – Gray Logic Stack

All notable changes to this project will be documented in this file.

This changelog tracks the **spec and business docs** for the Gray Logic Stack, not code releases (yet).

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
