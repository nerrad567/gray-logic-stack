---
title: "ADR-002: MQTT as Internal Message Bus"
status: accepted
date: 2026-01-15
---

# ADR-002: MQTT as Internal Message Bus

## Status

Accepted

## Context

Gray Logic Core needs to communicate with Protocol Bridges (KNX, DALI, Modbus, etc.). These bridges:
- May run as separate processes or containers
- Need to send state updates to Core
- Need to receive commands from Core
- Should be loosely coupled for independent deployment

We considered several messaging patterns:

| Option | Pros | Cons |
|--------|------|------|
| **MQTT** | Simple, mature, debuggable, good QoS, retained messages | External process (Mosquitto), potential SPOF |
| **ZeroMQ** | Fast, embedded, no broker | More complex, less debuggable |
| **NATS** | Modern, fast, clustering | Less mature, more complex |
| **gRPC** | Type-safe, efficient | Point-to-point, less flexible |
| **Redis Pub/Sub** | Fast, additional data structures | Another dependency, memory-based |
| **Direct IPC** | Simplest | Tight coupling, hard to debug |

## Decision

**Use MQTT (Eclipse Mosquitto) as the internal message bus.**

Key factors:
1. **Simplicity** — MQTT is well-understood and easy to debug with CLI tools
2. **Decoupling** — Pub/sub pattern allows bridges to operate independently
3. **Retained messages** — Bridges can recover current state on reconnect
4. **QoS levels** — Guaranteed delivery for commands
5. **Last Will and Testament** — Automatic offline detection
6. **Maturity** — Mosquitto has been stable for 15+ years
7. **Tooling** — MQTT Explorer, mosquitto_sub/pub for debugging

## Consequences

### Positive
- Easy to debug message flow with standard tools
- Bridges can be written in any language
- Retained messages simplify state recovery
- LWT provides automatic offline detection
- Well-documented, many examples available

### Negative
- External process (Mosquitto) is a potential single point of failure
- Additional deployment complexity vs embedded solution
- Network overhead vs direct function calls

### Mitigations for SPOF

The MQTT broker SPOF is addressed through:

1. **Docker restart policy** — `restart: unless-stopped` ensures automatic recovery
2. **Health monitoring** — Core monitors Mosquitto health and alerts on failure
3. **Physical controls still work** — KNX/DALI devices have direct links independent of software
4. **Future enhancement** — Consider embedded broker fallback for Year 4+

See: [docs/operations/monitoring.md](../../operations/monitoring.md) — MQTT Broker Health & Auto-Recovery

### Why Not Embedded?

We chose external Mosquitto over embedding an MQTT broker because:
- Mosquitto is battle-tested and well-maintained
- Separate process means broker survives Core restarts
- Easier to upgrade broker independently
- Simpler Core codebase

## Alternatives Considered

### ZeroMQ
Rejected because: harder to debug, no retained messages equivalent, more complex API

### gRPC
Rejected because: point-to-point model doesn't fit publish/subscribe pattern well

### NATS
Considered for future: if scaling to multiple Core instances becomes necessary

## Related

- [MQTT Protocol Specification](../../protocols/mqtt.md)
- [Bridge Interface Specification](../bridge-interface.md)
- [Monitoring - MQTT Health](../../operations/monitoring.md)
