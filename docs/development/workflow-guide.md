---
title: Gray Logic — Developer Workflow Guide
version: 1.0.0
status: active
last_updated: 2026-02-05
audience: project owner, future contributors
---

# Gray Logic Developer Workflow Guide

This guide explains the complete quality assurance toolset for Gray Logic Core — what each tool does, why it exists, when to use it, and how they all fit together.

---

## Table of Contents

1. [The Big Picture](#the-big-picture)
2. [Quick Reference Card](#quick-reference-card)
3. [Daily Development Flow](#daily-development-flow)
4. [The Tools — Explained](#the-tools--explained)
   - [Pre-Commit Check](#pre-commit-check)
   - [Specialist Review Agents](#specialist-review-agents)
   - [Adversarial & Performance Agents](#adversarial--performance-agents)
   - [Comprehensive Audits](#comprehensive-audits)
   - [Makefile Targets](#makefile-targets)
5. [When to Use What](#when-to-use-what)
6. [Milestone Workflow (Step by Step)](#milestone-workflow-step-by-step)
7. [Multi-Agent Workflow (Claude + Codex)](#multi-agent-workflow-claude--codex)
8. [Understanding the Results](#understanding-the-results)
9. [Common Scenarios](#common-scenarios)

---

## The Big Picture

Gray Logic has a three-tier quality assurance system. Think of it as security checkpoints — the further you go, the more thorough the checks:

```
Tier 1: EVERY COMMIT            Tier 2: DURING DEVELOPMENT        Tier 3: MILESTONE GATES
─────────────────────          ──────────────────────────        ─────────────────────────
/pre-commit                     /security                         /code-audit (7+ phases)
  ├── golangci-lint              /stability                       /milestone-audit (10 stages)
  ├── go test -race              /standards
  └── principles check           /optimise
                                 /red-team
                                 /chaos
                                 /bench
                                 /review-all
```

**Why three tiers?** Because different checks catch different things at different costs:

- **Tier 1** catches obvious mistakes in seconds — run it every time
- **Tier 2** catches subtle issues in minutes — run it when working on specific areas
- **Tier 3** catches systemic issues in 15-30 minutes — run it before marking milestones complete

---

## Quick Reference Card

### Before Every Commit
```
/pre-commit
```

### Quick Targeted Reviews (pick one or more)
```
/security internal/auth/           # "Is this secure?"
/red-team internal/api/            # "How do I break this?"
/stability internal/bridges/knx/   # "What happens when things fail?"
/chaos audit                       # "Do our recovery paths actually work?"
/standards internal/bridges/knx/   # "Does this follow KNX spec?"
/optimise internal/device/         # "Is this fast enough?"
/bench run                         # "Have we gotten slower?"
/review-all internal/api/          # Run all 4 specialists at once
```

### Full Audits
```
/code-audit                        # 7-phase comprehensive audit
/milestone-audit                   # 10-stage quality gate
```

### Makefile Shortcuts
```bash
make test-race        # All tests with race detection
make test-bench       # Run benchmarks, compare with baseline
make bench-baseline   # Save current performance as baseline
make test-fuzz        # Run fuzz tests (30s each)
make test-resilience  # Run resilience tests only
make lint             # golangci-lint
make check            # fmt + vet + lint + test (everything)
```

---

## Daily Development Flow

Here's what a typical development session looks like:

### 1. Start Your Session
```bash
cd code/core
make dev-services      # Start MQTT, KNXSim, VictoriaMetrics
make dev-run           # Build + run Gray Logic Core
```

### 2. Write Code

Write your feature, fix your bug, whatever the task is. Use Claude for pair programming.

### 3. Test As You Go

While developing, run the relevant specialist if you're touching sensitive code:

| If you're changing... | Run this |
|---|---|
| Auth, JWT, tokens, passwords | `/security internal/auth/` then `/red-team internal/auth/` |
| API handlers, middleware | `/red-team internal/api/` |
| KNX bridge, telegram parsing | `/standards internal/bridges/knx/` |
| Device registry, state management | `/stability internal/device/` |
| TSDB writes, queries | `/optimise internal/infrastructure/tsdb/` |
| WebSocket, real-time features | `/stability internal/api/` |

### 4. Before Committing
```
/pre-commit
```
This runs lint, tests with race detection, and checks for principle violations (cloud dependencies, life safety control attempts). If it passes, commit.

### 5. End of Session

If you completed significant work, update the project logs:
```
/docs-sync
```

---

## The Tools — Explained

### Pre-Commit Check

**Command:** `/pre-commit`

**What it does:**
1. Kills stale processes (prevents port conflicts in tests)
2. Runs `golangci-lint` (30+ linters including security checks via gosec)
3. Runs all tests with race detection (`go test -race -count=1`)
4. Scans staged changes for cloud/external dependencies
5. Scans staged changes for life safety control attempts

**Why it exists:** This is your safety net. It catches the most common issues — test failures, lint warnings, race conditions, and principle violations — in about 30 seconds. It's intentionally fast so you'll actually use it before every commit.

**When it fails:** Fix the issue before committing. The output tells you exactly what's wrong.

---

### Specialist Review Agents

These are focused code reviewers that each look at code through a different lens. Think of them as specialists you'd hire for a consultation.

#### `/security` — The Security Auditor

**What it does:** Reviews code against a comprehensive security checklist:
- Authentication and authorisation (Argon2id params, JWT validation, RBAC)
- Input validation (SQL injection, command injection, KNX telegram bounds)
- Secrets management (no passwords in logs, no hardcoded keys)
- Protocol-specific security (KNX bus access, MQTT ACLs, DALI rate limits)
- Cryptography (TLS, crypto/rand, no custom crypto)
- Error handling (no internal details leaked to clients)

**Think of it as:** A security consultant reviewing your code defensively — "is this secure?"

**Best used:** After any changes to auth, API endpoints, or protocol bridges.

#### `/red-team` — The Adversarial Tester

**What it does:** Thinks like an attacker trying to break your system:
- Authentication exploits (brute-force, token reuse, JWT manipulation)
- Privilege escalation (user→admin, room scope escape, panel→user)
- Input abuse (oversized payloads, malformed JSON, Unicode attacks)
- Business logic flaws (double-submit, race conditions, state manipulation)
- Information disclosure (error leakage, timing attacks, enumeration)
- Building-specific attacks (KNX command flooding, MQTT topic abuse)

**Think of it as:** A penetration tester trying to break in — "how do I exploit this?"

**The difference from `/security`:** Security asks "does this meet standards?" Red team asks "can I break it anyway?" They're complementary — run both on sensitive code.

**Best used:** After auth changes, at milestone boundaries, when adding new API endpoints.

#### `/stability` — The Reliability Reviewer

**What it does:** Reviews code for fault tolerance and reliability:
- Error handling patterns (are errors properly propagated?)
- Recovery mechanisms (what happens after a failure?)
- Race conditions (concurrent access patterns)
- Resource leaks (goroutines, file handles, connections)
- Graceful degradation (does the system still work when parts fail?)

**Think of it as:** A reliability engineer asking "what happens when things go wrong?"

**Best used:** After changes to infrastructure code, bridges, or concurrent code.

#### `/chaos` — The Resilience Tester

**What it does:** Systematically verifies that recovery mechanisms actually work:
- **Audit mode** (`/chaos audit`): Reviews all 22 failure scenarios across 5 tiers, reports which have protection and which are tested
- **Test mode** (`/chaos test IF-1`): Writes Go tests for specific failure scenarios
- **Report mode** (`/chaos report`): Generates a resilience scorecard

**The 5 failure tiers:**
1. Infrastructure failures (MQTT restart, VictoriaMetrics down, disk full)
2. Protocol bridge failures (KNX crash, knxd death, malformed telegrams)
3. API/WebSocket failures (client disconnect, slow clients, request floods)
4. Shutdown/startup (SIGTERM, SIGKILL, startup with services down)
5. Data integrity (duplicate IDs, orphaned references, state explosion)

**The difference from `/stability`:** Stability *reviews* the code for resilience patterns. Chaos *tests* whether those patterns actually work when failures are injected.

**Best used:** At milestone boundaries, after infrastructure changes, to build confidence in recovery paths.

#### `/standards` — The Standards Validator

**What it does:** Checks compliance with:
- KNX Standard v3.00 (DPT encoding, addressing, APCI commands)
- Internal project principles (offline-first, multi-decade, no cloud)
- MQTT topic structure (matches `topics.go`)
- Naming conventions and terminology (matches glossary)

**Best used:** After changes to KNX bridge, MQTT integration, or adding new protocols.

#### `/optimise` — The Performance Analyst

**What it does:** Reviews code for performance and efficiency:
- Memory allocation patterns (unnecessary allocations in hot paths)
- Concurrency patterns (lock contention, channel usage)
- Algorithm efficiency (N+1 queries, linear scans that should be maps)
- Real-time latency (anything blocking the KNX telegram processing path)

**Best used:** After changes to device registry, TSDB, or WebSocket broadcast.

#### `/review-all` — The Full Suite

**What it does:** Runs all 4 specialists (`/standards`, `/security`, `/optimise`, `/stability`) in sequence on the same target, then produces a consolidated report with a single score.

**Best used:** When you want a comprehensive review of a specific package without running the full `/code-audit`.

---

### Adversarial & Performance Agents

#### `/bench` — The Benchmark Runner

**What it does:** Manages Go benchmarks for performance tracking:

| Mode | Command | Purpose |
|------|---------|---------|
| **Run** | `/bench run` | Run all benchmarks, compare with saved baseline |
| **Baseline** | `/bench baseline` | Save results as the new baseline (commit to git) |
| **Create** | `/bench create internal/device` | Write missing benchmarks for a package |
| **Profile** | `/bench profile internal/device` | Run with CPU/memory profiling |

**Why baselines matter:** On embedded ARM hardware, a change that makes telegram processing 10x slower means lights feel sluggish. By tracking baselines, you catch regressions before they reach production.

**Workflow:**
1. At each milestone completion → `/bench baseline` (save the baseline)
2. During development → `make test-bench` (compare with baseline)
3. If regression detected → `/bench profile {package}` (find the cause)

---

### Comprehensive Audits

#### `/code-audit` — The 7-Phase Audit

**What it does:** A comprehensive, tool-assisted audit:

| Phase | What It Does | Time |
|-------|-------------|------|
| 1. Tests | `go test -race -cover` — tests pass, no races, coverage targets met | 30s |
| 2. Lint | `golangci-lint` — 30+ linters including gosec security checks | 15s |
| 3. Vulnerabilities | `govulncheck` — CVE scan of all dependencies | 10s |
| 4. AI Code Review | Deep semantic analysis for bugs, logic errors, security issues | 5-10min |
| 4b. Red Team Spot Check | Quick adversarial check on security-sensitive code | 2-5min |
| 5. Architecture | Hard Rules verification (offline-first, no cloud, safety boundaries) | 2min |
| 6. Dependencies | 20-year stability assessment of go.mod | 2min |
| 7. Documentation | Docs match implementation check | 2min |

**The audit tracking system:** The audit automatically tracks how many times it's been run on the same code. After 2 clean runs, it switches to "Final Advisory" mode which gives you an honest "ship or fix" recommendation instead of finding ever-more-marginal issues.

**Why this matters:** AI reviewers will always find *something* if you keep asking. The tracking system prevents infinite audit loops — the goal is "no real bugs", not "zero findings".

#### `/milestone-audit` — The 10-Stage Quality Gate

**What it does:** Everything in `/code-audit` plus:
- Protocol compliance verification (KNX DPT encoding against spec)
- Coverage enforcement (per-package targets by tier)
- Integration test verification
- Technical debt scan (TODO/FIXME/HACK counts)
- Formal report generation with "What Was Proven" and "What Was NOT Proven" sections

**When to use:** Before marking any milestone as complete. This is the final quality gate.

---

## When to Use What

### Decision Tree

```
What did you just change?
│
├── Auth, tokens, passwords
│   └── /security → /red-team → /pre-commit
│
├── API endpoints, handlers
│   └── /red-team → /pre-commit
│
├── KNX bridge, telegram parsing
│   └── /standards → /stability → /pre-commit
│
├── Infrastructure (MQTT, TSDB, SQLite)
│   └── /stability → /chaos audit → /pre-commit
│
├── Device registry, state management
│   └── /optimise → /stability → /pre-commit
│
├── Performance-sensitive code
│   └── make test-bench → /optimise → /pre-commit
│
├── Completing a milestone
│   └── /code-audit → /milestone-audit → /bench baseline
│
└── Small fix / typo
    └── /pre-commit (just this one is fine)
```

### Effort vs Thoroughness

```
 Thoroughness
      ▲
      │
 100% │                                    ┌──────────────┐
      │                                    │/milestone-audit│
  90% │                           ┌────────┴──────────────┘
      │                           │/code-audit
  80% │               ┌───────────┘
      │               │/review-all
  60% │    ┌──────────┘
      │    │ Individual specialists
  40% │    │ (/security, /red-team, etc.)
      │ ┌──┘
  20% │ │/pre-commit
      │ │
   0% └─┴──────────────────────────────────────────► Time
      30s    2min    5min    10min   15min   30min
```

---

## Milestone Workflow (Step by Step)

Here's the complete process for completing a milestone:

### During Development
1. Write code, use specialists as needed
2. `/pre-commit` before every commit
3. Commit with conventional commit message: `feat(scope): description`

### At Milestone Completion

**Step 1: Full Audit**
```
/code-audit
```
Fix any Critical or High issues. Re-run until clean.

**Step 2: Milestone Quality Gate**
```
/milestone-audit
```
This runs the full 10-stage process and generates a formal report.

**Step 3: Performance Baseline**
```
/bench baseline
```
Saves current benchmark results so future changes can be compared.

**Step 4: Resilience Check**
```
/chaos report
```
Generates a resilience scorecard. Address any unprotected failure scenarios.

**Step 5: Documentation Sync**
```
/docs-sync
```
Updates CHANGELOG.md and PROJECT-STATUS.md.

**Step 6: Commit the Milestone**
```bash
git add .
git commit -m "feat: M2.X description of milestone"
```

---

## Multi-Agent Workflow (Claude + Codex)

Gray Logic uses two AI agents:

| Agent | Role | Strengths |
|-------|------|-----------|
| **Claude** | Orchestrator, architect, pair programmer | Interactive debugging, rapid features, code review, real-time decisions |
| **Codex** | Bounded task executor | Security audits, large refactors, comprehensive test suites, cross-cutting concerns |

### How It Works

1. Claude plans the work and identifies sub-tasks
2. If a sub-task suits Codex's strengths → Claude writes a task brief to `tasks/`
3. You run Codex on the task brief
4. Claude reviews Codex's output (build, test, lint, architectural fit)

### The No-Duplicate Rule

Before starting any implementation, Claude checks `tasks/` for existing briefs. If a brief exists for the work, it's Codex's job — Claude works on different tasks to avoid two agents writing the same code.

---

## Understanding the Results

### Severity Levels

| Level | Meaning | Action Required |
|-------|---------|-----------------|
| **Critical** | Security hole, crash, data loss | Fix immediately — blocks commit |
| **High** | Significant bug, potential exploit | Fix before milestone — blocks audit |
| **Medium** | Defensive improvement, minor risk | Should fix — document if not |
| **Low** | Style, theoretical, nice-to-have | Optional — note in report |

### When Findings Seem Endless

AI reviewers will always find something. Use this calibration:

- **Audit run 1-2:** Take findings seriously — these are real issues
- **Audit run 3+:** Switch to advisory mode — "is this worth fixing?"
- **The stopping criterion:** No Critical or High issues, not zero findings

### False Positives

Sometimes a finding isn't actually a problem. Common reasons:
- The "issue" is by design (e.g., admin-creates-admin is intentional)
- The risk is mitigated by something the reviewer didn't see
- The theoretical scenario can't happen in practice

Document the rationale and move on. The audit report has a "False Positives (Withdrawn)" section for this.

---

## Common Scenarios

### "I changed the auth system"

```
/security internal/auth/ internal/api/
/red-team internal/auth/ internal/api/
/pre-commit
```

### "I added a new API endpoint"

```
/red-team internal/api/
/pre-commit
```

### "I optimised the device registry"

```
make test-bench                    # Check for regressions
/optimise internal/device/         # Review the optimisation
/pre-commit
```

### "Things feel slow after my changes"

```
/bench profile internal/device     # Profile the suspect package
make test-bench                    # Compare with baseline
```

### "I want to check the whole codebase"

```
/code-audit                        # Full 7-phase audit
```

### "I'm done with this milestone"

```
/code-audit                        # Fix any issues
/milestone-audit                   # Formal quality gate
/bench baseline                    # Save performance baseline
/chaos report                      # Resilience scorecard
/docs-sync                         # Update project logs
git commit -m "feat: milestone description"
```

### "I want a fresh start"

```
/nuke-rebuild                      # Tears everything down, rebuilds, restarts
```

### "Is my dev environment working?"

```
/health-check                      # Checks Go, Docker, services, tools
```

---

## Glossary

| Term | Meaning |
|------|---------|
| **Baseline** | Saved benchmark results to compare future runs against |
| **Chaos testing** | Deliberately injecting failures to verify recovery works |
| **Fuzz testing** | Feeding random/malformed input to find crashes and panics |
| **Race detection** | Go's `-race` flag that finds concurrent access bugs |
| **Red teaming** | Thinking like an attacker to find exploitable vulnerabilities |
| **Resilience test** | A test that verifies recovery from a specific failure scenario |
| **Specialist agent** | A Claude command focused on one aspect of code quality |

---

## File Locations

| File | Purpose |
|------|---------|
| `.claude/commands/*.md` | All Claude command definitions |
| `.claude/audit-history.json` | Audit iteration tracking |
| `code/core/.bench-baseline.txt` | Saved benchmark baseline |
| `code/core/docs/technical/audit-reports/` | Formal audit reports |
| `code/core/Makefile` | All make targets |
| `CLAUDE.md` | Master project instructions for AI assistants |
| `PROJECT-STATUS.md` | Current progress and next steps |
