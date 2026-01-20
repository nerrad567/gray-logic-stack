# AGENTS.md

This file provides guidance for AI coding agents (Amp, Claude, Cursor) when working on the Gray Logic Stack.

## Project Overview

**Gray Logic** is a building automation platform — the central nervous system for properties. Custom Go Core with protocol bridges for KNX, DALI, Modbus.

**Core Philosophy**: Offline-first, open standards, multi-decade stability, zero vendor lock-in.

## Architecture

```
User Interfaces (Wall Panels, Mobile, Voice, Web Admin)
                    │
            REST API + WebSocket
                    │
┌───────────────────▼───────────────────────────┐
│            GRAY LOGIC CORE (Go)                │
│  Intelligence │ Automation │ Device │ Infra   │
└───────────────────┬───────────────────────────┘
                    │
             Internal MQTT Bus
                    │
┌───────────────────▼───────────────────────────┐
│              PROTOCOL BRIDGES                  │
│  KNX │ DALI │ Modbus │ Audio │ Video │ CCTV   │
└───────────────────────────────────────────────┘
```

## Critical Boundaries — NEVER BREAK

1. **Physical controls always work** — Wall switches function if software is down
2. **Life safety is independent** — Fire alarms, E-stops: observe only, never control
3. **No cloud for core** — 99%+ works without internet
4. **Multi-decade stability** — Pin versions, no bleeding-edge
5. **Open standards** — KNX, DALI, Modbus at field layer
6. **Privacy** — Voice/surveillance processed locally

## Technology Stack

| Component | Tech | Rationale |
|-----------|------|-----------|
| Core | Go | Single binary, multi-decade stability |
| DB | SQLite | Embedded, zero maintenance |
| Time-series | InfluxDB | PHM, energy data |
| Message bus | MQTT | Simple, proven |
| UI | Flutter | Cross-platform |

## Common Commands

```bash
# Build
cd code/core && go build -o bin/graylogic ./cmd/graylogic

# Test
cd code/core && go test -race ./...

# Lint
cd code/core && golangci-lint run

# Dev services
docker compose -f docker-compose.dev.yml up -d
```

## Key Documents

- `docs/overview/principles.md` — Hard rules
- `docs/overview/glossary.md` — Terminology
- `docs/architecture/system-overview.md` — Technical design
- `docs/data-model/entities.md` — Data model
- `docs/development/CODING-STANDARDS.md` — Code style

## Patterns Discovered

<!-- Ralph will append learnings here -->

## Gotchas

- MQTT tests require running broker: `docker compose -f docker-compose.dev.yml up -d`
- SQLite uses CGO (mattn/go-sqlite3), cross-compile needs special handling
- Migrations are additive-only, never ALTER/DROP
