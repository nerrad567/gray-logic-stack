---
description: Comprehensive code audit for stability, security, and project fitness
---

# Code Audit — Rock Solid Verification

A comprehensive, iterative audit to ensure Gray Logic Core is production-quality, secure, and aligned with project principles.

**Run this after any significant code changes or before milestones.**

---

## Overview

This audit performs 7 verification phases:

| Phase | Tool | Purpose |
|-------|------|---------|
| 1. Tests | `go test -race -cover` | Correctness, race conditions, coverage |
| 2. Lint | `golangci-lint` | Code quality, style, security (gosec) |
| 3. Vulnerabilities | `govulncheck` | Known CVEs in dependencies |
| 4. AI Code Review | `code-reviewer` plugin | Bugs, logic errors, security issues |
| 5. Architecture | Manual review | Alignment with Hard Rules |
| 6. Dependencies | `go.mod` analysis | 20-year stability assessment |
| 7. Documentation | Sync check | Docs match implementation |

---

## Phase 1: Tests with Race Detection & Coverage

Run all tests with the race detector and coverage reporting.

```bash
cd code/core
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# Full test suite with race detection and coverage
go test -race -cover ./...

# For verbose output with individual test names:
go test -race -cover -v ./...
```

### Coverage Targets

| Package Type | Target | Minimum |
|--------------|--------|---------|
| Infrastructure packages | 80%+ | 70% |
| Security-critical code | 100% | 90% |
| Integration code (main.go) | 50%+ | 30% |

### What Fails This Phase

- Any test failure
- Race condition detected
- Coverage below minimum thresholds

---

## Phase 2: Lint Analysis

Run the full golangci-lint suite with all configured linters.

```bash
cd code/core
golangci-lint run --timeout=5m
```

### Key Linters

| Linter | Purpose | Severity |
|--------|---------|----------|
| **gosec** | Security vulnerabilities | Critical |
| **errcheck** | Unchecked errors | High |
| **govet** | Subtle bugs | High |
| **staticcheck** | Static analysis | High |
| **misspell** | UK English spelling | Medium |
| **mnd** | Magic numbers | Medium |

### What Fails This Phase

- Any gosec (G1xx) security warning in production code
- Unchecked errors in production code
- UK spelling violations (`unauthorized` → `unauthorised`)

### Exceptions

- M1.2+ code may have warnings if still in development
- Test files are excluded from some linters

---

## Phase 3: Vulnerability Scan

Scan dependencies against the Go vulnerability database.

```bash
cd code/core
govulncheck ./...
```

### What This Checks

- Known CVEs in direct dependencies
- Known CVEs in transitive dependencies
- Vulnerabilities in the Go stdlib version

### What Fails This Phase

- **Any vulnerability finding** — must be addressed or documented as accepted risk

---

## Phase 4: AI Code Review (CRITICAL)

Use the code-reviewer plugin for deep analysis that static tools miss.

**This is the most important phase** — it catches semantic issues like:
- Missing validation that comments promised
- Incorrect shutdown ordering
- Logic errors in concurrent code
- Security oversights

### How to Run

Ask Claude to run the code-reviewer:

```
Run the code-reviewer plugin on the infrastructure packages and main.go.
Focus on security, reliability, and correctness for a building automation system.
Report only high-confidence (80%+) issues.
```

### What the Plugin Reviews

| Category | Examples |
|----------|----------|
| Security | Auth bypass, injection, secrets exposure |
| Resource leaks | Unclosed connections, goroutine leaks |
| Error handling | Missing checks, panic risks |
| Concurrency | Race conditions, deadlocks |
| Reliability | Single points of failure, missing timeouts |

### What Fails This Phase

- Any Critical severity finding
- Any High severity finding affecting security or data integrity

---

## Phase 5: Architecture Review

Verify implementation aligns with Gray Logic's Hard Rules.

### Hard Rules Checklist

| Rule | Question to Verify |
|------|-------------------|
| **1. Physical Controls** | Does the code ever block or replace physical control signals? |
| **2. Life Safety** | Is there any code that controls (not just reads) fire/E-stop systems? |
| **3. No Cloud** | Are there any external HTTP calls, cloud SDKs, or internet dependencies? |
| **4. Multi-Decade** | Are all dependencies mature and likely to exist in 2046? |
| **5. Open Standards** | Does new code use KNX/DALI/Modbus/MQTT or proprietary protocols? |
| **6. Customer Owns** | Is configuration human-readable YAML? Any dealer locks? |
| **7. Privacy** | Is all processing local? Any external data transmission? |

### Code Patterns to Check

```bash
# Check for cloud/external calls
grep -r "http\.Get\|http\.Post\|https://" code/core/internal/ --include="*.go" | grep -v "_test.go"

# Check for life safety control
grep -r "fire.*control\|alarm.*set\|estop.*trigger\|emergency.*disable" code/core/ --include="*.go"

# Check for proprietary protocols
grep -r "loxone\|crestron\|savant\|control4" code/core/ --include="*.go" -i
```

### What Fails This Phase

- **Any Hard Rule violation** — these are absolute constraints

---

## Phase 6: Dependency Stability

Assess dependencies for 20-year viability.

```bash
cd code/core
cat go.mod
```

### Stability Criteria

| Criterion | Good | Risky |
|-----------|------|-------|
| Maintainer | Foundation/Company | Single developer |
| Age | 5+ years | <2 years |
| Usage | Widely adopted | Niche |
| API stability | Stable v1+ | Pre-1.0/frequent breaking |
| Standard | Implements RFC/standard | Proprietary |

### Current Dependencies Assessment

| Dependency | Maintainer | Risk Level |
|------------|------------|------------|
| `gopkg.in/yaml.v3` | Community | 🟢 Very Low |
| `paho.mqtt.golang` | Eclipse Foundation | 🟢 Very Low |
| `go-sqlite3` | Community (mattn) | 🟢 Very Low |
| `influxdb-client-go` | InfluxData | 🟢 Low |
| `gorilla/websocket` | Community | 🟢 Very Low |

### What Fails This Phase

- New dependency with single-developer maintenance
- Pre-1.0 dependency without justification
- Cloud-only or proprietary SDK

---

## Phase 7: Documentation Sync

Verify documentation matches implementation.

### Files to Check

| Document | Verify |
|----------|--------|
| `code/core/IMPLEMENTATION.md` | Task statuses match reality |
| `code/core/AGENTS.md` | Code examples compile/work |
| `docs/CONSTRAINTS.md` | Rules match implementation |
| `docs/protocols/mqtt.md` | Topics match `topics.go` |
| `configs/config.yaml` | Comments match actual behaviour |
| Package `doc.go` files | Descriptions accurate |

### Quick Sync Check

```bash
# Check MQTT topics match spec
grep -o "graylogic/[^\"]*" code/core/internal/infrastructure/mqtt/topics.go | sort -u

# Check config fields have docs
grep -E "^\s+\w+:" code/core/configs/config.yaml | wc -l
```

### What Fails This Phase

- Implementation status mismatches in IMPLEMENTATION.md
- Code examples in docs that don't compile
- Undocumented config fields

---

## Running the Full Audit

### Quick Audit (5 minutes)

Phases 1-3 only — for routine checks:

```bash
cd code/core
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
go test -race ./... && golangci-lint run && govulncheck ./...
```

### Full Audit (15-30 minutes)

All 7 phases — for milestone verification:

1. Run the quick audit (Phases 1-3)
2. Ask Claude to run code-reviewer plugin (Phase 4)
3. Review Hard Rules checklist manually (Phase 5)
4. Check go.mod for new dependencies (Phase 6)
5. Spot-check documentation sync (Phase 7)

### Audit Prompt for Claude

Copy this prompt to run a full AI-assisted audit:

```
Run a comprehensive code audit on Gray Logic Core:

1. Run tests with race detection: `go test -race -cover ./...`
2. Run linter: `golangci-lint run`
3. Run vulnerability scan: `govulncheck ./...`
4. Use the code-reviewer plugin on all infrastructure packages
5. Check architecture against docs/overview/principles.md Hard Rules
6. Assess go.mod dependencies for 20-year stability
7. Check IMPLEMENTATION.md is up to date

Report findings by severity and recommend fixes.
```

---

## Audit Report Template

After completing the audit, document findings:

```markdown
# Code Audit Report — [DATE]

## Summary
- **Tests:** ✅/❌ [X tests, Y% coverage]
- **Lint:** ✅/❌ [N issues]
- **Vulnerabilities:** ✅/❌ [N findings]
- **AI Review:** ✅/❌ [N issues]
- **Architecture:** ✅/❌ [Compliant/Violations]
- **Dependencies:** ✅/❌ [Stable/Concerns]
- **Documentation:** ✅/❌ [Synced/Drift]

## Issues Found
| Severity | File | Issue | Status |
|----------|------|-------|--------|
| Critical | ... | ... | Fixed/Open |

## Commits Made
- `abc123` fix(security): description
- `def456` fix(lint): description

## Verdict
[READY / BLOCKED — reason]
```

---

## When to Run This Audit

| Trigger | Audit Type |
|---------|-----------|
| Before any commit | Quick (Phases 1-3) via `/pre-commit` |
| After significant changes | Full (All phases) |
| Before milestone completion | Full + documented report |
| After dependency updates | Full with extra Phase 6 focus |
| After security-related changes | Full with extra Phase 4 focus |
| Weekly (during active development) | Full |

---

## Related Commands

- `/pre-commit` — Quick quality gates before committing
- `/check-principles` — Focused Hard Rules verification
- `/audit-deps` — Deep dependency analysis
- `/health-check` — Development environment verification
