---
description: End-of-milestone quality gate with coverage enforcement and integration testing
---

# Milestone Audit — End-of-Milestone Quality Gate

A comprehensive, resumable audit that sweeps all packages for coverage gaps, writes missing tests, runs cross-package integration tests, checks technical debt, and produces a milestone readiness report.

**Run this before marking any milestone as complete.**

---

## How It Differs from `/code-audit`

| Aspect | `/code-audit` | `/milestone-audit` |
|--------|--------------|-------------------|
| Scope | Single package (latest changes) | All packages, sequential |
| Focus | Bug-finding (correctness) | Coverage enforcement + integration + debt |
| Tests | Runs existing tests | Writes new tests to fill gaps |
| Context | Single session | Multi-session with tracker file |
| When | After each task / before commits | End of milestone (before sign-off) |
| Output | Fix issues, ship/fix verdict | Coverage report + milestone readiness |

They complement each other: run `/code-audit` during development, run `/milestone-audit` before marking a milestone complete.

---

## Step 0: Determine Milestone

1. Check if the user provided a milestone argument (e.g., `/milestone-audit M1.4`)
2. If not, read `PROJECT-STATUS.md` to determine the current milestone
3. Announce the milestone being audited

---

## Step 1: Load or Create Tracker

### Tracker File Location

```
code/core/docs/technical/audit-reports/milestone-audit-M{X}.{Y}.json
```

### Load Logic

1. Check if tracker file exists for this milestone
2. If exists AND `status == "in_progress"`:
   - **Resume**: Read the tracker, announce what's complete and what's pending
   - Re-verify actual state (re-run coverage on "complete" packages to confirm)
3. If not exists:
   - **Start fresh**: Create new tracker, begin from Phase 1

### Tracker File Schema

```json
{
  "milestone": "M1.4",
  "started": "2026-01-23T22:00:00Z",
  "completed": null,
  "status": "in_progress",
  "git_commit_start": "9497eb4",
  "available_tiers": ["unit", "integration"],
  "phases": {
    "lint": { "status": "complete", "passed": true, "issues": 0 },
    "vulncheck": { "status": "complete", "passed": true, "vulnerabilities": 0 },
    "coverage": { "status": "in_progress", "packages_complete": 5, "packages_total": 11 },
    "integration": { "status": "pending", "passed": null },
    "debt": { "status": "pending", "todos": null, "fixmes": null }
  },
  "packages": [
    {
      "path": "internal/infrastructure/config",
      "tier": "infrastructure",
      "target": 75,
      "coverage_before": 100.0,
      "coverage_after": 100.0,
      "status": "complete",
      "tests_written": 0,
      "functions_pending": [],
      "commit_sha": null,
      "notes": "Already exceeds target"
    }
  ],
  "debt_summary": {
    "todo_count": 0,
    "fixme_count": 0,
    "hack_count": 0,
    "items": []
  },
  "report_path": null
}
```

---

## Step 2: Detect Available Test Tiers

Before starting, determine which test tiers can be run:

```bash
cd code/core
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# Unit — always available
echo "unit: available"

# Integration — check Docker health
if docker compose -f ../../docker-compose.dev.yml ps --format '{{.Status}}' 2>/dev/null | grep -q "Up"; then
  echo "integration: available"
else
  echo "integration: unavailable (start with: docker compose -f docker-compose.dev.yml up -d)"
fi

# Hardware — check env var + knxd
if [ "$GRAYLOGIC_HARDWARE_TEST" = "true" ] && command -v knxd >/dev/null 2>&1; then
  echo "hardware: available"
else
  echo "hardware: unavailable"
fi

# Endurance — check env var
if [ "$GRAYLOGIC_ENDURANCE_TEST" = "true" ]; then
  echo "endurance: available"
else
  echo "endurance: unavailable"
fi
```

### Tier Definitions

| Tier | Build Tag | Requirements | Duration |
|------|-----------|--------------|----------|
| Unit | *(none)* | Nothing | <30s |
| Integration | `//go:build integration` | docker-compose.dev.yml healthy (MQTT + InfluxDB) | <2min |
| Hardware | `//go:build hardware` | Real KNX/DALI interface + physical bus | <5min |
| Endurance | `//go:build endurance` | Real hardware + extended time window | 24h+ |

### What Claude Should Say

```
Milestone Audit: M1.4 (REST API + WebSocket)
Status: Starting fresh

Available test tiers:
  ✅ Unit (always)
  ✅ Integration (docker-compose healthy)
  ❌ Hardware (no KNX interface configured)
  ❌ Endurance (not requested)

Running: unit + integration
Bridge coverage target: 50% (hardware unavailable)
```

---

## Phase 1: Lint Sweep

Run golangci-lint across the entire codebase (not just the milestone's package).

```bash
cd code/core
golangci-lint run --timeout=5m ./...
```

### What Fails This Phase

- Any gosec (security) warning in production code
- Any errcheck violation in production code
- New warnings in the milestone's package (pre-existing warnings in other packages are noted but not blocking)

### Tracker Update

```json
"lint": { "status": "complete", "passed": true, "issues": 0 }
```

---

## Phase 2: Vulnerability Check

```bash
cd code/core
govulncheck ./...
```

### What Fails This Phase

- Any vulnerability finding (must be addressed or accepted with rationale)

### Tracker Update

```json
"vulncheck": { "status": "complete", "passed": true, "vulnerabilities": 0 }
```

---

## Phase 3: Coverage Sweep (Main Phase)

This is the core of the milestone audit. Process each package in dependency order, writing tests where coverage is below target.

### Coverage Tiers

| Tier | Target (no hardware) | Target (with hardware) | Packages |
|------|------|------|------|
| **Critical** | 85%+ | 85%+ | `internal/api`, `internal/device` |
| **Infrastructure** | 75%+ | 75%+ | `internal/infrastructure/*`, `internal/knxd` |
| **Bridge** | 50%+ | 70%+ | `internal/bridges/knx`, future DALI/Modbus |
| **Integration** | 40%+ | 40%+ | `cmd/graylogic`, `internal/process` |
| **Exempt** | — | — | `migrations/`, `doc.go` files |

### Package Processing Order

Process in dependency order (foundations first):

| Order | Package | Tier | Target |
|-------|---------|------|--------|
| 1 | `internal/infrastructure/config` | Infrastructure | 75% |
| 2 | `internal/infrastructure/logging` | Infrastructure | 75% |
| 3 | `internal/infrastructure/database` | Infrastructure | 75% |
| 4 | `internal/infrastructure/mqtt` | Infrastructure | 75% |
| 5 | `internal/infrastructure/influxdb` | Infrastructure | 75% |
| 6 | `internal/device` | Critical | 85% |
| 7 | `internal/api` | Critical | 85% |
| 8 | `internal/process` | Integration | 40% |
| 9 | `internal/knxd` | Infrastructure | 75% |
| 10 | `internal/bridges/knx` | Bridge | 50%/70% |
| 11 | `cmd/graylogic` | Integration | 40% |

### Per-Package Procedure

For each package:

1. **Measure current coverage:**
   ```bash
   go test -coverprofile=/tmp/pkg-cover.out ./internal/path/to/pkg/...
   go tool cover -func=/tmp/pkg-cover.out
   ```

2. **Check against target:**
   - If coverage >= target → mark `status: "complete"`, record `coverage_after`, move to next
   - If coverage < target → continue to step 3

3. **Identify uncovered functions:**
   - Read the `go tool cover -func` output
   - List all functions at 0% or below the package's effective contribution to the target
   - Prioritise: security-critical > error handling > happy path > edge cases

4. **Write tests:**
   - Follow existing patterns (table-driven, `t.Run` subtests, interface mocks)
   - Standard library `testing` only (no testify)
   - Use existing mock types (MockRepository, MockMQTTClient, MockConnector) or create new ones following the same pattern
   - Name test files: `{source}_test.go` for unit, `integration_test.go` for integration-tagged
   - Use `t.Helper()` in helper functions, `t.Cleanup()` for teardown

5. **Verify tests pass:**
   ```bash
   go test -race -cover ./internal/path/to/pkg/...
   ```

6. **Record new coverage** in tracker

7. **Commit:**
   ```
   test(pkg): bring coverage to N% for milestone audit M{X}.{Y}
   ```

8. **Update tracker file** (write to disk)

### What Claude Should Say Per Package

```
Package: internal/api
  Tier: critical (target 85%)
  Current coverage: 44.6%
  Gap: 40.4% needed

  Uncovered functions (0%):
    - handleWebSocket (High risk — auth + upgrade)
    - readPump (High risk — client lifecycle)
    - writePump (High risk — client lifecycle)
    - Start (Medium risk — server lifecycle)
    - Close (Medium risk — graceful shutdown)
    - HealthCheck (Low risk — simple probe)

  Writing tests...
```

### Coverage-Exempt Functions

Some functions legitimately cannot be unit-tested. Mark these with a comment:

```go
// coverage:exempt — requires real KNX hardware
func (b *Bridge) handlePhysicalTelegram(telegram []byte) { ... }
```

Document exemptions in the tracker's `notes` field. These do NOT count against the coverage target.

### Important: Don't Over-Test

- Do NOT write tests just to hit a number
- Skip trivial getters/setters that have no logic
- Focus on functions that handle user input, state changes, error paths, and concurrency
- If a function is genuinely trivial (< 3 lines, no branching), it's acceptable to leave uncovered

---

## Phase 4: Integration Tests

If the integration tier is available, run all integration-tagged tests:

```bash
cd code/core

# Run integration tests
go test -tags=integration -race -cover -v ./... 2>&1

# If hardware tier is available
go test -tags=hardware -v ./internal/bridges/knx/... 2>&1
```

### What to Verify

| Suite | Checks |
|-------|--------|
| MQTT integration | Connect, publish, subscribe, reconnect |
| KNX bridge integration | Full command cycle: MQTT → KNX → state update |
| Device registry integration | Full lifecycle with real SQLite |
| API integration | Real HTTP server, real endpoints, WebSocket upgrade |

### What Fails This Phase

- Any integration test failure
- Race condition detected

### Tracker Update

```json
"integration": { "status": "complete", "passed": true, "suites": 4, "tests": 23 }
```

---

## Phase 5: Technical Debt Scan

Scan the entire codebase for accumulated technical debt:

```bash
cd code/core

# Count TODOs, FIXMEs, HACKs, XXXs
echo "=== TODO ===" && grep -rn "TODO" --include="*.go" | grep -v "_test.go" | grep -v "vendor/"
echo "=== FIXME ===" && grep -rn "FIXME" --include="*.go" | grep -v "_test.go"
echo "=== HACK ===" && grep -rn "HACK\|XXX" --include="*.go" | grep -v "_test.go"
```

### What to Record

For each TODO/FIXME found:
- File and line number
- Category (TODO/FIXME/HACK)
- Milestone it was introduced (if identifiable from git blame)
- Whether it's blocking or informational

### Debt Thresholds

| Category | Acceptable | Warning | Blocking |
|----------|-----------|---------|----------|
| TODO | Any count | — | — |
| FIXME | < 5 | 5-10 | > 10 |
| HACK/XXX | 0 | 1-2 | > 2 |

### Tracker Update

```json
"debt": {
  "status": "complete",
  "todos": 12,
  "fixmes": 2,
  "hacks": 0,
  "items": [
    {"file": "internal/api/middleware.go:106", "type": "TODO", "text": "TODO(M1.6): Full JWT validation", "blocking": false}
  ]
}
```

---

## Phase 6: Milestone Readiness Report

Generate the final report at:

```
code/core/docs/technical/audit-reports/milestone-audit-M{X}.{Y}-report.md
```

### Report Structure

```markdown
---
title: Milestone Audit Report — M{X}.{Y}
version: 1.0.0
status: complete
audit_date: YYYY-MM-DD
milestone: M{X}.{Y}
auditor: Claude Code (model)
---

# Milestone Audit Report — M{X}.{Y}

## Executive Summary

[One paragraph: what the milestone delivered, overall coverage improvement, verdict]

## Test Tier Availability

| Tier | Status | Notes |
|------|--------|-------|
| Unit | ✅ Available | Always |
| Integration | ✅/❌ | Docker status |
| Hardware | ✅/❌ | KNX interface status |
| Endurance | ❌ | Not requested |

## Coverage Results

| Package | Tier | Target | Before | After | Tests Written | Status |
|---------|------|--------|--------|-------|---------------|--------|
| internal/infrastructure/config | Infrastructure | 75% | 100% | 100% | 0 | ✅ |
| internal/api | Critical | 85% | 44.6% | N% | N | ✅/❌ |
| ... | ... | ... | ... | ... | ... | ... |

### Overall Statistics

- Packages audited: N
- Packages already meeting target: N
- Packages brought up to target: N
- Total tests written: N
- Overall coverage: N% (was N%)

## Integration Test Results

| Suite | Tests | Pass | Fail | Skip | Notes |
|-------|-------|------|------|------|-------|
| mqtt | N | N | 0 | 0 | ... |
| knx bridge | N | N | 0 | 0 | ... |
| api | N | N | 0 | 0 | ... |

## Technical Debt

| Category | Count | Delta from Previous Milestone |
|----------|-------|-------------------------------|
| TODO | N | +N |
| FIXME | N | +N |
| HACK | N | +N |

### Notable Items

[List significant TODOs that should be addressed in next milestone]

## Static Analysis

- Lint: PASS/FAIL (N new issues)
- Vulncheck: PASS/FAIL (N vulnerabilities)

## Verdict

**✅ READY FOR MILESTONE COMPLETION** / **❌ BLOCKED**

[If BLOCKED: list what must be resolved before milestone sign-off]
[If READY: confirm milestone can be marked complete in PROJECT-STATUS.md]
```

### Update Tracker

Mark the tracker as complete:

```json
{
  "status": "complete",
  "completed": "2026-01-23T23:00:00Z",
  "report_path": "code/core/docs/technical/audit-reports/milestone-audit-M1.4-report.md"
}
```

---

## Resumability

### How It Works

The tracker file is the source of truth. On each invocation:

1. **Read tracker** — determine what's done and what's pending
2. **Re-verify** — re-run `go test -cover` on "complete" packages to confirm actual state
3. **Continue** — pick up from the first incomplete phase/package
4. **Commit incrementally** — each package gets its own commit
5. **Update tracker** — write after each package is done

### What Claude Should Say When Resuming

```
Milestone Audit: M1.4 (REST API + WebSocket)
Status: Resuming from Phase 3 (Coverage) — internal/api
Available tiers: unit, integration

Phases completed:
  ✅ Phase 1 (lint): PASS
  ✅ Phase 2 (vulncheck): PASS
  ⏳ Phase 3 (coverage): 5/11 packages complete

Verified coverage of completed packages:
  config: 100% ✅ | logging: 100% ✅ | database: 82.7% ✅
  mqtt: 81.8% ✅ | influxdb: 81.8% ✅

Next package: internal/device (critical tier, target 85%, current 80.1%)
```

### Key Principle: Never Trust the Log Blindly

Always re-run coverage to verify the tracker's claims. This prevents:
- Stale data from uncommitted changes
- Hallucinated coverage numbers from previous context
- Regressions from other changes between sessions

---

## Commit Strategy

### Per-Package Test Commits

```
test(config): bring coverage to 100% for milestone audit M1.4
test(api): bring coverage to 87% for milestone audit M1.4
test(knx): add integration tests for milestone audit M1.4
```

### Final Report Commit

```
docs(audit): milestone audit M1.4 report — READY
```

### Rules

- One commit per package that receives new tests
- Do NOT batch multiple packages into one commit
- Always run `go test -race ./...` before committing (full suite, not just the package)
- Include the tracker file update in the package commit
- The final report is a separate commit

---

## What Claude Should Say at Completion

```
Milestone Audit Complete: M1.4

Coverage Results:
  11/11 packages meet targets
  Overall: 78.3% (was 62.1%)
  Tests written: 47

Integration Tests: PASS (4 suites, 23 tests)
Technical Debt: 12 TODOs (3 new since M1.3), 0 FIXMEs, 0 HACKs
Lint: PASS (0 new issues)
Vulncheck: PASS (0 vulnerabilities)

Verdict: ✅ READY FOR MILESTONE COMPLETION

Report: code/core/docs/technical/audit-reports/milestone-audit-M1.4-report.md
Tracker: code/core/docs/technical/audit-reports/milestone-audit-M1.4.json
```

---

## When to Run This Audit

| Trigger | Action |
|---------|--------|
| Before marking milestone complete | Run full milestone audit |
| After significant refactoring | Run coverage sweep (Phase 3 only) |
| Before release candidate | Run with hardware + endurance tiers |
| After dependency updates | Run vulncheck + coverage sweep |

---

## Related Commands

- `/code-audit` — Per-package bug-finding (run during development)
- `/pre-commit` — Quick lint + test gates (before every commit)
- `/health-check` — Development environment verification

> **Reference**: See `code/core/docs/technical/testing-strategy.md` for full testing philosophy,
> patterns, and tier documentation.
