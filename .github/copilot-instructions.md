# Gray Logic - AI Coding Agent Instructions

Gray Logic is a building intelligence platform (competing with Crestron/Savant/Loxone) built on **offline-first**, **open standards**, and **10-year deployment stability**. Currently in v1.0 Architecture Phase (January 2026) — pivoted from openHAB to custom Go-based Core.

## Hard Rules (Never Break)

1. **Physical controls always work** — Wall switches function even if all software is down (KNX actuators respond directly to switches)
2. **Life safety is independent** — Fire alarms, E-stops use certified hardware; Gray Logic *observes*, never *controls*
3. **No cloud dependencies for core** — 99%+ functionality without internet; remote access is nice-to-have only
4. **10-year deployment horizon** — Version-pin everything, prefer mature/stable over bleeding-edge
5. **Open standards at field layer** — KNX, DALI, Modbus only; no proprietary protocols as primary controls
6. **Privacy by design** — Voice processed locally, no cloud surveillance

## Architecture

```
User Interfaces → REST API + WebSocket → GRAY LOGIC CORE (Go) → MQTT → Protocol Bridges
```

- **Gray Logic Core**: Single Go binary containing all automation logic (~30MB RAM)
- **Protocol Bridges**: Separate processes for KNX (via knxd), DALI, Modbus, audio/video matrices
- **Database**: SQLite (config/state), InfluxDB (PHM time-series)
- **UIs**: Flutter (wall panels, mobile), React/Svelte (web admin)
- **Local AI**: Whisper (STT) + local LLM (NLU) + Piper (TTS)

## Key Documentation

| Document | Purpose |
|----------|---------|
| [docs/overview/vision.md](../docs/overview/vision.md) | What we're building and why |
| [docs/overview/principles.md](../docs/overview/principles.md) | Hard rules (read this first) |
| [docs/overview/glossary.md](../docs/overview/glossary.md) | Standard terminology |
| [docs/architecture/system-overview.md](../docs/architecture/system-overview.md) | Technical architecture |
| [docs/data-model/entities.md](../docs/data-model/entities.md) | Core entities (Site, Room, Device, Scene) |

## Data Model Hierarchy

```
Site → Area → Room → Device (with Capabilities, State, HealthData)
Scene (scoped to Room/Area/Site) → Actions
Mode (Home/Away/Night/Holiday) → affects automation site-wide
```

## Development Conventions

- **Documentation format**: YAML frontmatter (title, version, status, depends_on), cross-references
- **Terminology**: Use exact terms from `glossary.md` (Site, Area, Room, Device, Scene, Mode)
- **Modularity**: Each domain (lighting, climate, audio) independent; can be added/removed separately
- **Fail safe**: Failures degrade gracefully (remote → automation → manual control never fails)

## When Making Changes

1. Check **offline-first**: Will this work without internet?
2. Check **10-year viability**: Avoid dependencies that may not exist in 2036
3. Check **safety boundaries**: Does this touch life safety? Document explicit rationale
4. Keep **docs small and focused**: One concept per file
5. Update **related docs**: Cross-reference changes

## Protocol Specifications

| Protocol | Document | Status |
|----------|----------|--------|
| KNX | [docs/protocols/knx.md](../docs/protocols/knx.md) | Complete |
| DALI-2 | [docs/protocols/dali.md](../docs/protocols/dali.md) | Complete |
| Modbus | [docs/protocols/modbus.md](../docs/protocols/modbus.md) | Complete |
| MQTT | [docs/protocols/mqtt.md](../docs/protocols/mqtt.md) | Complete |
| BACnet | [docs/protocols/bacnet.md](../docs/protocols/bacnet.md) | Year 2 placeholder |

## Domain Specifications

| Domain | Document | Status |
|--------|----------|--------|
| Lighting | [docs/domains/lighting.md](../docs/domains/lighting.md) | Complete (residential + commercial) |
| Climate | [docs/domains/climate.md](../docs/domains/climate.md) | Complete (residential + commercial HVAC) |
| Blinds | [docs/domains/blinds.md](../docs/domains/blinds.md) | Complete |
| Plant | [docs/domains/plant.md](../docs/domains/plant.md) | Complete (pumps, VFDs, AHUs, PHM) |

## Integration Specifications

| System | Document | Status |
|--------|----------|--------|
| Access Control | [docs/integration/access-control.md](../docs/integration/access-control.md) | Complete (residential + commercial) |
| CCTV | [docs/integration/cctv.md](../docs/integration/cctv.md) | Complete (residential + commercial) |
| Fire Alarm | [docs/integration/fire-alarm.md](../docs/integration/fire-alarm.md) | Complete (monitoring only) |

## Archive

Previous openHAB-based approach (v0.4) archived in compressed files:
- `docs/archive/v0.4-openhab-era.zip`
- `code/archive/v0.4-openhab-era.zip`

See [CLAUDE.md](../CLAUDE.md) for complete project guidance.
