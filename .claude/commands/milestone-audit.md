---
description: End-of-milestone quality gate with full audit, coverage enforcement, and integration testing
---

# Milestone Audit — End-of-Milestone Quality Gate

A comprehensive, staged audit that verifies code correctness, security, architecture compliance, and test coverage before marking a milestone complete.

**Run this before marking any milestone as complete.**

---

## What This Audit Proves (and Doesn't Prove)

### ✅ What This Audit Validates
- Code compiles and tests pass
- No known security vulnerabilities in dependencies
- No lint/security warnings from static analysis
- AI reviewer found no critical bugs or security issues
- Architecture follows Hard Rules (offline-first, no cloud, etc.)
- Dependencies are stable for 20-year deployment
- Test coverage meets tier targets
- Documentation matches implementation

### ❌ What This Audit Does NOT Validate
- End-to-end system behavior (requires real hardware)
- Performance under load (requires load testing)
- Long-term reliability (requires production runtime)
- KNX protocol compliance (verify against `docs/protocols/knx-reference.md`)

### About Protocol Compliance

KNX is an open standard. Gray Logic MUST comply with:
- Datapoint Type encoding/decoding (DPT 1.xxx, 5.xxx, 9.xxx, etc.)
- Telegram structure and addressing formats
- EIB/cEMI protocol handshakes

Reference: `docs/protocols/knx-reference.md` contains the specification extracts.
The audit SHOULD verify implementations against this reference.

### Honesty Requirements

The report MUST include:
1. **Stages Summary Table** — Show PASS/FAIL/SKIPPED for each of the 10 stages
2. **"What Was Proven" section** — List specific validations with evidence
3. **"What Was NOT Proven" section** — List gaps and why (e.g., "no KNX devices on bus")
4. **Skipped stages clearly marked** — If a stage wasn't run, say so explicitly
5. **Coverage reality check** — State that "coverage % ≠ correctness %" 

Never claim "all tests pass" implies the code is correct. Tests verify code does what code does, not what it should do.

---

## Audit Stages Overview

| Stage | Name | Purpose | What It Proves |
|-------|------|---------|----------------|
| 0 | Setup | Load tracker, detect tiers | Audit can proceed |
| 1 | Static Analysis | Lint + security scan | No obvious code quality issues |
| 2 | Vulnerability Scan | CVE check on deps | No known security holes in dependencies |
| 3 | AI Code Review | Deep semantic analysis | No logic errors, security issues, or bugs |
| 3.5 | Protocol Compliance | Verify against specs | KNX/DALI/Modbus follow official standards |
| 4 | Architecture Review | Hard Rules compliance | System follows Gray Logic principles |
| 5 | Dependency Audit | 20-year stability check | Deps will exist long-term |
| 6 | Coverage Sweep | Test coverage enforcement | Code paths are exercised |
| 7 | Integration Tests | Cross-package verification | Components work together |
| 8 | Documentation Sync | Docs match code | Documentation is accurate |
| 9 | Technical Debt | TODO/FIXME scan | Debt is tracked and acceptable |
| 10 | Report | Generate final report | Milestone readiness documented |

---

## Stage 0: Setup

### What Claude Must Say

```
╔══════════════════════════════════════════════════════════════════╗
║  MILESTONE AUDIT: [Milestone Name]                               ║
║  Date: YYYY-MM-DD                                                ║
║  Commit: [git hash]                                              ║
╚══════════════════════════════════════════════════════════════════╝

STAGE 0: SETUP
══════════════════════════════════════════════════════════════════

Purpose: Initialize audit, load/create tracker, detect available test tiers.

[Tracker status]
[Available tiers]

Proceeding to Stage 1...
```

### Actions

1. Determine milestone from argument or `PROJECT-STATUS.md`
2. Load or create tracker file at `docs/technical/audit-reports/milestone-audit-{milestone}.json`
3. Detect available test tiers (unit, integration, hardware)
4. Record git commit hash

---

## Stage 1: Static Analysis (Lint)

### What Claude Must Say

```
STAGE 1: STATIC ANALYSIS (LINT)
══════════════════════════════════════════════════════════════════

Purpose: Run golangci-lint to catch code quality and security issues.

What this proves:
  ✓ No gosec security warnings
  ✓ No unchecked errors
  ✓ No obvious bugs (govet, staticcheck)
  ✓ Code follows style guidelines

What this does NOT prove:
  ✗ Logic is correct
  ✗ Business requirements are met
  ✗ Code is secure against sophisticated attacks

Running: golangci-lint run --timeout=5m ./...
```

### Commands

```bash
cd code/core
golangci-lint run --timeout=5m ./...
```

### Pass Criteria
- Zero gosec (G1xx) warnings in production code
- Zero errcheck violations in production code
- New warnings in milestone packages must be fixed

### Tracker Update
```json
"stage_1_lint": { "status": "complete", "passed": true, "issues_found": 0, "issues_fixed": 0 }
```

---

## Stage 2: Vulnerability Scan

### What Claude Must Say

```
STAGE 2: VULNERABILITY SCAN
══════════════════════════════════════════════════════════════════

Purpose: Check dependencies against Go vulnerability database.

What this proves:
  ✓ No known CVEs in direct dependencies
  ✓ No known CVEs in transitive dependencies
  ✓ Go stdlib version is not vulnerable

What this does NOT prove:
  ✗ Dependencies are bug-free
  ✗ Dependencies are maintained
  ✗ Zero-day vulnerabilities don't exist

Running: govulncheck ./...
```

### Commands

```bash
cd code/core
govulncheck ./...
```

### Pass Criteria
- Zero vulnerability findings (or documented accepted risks)

### Tracker Update
```json
"stage_2_vulncheck": { "status": "complete", "passed": true, "vulnerabilities": 0 }
```

---

## Stage 3: AI Code Review

### What Claude Must Say

```
STAGE 3: AI CODE REVIEW
══════════════════════════════════════════════════════════════════

Purpose: Deep semantic analysis for bugs, logic errors, and security issues
         that static tools cannot catch.

What this proves:
  ✓ No obvious logic errors in critical paths
  ✓ No security oversights (auth bypass, injection, etc.)
  ✓ Error handling is appropriate
  ✓ Concurrency patterns are safe

What this does NOT prove:
  ✗ Code is 100% bug-free (AI has limits)
  ✗ Business logic matches requirements
  ✗ Edge cases are all handled

Running code-reviewer on packages...
```

### Actions

Use the `code-reviewer` agent (or Task tool with code-reviewer subagent) to review:
- All packages in the milestone
- Focus areas: security, reliability, correctness
- Report only high-confidence (80%+) issues

### Issue Categories

| Category | Action Required |
|----------|-----------------|
| 🛑 Critical | Must fix before proceeding |
| 🔴 High | Must fix before milestone completion |
| 🟡 Medium | Should fix, document if not |
| 🟢 Low | Optional, note in report |

### Pass Criteria
- Zero Critical issues
- Zero High issues (or fixed during audit)

### Tracker Update
```json
"stage_3_ai_review": { 
  "status": "complete", 
  "passed": true, 
  "critical": 0, 
  "high": 0, 
  "medium": 2, 
  "low": 5,
  "issues_fixed": 3
}
```

---

## Stage 3.5: Protocol Compliance

### What Claude Must Say

```
STAGE 3.5: PROTOCOL COMPLIANCE
══════════════════════════════════════════════════════════════════

Purpose: Verify protocol implementations match official specifications.

What this proves:
  ✓ DPT encoding/decoding matches KNX Standard v3.0.0
  ✓ Address formats are spec-compliant
  ✓ APCI commands use correct binary codes
  ✓ Invalid value handling is implemented
  ✓ Test vectors from spec are included

What this does NOT prove:
  ✗ Interoperability with all KNX devices (needs real hardware)
  ✗ Edge cases in obscure DPTs are handled
  ✗ Performance under bus load

Reference: docs/protocols/knx-reference.md

Checking protocol compliance...
```

### Specification References

| Protocol | Reference Document | Key Sections |
|----------|-------------------|--------------|
| KNX | `docs/protocols/knx-reference.md` | DPT encoding, APCI commands, addressing |
| MQTT | `docs/protocols/mqtt.md` | Topic structure, QoS levels |

### KNX Compliance Checks

**1. DPT Encoding Verification**

Read the DPT implementation and verify against spec:

```bash
# Find DPT encoding functions
grep -rn "func.*[Ee]ncode.*DPT\|func.*[Dd]ecode.*DPT" code/core/internal/bridges/knx/
```

For each DPT implemented, verify:

| DPT | Spec Requirement | Verification |
|-----|------------------|--------------|
| 1.xxx | 1 bit, value 0 or 1 | Check no other values accepted |
| 3.xxx | 4 bits: B1U3 (direction + step) | Check step codes 0-7 |
| 5.001 | 8-bit scaled 0-100% (0x00-0xFF) | Check scaling formula |
| 5.004 | 8-bit linear 0-255% | Check NOT scaled |
| 9.xxx | KNX float: `(0.01 × M) × 2^E` | Check NOT IEEE 754 |
| 9.xxx | `0x7FFF` = invalid data | Check error returned |

**2. Address Validation**

```bash
# Find address parsing/validation
grep -rn "ParseGroupAddress\|ParseIndividualAddress\|ValidateAddress" code/core/internal/bridges/knx/
```

Verify ranges match spec:
- Individual: Area 0-15, Line 0-15, Device 0-255
- Group (3-level): Main 0-31, Sub 0-7, Group 0-255

**3. APCI Command Codes**

```bash
# Find APCI constants
grep -rn "APCI\|GroupValue_Read\|GroupValue_Write\|GroupValue_Response" code/core/internal/bridges/knx/
```

Verify codes match spec Table 1:
- `A_GroupValue_Read` = 0x0000
- `A_GroupValue_Response` = 0x0040
- `A_GroupValue_Write` = 0x0080

**4. Test Vector Verification**

Check that tests include spec-derived vectors:

```bash
# Find DPT test files
find code/core -name "*dpt*test*.go" -exec grep -l "TestVector\|testCase\|0x7FFF" {} \;
```

Required test vectors (from KNX Standard):
- DPT 9.001: Temperature encoding at 0°C, 20.48°C, -273°C
- DPT 9.xxx: Invalid marker `0x7FFF` handling
- DPT 5.001 vs 5.004: Different scaling verification

### Pass Criteria

| Check | Required |
|-------|----------|
| DPT encoding matches spec | Yes |
| Address validation enforces ranges | Yes |
| APCI codes are correct | Yes |
| Invalid data (0x7FFF) handled | Yes |
| At least 3 spec test vectors per DPT | Recommended |

### Issue Categories

| Issue | Severity | Example |
|-------|----------|---------|
| Wrong encoding formula | 🛑 Critical | DPT 9 using IEEE 754 instead of KNX float |
| Missing range validation | 🔴 High | Accepting group address 32/0/0 |
| Wrong APCI code | 🔴 High | Using 0x02 instead of 0x0080 for write |
| Missing invalid handling | 🟡 Medium | Not checking 0x7FFF |
| No spec test vectors | 🟡 Medium | Tests exist but not from spec |

### If Violations Found

1. Read the relevant section in `docs/protocols/knx-reference.md`
2. Fix the implementation to match spec
3. Add test vectors from the specification
4. Commit: `fix(knx): correct DPT X encoding per KNX Standard v3.0.0`

### Tracker Update

```json
"stage_3_5_protocol": { 
  "status": "complete", 
  "passed": true, 
  "protocols_checked": ["knx"],
  "dpts_verified": 6,
  "violations_found": 0,
  "violations_fixed": 0,
  "test_vectors_added": 12
}
```

---

## Stage 4: Architecture Review

### What Claude Must Say

```
STAGE 4: ARCHITECTURE REVIEW
══════════════════════════════════════════════════════════════════

Purpose: Verify implementation follows Gray Logic Hard Rules.

What this proves:
  ✓ Physical controls are never blocked by software
  ✓ Life safety systems are observe-only
  ✓ No cloud/internet dependencies for core features
  ✓ Open standards used (KNX, DALI, Modbus, MQTT)
  ✓ Customer owns their system (no dealer locks)
  ✓ Privacy by design (local processing)

What this does NOT prove:
  ✗ System will work for 20 years (needs production testing)
  ✗ All failure modes are handled

Checking Hard Rules...
```

### Hard Rules Checklist

| Rule | Verification Command | Pass Criteria |
|------|---------------------|---------------|
| No cloud calls | `grep -r "http\.Get\|http\.Post\|https://" internal/` | Zero matches in prod code |
| No life safety control | `grep -r "fire.*control\|alarm.*set\|estop" internal/` | Zero matches |
| No proprietary protocols | `grep -r "loxone\|crestron\|savant\|control4" -i` | Zero matches |
| Config is human-readable | Check config files are YAML | Confirmed |
| No dealer locks | Check for license/activation code | None found |

### Pass Criteria
- All Hard Rules verified
- Zero violations

### Tracker Update
```json
"stage_4_architecture": { 
  "status": "complete", 
  "passed": true, 
  "hard_rules_checked": 7, 
  "violations": 0 
}
```

---

## Stage 5: Dependency Audit

### What Claude Must Say

```
STAGE 5: DEPENDENCY AUDIT
══════════════════════════════════════════════════════════════════

Purpose: Assess dependencies for 20-year deployment viability.

What this proves:
  ✓ Dependencies have stable maintainers (foundations, companies)
  ✓ Dependencies are mature (5+ years old, stable API)
  ✓ Dependencies are widely adopted
  ✓ No single-developer abandonware

What this does NOT prove:
  ✗ Maintainers won't disappear
  ✗ APIs won't break in 20 years
  ✗ Security patches will always be available

Analyzing go.mod...
```

### Stability Criteria

| Criterion | Good | Risky |
|-----------|------|-------|
| Maintainer | Foundation/Company | Single developer |
| Age | 5+ years | <2 years |
| Usage | Widely adopted | Niche |
| API stability | Stable v1+ | Pre-1.0/frequent breaking |

### Pass Criteria
- No new high-risk dependencies
- All dependencies documented in report

### Tracker Update
```json
"stage_5_dependencies": { 
  "status": "complete", 
  "passed": true, 
  "total_deps": 12, 
  "high_risk": 0, 
  "medium_risk": 1 
}
```

---

## Stage 6: Coverage Sweep

### What Claude Must Say

```
STAGE 6: COVERAGE SWEEP
══════════════════════════════════════════════════════════════════

Purpose: Ensure test coverage meets tier targets.

What this proves:
  ✓ Code paths are exercised by tests
  ✓ Tests exist for critical functionality
  ✓ Regressions will likely be caught

What this does NOT prove:
  ✗ Tests are correct (tests verify code does what code does)
  ✗ All edge cases are covered
  ✗ Business logic is correct
  ✗ Code works in production

IMPORTANT: Coverage % ≠ Confidence %. A test that passes doesn't
mean the code is correct—it means the code behaves as written.

Running coverage analysis...
```

### Coverage Tiers

| Tier | Target | Packages |
|------|--------|----------|
| Critical | 85% | `internal/api`, `internal/device` |
| Infrastructure | 75% | `internal/infrastructure/*`, `internal/knxd` |
| Bridge | 50% (no HW) / 70% (with HW) | `internal/bridges/*` |
| Integration | 40% | `cmd/graylogic`, `internal/process` |

### Per-Package Process

1. Measure current coverage
2. Compare against tier target
3. If below target: identify uncovered functions, write tests
4. Commit per-package: `test(pkg): bring coverage to N% for milestone audit`

### Pass Criteria
- All packages meet tier targets (or exemptions documented)

### Tracker Update
```json
"stage_6_coverage": { 
  "status": "complete", 
  "passed": true, 
  "packages_audited": 11,
  "packages_passed": 11,
  "overall_coverage": 67.3,
  "tests_written": 47
}
```

---

## Stage 7: Integration Tests

### What Claude Must Say

```
STAGE 7: INTEGRATION TESTS
══════════════════════════════════════════════════════════════════

Purpose: Verify components work together correctly.

What this proves:
  ✓ Packages integrate without errors
  ✓ No race conditions (with -race flag)
  ✓ Real infrastructure connections work (MQTT, InfluxDB, etc.)

What this does NOT prove:
  ✗ End-to-end system works (needs real hardware)
  ✗ Performance is acceptable
  ✗ System handles failure gracefully

Running: go test -tags=integration -race ./...
```

### Commands

```bash
cd code/core
# Run with race detector
go test -race ./...

# Run integration-tagged tests if available
go test -tags=integration -race ./...

# Run hardware tests if available
go test -tags=hardware ./internal/bridges/knx/...
```

### Pass Criteria
- All tests pass
- No race conditions detected

### Tracker Update
```json
"stage_7_integration": { 
  "status": "complete", 
  "passed": true, 
  "suites_run": 14, 
  "tests_passed": 156, 
  "race_conditions": 0 
}
```

---

## Stage 8: Documentation Sync

### What Claude Must Say

```
STAGE 8: DOCUMENTATION SYNC
══════════════════════════════════════════════════════════════════

Purpose: Verify documentation matches implementation.

What this proves:
  ✓ PROJECT-STATUS.md reflects actual state
  ✓ API documentation matches endpoints
  ✓ Config documentation matches actual fields
  ✓ Code examples in docs compile

What this does NOT prove:
  ✗ Documentation is complete
  ✗ Documentation is easy to understand

Checking doc sync...
```

### Files to Check

| Document | Verification |
|----------|--------------|
| `PROJECT-STATUS.md` | Task statuses match reality |
| `docs/protocols/mqtt.md` | Topics match `topics.go` |
| `configs/config.yaml` | Comments match behavior |
| Code examples | They compile |

### Pass Criteria
- No critical mismatches
- PROJECT-STATUS.md is accurate

### Tracker Update
```json
"stage_8_docs": { 
  "status": "complete", 
  "passed": true, 
  "files_checked": 5, 
  "mismatches": 0 
}
```

---

## Stage 9: Technical Debt

### What Claude Must Say

```
STAGE 9: TECHNICAL DEBT SCAN
══════════════════════════════════════════════════════════════════

Purpose: Track and assess accumulated technical debt.

What this proves:
  ✓ Debt is documented and tracked
  ✓ No critical FIXMEs blocking release
  ✓ HACKs are minimal or justified

What this does NOT prove:
  ✗ All debt is known
  ✗ Debt won't cause future problems

Scanning for TODO/FIXME/HACK...
```

### Commands

```bash
cd code/core
echo "=== TODO ===" && grep -rn "TODO" --include="*.go" | grep -v "_test.go" | grep -v "vendor/"
echo "=== FIXME ===" && grep -rn "FIXME" --include="*.go" | grep -v "_test.go"
echo "=== HACK ===" && grep -rn "HACK\|XXX" --include="*.go" | grep -v "_test.go"
```

### Debt Thresholds

| Category | Acceptable | Warning | Blocking |
|----------|-----------|---------|----------|
| TODO | Any count | — | — |
| FIXME | < 5 | 5-10 | > 10 |
| HACK/XXX | 0 | 1-2 | > 2 |

### Pass Criteria
- FIXMEs below blocking threshold
- HACKs below blocking threshold or justified

### Tracker Update
```json
"stage_9_debt": { 
  "status": "complete", 
  "passed": true, 
  "todos": 12, 
  "fixmes": 2, 
  "hacks": 0 
}
```

---

## Stage 10: Final Report

### What Claude Must Say

```
STAGE 10: FINAL REPORT
══════════════════════════════════════════════════════════════════

Generating milestone audit report...

Report: docs/technical/audit-reports/milestone-audit-{milestone}-report.md
Tracker: docs/technical/audit-reports/milestone-audit-{milestone}.json
```

### Report Structure

The report MUST include:

1. **Executive Summary** — One paragraph verdict
2. **Stages Summary Table** — All 10 stages with pass/fail
3. **What Was Proven** — Honest list of validations
4. **What Was NOT Proven** — Honest list of gaps
5. **Issues Found and Fixed** — During the audit
6. **Coverage Results** — Per-package table
7. **Technical Debt** — Current state
8. **Recommendations** — For next milestone
9. **Conclusion** — READY / BLOCKED verdict

### Verdict Criteria

| Verdict | Criteria |
|---------|----------|
| ✅ READY | All stages pass, no Critical/High issues |
| ⚠️ READY WITH NOTES | All stages pass, Medium issues documented |
| ❌ BLOCKED | Any stage failed, Critical/High issues unfixed |

---

## Tracker File Schema

```json
{
  "milestone": "Y1-Foundation",
  "started": "2026-01-25T10:00:00Z",
  "completed": "2026-01-25T15:00:00Z",
  "status": "complete",
  "git_commit_start": "abc123",
  "git_commit_end": "def456",
  "stages": {
    "stage_0_setup": { "status": "complete", "notes": "" },
    "stage_1_lint": { "status": "complete", "passed": true, "issues": 0 },
    "stage_2_vulncheck": { "status": "complete", "passed": true, "vulnerabilities": 0 },
    "stage_3_ai_review": { "status": "complete", "passed": true, "critical": 0, "high": 0 },
    "stage_4_architecture": { "status": "complete", "passed": true, "violations": 0 },
    "stage_5_dependencies": { "status": "complete", "passed": true, "high_risk": 0 },
    "stage_6_coverage": { "status": "complete", "passed": true, "overall": 67.3 },
    "stage_7_integration": { "status": "complete", "passed": true, "race_conditions": 0 },
    "stage_8_docs": { "status": "complete", "passed": true, "mismatches": 0 },
    "stage_9_debt": { "status": "complete", "passed": true, "fixmes": 2, "hacks": 0 },
    "stage_10_report": { "status": "complete", "report_path": "..." }
  },
  "what_was_proven": [
    "Code compiles and tests pass",
    "No known CVEs in dependencies",
    "..."
  ],
  "what_was_not_proven": [
    "End-to-end with real KNX devices",
    "..."
  ]
}
```

---

## Resumability

The tracker file is the source of truth. On resume:

1. Read tracker
2. Find first incomplete stage
3. Re-verify previous stages still pass
4. Continue from incomplete stage

### What Claude Should Say When Resuming

```
╔══════════════════════════════════════════════════════════════════╗
║  MILESTONE AUDIT: [Milestone] — RESUMING                         ║
╚══════════════════════════════════════════════════════════════════╝

Previous progress:
  ✅ Stage 1: Lint — PASS
  ✅ Stage 2: Vulncheck — PASS
  ✅ Stage 3: AI Review — PASS
  ⏳ Stage 4: Architecture — IN PROGRESS

Resuming from Stage 4...
```

---

## Commit Strategy

| Stage | Commit Message |
|-------|----------------|
| Stage 1 fixes | `fix(lint): resolve golangci-lint issues for milestone audit` |
| Stage 3 fixes | `fix(security): address AI review findings for milestone audit` |
| Stage 6 tests | `test(pkg): bring coverage to N% for milestone audit` |
| Final report | `docs(audit): milestone audit {milestone} report — READY/BLOCKED` |

---

## Related Commands

- `/code-audit` — Per-package bug-finding (run during development)
- `/pre-commit` — Quick lint + test gates (before every commit)
- `/health-check` — Development environment verification
