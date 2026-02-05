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
8. **Codex delegation check**: Before starting any non-trivial task, evaluate whether part or all of it would be better suited to GPT-5.2-Codex. See "Multi-Agent Workflow" below.
9. **Atomic commits per milestone/feature**: Never combine work from multiple milestones or independent features into a single commit. Each milestone or logical feature gets its own commit so it can be reverted independently. When working on multiple milestones in parallel (e.g. Claude on M2.1 API layer while Codex handles M2.1 repos), commit each milestone's work separately — not as one combined blob. If shared files (router.go, server.go, main.go) have changes from multiple milestones, commit the first milestone's changes to those files before starting the second milestone's modifications.

## Multi-Agent Workflow — Claude as Orchestrator

Gray Logic uses a **two-agent strategy**. Claude Opus is the **orchestrator, controller, and architect** — responsible for planning work, maintaining architectural consistency, and writing specific task instructions for GPT-5.2-Codex when its strengths apply. Both agents share the same repository and folder structure.

### Claude's Orchestration Responsibilities

1. **Assess every non-trivial task** for Codex delegation opportunities
2. **Write Codex task briefs** to `tasks/` — complete, self-contained instructions
3. **Review Codex output** when the user brings it back — verify it meets standards
4. **Maintain architectural coherence** — Claude owns the big picture, Codex executes bounded tasks

### CRITICAL — No Duplicate Work Rule

**Before starting implementation of ANY task, Claude MUST:**

1. **Check `tasks/` directory** for existing task briefs that cover the same work
2. **Ask the user** if Codex is currently running or queued to run on overlapping files/packages
3. **If a task brief exists for the work** — DO NOT implement it. The brief was written for Codex. Claude's job is to work on *different* sub-tasks that don't overlap, or wait.

**This rule exists because:** Context compaction can cause Claude to forget that it already delegated work to Codex. After compaction, Claude may see "Phase 1-3 pending" in a plan and start implementing — not realising those phases were already assigned to Codex via a task brief. The result is two agents writing identical code in parallel, wasting time and causing conflicts.

**The safeguard is simple:** Always check `tasks/` before writing code. If a brief covers the work, it's Codex's job. If no brief exists, it's Claude's job.

### Codex Strengths (Delegate These)

| Task Type | Why Codex Excels | Example |
|-----------|-----------------|---------|
| **Security audit / hardening** | Stronger cybersecurity capabilities | "Audit auth middleware for OWASP Top 10" |
| **Long-horizon refactoring** | Context compaction across 20+ files | "Add domain scoping to all handlers" |
| **Large migration tasks** | Methodical, consistent, patient | "Migrate all error checks to `errors.Is()`" |
| **Novel architecture design** | Deep reasoning on unconventional problems | "Design commercial multi-tenant framework" |
| **Comprehensive test suites** | Thorough, won't rush edge cases | "Full auth flow integration tests" |
| **Cross-cutting concerns** | Consistency across entire codebase | "Add audit logging to every write handler" |

### Claude Strengths (Keep In-House)

| Task Type | Why Claude Excels | Example |
|-----------|------------------|---------|
| **Interactive debugging** | Higher root cause accuracy, fewer iterations | "This handler returns 500, find why" |
| **Rapid feature sprints** | Fast, matches existing codebase patterns | "Add a new GET endpoint for panel status" |
| **Code review / explanation** | Understands existing code intent deeply | "Walk me through KNX bridge flow" |
| **Real-time pair programming** | Adapts to feedback instantly | "Let's build this together" |
| **Architectural decisions** | Owns the big picture, maintains consistency | "Should we use Option A or B?" |

### Task Brief Format

When delegating to Codex, Claude writes a task brief to `tasks/<task-name>.md`:

```markdown
# Task: [Short Description]

## Context
[What exists, why this task matters, architectural constraints]

## Objective
[Exactly what Codex should produce — files, functions, tests]

## Key Files
[List every file Codex needs to read or modify, with brief descriptions]

## Constraints
- Must pass `golangci-lint run` with zero issues
- Must pass `go test -race -count=1 ./...`
- Follow existing patterns in [reference file]
- Do NOT modify [protected files] without explicit instruction
- [Any project-specific rules: no cloud deps, offline-first, etc.]

## Acceptance Criteria
- [ ] [Specific, testable criterion]
- [ ] [Specific, testable criterion]
- [ ] All tests pass, lint clean, builds successfully

## Reference Patterns
[Code snippets or file references showing the style/approach to follow]
```

### Orchestration Flow

```
User requests work
       │
  Claude plans task
       │
  ┌────▼─────────────────────────────────┐
  │ Delegation assessment:                │
  │                                       │
  │ 📋 Task: [description]               │
  │ ├── A: [Claude — reason]             │
  │ ├── B: [Codex — reason]              │
  │ └── C: [Claude — reason]             │
  │                                       │
  │ For Codex sub-tasks:                  │
  │ → Write brief to tasks/              │
  │ → Tell user: "Task B ready for Codex"│
  └──────────────────────────────────────┘
       │
  Claude executes own sub-tasks
       │
  User runs Codex on its brief
       │
  Claude reviews Codex output
  (build, test, lint, architectural fit)
       │
  Integration complete
```

### Review Checklist (After Codex Completes)

When reviewing Codex output, Claude should verify:
- [ ] Builds cleanly (`go build ./...`)
- [ ] Tests pass with race detection (`go test -race -count=1 ./...`)
- [ ] Lint clean (`golangci-lint run`)
- [ ] Follows existing code patterns and naming conventions
- [ ] No security regressions (no hardcoded secrets, no SQL injection, etc.)
- [ ] Respects project principles (offline-first, no cloud deps, safety boundaries)
- [ ] Consistent with the architectural decisions in the plan

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

# Performance + resilience testing
cd code/core && make test-bench         # Run benchmarks, compare with baseline
cd code/core && make bench-baseline     # Save benchmark baseline
cd code/core && make test-fuzz          # Run fuzz tests (30s each)
cd code/core && make test-resilience    # Run resilience tests (TestResilience_*)

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

### Adversarial & Performance Agents

Proactive testing agents that verify the system works under stress and attack.

| Command | Agent | Purpose |
|---------|-------|---------|
| `/red-team` | Adversarial Tester | Business logic exploits, auth bypass, privilege escalation, input abuse |
| `/bench` | Benchmark Runner | Performance baselines, regression detection, profiling |
| `/chaos` | Resilience Tester | Fault injection, recovery validation, failure mode analysis |

**Usage:** `/standards internal/bridges/knx/` or `/review-all internal/api/`

**When to use which:**
- `/standards`, `/security`, `/optimise`, `/stability` — Quick focused review of a specific concern during development
- `/review-all` — Comprehensive review of a file or package across all 4 dimensions
- `/code-audit` — Full milestone-grade audit that also runs tools (tests, lint, vulncheck)
- `/red-team` — Offensive security testing (complements `/security` which is defensive)
- `/bench` — Run after milestones to establish performance baselines; run before commits to detect regressions
- `/chaos` — Validate resilience patterns actually work; run at milestone boundaries

**Orchestration:** When any single specialist completes, it will offer to run the remaining specialists. Track which have run to avoid duplicates within a session.

### Makefile Test Targets

| Target | Purpose |
|--------|---------|
| `make test-bench` | Run all Go benchmarks, compare with saved baseline |
| `make bench-baseline` | Save current benchmark results as the new baseline |
| `make test-fuzz` | Run Go fuzz tests (30s per fuzz function) |
| `make test-resilience` | Run resilience tests only (`TestResilience_*` prefix) |

## Current Focus

**Phase**: Year 1 Complete — Transitioning to Year 2  
**Completed**: M1.1 (Infrastructure), M1.2 (KNX Bridge), M1.3 (Device Registry), M1.4 (REST API + WebSocket), M1.5 (Flutter Wall Panel), M1.6 (Basic Scenes), M1.7 (Auth Hardening)  
**Active Work**: See `PROJECT-STATUS.md` for current progress  
**Next**: Year 2 kick-off (M2.1 Area/Room hierarchy), Flutter auth integration
