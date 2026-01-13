# Gray Logic — Project Status

> **Last Updated:** 2026-01-12 (Session 2)  
> **Current Phase:** Documentation (Pre-Development)

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
| Business Docs | ❌ Empty |
| Code | ❌ Not started |

---

## Documentation Status

### ✅ Complete

#### Overview (`docs/overview/`)
- [x] `vision.md` — Product vision and goals
- [x] `principles.md` — Hard rules and design principles
- [x] `glossary.md` — Standard terminology

#### Architecture (`docs/architecture/`)
- [x] `system-overview.md` — High-level architecture
- [x] `core-internals.md` — Go Core package structure
- [x] `bridge-interface.md` — MQTT bridge contract
- [x] `energy-model.md` — Bidirectional energy flows
- [x] `security-model.md` — Authentication and authorization ✓

#### Data Model (`docs/data-model/`)
- [x] `entities.md` — Core entities (Site, Area, Room, Device, Scene, etc.)
- [ ] `schemas/` — JSON Schema definitions (empty)

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

---

### ❌ Empty / Not Started

#### Business (`docs/business/`)
- [ ] `business-case.md` — Business case and market positioning
- [ ] `pricing.md` — Support tiers and pricing model
- [ ] `sales-spec.md` — Sales specification

#### Data Model Schemas (`docs/data-model/schemas/`)
- [ ] JSON Schema definitions for all entities

---

## Code Status

| Component | Status | Notes |
|-----------|--------|-------|
| Gray Logic Core (Go) | ❌ Not started | Documentation complete |
| KNX Bridge | ❌ Not started | Spec complete |
| DALI Bridge | ❌ Not started | Spec complete |
| Modbus Bridge | ❌ Not started | Spec complete |
| Flutter UI | ❌ Not started | API spec complete |
| Voice Pipeline | ❌ Not started | Needs spec |

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

## Session Log

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
- [ ] JSON Schemas for entities
- [ ] Business case documentation

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
