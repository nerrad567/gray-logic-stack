---
title: Audit Iteration 3 Log
version: 1.0.0
status: active
audit_date: 2026-01-18
auditor: Gemini CLI (Antigravity)
scope: Full repository forensic re-audit
previous_audit: audit-iteration-2-log.md
---

# Audit Iteration 3 Log — Gray Logic Stack

**Audit Date:** 2026-01-18  
**Auditor:** Gemini CLI (Antigravity)  
**Scope:** Third-layer forensic documentation audit  
**Mission:** Identify subtle, long-term problems that could cause significant pain in 2030–2036 deployments

---

## Executive Threat Summary

### Context

Iteration 2 successfully closed five surgical strikes: MQTT Resilience, Authentication Hardening, Migration Safety Net, Frost Protection Enforcement, and Documentation Hygiene. The repository is now significantly more robust. This third-layer audit digs deeper, looking for:

- Edge cases in newly-added content
- Subtle inconsistencies between related specifications
- Over-specification that creates unrealistic implementation expectations
- Under-specification in critical failure modes
- Gaps in commissioning verification

### Top 7 Nastiest Findings (Ranked by Long-Term Pain)

| Rank | Finding | Pain Level | Time to Detonate |
|------|---------|------------|------------------|
| 1 | **Scene Conflict Resolution Still Absent** | HIGH | First complex multi-scene home, race condition causes flickering |
| 2 | **Timestamp-Based State Reconciliation Trusts Untrusted Clocks** | HIGH | First bridge with clock drift poisons authoritative state |
| 3 | **PHM Learning Period Creates Critical Equipment Blind Spot** | MEDIUM | Device fails during 7-30 day commissioning window |
| 4 | **Voice-to-PIN Authentication Flow Undocumented** | MEDIUM | Security audit, voice-based alarm arming exploited |
| 5 | **Schedule Catch-Up Lacks Context Awareness** | MEDIUM | Embarrassing automation replay after brief Core restart |
| 6 | **Frost Protection Hardware Verification Has No Commissioning Gate** | MEDIUM | Thermostat without frost mode installed, pipe freezes |
| 7 | **API Key Self-Rotation Pattern Missing** | LOW | Integration key expires, automated system can't renew |

### Readiness Score v3

| Category | Score | Notes |
|----------|-------|-------|
| **Core Architecture** | 9/10 | Solid, well-documented, offline-first proven |
| **Security Model** | 8/10 | Good iteration, minor voice-PIN gap |
| **Resilience** | 8/10 | MQTT resilience addressed, timestamp trust issue remains |
| **Automation** | 7/10 | Scene tracking added, conflict resolution needed |
| **Commissioning** | 6/10 | Verification gaps for critical hardware requirements |
| **PHM** | 5/10 | Over-specified, creates expectation gap |
| **Documentation Quality** | 8/10 | Excellent overall, minor inconsistencies |

**Overall Readiness: 7.5/10** *(Up from 7.2 in Iteration 2)*

The improvements are real and meaningful. Remaining issues are subtler and lower-priority than Iteration 2 findings, but still warrant attention before v1.0 release.

---

## Detailed Issues Table

### Critical Issues (0)

*None identified. Iteration 2 successfully addressed the critical issues.*

---

### High Severity Issues (2)

#### H1: Scene Conflict Resolution Still Absent

| Attribute | Value |
|-----------|-------|
| **File** | `docs/automation/automation.md` |
| **Section** | Scene Execution Tracking (lines 116-173) |
| **Quote** | Scene execution tracking exists but no mechanism to prevent or resolve conflicts |
| **Why It Hurts** | When Cinema Mode and Bright Light are triggered within milliseconds (e.g., user double-taps, voice command races with schedule), both execute. Device states flicker between conflicting targets. Customer perception: "the system is broken." |
| **Suggested Fix** | Add scene conflict rules: (1) Scene mutex per device, (2) Last-write-wins with minimum stabilization window (e.g., 500ms), (3) Explicit scene priority for conflicts |

**Evidence:**

`automation.md` defines scene execution with parallel actions but no conflict handling:

```yaml
# Line 294-309: Parallel execution defined
- device_id: "blind-main"      # Runs simultaneously with lights
  command: "position"
  parameters: { position: 0 }
  parallel: true
```

No equivalent for "what if another scene also targets this device?"

---

#### H2: Timestamp-Based State Reconciliation Trusts Untrusted Clocks

| Attribute | Value |
|-----------|-------|
| **File** | `docs/resilience/offline.md` |
| **Section** | Timestamp-Based Conflict Resolution (lines 373-420) |
| **Quote** | `timestamp_source: "bridge"  # Bridge timestamps state at observation time` |
| **Why It Hurts** | Bridges are edge devices (Raspberry Pi, embedded systems). If a bridge's clock drifts (no NTP, RTC battery dead), it could report timestamps hours in the future or past. Core accepts these as authoritative. A future timestamp would permanently "win" conflicts, preventing legitimate state updates. |
| **Suggested Fix** | (1) Require NTP for all bridges (document as commissioning requirement), (2) Add clock skew detection in Core (reject timestamps >60s from Core clock), (3) Log warning when bridge clock drift detected |

**Evidence:**

`offline.md` line 398:
```yaml
timestamp_source: "bridge"      # Bridge timestamps state at observation time
```

No clock validation. No NTP requirement documented.

---

### Medium Severity Issues (5)

#### M1: PHM Learning Period Creates Critical Equipment Blind Spot

| Attribute | Value |
|-----------|-------|
| **File** | `docs/intelligence/phm.md` |
| **Section** | Baseline Learning Status (lines 332-367) |
| **Quote** | `during_learning: extreme_only: true, threshold_multiplier: 3` |
| **Why It Hurts** | Critical plant equipment (pumps, boilers) could have developing faults during the 7-30 day learning period. A pump bearing with baseline vibration of 2.1 mm/s could be at 4.0 mm/s and climbing—but PHM won't flag it because "3x baseline" hasn't been calculated yet. By the time baseline is ready, bearing is failed. |
| **Suggested Fix** | (1) Document absolute external limits (e.g., ISO 10816 for vibration) that apply DURING learning, (2) Add "known-good reference baseline" option for common equipment types, (3) Require manufacturer baseline for critical equipment at commissioning |

---

#### M2: Voice-to-PIN Authentication Flow Undocumented

| Attribute | Value |
|-----------|-------|
| **File** | `docs/intelligence/voice.md`, `docs/architecture/security-model.md` |
| **Section** | N/A - Missing |
| **Quote** | Security model documents PIN rate limiting; Voice docs don't mention PIN flow |
| **Why It Hurts** | User says "Hey Gray, arm the alarm, code one two three four." Does this: (A) rate-limit like keypad PIN entry? (B) log the PIN audio? (C) require confirmation? If not rate-limited, voice becomes a brute-force vector. If audio is logged, PINs are in logs. |
| **Suggested Fix** | Add explicit "Voice Authentication" section to `voice.md` covering: (1) Voice PIN never logged (even as transcript), (2) Rate limiting applies to voice commands containing PIN intent, (3) Confirmation required: "Arm away? Say your code." |

---

#### M3: Schedule Catch-Up Lacks Context Awareness

| Attribute | Value |
|-----------|-------|
| **File** | `docs/resilience/offline.md` |
| **Section** | Schedule Handling (lines 643-668) |
| **Quote** | `core_restart: behavior: "Catch up missed schedules on restart", max_catch_up_minutes: 30` |
| **Why It Hurts** | Core was down for 20 minutes. "Good Morning" schedule fired 15 minutes ago (within catch-up window). On restart, Core replays "Good Morning" scene—opening blinds, turning on coffee machine—but family already left for work. Office use case: "Presentation Mode" scheduled for 9am meeting replays at 9:25 after brief restart, interrupting ongoing meeting. |
| **Suggested Fix** | Add catch-up conditions: (1) Check current mode (if `away`, skip comfort scenes), (2) Check presence (if room vacant, skip room-specific scenes), (3) Add `catch_up_condition` field to schedules (configurable per schedule) |

---

#### M4: Frost Protection Hardware Verification Has No Commissioning Gate

| Attribute | Value |
|-----------|-------|
| **File** | `docs/domains/climate.md` |
| **Section** | Hardware Requirements (lines 794-800) |
| **Quote** | `Built-in frost mode: Activates heating below threshold... Check device spec sheet` (Verification column) |
| **Why It Hurts** | The requirement exists but there's no enforcement. An installer could commission any thermostat and never verify it has hardware frost protection. Years later, during a Christmas vacation with Gray Logic down, pipes freeze. Insurance claim, liability question: "Did the integrator verify hardware frost protection?" |
| **Suggested Fix** | (1) Add frost protection to commissioning checklist as MANDATORY verification step, (2) Require signed acknowledgment: "I have verified thermostat model [X] has built-in frost protection at [Y]°C threshold", (3) Document "approved thermostat list" with confirmed frost protection |

---

#### M5: API Key Self-Rotation Pattern Missing

| Attribute | Value |
|-----------|-------|
| **File** | `docs/interfaces/api.md` |
| **Section** | API Keys (lines 222-274) |
| **Quote** | API key defaults to 1-year expiry, rotation recommended, but no self-rotation mechanism |
| **Why It Hurts** | Home Assistant integration uses API key (no user session). Key expires in 11 months. How does the integration rotate its own key? It can't create a new key without `users:manage` permission. Admin must manually rotate, but admin forgot. Integration silently fails. |
| **Suggested Fix** | Add "Service Account" pattern: (1) Document dedicated service account role with `apikeys:rotate_self` permission, (2) API endpoint `POST /api/v1/auth/apikeys/{id}/rotate` returns new key, invalidates old after grace period, (3) Document rotation reminder notifications (30/7/1 days) |

---

### Low Severity Issues (8)

#### L1: Security Domain PIN Transmission Plaintext on MQTT

| Attribute | Value |
|-----------|-------|
| **File** | `docs/domains/security.md` |
| **Section** | Commands (lines 279-327) |
| **Quote** | `user_pin: "1234"  # Required for remote arm/disarm` |
| **Why It Hurts** | While MQTT is local-only, PINs transit in plaintext. A compromised device on the Control VLAN could sniff MQTT and capture alarm PINs. Low probability but high impact if exploited. |
| **Suggested Fix** | Document: (1) MQTT TLS is REQUIRED for security bridge communication, OR (2) PIN is hashed/encrypted before MQTT transmission (bridge validates hash, not plaintext) |

---

#### L2: PHM Sensor Health Layer Missing

| Attribute | Value |
|-----------|-------|
| **File** | `docs/intelligence/phm.md` |
| **Section** | External Monitoring via Device Associations (lines 118-175) |
| **Quote** | CT clamp reports pump power, but no monitoring of CT clamp health |
| **Why It Hurts** | CT clamp fails open (reads 0). PHM thinks pump stopped. Or CT clamp drifts +20%. PHM thinks pump power elevated, triggers false alert. No "sensor reasonableness" checks. |
| **Suggested Fix** | Add sensor validation layer: (1) Cross-reference multiple sensors where available, (2) Flag readings that are physically implausible (e.g., pump running but power=0), (3) Document "monitoring device health" as PHM extension |

---

#### L3: Bootstrapping Claim Token Log Scrubbing Incomplete

| Attribute | Value |
|-----------|-------|
| **File** | `docs/operations/bootstrapping.md` |
| **Section** | Claim Token Security (lines 220-227) |
| **Quote** | `Post-Claim Cleanup: After successful claim, the token is scrubbed from logs where possible (e.g., journald --vacuum)` |
| **Why It Hurts** | `journald --vacuum` removes OLD logs (by size/time), not SPECIFIC entries. Claim token remains in recent logs until natural rotation. During this window, log access = claim token access. |
| **Suggested Fix** | (1) Change to: "Claim token is displayed ONLY to stdout, never written to journal", (2) Use ephemeral display (e.g., print, then print blank lines to scroll out of view), (3) Or accept the risk with documented mitigation: "Secure log access during setup" |

---

#### L4: KNX Bridge Command Timeout Assumes Success

| Attribute | Value |
|-----------|-------|
| **File** | `docs/protocols/knx.md` |
| **Section** | Error Handling (lines 449-455) |
| **Quote** | `Telegram timeout: Log warning, assume command delivered` |
| **Why It Hurts** | Scene execution tracking shows "completed" but device never responded. User sees "Cinema Mode activated" but blinds didn't close. Inconsistent with "Explicit over Implicit" principle. |
| **Suggested Fix** | (1) Add "unconfirmed" device state for devices without status feedback GA, (2) Scene execution should show "partial (unconfirmed)" when some devices lack acknowledgment, (3) Document which device types REQUIRE status feedback for reliable automation |

---

#### L5: WebSocket Ticket 60-Second Window Too Tight

| Attribute | Value |
|-----------|-------|
| **File** | `docs/interfaces/api.md` |
| **Section** | Token Lifetimes (lines 206-219) |
| **Quote** | `WebSocket Ticket: 60 seconds, single-use` |
| **Why It Hurts** | Slow mobile network. Fetch ticket (2s network), UI transition (1s), connect WebSocket (2s)... usually fine. But add network wobble (10s retry), cold start (15s), and 60s expires. User sees cryptic connection error. |
| **Suggested Fix** | (1) Extend to 120 seconds, OR (2) Document client retry behavior: "On ticket expiry, silently fetch new ticket and retry connect" |

---

#### L6: Handover Pack References Undocumented Weather Feature

| Attribute | Value |
|-----------|-------|
| **File** | `docs/deployment/handover-pack-template.md` |
| **Section** | 6.1 Weather Nowcast (lines 107-114) |
| **Quote** | `Weather/nowcast features are informational and advisory only` |
| **Why It Hurts** | Template references "satellite ingest" and "nowcast features" but no corresponding specification exists in `docs/`. Installer confusion: "Where's the weather integration docs?" Creates perception of incomplete documentation. |
| **Suggested Fix** | (1) Add `docs/intelligence/weather.md` stub with "Planned for Year 2+" status, OR (2) Mark section in handover template as "IF APPLICABLE - weather integration is optional premium feature" |

---

#### L7: PHM Over-Specification Creates Expectation Gap

| Attribute | Value |
|-----------|-------|
| **File** | `docs/intelligence/phm.md` |
| **Section** | Entire document (1179 lines) |
| **Quote** | Detailed specifications for vibration sensors, bearing temperature, defrost cycles, etc. |
| **Why It Hurts** | A typical residential installation has: lights, blinds, HVAC, maybe a heat pump. No vibration sensors, no CT clamps on individual pumps. PHM spec implies rich monitoring capability, but most sites will have minimal telemetry. Installer/customer disappointment: "Where's the predictive maintenance you promised?" |
| **Suggested Fix** | Add "PHM Capability Tiers" section: (1) **Tier 1 (Standard)**: Runtime hours, error counts, cycle counting - works with any device, (2) **Tier 2 (Enhanced)**: Power monitoring via CT clamps/smart plugs, (3) **Tier 3 (Advanced)**: Dedicated sensors (vibration, temperature probes) - requires additional hardware |

---

#### L8: Update Rollback with Partial Migration State

| Attribute | Value |
|-----------|-------|
| **File** | `docs/operations/updates.md`, `docs/resilience/backup.md` |
| **Section** | Database Migration, Automatic Rollback |
| **Quote** | Migration creates backup, rollback restores backup |
| **Why It Hurts** | Migration succeeds (new schema), new binary starts, 5 minutes of operation, new binary crashes. Rollback restores old binary + old database. 5 minutes of new data lost. Acceptable for most scenarios, but what if that 5 minutes included critical config changes? |
| **Suggested Fix** | Document the rollback data loss window explicitly: "Rollback restores pre-update state. Any changes made between update and rollback are lost. For critical installations, verify operation for N hours before making configuration changes." |

---

## Suggested Surgical Strikes (Prioritized)

### Strike 1: Scene Conflict Resolution (2-3 hours) — COMPLETED 2026-01-18

**Priority:** HIGH  
**Files:** `docs/automation/automation.md`

**Tasks:**
1. Add "Scene Conflict Rules" section after "Timing and Sequencing"
2. Define device mutex behavior (same device targeted by multiple scenes)
3. Define stabilization window (minimum time between conflicting commands to same device)
4. Add scene priority field for explicit conflict resolution
5. Document UI behavior when conflict occurs (show user which scene "won")

**Changes Made:**
- Added `priority` field (1-100) to Scene structure with recommended ranges
- Added `conflicts` field to SceneExecution for tracking
- Added new "Scene Conflict Resolution" section with:
  - Conflict detection rules
  - Resolution rules (physical control wins, priority wins, last-write wins)
  - Stabilization window (500ms default)
  - ConflictEvent and ConflictRecord structures
  - UI behavior specification
  - Cross-reference to Physical Controls Always Work principle

---

### Strike 2: Clock Trust and NTP Requirements (1-2 hours) — COMPLETED 2026-01-18

**Priority:** HIGH  
**Files:** `docs/resilience/offline.md`, `docs/commissioning/discovery.md`

**Tasks:**
1. Add NTP requirement for all bridges to offline.md state reconciliation section
2. Add clock skew detection logic (reject timestamps differing >60s from Core clock)
3. Add bridge health indicator for clock sync status
4. Document commissioning verification: "Verify bridge time matches Core within 5 seconds"

**Changes Made:**
- Updated `offline.md` with:
  - Clock skew protection logic (rejects timestamps >60s drift)
  - Bridge clock health indicator API
  - Mandatory NTP requirements (local sources preferred)
  - Extended time commissioning checklist
- Updated `commissioning/discovery.md` with:
  - Time synchronization verification workflow
  - Commissioning commands for checking offset using `ssh` and `date`
  - Handover documentation requirements

---

### Strike 3: Voice Authentication Security (1.5 hours) — COMPLETED 2026-01-18

**Priority:** MEDIUM  
**Files:** `docs/intelligence/voice.md`, `docs/architecture/security-model.md`

**Tasks:**
1. Add "Voice Authentication" section to voice.md
2. Explicitly state: voice PIN NEVER logged (transcript sanitized)
3. Apply rate limiting to voice commands with authentication intent
4. Require confirmation flow for security-sensitive voice commands
5. Cross-reference from security-model.md PIN section

**Changes Made:**
-   Updated `docs/intelligence/voice.md`:
    -   Added new "Voice Authentication" section.
    -   Defined "Secure PIN Handling" (redaction rules).
    -   Defined "Challenge-Response Flow" for sensitive actions.
    -   Linked Voice PIN attempts to global security rate limits.
-   Updated `docs/architecture/security-model.md`:
    -   Added cross-reference to `voice.md` in the Voice PIN Rate Limiting section.

---

### Strike 4: Commissioning Verification Checklist (2 hours) — COMPLETED 2026-01-18

**Priority:** MEDIUM  
**Files:** `docs/domains/climate.md`, new file `docs/deployment/commissioning-checklist.md`

**Tasks:**
1. Create formal commissioning checklist document
2. Include MANDATORY frost protection hardware verification with sign-off
3. Include NTP/time sync verification for all bridges
4. Include status feedback verification for critical devices
5. Cross-reference from climate.md frost protection section

**Changes Made:**
-   Created `docs/deployment/commissioning-checklist.md`:
    -   Formal "gate-based" checklist for deployment.
    -   Added **mandatory** "Frost Protection Hardware Gate" with integrator signature requirement.
    -   Added "Clock Trust" verification (NTP limit <5s).
    -   Added "Offline Capability" disconnect tests.
-   Updated `docs/domains/climate.md`:
    -   Changed verification column to "MANDATORY: Sign-off in Commissioning Checklist".
    -   Added IMPORTANT alert linking relevant section to the new checklist.

---

### Strike 5: PHM Capability Tiers (1 hour) — COMPLETED 2026-01-18

**Priority:** LOW  
**Files:** `docs/intelligence/phm.md`

**Tasks:**
1. Add "PHM Capability Tiers" section near top of document
2. Define Tier 1 (Standard), Tier 2 (Enhanced), Tier 3 (Advanced)
3. Clarify which PHM features require which tier
4. Add "Typical Residential" vs "Commercial with BMS" capability expectations
5. Prevent expectation gap during sales/commissioning

**Changes Made:**
-   Updated `docs/intelligence/phm.md`:
    -   Added "PHM Capability Tiers" section defining three levels:
        -   **Tier 1 (Standard):** Native intelligence (DALI status, runtime) - Works everywhere.
        -   **Tier 2 (Enhanced):** Power/Energy analysis - Requires CT clamps/Smart plugs.
        -   **Tier 3 (Advanced):** Physical sensing (Vibration/Temp) - Requires industrial sensors.
    -   Added "Typical Deployment Expectations" table comparing Residential vs Commercial.

---

## Summary

Iteration 3 found **15 issues** total:
- **0 Critical** (Iteration 2 resolved all critical issues)
- **2 High** (Scene conflicts, Clock trust)
- **5 Medium** (PHM blind spot, Voice PIN, Schedule catch-up, Frost verification, API key rotation)
- **8 Low** (Various edge cases and over-specification)

The repository has matured significantly. The issues identified are subtler and would only manifest in edge cases or over longer deployment timelines. The five surgical strikes proposed are incremental improvements, not fundamental architecture changes.

**Recommendation:** Address Strikes 1-4 before v1.0 GA release. Strike 5 can be deferred to v1.1 if timeline is tight.

---

## Appendix: Files Reviewed

| File | Lines | Status |
|------|-------|--------|
| `docs/overview/principles.md` | 292 | ✓ Reviewed |
| `docs/overview/vision.md` | 213 | ✓ Reviewed |
| `docs/architecture/system-overview.md` | 527 | ✓ Reviewed |
| `docs/architecture/core-internals.md` | 1414 | ✓ Reviewed |
| `docs/architecture/security-model.md` | 800 | ✓ Reviewed |
| `docs/architecture/mqtt-resilience.md` | 438 | ✓ Reviewed |
| `docs/domains/climate.md` | 1397 | ✓ Reviewed |
| `docs/domains/security.md` | 991 | ✓ Reviewed |
| `docs/resilience/offline.md` | 984 | ✓ Reviewed |
| `docs/resilience/backup.md` | 877 | ✓ Reviewed |
| `docs/automation/automation.md` | 1290 | ✓ Reviewed |
| `docs/interfaces/api.md` | 2403 | ✓ Reviewed |
| `docs/intelligence/voice.md` | 1450 | ✓ Reviewed |
| `docs/intelligence/phm.md` | 1179 | ✓ Reviewed |
| `docs/protocols/knx.md` | 685 | ✓ Reviewed |
| `docs/operations/bootstrapping.md` | 267 | ✓ Reviewed |
| `docs/operations/updates.md` | 538 | ✓ Reviewed |
| `docs/deployment/handover-pack-template.md` | 179 | ✓ Reviewed |
| `docs/audit-iteration-2-log.md` | 263 | ✓ Baseline reference |

**Total Lines Reviewed:** ~14,000+ lines across 19 documents
