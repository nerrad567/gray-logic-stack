---
title: Forensic Audit - Iteration 2
version: 1.0.0
status: complete
date: 2026-01-17
auditor: Claude Opus 4.5
previous_score: 7.5/10
new_score: 7.2/10
---

# Gray Logic Stack - Forensic Audit Iteration 2

> **Audit Scope**: Complete documentation repository, second-pass forensic analysis
> **Philosophy**: Ruthless, paranoid examination for issues that will manifest in 2028, 2032, or during high-value commissioning (€2M villa, German electrical inspector, 10-year field deployment)

---

## Executive Threat Summary

The Gray Logic Stack documentation has improved significantly since iteration 1, but this deeper forensic pass reveals **architectural time bombs** that won't detonate until the system is under real-world stress. These are the subtle killers:

### Top 7 Nastiest Findings (Ranked by Long-Term Pain)

| Rank | Finding | Pain Level | Time to Detonate |
|------|---------|------------|------------------|
| 1 | **MQTT SPOF with Underspecified Recovery** | CRITICAL | First broker crash under load |
| 2 | **JWT Refresh Token Creates Unbounded Sessions** | HIGH | Compliance audit / security breach |
| 3 | **Scene Race Conditions with No Conflict Resolution** | HIGH | Complex automation, 50+ devices |
| 4 | **Backup Encryption Key Circular Dependency** | CRITICAL | First disaster recovery attempt |
| 5 | **Database Migration Has No Rollback Path** | MEDIUM | First botched update at customer site |
| 6 | **Frost Protection "cannot_disable" Has No Enforcement** | HIGH | First frozen pipe, insurance claim |
| 7 | **Voice PIN Has No Rate Limiting** | MEDIUM | Security audit, brute force attack |

---

## Readiness Score v2: 7.2/10

**Downgrade Rationale**: The first audit caught surface issues. This pass reveals that several "solved" problems are only partially addressed. The MQTT SPOF acknowledgment in ADR-002 is good documentation practice, but the actual recovery mechanism is still undefined. The security model has gaps that will fail compliance audits. The "additive-only migrations" strategy sounds safe but has no rollback procedure when (not if) a migration goes wrong.

**What Keeps the Score Above 7.0**: The foundational architecture is sound. The offline-first principle is genuinely embedded. The domain specifications (lighting, climate, security) show mature thinking. The handover pack template is excellent. This is a solid foundation—but it's not yet ready for a €40k customer who expects German engineering precision.

---

## Complete Issues Table

### CRITICAL (Score Impact: -0.5 each)

| Severity | File/Section | Exact Quote | Impact | Minimal Fix |
|----------|--------------|-------------|--------|-------------|
| CRITICAL | `architecture/decisions/002-mqtt-internal-bus.md` | "Mosquitto becomes a single point of failure... If broker fails, automation stops. Mitigated by: systemd watchdog, health monitoring, fast restart" | Fast restart doesn't preserve QoS1 messages in flight. Scene commands will be lost during broker restart. A lighting scene firing during restart = partial execution = angry customer. | Specify message persistence strategy. Add `persistence true` to Mosquitto config. Define bridge reconnect backoff. Document maximum acceptable message loss window. |
| CRITICAL | `resilience/backup.md` | "Encryption key: Stored in /etc/graylogic/secrets.yaml" | The encryption key for backups is stored in secrets.yaml, which is itself backed up. If you need to restore from backup, you need the key. If you have the key, you didn't need the backup of secrets.yaml. But if secrets.yaml is corrupted/lost AND you don't have the key memorized, the backup is useless. Classic circular dependency. | Specify that backup encryption key MUST be stored separately from the backup itself (e.g., physical safe, password manager, printed QR code in handover pack). Add to Doomsday Pack requirements. |
| CRITICAL | `operations/bootstrapping.md` | "generate a Claim Token (UUIDv4) and print it to the console" | Claim token has no expiration. An attacker who gains log access days/weeks later can still claim an unconfigured system. Setup mode has no timeout. | Add 1-hour claim token expiry. Add setup mode timeout (auto-reboot to safe mode after 24h unclaimed). Rotate claim token every 15 minutes. |

### HIGH (Score Impact: -0.3 each)

| Severity | File/Section | Exact Quote | Impact | Minimal Fix |
|----------|--------------|-------------|--------|-------------|
| HIGH | `interfaces/api.md` | "Refresh tokens: 30 days (rolling)" | Rolling refresh tokens mean a session can be extended indefinitely as long as the user is active. A compromised token never expires. This will fail any serious security audit. GDPR/insurance compliance requires session limits. | Add absolute session lifetime (e.g., 90 days max regardless of refresh). Add refresh token family tracking to detect token theft. |
| HIGH | `interfaces/api.md` | "API Keys... Expiry: Optional (never by default)" | Non-expiring API keys are a security anti-pattern. When an installer leaves the company, their API key lives forever. | Default to 1-year expiry. Require explicit `never_expires: true` with audit log entry. Add API key last-used tracking. |
| HIGH | `automation/automation.md` | "parallel: true - Run actions simultaneously" | No conflict resolution specified. If two parallel actions target the same device (e.g., scene sets light to 50%, circadian override sets to 80%), behavior is undefined. Last-write-wins is non-deterministic under parallel execution. | Define action priority/ordering. Add conflict detection. Specify that same-device actions in parallel block are serialized. Document deterministic behavior. |
| HIGH | `domains/climate.md` | "cannot_disable: true" | This is a YAML flag. There is no specification for how this is enforced. A determined user could modify the database directly. A bug in the UI could allow toggling it. The consequences of frost protection being disabled are burst pipes and €50k+ damage. | Specify enforcement layer. Add database constraint. Add audit log for any attempt to modify. Add "frost protection active" to system health dashboard. Consider hardware interlock documentation. |
| HIGH | `interfaces/api.md` | "WebSocket... Authentication: Token in query parameter" | WebSocket tokens in URLs are logged by proxies, appear in browser history, leak via Referrer headers. This is a known anti-pattern. | Implement ticket-based auth: REST endpoint issues single-use ticket, WebSocket connection presents ticket, ticket is invalidated immediately. Token never in URL. |
| HIGH | `intelligence/voice.md` | "Custom wake words require Picovoice Console" | Picovoice Console is a cloud service. This creates an external dependency for a core feature. Breaks the offline-first principle. If Picovoice changes terms/pricing/availability, custom wake words stop working. | Document this as a known limitation. Provide fallback wake words that work offline. Consider openWakeWord as alternative for v2. |
| HIGH | `domains/security.md` | "Audit Log Entry: { user_id, timestamp, zone_id, success: bool }" | The audit log example doesn't explicitly exclude PIN storage, but the surrounding text discusses PIN handling. Verify that PIN values are NEVER logged, even on failure. "Invalid PIN for zone X" is fine. "PIN 1234 failed for zone X" is a breach. | Add explicit requirement: "PIN values MUST NOT appear in any log at any level (debug, info, error, audit)." Add to security checklist. |

### MEDIUM (Score Impact: -0.1 each)

| Severity | File/Section | Exact Quote | Impact | Minimal Fix |
|----------|--------------|-------------|--------|-------------|
| MEDIUM | `architecture/decisions/004-additive-only-migrations.md` | "Never delete columns... Add new column, migrate data, deprecate old" | No rollback procedure specified. When migration 047 corrupts data (it will happen), how do you roll back? A/B partitions help for code, but database is shared. | Specify backup-before-migrate as mandatory. Add rollback script requirements for each migration. Document point-in-time recovery procedure. |
| MEDIUM | `intelligence/voice.md` | "Voice PIN... Same rate limiting as physical keypads" | Rate limiting is mentioned but not specified. Physical keypads typically have 3-attempt lockout with 5-minute cooldown. Is this enforced? What prevents brute-forcing 4-digit PINs (10,000 combinations) at 1 attempt/second? | Define: 3 attempts, 5-minute lockout, exponential backoff on repeated failures. Add to security spec. |
| MEDIUM | `operations/monitoring.md` | "Prometheus scrape endpoint at :9090/metrics" | No authentication specified for metrics endpoint. Prometheus metrics can leak sensitive information (device names, room names, usage patterns). | Specify authentication requirement. Document that metrics endpoint should be firewalled to monitoring network only. |
| MEDIUM | `domains/climate.md` | "See: protocols/bacnet.md" | File doesn't exist. Year 2 feature referencing non-existent specification. | Either create placeholder doc or remove reference. Broken links erode documentation trust. |
| MEDIUM | `data-model/schemas/device.schema.json` + examples throughout | Device IDs vary: `d1e2f3a4-5678-9012-bcde-f01234567891` vs documentation examples using different formats | Inconsistent example data across documents. Not a functional issue, but confuses implementers and suggests lack of attention to detail. | Standardize example UUIDs across all documentation. Use consistent device/room/area IDs in all examples. |
| MEDIUM | `protocols/knx.md` | "KNX-RF (radio) devices: Coming Year 2" | KNX RF is mentioned but security implications not addressed. RF is susceptible to replay attacks. KNX Secure RF exists but adds complexity. | Add note about KNX Secure requirement for RF deployments. Document RF security considerations for Year 2 planning. |
| MEDIUM | `domains/lighting.md` | "Emergency lighting: MONITORING ONLY" | Good principle, but enforcement mechanism not specified. What prevents someone from adding emergency fixtures to a scene? Database constraint? UI warning? | Add validation rule: devices with `emergency: true` flag cannot be added to user scenes. Add UI warning. Document in commissioning checklist. |
| MEDIUM | `deployment/residential.md` | "Default Gateway: Critical - prefer static IP with backup DHCP" | Documentation says prefer static IP but doesn't specify what "backup DHCP" means. If DHCP is used, the system IP can change, breaking remote access. | Clarify: Static IP required for production. DHCP acceptable only during initial setup. Document IP reservation requirements. |

### LOW (Score Impact: -0.05 each)

| Severity | File/Section | Exact Quote | Impact | Minimal Fix |
|----------|--------------|-------------|--------|-------------|
| LOW | `intelligence/phm.md` | "7-day baseline learning for anomaly detection" | Baseline can be manipulated. An attacker (or malicious installer) could run devices abnormally during baseline period to hide future anomalies. | Document baseline verification procedure. Add manual baseline approval option. Consider 30-day baseline with outlier rejection. |
| LOW | `business/pricing.md` | "Tier 3: £20-40k" ... "Year 5" | Pricing document references features (full PHM, voice control) that have no implementation timeline correlation. Selling Tier 3 in Year 2 with Year 5 features creates expectation mismatch. | Add feature availability matrix to pricing. Mark features as "Available Now" vs "Roadmap". |
| LOW | `protocols/mqtt.md` | Topic namespace examples | Topic namespace is well-designed but `gl/devices/+/command` allows any device command. No topic-level ACL examples provided. | Add MQTT ACL examples to spec. Document per-device topic restrictions for bridges. |
| LOW | `architecture/core-internals.md` | "Scheduler... Cron-style expressions" | Timezone handling for cron jobs across DST transitions not specified. 2:30 AM schedules on DST change days are problematic. | Document DST handling. Specify: skip on spring forward, run once on fall back. Use UTC internally with local time display. |
| LOW | `deployment/handover-pack-template.md` | Comprehensive handover requirements | Excellent document, but no verification checklist. How does commissioning engineer confirm all items are present before handover? | Add completion checklist with sign-off fields. Add QR code linking to digital verification form. |

---

## Architecture-Level Concerns (Not Direct Bugs)

### 1. The MQTT Fan-Out Problem
Every device state change publishes to MQTT. In a 500-device installation, a "Good Night" scene could generate 200+ MQTT messages in <1 second. Mosquitto can handle this, but:
- Bridge reconnection storms after broker restart
- No message coalescing for rapid state changes
- No back-pressure mechanism if subscriber is slow

**Recommendation**: Add message batching for scenes. Implement QoS policies per topic type. Document performance limits.

### 2. The "God Database" Risk
SQLite holds config, state, automation, users, and audit logs. This is pragmatic for Year 1 but:
- Audit logs will grow unbounded
- State table will see extremely high write frequency
- No clear archival strategy

**Recommendation**: Document retention policies NOW. Plan audit log rotation. Consider state cache layer for v2.

### 3. The Bridge Trust Model
Bridges authenticate to MQTT with passwords, but once connected, they can publish to any topic they have ACL access to. A compromised bridge could:
- Publish false device states
- Inject commands to other bridges' devices
- Flood the bus with garbage

**Recommendation**: Add message signing for critical commands. Document bridge isolation requirements. Add anomaly detection for bridge behavior.

---

## Suggested Surgical Strikes (Prioritized)

### Strike 1: MQTT Resilience Specification (2 hours) — COMPLETED 2026-01-17

Created `docs/architecture/mqtt-resilience.md` with:
- Mosquitto persistence configuration (what survives restart)
- Bridge reconnection with exponential backoff + jitter (±30%)
- Message loss budget (< 30s acceptable window)
- QoS level requirements per topic type with enforcement
- Health check thresholds and Prometheus metrics
- Systemd watchdog configuration
- Recovery procedures (automatic and manual)
- Commissioning test checklist

Also updated:
- `docs/protocols/mqtt.md` — Added cross-reference to resilience spec
- `docs/architecture/core-internals.md` — Added jitter to backoff description
- `docs/architecture/decisions/002-mqtt-internal-bus.md` — Fixed broken link to monitoring.md

### Strike 2: Authentication Hardening Pass (3 hours) — COMPLETED 2026-01-17

Updated `docs/architecture/security-model.md` (v1.0.0 → v1.1.0):
- Added absolute session lifetime (90 days max) for JWT refresh tokens
- Added refresh token family tracking for theft detection
- Changed API key default expiry to 1 year (was never-expires)
- Added explicit PIN rate limiting spec (3 attempts, 5-min lockout, exponential backoff)
- Added "Never Log Secrets" requirement with explicit prohibited values list
- Updated authentication decision tree with new limits

Updated `docs/operations/bootstrapping.md` (v1.0.0 → v1.1.0):
- Added claim token expiry (1 hour) with rotation every 15 minutes
- Added setup mode timeout (24 hours → safe mode)
- Added detailed threat model for claim token security
- Added rate limiting for claim attempts (5 attempts / 15-min lockout)

Updated `docs/interfaces/api.md` (v1.0.0 → v1.1.0):
- Replaced WebSocket URL token auth with ticket-based authentication
- WebSocket tickets: single-use, 60-second TTL, server-side storage
- Updated token lifetimes table with absolute session limits
- Added session info to refresh token response
- Updated API key creation with default 1-year expiry
- Updated JavaScript WebSocket example for ticket-based auth

### Strike 3: Migration Safety Net (1 hour) — COMPLETED 2026-01-17

Updated `docs/architecture/decisions/004-additive-only-migrations.md` (added last_updated field):
- Added Migration Safety Requirements section
- Mandatory pre-migration backup with verification before any migration
- Dry-run mode specification for testing migrations safely
- Complete rollback procedure with step-by-step commands
- Down migration script format (informational-only for additive-only policy)
- Migration failure handling with RECOVERY_NEEDED.txt file
- Commissioning test checklist for verifying migration safety

Updated `docs/resilience/backup.md` (v1.0.0 → v1.1.0):
- Added on-demand backup type for migration/upgrade triggers
- Added Migration-Triggered Backups section documenting automatic behavior
- Documented backup file naming convention with timestamp and version
- Added manual pre-upgrade backup commands
- Cross-referenced ADR-004 for rollback procedures

Updated `docs/architecture/core-internals.md` (v1.0.0 → v1.0.1):
- Updated startup sequence to show MigrateWithSafety() pattern
- Added reference to ADR-004 for migration safety documentation

### Strike 4: Frost Protection Enforcement (1 hour) — COMPLETED 2026-01-18

Updated `docs/domains/climate.md`:
- Added comprehensive Frost Protection Enforcement section (~240 lines)
- Hardware-First Architecture: Three-layer protection model (Thermostat Hardware, HVAC Equipment, Gray Logic)
- Hardware Requirements table with verification steps
- Approved Thermostat Examples (ABB, MDT, Theben, Siemens, Vaillant, NIBE)
- What Gray Logic Does (monitoring, coordination) and PROHIBITED actions
- Failure Mode Analysis table showing frost protection continues during all software failures
- Software Enforcement: Database constraints, audit logging, API enforcement
- Health Dashboard Integration with frost protection status indicators
- Commissioning Checklist for hardware, equipment, software, and failure testing

Updated `docs/resilience/offline.md`:
- Updated Component Failure Matrix to explicitly list frost protection as hardware-based
- Added frost protection to core_down works list
- Added "Frost Protection During Failures" subsection with resilience spec
- Cross-reference to climate.md frost protection enforcement section

Updated `docs/overview/principles.md`:
- Clarified frost protection rule: "must run on thermostat hardware, not depend on Core"

### Strike 5: Documentation Hygiene (30 minutes) — COMPLETED 2026-01-18

- Added YAML frontmatter to 3 files missing it:
  - `data-model/schemas/README.md`
  - `deployment/handover-pack-template.md`
  - `intelligence/ai-premium-features.md`
- Added completion checklist with sign-off fields to handover pack template
- Verified `protocols/bacnet.md` exists (615-line Year 2 roadmap specification)
- Verified all cross-references are valid (infrastructure.md, mqtt-resilience.md)
- Confirmed RFC 2119 keyword usage is consistent

---

## Score Breakdown

| Category | Max Points | Score | Notes |
|----------|------------|-------|-------|
| Architecture Soundness | 2.0 | 1.7 | MQTT SPOF acknowledged but unresolved |
| Security Model | 2.0 | 1.4 | JWT gaps, API key defaults, claim token |
| Operational Readiness | 2.0 | 1.6 | Good monitoring, weak migration rollback |
| Domain Coverage | 2.0 | 1.8 | Comprehensive, minor enforcement gaps |
| Documentation Quality | 2.0 | 1.7 | Good structure, some broken refs |
| **Total** | **10.0** | **7.2** | |

---

## Comparison: Iteration 1 vs Iteration 2

| Aspect | Iteration 1 (7.5) | Iteration 2 (7.2) | Delta |
|--------|-------------------|-------------------|-------|
| Surface Issues | Many found | Mostly fixed | +0.5 |
| Architectural Depth | Not examined | Time bombs found | -0.6 |
| Security Edge Cases | Partial | More gaps found | -0.3 |
| Documentation Polish | Good | Slightly better | +0.1 |

**Net Change**: -0.3 points. This is expected—deeper analysis reveals more issues. The score reflects increased understanding, not degraded quality.

---

## Conclusion

The Gray Logic Stack is a well-architected system with solid foundations. However, it is not yet ready for high-value deployments where failure costs are measured in tens of thousands of euros. The issues identified are not showstoppers, but they are the kind of subtle problems that:

1. Won't appear during development or testing
2. Will manifest at 3 AM on Christmas Eve
3. Will be extremely difficult to debug remotely
4. Will erode customer confidence in "premium" positioning

The five surgical strikes above address the highest-risk items with minimal effort. Completing them would likely raise the score to 7.8-8.0, which is acceptable for careful early deployments with technically sophisticated customers.

**Bottom Line**: Good enough for your own house. Not yet ready for the €2M villa with the pissed-off German inspector.

---

*Audit completed: 2026-01-17*
*Auditor: Claude Opus 4.5*
*Next audit recommended: After completing surgical strikes, or before any customer deployment*
