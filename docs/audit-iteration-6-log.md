---
title: "Audit Iteration 6 — Pre-Implementation Paranoid Audit"
version: 1.0.0
status: complete
date: 2026-01-18
auditor: "Gemini (Antigravity)"
scope: "Final 6-layer deep analysis before implementation"
---

# Audit Iteration 6 — Pre-Implementation Paranoid Audit

This audit represents the final paranoid pre-implementation review of the Gray Logic Stack documentation. It builds upon the resolutions from Iterations 2, 3, and 5, specifically verifying their implementation and searching for any remaining gaps, contradictions, or 10-year time bombs.

---

## Executive Summary

### Previous Audit Status

| Iteration | Critical Issues | Status |
|-----------|-----------------|--------|
| **Iteration 2** | MQTT SPOF, Backup Key Circular Dependency | ✅ Resolved |
| **Iteration 3** | Scene Conflict, Clock Trust, Voice Auth, Frost Protection | ✅ Resolved |
| **Iteration 5** | Time-Travel Deadlock, Archive Backup, USB Schrodinger | ✅ Resolved |

### Iteration 6 Findings Summary

| Severity | Count | Description |
|----------|-------|-------------|
| **CRITICAL** | 0 | None identified |
| **HIGH** | 2 | Operational gaps requiring attention before production |
| **MEDIUM** | 4 | Issues that should be addressed within first year |
| **LOW** | 3 | Minor documentation improvements or edge cases |

### 🎯 Readiness Score: **9.4 / 10**

**Verdict: GO FOR CODE** — No blocking issues. High severity items are operational considerations, not architectural flaws.

---

## Verification of Previous Audit Resolutions

### ✅ Time-Travel Deadlock (Iteration 5 - C1)
- **Resolution Verified**: `docs/resilience/offline.md` lines 396-401 now include:
  - `minimum_valid_year: 2026` sanity check
  - `action_on_rtc_failure: "Accept external timestamp if > minimum_valid_year"`
- **Status**: RESOLVED

### ✅ Archive Backup Gap (Iteration 5 - C2)
- **Resolution Verified**: `docs/resilience/backup.md` line 233 now includes:
  - `/var/lib/graylogic/archive/` in secrets backup tar
- **Status**: RESOLVED

### ✅ USB Schrodinger Contradiction (Iteration 5 - M1)
- **Resolution Verified**: `docs/resilience/backup.md` line 763:
  - `status: "mandatory_for_commissioning"` with comment `# Required for "Gold Master" handover`
- **Cross-Verified**: `docs/deployment/commissioning-checklist.md` line 94:
  - `- [ ] **Backup**: USB stick with "Gold Master" full system backup physically left on site.`
- **Status**: RESOLVED

### ✅ Clock Skew Protection (Iteration 3 - Strike 2)
- **Resolution Verified**: `docs/resilience/offline.md` lines 392-437:
  - 60-second max acceptable skew
  - Bridge clock health status API
  - Commissioning verification requirement
- **Cross-Verified**: `docs/deployment/commissioning-checklist.md` lines 28-34
- **Status**: RESOLVED

### ✅ Voice Authentication Security (Iteration 3 - Strike 3)
- **Resolution Verified**: `docs/intelligence/voice.md` lines 599-643:
  - PIN transcript sanitization
  - Challenge-response flow documented
  - 3-attempt rate limit with 5-min lockout
- **Status**: RESOLVED

### ✅ Frost Protection Commissioning (Iteration 3 - Strike 4)
- **Resolution Verified**: `docs/deployment/commissioning-checklist.md` lines 52-66:
  - Hardware Gate with thermostat audit
  - Signed integrator declaration
  - Bus failure simulation test
- **Status**: RESOLVED

---

## Layer 1: Security Attack Vectors

### H1: Load Shedding Targets Non-Controllable Devices

**Severity**: HIGH  
**Location**: `docs/domains/energy.md` lines 366-407

**Finding**: The load shedding configuration in `LoadPriority` references device IDs without verifying the device has the `on_off` capability. A configuration referencing a CT clamp (monitoring-only) as a shed target would silently fail.

**Evidence**:
```yaml
# energy.md example shows:
shed_loads:
  - device_id: "fridge-kitchen"  # What if this is just a CT clamp?
    priority: 1
```

There is a comment at line 373-374:
```yaml
# NOTE: Targets must have the 'on_off' capability (Smart Plug, DIN-rail Relay, Contactor).
```

However, this is documentation-level guidance only. There is no schema validation or Core enforcement specified.

**Impact**: During a demand response event, Core may attempt to shed loads it cannot control, leading to:
1. Silent failure (load not actually shed)
2. False confidence in load reduction
3. Potential grid demand charges

**Proposed Fix (Surgical Strike 1)**:
1. Add to `docs/domains/energy.md` → "Load Shedding Validation" section:
   ```yaml
   load_shedding_validation:
     on_startup:
       validate_controllable: true
       action_on_invalid: "warn"   # Log warning, remove from active shed list
     on_command:
       verify_capability: "on_off"
       on_missing_capability:
         log: "error"
         skip_device: true
         notify_admin: true
   ```

2. Add to `docs/deployment/commissioning-checklist.md`:
   ```markdown
   - [ ] **Load Shedding Audit**: Verified all devices in load shedding configuration have `on_off` capability.
   ```

---

### H2: PHM Cascade Failure During Learning Phase

**Severity**: HIGH  
**Location**: `docs/intelligence/phm.md` lines 260-275, 482-495

**Finding**: The PHM specification correctly documents "Safe Operating Limits (SOL)" for Day 0 protection, but there is no specification for what happens if both:
1. The sensor fails (reports 0W when pump is ON)
2. AND the pump is genuinely failing (bearing wear)

The state correlation check at line 246-262 will flag the sensor, not the pump. If the sensor is then ignored, genuine pump failures during the learning phase will go undetected.

**Evidence**:
```yaml
# phm.md lines 246-262
correlation_check:
  - device_id: "pump-chw-1"
    condition: "state == 'on'"
    expect:
      parameter: "power_kw"
      operator: ">"
      value: 0.1
    on_fail:
      # Flags sensor, not pump
      flag_sensor("ct-clamp-pump-1", "...")
```

**Impact**: Real equipment failures could be masked by sensor validation logic, particularly during the baseline learning phase.

**Proposed Fix (Surgical Strike 2)**:
Add to `docs/intelligence/phm.md` → "Dual Fault Detection" section:
```yaml
dual_fault_awareness:
  # When sensor correlation fails, ALSO consider equipment failure
  on_correlation_failure:
    primary_action: "flag_sensor"
    secondary_action:
      notify: "Check both sensor AND equipment"
      severity: "warning"
      message: |
        Correlation failure detected for {device_id}.
        Possible causes:
        1. Sensor failure (most likely)
        2. Equipment failure (verify manually)
        3. Wiring issue

  # During learning phase, apply extra caution
  during_learning:
    correlation_failure_action:
      escalate: true
      require_manual_verification: true
```

---

## Layer 2: Data Integrity & Consistency

### M1: API Key Rotation Self-Permission Not in Default Roles

**Severity**: MEDIUM  
**Location**: `docs/interfaces/api.md` lines 336, 276-297

**Finding**: The `apikeys:rotate_self` permission is documented as enabling service accounts to auto-rotate their keys. However, this permission is not included in ANY default role definition in `docs/architecture/security-model.md`.

**Evidence**:
- `api.md` line 336: `| apikeys:rotate_self | Allow key to rotate itself |`
- `security-model.md` defines roles but does not include this permission in `integration` role

**Impact**: Service accounts created with `integration` role cannot auto-rotate keys without manual permission assignment.

**Proposed Fix (Surgical Strike 3)**:
Add to the `integration` role in `docs/architecture/security-model.md`:
```yaml
roles:
  integration:
    permissions:
      - devices:read
      - devices:control
      - scenes:execute
      - apikeys:rotate_self    # ADD THIS
```

---

### M2: Secrets Backup Encryption Key Location Ambiguity

**Severity**: MEDIUM  
**Location**: `docs/resilience/backup.md` lines 219-241, 237-240

**Finding**: The secrets backup uses GPG encryption with key `backup@graylogic.local`, but the key storage specifies:
```yaml
key_storage:
  - "Password manager (owner)"
  - "Sealed envelope in doomsday pack"
```

There is no specification for:
1. Who generates this GPG key initially?
2. Where is the key stored on the running system?
3. How is it accessed during automated backups?

**Impact**: Circular dependency risk — automated backups cannot run if the key is only in password manager/sealed envelope.

**Proposed Fix (Surgical Strike 4)**:
Add to `docs/resilience/backup.md` → "GPG Key Management" section:
```yaml
gpg_key_management:
  generation:
    when: "During commissioning via setup wizard"
    command: "graylogic setup --generate-backup-key"
    output:
      private_key: "/etc/graylogic/backup.key"   # Mode 0600
      public_key: "/etc/graylogic/backup.pub"

  automated_backup_access:
    key_location: "/etc/graylogic/backup.key"
    permissions: "0600 graylogic:graylogic"
    note: "Private key remains on server for automated backups"

  offsite_recovery:
    method: "Private key ALSO stored in doomsday pack"
    rationale: "Server destruction requires external key copy"
```

---

### M3: WebSocket Ticket Lifetime Inconsistency

**Severity**: MEDIUM  
**Location**: `docs/interfaces/api.md` line 213

**Finding**: API documentation states WebSocket ticket lifetime as "120 seconds", but there is no cross-reference to where this is defined in the security model or how it's enforced.

**Evidence**:
```markdown
| WebSocket Ticket | 120 seconds | No | Single-use, for secure WS connection |
```

The `security-model.md` document extensively covers JWT and refresh tokens but makes no mention of WebSocket ticket authentication.

**Impact**: Implementation ambiguity — developers may implement inconsistent ticket validation.

**Proposed Fix (Surgical Strike 5)**:
Add to `docs/architecture/security-model.md` → "WebSocket Authentication" section:
```yaml
websocket_authentication:
  method: "ticket_based"
  flow:
    1: "Client authenticates via REST (gets JWT)"
    2: "Client requests ticket: POST /api/v1/auth/ws-ticket"
    3: "Server issues single-use ticket (random, 120s TTL)"
    4: "Client connects: ws://host/ws?ticket={ticket}"
    5: "Server validates ticket, establishes session"

  ticket_properties:
    format: "32-byte random, base64url encoded"
    lifetime_seconds: 120
    single_use: true
    bound_to_user: true
    storage: "in-memory only"
```

---

### M4: Certificate Rotation "Renewal Handshake" Undefined

**Severity**: MEDIUM  
**Location**: `docs/operations/maintenance.md` lines 96-99

**Finding**: The maintenance specification states:
```markdown
If a Bridge connects with an expired cert, the Core rejects it but offers a specific "Renewal" handshake (requires shared secret).
```

This "Renewal handshake" is not defined anywhere in the documentation. The security model doesn't mention it, and the bridge interface specification doesn't include it.

**Impact**: Critical operational procedure is undocumented. In a real deployment, an installer finding an expired bridge cert has no documented recovery path except manual certificate replacement.

**Proposed Fix (Surgical Strike 6)**:
Either:
1. Document the renewal handshake protocol fully in `docs/architecture/security-model.md`, OR
2. Remove the reference and document the manual certificate replacement procedure

Recommendation: Given the 10-year horizon, automatic renewal is valuable. Add to `security-model.md`:
```yaml
certificate_renewal:
  automatic:
    enabled: true
    renew_before_days: 30
    method: "internal_ca_auto_issue"

  expired_recovery:
    # Bridge with expired cert attempts connection
    scenario: "Bridge certificate expired"
    procedure:
      1: "Core rejects TLS handshake (expired)"
      2: "Bridge falls back to renewal endpoint (plaintext localhost or separate renewal port)"
      3: "Bridge presents: bridge_id + HMAC(timestamp, shared_secret)"
      4: "Core validates HMAC against stored bridge secret"
      5: "Core issues new certificate to bridge"
      6: "Bridge retries normal connection with new cert"

    shared_secret:
      location: "/etc/graylogic/bridges/{bridge_id}.secret"
      generated: "During bridge provisioning"
```

---

## Layer 3: Failure Modes & Edge Cases

### L1: PHM Baseline Staleness Threshold Not Defined

**Severity**: LOW  
**Location**: `docs/intelligence/phm.md` lines 454-461

**Finding**: The PHM baseline status includes a `stale` state with `ui_indicator: "Baseline outdated"`, but there is no definition of WHEN a baseline becomes stale.

**Evidence**:
```yaml
stale:
  description: "Baseline too old, may not reflect current normal"
  # Missing: threshold_days, conditions
```

**Impact**: Minor — affects PHM accuracy reporting, not safety.

**Proposed Fix**: Add `stale_after_days: 90` or similar threshold to the baseline status configuration.

---

### L2: Offline Catch-Up Schedule Skip Notification

**Severity**: LOW  
**Location**: `docs/resilience/offline.md` lines 726-727

**Finding**: When schedules are skipped during catch-up due to mode/presence conditions:
```yaml
on_skip:
  log: "info"
  message: "Skipped catch-up for schedule '{name}': {reason}"
  ui_notification: true
```

There is no specification of WHAT notification is sent or how the user dismisses it. This could lead to persistent notifications cluttering the UI.

**Impact**: Minor UX concern.

**Proposed Fix**: Add `notification_auto_dismiss_minutes: 60` or similar.

---

### L3: Energy Domain Phantom Load Detection Threshold Undefined

**Severity**: LOW  
**Location**: `docs/domains/energy.md` lines 67-71 (referenced as "Phantom load detection" in Tier 2)

**Finding**: "Phantom load detection" is listed as a Tier 2 PHM feature in the specification, but there is no definition of what constitutes a phantom load or the detection threshold.

**Impact**: Implementation ambiguity.

**Proposed Fix**: Add definition:
```yaml
phantom_load_detection:
  definition: "Continuous power draw when device should be off"
  threshold_w: 5        # >5W when state='off'
  duration_minutes: 30  # Must persist for 30 minutes
  action: "alert"
```

---

## Layer 4: Operational Landmines

No new operational landmines identified. Previous issues have been addressed via commissioning checklist and documentation updates.

---

## Layer 5: Cross-Document Contradictions

### Contradiction 1: Token Lifetime Descriptions

**Documents**: `api.md` vs general narrative

**Finding**: `api.md` line 212 states "30 days (rolling)" for refresh tokens but the phrase "rolling" is not explained in context. A reader might interpret this as "30 days from last use" (correct) or "30 days total" (incorrect).

**Resolution**: The security-model.md correctly explains rolling vs absolute (90-day) limits. This is a clarity issue, not a contradiction.

**Recommendation**: Add parenthetical to api.md: "30 days (rolling, 90-day absolute max)"

---

## Layer 6: 10-Year Time Bombs

### Observation: No New 10-Year Risks Identified

The documentation now includes:
- 10-year certificate rotation procedures
- Database cruft mitigation via v2 rewrite path
- Service account auto-rotation capability
- PHM baseline staleness handling (with L1 fix)

**Status**: The 10-year deployment horizon appears well-considered.

---

## Surgical Strikes Priority Order

| Priority | Strike | Severity | Effort | Description |
|----------|--------|----------|--------|-------------|
| 1 | H1 | HIGH | Small | Load shedding capability validation |
| 2 | H2 | HIGH | Small | PHM dual fault detection awareness |
| 3 | M1 | MEDIUM | Trivial | Add `apikeys:rotate_self` to integration role |
| 4 | M2 | MEDIUM | Small | Document GPG key generation and storage |
| 5 | M3 | MEDIUM | Small | Document WebSocket ticket authentication in security model |
| 6 | M4 | MEDIUM | Medium | Document certificate renewal handshake |
| 7 | L1 | LOW | Trivial | Define baseline staleness threshold |
| 8 | L2 | LOW | Trivial | Add notification auto-dismiss |
| 9 | L3 | LOW | Trivial | Define phantom load detection threshold |

---

## Conclusion

The Gray Logic Stack documentation has reached a mature state suitable for implementation. The issues identified in this audit are:

1. **Not blocking** — None prevent starting development
2. **Operational clarifications** — Most are "how exactly does X work" rather than missing features
3. **Edge case handling** — Natural refinements discovered through paranoid analysis

### Recommendation

**Proceed to implementation** with the following approach:

1. **During Alpha**: Address H1, H2, M1, M2 (high-impact, low-effort)
2. **During Beta**: Address M3, M4 (security model completeness)
3. **During Commissioning Framework**: Address L1, L2, L3 (polish)

---

## Appendix: Files Reviewed

| Category | Files |
|----------|-------|
| Architecture | security-model.md, core-internals.md, mqtt-resilience.md, bridge-interface.md, decisions/004-additive-only-migrations.md |
| Resilience | offline.md, backup.md |
| Operations | bootstrapping.md, updates.md, maintenance.md |
| Interfaces | api.md |
| Domains | security.md, climate.md, energy.md |
| Intelligence | voice.md, phm.md |
| Deployment | commissioning-checklist.md |
| Development | SECURITY-CHECKLIST.md |
| Previous Audits | audit-iteration-2-log.md, audit-iteration-3-log.md, audit-iteration-5-log.md |

---

*Audit completed: 2026-01-18T19:00:00Z*
