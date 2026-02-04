# KNXSim Vision & Roadmap

> **Last Updated:** 2026-02-04
> **Status:** Active Development

---

## Vision

**KNXSim** is a powerful KNX simulator/emulator that acts like a **live installation**, allowing engineers to mimic real-world on-site conditions during development and testing.

### Core Principles

1. **Real-World Accurate** — Only simulate what's possible with actual KNX hardware
   - Respect bus limitations (telegram rate, device counts, addressing)
   - Use real DPT encodings (no magic values)
   - Devices behave like their physical counterparts
   - If it can't exist on a real bus, don't allow it in the simulator
2. **Feel Like a Real Installation** — Devices behave realistically, state persists, scenarios run autonomously
3. **Full Technical Visibility** — Inspect GAs, telegrams, actions, and live device status in real-time
4. **Engineer-Focused** — Built for developers and integrators, not end users
5. **Lightweight Stack** — Python backend + simple JS frontend (no heavy frameworks)
6. **Easy to Use** — Spin up, configure devices, start testing immediately

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Web Dashboard (JS)                         │
│  • Live device status (lights on/off, blinds %, temps)          │
│  • Telegram inspector (GA, direction, payload, decoded value)   │
│  • Device controls (toggle, sliders, buttons)                   │
│  • Floor plan view (drag-drop device placement)                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                         WebSocket + REST
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Python Backend (FastAPI)                      │
│  • Premise/Device/Scenario management                           │
│  • Telegram history ring buffer                                 │
│  • Real-time event broadcast                                    │
│  • Template-driven device creation                              │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    KNXnet/IP Server (UDP 3671)                  │
│  • Full tunnelling protocol                                     │
│  • Virtual device state machines                                │
│  • Scenario engine (sine wave, presence, random walk)           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Milestones

### Phase 1: Core Simulator ✅ COMPLETE
- [x] KNXnet/IP tunnelling protocol (UDP 3671)
- [x] Virtual devices: switch, dimmer, blind, sensor, presence
- [x] DPT codec (35+ datapoint types)
- [x] Scenario engine (sine wave, random walk, presence patterns)
- [x] SQLite persistence (devices, state, scenarios)
- [x] REST API (CRUD for premises, devices, floors, rooms)
- [x] WebSocket hub (real-time telegram + state broadcasts)
- [x] Template system (57 YAML templates, 7 domains)
- [x] Multi-premise architecture

### Phase 2: Web Dashboard ✅ COMPLETE

**Target Use Cases:** Any building type — the UI adapts to the context:
- 🏠 **Residential** — Single home, simple floor/room layout
- 🏢 **Commercial** — Office, retail, zones and open-plan areas
- 🏬 **Multi-Occupancy** — Apartments, HMOs, units + communal facilities
- 🏗️ **Mixed Use** — Commercial ground floor, residential above

#### 2.1 Building Overview ✅ COMPLETE
- [x] **Premise Selector** — Switch between multiple premises/buildings
- [x] **Floor Navigation** — Tab-based floor switching with auto-select
- [x] **Room List/Grid** — Card grid showing all rooms on a floor
  - Room name, key status (lights on/off, current temp, occupancy)
  - Click to expand/drill down to devices
- [x] **Building Summary Stats** — Footer bar showing lights on, avg temperature, presence rooms, blinds open (reactive, computed from live device state)

#### 2.3 Device Status & Control ✅ COMPLETE
- [x] **Entity Tiles** — Live state display:
  - 💡 Lights: on/off indicator, brightness slider
  - 🌡️ Sensors: temperature values with units
  - 🪟 Blinds: position and slat angle sliders
  - 👤 Presence: motion detected/clear toggle + lux slider
- [x] **Quick Controls** — Toggle lights, adjust brightness/blinds
- [ ] **State History** — Sparkline showing recent values (deferred to Phase 3 — needs time-series storage)

#### 2.4 Technical Inspection (Engineer Mode) ✅ COMPLETE
- [x] **Device Detail Panel** — Click any device to see:
  - Individual address (1.1.1)
  - All group addresses with DPT types (auto-detected)
  - Current raw state (JSON)
  - Interactive controls by device type
- [x] **Telegram Inspector** — Live bus traffic:
  - Direction indicator (→ TX, ← RX)
  - Source/destination addresses (decoded)
  - GA name and decoded value
  - Real-time WebSocket streaming
- [x] **Bus Statistics** — Collapsible stats panel: total telegrams, TX/RX split, TPS, unique GAs, top 5 busiest GAs (auto-refreshes every 5s)
- [x] **Filter by device/room/GA** — Direction dropdown (RX/TX/All) + text search across device, GA, source, value

#### 2.5 Device & Room Management (ETS-like) ✅ MOSTLY COMPLETE

Like real-world KNX commissioning with ETS, the simulator allows dynamic configuration:

**Floor Management:** ✅ COMPLETE
- [x] **Create Floor** — Add new floor (name, level number, optional plan image)
- [x] **Edit Floor** — Rename, reorder, update plan image
- [x] **Delete Floor** — Remove floor (moves rooms to "Unassigned")

**Room Management:** ✅ COMPLETE
- [x] **Create Room** — Add new rooms via UI (name, floor, type)
- [x] **Edit Room** — Rename, change floor assignment, set room type
- [x] **Delete Room** — Remove room (devices become "Unassigned")
- [x] **Room Types** — Living, bedroom, bathroom, kitchen, hallway, office, utility, etc.
- [x] **Room Positioning** — Grid col/row/width/height for layout control

**Device Management:** ✅ MOSTLY COMPLETE
- [x] **Add Device** — Create new virtual device:
  - Select from template library (dimmer, switch, blind, sensor, etc.)
  - Auto-assign next available individual address
  - Auto-generate group addresses based on room/device naming convention
  - Or manually specify all addresses (advanced mode)
- [x] **Edit Device** — Modify existing device:
  - Change group addresses (rewire the device)
  - Update individual address
  - Modify initial state
- [x] **Delete Device** — Remove device from installation
  - Confirmation dialog
- [x] **Assign to Room** — Drag-drop or dropdown to move device between rooms
- [ ] **Bulk Operations** — Select multiple devices for room assignment or deletion (deferred to Phase 3)

**Address Management:**
- [x] **Individual Address Picker** — Visual tree (Area.Line.Device) with topology + manual modes
- [x] **Group Address Builder** — Guided GA creation with suggestion endpoint
- [x] **Address Conflict Detection** — Proactive warnings for IA and device ID conflicts in create/edit modal
- [ ] **Address Range View** — See which addresses are used/available (deferred to Phase 3)

**Template System:**
- [x] **Template Browser** — Browse available device templates by category
- [x] **Template Preview** — See what GAs/state a template creates before instantiating
- [ ] **Custom Templates** — Save device configuration as reusable template (deferred to Phase 3)
- [ ] **Template Import/Export** — Share templates between installations (deferred to Phase 3)

### Phase 3: Advanced Scenarios & Simulation

#### 3.1 Scenario Management
- [ ] **Scenario Editor** — Create/edit scenarios via UI
- [ ] **Time-of-Day Profiles** — Morning routines, evening modes
- [ ] **Occupancy Simulation** — Whole-building presence patterns

#### 3.2 Device Behaviour Fidelity
Simulated devices should respond realistically over time, not instantly.
This enables testing of Gray Logic Core's intelligence layer (smart PID, pre-heat scheduling, etc.)
against a realistic thermal/mechanical model.

- [ ] **Thermal model for climate devices**
  - Room thermal mass (temperature changes gradually, not instantly)
  - Heat loss rate (temperature drifts down when heating is off)
  - Valve-to-temperature lag (5-15 min for UFH, 1-2 min for radiators)
  - Basic PID response in thermostat (boost to 100%, wind back as target approaches)
- [ ] **Motor simulation for blinds**
  - Travel time (real blinds take 30-60s to fully open/close)
  - Position updates during travel (not just start/end)
- [ ] **Dimmer fade simulation**
  - Transition time for brightness changes
  - Soft start behaviour

### Phase 4: Testing & Debug Tools
- [ ] **Telegram Injection** — Send arbitrary telegrams to test responses
- [ ] **Record/Replay** — Capture traffic, replay for regression testing
- [ ] **Failure Injection** — Simulate device failures, bus errors
- [ ] **Latency Simulation** — Add delays to mimic slow devices

### Phase 5: Integration & Export
- [ ] **ETS Export** — Generate ETS project from simulator config
- [ ] **Config Import** — Load device config from ETS export
- [ ] **API Documentation** — OpenAPI/Swagger UI

### Future / Wishlist
Ideas that are valuable but out of scope for the current roadmap:

- **Visual Floor Plan Editor** — Draw floor plans, drag-and-drop KNX devices onto rooms, wire group addresses visually. Essentially a lightweight ETS-like CAD tool. Could be a standalone project/tool.
- **Multi-Unit Coordination** — Shared HVAC, lift calls, parking simulation for large multi-tenant buildings. Deferred until Gray Logic Core has multi-zone climate control.
- **KNX Secure (S-mode)** — Encrypted telegram support for security-sensitive devices. Requires implementing AES-128-CCM encryption layer on the tunnelling protocol. Important for production deployments but does not change device behaviour on the bus — only adds an encryption wrapper around existing cEMI frames.

---

## Tech Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| Backend | Python 3.12 + FastAPI | Simple, fast, async WebSocket support |
| Frontend | Vanilla JS + minimal CSS | Lightweight, no build step, engineer-friendly |
| Protocol | Pure Python (no external KNX libs) | Full control, minimal dependencies |
| Database | SQLite | Embedded, zero config, portable |
| Container | Docker | Easy deployment, isolation |

**Explicitly Avoided:**
- Heavy JS frameworks (React, Vue, Angular)
- Complex build tooling (webpack, vite)
- External KNX libraries (xknx, knxpy)

---

## Current Status

**Phase 1:** ✅ Complete (4,500+ lines Python, 57 device templates)
**Phase 2:** ✅ Complete — Web Dashboard

**All Phase 2 items completed:**
- 2.1 Building Overview: Premise selector, floor navigation, room grid, building summary stats ✅
- 2.3 Device Status & Control: Entity tiles, quick controls, multi-channel support ✅
- 2.4 Technical Inspection: Device detail panel, telegram inspector, bus statistics, telegram filtering ✅
- 2.5 Floor/Room/Device CRUD: Full create/edit/delete modals ✅
- 2.5 Template browser + create-from-template ✅
- 2.5 Topology management: Areas, lines, device addressing (topology + manual) ✅
- 2.5 Group address hierarchy: Main/middle groups, GA suggestion, drag-drop assignment ✅
- 2.5 Address conflict detection: Proactive IA and device ID warnings in modals ✅
- Loads system: Physical equipment (lights, valves) with actuator channel binding ✅
- Export: ETS .knxproj and .esf symbol file export ✅
- Codex consolidation: DPT codec tests (73 roundtrips), cEMI frame tests, GA normalisation ✅

**Deferred to Phase 3:**
- State history sparklines (needs time-series storage)
- Bulk device operations
- Custom template save-as
- Template import/export
- Address range view

**Next:** Phase 3 — Advanced Scenarios & Simulation

---

## API Endpoints (Current)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/premises` | List premises |
| GET | `/api/v1/premises/{id}/devices` | List devices with live state |
| POST | `/api/v1/premises/{id}/devices` | Create device (IA or line_id + device_number) |
| PATCH | `/api/v1/premises/{id}/devices/{did}` | Update device (GAs, room, addressing) |
| DELETE | `/api/v1/premises/{id}/devices/{did}` | Delete device |
| POST | `/api/v1/premises/{id}/devices/{did}/command` | Send command (switch, brightness, position, presence, lux) |
| GET | `/api/v1/premises/{id}/floors` | List floors with rooms |
| POST | `/api/v1/premises/{id}/floors` | Create floor |
| PATCH | `/api/v1/premises/{id}/floors/{fid}` | Update floor |
| DELETE | `/api/v1/premises/{id}/floors/{fid}` | Delete floor |
| POST | `/api/v1/premises/{id}/floors/{fid}/rooms` | Create room |
| PATCH | `/api/v1/premises/{id}/floors/{fid}/rooms/{rid}` | Update room |
| DELETE | `/api/v1/premises/{id}/floors/{fid}/rooms/{rid}` | Delete room |
| GET | `/api/v1/premises/{id}/telegrams` | Telegram history |
| GET | `/api/v1/templates` | List device templates |
| GET | `/api/v1/premises/{id}/areas` | List areas |
| POST | `/api/v1/premises/{id}/areas` | Create area |
| GET | `/api/v1/premises/{id}/areas/{area_id}/lines` | List lines in area |
| POST | `/api/v1/premises/{id}/areas/{area_id}/lines` | Create line in area |
| GET | `/api/v1/premises/{id}/topology` | Full topology tree (Areas → Lines → Devices) |
| GET | `/api/v1/premises/{id}/lines/{line_id}/next-device-number` | Next available device # on line |
| WS | `/ws/telegrams?premise={id}` | Live telegram stream |
| WS | `/ws/state?premise={id}` | Live device state updates |

### Command Types

| Command | Value | DPT | Description |
|---------|-------|-----|-------------|
| `switch` | bool | 1.001 | Light on/off |
| `brightness` | 0-100 | 5.001 | Dimmer level % |
| `position` | 0-100 | 5.001 | Blind position % |
| `slat` | 0-100 | 5.001 | Blind slat angle % |
| `presence` | bool | 1.018 | Motion detected |
| `lux` | float | 9.004 | Ambient light (lux) |
| `setpoint` | float | 9.001 | Temperature setpoint °C |

### Planned Endpoints (Remaining Helpers)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/addresses/available` | Available individual addresses (by line) |
| GET | `/api/v1/addresses/conflicts` | GA/IA conflict checks |

---

## Running

```bash
# Start simulator
docker compose -f docker-compose.dev.yml up -d knxsim

# Check health
curl http://localhost:9090/api/v1/health

# View devices
curl http://localhost:9090/api/v1/premises/default/devices

# Web dashboard (when ready)
open http://localhost:9090/ui/
```
