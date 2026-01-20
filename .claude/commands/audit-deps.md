---
description: Check Go dependencies for multi-decade viability
---

# Audit Dependencies

Reviews Go module dependencies against Gray Logic's multi-decade stability goal.

## What It Checks

1. **Abandoned packages** — No commits in 2+ years
2. **License compatibility** — GPL v3 compatible
3. **Transitive deps** — Hidden dependencies that could cause issues
4. **Security advisories** — Known vulnerabilities
5. **Alternatives** — More stable alternatives exist

## Commands

```bash
cd code/core

# List all dependencies with versions
go list -m all

# Check for known vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Show dependency graph
go mod graph | head -50

# Check outdated (but don't auto-update — manual review)
go list -u -m all
```

## Evaluation Criteria

| Criterion | Pass | Fail |
|-----------|------|------|
| Last commit | < 2 years ago | > 2 years, no activity |
| License | MIT, BSD, Apache, ISC | GPL v2 only, proprietary |
| Maintainers | Active, multiple | Single inactive maintainer |
| Alternatives | None better | stdlib or better option exists |
| Security | No CVEs | Unpatched vulnerabilities |

## Current Dependencies (code/core)

```
github.com/eclipse/paho.mqtt.golang  — MQTT client
github.com/mattn/go-sqlite3          — SQLite driver (cgo)
gopkg.in/yaml.v3                     — YAML parsing
github.com/influxdata/influxdb-client-go/v2 — InfluxDB client
```

## When to Run

- Before adding any new dependency
- Quarterly review of existing deps
- When Go releases a new major version

## Example Use

```
> /audit-deps

Checking: github.com/eclipse/paho.mqtt.golang
✓ Active (last commit: 3 months ago)
✓ License: EPL-2.0 + EDL-1.0 (compatible)
✓ Backed by Eclipse Foundation
✓ No CVEs

Checking: github.com/mattn/go-sqlite3
✓ Active (last commit: 1 month ago)
✓ License: MIT
⚠ Single maintainer (mattn) — monitor for successor
✓ No CVEs

All dependencies pass multi-decade viability check.
```

## Red Flags (Immediate Action)

- Sole maintainer abandons project
- License change to incompatible
- Unpatched CVE for > 30 days
- Breaking API changes with no migration path
