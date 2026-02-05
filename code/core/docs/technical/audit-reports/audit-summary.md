---
title: Audit Summary & History
version: 2.0.0
status: active
last_updated: 2026-02-05
auditor: Multiple (Gemini, Claude Code)
---

# Audit Summary & History

> **Current Readiness Score:** 8.6 / 10 (Fix High Items → Production Ready)

This document summarizes the comprehensive audit process conducted across two phases:
1. **Documentation Phase** (January 17-18, 2026) — Specification audits by Gemini
2. **Implementation Phase** (January 20-22, 2026) — Code audits by Claude Code

---

## 1. Audit Progression

### Phase 1: Documentation Audits (Specifications)

| Iteration | Date | Focus | Score | Key Result |
|-----------|------|-------|-------|------------|
| **2** | Jan 17 | Forensic Deep Dive | **7.2** | Found MQTT SPOF, JWT gaps, Backup circular dep |
| **3** | Jan 18 | Edge Cases & Hygiene | **7.5** | Found Scenes conflict, Clock trust issues |
| **4** | Jan 18 | Long-term Stability | **8.8** | Confirmed critical ops procedures (Cert rotation) |
| **5** | Jan 18 | Holistic Consistency | **9.0** | Fixed Time-Travel deadlock, Archive backup gap |
| **6** | Jan 18 | Paranoid Pre-Impl | **9.4** | Found Load Shedding gaps, PHM blind spots |
| **7** | Jan 18 | Surgical Verification | **9.8** | Verified all Strike fixes (H1-H2, M1-M4) |
| **8** | Jan 18 | New Docs Verification | **9.6** | Verified Capabilities v2.0, DIY Spec, Cloud Relay |

### Phase 2: Code Audits (Implementation)

| Iteration | Date | Focus | Score | Key Result |
|-----------|------|-------|-------|------------|
| **9** | Jan 20 | M1.1 Infrastructure | **9.2** | Fixed double-close panic, unbounded goroutines |
| **10** | Jan 22 | M1.2 knxd Manager | **9.6** | Fixed 8 issues: race conditions, PID handling, D-state |
| **11** | Jan 23 | M1.4 REST API + WebSocket | **9.4** | Fixed 12 issues across 7 iterations: DoS, panics, races |
| **12** | Jan 23-24 | M1.6 Basic Scenes | **9.4** | Fixed 10 issues (5 iterations): race, DoS, timeout, validation, abort logic, DeepCopy. Final Advisory: SHIP IT. |
| **13** | Jan 24 | M1.5 Wall Panel (Go) | **9.5** | Fixed 7 issues (1 iteration): goroutine leak, unchecked errors, unused type, errorlint, staticcheck. SHIP IT. |
| **14** | Feb 03 | MQTT Topic Unification | **9.3** | 0 Critical/High, 3 Medium (defense-in-depth). Fixed VictoriaMetrics token auth. SHIP IT. |
| **15** | Feb 05 | M1.7 Auth + M2.1 + State Pipeline | **8.6** | 3 High (log leak, TSDB injection, rate limit bypass), 15 Medium, 10 Low. Fix H1-H3 before shipping. |

> **Note:** Code audits follow 7-phase verification (tests, lint, vulncheck, AI review, architecture, deps, docs).

---

## 2. Key Findings & Resolutions

### 🔴 Critical Issues — Documentation Phase (Resolved)

1.  **Backup Circular Dependency** (Iteration 2)
    *   *Risk:* Backup encryption key stored inside the backup itself.
    *   *Fix:* Separated key storage; mandated "Doomsday Pack" physical key copy.

2.  **Time-Travel Deadlock** (Iteration 5)
    *   *Risk:* Core RTC failure + internet down = 1970 timestamp, rejecting valid bridge data.
    *   *Fix:* Added "Sanity Check" logic to accept external timestamps if Core time is clearly wrong.

3.  **The Forgotten Archive** (Iteration 5)
    *   *Risk:* Log archives excluded from backup script.
    *   *Fix:* Added explicit archive directory to backup manifest.

### 🔴 Critical Issues — Code Phase (Resolved)

1.  **PID File Race Condition** (Iteration 10)
    *   *Risk:* PID file removed before process stopped, allowing duplicate instances.
    *   *Fix:* Moved `removePIDFile()` to after `process.Stop()` completes.

2.  **BusMonitor Statement Race** (Iteration 10)
    *   *Risk:* Prepared statements accessed during shutdown without synchronisation.
    *   *Fix:* Added `isClosed()` check before statement access.

3.  **D-State Tolerance Indefinite** (Iteration 10)
    *   *Risk:* Hung USB (D-state) tolerated indefinitely, freezing KNX bus.
    *   *Fix:* Added `dStateCount` tracker; fail after 3 consecutive D-state checks.

4.  **PID File Path Inconsistency** (Iteration 10)
    *   *Risk:* Different paths used for acquire vs remove if permissions changed.
    *   *Fix:* Store path at acquisition in `activePIDFilePath`.

### 🟠 High Severity Issues (Selected Resolved)

*   **MQTT Single Point of Failure** — Documented persistence, backoff + jitter, watchdog.
*   **Scene Conflict Resolution** — Added mutex logic, "Last-Write-Wins" with stabilisation.
*   **Untrusted Bridge Clocks** — Mandated NTP, clock skew rejection (>60s).
*   **Statement Leak on Start() Failure** — Added `closeStatements()` helper on all error paths.
*   **Busy Loop on Invalid Message** — Close connection on protocol violation.

### 🟡 Medium Severity Issues (Selected Resolved)

*   **Certificate Rotation**: Fully documented auto-renewal handshake.
*   **UK Spelling Consistency**: Standardised on UK English throughout codebase.
*   **Variable Shadowing**: Renamed shadow variables to avoid govet warnings.
*   **Magic Numbers**: Added constants with nolint directives where bounds validated.

---

## 3. Lessons Learned

### From Documentation Phase

1.  **"Hardware First" is not just a slogan.**
    *   Software logic (Frost Protection, Load Shedding) relies on hardware capabilities that must be verified physically. *Commissioning checklists are code.*

2.  **Time is the enemy.**
    *   Timestamps, clock drift, RTC failure, and certificate expiry were 40% of high-severity findings. *Always validate time sources.*

3.  **Silence is dangerous.**
    *   Silent failures (shedding a clamp, ignoring a failed sensor) were common spec bugs. *Explicitly handle and log every failure mode.*

### From Code Phase

4.  **Race conditions hide in shutdown paths.**
    *   Most race conditions were in graceful shutdown, restart, and error recovery paths. *Test shutdown as rigorously as startup.*

5.  **AI reviewers have diminishing returns.**
    *   After 2-3 audit rounds, AI findings become false positives or style preferences. *Know when to stop auditing and ship.*

6.  **Store state, don't re-derive it.**
    *   The PID file path bug came from re-calling `getPIDFilePath()` instead of storing the result. *Capture decisions at decision time.*

---

## 4. Audit Reports Index

### Code Audits (Phase 2)

| File | Date | Scope | Issues Fixed |
|------|------|-------|--------------|
| `audit-2026-02-05-m17-m21-full-stack.md` | Feb 05 | M1.7 auth, M2.1 tags/groups/zones, TSDB, state pipeline | 0 (findings reported, fixes pending) |
| `audit-2026-02-04-pipeline-refactor.md` | Feb 04 | KNX pipeline robustness refactor | 12 |
| `audit-2026-02-03-mqtt-unification.md` | Feb 03 | MQTT topic unification | 1 |
| `audit-2026-01-24-m1.5-panel.md` | Jan 24 | Wall panel Go embed, panel handler | 7 |
| `audit-2026-01-23-m1.6-scenes.md` | Jan 23-24 | automation package, scene API handlers | 10 (5 iterations, Final Advisory) |
| `audit-2026-01-23-m1.4-api.md` | Jan 23 | REST API, WebSocket hub, middleware | 12 |
| `audit-2026-01-22-m1.2-knxd.md` | Jan 22 | knxd manager, process manager, bus monitor | 8 |

### Documentation Audits (Phase 1)

| File | Date | Focus |
|------|------|-------|
| `audit-iteration-2-log.md` | Jan 17 | The wake-up call (MQTT, Backups) |
| `audit-iteration-3-log.md` | Jan 18 | The deep dive (Scenes, Clocks) |
| `audit-iteration-4-log.md` | Jan 18 | The stability check (Certs, RMA) |
| `audit-iteration-5-log.md` | Jan 18 | The consistency check (Time, Archives) |
| `audit-iteration-5-prompt.md` | Jan 18 | Prompt for iteration 5 |
| `audit-iteration-6-log.md` | Jan 18 | The paranoid check (Load Shedding) |
| `audit-iteration-6-prompt.md` | Jan 18 | Prompt for iteration 6 |
| `audit-iteration-7-log.md` | Jan 18 | The fix verification |
| `audit-iteration-8-log.md` | Jan 18 | The new features check |

---

## 5. Running Audits

### Quick Audit (5 minutes)

```bash
cd code/core
go test -race ./... && golangci-lint run && govulncheck ./...
```

### Full Audit

Use the `/code-audit` command in Claude Code, which:
1. Tracks audit runs automatically
2. Creates detailed markdown reports
3. Updates this summary
4. Follows Standard → Final Advisory progression

See `.claude/commands/code-audit.md` for full documentation.
