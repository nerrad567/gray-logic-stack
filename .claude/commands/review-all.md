---
description: Run all specialist review agents sequentially on a target
---

# Full Review Orchestrator

Run a comprehensive review using all specialist agents on the specified target.

---

## IMPORTANT: Persistent Review Tracking

**Before doing ANYTHING, Claude MUST:**

1. **Read the tracker file:** `.claude/review-tracker.json`
2. **Display the current state** — which sessions are complete, which is next
3. **If no target specified** — resume from the next pending session in the plan
4. **After completing a review** — update the tracker file with results

This ensures systematic coverage of the entire codebase across multiple sessions.

### What Claude Must Say First

```
╔══════════════════════════════════════════════════════════════════╗
║  REVIEW-ALL — Full Specialist Review                            ║
║  Date: YYYY-MM-DD                                               ║
║  Commit: [hash]                                                  ║
╚══════════════════════════════════════════════════════════════════╝

REVIEW TRACKER STATUS:
═══════════════════════════════════════════════════════════════════

Session 1: Latest Changes + Security Core     [✅ COMPLETE / ⏳ IN PROGRESS / ⬚ PENDING]
Session 2: API Layer + WebSocket              [status]
Session 3: Domain Logic + State Management    [status]
Session 4: Protocol Layer + Subprocess Mgmt   [status]
Session 5: Infrastructure + Integration       [status]

Cumulative Findings: {N} (C:{n} H:{n} M:{n} L:{n})

[Today's target / Resuming from session N]
```

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
After all specialists complete, produce the consolidated report (format below).

### Step 6: Update Tracker
Update `.claude/review-tracker.json`:
- Mark session as complete
- Add entry to `completed_reviews` array
- Update `findings_summary` counts
- Set `last_updated` date

---

## Target for Review

$ARGUMENTS

If no target is specified, resume from the next pending session in the review plan.

---

## Review Plan (5 Sessions)

The codebase has been mapped and divided into 5 review sessions, ordered by risk:

| Session | Name | Packages | LOC | Focus |
|---------|------|----------|-----|-------|
| **1** | Latest Changes + Security Core | `auth` (1,900), `api` auth handlers (~800) | ~2,700 | Auth system deep dive, latest audit fixes |
| **2** | API Layer + WebSocket | `api` remaining handlers (~2,400) | ~2,400 | All non-auth endpoints, WebSocket, routing |
| **3** | Domain Logic + State | `device` (2,800), `automation` (1,700), `location` (1,200) | ~5,700 | Registry, scenes, groups, zones, hierarchy |
| **4** | Protocol Layer | `bridges/knx` (4,500), `knxd` (1,400), `process` (850), `etsimport` (2,800) | ~9,550 | KNX, DPT, telegrams, subprocess mgmt |
| **5** | Infrastructure + Wiring | `database`, `mqtt`, `tsdb`, `config`, `logging`, `panel`, `audit`, `main.go` | ~4,000 | Cross-cutting, shutdown, integration |

**Total:** ~23,500 LOC across 155 Go files

---

## Consolidated Report Format

```markdown
# Full Code Review: {target}

**Session:** {N} of 5 — {session name}
**Date:** YYYY-MM-DD
**Commit:** {hash}
**Packages Reviewed:** {list}

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

## Cumulative Progress

| Session | Status | Findings | Score |
|---------|--------|----------|-------|
| 1. Security Core | ✅/⬚ | C:0 H:0 M:0 L:0 | X/10 |
| 2. API Layer | ✅/⬚ | C:0 H:0 M:0 L:0 | X/10 |
| 3. Domain Logic | ✅/⬚ | C:0 H:0 M:0 L:0 | X/10 |
| 4. Protocol Layer | ✅/⬚ | C:0 H:0 M:0 L:0 | X/10 |
| 5. Infrastructure | ✅/⬚ | C:0 H:0 M:0 L:0 | X/10 |
| **TOTAL** | | **C:0 H:0 M:0 L:0** | **X/10** |
```

---

## Tracker Update Format

After each session, add to `completed_reviews` in `.claude/review-tracker.json`:

```json
{
  "session": 1,
  "date": "2026-02-05",
  "commit": "abc123",
  "packages_reviewed": ["internal/auth", "internal/api (auth handlers)"],
  "specialists_run": ["standards", "security", "performance", "stability"],
  "findings": { "critical": 0, "high": 0, "medium": 2, "low": 3 },
  "score": 8.5,
  "key_findings": ["brief summary of important findings"],
  "status": "complete"
}
```

---

## Relationship to Other Commands

| Command | Use When |
|---------|----------|
| `/review-all` | Systematic codebase review with tracking across sessions |
| `/review-all {target}` | Quick focused review of specific files or packages |
| `/code-audit` | Full 7-phase audit with tooling (tests, lint, vulncheck) + AI review |
| `/milestone-audit` | End-of-milestone quality gate with coverage enforcement |

`/review-all` is a **pure code review** — it reads and analyses code against standards.
`/code-audit` additionally **runs tools** (tests, linters, vulnerability scanners) and produces formal reports.

Use `/review-all` for systematic codebase-wide reviews and targeted package reviews.
Use `/code-audit` for milestone gates and formal verification.
