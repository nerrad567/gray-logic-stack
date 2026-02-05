---
title: Code Audit Report — M1.7 Auth Hardening + M2.1 Location Hierarchy + State Pipeline
version: 1.0.0
status: complete
audit_date: 2026-02-05
auditor: Claude Code (claude-opus-4-5-20251101)
scope: Full stack audit covering M1.7 auth, M2.1 tags/groups/zones, VictoriaMetrics TSDB, state history pipeline
previous_audit: audit-2026-02-04-pipeline-refactor.md
commit: 94e1d5b3f279e34ccb86db6086308ea0b947cce7
---

# Code Audit Report — M1.7 + M2.1 + State Pipeline

**Audit Date:** 2026-02-05
**Auditor:** Claude Code (Opus 4.5)
**Scope:** Full 7-phase audit of all changes since last audit (a82ab57..94e1d5b)
**Packages Reviewed:**
- `internal/auth/` (19 files — M1.7 Auth Hardening)
- `internal/device/` (tags, groups, resolver — M2.1)
- `internal/location/` (zones — M2.1)
- `internal/api/` (auth, users, panels, tags, groups, zones, hierarchy, middleware, websocket)
- `internal/infrastructure/tsdb/` (VictoriaMetrics client)
- `internal/device/state_history*.go` (audit trail)
- `cmd/graylogic/main.go` (wiring)

---

## Executive Summary

This is the first audit on commit `94e1d5b` — a major update that added production-grade authentication (M1.7), location hierarchy with tags/groups/zones (M2.1), VictoriaMetrics migration, and a state history audit trail. Three parallel AI review agents examined auth security, M2.1 data integrity, and the state pipeline respectively.

The codebase is fundamentally sound: all tests pass with race detection, lint is clean, SQL is parameterized throughout, and the auth system follows OWASP 2025 best practices. However, the review identified several issues that warrant attention — most notably around rate limit bypass, unauthenticated endpoints exposing sensitive data, and a TSDB line protocol injection vector.

### Audit Progression

| Run | Mode | Critical | High | Medium | Low | Notes |
|-----|------|----------|------|--------|-----|-------|
| 1 | Standard | 0 | 3 | 15 | 10 | First audit on new code — auth, M2.1, TSDB pipeline |

### Readiness Score

| Category | Score | Notes |
|----------|-------|-------|
| Tests | 9/10 | All 16 packages pass with `-race`. Coverage good (81% auth, 91% automation). API coverage (33%) could improve. |
| Security | 7/10 | Auth system well-designed, but rate limit bypassable via X-Forwarded-For; 2 endpoints unprotected; panel update not persisted |
| Reliability | 8/10 | Solid error handling. TSDB write failures isolated from device control. Some edge cases in shutdown ordering and batch buffer growth |
| Architecture | 10/10 | All 7 Hard Rules pass. Zero cloud dependencies. All deps 20-year viable. |
| Documentation | 9/10 | All packages have doc.go. Minor stale InfluxDB references in 2 architecture docs |

**Overall Readiness: 8.6/10**

---

## Phase 1: Tests — ✅ PASS

All 16 test packages pass with `-race -cover`:

| Package | Coverage | Target Met? |
|---------|----------|-------------|
| `auth` | 81.1% | ✅ (security: 80%+) |
| `automation` | 91.5% | ✅ |
| `config` | 95.9% | ✅ |
| `database` | 82.3% | ✅ |
| `logging` | 100.0% | ✅ |
| `mqtt` | 82.5% | ✅ |
| `tsdb` | 82.4% | ✅ |
| `panel` | 92.3% | ✅ |
| `device` | 78.6% | ✅ |
| `etsimport` | 74.7% | ✅ |
| `knx bridge` | 62.5% | ✅ (integration code) |
| `knxd` | 56.7% | ✅ (hardware-dependent) |
| `location` | 55.1% | ⚠️ Below 70% target |
| `process` | 61.6% | ⚠️ Below 70% target |
| `api` | 33.6% | ⚠️ Below 50% target |
| `graylogic` (main) | 3.6% | ✅ (wiring, expected low) |

**Note:** `location` and `api` coverage drops are expected — M2.1 added 22 new API endpoints and zone repositories. Test coverage for new M2.1 code exists in dedicated test files but the package-level percentage is diluted by untested older handlers.

## Phase 2: Lint — ✅ PASS

Zero warnings. Clean baseline maintained.

## Phase 3: Vulnerability Scan — ⚠️ 1 Finding

**GO-2026-4337**: Unexpected session resumption in `crypto/tls` (Go 1.25.6 stdlib).
- **Fixed in:** Go 1.25.7
- **Impact:** Affects MQTT TLS, API HTTPS, and TSDB HTTP connections
- **Action:** Update Go to 1.25.7 when available. Low urgency — requires active MITM to exploit.

## Phase 4: AI Deep Code Review

Three specialised review agents ran in parallel. Findings consolidated below by severity.

## Phase 5: Architecture — ✅ PASS

All 7 Hard Rules verified. No cloud calls, no life safety control, no proprietary protocols.

## Phase 6: Dependencies — ✅ PASS

8 direct dependencies, all mature (7+ years), all v1+ stable APIs. No new deps since last audit.

## Phase 7: Documentation — ⚠️ Minor Gaps

- 2 stale InfluxDB references in architecture decision docs (non-critical, historical context)
- `configs/config.yaml` mentions InfluxDB line protocol in TSDB comment (accurate — VM accepts it)
- All 15 packages have `doc.go` files

---

## Consolidated Findings

### High Severity

#### H1: Seed Owner Password Logged in Structured Logs

| Attribute | Value |
|-----------|-------|
| **File** | `internal/auth/seed.go:52-56` |
| **Confidence** | 95% |
| **Issue** | First-boot owner password written to slog with key `"password"`. Structured logs may be shipped to aggregation systems, persisting the plaintext credential indefinitely. |
| **Impact** | Owner account has `system:dangerous` permissions (factory reset). Log exfiltration = full compromise. |
| **Fix** | Print to stdout directly via `fmt.Println` with a visual banner — not through the structured logging pipeline. |

#### H2: Line Protocol Injection via Unescaped Newlines in TSDB Tags

| Attribute | Value |
|-----------|-------|
| **File** | `internal/infrastructure/tsdb/write.go:175-179` |
| **Confidence** | 95% |
| **Issue** | `escapeTag()` escapes spaces, commas, equals — but NOT newlines (`\n`). Newlines are the record delimiter in InfluxDB line protocol. A device ID containing `\n` can inject arbitrary data points. Reachable via MQTT state pipeline. |
| **Impact** | Attacker on MQTT bus can inject false telemetry data into VictoriaMetrics. |
| **Fix** | Add `strings.ReplaceAll(s, "\n", "")` and `strings.ReplaceAll(s, "\r", "")` to `escapeTag()`. Also escape measurement name at line 126. |

#### H3: X-Forwarded-For Rate Limit Bypass

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/middleware.go:526-533` |
| **Confidence** | 80% |
| **Issue** | `clientIP()` trusts `X-Forwarded-For` unconditionally. Rate limit key is derived from this. Attacker rotates the header to get unlimited login attempts, defeating the 5/15min brute-force protection. |
| **Impact** | Unlimited brute-force against owner/admin accounts from LAN. |
| **Fix** | Default to `r.RemoteAddr` for rate limiting. Only trust `X-Forwarded-For` when `api.trust_proxy: true` is explicitly configured. |

### Medium Severity

#### M1: Unauthenticated `/api/v1/metrics` Endpoint

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/router.go:42` |
| **Confidence** | 90% |
| **Issue** | Exposes runtime memory stats, goroutine counts, DB pool stats without auth. Useful for reconnaissance. |
| **Fix** | Move behind `authMiddleware` with `requirePermission(auth.PermSystemAdmin)`. |

#### M2: Unauthenticated `/api/v1/discovery` Endpoint

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/router.go:45` |
| **Confidence** | 90% |
| **Issue** | Exposes KNX bus topology — device addresses, group addresses, physical layout. KNX has no built-in auth; knowing topology is the primary attack barrier. |
| **Fix** | Move behind `authMiddleware` with `requirePermission(auth.PermCommissionManage)`. |

#### M3: Missing Role Validation on User PATCH

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/users.go:190-191` |
| **Confidence** | 90% |
| **Issue** | `handleUpdateUser` applies role directly without `IsValidUserRole()` check (unlike `handleCreateUser`). Admin could set role to `"panel"` or arbitrary strings. |
| **Fix** | Add `auth.IsValidUserRole(*req.Role)` validation before applying. |

#### M4: Panel Update Not Persisted (Silent Data Loss)

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/panels.go:139-172` |
| **Confidence** | 95% |
| **Issue** | `handleUpdatePanel` modifies name in memory, returns 200 OK, but never writes to database. Next GET returns old name. |
| **Fix** | Implement `PanelRepository.Update()` or return 501 until method exists. |

#### M5: Refresh Token TOCTOU Race Condition

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/auth.go:200-237` |
| **Confidence** | 80% |
| **Issue** | Read-check-write pattern without transaction. Two concurrent requests with same token can both succeed, defeating single-use rotation guarantee. |
| **Fix** | Wrap refresh flow in a DB transaction, or use atomic `UPDATE ... WHERE revoked = 0`. |

#### M6: Truncated UUID IDs (32-bit Collision Risk)

| Attribute | Value |
|-----------|-------|
| **File** | `internal/auth/user_repository.go:38`, `token_repository.go:46`, `panel_repository.go:40` |
| **Confidence** | 85% |
| **Issue** | IDs use `uuid.NewString()[:8]` (32 bits). Birthday paradox: 50% collision at ~65K entities. Refresh tokens accumulate over 20 years. |
| **Fix** | Use full UUIDs or at minimum 16 hex chars (64 bits). |

#### M7: No Maximum Password Length (Argon2id DoS)

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/auth.go:81`, `internal/api/users.go:64` |
| **Confidence** | 85% |
| **Issue** | No max password length. 1MB body + Argon2id (64 MiB memory, 3 iterations) = significant CPU/memory per request on edge hardware. |
| **Fix** | Cap at 128 characters in login, create, and change-password handlers. |

#### M8: N+1 Query in Hierarchy Endpoint

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/hierarchy.go:173` |
| **Confidence** | 95% |
| **Issue** | `buildRoomZoneMap` issues one `GetZonesForRoom` SQL query per room. 50 rooms = 50 queries per hierarchy call. |
| **Fix** | Add bulk `GetAllZoneRoomMappings()` method to `ZoneRepository`. |

#### M9: No Input Validation on Tag Values

| Attribute | Value |
|-----------|-------|
| **File** | `internal/device/tags.go:333`, `internal/api/tags.go:49` |
| **Confidence** | 90% |
| **Issue** | Tags normalised (trim+lowercase) but no max length, min length, or character set validation. Thousands-of-chars tags possible. |
| **Fix** | Add `ValidateTag()`: max 100 chars, pattern `^[a-z0-9_-]+$`. |

#### M10: No Input Validation on Group/Zone Name Length

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/groups.go:43`, `internal/api/zones.go:64` |
| **Confidence** | 85% |
| **Issue** | Group and zone names validated as non-empty but no max length check. |
| **Fix** | Reuse `device.ValidateName()` or equivalent. |

#### M11: Group Type Change Without Consistency Check

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/groups.go:131-136` |
| **Confidence** | 80% |
| **Issue** | PATCH can change group type (static→dynamic) without adjusting members/filter_rules. Orphaned data remains. |
| **Fix** | Validate consistency on type change or clear irrelevant data. |

#### M12: Silent TSDB Data Loss on Write Failure

| Attribute | Value |
|-----------|-------|
| **File** | `internal/infrastructure/tsdb/client.go:247-249` |
| **Confidence** | 88% |
| **Issue** | Failed flush discards batch data permanently. No retry, no WAL. VictoriaMetrics restarts = telemetry gaps. |
| **Fix** | Re-enqueue failed lines (bounded retry count). Long-term: WAL for failed batches. |

#### M13: Shutdown Order — MQTT Closed After TSDB Flush

| Attribute | Value |
|-----------|-------|
| **File** | `cmd/graylogic/main.go:180-185, 261-266` |
| **Confidence** | 85% |
| **Issue** | Defer order: API→TSDB→MQTT→DB. Late MQTT messages after TSDB close are silently dropped. Loses last few seconds of telemetry on every restart. |
| **Fix** | Unsubscribe from MQTT state topics before TSDB close, or swap defer order. |

#### M14: `sendResponse` Can Panic on Closed Channel

| Attribute | Value |
|-----------|-------|
| **File** | `internal/api/websocket.go:447-450` |
| **Confidence** | 82% |
| **Issue** | `sendResponse()` writes to `c.send` without `recover()` guard (unlike `trySend()`). Channel close from another goroutine = panic. |
| **Fix** | Route all sends through `trySend()` or add `defer recover()`. |

#### M15: Unbounded TSDB Query Response Read

| Attribute | Value |
|-----------|-------|
| **File** | `internal/infrastructure/tsdb/query.go:91` |
| **Confidence** | 90% |
| **Issue** | `io.ReadAll(resp.Body)` with no size limit. Wide-range PromQL query = OOM on embedded hardware. |
| **Fix** | Use `io.LimitReader(resp.Body, 10*1024*1024)`. |

### Low Severity

| # | Finding | File | Confidence |
|---|---------|------|------------|
| L1 | X-Panel-Token missing from CORS defaults | `middleware.go:94` | 85% |
| L2 | UUID v4 refresh tokens (122 bits, not 256) | `auth/claims.go:47` | 75% |
| L3 | Admin can create other admin accounts (peer escalation) | `api/users.go:78` | 80% |
| L4 | No limit on tags per device | `device/tags.go` | 80% |
| L5 | No limit on group members | `device/group_repository.go:306` | 75% |
| L6 | N+1 query fallback in group tag resolution | `device/group_resolver.go:236` | 90% |
| L7 | Hierarchy swallows room listing errors silently | `api/hierarchy.go:72` | 80% |
| L8 | State history no deduplication (high-freq sensors) | `device/state_history_sqlite.go` | 88% |
| L9 | TSDB batch buffer can spike during degraded conditions | `tsdb/client.go:209` | 83% |
| L10 | Map iteration order non-determinism in line protocol tags | `tsdb/write.go:129` | 90% |

---

## Positive Findings (Things Done Well)

The review agents identified numerous security measures correctly implemented:

1. **Argon2id parameters** — OWASP 2025 compliant (3 iterations, 64 MiB, parallelism 1). Timing-safe comparison via `subtle.ConstantTimeCompare`.
2. **JWT validation** — `WithValidMethods` prevents algorithm confusion attacks. Subject/role validated non-empty.
3. **SQL injection** — 100% parameterized queries across all packages. The `GetTagsForDevices` IN-clause correctly builds individual `?` placeholders.
4. **Secrets not serialized** — `PasswordHash`, `TokenHash` use `json:"-"` tags. Error messages are generic.
5. **Refresh token storage** — Only SHA-256 hashes stored, never raw tokens.
6. **Token theft detection** — Family-based revocation on reuse correctly implemented.
7. **Self-modification guards** — Users cannot deactivate/demote/delete themselves.
8. **Body size limits** — 1MB `MaxBytesReader` applied globally.
9. **PromQL injection prevention** — `strconv.Quote()` escaping on label values.
10. **TSDB error isolation** — Write failures never cascade to device control.
11. **Transaction safety** — `SetTags`, `SetMembers`, `SetZoneRooms`, deletes all use proper transactions.
12. **Thread safety** — Registry uses `sync.RWMutex` correctly with deep copies.
13. **One-zone-per-domain constraint** — Properly enforced with self-exclusion check.
14. **Group resolution** — No infinite loop risk. Deduplication via `mergeDevices`.
15. **Session revocation** — All tokens revoked on password change and account deletion.

---

## Recommendations

### Immediate (Before Next Feature Work)

1. **Fix H1** — Seed password out of structured logs (5 min, high impact)
2. **Fix H2** — Escape newlines in TSDB tag values (5 min, high impact)
3. **Fix H3** — Default rate limiting to `RemoteAddr` (15 min, high impact)
4. **Fix M1+M2** — Move metrics/discovery behind auth (10 min)
5. **Fix M3** — Add role validation to user PATCH (5 min)
6. **Fix M4** — Panel update persistence or 501 (15 min)

### Short-Term (Next Milestone)

7. **Fix M5** — Wrap refresh flow in DB transaction
8. **Fix M6** — Extend entity ID length to 16 hex chars
9. **Fix M7** — Add max password length (128 chars)
10. **Fix M14** — Route WebSocket sends through `trySend()`
11. **Fix M9+M10** — Add tag/name length validation
12. **Update Go** to 1.25.7 when released (GO-2026-4337)

### Long-Term (Future Milestones)

13. **Fix M8** — Bulk zone-room query for hierarchy
14. **Fix M12** — TSDB write retry or WAL
15. **Fix L8** — State history deduplication for high-frequency sensors
16. **Improve API test coverage** (33.6% → 50%+)

---

## Conclusion

The Gray Logic Core has grown significantly since the last audit — M1.7 alone added 3,009 lines of auth code, and M2.1 added 22 new API endpoints. The auth system is well-architected (Argon2id, JWT rotation, family-based theft detection, room scoping) and follows industry best practices. The M2.1 location hierarchy is clean with proper transaction safety and thread-safe caching.

The 3 High-severity findings are all straightforward fixes (log redaction, string escaping, rate limit source). None represent fundamental architectural problems. The Medium findings are a mix of input validation gaps, edge cases in shutdown/concurrency, and performance concerns — all typical for code at this stage of development.

**Verdict: ⚠️ FIX HIGH-SEVERITY ITEMS, THEN PROCEED**

The 3 High items (H1-H3) should be fixed before shipping to any deployment. They are all quick fixes (combined estimate: 25 minutes). After those, the codebase is solid for continued development.
