# Gray Logic — Project Status

> **Last Updated:** 2026-01-12  
> **Current Phase:** Documentation (Pre-Development)

---

## Quick Summary

| Area | Status |
|------|--------|
| Core Documentation | ✅ Complete |
| Architecture | ✅ Complete |
| Domain Specs | 🟡 4/7 complete |
| Protocol Specs | ✅ Complete |
| Integration Specs | ✅ Complete |
| API Specification | ✅ Complete |
| Automation Spec | ✅ Complete |
| Intelligence Specs | 🟡 1/3 complete |
| Resilience Specs | ❌ Empty |
| Deployment Guides | 🟡 1/2 complete |
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

#### Data Model (`docs/data-model/`)
- [x] `entities.md` — Core entities (Site, Area, Room, Device, Scene, etc.)
- [ ] `schemas/` — JSON Schema definitions (empty)

#### Protocols (`docs/protocols/`)
- [x] `knx.md` — KNX integration via knxd
- [x] `dali.md` — DALI lighting control
- [x] `modbus.md` — Modbus RTU/TCP for plant equipment
- [x] `mqtt.md` — Internal message bus
- [x] `bacnet.md` — BACnet roadmap (Year 2 placeholder)

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
- [x] `handover-pack-template.md` — Customer handover template

#### Intelligence (`docs/intelligence/`)
- [x] `ai-premium-features.md` — AI feature boundaries

---

### 🟡 Partially Complete / In Progress

#### Domains — Missing Specs
- [ ] `audio.md` — Multi-room audio
- [ ] `security.md` — Alarm system integration
- [ ] `energy.md` — Energy management domain

#### Deployment — Missing Guides
- [ ] `residential.md` — Residential deployment guide

#### Intelligence — Missing Specs
- [ ] `voice.md` — Voice pipeline (Whisper, NLU, Piper)
- [ ] `phm.md` — Predictive Health Monitoring details

---

### ❌ Empty / Not Started

#### Resilience (`docs/resilience/`)
- [ ] `offline.md` — Offline behavior and graceful degradation
- [ ] `backup.md` — Backup and recovery procedures
- [ ] `satellite-weather.md` — Weather nowcast integration
- [ ] `mesh-comms.md` — LoRa/Meshtastic out-of-band comms

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

**Reorganized:**
- `ai-premium-features.md` → `docs/intelligence/`
- `handover-pack-template.md` → `docs/deployment/`

**Fixed:**
- Updated all openHAB/Node-RED references to Go Core architecture
- Fixed broken links to non-existent files
- Added cross-references between related documents

**Infrastructure:**
- Set GitHub repository to private

---

## Next Actions

### High Priority (Documentation)
1. [ ] Audio domain spec (`docs/domains/audio.md`)
2. [ ] Residential deployment guide (`docs/deployment/residential.md`)
3. [ ] Voice pipeline spec (`docs/intelligence/voice.md`)

### Medium Priority (Documentation)
4. [ ] Security domain spec (`docs/domains/security.md`)
5. [ ] Energy domain spec (`docs/domains/energy.md`)
6. [ ] PHM specification (`docs/intelligence/phm.md`)
7. [ ] Backup & recovery (`docs/resilience/backup.md`)

### Lower Priority (Can Wait)
8. [ ] JSON Schemas for entities
9. [ ] Business case refresh
10. [ ] Offline behavior spec

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
