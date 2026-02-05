---
title: Gray Logic Constraints — Canonical Reference
version: 1.0.0
status: active
last_updated: 2026-01-20
extracted_from: ~80 documentation files
---

# Gray Logic Constraints — Canonical Reference

> **Single source of truth. AI agents MUST read this entire file before any work.**
>
> Extracted from ~80 documentation files on 2026-01-20

---

## 1. Critical Boundaries (NEVER BREAK)

**These rules are ABSOLUTE. If a decision requires breaking any of these, it is not a Gray Logic decision.**

> Source: [docs/overview/principles.md](overview/principles.md)

### 1.1 Physical Controls Always Work

Wall switches, buttons, and manual controls MUST function even if:
- Gray Logic server is off
- Local network is down
- Internet is unavailable
- All software has crashed

**Implementation:**
- KNX actuators respond directly to KNX switches (bus-level pairing)
- DALI drivers maintain last state on controller failure
- Critical circuits have manual override capability
- Automation *observes* and *enhances* physical controls, never *replaces* them

### 1.2 Life Safety Systems Are Independent

Fire alarms, emergency lighting, and emergency stops:
- Use their own certified hardware
- Have independent power supplies
- Never depend on Gray Logic for operation
- May *send signals to* Gray Logic (e.g., "fire alarm active")
- Are **NEVER controlled by** Gray Logic

**Specifically prohibited:**
- Remote silencing or resetting of fire alarms
- Software control of emergency stop circuits
- Any logic that could prevent emergency systems from activating

### 1.3 No Cloud Dependencies for Core Operation

The building MUST continue full operation if:
- Internet connection is lost
- A vendor's cloud service is discontinued
- DNS fails globally
- VPN is unreachable

**Cloud/remote services are permitted only for:**
- Remote access (nice-to-have, not required)
- External weather data (graceful fallback to local)
- Push notifications (queue locally if unavailable)
- AI enhancement queries (local AI handles basics)

### 1.4 Multi-Decade Deployment Horizon

Systems deployed today MUST work in 2046 with:
- Security patches only (no forced feature updates)
- No mandatory hardware replacement
- No dependency on services that may not exist
- Maintainability by future technicians

**Implementation:**
- Version-pin all software
- Use mature, stable technologies
- Avoid bleeding-edge dependencies
- Document everything for future maintainers

### 1.5 Open Standards at Field Layer

Physical infrastructure uses open protocols:
- **KNX** for switching, dimming, sensors, keypads
- **DALI** for lighting control
- **Modbus** for plant and energy monitoring
- **SIP** for intercom and door stations
- **RTSP/ONVIF** for CCTV

**Prohibited at field layer:**
- Proprietary-only protocols (Loxone Tree, Crestron, etc.)
- Cloud-dependent devices as primary controls
- Devices without documented local APIs

### 1.6 Customer Owns Their System

Every installation includes:
- Complete documentation (wiring, configuration, network)
- Full access credentials (no dealer-only locks)
- Handover pack enabling another integrator to take over
- No licensing that revokes functionality if support lapses

**The "bus factor" MUST be >1.** If Gray Logic disappears, the system continues.

### 1.7 Privacy by Design

Local processing is the default:
- Voice commands processed on-site
- No audio or video sent to cloud services
- No user behaviour profiling sent externally
- Audit logs stay local (or customer-controlled)

**If external AI is used (optional premium):**
- Customer explicitly enables it
- Only query text sent, not device state
- Can be disabled without losing core functionality

---

## 2. Implementation Priorities (Three Pillars)

> Source: [docs/overview/principles.md](overview/principles.md), [docs/development/DEVELOPMENT-STRATEGY.md](development/DEVELOPMENT-STRATEGY.md)

Within the boundaries of the Hard Rules, all implementation decisions MUST satisfy these three pillars **in order**:

| Pillar | Question | Failure Mode |
|--------|----------|--------------|
| **1. Security** | Can this be exploited? | Compromise, data breach, unauthorised control |
| **2. Resilience** | What happens when this fails? | System down, unsafe state, data loss |
| **3. Speed** | Is this efficient enough for real-time? | Missed deadlines, poor UX, cascading delays |

**Decision Framework:**
1. Hard Rules gate what we build (reject anything that violates them)
2. Security gates how we expose it (reject unsafe designs)
3. Resilience shapes how we build it (fail-safe patterns)
4. Speed validates that it works (performance requirements)

---

## 3. Code Standards

> Source: [docs/development/CODING-STANDARDS.md](development/CODING-STANDARDS.md)

### 3.1 Documentation-First

**Every module, every function, every non-trivial block MUST be documented.**

- Package-level `doc.go` files are REQUIRED
- Exported functions MUST include: Purpose, Parameters, Returns, Security, Example
- Comment intent, not implementation
- Keep docs in sync with code

### 3.2 Error Handling

- Define domain-specific errors (e.g., `ErrSceneNotFound`, `ErrDeviceOffline`)
- Use `fmt.Errorf` with `%w` to preserve error chains
- Log errors with context (entity IDs, operation, error details)
- NEVER log passwords, tokens, or secrets

### 3.3 Testing Requirements

| Code Type | Coverage Requirement |
|-----------|---------------------|
| New code | 80% minimum |
| Security-critical code | 100% |
| Error paths | 100% |

**Test naming:** `Test<FunctionName>_<Scenario>` (e.g., `TestActivateScene_DeviceOffline`)

Use table-driven tests for multiple scenarios.

### 3.4 Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Package | lowercase, single word | `lighting`, `climate` |
| Exported type | PascalCase | `Scene`, `DeviceRegistry` |
| Unexported type | camelCase | `sceneValidator` |
| Interface | PascalCase, often `-er` suffix | `SceneActivator` |
| Constant | PascalCase or UPPER_SNAKE | `MaxDevices` |

### 3.5 Git Commit Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:** feat, fix, docs, style, refactor, perf, test, chore

**One logical change per commit. Small commits preferred.**

---

## 4. Database Rules

> Source: [docs/development/database-schema.md](development/database-schema.md)

### 4.1 The Golden Rule: Additive Changes Only

Once a column or table is created in production, it **MUST NEVER be dropped or renamed** during the active life of that version series (v1.x).

**Why?**
1. Rollback Safety: Old binary ignores new columns but crashes on missing columns
2. Data Preservation: Never accidentally delete user data

### 4.2 Schema Evolution Rules

| Action | Allowed | Constraint |
|--------|---------|------------|
| `CREATE TABLE` | Yes | |
| `ADD COLUMN` | Yes | Must be NULLABLE or have DEFAULT |
| `DROP COLUMN` | NO | Mark deprecated instead |
| `RENAME COLUMN` | NO | Add new column, copy data, keep old |
| Adding indexes | Yes | Ensure data integrity first |

### 4.3 SQLite Specifics

- Use **STRICT** mode for type safety (SQLite 3.37+)
- Enable **WAL** mode (Write-Ahead Logging)
- Set **Synchronous to NORMAL**
- All timestamps stored as **UTC**

### 4.4 Migration Testing

Every schema change MUST pass the **Rollback Test**:
1. Start DB at Version A
2. Migrate to Version B
3. Add data using Version B
4. **Run Version A binary against Version B database**
5. Version A must start and operate normally

---

## 5. Security Requirements

> Source: [docs/development/SECURITY-CHECKLIST.md](development/SECURITY-CHECKLIST.md), [docs/architecture/security-model.md](architecture/security-model.md)

### 5.1 Authentication

- Passwords: Argon2id hashing (64MB memory, 3 iterations)
- JWT: HS256 minimum, validate `alg` header, enforce expiration
- API Keys: 1-year default expiry, hash stored (never plaintext)
- PIN: 3-attempt lockout, 5-minute initial lockout with exponential backoff
- Sessions: 90-day absolute lifetime maximum

### 5.2 Authorization (RBAC)

| Role | Permissions |
|------|-------------|
| admin | `all` |
| facility_manager | Control, configure, view all |
| user | Control own area, limited view |
| guest | View only, limited areas |

**Default deny.** If no permission rule, deny.

### 5.3 Input Validation

- Validate ALL external inputs (API, MQTT, WebSocket, files, config)
- Whitelist approach (allowed values/patterns)
- Enforce length limits and type safety
- SQL: Parameterized queries ONLY (no string concatenation)
- Command: Never pass unsanitized input to shell

### 5.4 Cryptography

- TLS 1.2 minimum (1.3 preferred)
- No `InsecureSkipVerify` in production
- Use `crypto/rand`, not `math/rand`
- Secrets from environment variables, never hardcoded

### 5.5 Logging

**MUST log:**
- Login attempts (success + failure)
- Permission denials
- Configuration changes
- Mode changes, scene activations

**NEVER log:**
- Passwords or secrets
- Full JWT tokens (log last 8 chars only)
- PIN values
- Audio/video content

---

## 6. Testing Standards

> Source: [docs/development/testing-strategy.md](development/testing-strategy.md)

### 6.1 Testing Pyramid

| Level | Speed | Dependencies | Coverage Goal |
|-------|-------|--------------|---------------|
| Unit Tests | <10ms | Mocked interfaces | 80%+ |
| Integration Tests (Simulated) | <5s | Docker (MQTT, VictoriaMetrics) | Critical paths |
| Hardware-in-Loop | Minutes | Physical devices | Pre-release |

### 6.2 Virtual Site Simulator

Use `bridge-sim` for testing without physical hardware:
- Mock 50+ devices across protocols
- Inject events (motion, temperature)
- Verify commands received
- Simulate physics (optional)

---

## 7. MQTT Patterns

> Source: [docs/protocols/mqtt.md](protocols/mqtt.md), [docs/architecture/bridge-interface.md](architecture/bridge-interface.md)

### 7.1 Topic Structure

```
graylogic/
├── command/{protocol}/{address}     # Core → Bridge
├── ack/{protocol}/{address}         # Bridge → Core (acknowledgments)
├── state/{protocol}/{address}       # Bridge → Core (state updates)
├── health/{protocol}                # Bridge health (retained)
├── discovery/{protocol}             # Discovered devices
└── config/{protocol}                # Configuration
```

### 7.2 QoS Requirements

| Topic Pattern | QoS | Retained |
|---------------|-----|----------|
| Commands | 1 | No |
| State updates | 1 | No |
| Health status | 1 | Yes |
| Canonical device state | 1 | Yes |

### 7.3 Bridge Requirements

1. Configure Last Will and Testament (LWT)
2. Publish health every 30 seconds
3. Acknowledge ALL commands immediately
4. Reconnect with exponential backoff

---

## 8. Protocol-Specific Rules

> Sources: [docs/protocols/](protocols/)

### 8.1 KNX

> **Specification Reference:** [docs/protocols/knx-reference.md](protocols/knx-reference.md)
> 
> AI agents MUST read the KNX reference before implementing or modifying any KNX-related code.

**Operational Rules:**
- Physical switches MUST work without Core (bus-level pairing)
- Use knxd as gateway
- Group addresses URL-encoded in MQTT topics (e.g., `1%2F2%2F3`)
- Status telegrams expected from actuators

**Protocol Compliance (from KNX Standard v3.0.0):**
- DPT encoding/decoding MUST match specification exactly
- Telegram structure MUST follow cEMI format for KNXnet/IP
- APCI commands MUST use correct binary codes (see reference Table 1)
- Address formats: Individual `A.L.D`, Group `M/S/G` (3-level)
- Invalid value `0x7FFF` for DPT 9.xxx MUST be handled

**Validation Requirements:**
- Individual address: Area 0-15, Line 0-15, Device 0-255
- Group address (3-level): Main 0-31, Sub 0-7, Group 0-255
- DPT ranges enforced (e.g., DPT 1.xxx = 0 or 1 only)
- Scene numbers: 0-63 (displayed as 1-64)

### 8.2 DALI

- Any DALI gateway works (protocol-agnostic)
- Drivers maintain last state on controller failure
- Individual lamp addressing supported

### 8.3 Modbus

- RTU and TCP variants supported
- Polling architecture (bridges poll devices)
- Host:port:unit address format

---

## 9. Domain Constraints

> Sources: [docs/domains/](domains/), [docs/overview/principles.md](overview/principles.md)

### 9.1 Lighting

- Manual switches MUST always work
- Scene recall MUST complete within 500ms
- Fade times MUST be predictable and consistent
- Track lumen depreciation for PHM

### 9.2 Climate

- **Frost protection runs on thermostat hardware, NOT Core**
- Safe setback temperatures maintained if automation fails
- Manual override possible at the device
- Sensor failure defaults to safe operation

### 9.3 Security

- Alarm panel is primary, Gray Logic is secondary
- Remote arm/disarm requires PIN + confirmation
- All security actions logged
- Physical keypads ALWAYS work

### 9.4 Voice Control

- Wake word detection: LOCAL
- Command processing: LOCAL
- Cloud AI: OPTIONAL enhancement only
- System MUST work with voice disabled

### 9.5 PHM (Predictive Health Monitoring)

- Static logic is primary (rolling averages, thresholds)
- AI enhancement is optional premium
- Alerts are warnings, NOT guarantees
- No automatic action on PHM alerts (notify only)

### 9.6 DIY Devices

- MUST support local control (no cloud-only)
- Isolated on separate VLAN (untrusted)
- No SLA for customer-managed integrations
- Customers responsible for firmware updates

---

## 10. Resilience Requirements

> Sources: [docs/resilience/](resilience/), [docs/overview/principles.md](overview/principles.md)

### 10.1 Graceful Degradation Order

```
Full Function → Automation Disabled → Manual Control Only
```

**Degradation order (from principles.md):**
1. Remote features degrade first
2. Automation degrades second
3. **Manual control NEVER degrades**

### 10.2 Failure Handling

For every failure, define:
1. **Fail-safe state** — What is the safe fallback?
2. **User notification** — How do we alert the user?
3. **Recovery** — Auto-recover or require intervention?
4. **Logging** — What to log for diagnostics?

### 10.3 Hardware Requirements

- **Battery-backed RTC is MANDATORY** (prevents boot deadlocks)
- Industrial-grade hardware only (no Raspberry Pi for Core)
- AI workloads require dedicated NPU/TPU
- Industrial-grade NVMe for VictoriaMetrics

---

## 11. Operations Standards

> Sources: [docs/operations/](operations/)

### 11.1 Updates

- Version-pin all software
- Security patches only (no forced upgrades)
- Rollback plan REQUIRED before updates
- Test rollback in lab first

### 11.2 Backups

- Daily automatic backup to USB
- Weekly integrity verification
- Quarterly restore testing
- Golden backup at commissioning (sealed)

### 11.3 Monitoring

- Health endpoint: `/health`
- Prometheus metrics for all components
- Dead man's switch for critical systems
- Alert on: latency p95 >500ms, error rate >1%, memory >80MB

---

## 12. API & Interface Patterns

> Sources: [docs/interfaces/api.md](interfaces/api.md), [docs/architecture/security-model.md](architecture/security-model.md)

### 12.1 REST API

- HTTPS required (TLS 1.2+)
- JWT Bearer tokens for authentication
- Rate limiting on all endpoints
- Request validation before processing

### 12.2 WebSocket

- Ticket-based authentication (not JWT in URL)
- Single-use tickets, 2-minute expiry
- Use for real-time state updates only

---

## 13. Performance Requirements

> Source: [docs/development/DEVELOPMENT-STRATEGY.md](development/DEVELOPMENT-STRATEGY.md)

| Operation | Target | Maximum |
|-----------|--------|---------|
| UI tap → command sent | 50ms | 100ms |
| Command → device activation | 100ms | 200ms |
| Scene recall (10 devices) | 200ms | 500ms |
| Voice command → action | 1500ms | 2000ms |

| Resource | Baseline | Maximum |
|----------|----------|---------|
| RAM | 30MB | 100MB |
| CPU (idle) | <1% | 20% |

---

## 14. Terminology (MUST Use Correctly)

> Source: [docs/overview/glossary.md](overview/glossary.md)

| Term | Definition |
|------|------------|
| **Site** | Single Gray Logic installation |
| **Area** | Logical grouping (e.g., "Ground Floor") |
| **Room** | Physical space within an area |
| **Device** | Controllable or monitorable entity |
| **Scene** | Predefined collection of device states |
| **Mode** | System-wide state (Home/Away/Night/Holiday) |
| **PHM** | Predictive Health Monitoring |
| **Bridge** | Protocol translator component |

---

## 15. Verification Checklist

Before any deployment or major change, verify:

- [ ] Physical controls work without software
- [ ] Life safety systems are independent
- [ ] System works without internet
- [ ] All components use open standards at field layer
- [ ] Documentation enables handover
- [ ] Privacy defaults to local processing
- [ ] Failures are safe and visible
- [ ] Security is layered appropriately
- [ ] Tests pass (unit + integration)
- [ ] Security checklist completed
- [ ] Database migrations pass rollback test

---

## Related Documents

- [Vision](overview/vision.md) — What Gray Logic is and why
- [Principles](overview/principles.md) — Hard rules (source of truth)
- [Glossary](overview/glossary.md) — Standard terminology
- [Coding Standards](development/CODING-STANDARDS.md) — How to write code
- [Security Checklist](development/SECURITY-CHECKLIST.md) — Security gates
- [Security Model](architecture/security-model.md) — Auth architecture
- [System Overview](architecture/system-overview.md) — Technical architecture
- [MQTT Protocol](protocols/mqtt.md) — Internal message bus
- [Bridge Interface](architecture/bridge-interface.md) — Bridge contract
