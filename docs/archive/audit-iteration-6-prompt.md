# Pre-Implementation Paranoid Audit Prompt

**Copy everything below the line and paste into a fresh Gemini session.**

---

You are performing a **final pre-implementation audit** of the Gray Logic Stack documentation. This is the last review before coding begins. Your mission is to be **extremely paranoid, deeply critical, and forensically thorough**. 

**Mindset**: Assume you are a hostile attacker, a lazy installer, a careless developer, an angry ex-employee, and a building owner suing for damages—all at once. Find every gap, every contradiction, every implicit assumption that will explode in production.

### Context

Gray Logic is a building automation platform with a **10-year deployment horizon**. It controls:
- Lighting, blinds, HVAC (comfort)
- Alarm systems, access control (security)
- CCTV integration (surveillance)
- Energy management (cost/safety)

**Core principles that MUST NOT be violated:**
1. Physical controls always work (software failure ≠ broken switches)
2. Life safety is independent (fire alarms never touched)
3. Works 100% offline (no cloud dependencies)
4. 10-year stability (no breaking changes, no deprecated dependencies)
5. Privacy by design (local processing, no cloud upload of voice/video)

### Your Audit Scope

Perform a **6-layer deep analysis**:

#### Layer 1: Security Attack Vectors
- Can an attacker brute-force PINs via any interface?
- Can a compromised bridge poison state or escalate privileges?
- Are there any plaintext secrets on the network?
- What happens if MQTT broker is compromised?
- Can a malicious device discovery flood crash the system?
- Is there a path from guest user → admin?
- What happens if JWT secret is leaked?
- Are there any TOCTOU (time-of-check-time-of-use) vulnerabilities?

#### Layer 2: Data Integrity & Consistency
- Can two scenes target the same device simultaneously?
- What happens if a schedule fires during a scene execution?
- Can state be corrupted by out-of-order MQTT messages?
- What if a bridge clock is 6 months in the future?
- Can a failed migration leave the database in an inconsistent state?
- What if InfluxDB fills up the disk?
- Are there any circular dependencies in configuration?

#### Layer 3: Failure Modes & Edge Cases
- What happens when Core starts with RTC battery dead (year 1970)?
- What happens when all bridges are offline at startup?
- What happens when Core crashes mid-scene execution?
- What happens when backup runs during heavy write load?
- What happens when user upgrades, migration fails, they rollback, then use system—is data lost?
- What happens when a device is replaced but old one comes back online?
- What happens when NTP server is unreachable for 30 days?

#### Layer 4: Operational Landmines
- Can an installer commission a system without verifying frost protection?
- Can a user accidentally disable the alarm with no confirmation?
- Can a scheduled scene run at 3am with 100% brightness?
- Can API keys be created with no expiry if someone tries hard enough?
- Can logs fill up the disk and crash the system?
- Can a USB backup be restored to a different site accidentally?
- Are there any "TODO" or "FIXME" comments in specifications?

#### Layer 5: Cross-Document Contradictions
- Does `backup.md` match what `commissioning-checklist.md` requires?
- Does `api.md` token lifetimes match `security-model.md`?
- Does `energy.md` load shedding work with actual device capabilities?
- Does `voice.md` rate limiting match `security-model.md`?
- Does `offline.md` time handling match `core-internals.md`?
- Do all `depends_on` references point to existing files?
- Are there orphaned features mentioned but never specified?

#### Layer 6: 10-Year Time Bombs
- What dependencies might be deprecated in 2030?
- What happens when Let's Encrypt changes their API?
- What happens when KNX 3.0 is released?
- What if SQLite hits the 281TB limit?
- What if a manufacturer goes bankrupt and devices need replacing?
- What if the original installer's company closes?
- What certificates will expire and how are they renewed?

### Files to Review (MANDATORY)

Load and analyze ALL of these files thoroughly:

**Architecture (Critical)**
- `docs/architecture/security-model.md`
- `docs/architecture/core-internals.md`
- `docs/architecture/mqtt-resilience.md`
- `docs/architecture/bridge-interface.md`

**Resilience (Critical)**
- `docs/resilience/offline.md`
- `docs/resilience/backup.md`

**Operations (Critical)**
- `docs/operations/bootstrapping.md`
- `docs/operations/updates.md`
- `docs/operations/maintenance.md`

**Interfaces (Critical)**
- `docs/interfaces/api.md`

**Domains (High)**
- `docs/domains/security.md`
- `docs/domains/climate.md`
- `docs/domains/energy.md`

**Intelligence (High)**
- `docs/intelligence/voice.md`
- `docs/intelligence/phm.md`

**Protocols (High)**
- `docs/protocols/mqtt.md`
- `docs/protocols/knx.md`

**Development (Medium)**
- `docs/development/SECURITY-CHECKLIST.md`
- `docs/development/CODING-STANDARDS.md`

**Deployment (Medium)**
- `docs/deployment/commissioning-checklist.md`
- `docs/deployment/handover-pack-template.md`

**Previous Audits (Reference)**
- `docs/audit-iteration-2-log.md`
- `docs/audit-iteration-3-log.md`
- `docs/audit-iteration-5-log.md`

### Output Format

Create a new file: `docs/audit-iteration-6-log.md`

Use this EXACT structure:

```markdown
---
title: Audit Iteration 6 Log
version: 1.0.0
status: active
audit_date: 2026-01-18
auditor: Gemini CLI (Antigravity)
scope: Final pre-implementation paranoid audit
previous_audit: audit-iteration-5-log.md
---

# Audit Iteration 6 Log — Gray Logic Stack

**Audit Date:** 2026-01-18  
**Auditor:** Gemini CLI (Antigravity)  
**Scope:** Final paranoid audit before implementation begins  
**Mission:** Find every issue that will cause pain in 2026-2036 deployments

---

## Executive Threat Summary

### Context
[Brief summary of what this audit found]

### Top N Nastiest Findings (Ranked by Long-Term Pain)

| Rank | Finding | Pain Level | Time to Detonate |
|------|---------|------------|------------------|
| 1 | **[Finding Name]** | CRITICAL/HIGH/MEDIUM/LOW | [When this will hurt] |
...

### Readiness Score vN

| Category | Score | Notes |
|----------|-------|-------|
| **Core Architecture** | X/10 | ... |
| **Security Model** | X/10 | ... |
| **Resilience** | X/10 | ... |
| **Automation** | X/10 | ... |
| **Commissioning** | X/10 | ... |
| **PHM** | X/10 | ... |
| **Documentation Quality** | X/10 | ... |

**Overall Readiness: X.X/10**

---

## Detailed Issues Table

### Critical Issues (N)
[If any]

### High Severity Issues (N)

#### H1: [Issue Name]

| Attribute | Value |
|-----------|-------|
| **File** | `path/to/file.md` |
| **Section** | Section Name (lines X-Y) |
| **Quote** | `"Exact quote from spec"` |
| **Why It Hurts** | [Detailed explanation of the pain] |
| **Suggested Fix** | [Concrete fix with specific changes] |

[Repeat for each issue]

### Medium Severity Issues (N)
[Same format]

### Low Severity Issues (N)
[Same format]

---

## Suggested Surgical Strikes (Prioritized)

### Strike 1: [Name] (Estimated hours)

**Priority:** HIGH/MEDIUM/LOW  
**Files:** `list of files`

**Tasks:**
1. [Specific task]
2. [Specific task]
...

[Repeat for each strike]

---

## Summary

Iteration 6 found **N issues** total:
- **N Critical** 
- **N High** 
- **N Medium** 
- **N Low** 

**Recommendation:** [BLOCK / GO WITH CONDITIONS / GO]

---

## Appendix: Files Reviewed

| File | Lines | Status |
|------|-------|--------|
| `path/to/file.md` | N | ✓ Reviewed |
...

**Total Lines Reviewed:** ~N lines across N documents
```

### Attitude

- **Be paranoid.** Assume every undocumented edge case will be exploited.
- **Be specific.** Quote exact lines. Name exact files. 
- **Be actionable.** Every finding must have a concrete fix.
- **Be honest.** If something is fine, say so. Don't invent problems.
- **Be thorough.** Load every file. Read every line. Check every cross-reference.

### Start Now

Begin by loading `docs/audit-iteration-5-log.md` to understand prior findings, then systematically work through each file category above. Create the audit log as you go, updating it with each finding.

This is the LAST CHANCE to catch issues before coding begins. Find them all.
