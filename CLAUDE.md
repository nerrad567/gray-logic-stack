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
- **VictoriaMetrics**: Time-series for PHM, energy monitoring, and device telemetry
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
│   ├── operations/            # Updates, monitoring, maintenance
│   ├── commissioning/         # Device discovery
│   ├── development/           # Coding standards, strategy, security
│   └── archive/               # Previous documentation versions
├── code/
│   └── core/                  # Gray Logic Core (Go)
│       ├── cmd/graylogic/     # Application entry point (main.go)
│       ├── internal/
│       │   ├── api/           # REST API + WebSocket server (M1.4)
│       │   ├── bridges/knx/   # KNX protocol bridge (M1.2)
│       │   ├── device/        # Device registry (M1.3)
│       │   ├── infrastructure/# Config, database, MQTT, TSDB (M1.1)
│       │   ├── knxd/          # knxd subprocess manager
│       │   └── process/       # Generic subprocess lifecycle
│       ├── configs/           # Configuration templates
│       └── migrations/        # SQL migration files
├── notes/                     # Brainstorming, meeting notes
├── CHANGELOG.md               # Project evolution
└── PROJECT-STATUS.md          # Current progress tracker
```

## Key Documents

Start here:
- **Constraints**: `docs/CONSTRAINTS.md` — Canonical constraints reference (REQUIRED)
- **Go Guidance**: `code/core/AGENTS.md` — Go-specific development guidance
- **Vision**: `docs/overview/vision.md` — What we're building and why
- **Principles**: `docs/overview/principles.md` — Hard rules (source document)
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
| Time-Series | VictoriaMetrics | PHM data, energy monitoring, device telemetry (zero external Go deps) |
| Message Bus | MQTT | Simple, proven, debuggable |
| Wall Panel/Mobile | Flutter | Cross-platform native |
| Voice STT | Whisper | Local, accurate, open |
| Voice TTS | Piper | Local, natural |
| Local AI | Llama/Phi | On-device intelligence |

## Development Workflow

**DEFAULT: Dev mode.** Unless the user explicitly says "production build" or
"full Docker", always assume native Go development with containerised support services.

### Dev Mode (Default — every session)

```bash
cd code/core && make dev-services    # starts mosquitto, knxsim, victoriametrics
cd code/core && make dev-run         # builds + runs Go core natively (~2-3s)
```

Flutter panel (no Go recompile needed):
```bash
cd code/ui/wallpanel && flutter build web --base-href="/panel/"
rm -rf ../../core/internal/panel/web && cp -r build/web ../../core/internal/panel/web
# Or simply: cd code/core && make build-panel-dev
# Panel refreshes in browser — Go core serves from filesystem via GRAYLOGIC_PANEL_DIR
```

### Prod Mode (Explicit request only)

```bash
cd code/core && make docker-up       # builds + starts full containerised stack
```

See `docs/development/dev-workflow.md` for the full reference.

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
7. **Session log sync**: After completing significant coding work (milestone completion, multi-file feature implementation, or when the user indicates they're done), offer to run `/docs-sync` to update CHANGELOG.md and PROJECT-STATUS.md. Ask: *"Session work looks complete — shall I run `/docs-sync` to update project logs?"*

## Common Commands

```bash
# Dev workflow (default)
cd code/core && make dev-services       # Start support services
cd code/core && make dev-run            # Build + run Go core (~2-3s)
cd code/core && make dev-run-quick      # Fastest loop (skip fmt/vet)
cd code/core && make dev-services-down  # Stop support services

# Build + test
cd code/core && make build              # Build binary
cd code/core && go test -v ./...        # Run tests
cd code/core && golangci-lint run       # Lint

# Production Docker (explicit only)
cd code/core && make docker-up          # Full containerised stack
cd code/core && make docker-down        # Stop full stack
```

## Custom Commands

These are available in `.claude/commands/`:

### Workflow Commands

| Command | Purpose |
|---------|---------|
| `/pre-commit` | Run lint + tests + principles check before committing |
| `/code-audit` | Comprehensive 7-phase code audit for stability and security |
| `/milestone-audit` | End-of-milestone quality gate with coverage enforcement |
| `/health-check` | Verify dev environment (Go, Docker, services) |
| `/new-bridge` | Scaffold a new protocol bridge |
| `/nuke-rebuild` | Scorched-earth teardown, rebuild, and fresh stack launch |

### Specialist Review Agents

Focused code reviewers for targeted analysis. Each reviews against our documented standards.

| Command | Agent | Purpose |
|---------|-------|---------|
| `/standards` | Standards Validator | KNX 3.00 compliance + internal principles + MQTT topics |
| `/security` | Security Auditor | Auth, injection, secrets, MQTT security, protocol attack surfaces |
| `/optimise` | Performance Analyst | Efficiency, memory, concurrency, real-time latency |
| `/stability` | Stability Reviewer | Error handling, recovery, race conditions, fault tolerance |
| `/review-all` | Orchestrator | Runs all 4 specialists sequentially with consolidated report |

**Usage:** `/standards internal/bridges/knx/` or `/review-all internal/api/`

**When to use which:**
- `/standards`, `/security`, `/optimise`, `/stability` — Quick focused review of a specific concern during development
- `/review-all` — Comprehensive review of a file or package across all 4 dimensions
- `/code-audit` — Full milestone-grade audit that also runs tools (tests, lint, vulncheck)

**Orchestration:** When any single specialist completes, it will offer to run the remaining specialists. Track which have run to avoid duplicates within a session.

## Current Focus

**Phase**: Year 1 — Foundation  
**Completed**: M1.1 (Infrastructure), M1.2 (KNX Bridge), M1.3 (Device Registry), M1.4 (REST API + WebSocket)  
**Active Work**: See `PROJECT-STATUS.md` for current progress  
**Next**: M1.5
