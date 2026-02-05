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

Session 1: Security Core + Auth Handlers      [✅ COMPLETE / ⏳ IN PROGRESS / ⬚ PENDING]
Session 2: API Layer + WebSocket              [status]
Session 3: Domain Logic + State Management    [status]
Session 4: Protocol Layer + Subprocess Mgmt   [status]
Session 5: Infrastructure + Integration       [status]

Cumulative Findings: {N} (C:{n} H:{n} M:{n} L:{n})

[Today's target / Resuming from session N]
```

---

## Review Sequence (6 Specialists)

Execute each specialist in order, accumulating findings:

1. **Standards Validation** — KNX 3.00 compliance + internal principles + MQTT topics
2. **Security Audit** — Defensive vulnerability review and hardening
3. **Red Team** — Adversarial/offensive business logic exploit testing
4. **Optimise** — Performance, efficiency, and optimisation review
5. **Stability Review** — Reliability, error handling, and fault tolerance
6. **Chaos Audit** — Resilience pattern verification (audit mode — code review only)

### Why 6 Specialists?

| Specialist | Lens | Question It Answers |
|------------|------|---------------------|
| Standards | Compliance | "Does this follow our rules?" |
| Security | Defensive | "Is this secure against known attack classes?" |
| Red Team | Offensive | "How would I break this?" |
| Optimise | Performance | "Is this efficient on embedded ARM hardware?" |
| Stability | Reliability | "Does this handle errors and edge cases gracefully?" |
| Chaos Audit | Resilience | "Will this recover from infrastructure failures?" |

**Note:** `/bench` (benchmark runner) is intentionally excluded — it's a runtime tool that executes benchmarks, not a code reviewer. Use `/bench` at milestone boundaries via `/milestone-audit`.

---

## Process

For the target specified below, run each specialist review sequentially:

### Step 1: Standards
Run the standards review checklist from `.claude/commands/standards.md` against the target. Record all findings.

### Step 2: Security
Run the security review checklist from `.claude/commands/security.md` against the target. Record all findings.

### Step 3: Red Team
Run the adversarial review checklist from `.claude/commands/red-team.md` against the target (code review mode — identify exploitable logic, do NOT write exploit code). Focus on:
- Authentication/authorisation bypass vectors
- Business logic flaws specific to the packages under review
- Building automation attack surfaces (if protocol code)
- Information disclosure opportunities

### Step 4: Optimise
Run the performance review checklist from `.claude/commands/optimise.md` against the target. Record all findings.

### Step 5: Stability
Run the stability review checklist from `.claude/commands/stability.md` against the target. Record all findings.

### Step 6: Chaos Audit
Run `.claude/commands/chaos.md` in **audit mode** against the target. For each failure scenario relevant to the packages under review:
- Does protection exist? (code location)
- Is the protection tested?
- What's the recovery behaviour?
- Rate the gap risk

### Step 7: Consolidated Report
After all specialists complete, produce the consolidated report (format below).

### Step 8: Update Tracker
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
| **1** | Security Core + Auth Handlers | `auth` (1,900), `api` auth handlers (~800) | ~2,700 | Auth system deep dive, RBAC, JWT, token lifecycle |
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
| Red Team | {risk level} | {count} |
| Optimise | X/10 | {count} |
| Stability | X/10 | {count} |
| Chaos Audit | X/10 | {count} |

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
  "specialists_run": ["standards", "security", "red-team", "optimise", "stability", "chaos-audit"],
  "findings": { "critical": 0, "high": 0, "medium": 2, "low": 3 },
  "score": 8.5,
  "key_findings": ["brief summary of important findings"],
  "status": "complete"
}
```

---

## Relationship to Other Commands

### Tier 2 Command Roles

| Command | Role in `/review-all` | Standalone Use |
|---------|----------------------|----------------|
| `/standards` | ✅ Integrated — Step 1 | Quick standards check on a file |
| `/security` | ✅ Integrated — Step 2 | Quick defensive security review |
| `/red-team` | ✅ Integrated — Step 3 | Deep adversarial testing session |
| `/optimise` | ✅ Integrated — Step 4 | Quick performance review |
| `/stability` | ✅ Integrated — Step 5 | Quick reliability review |
| `/chaos` | ✅ Audit mode integrated — Step 6 | Full chaos testing (audit/test/report) |
| `/bench` | ❌ Not integrated | Runtime benchmarks — use at milestones |

### Relationship to Tier 3

| Command | Use When |
|---------|----------|
| `/review-all` | Systematic codebase review with tracking across sessions |
| `/review-all {target}` | Quick focused review of specific files or packages |
| `/code-audit` | Full 7-phase audit with tooling (tests, lint, vulncheck) + AI review |
| `/milestone-audit` | End-of-milestone quality gate with coverage + `/bench` |

`/review-all` is a **pure code review** — it reads and analyses code against 6 specialist dimensions.
`/code-audit` additionally **runs tools** (tests, linters, vulnerability scanners) and produces formal reports.
`/milestone-audit` is the full quality gate that includes `/bench` for performance baselines.

Use `/review-all` for systematic codebase-wide reviews and targeted package reviews.
Use `/code-audit` for milestone gates and formal verification.
