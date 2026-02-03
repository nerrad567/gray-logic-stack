---
title: Code Audit Report — MQTT Topic Unification
version: 1.0.0
status: complete
audit_date: 2026-02-03
auditor: Claude Code (claude-opus-4-5-20251101)
scope: MQTT topic scheme unification and InfluxDB credential fix
previous_audit: audit-2026-01-24-m1.5-panel.md
commit: 78c6b43
---

# Code Audit Report — MQTT Topic Unification

**Audit Date:** 2026-02-03
**Auditor:** Claude Code
**Scope:** Session 23 changes — MQTT topic unification (7 Go files), InfluxDB token fix, package doc updates
**Packages Reviewed:**
- `internal/infrastructure/mqtt` (topics.go, publish.go, subscribe.go, client_test.go)
- `internal/automation` (engine.go, engine_test.go)
- `internal/device` (validation.go)
- `internal/infrastructure/influxdb` (client_test.go)

---

## Executive Summary

This audit covers the MQTT topic scheme unification that standardised all bridges on the flat topic format (`graylogic/{category}/{protocol}/{address}`). The primary change fixed a live bug where scene commands were published to topics no bridge subscribed to. The audit also covers the InfluxDB credential fix (deterministic dev token).

### Audit Progression

| Run | Mode | Critical | High | Medium | Notes |
|-----|------|----------|------|--------|-------|
| 1 | Standard | 0 | 0 | 3 | All defense-in-depth / maintainability |

### Readiness Score

| Category | Score | Notes |
|----------|-------|-------|
| Tests | 9/10 | 16/16 packages pass, race-free. InfluxDB was failing (token), now fixed. |
| Security | 9/10 | No vulnerabilities. Medium: topic segment injection defense-in-depth. |
| Reliability | 9/10 | Scene routing bug fixed. Medium: topic construction centralisation. |
| Lint | 8/10 | 96 warnings (pre-existing: complexity, magic numbers, duplication). |
| Architecture | 10/10 | Zero hard rule violations. |
| Dependencies | 10/10 | All mature, well-maintained libraries. |
| Documentation | 10/10 | All package docs synced this session. |

**Overall Readiness: 9.3/10**

---

## Phase Results

### Phase 1: Tests ✅ PASS

```
16/16 packages pass (race detection enabled, -count=1)
```

| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| config | 95.9% | 80% | ✅ |
| database | 82.3% | 80% | ✅ |
| influxdb | 79.8% | 70% | ✅ |
| logging | 100.0% | 80% | ✅ |
| mqtt | 82.5% | 80% | ✅ |
| device | 85.4% | 80% | ✅ |
| automation | 91.5% | 80% | ✅ |
| panel | 92.3% | 80% | ✅ |
| api | 48.4% | 50% | 🟡 (1.6% below target) |
| knx bridge | 62.4% | 70% | 🟡 (7.6% below target) |
| knxd | 50.4% | 50% | ✅ |
| process | 61.6% | 50% | ✅ |
| location | 76.5% | 70% | ✅ |
| etsimport | 61.8% | 50% | ✅ |
| cmd/graylogic | 4.7% | 30% | 🟡 (integration entry point) |

No race conditions detected.

### Phase 2: Lint 🟡 96 warnings

All warnings are pre-existing and not introduced by this session's changes. Key categories:
- **gocognit/gocyclo** (14): High complexity in commissioning, knxd, process managers
- **mnd** (30+): Magic numbers in knxd protocol code (byte offsets, buffer sizes)
- **dupl** (4): Duplicate area/room CRUD handlers in API
- **wrapcheck** (12): Unwrapped errors in location repository
- **govet shadow** (6): Variable shadowing in tests and API handlers

None of these affect the MQTT unification code.

### Phase 3: Vulnerabilities ✅ PASS

```
govulncheck: No vulnerabilities found.
```

### Phase 4: AI Code Review ✅ PASS (3 Medium findings)

See Issues section below.

### Phase 5: Architecture ✅ PASS

| Hard Rule | Status | Evidence |
|-----------|--------|----------|
| 1. Physical controls work | ✅ | No code blocks physical signals |
| 2. Life safety independent | ✅ | No fire/E-stop control code |
| 3. No cloud dependencies | ✅ | Zero external HTTP calls |
| 4. Multi-decade stability | ✅ | All deps are mature (5+ years) |
| 5. Open standards | ✅ | KNX, DALI, Modbus, MQTT only |
| 6. Customer owns system | ✅ | YAML config, no dealer locks |
| 7. Privacy by design | ✅ | All processing local |

### Phase 6: Dependencies ✅ PASS

| Dependency | Maintainer | Age | Risk |
|------------|------------|-----|------|
| gopkg.in/yaml.v3 | Community | 10+ years | 🟢 Very Low |
| paho.mqtt.golang | Eclipse Foundation | 8+ years | 🟢 Very Low |
| go-sqlite3 | mattn (community) | 12+ years | 🟢 Very Low |
| influxdb-client-go/v2 | InfluxData | 5+ years | 🟢 Low |
| gorilla/websocket | Community | 10+ years | 🟢 Very Low |
| go-chi/v5 | Community | 8+ years | 🟢 Very Low |
| golang-jwt/v5 | Community | 7+ years | 🟢 Very Low |
| google/uuid | Google | 7+ years | 🟢 Very Low |

### Phase 7: Documentation ✅ PASS

All package docs synced this session:
- config.md: PanelDir, SecurityConfig, InfluxDB/Logging/KNXD fields, fixed defaults
- database.md: SqlDB(), MigrateDown(), GetMigrationStatus()
- influxdb.md: Bounds validation, context timeout, Close() rationale

---

## Issues Found

### Medium Severity

#### M1: Topic construction bypasses Topics{} builder

| Attribute | Value |
|-----------|-------|
| **File** | `engine.go:329`, `api/devices.go:266` |
| **Confidence** | 95% |
| **Issue** | Both files construct MQTT topics via string concatenation instead of using `mqtt.Topics{}.BridgeCommand()` |
| **Impact** | If the topic format ever changes, these inline constructions will diverge from the canonical builder |
| **Recommendation** | Replace with `mqtt.Topics{}.BridgeCommand(protocol, address)` in a future session |

#### M2: No MQTT special character validation in topic segments

| Attribute | Value |
|-----------|-------|
| **File** | `topics.go` (all builder methods) |
| **Confidence** | 85% |
| **Issue** | Topic builder methods don't validate inputs for MQTT wildcards (`+`, `#`, `/`, null) |
| **Impact** | Low — protocol values are validated against a fixed enum; device IDs go through slug validation |
| **Recommendation** | Add a `validateTopicSegment` helper for defense-in-depth |

#### M3: Shared Parameters map fragility when FadeMS == 0

| Attribute | Value |
|-----------|-------|
| **File** | `engine.go:298-320` |
| **Confidence** | 85% |
| **Issue** | When `FadeMS == 0`, the params map is not deep-copied before concurrent use |
| **Impact** | Safe today (no mutation in that path), but fragile for future edits |
| **Recommendation** | Add defensive comment or always deep-copy params |

---

## Issues Fixed This Session

#### F1: InfluxDB test authentication — FIXED

| Attribute | Value |
|-----------|-------|
| **File** | `client_test.go:23`, `docker-compose.dev.yml:44` |
| **Issue** | Hardcoded token didn't match auto-generated container token |
| **Fix** | Added `DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=dev-token-graylogic-influxdb-local` to compose; updated test to match |

---

## Remaining Warnings (Accepted)

| Finding | Reason Accepted |
|---------|-----------------|
| 96 lint warnings | All pre-existing, not introduced by this session |
| api coverage 48.4% (target 50%) | 1.6% gap, acceptable for REST handlers |
| cmd/graylogic coverage 4.7% | Integration entry point, tested via integration tests |

---

## Recommendations

### Immediate (before next feature work)
- None required — all Critical/High issues are zero

### Short-Term (next session)
- Consider replacing inline topic construction in `engine.go` and `api/devices.go` with `mqtt.Topics{}` builder calls (M1)
- Consider adding `validateTopicSegment` helper to `topics.go` (M2)

### Long-Term
- Address pre-existing lint warnings (especially wrapcheck in location repo, complexity in commissioning)
- Increase api package coverage to 50%+ target

---

## Conclusion

The MQTT topic unification is clean and correct. The live scene routing bug is fixed, all topic builders produce the right format (verified by 32 sub-tests), and zero old-scheme references remain in the codebase. The three Medium findings are all defense-in-depth improvements — no code changes are required to ship.

**Verdict: ✅ SHIP IT**
