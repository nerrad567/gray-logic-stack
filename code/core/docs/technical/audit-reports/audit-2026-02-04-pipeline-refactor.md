# Code Audit: KNX Pipeline Robustness Refactor

**Date**: 2026-02-04  
**Scope**: KNX Pipeline Robustness Refactor (Phases A-D)  
**Commit**: a82ab57 (+ uncommitted refactor work)  
**Mode**: Standard 7-phase audit  
**Run**: #2 (overall), #1 for this commit range

---

## Summary

| Phase | Result |
|-------|--------|
| 1. Tests + Race Detection | **PASS** — all tests pass, no races, coverage 64-100% on core packages |
| 2. Lint Analysis | **PASS** — fixed UK spelling in new files; remaining warnings are pre-existing |
| 3. Vulnerability Scan | **PASS** — no known vulnerabilities |
| 4. AI Code Review | **3 HIGH, 4 MEDIUM** findings in bridge reload path |
| 5. Architecture (Hard Rules) | **PASS** — all 7 Hard Rules satisfied |
| 6. Dependency Stability | **PASS** — zero new dependencies; 2 medium-term concerns noted |
| 7. Documentation Sync | **5 STALE** docs reference old flat address format |

**Overall**: The refactor is well-engineered with zero new dependencies and full Hard Rule compliance. Three high-priority concurrency issues in the bridge reload path need attention. Five documentation files reference the superseded flat address format.

---

## Phase 1: Tests with Race Detection

```
go test -race -count=1 ./...
```

- **Result**: All tests pass (except 3 pre-existing `internal/knxd` integration tests requiring running knxd)
- **Race conditions**: None detected
- **Coverage** (key packages):
  - `internal/bridges/knx`: 64%
  - `internal/device`: 82%
  - `internal/api`: 78%
  - `internal/infrastructure`: 100%

New test files added by refactor:
- `pipeline_test.go`: 12 telegram-to-state tests, 4 command-to-telegram tests, device count, channel state keys
- `functions_test.go`: canonical registry consistency, normalisation, alias resolution, channel prefix handling

---

## Phase 2: Lint Analysis

```
golangci-lint run ./...
```

- **New file issues found and fixed**: 12 UK spelling corrections in `functions.go`, `functions_test.go`, `commissioning.go` (comments only — `recognised`, `normalise`, `unrecognised`)
- **Pre-existing warnings**: `misspell` (US/UK) in older files, some `unused` and `errcheck` in test helpers — not introduced by this refactor

---

## Phase 3: Vulnerability Scan

```
govulncheck ./...
```

- **Result**: No known vulnerabilities in any dependency
- **Scan date**: 2026-02-04

---

## Phase 4: AI Code Review

### HIGH Priority

| # | Finding | Location | Description |
|---|---------|----------|-------------|
| H1 | Data race on `infrastructureIDs` | `bridge.go:386` | `readAllDevices` goroutine writes to `infrastructureIDs` map concurrently with main goroutine reading bridge maps. No mutex protection. |
| H2 | Untracked goroutine in `readAllDevices` | `bridge.go:365` | Background goroutine launched for `readAllDevices` is not tracked by the bridge's `sync.WaitGroup`, meaning `Stop()` can return while the goroutine is still running. |
| H3 | Stale maps on `loadDevicesFromRegistry` reload | `bridge.go:295-296` | `gaToDevice` and `gaToFunction` maps are appended to but never cleared on reload. If a device is removed or a GA reassigned, stale entries persist until restart. |

### MEDIUM Priority

| # | Finding | Location | Description |
|---|---------|----------|-------------|
| M1 | `DefaultFlagsForFunction` returns shared slice | `functions.go:224-227` | Returns the internal slice reference directly. A caller mutating the returned slice would corrupt the canonical registry. Should return a copy. |
| M2 | `migrateInferDPT` matches "motion" via `Contains(fn, "on")` | `registry.go:534` | Substring `"on"` matches `"moti**on**"`, `"positi**on**"`, `"functi**on**"` — potentially assigning DPT 1.001 to non-boolean functions during migration. |
| M3 | `NormalizeFunction` is case-sensitive, `NormalizeChannelFunction` lowercases | `functions.go:137 vs 160` | Inconsistent case handling: `NormalizeFunction("Switch")` returns unknown, but `NormalizeChannelFunction("Ch_A_Switch")` would lowercase and find it. |
| M4 | `readAllDevices` shallow-copies `deviceToGAs` map values | `bridge.go:376-378` | Slice values in `deviceToGAs` are copied by reference, meaning mutations in the goroutine could affect the bridge's working copy. |

### Recommended Actions

- **H1+H2+H4**: Refactor `readAllDevices` — run synchronously during `ReloadDevices()` or add proper mutex + WaitGroup tracking
- **H3**: Clear `gaToDevice` and `gaToFunction` maps at the start of `loadDevicesFromRegistry`
- **M1**: Return `slices.Clone()` or `append([]string{}, flags...)` from `DefaultFlagsForFunction`
- **M2**: Use exact match or `HasPrefix`/`HasSuffix` instead of `Contains` for short substrings in migration
- **M3**: Decide on a consistent casing strategy (recommend: lowercase both paths)

---

## Phase 5: Architecture Review (Hard Rules)

| Hard Rule | Verdict |
|-----------|---------|
| 1. Physical controls always work | **PASS** — function registry is observation/interpretation only; unknown functions pass through gracefully |
| 2. Life safety is independent | **PASS** — `alarm` function is read/transmit only; no write flags on safety functions |
| 3. No cloud dependencies | **PASS** — zero network calls; all logic compiled in; local SQLite + MQTT |
| 4. Multi-decade deployment | **PASS** — idempotent migration, non-fatal on failure, graceful fallback chain |
| 5. Open standards (KNX) | **PASS** — all DPTs conform to KNX specification; ETS6-compatible XML export |
| 6. Customer owns system | **PASS** — human-readable JSON storage; standard ETS interop; no dealer locks |
| 7. Privacy by design | **PASS** — no PII, no telemetry, no external data transmission |

No violations or concerns. The fallback-chain design (stored DPT → canonical registry → heuristic inference) is well-suited for multi-decade resilience.

---

## Phase 6: Dependency Stability

**New dependencies added**: None (excellent discipline)

### Existing Dependency Concerns

| Dependency | Risk | Action |
|------------|------|--------|
| `influxdata/net/http (stdlib)/v2` | **MEDIUM-HIGH** — v2 client in maintenance mode; VictoriaMetrics 3 transition underway | Year 2-3 roadmap: evaluate v3 migration or alternative time-series store |
| `gorilla/websocket` | **MEDIUM** — rocky maintenance history (archived 2022, revived 2023) | Consider `coder/websocket` migration when convenient |
| `gopkg.in/yaml.v3` | **LOW** — unmaintained since April 2025 | Swap to `go.yaml.in/yaml/v3` (drop-in replacement) |

**License compliance**: All clear. No conflicts.

---

## Phase 7: Documentation Sync

| Document | Status | Issue |
|----------|--------|-------|
| `CHANGELOG.md` | **SYNCED** | v1.0.20 entry present |
| `PROJECT-STATUS.md` | **SYNCED** | Session 29 + RESUME HERE updated |
| `docs/data-model/entities.md` | **STALE** | Shows old `group_address`/`feedback_address` format |
| `docs/architecture/system-overview.md` | **SYNCED** | High-level, no format specifics |
| `docs/protocols/knx-reference.md` | **SYNCED** | Pure KNX standard reference |
| `docs/protocols/knx.md` | **SYNCED** | Already uses structured format |
| `docs/protocols/knx-howto.md` | **STALE (minor)** | Uses `direction` vs `flags` terminology |
| `code/core/AGENTS.md` | **STALE** | `device.Address` shown as string |
| `docs/interfaces/api.md` | **STALE** | API example shows old flat format |
| `docs/data-model/schemas/device.schema.json` | **STALE** | Schema uses `group_address`/`feedback_address` |
| `docs/data-model/schemas/common.schema.json` | **STALE** | `KnxAddress` type uses old format |

**Recommendation**: Update the 5 stale documents to reference the new `functions` map format. Priority order: `entities.md` and JSON schemas first (used as references), then `AGENTS.md` and `api.md`.

---

## Issue Tally

| Severity | Count | Fixed in Audit | Remaining |
|----------|-------|----------------|-----------|
| Critical | 0 | — | 0 |
| High | 3 | 0 | 3 |
| Medium | 4 | 0 | 4 |
| Low (lint) | 12 | 12 | 0 |
| Docs stale | 5 | 0 | 5 |

---

## Actionable Next Steps

1. **Fix H1+H2+H4**: Refactor `readAllDevices` concurrency in `bridge.go` (highest priority)
2. **Fix H3**: Clear maps at start of `loadDevicesFromRegistry`
3. **Fix M1**: Clone slice in `DefaultFlagsForFunction`
4. **Fix M2**: Tighten `migrateInferDPT` substring matching
5. **Fix M3**: Consistent casing in normalisation functions
6. **Update 5 stale docs** to new `functions` format
7. **Roadmap**: Track VictoriaMetrics v3 migration, gorilla/websocket replacement, yaml.v3 swap
