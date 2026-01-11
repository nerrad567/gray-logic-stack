# Gray Logic Stack - AI Coding Agent Instructions

## Project Overview

The **Gray Logic Stack** is a productized building automation platform for high-end homes, small estates, pools/spas, and light commercial buildings. Built by an electrician who runs production Linux/Docker systems, it bridges the gap between DIY smart home platforms and expensive proprietary BMS solutions.

**Core Philosophy**: Offline-first (99%+ uptime without internet), open standards, safety-first, and maintainable by competent electricians—not just the original installer.

## Architecture & Key Components

### Stack Layers (Bottom-Up)

1. **Field Layer** (Open Standards): KNX, DALI, Modbus, MQTT, dry contacts
2. **Control Layer** (Linux/Docker on-site "mini-NOC"):
   - **openHAB**: Main automation brain (devices, scenes, modes, schedules)
   - **Node-RED**: Cross-system glue logic and Predictive Health Monitoring (PHM)
   - **Traefik**: Reverse proxy, HTTPS termination, single front door
   - **Mosquitto**: MQTT broker (loose coupling between components)
3. **Remote Layer** (Optional VPS): Long-term data retention, pretty dashboards, remote monitoring via WireGuard

### Critical Boundaries

- **Internal (on-site)**: Must work 99%+ offline—lighting, plant control, modes, local UI, PHM checks
- **External (remote)**: Premium bonuses only—multi-year trends, cross-site analytics, remote admin
- **Consumer Overlay**: Non-critical devices (Hue bulbs, smart plugs) segregated from core stack
- **Life Safety**: Fire alarms, E-stops, emergency lighting are *independent*—stack observes, never controls

## Development Patterns & Conventions

### Documentation Structure

- **[docs/gray-logic-stack.md](../docs/gray-logic-stack.md)**: Complete technical spec (1200+ lines, v0.4)
- **[docs/sales-spec.md](../docs/sales-spec.md)**: Canonical sales specification reference (drives future marketing outputs)
- **[docs/business-case.md](../docs/business-case.md)**: Commercial justification, pricing tiers
- **[docs/ai-premium-features.md](../docs/ai-premium-features.md)**: Optional AI usage, guardrails, data-handling defaults
- **[docs/modules/](../docs/modules/)**: Per-module details (core, lighting, security, etc.)
- **[CHANGELOG.md](../CHANGELOG.md)**: Spec evolution (not code releases yet—this is pre-v1.0)

### Docker Compose Patterns

- All services run in Docker containers via [code/stack/docker-compose.yml](../code/stack/docker-compose.yml) (currently placeholder)
- Config stored in **bind mounts/volumes** for easy backup
- **Pinned image tags** per site (no `:latest`)
- Disaster recovery: rebuild site from `docker-compose.yml` + config backups *without* relying on remote server

### Logic Placement Rules

**openHAB**: Device-centric logic (scenes, modes, schedules, bindings)  
**Node-RED**: Cross-system flows (e.g., "alarm armed + time window → heating setpoints + perimeter lights"), PHM logic  
**MQTT**: Optional loose coupling—not mandatory for all integrations

### Predictive Health Monitoring (PHM)

PHM tracks "heartbeat" metrics (pump current, vibration proxies, boiler flow/return temps, run hours) using:

- **Rolling averages** (7-day baselines)
- **Deviation thresholds** (≥20% deviation for >2h → maintenance warning)
- **Local execution**: PHM rules run on-site (offline-capable)
- **Remote premium**: Multi-year trend storage for comparing failures

**Example**: Pool pump current deviates 25% from 7-day average for 3 hours → raise maintenance warning, not full fault.

## Safety-Critical Design Rules (Never Break)

1. **Physical controls always work**: Wall switches, panic buttons, plant room controls function even if server/LAN/internet is down
2. **Life safety is independent**: Fire alarms, E-stops controlled by certified hardware—stack may receive signals but never controls them
3. **No cloud dependencies for core operation**: Internet down = lighting/plant/modes still functional
4. **Consumer Overlay non-critical**: Hue/LIFX/smart plugs never drive plant/security/safety logic

## Coding Guidelines

### When Adding Features

- **Check offline-first**: Will this work with internet down?
- **Document safety boundaries**: Does this touch plant/security/life safety? Document decision rationale.
- **Segregate consumer gear**: Tag overlay items (`Consumer_Overlay_*`), display in separate UI sections
- **No cloud-only CCTV/doorbells**: Ring/Nest-style devices incompatible—require local RTSP/ONVIF (Amcrest, DoorBird, Uniview examples)

### If Adding “AI” Features

AI is allowed only as an optional, premium **insights/summarisation** layer. It must not break the stack’s hard rules.

- **Advisory, not authority**: AI may explain/suggest; it must not directly control plant/security actions.
- **No cloud dependency for core**: Loss of AI/VPS/internet must only pause bonuses, not core operation.
- **Data minimisation by default**: Do not export CCTV media, occupancy/presence timelines, detailed security event timelines, secrets, or raw network identifiers off-site unless explicitly opt-in per site.

### Backup/Restore Workflows

- **[code/scripts/backup.sh](../code/scripts/backup.sh)**: Backup openHAB/Node-RED/Docker volumes (placeholder—implement comprehensively)
- **[code/scripts/restore.sh](../code/scripts/restore.sh)**: Restore to bare metal without VPS dependency (placeholder—document steps clearly)

### Configuration Management

- openHAB: Things, items, rules in mounted volumes
- Node-RED: Flows + credentials in mounted volumes
- Traefik: Configs + ACME storage in mounted volumes
- Use **site repos** (version-controlled per deployment)

## Current State & Roadmap

**Status**: Working Draft v0.4 (spec phase—no production code yet)

### Next Milestones

- **v0.5**: Lab prototype (Docker Compose + simulated KNX/DALI, offline resilience test)
- **v0.6**: Domain demos (Environment + Lighting, Security + CCTV, PHM with real/simulated assets)
- **v1.0**: First real site deployment (home or small client pool)

## Key Files Reference

- Technical spec: [docs/gray-logic-stack.md](../docs/gray-logic-stack.md)
- Business model: [docs/business-case.md](../docs/business-case.md)
- Network design: [docs/diagrams/src/network-segmentation.puml](../docs/diagrams/src/network-segmentation.puml)
- Docker stack: [code/stack/docker-compose.yml](../code/stack/docker-compose.yml)

## Integration Constraints

- **No golden handcuffs**: Open field standards (KNX/DALI/Modbus) so other electricians can maintain
- **Local-first devices**: Avoid cloud-dependent gadgets for core functions
- **Runbook-first**: Every site gets documented panel schedules, I/O maps, KNX group tables, rebuild procedures
