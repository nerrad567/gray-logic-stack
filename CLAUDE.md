# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) and other AI assistants when working with code in this repository.

## Project Overview

**Gray Logic** is a complete building intelligence platform — the central nervous system of a property. It rivals and aims to surpass systems like Crestron, Savant, Lutron, Control4, and Loxone while maintaining complete openness, true offline capability, and zero vendor lock-in.

**Core Philosophy**: Offline-first (99%+ functionality without internet), open standards at field layer (KNX, DALI, Modbus), safety-first, multi-decade deployment stability, and zero vendor lock-in.

**Current Status**: v1.0 Architecture Phase (January 2026). We have pivoted from an openHAB-based approach to building a custom **Gray Logic Core** in Go. Previous openHAB/Node-RED documentation is archived in `docs/archive/v0.4-openhab-era.zip`.

## Architecture

Gray Logic uses a custom-built automation core with protocol bridges:

```
User Interfaces (Wall Panels, Mobile App, Voice, Web Admin)
                            │
                    REST API + WebSocket
                            │
┌───────────────────────────▼───────────────────────────────────┐
│                    GRAY LOGIC CORE (Go)                        │
│  Intelligence Layer: AI Engine, Voice/NLU, Presence, Learning  │
│  Automation Layer: Scenes, Scheduler, Modes, Events, Logic     │
│  Device Layer: Registry, State, Commands, Discovery, PHM       │
│  Infrastructure: API Server, WebSocket, SQLite, MQTT, Auth     │
└───────────────────────────┬───────────────────────────────────┘
                            │
                     Internal MQTT Bus
                            │
┌───────────────────────────▼───────────────────────────────────┐
│                    PROTOCOL BRIDGES                            │
│  KNX │ DALI │ Modbus │ Audio Matrix │ Video │ Security │ CCTV │
└───────────────────────────────────────────────────────────────┘
```

### Key Components

- **Gray Logic Core**: Single Go binary containing all automation logic
- **Protocol Bridges**: Separate processes for KNX (via knxd), DALI, Modbus, audio/video matrices
- **SQLite**: Configuration and state persistence
- **InfluxDB**: Time-series for PHM and energy monitoring
- **MQTT**: Internal message bus between Core and bridges
- **Flutter**: Wall panel and mobile app UIs
- **Local AI**: Whisper (STT) + local LLM (NLU) + Piper (TTS)

### Critical Boundaries (Hard Rules - Never Break)

1. **Physical controls always work**: Wall switches, buttons function even if all software is down
2. **Life safety is independent**: Fire alarms, E-stops use certified hardware — stack observes, never controls
3. **No cloud dependencies for core**: Internet down = everything except remote access still works
4. **Multi-decade deployment horizon**: Version-pin, security patches only, no forced upgrades
5. **Open standards at field layer**: KNX, DALI, Modbus — no proprietary lock-in
6. **Customer owns their system**: Full documentation, handover pack, no dealer locks
7. **Privacy by design**: Voice processed locally, no cloud surveillance

## Repository Structure

```
gray-logic-stack/
├── docs/
│   ├── overview/              # Vision, principles, glossary
│   │   ├── vision.md          # What Gray Logic is and why
│   │   ├── principles.md      # Hard rules that cannot be broken
│   │   └── glossary.md        # Standard terminology
│   ├── architecture/          # System design
│   │   └── system-overview.md # Component architecture, data flow
│   ├── data-model/            # Entity definitions
│   │   ├── entities.md        # Site, Area, Room, Device, Scene, etc.
│   │   └── schemas/           # JSON Schema definitions
│   ├── domains/               # Per-domain specifications (lighting, climate, etc.)
│   ├── automation/            # Scenes, schedules, modes, events
│   ├── intelligence/          # Voice, PHM, AI features
│   ├── resilience/            # Offline operation, satellite weather, radio
│   ├── protocols/             # Protocol bridge specifications
│   ├── interfaces/            # API specs, UI specifications
│   ├── deployment/            # Installation, commissioning, hardware
│   ├── business/              # Business case, pricing
│   └── archive/               # Previous documentation versions
│       └── v0.4-openhab-era.zip  # Archived openHAB-based approach (compressed)
├── code/                      # (To be created as development proceeds)
│   └── archive/
│       └── v0.4-openhab-era.zip  # Archived Docker Compose stack (compressed)
├── notes/                     # Brainstorming, meeting notes
└── CHANGELOG.md               # Project evolution
```

## Key Documents

Start here:
- **Vision**: `docs/overview/vision.md` — What we're building and why
- **Principles**: `docs/overview/principles.md` — Hard rules that cannot be broken
- **System Overview**: `docs/architecture/system-overview.md` — Technical architecture
- **Entities**: `docs/data-model/entities.md` — Core data model

Archived (for reference):
- `docs/archive/v0.4-openhab-era.zip` — Previous openHAB-based specifications (compressed)

## Development Approach

### Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Core | Go | Single binary, no runtime, multi-decade stability |
| Database | SQLite | Embedded, zero maintenance |
| Time-Series | InfluxDB | PHM data, energy monitoring |
| Message Bus | MQTT | Simple, proven, debuggable |
| Wall Panel/Mobile | Flutter | Cross-platform native |
| Voice STT | Whisper | Local, accurate, open |
| Voice TTS | Piper | Local, natural |
| Local AI | Llama/Phi | On-device intelligence |

### When Adding Features

1. **Check offline-first**: Will this work without internet?
2. **Check multi-decade viability**: Will this still work in 2036 and beyond?
3. **Check safety boundaries**: Does this touch life safety? Document rationale.
4. **Maintain modularity**: Can this component be replaced independently?
5. **Document for AI**: Keep docs modular, use YAML frontmatter, be explicit

### Documentation Standards

- All docs have YAML frontmatter (title, version, status, dependencies)
- Cross-reference related documents
- Keep docs small and focused (one concept per file)
- Use consistent terminology from `glossary.md`
- Machine-readable schemas where applicable

## Roadmap (5-Year Part-Time)

| Phase | Focus | Key Milestone |
|-------|-------|---------------|
| Year 1 | Foundation | Core + KNX + lighting in own home |
| Year 2 | Automation | Full scenes, modes, blinds, climate |
| Year 3 | Integration | Audio, video, security, CCTV |
| Year 4 | Intelligence | Voice control, PHM, AI |
| Year 5 | Product | Commissioning tools, first customer |

## AI Assistant Guidelines

When working on this project:

1. **Read the principles first**: `docs/overview/principles.md` contains hard rules
2. **Use standard terminology**: See `docs/overview/glossary.md`
3. **Maintain modularity**: Don't create monolithic files
4. **Respect the multi-decade goal**: Avoid bleeding-edge, prefer stable
5. **Offline-first always**: Never add cloud dependencies for core features
6. **Document changes**: Update relevant docs when making changes

## Common Commands

```bash
# Build
cd code/core && go build -o bin/graylogic ./cmd/graylogic

# Run tests
cd code/core && go test -v ./...

# Lint
cd code/core && golangci-lint run

# Run locally (once complete)
./code/core/bin/graylogic --config config/dev.yaml

# Start dev services (MQTT + InfluxDB)
docker compose -f docker-compose.dev.yml up -d
```

## Custom Commands

These are available in `.claude/commands/`:

| Command | Purpose |
|---------|---------|
| `/pre-commit` | Run lint + tests + principles check before committing |
| `/check-principles` | Validate changes against Gray Logic hard boundaries |
| `/audit-deps` | Check Go dependencies for multi-decade viability |
| `/health-check` | Verify dev environment (Go, Docker, services) |
| `/add-domain` | Scaffold a new domain specification |
| `/new-bridge` | Scaffold a new protocol bridge |
| `/add-entity` | Add entity to data model with JSON schema |

## Current Focus

**Phase**: Year 1 — Foundation  
**Milestone**: M1.1 Core Infrastructure  
**Active Work**: See `PROJECT-STATUS.md` for current progress  
**Next**: Docker Compose for dev services, main.go wiring
