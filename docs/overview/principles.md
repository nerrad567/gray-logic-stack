---
title: Gray Logic Principles
version: 1.0.0
status: active
implementation_status: n/a
last_updated: 2026-01-17
depends_on:
  - overview/vision.md
---

# Gray Logic Principles

This document defines the **hard rules** and **design principles** that govern all Gray Logic development and deployment. These are not guidelines — they are constraints that cannot be violated.

---

## Hard Rules (Never Break)

These rules are **absolute**. If a decision requires breaking one of these, it is not a Gray Logic decision.

### 1. Physical Controls Always Work

**Wall switches, buttons, and manual controls must function even if:**
- The Gray Logic server is off
- The local network is down
- The internet is unavailable
- All software has crashed

**Implementation:**
- KNX actuators respond directly to KNX switches (bus-level pairing)
- DALI drivers maintain last state on controller failure
- Critical circuits have manual override capability
- Automation *observes* and *enhances* physical controls, never *replaces* them

### 2. Life Safety Systems Are Independent

**Fire alarms, emergency lighting, and emergency stops:**
- Use their own certified hardware
- Have independent power supplies
- Never depend on Gray Logic for operation
- May *send signals to* Gray Logic (e.g., "fire alarm active")
- Are **never controlled by** Gray Logic

**Specifically prohibited:**
- Remote silencing or resetting of fire alarms
- Software control of emergency stop circuits
- Any logic that could prevent emergency systems from activating

### 3. No Cloud Dependencies for Core Operation

**The building must continue full operation if:**
- Internet connection is lost
- A vendor's cloud service is discontinued
- DNS fails globally
- VPN is unreachable

**Cloud/remote services are permitted only for:**
- Remote access (nice-to-have, not required)
- External weather data (graceful fallback to local)
- Push notifications (queue locally if unavailable)
- AI enhancement queries (local AI handles basics)

### 4. Ten-Year Deployment Horizon

**Systems deployed today must work in 2036 with:**
- Security patches only (no forced feature updates)
- No mandatory hardware replacement
- No dependency on services that may not exist
- Maintainability by future technicians

**Implementation:**
- Version-pin all software
- Use mature, stable technologies
- Avoid bleeding-edge dependencies
- Document everything for future maintainers

### 5. Open Standards at Field Layer

**Physical infrastructure uses open protocols:**
- **KNX** for switching, dimming, sensors, keypads
- **DALI** for lighting control
- **Modbus** for plant and energy monitoring
- **SIP** for intercom and door stations
- **RTSP/ONVIF** for CCTV

**Prohibited at field layer:**
- Proprietary-only protocols (Loxone Tree, Crestron, etc.)
- Cloud-dependent devices as primary controls
- Devices without documented local APIs

### 6. Customer Owns Their System

**Every installation includes:**
- Complete documentation (wiring, configuration, network)
- Full access credentials (no dealer-only locks)
- Handover pack enabling another integrator to take over
- No licensing that revokes functionality if support lapses

**The "bus factor" must be >1.** If Gray Logic disappears, the system continues and can be maintained.

### 7. Privacy by Design

**Local processing is the default:**
- Voice commands processed on-site
- No audio or video sent to cloud services
- No user behaviour profiling sent externally
- Audit logs stay local (or customer-controlled)

**If external AI is used (optional premium):**
- Customer explicitly enables it
- Only query text sent, not device state
- Can be disabled without losing core functionality

---

## Implementation Priorities (Three Pillars)

Within the boundaries of the Hard Rules, all implementation decisions must satisfy these three pillars in order:

| Pillar | Question | Failure Mode |
|--------|----------|--------------|
| **Security** | Can this be exploited? | Compromise, data breach, unauthorised control |
| **Resilience** | What happens when this fails? | System down, unsafe state, data loss |
| **Speed** | Is this efficient enough for real-time? | Missed deadlines, poor UX, cascading delays |

**Decision Framework:**
1. **Hard Rules** gate what we build (reject anything that violates them)
2. **Security** gates how we expose it (reject unsafe designs)
3. **Resilience** shapes how we build it (fail-safe patterns)
4. **Speed** validates that it works (performance requirements)

**Example:**

> *"Should we add cloud-based voice recognition?"*
> 
> 1. **Hard Rules check:** Violates Rule 7 (Privacy by Design) as default → Must be opt-in only, with local fallback
> 2. **Security check:** Voice data in transit → Requires TLS, minimal data sent
> 3. **Resilience check:** What if cloud is down? → Local processing must work
> 4. **Speed check:** Latency acceptable? → <2s for voice commands

See [Development Strategy](../development/DEVELOPMENT-STRATEGY.md) for detailed application of these pillars.

---

## Design Principles (Strong Defaults)

These are defaults that should be followed unless there's explicit justification to deviate.

### Modularity

**Every component should be:**
- Independently deployable
- Independently testable
- Replaceable without rewriting other components
- Documented with clear interfaces

**Domain separation:**
- Lighting doesn't depend on audio implementation
- Security doesn't depend on climate implementation
- Each domain can be added, removed, or upgraded independently

### Simplicity Over Cleverness

**Prefer:**
- Direct integrations over complex chains
- Explicit configuration over magic inference
- Boring, proven technology over exciting new options
- Fewer features done well over many features done poorly

**Avoid:**
- Rube Goldberg automation chains
- Implicit behaviour that surprises users
- "Smart" defaults that are wrong 20% of the time

### Fail Safe, Fail Visible

**When things go wrong:**
- Fail to a safe state (lights on, heating safe setback)
- Make failures visible (dashboard alerts, local indicators)
- Log failures for diagnosis
- Never fail silently

**System degradation order:**
1. Remote features degrade first
2. Automation degrades second
3. Manual control never degrades

### Local First

**Processing happens on-site:**
- All automation logic runs locally
- All state management is local
- All UIs can operate from LAN only
- Remote is a *viewer* into local state, not the *source* of truth

### Explicit Over Implicit

**Configuration should be:**
- Readable by humans
- Version-controllable
- Diff-able between versions
- Self-documenting where possible

**Avoid:**
- Binary configuration blobs
- GUI-only configuration
- State that can't be inspected or exported

### Defence in Depth

**Security is layered:**
- Network segmentation (VLANs)
- Authentication at every boundary
- Authorization (role-based access)
- Audit logging for sensitive actions
- Encryption for remote access

**Assume breach:**
- Compromise of one component shouldn't compromise all
- Credentials are scoped to minimum necessary access
- Sensitive actions require confirmation

---

## Domain-Specific Rules

### Lighting

- Manual switches must always work
- Minimum illumination level must be achievable without automation
- Scene recall must complete within 500ms of trigger
- Fade times must be predictable and consistent

### Climate

- Safe setback temperatures must be maintained if automation fails
- Frost protection cannot be disabled by software **and must run on thermostat hardware, not depend on Core**
- Manual override must be possible at the device
- Sensor failure defaults to safe operation

### Security

- Alarm panel is the primary system, Gray Logic is secondary
- Remote arm/disarm requires PIN + confirmation
- All security actions are logged
- Physical keypads always work

### Audio/Video

- Local sources must work without internet
- Streaming sources are convenience, not core
- Volume limits enforceable at system level
- Mute/emergency override always available

### Voice Control

- Wake word detection happens locally
- Command processing happens locally
- Cloud AI is optional enhancement only
- System must work with voice disabled

### PHM (Predictive Health Monitoring)

- Static logic is primary (rolling averages, thresholds)
- AI enhancement is optional premium feature
- Alerts are warnings, not guarantees
- No action is taken automatically on PHM alerts (notify only)

---

## Compliance Checklist

Before any deployment or major change, verify:

- [ ] Physical controls work without software
- [ ] Life safety systems are independent
- [ ] System works without internet
- [ ] All components use open standards at field layer
- [ ] Documentation enables handover
- [ ] Privacy defaults to local processing
- [ ] Failures are safe and visible
- [ ] Security is layered appropriately

---

## Related Documents

- [Vision](vision.md) — What Gray Logic is and why
- [System Overview](../architecture/system-overview.md) — Technical architecture
- [Security Model](../architecture/security-model.md) — Authentication and authorization
- [Development Strategy](../development/DEVELOPMENT-STRATEGY.md) — Applying principles to development
- [Security Checklist](../development/SECURITY-CHECKLIST.md) — Security verification gates