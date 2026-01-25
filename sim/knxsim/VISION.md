# KNXSim Vision & Roadmap

> **Last Updated:** 2026-01-25
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
- [x] Template system (20 YAML templates, 6 domains)
- [x] Multi-premise architecture

### Phase 2: Web Dashboard — IN PROGRESS

**Target Use Cases:** Any building type — the UI adapts to the context:
- 🏠 **Residential** — Single home, simple floor/room layout
- 🏢 **Commercial** — Office, retail, zones and open-plan areas
- 🏬 **Multi-Occupancy** — Apartments, HMOs, units + communal facilities
- 🏗️ **Mixed Use** — Commercial ground floor, residential above

#### 2.1 Building Overview
- [ ] **Premise Selector** — Switch between multiple premises/buildings
- [ ] **Floor Navigation** — Tab/dropdown to switch between floors
- [ ] **Simple Floor Plan Layout** — Grid-based room layout (not pixel-perfect CAD)
  - Rooms as clickable regions
  - Visual indication of room status (occupied, temp, lights on)
- [ ] **Building Summary Stats** — Total devices, active alarms, energy usage

#### 2.2 Floor View
- [ ] **Room Grid** — Simple rectangular layout showing all rooms on floor
- [ ] **Room Cards** — Each room shows:
  - Room name
  - Key status (lights on/off, current temp, occupancy)
  - Click to expand/drill down
- [ ] **Communal Areas** — Highlighted differently (lobbies, gyms, parking)
- [ ] **Unit Boundaries** — Visual grouping for apartments/units

#### 2.3 Device Status & Control ✅ COMPLETE
- [x] **Entity Tiles** — Live state display:
  - 💡 Lights: on/off indicator, brightness slider
  - 🌡️ Sensors: temperature values with units
  - 🪟 Blinds: position and slat angle sliders
  - 👤 Presence: motion detected/clear toggle + lux slider
- [x] **Quick Controls** — Toggle lights, adjust brightness/blinds
- [ ] **State History** — Sparkline showing recent values

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
- [ ] **Bus Statistics** — Telegrams/sec, error count, connected clients
- [ ] **Filter by device/room/GA** — Telegram filtering

#### 2.5 Multi-Occupancy Features
- [ ] **Unit View** — Filter to show only one apartment's devices
- [ ] **Tenant Isolation** — Visual separation between private/communal
- [ ] **Shared Facilities** — Lobby, gym, parking, pool status
- [ ] **Per-Unit Summary** — Energy usage, comfort metrics per apartment

#### 2.6 Device & Room Management (ETS-like)

Like real-world KNX commissioning with ETS, the simulator should allow dynamic configuration:

**Floor Management:**
- [ ] **Create Floor** — Add new floor (name, level number, optional plan image)
- [ ] **Edit Floor** — Rename, reorder, update plan image
- [ ] **Delete Floor** — Remove floor (moves rooms to "Unassigned")

**Room Management:**
- [ ] **Create Room** — Add new rooms via UI (name, floor, type)
- [ ] **Edit Room** — Rename, change floor assignment, set room type
- [ ] **Delete Room** — Remove room (devices become "Unassigned")
- [ ] **Room Types** — Living, bedroom, bathroom, kitchen, hallway, office, utility, etc.

**Device Management:**
- [ ] **Add Device** — Create new virtual device:
  - Select from template library (dimmer, switch, blind, sensor, etc.)
  - Auto-assign next available individual address
  - Auto-generate group addresses based on room/device naming convention
  - Or manually specify all addresses (advanced mode)
- [ ] **Edit Device** — Modify existing device:
  - Change group addresses (rewire the device)
  - Update individual address
  - Modify initial state
- [ ] **Delete Device** — Remove device from installation
  - Confirmation dialog
  - Option to reassign GAs to another device
- [ ] **Assign to Room** — Drag-drop or dropdown to move device between rooms
- [ ] **Bulk Operations** — Select multiple devices for room assignment or deletion

**Address Management:**
- [ ] **Individual Address Picker** — Visual tree (Area.Line.Device) with conflict detection
- [ ] **Group Address Builder** — Guided GA creation (Main/Middle/Sub format)
- [ ] **Address Conflict Detection** — Warn if GA already in use by another device
- [ ] **Address Range View** — See which addresses are used/available

**Template System:**
- [ ] **Template Browser** — Browse available device templates by category
- [ ] **Template Preview** — See what GAs/state a template creates
- [ ] **Custom Templates** — Save device configuration as reusable template
- [ ] **Template Import/Export** — Share templates between installations

### Phase 3: Advanced Scenarios & Simulation
- [ ] **Scenario Editor** — Create/edit scenarios via UI
- [ ] **Time-of-Day Profiles** — Morning routines, evening modes
- [ ] **Occupancy Simulation** — Whole-building presence patterns
- [ ] **Weather Integration** — Simulate blinds responding to sun position
- [ ] **Multi-Unit Coordination** — Shared HVAC, lift calls, parking

### Phase 4: Testing & Debug Tools
- [ ] **Telegram Injection** — Send arbitrary telegrams to test responses
- [ ] **Record/Replay** — Capture traffic, replay for regression testing
- [ ] **Failure Injection** — Simulate device failures, bus errors
- [ ] **Latency Simulation** — Add delays to mimic slow devices

### Phase 5: Integration & Export
- [ ] **ETS Export** — Generate ETS project from simulator config
- [ ] **Config Import** — Load device config from ETS export
- [ ] **API Documentation** — OpenAPI/Swagger UI

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

**Phase 1:** ✅ Complete (4,500+ lines Python)
**Phase 2:** 🚧 In Progress — Web Dashboard (2.3, 2.4 complete)

**Completed This Session (2026-01-25):**
- Engineer Mode UI with device controls (lights, blinds, presence, sensors)
- Telegram Inspector with live WebSocket streaming
- Device detail panel with GA inspection and DPT auto-detection
- Presence sensor controls (motion trigger + lux slider)
- Bidirectional sync: UI commands → KNX telegrams → Core updates

**Next Steps:**
1. Device & Room Management UI (2.6) — add/remove devices and rooms like ETS
2. Building overview and floor navigation (2.1, 2.2)
3. Bus statistics panel
4. Telegram filtering by device/room/GA

---

## API Endpoints (Current)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/premises` | List premises |
| GET | `/api/v1/premises/{id}/devices` | List devices with live state |
| POST | `/api/v1/premises/{id}/devices` | Create device |
| POST | `/api/v1/premises/{id}/devices/{did}/command` | Send command (switch, brightness, position, presence, lux) |
| GET | `/api/v1/premises/{id}/floors` | List floors with rooms |
| GET | `/api/v1/premises/{id}/telegrams` | Telegram history |
| GET | `/api/v1/templates` | List device templates |
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

### Planned Endpoints (2.6 Device & Room Management)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/premises/{id}/floors` | Create floor |
| PATCH | `/api/v1/premises/{id}/floors/{fid}` | Update floor |
| DELETE | `/api/v1/premises/{id}/floors/{fid}` | Delete floor |
| POST | `/api/v1/premises/{id}/rooms` | Create room |
| PATCH | `/api/v1/premises/{id}/rooms/{rid}` | Update room |
| DELETE | `/api/v1/premises/{id}/rooms/{rid}` | Delete room |
| PATCH | `/api/v1/premises/{id}/devices/{did}` | Update device (GAs, address) |
| DELETE | `/api/v1/premises/{id}/devices/{did}` | Delete device |
| POST | `/api/v1/premises/{id}/devices/{did}/assign` | Assign device to room |
| GET | `/api/v1/addresses/available` | Get available individual addresses |
| GET | `/api/v1/addresses/conflicts` | Check for GA conflicts |

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
