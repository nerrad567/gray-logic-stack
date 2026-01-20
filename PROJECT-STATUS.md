# Gray Logic — Project Status

> **Last Updated:** 2026-01-20
> **Current Phase:** Implementation (M1.2 - KNX Bridge)

---

## Quick Summary

| Area | Status |
|------|--------|
| Core Documentation | ✅ Complete |
| Architecture | ✅ Complete |
| Domain Specs | ✅ 13/13 complete |
| Protocol Specs | ✅ Complete |
| Integration Specs | ✅ Complete |
| API Specification | ✅ Complete |
| Automation Spec | ✅ Complete |
| Intelligence Specs | ✅ Complete |
| Resilience Specs | ✅ Complete |
| Deployment Guides | ✅ Complete |
| Business Docs | ✅ Complete |
| Development Docs | ✅ Complete |
| Operations Docs | ✅ Complete |
| Commissioning Docs | ✅ Complete |
| Code | 🟢 M1.2 In Progress |

---

## Documentation Status

### ✅ Complete

#### Overview (`docs/overview/`)
- [x] `vision.md` — Product vision and goals
- [x] `principles.md` — Hard rules and design principles
- [x] `vision.md` — Product vision and goals
- [x] `principles.md` — Hard rules and design principles
- [x] `capabilities-and-benefits.md` — Capabilities summary v2.0
- [x] `glossary.md` — Standard terminology

#### Architecture (`docs/architecture/`)
- [x] `system-overview.md` — High-level architecture
- [x] `core-internals.md` — Go Core package structure
- [x] `bridge-interface.md` — MQTT bridge contract
- [x] `energy-model.md` — Bidirectional energy flows
- [x] `security-model.md` — Authentication and authorization ✓
- [x] `cloud-relay.md` — Cloud services architecture v0.1.0

#### Data Model (`docs/data-model/`)
- [x] `entities.md` — Core entities (Site, Area, Room, Device, Scene, etc.)
- [x] `schemas/` — JSON Schema definitions for all entities ✓

#### Protocols (`docs/protocols/`)
- [x] `knx.md` — KNX integration via knxd
- [x] `dali.md` — DALI lighting control
- [x] `modbus.md` — Modbus RTU/TCP for plant equipment
- [x] `mqtt.md` — Internal message bus
- [x] `bacnet.md` — BACnet roadmap (Year 2 placeholder)
- [x] `ocpp.md` — EV charging protocol ✓

#### Integrations (`docs/integration/`)
- [x] `cctv.md` — CCTV and video surveillance
- [x] `access-control.md` — Door access, intercoms, gates
- [x] `fire-alarm.md` — Fire alarm monitoring (observation only)
- [x] `diy-integration.md` — DIY device integration spec (Matter/Zigbee)
- [x] `access-control.md` — Door access, intercoms, gates
- [x] `fire-alarm.md` — Fire alarm monitoring (observation only)

#### Interfaces (`docs/interfaces/`)
- [x] `api.md` — REST and WebSocket API specification

#### Automation (`docs/automation/`)
- [x] `automation.md` — Scenes, schedules, modes, conditions, events

#### Domains (`docs/domains/`)
- [x] `lighting.md` — Lighting control + commercial
- [x] `climate.md` — HVAC + commercial
- [x] `blinds.md` — Shading and blind control
- [x] `plant.md` — Plant room equipment

#### Deployment (`docs/deployment/`)
- [x] `office-commercial.md` — Commercial deployment guide
- [x] `residential.md` — Residential deployment guide ✓
- [x] `handover-pack-template.md` — Customer handover template

#### Intelligence (`docs/intelligence/`)
- [x] `ai-premium-features.md` — AI feature boundaries
- [x] `phm.md` — Predictive Health Monitoring framework ✓
- [x] `voice.md` — Voice pipeline specification (Whisper, NLU, Piper) ✓
- [x] `weather.md` — Weather integration ✓

#### Resilience (`docs/resilience/`)
- [x] `offline.md` — Offline behavior and graceful degradation ✓
- [x] `backup.md` — Backup and recovery procedures ✓
- [x] `satellite-weather.md` — Satellite weather decode ✓
- [x] `mesh-comms.md` — LoRa/Meshtastic mesh communications ✓

#### Operations (`docs/operations/`)
- [x] `updates.md` — Update and upgrade strategy ✓
- [x] `monitoring.md` — Monitoring and alerting ✓
- [x] `maintenance.md` — System maintenance procedures
- [x] `monitoring.md` — Monitoring and alerting ✓

#### Commissioning (`docs/commissioning/`)
- [x] `discovery.md` — Device discovery specification ✓

---

### 🟡 Partially Complete / In Progress

#### Domains — Complete ✓
- [x] `lighting.md` — Lighting control
- [x] `climate.md` — HVAC and climate
- [x] `blinds.md` — Blinds and shading
- [x] `plant.md` — Plant room equipment
- [x] `audio.md` — Multi-room audio
- [x] `security.md` — Alarm system integration
- [x] `energy.md` — Energy management
- [x] `video.md` — Video/AV distribution
- [x] `irrigation.md` — Garden and outdoor
- [x] `leak-protection.md` — Leak detection and auto-shutoff
- [x] `water-management.md` — Rainwater, greywater, treatment
- [x] `presence.md` — Occupancy and presence detection
- [x] `pool.md` — Pool chemistry, covers, water features

#### Deployment — Complete ✓

#### Intelligence — Complete ✓

#### Resilience — Complete ✓

#### Business (`docs/business/`) — Complete ✓
- [x] `business-case.md` — Business case, market analysis, open source strategy
- [x] `pricing.md` — Installation tiers, hardware strategy, support tiers
- [x] `sales-spec.md` — Customer journey, proposals, contracts, installation
- [x] `go-to-market.md` — Phased growth strategy, marketing channels
- [x] `certification.md` — Training curriculum, partner benefits, quality control
- [x] `certification.md` — Training curriculum, partner benefits, quality control
- [x] `institutional-principles.md` — Building for generations, succession, knowledge preservation
- [x] `subscription-pricing.md` — Cloud subscription tier details

---

### ❌ Empty / Not Started

*All documentation complete.*

---

## Code Status

| Component | Status | Notes |
|-----------|--------|-------|
| Gray Logic Core (Go) | 🟢 M1.2 In Progress | M1.1 complete, KNX bridge 95% done |
| KNX Bridge | 🟢 95% complete | Core complete, integration tests pending |
| DALI Bridge | ❌ Not started | Spec complete (Year 2) |
| Modbus Bridge | ❌ Not started | Spec complete (Year 2) |
| Flutter UI | ❌ Not started | M1.5 (later Year 1) |
| Voice Pipeline | ❌ Not started | Year 4 |

### M1.1 Progress (Core Infrastructure) — ✅ Complete
- [x] Go module initialised
- [x] Directory structure created
- [x] Makefile with build automation
- [x] golangci-lint configured
- [x] Configuration system (YAML + env vars)
- [x] SQLite database package with migrations
- [x] MQTT client package with auto-reconnect
- [x] InfluxDB client package
- [x] Docker Compose (Mosquitto, InfluxDB)
- [x] Structured logging
- [x] Infrastructure wired into main.go

### M1.2 Progress (KNX Bridge) — 🔨 In Progress (95%)
- [x] telegram.go — KNX telegram parsing/encoding
- [x] knxd.go — knxd client (TCP/Unix socket)
- [x] address.go — Group address parsing
- [x] dpt.go — Datapoint type encoding/decoding
- [x] config.go — Bridge configuration with YAML + env vars
- [x] messages.go — MQTT message types (command, ack, state, health)
- [x] health.go — Health status reporting to MQTT
- [x] bridge.go — Main orchestration (KNX ↔ MQTT translation)
- [x] Comprehensive unit tests (91 tests passing)
- [ ] Integration tests with real MQTT + mock knxd

---

## Roadmap

### Year 1 (2026) — Foundation
- [ ] Complete all documentation
- [ ] Gray Logic Core MVP (Go)
- [ ] KNX Bridge
- [ ] SQLite database
- [ ] Basic REST API
- [ ] Lighting control in own home

### Year 2 (2027) — Expansion
- [ ] Full scenes, modes, schedules
- [ ] Climate control
- [ ] Blinds control
- [ ] Flutter mobile app
- [ ] DALI Bridge
- [ ] Modbus Bridge

### Year 3 (2028) — Features
- [ ] Multi-room audio
- [ ] Video distribution
- [ ] Security integration
- [ ] CCTV integration
- [ ] BACnet Bridge

### Year 4 (2029) — Intelligence
- [ ] Voice control (local)
- [ ] PHM (Predictive Health)
- [ ] Local AI insights

### Year 5 (2030) — Commercial
- [ ] Commissioning tools
- [ ] First customer deployment
- [ ] Support tier implementation

---

## Change Log

### 2026-01-12 — Documentation Sprint

**Created:**
- `docs/interfaces/api.md` — Full REST/WebSocket API specification (~1,100 lines)
- `docs/automation/automation.md` — Comprehensive automation spec (~750 lines)
- `docs/domains/audio.md` — Multi-room audio domain specification (~600 lines)
- `docs/intelligence/phm.md` — Predictive Health Monitoring specification (~750 lines)
- `docs/domains/security.md` — Security/alarm domain specification (~700 lines)
- `docs/domains/energy.md` — Energy management domain specification (~800 lines)
- `docs/domains/video.md` — Video/AV distribution (~650 lines)
- `docs/domains/irrigation.md` — Garden and outdoor (~650 lines)
- `docs/domains/leak-protection.md` — Leak detection and shutoff (~600 lines)
- `docs/domains/water-management.md` — Water infrastructure (~550 lines)
- `docs/domains/presence.md` — Occupancy and presence (~650 lines)
- `docs/domains/pool.md` — Pool chemistry and automation (~750 lines)
- `docs/intelligence/voice.md` — Voice pipeline specification (Whisper, NLU, Piper) (~1,000 lines)
- `docs/deployment/residential.md` — Residential deployment guide (~1,200 lines)

### 2026-01-13 — Resilience & Infrastructure Sprint

**Created:**
- `docs/architecture/security-model.md` — Authentication, authorization, encryption (~750 lines)
- `docs/resilience/offline.md` — Offline behavior and graceful degradation (~650 lines)
- `docs/resilience/backup.md` — Backup and recovery procedures (~700 lines)
- `docs/intelligence/weather.md` — Weather integration specification (~700 lines)
- `docs/resilience/satellite-weather.md` — Satellite weather decode (~550 lines)
- `docs/resilience/mesh-comms.md` — LoRa/Meshtastic mesh communications (~650 lines)
- `docs/protocols/ocpp.md` — EV charging protocol specification (~750 lines)

**Fixed:**
- Broken cross-references to security-model.md, weather.md, ocpp.md
- Added Resilience category (was empty)
- Completed all referenced but missing documents

**Architecture additions:**
- `DeviceAssociation` entity — External monitoring and control proxy relationships
- `Association Resolver` component — Handles data attribution and command routing
- I/O device types — Relay modules, analog/digital I/O, external sensors

**Reorganized:**
- `ai-premium-features.md` → `docs/intelligence/`
- `handover-pack-template.md` → `docs/deployment/`

**Fixed:**
- Updated all openHAB/Node-RED references to Go Core architecture
- Fixed broken links to non-existent files
- Added cross-references between related documents
- Standardized PHM Integration sections across all domain specs (lighting, climate, blinds, plant, audio)
- Added DeviceAssociation entity for external monitoring and control proxying
- Documented Association Resolver in Core architecture
- Added device-level energy attribution via associations

**Infrastructure:**
- Set GitHub repository to private
- Configured sparse-checkout to exclude archive folders locally

### 2026-01-14 — Business Documentation Sprint

**Created:**
- `docs/business/business-case.md` — Market analysis, competitor landscape, open source strategy, positioning (~650 lines)
- `docs/business/pricing.md` — Installation tiers (Essential/Standard/Premium/Estate), hardware pricing, support tiers, margin guidance (~600 lines)
- `docs/business/sales-spec.md` — Full customer journey from enquiry to post-installation support (~700 lines)
- `docs/business/go-to-market.md` — Phased growth strategy (Foundation → Growth → Scale), marketing channels, portfolio development (~550 lines)
- `docs/business/certification.md` — Training curriculum, certification levels, partner benefits, quality control framework (~600 lines)

**Business model defined:**
- Phase 1: Boutique installer (Years 1-3) — direct installation, prove the product
- Phase 2: Growth (Years 3-5) — training courses, certification pilot, referral network
- Phase 3: Scale (Year 5+) — certification programme, hardware wholesale, exit viability

**Pricing tiers established:**
- Essential: £8k-15k (lighting + scenes)
- Standard: £15k-25k (+ climate + blinds)
- Premium: £25k-40k (+ audio + security)
- Estate: £40k+ (multiple buildings)

**Open source strategy documented:**
- Software open source (GPL v3) for transparency and longevity
- Revenue from: installation services, custom hardware, support contracts, training, certification

**Licensing and trademark:**
- Changed from MIT to GPL v3 (copyleft ensures derivatives stay open)
- Created LICENSE file with GPL v3 text
- Documented trademark strategy (brand protection + GPL work together)
- Updated all license references across documentation

**Institutional framing added:**
- Created `docs/business/institutional-principles.md` — building for generations
- Focus on enduring value: knowledge, reputation, network, brand, physical assets
- Decision framework prioritising 30-year impact over short-term gains
- Succession principles (choice, not obligation)
- Acknowledgement that economic systems may change, but human needs remain

**JSON Schemas created:**
- Created `docs/data-model/schemas/` with 13 schema files + README
- `common.schema.json` — Shared enums, embedded types (~400 lines)
- `device.schema.json` — All device types, protocols, capabilities (~200 lines)
- Core entities: site, area, room, scene, schedule, mode, condition, user
- Supporting: device-association, audio-zone, climate-zone
- README with validation and code generation examples
- Total: ~1,800 lines of JSON Schema definitions

### 2026-01-15 — Pre-Development Review & Refinements

**Comprehensive documentation review completed.** Identified and addressed gaps, inconsistencies, and missing specifications.

**New Documents Created:**
- `docs/operations/updates.md` — Update and upgrade strategy, rollback procedures, offline updates (~450 lines)
- `docs/operations/monitoring.md` — Customer-facing and installer monitoring, dead man's switch, Prometheus metrics (~400 lines)
- `docs/commissioning/discovery.md` — Device discovery per protocol (KNX, DALI, Modbus, IP), staging workflow (~350 lines)

**Major Updates:**
- `docs/architecture/bridge-interface.md` — Added MQTT command acknowledgment (`graylogic/ack/`) for tracking command delivery
- `docs/architecture/core-internals.md` — Clarified `monitors_and_controls` association behavior and resolution priority
- `docs/automation/automation.md` — Added `SceneExecution` entity for tracking scene activation progress
- `docs/resilience/offline.md` — Added timestamp-based conflict resolution, race condition prevention, time synchronization spec
- `docs/intelligence/voice.md` — Added fallback path (CPU Whisper, pre-recorded responses), error tones, i18n roadmap
- `docs/intelligence/phm.md` — Added device-type-specific baseline requirements (immediate vs gradual feedback)
- `docs/architecture/security-model.md` — Added JWT rotation procedure, API key regeneration, MQTT mTLS option
- `docs/development/CODING-STANDARDS.md` — Added database migration strategy, structured logging standard, testing with hardware strategy
- `docs/architecture/system-overview.md` — Added multi-site architecture section, capacity planning guide, confirmed Svelte for Web Admin

**Technology Decisions Finalized:**
- Web Admin framework: **Svelte** (not React)
- DALI gateway: Protocol-agnostic (any gateway works, not vendor-specific)

**Review Findings Addressed:**
- MQTT command acknowledgment gap → Added `graylogic/ack/{protocol}/{address}` topic
- Scene execution tracking → Added `SceneExecution` entity with status tracking
- State reconciliation races → Added timestamp-based conflict resolution
- Voice fallback missing → Added degradation hierarchy with audible/visual feedback
- PHM baseline data requirements → Added device-type categories (immediate/gradual/event/inferred)
- No upgrade strategy → Created `operations/updates.md`
- No monitoring strategy → Created `operations/monitoring.md` with front-end dead man's switch
- Device discovery under-specified → Created `commissioning/discovery.md`

### 2026-01-17 — AI Assistant Context

**Created:**
- `GEMINI.md` — Project-specific guidance for the Gemini CLI agent, including architecture overview, philosophy, coding standards, and interaction rules.

**Updated:**
- [x] `CHANGELOG.md` — Recorded creation of `GEMINI.md`.
- [x] `PROJECT-STATUS.md` — Updated status and change log.

### 2026-01-18 — Audit Completion & Readiness
**Verified & Audited:**
- Completed Audit Iterations 5-8 (Consistency, Pre-Implementation, Surgical Strikes, Final Verification)
- Applied all surgical strikes (H1-H2, M1-M4, L1-L8) to documentation
- Achieved **9.8/10 Readiness Score** (Ready for Code)

**New Documents:**
- `docs/overview/capabilities-and-benefits.md` (v2.0) — Major rewrite for clarity and feature definition
- `docs/integration/diy-integration.md` — Full spec for Matter, Zigbee, and DIY device handling
- `docs/architecture/cloud-relay.md` — Architecture for optional cloud services
- `docs/business/subscription-pricing.md` — Detailed pricing for cloud tiers
- `docs/operations/maintenance.md` — Certificate rotation, backup limits, device replacement

**Status:**
- Documentation phase explicitly marked COMPLETE
- Ready to begin Go Core implementation (Year 1 Roadmap)

### 2026-01-15 — Development Documentation Sprint

**Created:**
- `docs/development/DEVELOPMENT-STRATEGY.md` — 5-year roadmap with milestones, Three Pillars framework, security SDL (~550 lines)
- `docs/development/CODING-STANDARDS.md` — Go code standards, project structure, testing, git commits (~1,000 lines)
- `docs/development/SECURITY-CHECKLIST.md` — Mandatory security gates for components, PRs, releases (~800 lines)

**Three Pillars framework established:**
- Security → Resilience → Speed (implementation priorities within Hard Rules)
- Integrated into principles.md as "Implementation Priorities" section
- Decision framework: Hard Rules gate what we build, Pillars guide how

**Milestones defined:**
- Year 1: M1.1-M1.6 (Infrastructure → KNX → Device Registry → API → Flutter → Scenes)
- Year 2: M2.1-M2.8 (Rooms → Scenes → Modes → Scheduler → DALI → Blinds → Climate → Mobile)
- Year 3: M3.1-M3.6 (Audio → Video → Security → CCTV → BACnet → Logic Engine)
- Year 4: M4.1-M4.5 (Voice → PHM → AI → Learning → Energy Insights)
- Year 5: M5.1-M5.7 (Commissioning → Backup → Remote → Docs → Handover → Testing → Customer)

**Updated:**
- principles.md — Added Three Pillars section
- README.md — Added development docs to Quick Links
- system-overview.md — Fixed broken api-rest.md link

**Reviewed and rejected:**
- Copilot-generated `copilot/add-development-guidance-docs` branch
- Issues: Wrong roadmap years, incorrect package structure, missing InfluxDB
- Created corrected versions aligned with existing documentation

---

## Next Actions

### High Priority (Documentation)
1. [x] Audio domain spec (`docs/domains/audio.md`) ✓
2. [x] Residential deployment guide (`docs/deployment/residential.md`) ✓
3. [x] Voice pipeline spec (`docs/intelligence/voice.md`) ✓

### Medium Priority (Documentation)
4. [x] Security domain spec (`docs/domains/security.md`) ✓
5. [x] Energy domain spec (`docs/domains/energy.md`) ✓
6. [x] PHM specification (`docs/intelligence/phm.md`) ✓
7. [x] Backup & recovery (`docs/resilience/backup.md`) ✓
8. [x] Security model (`docs/architecture/security-model.md`) ✓
9. [x] Weather integration (`docs/intelligence/weather.md`) ✓
10. [x] Resilience specs (offline, satellite-weather, mesh-comms) ✓
11. [x] OCPP protocol (`docs/protocols/ocpp.md`) ✓

### Lower Priority (Can Wait)
- [x] JSON Schemas for entities ✓
- [x] Business documentation ✓

### Code (When Ready)
- [ ] Set up Go project structure
- [ ] Implement Core skeleton
- [ ] SQLite schema from entities.md

---

## Notes

- Architecture pivoted from openHAB/Node-RED to custom Go Core (v1.0.0)
- Old documentation archived in `docs/archive/v0.4-openhab-era.zip`
- This is a 5-year part-time project
- First real deployment target: own home (Year 1)
