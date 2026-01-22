# Audit Reports

This directory contains all audit reports for Gray Logic — both documentation-phase (specifications) and implementation-phase (code) audits.

## Current Status

> **Latest Readiness Score:** 9.6 / 10 (Production Ready)
> **Last Audit:** 2026-01-22

## Directory Contents

| File | Type | Description |
|------|------|-------------|
| `audit-summary.md` | Index | Master summary of all audits |
| `audit-2026-01-22-m1.2-knxd.md` | Code Audit | M1.2 knxd manager audit (8 issues fixed) |
| `audit-iteration-2-log.md` | Doc Audit | Forensic deep dive (MQTT, Backups) |
| `audit-iteration-3-log.md` | Doc Audit | Edge cases (Scenes, Clocks) |
| `audit-iteration-4-log.md` | Doc Audit | Stability check (Certs, RMA) |
| `audit-iteration-5-log.md` | Doc Audit | Consistency check (Time, Archives) |
| `audit-iteration-6-log.md` | Doc Audit | Paranoid check (Load Shedding) |
| `audit-iteration-7-log.md` | Doc Audit | Fix verification |
| `audit-iteration-8-log.md` | Doc Audit | New features check |

## Audit Process

### Running an Audit

Use the `/code-audit` command, which automatically:

1. Tracks audit runs (Standard → Final Advisory progression)
2. Runs 7-phase verification (tests, lint, vulncheck, AI review, architecture, deps, docs)
3. Creates a detailed markdown report in this directory
4. Updates `audit-summary.md`

### Quick Audit

```bash
cd code/core
go test -race ./... && golangci-lint run && govulncheck ./...
```

## Report Format

All reports follow a standard structure with:
- YAML frontmatter (title, date, auditor, scope, commit)
- Executive summary with readiness scores
- Phase-by-phase results
- Issues table with file references and fixes
- False positives section
- Recommendations

See `.claude/commands/code-audit.md` for full documentation.
