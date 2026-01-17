---
title: "ADR-001: Go for Core Implementation"
status: accepted
date: 2026-01-15
---

# ADR-001: Go for Core Implementation

## Status

Accepted

## Context

Gray Logic Core is the central automation engine that must:
- Run reliably for 10+ years without major refactoring
- Deploy as a single artifact to edge hardware (NUC, Raspberry Pi)
- Start quickly (<5 seconds cold start)
- Use minimal resources (target 30-50MB RAM)
- Be maintainable by a small team

We considered several languages:

| Language | Pros | Cons |
|----------|------|------|
| **Go** | Single binary, fast compilation, good concurrency, mature ecosystem | Less expressive than some languages |
| **Rust** | Performance, memory safety, single binary | Steep learning curve, slower development |
| **Node.js** | Rapid development, large ecosystem | Runtime dependency, memory usage, less stable for long-running processes |
| **Python** | Rapid development, AI/ML ecosystem | Runtime dependency, performance, typing |
| **Java** | Mature, enterprise support | JVM dependency, memory footprint, startup time |

## Decision

**Use Go for Gray Logic Core.**

Key factors:
1. **Single binary deployment** — No runtime dependencies means simpler deployment and fewer failure modes over 10 years
2. **Concurrency model** — Goroutines and channels are ideal for handling concurrent device state updates
3. **Fast compilation** — Rapid development cycle
4. **Mature ecosystem** — Stable libraries for MQTT, SQLite, HTTP
5. **Cross-compilation** — Easy to build for ARM (Raspberry Pi) and x64 (NUC)
6. **10-year viability** — Go has strong backward compatibility guarantees

## Consequences

### Positive
- Single `graylogic` binary, ~30MB, runs anywhere
- No runtime version management or dependency hell
- Easy cross-compilation for different target platforms
- Strong standard library reduces external dependencies
- Excellent tooling (gofmt, go vet, go test)

### Negative
- Less expressive than Rust or modern TypeScript
- Generic support added relatively recently (may encounter limitations)
- Some boilerplate required compared to dynamic languages
- Team must learn Go if not already familiar

### Mitigations
- Use `sqlc` for type-safe database access to reduce boilerplate
- Use interface-based design for testability
- Follow Go idioms rather than forcing patterns from other languages

## Related

- [ADR-003: SQLite as Primary Database](003-sqlite-primary-database.md)
