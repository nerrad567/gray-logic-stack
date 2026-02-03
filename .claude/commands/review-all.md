---
description: Run all specialist review agents sequentially on a target
---

# Full Review Orchestrator

Run a comprehensive review using all specialist agents on the specified target.

---

## Review Sequence

Execute each specialist in order, accumulating findings:

1. **Standards Validation** — KNX 3.00 compliance + internal principles
2. **Security Audit** — Vulnerabilities and hardening
3. **Performance Analysis** — Efficiency and optimisation
4. **Stability Review** — Reliability and fault tolerance

## Process

For the target specified below, run each specialist review sequentially:

### Step 1: Standards
Run the standards review checklist from `.claude/commands/standards.md` against the target. Record all findings.

### Step 2: Security
Run the security review checklist from `.claude/commands/security.md` against the target. Record all findings.

### Step 3: Performance
Run the performance review checklist from `.claude/commands/optimise.md` against the target. Record all findings.

### Step 4: Stability
Run the stability review checklist from `.claude/commands/stability.md` against the target. Record all findings.

### Step 5: Consolidated Report
After all specialists complete, produce the report below.

---

## Target for Review

$ARGUMENTS

---

## Consolidated Report Format

```markdown
# Full Code Review: {target}

## Executive Summary
- Overall health: {GOOD/NEEDS ATTENTION/CRITICAL}
- Highest priority items: {top 3 issues across all reviews}

## Scores

| Aspect | Score | Priority Issues |
|--------|-------|-----------------|
| Standards | X/10 | {count} |
| Security | {risk level} | {count} |
| Performance | X/10 | {count} |
| Stability | X/10 | {count} |

## Critical Items (Must Fix)
{items from any specialist marked CRITICAL}

## High Priority Items
{consolidated HIGH/WARNING items}

## Improvement Backlog
{MODERATE/MINOR/INFO items for future consideration}

## Strengths
{positive patterns noted across reviews}

## Recommended Next Steps
1. {prioritised action}
2. {prioritised action}
3. {prioritised action}
```

---

## Relationship to Other Commands

| Command | Use When |
|---------|----------|
| `/review-all` | Quick focused review of specific files or packages |
| `/code-audit` | Full 7-phase audit with tooling (tests, lint, vulncheck) + AI review |
| `/milestone-audit` | End-of-milestone quality gate with coverage enforcement |

`/review-all` is a **pure code review** — it reads and analyses code against standards.
`/code-audit` additionally **runs tools** (tests, linters, vulnerability scanners) and produces formal reports.

Use `/review-all` for quick targeted reviews during development.
Use `/code-audit` for milestone gates and formal verification.
