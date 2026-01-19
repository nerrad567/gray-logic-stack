---
title: Development Strategy
version: 1.0.0
status: active
last_updated: 2026-01-15
depends_on:
  - overview/principles.md
  - architecture/core-internals.md
  - architecture/security-model.md
---

# Gray Logic Development Strategy

This document defines how we build Gray Logic Core — methodically, securely, and with resilience as a first-class requirement.

---

## Philosophy

### Hard Rules First

All development decisions must satisfy the **7 Hard Rules** defined in [principles.md](../overview/principles.md):

1. **Physical controls always work** — Wall switches function even if all software is down
2. **Life safety systems are independent** — Fire alarms, E-stops use certified hardware
3. **No cloud dependencies for core operation** — Full function without internet
4. **Multi-decade deployment horizon** — Systems deployed today work in 2046
5. **Open standards at field layer** — KNX, DALI, Modbus — no proprietary lock-in
6. **Customer owns their system** — Full documentation, no dealer locks
7. **Privacy by design** — Voice processed locally, no cloud surveillance

**If a design or feature requires breaking any of these rules, it is not a Gray Logic feature.**

### The Three Pillars (Implementation Priorities)

Within the boundaries of the Hard Rules, every implementation decision must satisfy these three pillars:

| Pillar | Question | Failure Mode |
|--------|----------|--------------|
| **Security** | Can this be exploited? | Compromise, data breach, unauthorised control |
| **Resilience** | What happens when this fails? | System down, unsafe state, data loss |
| **Speed** | Is this efficient enough for real-time? | Missed deadlines, poor UX, cascading delays |

**Decision Framework:**
1. Hard Rules gate what we build (reject anything that violates them)
2. Security gates how we expose it (reject unsafe designs)
3. Resilience shapes how we build it (fail-safe patterns)
4. Speed validates that it works (performance requirements)

---

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Core | Go | Single binary, no runtime, multi-decade stability |
| Database | SQLite | Embedded, zero maintenance, portable |
| Time-Series | InfluxDB | PHM data, energy monitoring, historical trends |
| Message Bus | MQTT (Mosquitto) | Simple, proven, debuggable |
| Wall Panel/Mobile | Flutter | Cross-platform native performance |
| Voice STT | Whisper | Local, accurate, open source |
| Voice TTS | Piper | Local, natural, offline |
| Local AI | Llama/Phi | On-device intelligence, no cloud |

---

## Build Order (5-Year Roadmap)

### Year 1: Foundation

**Goal:** Single-room lighting control in developer's home with manual override always working.

```
Infrastructure → Device Layer → Basic Automation → UI
```

**Milestones:**
1. **M1.1** — SQLite database, MQTT broker, InfluxDB running
2. **M1.2** — KNX bridge operational (read/write telegrams)
3. **M1.3** — Device registry with KNX actuators and switches
4. **M1.4** — REST API + WebSocket serving state
5. **M1.5** — Flutter wall panel controlling one room
6. **M1.6** — Basic scenes (saved device states)

**Success Criteria:**
- Physical switches work with Core off
- Wall panel controls lights with <200ms latency
- System survives server reboot (state persisted)
- Single integrator (developer) can commission new devices

**Security Requirements:**
- TLS on all network connections
- HTTP Basic auth placeholder (refined in Year 2)
- No remote access yet (local network only)

---

### Year 2: Automation Expansion

**Goal:** Whole-home lighting + blinds + climate with scenes and scheduling.

```
Multiple Rooms → Scenes → Modes → Scheduling → Protocols
```

**Milestones:**
1. **M2.1** — Area/Room hierarchy in data model
2. **M2.2** — Scene engine supporting multi-device, timed actions
3. **M2.3** — Mode system (Home/Away/Night/Holiday)
4. **M2.4** — Astronomical clock + scheduler
5. **M2.5** — DALI bridge for commercial lighting
6. **M2.6** — Blind control via KNX actuators
7. **M2.7** — Climate integration (Modbus → HVAC)
8. **M2.8** — Flutter mobile app

**Success Criteria:**
- 50+ devices across 10+ rooms
- "Good Night" scene executes in <1 second
- Modes affect automation behaviour site-wide
- Astronomical triggers work without internet
- System runs for 30 days without intervention

**Security Requirements:**
- JWT-based authentication
- Role-based access control (Admin/Facility Manager/User/Guest)
- Audit log for mode changes and scene triggers

---

### Year 3: Integration & Resilience

**Goal:** Audio/video distribution, security panel integration, CCTV, and System Supervisor foundation.

```
Audio Matrix → Video Matrix → Security Panel → CCTV → Supervisor Foundation
```

**Milestones:**
1. **M3.1** — Audio matrix integration (HTD/Russound)
2. **M3.2** — Video matrix integration (Atlona)
3. **M3.3** — Security panel integration (Texecom/Galaxy - monitoring only)
4. **M3.4** — CCTV integration (RTSP/ONVIF)
5. **M3.5** — BACnet bridge (commercial HVAC)
6. **M3.6** — Advanced conditional logic engine
7. **M3.7** — Health aggregation system (Supervisor Level 1)
8. **M3.8** — Machine-readable error catalog (see [Error Catalog Specification](../errors/catalog.md))

**Success Criteria:**
- Security events trigger automation (e.g., "Armed Away" → activate scene)
- Audio zones respond to presence detection
- Complex conditions work: "If motion in hallway AND mode=Night AND time=22:00-06:00 THEN pathway lights"
- CCTV streams accessible from wall panels
- Health endpoint returns unified status of all subsystems
- Error logs include machine-readable error IDs

**Security Requirements:**
- CCTV streams encrypted in transit
- Security panel is read-only (no remote arm/disarm in v1)
- Audit log for all security-related automation

---

### Year 4: Intelligence & Autonomous Recovery

**Goal:** Voice control, predictive health monitoring, local AI insights, and full System Supervisor with chaos testing.

```
Voice Pipeline → PHM Baselines → AI Engine → Learning → Supervisor → Chaos Testing
```

**Milestones:**
1. **M4.1** — Voice pipeline (Whisper STT → local NLU → Piper TTS)
2. **M4.2** — PHM baseline learning and anomaly detection
3. **M4.3** — Local AI engine for natural language commands
4. **M4.4** — Pattern learning for occupancy prediction
5. **M4.5** — Energy insights and recommendations
6. **M4.6** — System Supervisor with playbook-based recovery (see [Supervisor Specification](../architecture/supervisor.md))
7. **M4.7** — Simulation framework for chaos testing (see [Simulation Specification](simulation.md))
8. **M4.8** — Circuit breaker patterns for all external connections
9. **M4.9** — Failure Memory system (see [Failure Memory Specification](../architecture/failure-memory.md))
10. **M4.10** — Workflow Learning system (see [Workflow Learning Specification](../architecture/workflow-learning.md))

**Success Criteria:**
- Voice commands execute in <2 seconds
- Voice data never leaves site (local processing only)
- PHM detects failing HVAC fan 48hrs before failure
- AI provides actionable energy recommendations
- Supervisor automatically recovers from MQTT broker failure
- Chaos tests validate recovery under simulated failures
- AI agents can reference error catalog for automated diagnosis
- System learns from failures and avoids repeating mistakes
- Playbooks improve over time based on actual outcomes

**Security Requirements:**
- Voice data processed and discarded locally
- PHM alerts are warnings only (no automatic action)
- AI queries don't expose device state externally
- Supervisor actions logged in immutable audit trail
- Playbook changes require human approval

---

### Year 5: Commercial Readiness

**Goal:** Install in first customer's home with professional support.

```
Commissioning Tools → Backup/Restore → Remote Diagnostics → Documentation
```

**Milestones:**
1. **M5.1** — Web-based commissioning tool
2. **M5.2** — Configuration backup/restore
3. **M5.3** — Remote diagnostics (via WireGuard VPN)
4. **M5.4** — Installer documentation and training materials
5. **M5.5** — Customer handover pack generator
6. **M5.6** — Automated testing framework (integration tests)
7. **M5.7** — First customer deployment

**Success Criteria:**
- Non-developer can commission a system using docs
- Backup/restore completes in <5 minutes
- VPN remote access works from anywhere
- Handover pack enables new integrator to take over
- System passes 1000-hour soak test

**Security Requirements:**
- WireGuard VPN for remote access
- Multi-factor authentication for remote admin
- Encrypted backups
- Security audit by third party
- Penetration test passed

---

## Security-First Development

### Threat Model

**Attackers:**
1. **Script kiddies** — Opportunistic scanners looking for default passwords
2. **Malicious insiders** — Disgruntled staff or guests with temporary access
3. **Targeted attacks** — Someone specifically targeting a wealthy homeowner
4. **Supply chain** — Compromised dependencies or hardware

**Assets to Protect:**
1. Control of building systems (lights, climate, locks)
2. Surveillance footage and audio
3. Presence/occupancy data (when residents are home)
4. Network access (pivot point to other devices)

**Attack Vectors:**
1. Network ingress (exposed services)
2. Credential theft (weak passwords, phishing)
3. Insider abuse (overprivileged users)
4. Physical access (server, panels)
5. Supply chain compromise (malicious code in dependencies)

### Security Development Lifecycle

```
Requirements → Design Review → Implementation → Code Review → Security Test → Audit
     ↓              ↓                ↓              ↓               ↓            ↓
 Threat       Architecture     Secure Code    Peer Review    Pentest      Annual
 Model        Analysis         Patterns       Checklist      + Fuzzing    Review
```

**Every component must:**
1. Define its attack surface
2. Document trust boundaries
3. Implement input validation
4. Use least privilege
5. Log security-relevant events

**Security gates:**
- No PR merged without security checklist completion
- No release without penetration test (Year 5+)
- Annual security audit by external party (when deployed)

---

## Resilience Patterns

### Graceful Degradation

```
Full Function → Automation Disabled → Manual Control Only
     ↓                   ↓                      ↓
 Everything        Remote fails        Physical switches
  works         Scenes disabled          always work
               Automation disabled
```

**Degradation order (from principles.md):**
1. Remote features degrade first
2. Automation degrades second
3. Manual control never degrades

**Never degrade:**
- Physical switch response
- Life safety systems
- Frost protection
- Emergency lighting

### Failure Handling

**For every failure, decide:**
1. **Fail-safe state** — What is the safe fallback?
2. **User notification** — How do we alert the user?
3. **Recovery** — Can we auto-recover or require intervention?
4. **Logging** — What do we log for diagnostics?

**Examples:**

| Failure | Fail-Safe | Notify | Recovery |
|---------|-----------|--------|----------|
| MQTT broker down | Core buffers commands, retries | Dashboard warning | Auto-retry every 30s |
| KNX bridge crash | Physical switches work | Alert on panel | Auto-restart bridge |
| Database corruption | Read-only mode, use backup | Critical alert | Manual restore required |
| Internet down | Full local operation | Info message | Auto-resume when back |
| Power loss | Resume from saved state | Log event | Automatic on power restore |
| InfluxDB down | PHM disabled, Core continues | Warning | Auto-reconnect |

### Data Integrity

**State persistence:**
- Device state written to SQLite on every change
- Scene definitions in version-controlled YAML
- Configuration changes logged with timestamp + user
- Time-series data in InfluxDB (PHM, energy)

**Backup strategy:**
- Automatic daily backup to USB drive
- Weekly verification of backup integrity
- Restore tested quarterly

---

## Performance Requirements

### Latency Targets

| Operation | Target | Maximum | User Impact |
|-----------|--------|---------|-------------|
| Physical switch → actuator | N/A (bus-level) | N/A | Instantaneous |
| UI tap → command sent | 50ms | 100ms | Feels instant |
| Command → device activation | 100ms | 200ms | Acceptable delay |
| Scene recall (10 devices) | 200ms | 500ms | Noticeable but OK |
| Voice command → action | 1500ms | 2000ms | Expected delay |
| Page load (wall panel) | 200ms | 500ms | Smooth UX |

### Throughput Targets

| Metric | Target | Peak Handling |
|--------|--------|---------------|
| Devices supported | 500 | 1000 |
| Concurrent UI clients | 10 | 20 |
| Commands per second | 100 | 200 |
| State updates per second | 200 | 500 |
| MQTT messages per second | 500 | 1000 |

### Resource Limits

| Resource | Baseline | Under Load | Maximum |
|----------|----------|------------|---------|
| RAM | 30MB | 50MB | 100MB |
| CPU (idle) | <1% | 5% | 20% |
| Disk I/O | <1MB/s | 5MB/s | 10MB/s |
| Network | <100KB/s | 1MB/s | 5MB/s |

**Performance testing:**
- Load test with 100 devices, 10 clients before every release
- Soak test for 1000 hours (6 weeks) before v1.0
- Memory leak detection via continuous monitoring

---

## Quality Gates

### Code Merge Checklist

- [ ] Code compiles without warnings
- [ ] Unit tests pass (min 80% coverage for new code)
- [ ] Integration tests pass
- [ ] golangci-lint passes with no errors
- [ ] Security checklist completed (see [SECURITY-CHECKLIST.md](SECURITY-CHECKLIST.md))
- [ ] Documentation updated (inline comments + docs/)
- [ ] CHANGELOG.md updated
- [ ] Peer review approved

### Release Checklist

- [ ] All tests pass (unit + integration)
- [ ] Performance benchmarks meet targets
- [ ] Security checklist completed
- [ ] Penetration test passed (Year 5+)
- [ ] Documentation complete
- [ ] Backup/restore tested
- [ ] Upgrade path tested (preserves data)
- [ ] Rollback plan documented
- [ ] Release notes written
- [ ] Version tagged in git

---

## Monitoring & Observability

### Logging Levels

| Level | Use Case | Example |
|-------|----------|---------|
| **DEBUG** | Development only | "Received MQTT message: topic=graylogic/state/knx/1.2.3 payload={...}" |
| **INFO** | Normal operations | "KNX Bridge connected", "Scene 'Good Night' activated" |
| **WARN** | Recoverable issues | "MQTT reconnecting", "Device 1.2.3 not responding (retry 1/3)" |
| **ERROR** | Failures requiring attention | "Database write failed", "Bridge crashed" |
| **FATAL** | System cannot continue | "Database corrupted", "Critical config missing" |

**Log outputs:**
- Console (systemd journal)
- File (`/var/log/graylogic/core.log`, rotated daily, 7-day retention)
- Syslog (optional, for centralized logging)

### Metrics

**Collected via Prometheus exporter:**
- Command latency (p50, p95, p99)
- Device state update rate
- MQTT message rate
- API request rate and errors
- Memory usage
- Goroutine count
- Database query time

**Alerting thresholds:**
- Command latency p95 > 500ms
- Error rate > 1%
- Memory usage > 80MB
- Bridge disconnected > 60s

### Health Checks

**`/health` endpoint returns:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-15T12:00:00Z",
  "uptime_seconds": 86400,
  "components": {
    "database": "healthy",
    "mqtt": "healthy",
    "influxdb": "healthy",
    "knx_bridge": "healthy",
    "dali_bridge": "degraded"
  },
  "metrics": {
    "devices_total": 127,
    "devices_online": 125,
    "commands_last_minute": 23,
    "memory_mb": 45
  }
}
```

---

## Milestone Acceptance Criteria

### M1.1: Infrastructure Running

**Deliverables:**
- SQLite database schema created and migrated
- MQTT broker (Mosquitto) running
- InfluxDB running with graylogic bucket
- Core binary compiles and starts

**Acceptance:**
- `sqlite3 /var/lib/graylogic/core.db ".tables"` shows schema
- `mosquitto_sub -t '#' -v` receives test messages
- InfluxDB health check passes
- Core logs "Server started on :8080"

---

### M1.2: KNX Bridge Operational

**Deliverables:**
- KNX bridge connects to knxd
- Receives KNX telegrams
- Sends KNX telegrams
- Publishes state changes to MQTT

**Acceptance:**
- Physical switch triggers telegram visible in logs
- `mosquitto_sub -t 'graylogic/state/knx/#'` shows state updates
- Core can send command → light turns on within 200ms

---

### M1.3: Device Registry

**Deliverables:**
- Database schema for devices, capabilities, addresses
- API to add/remove/list devices
- KNX devices can be registered with group addresses

**Acceptance:**
- `curl http://localhost:8080/api/devices` returns JSON list
- Device added via API appears in database
- Device state updates flow to API clients via WebSocket

---

### M1.4: REST API + WebSocket

**Deliverables:**
- REST API for device control
- WebSocket for real-time state updates
- TLS enabled (self-signed cert acceptable for M1)

**Acceptance:**
- `curl -k https://localhost:8080/api/devices` returns devices (or 8443 if TLS-only)
- WebSocket client receives state updates in <100ms
- Load test: 10 concurrent clients, 100 commands/s

---

### M1.5: Flutter Wall Panel

**Deliverables:**
- Flutter app running on Android/Linux
- Connects to Core API
- Displays room devices
- Sends control commands

**Acceptance:**
- App launches in <2 seconds
- Device list loads in <500ms
- Tap → light response in <200ms (UI to device activation)
- UI updates automatically on device state change

---

### M1.6: Basic Scenes

**Deliverables:**
- Scene definition (YAML or database)
- Scene engine that recalls device states
- API to activate scenes

**Acceptance:**
- Scene defined with 5 devices, different states
- `POST /api/scenes/{id}/activate` triggers scene
- All devices reach target state within 500ms
- Scene activation survives Core restart

---

## Related Documents

- [Principles](../overview/principles.md) — Hard rules governing all development
- [Core Internals](../architecture/core-internals.md) — Technical architecture of the Core
- [Security Model](../architecture/security-model.md) — Authentication and authorization
- [Coding Standards](CODING-STANDARDS.md) — How to write and document code
- [Security Checklist](SECURITY-CHECKLIST.md) — Security verification for components and releases
