---
description: Comprehensive code audit for stability, security, and project fitness
---

# Code Audit — Rock Solid Verification

A comprehensive, iterative audit to ensure Gray Logic Core is production-quality, secure, and aligned with project principles.

**Run this after any significant code changes or before milestones.**

---

## IMPORTANT: Automatic Audit Tracking

**Claude MUST track audit runs automatically.** Before starting any audit:

### Step 0: Check and Update Audit History

1. **Read the tracking file** (create if missing):
   ```
   .claude/audit-history.json
   ```

2. **Check if code has changed** since last audit:
   ```bash
   git rev-parse HEAD
   ```
   Compare with `last_commit` in tracking file.

3. **Determine audit mode**:
   - If `last_commit` differs from current HEAD → **Reset count to 1, run Standard mode**
   - If the previous audit fixed any Critical or High issues → **Reset count to 1, run Standard mode** (code changed!)
   - If count is 1 or 2 AND previous audit found 0 Critical/High issues → **Increment count, run Standard mode**
   - If count is 3+ AND previous audit found 0 Critical/High issues → **Run Final Advisory mode automatically**
   - **Key rule:** Final Advisory mode is ONLY allowed after a clean standard audit (zero Critical/High findings). Low/Medium observations that don't require code changes do NOT reset the count.

4. **Update the tracking file** with new count and commit hash.

5. **Create/Update Audit Reports** (IMPORTANT):
   - Create a detailed markdown report at:
     ```
     code/core/docs/technical/audit-reports/audit-YYYY-MM-DD-[scope].md
     ```
   - Update the summary file at:
     ```
     code/core/docs/technical/audit-reports/audit-summary.md
     ```
   - Add a new row to the "Phase 2: Code Audits" table in the summary

### Audit Report Location

All audit reports are stored in `code/core/docs/technical/audit-reports/`:
- `audit-summary.md` — Master index of all audits
- `audit-YYYY-MM-DD-[scope].md` — Individual audit reports
- `audit-iteration-N-log.md` — Legacy documentation-phase audits

### Tracking File Format

```json
{
  "last_commit": "abc123def",
  "audit_count": 2,
  "last_audit": "2026-01-20T14:30:00Z",
  "audits": [
    {"date": "2026-01-20T10:00:00Z", "commit": "abc123def", "mode": "standard", "issues_fixed": 2},
    {"date": "2026-01-20T12:00:00Z", "commit": "abc123def", "mode": "standard", "issues_fixed": 4}
  ]
}
```

### What Claude Should Say

**On runs 1-2:**
```
📋 Audit Run #1 (Standard Mode)
Code: abc123d | Last audit: Never
Running full 7-phase audit...
```

**On run 3+ (only if previous run was clean):**
```
📋 Audit Run #3 (Final Advisory Mode)
Code: abc123d | Previous audits: 2 (0 issues on last run)

This is your 3rd audit on unchanged code with a clean previous run.
Switching to Final Advisory mode.
I'll tell you honestly whether findings are worth fixing or if you should ship.
```

**When code changes (or previous audit fixed issues):**
```
📋 Audit Run #1 (Standard Mode) — Reset
Code changed: def456g (was abc123d)
Starting fresh audit cycle...
```

---

## Audit Modes

| Mode | Triggered | Behaviour |
|------|-----------|-----------|
| **Standard** | Any run where previous audit found Critical/High issues, OR runs 1-2 on clean code | Find and fix issues |
| **Final Advisory** | Run 3+ on same code AND previous run found 0 Critical/High issues | Honest ship/fix recommendation |

**Key principle:** Final Advisory is earned, not automatic. The code must pass a clean standard audit (zero Critical/High findings) before advisory mode is unlocked. Low/Medium observations that don't require code changes are acceptable and don't reset the count.

**Important Reality Check**: AI code reviewers will *always* find something if you keep asking. There is no such thing as perfect code. The goal is not zero findings — the goal is **code that is safe, reliable, and maintainable**.

---

## Overview

This audit performs 7 verification phases:

| Phase | Tool | Purpose |
|-------|------|---------|
| 1. Tests | `go test -race -cover` | Correctness, race conditions, coverage |
| 2. Lint | `golangci-lint` | Code quality, style, security (gosec) |
| 3. Vulnerabilities | `govulncheck` | Known CVEs in dependencies |
| 4. AI Code Review | `code-reviewer` plugin | Bugs, logic errors, security issues |
| 4b. Red Team Spot Check | Adversarial checklist | Auth bypass, privilege escalation, input abuse |
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

## Phase 4b: Red Team Spot Check (Optional but Recommended)

After the defensive AI review, run a targeted adversarial check on security-sensitive code.

This is a lightweight version of `/red-team` — focused on the packages being audited rather than the full system.

### Quick Red Team Checklist

For each audited package, ask:

| Check | Question |
|-------|----------|
| **Auth bypass** | Can any endpoint be reached without valid credentials? |
| **Privilege escalation** | Can a lower-role user trigger higher-role actions? |
| **Input abuse** | Can oversized, malformed, or malicious input cause unexpected behaviour? |
| **Race conditions** | Can concurrent requests create inconsistent state? |
| **Information leakage** | Do error responses reveal internal details? |

### When to Run Full /red-team

- If Phase 4b finds any Medium+ issue → run full `/red-team` on the affected package
- At milestone boundaries → run `/red-team` on all security-critical packages
- After any auth-related changes → run `/red-team internal/auth/ internal/api/`

### What Fails This Sub-Phase
- Any finding that would allow unauthenticated access to protected resources
- Any finding that allows privilege escalation

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
| `net/http` (stdlib) | Go standard library | 🟢 Low |
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
| `PROJECT-STATUS.md` | Task statuses match reality |
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

- Implementation status mismatches in PROJECT-STATUS.md
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
7. Check PROJECT-STATUS.md is up to date

Report findings by severity and recommend fixes.
```

---

## Audit Report Requirements

**Claude MUST create a proper audit report for every audit run.**

### Report File

Create at: `code/core/docs/technical/audit-reports/audit-YYYY-MM-DD-[scope].md`

Example: `audit-2026-01-22-m1.2-knxd.md`

### Required Report Structure

```markdown
---
title: Code Audit Report — [Component/Milestone]
version: 1.0.0
status: complete
audit_date: YYYY-MM-DD
auditor: Claude Code (model)
scope: [What was audited]
previous_audit: [Link to previous report]
commit: [Git commit hash]
---

# Code Audit Report — [Component]

**Audit Date:** YYYY-MM-DD
**Auditor:** Claude Code
**Scope:** [Detailed scope description]
**Packages Reviewed:**
- [List of packages]

---

## Executive Summary

[Brief overview of audit progression and results]

### Audit Progression

| Run | Mode | Issues Found | Issues Fixed | Notes |
|-----|------|--------------|--------------|-------|
| 1 | Standard | N | N | [Key finding] |

### Readiness Score

| Category | Score | Notes |
|----------|-------|-------|
| Tests | X/10 | ... |
| Security | X/10 | ... |
| Reliability | X/10 | ... |

**Overall Readiness: X.X/10**

---

## Phase Results

[Detail each of the 7 phases with pass/fail and evidence]

---

## Issues Fixed

### Critical Severity

#### C1: [Issue Title] — [FIXED]

| Attribute | Value |
|-----------|-------|
| **File** | `path/to/file.go:line` |
| **Confidence** | X% |
| **Issue** | [Description] |
| **Impact** | [Why it matters] |
| **Fix** | [What was done] |

[Repeat for each issue by severity: Critical, High, Medium, Low]

---

## False Positives (Withdrawn)

| Finding | Reason Withdrawn |
|---------|------------------|
| ... | ... |

---

## Remaining Warnings (Accepted)

[List accepted warnings with rationale]

---

## Recommendations

### Immediate
[Before commit]

### Short-Term
[Next milestone]

### Long-Term
[Future work]

---

## Conclusion

[Final assessment and verdict]

**Verdict: ✅ SHIP IT / ❌ BLOCKED**
```

### Update Summary File

After creating the report, update `code/core/docs/technical/audit-reports/audit-summary.md`:

1. Update the `last_updated` date in frontmatter
2. Add a new row to the "Phase 2: Code Audits" table
3. Add any new critical issues to the "Key Findings" section
4. Update the "Current Readiness Score" if changed

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
| After 2-3 clean audit rounds | **Final advisory mode** |

---

## Final Advisory Mode (`/code-audit final`)

Use this mode when you've already done multiple audit rounds and want realistic guidance on whether to ship or keep fixing.

### What Changes in Final Mode

| Aspect | Standard Mode | Final Advisory Mode |
|--------|---------------|---------------------|
| Tone | "Fix these issues" | "Here's what I found — here's whether it matters" |
| AI Review | Lists all findings | Categorises as "Must Fix" vs "Can Ship With" |
| Verdict | READY/BLOCKED | Ship recommendation with rationale |
| Output | Issue list | Decision framework |

### Final Mode Instructions for Claude

When running `/code-audit final`, Claude should:

1. **Run Phases 1-3 as normal** (tests, lint, vulncheck)
   - If any of these fail → **BLOCKED, must fix**

2. **Run Phase 4 (AI Review) with advisory framing**:
   ```
   Review the infrastructure packages with a "final audit" mindset:

   For each finding, categorise it as:
   - 🛑 MUST FIX: Real bugs, security holes, data loss risks, crashes
   - 🟡 CONSIDER: Defensive improvements that add measurable safety
   - 🟢 CAN SHIP: Theoretical issues, style preferences, over-engineering suggestions

   Remember: This code has already passed 2-3 audit rounds. Be honest about
   whether findings are genuine risks or just "things that could be different".
   ```

3. **Skip Phases 5-7** (already verified in previous audits)

4. **Provide a Ship/Fix Verdict**:
   ```markdown
   ## Final Verdict

   ### Static Tools: [PASS/FAIL]
   - Tests: ✅/❌
   - Lint: ✅/❌
   - Vulncheck: ✅/❌

   ### AI Review Findings

   | Category | Count | Action |
   |----------|-------|--------|
   | 🛑 Must Fix | N | Fix before shipping |
   | 🟡 Consider | N | Your call — risk is [low/medium] |
   | 🟢 Can Ship | N | Ignore — these are not real issues |

   ### Recommendation

   **[SHIP IT / FIX FIRST]**

   [Rationale: Why this code is ready, or what specifically must be fixed]

   ### The Reality Check

   This codebase has now been audited [N] times. The findings above represent
   [real risks / diminishing returns / style preferences].

   Remember: Perfect is the enemy of shipped. For a 20-year deployment system,
   stability comes from *not changing working code*, not from endless refinement.
   ```

### Decision Framework for Findings

Use this to evaluate each AI finding in final mode:

| Question | If Yes → | If No → |
|----------|----------|---------|
| Will this cause a crash in production? | 🛑 Must Fix | Continue |
| Will this leak secrets or allow unauthorised access? | 🛑 Must Fix | Continue |
| Will this cause data loss or corruption? | 🛑 Must Fix | Continue |
| Is this a real bug that will manifest in normal use? | 🛑 Must Fix | Continue |
| Does this add meaningful protection against a plausible threat? | 🟡 Consider | Continue |
| Has similar code been running fine in production systems? | 🟢 Can Ship | 🟡 Consider |
| Is this primarily a style/readability preference? | 🟢 Can Ship | — |
| Does fixing this add complexity? | 🟢 Can Ship | 🟡 Consider |
| Would a senior engineer call this "over-engineering"? | 🟢 Can Ship | — |

### Why AI Always Finds Something

Understanding this helps calibrate expectations:

1. **LLMs aren't deterministic** — different runs surface different patterns
2. **Code can always be "improved"** — more validation, more checks, more abstraction
3. **Each fix creates new surface** — changes expose new code to critique
4. **Confidence isn't calibrated** — "High" on run 4 ≠ "High" on run 1
5. **AI errs toward caution** — suggesting more checks is "safer" for the AI

**The stopping criterion isn't "zero findings" — it's "no real bugs".**

---

## Related Commands

- `/pre-commit` — Quick quality gates before committing
- `/health-check` — Development environment verification

> **Note**: Dependency auditing and principles checking are integrated into Phases 5-6 of this audit.
> Use `/code-audit` for comprehensive verification rather than separate commands.
