# GEMINI.md

This file provides project-specific guidance for the Gemini CLI agent when working on the **Gray Logic Stack**.

## Project Overview

**Gray Logic** is a next-generation building intelligence platform designed as a central nervous system for properties. It emphasizes complete openness, true offline capability, and zero vendor lock-in.

- **Core Philosophy**: Offline-first (99%+ functionality without internet), open standards (KNX, DALI, Modbus), safety-first, multi-decade deployment stability.
- **Current Status**: v1.0 Architecture Phase (January 2026). Custom **Gray Logic Core** being built in Go.

## Architecture

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

### Key Technologies
- **Core**: Go (Single binary, stability)
- **Database**: SQLite (Configuration/State), InfluxDB (Time-series/PHM)
- **Message Bus**: MQTT (Internal communication)
- **UI**: Flutter (Wall panels and mobile)
- **Local AI**: Whisper (STT), Piper (TTS), local LLM (NLU)

## Critical Boundaries (Never Break)

1. **Physical controls always work**: Software failure must not break wall switches.
2. **Life safety is independent**: Fire alarms and E-stops use certified hardware; the stack only observes.
3. **No cloud dependencies for core**: Internet loss must not affect local automation.
4. **Multi-decade deployment horizon**: Use stable, version-pinned dependencies.
5. **Privacy by design**: Local processing of voice and surveillance data.

## Development Standards

### Documentation-First
- Every module, function, and non-trivial block must be documented.
- Package-level `doc.go` files are required.
- Use Go doc comments for exported symbols (Purpose, Parameters, Returns, Security, Example).

### Go Coding Patterns
- **Error Handling**: Use domain-specific errors and wrap them using `%w`.
- **Concurrency**: Use mutexes or channels to prevent race conditions in scene/device management.
- **Testing**: 80%+ coverage for new code; 100% for security-critical paths. Use table-driven tests.

### File Organization
Follow the directory structure defined in `docs/architecture/core-internals.md`:
- `cmd/`: Application entry points.
- `internal/`: Private code (infrastructure, device, automation, intelligence, domain layers).
- `pkg/`: Public, reusable packages (models, protocols).
- `migrations/`: Versioned SQL migrations.

## AI Assistant Guidelines (Gemini Specific)

When working on this project, adhere to these rules:

1. **Be Proactive & Analytical**: Gemini is expected to analyze the entire context (architecture, data models, domain specs) before proposing or implementing changes.
2. **Respect the Multi-Decade Goal**: Prefer stable Go patterns over bleeding-edge features. Avoid unnecessary external dependencies.
3. **Offline-First Implementation**: When adding features, ensure they work entirely on the local network.
4. **Security Mindset**: Always validate inputs, use parameterized queries, and never log sensitive data. Check `docs/development/SECURITY-CHECKLIST.md`.
5. **Documentation Maintenance**: If you change code, you MUST update the corresponding documentation in `docs/` and the inline comments.
6. **Git Excellence**: Follow the commit message format: `<type>(<scope>): <subject>`. Keep commits atomic.

## Key Documents to Reference
- `docs/overview/vision.md`: Core mission.
- `docs/overview/principles.md`: Hard rules.
- `docs/architecture/system-overview.md`: Technical design.
- `docs/development/CODING-STANDARDS.md`: Detailed style guide.
- `docs/data-model/entities.md`: Core entity definitions.

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

## Current Focus

**Phase**: Year 1 — Foundation  
**Milestone**: M1.1 Core Infrastructure  
**Active Work**: See `PROJECT-STATUS.md` for current progress  
**Next**: Docker Compose for dev services, main.go wiring
