# Technical Documentation

> Deep technical documentation for Gray Logic Core maintainers (human and AI).

This directory contains comprehensive documentation covering the **what**, **how**, **why**, and **expected outcomes** of all implementations.

---

## Quick Navigation

### Package Design Docs

| Package | Purpose | Status |
|---------|---------|--------|
| [database](packages/database.md) | SQLite with WAL mode, embedded migrations | Active |
| [mqtt](packages/mqtt.md) | MQTT client with auto-reconnect | Active |
| [config](packages/config.md) | YAML configuration with env overrides | Active |
| [influxdb](packages/influxdb.md) | InfluxDB time-series for PHM/energy | Active |
| [logging](packages/logging.md) | Structured logging with slog | Active |
| [knx-bridge](packages/knx-bridge.md) | KNX protocol bridge via knxd daemon | Active |
| [knxd-manager](packages/knxd-manager.md) | knxd daemon lifecycle management | Active |
| [device-registry](packages/device-registry.md) | Device catalogue with caching | Active |
| [process-manager](packages/process-manager.md) | Generic subprocess management | Active |

### Architecture Docs

| Topic | Purpose |
|-------|---------|
| [Data Flow](architecture/data-flow.md) | How data moves through the system |
| [Error Handling](architecture/error-handling.md) | Error patterns and propagation |

### Implementation Decisions

| ID | Title | Package |
|----|-------|---------|
| [IMP-001](decisions/IMP-001-embedded-migrations.md) | Embedded Migrations with embed.FS | database |
| [IMP-002](decisions/IMP-002-wal-mode.md) | WAL Mode for SQLite | database |
| [IMP-003](decisions/IMP-003-subscription-tracking.md) | Subscription Tracking for Reconnect | mqtt |

---

## Documentation Standards

### Package Design Docs

Each package doc includes:
- **Purpose** — What and why
- **Architecture** — Key types, component diagram
- **How It Works** — Initialization → Operations → Shutdown
- **Design Decisions** — Rationale, alternatives
- **Interactions** — Dependencies and dependents
- **Error Handling** — Error types and propagation
- **Thread Safety** — Concurrency patterns
- **Configuration** — YAML options
- **Testing** — How to test
- **Known Limitations** — Constraints and future work

### Implementation Decision Records (IMP-ADRs)

For significant implementation choices:
- **Status** — Accepted/Proposed/Deprecated
- **Context** — Problem being solved
- **Decision** — What we chose
- **Consequences** — Pros, cons, risks
- **Alternatives** — What else we considered

---

## Related Documentation

- [GETTING-STARTED.md](../GETTING-STARTED.md) — Conceptual introduction
- [PROJECT-STATUS.md](../../../../PROJECT-STATUS.md) — Session-by-session development log
- [docs/architecture/decisions/](../../../../docs/architecture/decisions/) — High-level ADRs (Go, MQTT, SQLite)
- [docs/development/CODING-STANDARDS.md](../../../../docs/development/CODING-STANDARDS.md) — Style guide

---

*Last updated: 2026-01-22*
