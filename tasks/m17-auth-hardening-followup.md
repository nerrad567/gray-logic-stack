# Task: M1.7 Auth Hardening — Room Scope Enforcement, Security Hardening & Integration Tests

## Context

M1.7 Auth Hardening has been implemented with a 4-tier RBAC system (panel/user/admin/owner), Argon2id password hashing, JWT access/refresh token rotation with family-based theft detection, panel device identity, per-user room scoping, and audit logging.

**Critical gap discovered post-implementation:** The `resolveRoomScopeMiddleware` correctly extracts room scope into the request context, but **no handler actually reads it**. A room-scoped `user` can currently request any device or scene in the system — the authorisation bypass is silent.

Additionally, the auth system lacks security headers (OWASP), rate limiting on login/refresh endpoints, and comprehensive integration test coverage for the new auth flows.

This is a Go codebase using Chi router, SQLite, and the standard library `net/http` pattern. UK English is used throughout (colour, not color; authorisation, not authorization).

## Objective

Three deliverables, in priority order:

### Deliverable 1: Room Scope Enforcement (CRITICAL)

Implement room scope filtering in all device and scene handlers so that:
- A `user` role only sees/operates devices in their granted rooms
- A `user` role only sees/executes scenes in their granted rooms  
- A `user` with `can_manage_scenes=true` for a room can create/edit/delete scenes in that room
- A `user` with zero room assignments sees nothing (empty lists, 403 on direct access)
- Panels are scoped to their `PanelContext.RoomIDs`
- Admin/owner bypass all room scoping (nil scope = unrestricted)

### Deliverable 2: Security Hardening

1. **Security headers middleware** — Add to the global middleware stack:
   - `X-Content-Type-Options: nosniff`
   - `X-Frame-Options: DENY`
   - `X-XSS-Protection: 0` (modern browsers, CSP is better)
   - `Content-Security-Policy: default-src 'self'` (for API responses only — NOT for panel routes)
   - `Strict-Transport-Security: max-age=31536000; includeSubDomains` (only when TLS enabled)

2. **Rate limiting on auth endpoints** — Simple in-memory rate limiter:
   - `POST /auth/login`: max 5 attempts per IP per 15-minute window
   - `POST /auth/refresh`: max 10 attempts per IP per 15-minute window
   - Return `429 Too Many Requests` with `Retry-After` header when exceeded
   - Use `sync.Map` for storage, no external dependencies
   - Periodic cleanup of expired entries (every 15 minutes)

3. **Input validation consistency** — Add `maxQueryParamLen` checks to `devices.go` query parameters (already present in `scenes.go`)

### Deliverable 3: Integration Tests

Write comprehensive auth flow integration tests covering gaps in `server_test.go`:

1. `TestRefresh_TokenRotation` — Login, refresh, verify new tokens issued, old refresh token consumed
2. `TestRefresh_TheftDetection` — Use consumed refresh token → entire family revoked → 401
3. `TestRefresh_ExpiredToken` — Attempt refresh with expired token → 401
4. `TestPermission_UserCannotConfigureDevice` — User role tries `POST /devices` → 403
5. `TestPermission_UserCannotManageUsers` — User role tries `GET /users` → 403
6. `TestRoomScope_UserSeesOnlyGrantedRooms` — User with rooms [A] lists devices → only room A devices
7. `TestRoomScope_UserCannotAccessOtherRoom` — User with rooms [A] gets device in room B → 403
8. `TestRoomScope_UserCannotOperateOtherRoom` — User with rooms [A] sends command to room B device → 403
9. `TestRoomScope_AdminBypassesRoomScope` — Admin sees all devices regardless of room assignments
10. `TestRoomScope_UserWithNoRooms` — User with zero room assignments → empty device list
11. `TestPanelAuth_RoomScoped` — Panel with room A token can access room A devices, not room B
12. `TestChangePassword_RevokesSessions` — After password change, existing refresh tokens fail
13. `TestInactiveUser_CannotLogin` — Deactivated user → 401 on login
14. `TestLogout_RevokesTokenFamily` — Logout revokes the family, refresh with same family → 401
15. `TestSceneManage_RoomScoped` — User with `can_manage_scenes` in room A can create scene there, not in room B

Additionally, use your own judgement to identify and fix any other security issues, code quality problems, or missing edge cases you discover while working through the codebase. Document anything you find and fix in a summary at the end.

## Key Files

### Must Read (understand before modifying)

| File | Purpose |
|------|---------|
| `code/core/internal/auth/types.go` | `RoomScope` struct with `CanAccessRoom()`, `CanManageScenesInRoom()` methods |
| `code/core/internal/auth/permissions.go` | `Permission` constants, `rolePermissions` map, `HasPermission()`, `IsRoomScoped()` |
| `code/core/internal/auth/room_access.go` | `RoomAccessRepository` with `ResolveRoomScope()` returning `*RoomScope` |
| `code/core/internal/api/middleware.go` | `authMiddleware`, `requirePermission()`, `resolveRoomScopeMiddleware`, context helpers (`claimsFromContext`, `panelFromContext`, `roomScopeFromContext`) |
| `code/core/internal/api/router.go` | Route groups showing which middleware is applied to which handlers |
| `code/core/internal/api/server.go` | `Server` struct, `Deps` struct showing all available dependencies |
| `code/core/internal/api/errors.go` | Error response helpers (`writeForbidden`, `writeUnauthorized`, etc.) |
| `code/core/internal/api/server_test.go` | Existing test patterns: `testServer()`, `testServerWithAuth()`, `authReq()`, `testAdminToken()` |
| `code/core/.golangci.yml` | Linter config — strict, UK English, test files exempt from some linters |

### Must Modify

| File | What to change |
|------|---------------|
| `code/core/internal/api/devices.go` | Add room scope filtering to `handleListDevices`, `handleGetDevice`, `handleGetDeviceState`, `handleSetDeviceState`, `handleGetDeviceHistory`, `handleGetDeviceMetrics`, `handleGetDeviceMetricsSummary` |
| `code/core/internal/api/scenes.go` | Add room scope filtering to `handleListScenes`, `handleGetScene`, `handleActivateScene`, `handleListSceneExecutions`. Add `can_manage_scenes` check to `handleCreateScene`, `handleUpdateScene`, `handleDeleteScene` |
| `code/core/internal/api/middleware.go` | Add `securityHeadersMiddleware()`, `rateLimitMiddleware()`. Remove `//nolint:unused` from `roomScopeFromContext` (it will now be used). Remove `//nolint:unused` from `requireUsersOnly` if you use it. |
| `code/core/internal/api/router.go` | Wire `securityHeadersMiddleware` into global middleware stack. Wire `rateLimitMiddleware` onto `/auth/login` and `/auth/refresh` routes. |
| `code/core/internal/api/server_test.go` | Add all 15+ integration tests listed above |

### Reference (read if needed, do NOT modify unless you find a bug)

| File | Purpose |
|------|---------|
| `code/core/internal/api/auth.go` | Login, refresh, logout, change-password handlers |
| `code/core/internal/api/users.go` | User CRUD handlers |
| `code/core/internal/api/panels.go` | Panel CRUD handlers |
| `code/core/internal/api/audit.go` | Async audit logging helper |
| `code/core/internal/device/registry.go` | Device registry with `ListDevices`, `GetDevice`, `GetDevicesByRoom` etc. |
| `code/core/internal/automation/types.go` | Scene struct — has `RoomID *string` field |
| `code/core/internal/automation/registry.go` | Scene registry with `ListScenes`, `ListScenesByRoom` etc. |

## Implementation Guidance

### Room Scope Pattern for Device Handlers

The middleware already resolves the scope. Handlers need to consume it. Here's the pattern:

```go
func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Get room scope (nil = unrestricted for admin/owner)
    scope := roomScopeFromContext(ctx)
    
    // For panels, build scope from PanelContext
    if pc := panelFromContext(ctx); pc != nil && scope == nil {
        scope = &auth.RoomScope{RoomIDs: pc.RoomIDs}
    }
    
    // If scope is non-nil and has zero rooms, return empty (user locked out)
    if scope != nil && len(scope.RoomIDs) == 0 {
        writeJSON(w, http.StatusOK, map[string]any{"devices": []any{}, "count": 0})
        return
    }
    
    // For room-scoped requests, filter results
    // ... existing filter logic ...
    // After getting devices, filter by scope:
    if scope != nil {
        devices = filterDevicesByRooms(devices, scope.RoomIDs)
    }
    
    writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices)})
}
```

For single-device endpoints (`handleGetDevice`, `handleSetDeviceState`, etc.):

```go
// After fetching the device:
scope := roomScopeFromContext(r.Context())
if pc := panelFromContext(r.Context()); pc != nil && scope == nil {
    scope = &auth.RoomScope{RoomIDs: pc.RoomIDs}
}
if scope != nil && !scope.CanAccessRoom(derefString(dev.RoomID)) {
    writeForbidden(w, "device not in accessible rooms")
    return
}
```

**Important:** Devices have `RoomID *string` (pointer — can be nil for unassigned devices). A nil/empty `RoomID` on a device means it's not assigned to any room. Room-scoped users should NOT see unassigned devices. Admin/owner can see all devices.

### Room Scope Pattern for Scene Handlers

Scenes also have `RoomID *string`. Same pattern as devices.

For `scene:manage` operations (create/update/delete), additionally check `can_manage_scenes`:

```go
scope := roomScopeFromContext(r.Context())
if scope != nil {
    roomID := derefString(scene.RoomID)
    if !scope.CanAccessRoom(roomID) {
        writeForbidden(w, "scene not in accessible rooms")
        return
    }
    // For create/update/delete, also check scene management permission
    if !scope.CanManageScenesInRoom(roomID) {
        writeForbidden(w, "scene management not permitted in this room")
        return
    }
}
```

### Helper Function

You'll need a `derefString` helper (or use inline):

```go
func derefString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
```

And a `filterDevicesByRooms` helper:

```go
func filterDevicesByRooms(devices []*device.Device, roomIDs []string) []*device.Device {
    roomSet := make(map[string]struct{}, len(roomIDs))
    for _, id := range roomIDs {
        roomSet[id] = struct{}{}
    }
    var filtered []*device.Device
    for _, d := range devices {
        if d.RoomID != nil {
            if _, ok := roomSet[*d.RoomID]; ok {
                filtered = append(filtered, d)
            }
        }
    }
    return filtered
}
```

Similar for scenes — `filterScenesByRooms`.

### Test Helper Pattern

Extend the existing `testServerWithAuth` to support room-scoped users:

```go
func testServerWithRoomScope(t *testing.T) (*Server, *device.Registry) {
    // Create server with auth + room access repos
    // Seed: admin user, regular user with room grants
    // Create devices in different rooms
    // Return server + registry for assertions
}
```

Generate tokens for different roles:

```go
func testUserToken(t *testing.T, role auth.Role, userID string) string {
    user := &auth.User{ID: userID, Username: userID, Role: role, IsActive: true}
    token, _ := auth.GenerateAccessToken(user, testJWTSecret, 15)
    return token
}
```

### Rate Limiter Design

Simple in-memory approach (no external deps):

```go
type rateLimiter struct {
    attempts sync.Map // key: IP string, value: *attemptRecord
}

type attemptRecord struct {
    count    int
    windowStart time.Time
    mu       sync.Mutex
}
```

Clean up expired windows periodically. Don't overthink it — this is a home automation system, not a public API with millions of users.

### Security Headers

Create as a simple middleware function. Skip CSP for `/panel/*` routes (Flutter needs inline scripts):

```go
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        // ... etc
        next.ServeHTTP(w, r)
    })
}
```

## Constraints

- Must pass `golangci-lint run` with zero warnings
- Must pass `go test -race -count=1 ./...` — all 16 packages
- Follow existing patterns in `server_test.go` (httptest.NewRequest, httptest.NewRecorder, buildRouter)
- Use UK English throughout (authorisation, colour, etc.)
- Do NOT modify files in `internal/auth/` unless you find a genuine bug
- Do NOT add external dependencies — stdlib + existing deps only
- Do NOT change the database schema or migrations
- Do NOT modify `internal/device/` or `internal/automation/` packages — only `internal/api/`
- Keep helper functions in the files where they're used (no new files except test helpers if needed)
- When adding `//nolint` directives, always include a justification comment

## Acceptance Criteria

### Deliverable 1: Room Scope Enforcement
- [ ] `handleListDevices` filters results by room scope (user + panel)
- [ ] `handleGetDevice` returns 403 for devices outside scope
- [ ] `handleGetDeviceState` returns 403 for devices outside scope
- [ ] `handleSetDeviceState` returns 403 for devices outside scope
- [ ] `handleGetDeviceHistory` returns 403 for devices outside scope
- [ ] `handleGetDeviceMetrics` returns 403 for devices outside scope
- [ ] `handleGetDeviceMetricsSummary` returns 403 for devices outside scope
- [ ] `handleListScenes` filters results by room scope
- [ ] `handleGetScene` returns 403 for scenes outside scope
- [ ] `handleActivateScene` returns 403 for scenes outside scope
- [ ] `handleCreateScene` checks `can_manage_scenes` for target room
- [ ] `handleUpdateScene` checks `can_manage_scenes` for target room
- [ ] `handleDeleteScene` checks `can_manage_scenes` for target room
- [ ] Admin/owner see all devices and scenes regardless of room assignments
- [ ] Panels are scoped to their `PanelContext.RoomIDs`
- [ ] Users with zero room assignments see empty lists
- [ ] Unassigned devices (nil RoomID) are hidden from room-scoped users
- [ ] `//nolint:unused` removed from `roomScopeFromContext`

### Deliverable 2: Security Hardening
- [ ] Security headers middleware added and wired into global middleware stack
- [ ] CSP header skipped for `/panel/*` routes
- [ ] Rate limiter on `/auth/login` (5 per 15 min per IP)
- [ ] Rate limiter on `/auth/refresh` (10 per 15 min per IP)
- [ ] Rate limiter returns 429 with `Retry-After` header
- [ ] Rate limiter has periodic cleanup of expired windows
- [ ] `devices.go` query params validated for max length

### Deliverable 3: Integration Tests
- [ ] All 15 test cases listed above are implemented and passing
- [ ] Tests follow existing patterns (httptest, buildRouter, authReq)
- [ ] Tests cover both positive and negative cases
- [ ] No test flakiness (no timing-dependent assertions)

### Quality Gates
- [ ] `cd code/core && go build ./...` succeeds
- [ ] `cd code/core && go test -race -count=1 ./...` — all 16 packages pass
- [ ] `cd code/core && golangci-lint run` — zero warnings
- [ ] No new `//nolint` directives without justification comments

## Reference Patterns

### How existing tests create authenticated requests

```go
// Admin JWT (most tests use this)
req := authReq(t, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))

// Direct token generation for specific roles
token := testAdminToken(t) // generates admin JWT
req.Header.Set("Authorization", "Bearer "+token)
```

### How existing tests set up the server

```go
srv, _ := testServer(t)           // basic server with device registry
srv := testServerWithAuth(t)      // server with user/token repos for auth tests
srv, locRepo := testServerWithLocation(t) // server with mock location repo
```

### How the device struct exposes RoomID

```go
type Device struct {
    ID     string  `json:"id"`
    RoomID *string `json:"room_id,omitempty"` // pointer — can be nil
    // ...
}
```

### How the scene struct exposes RoomID

```go
type Scene struct {
    ID     string  `json:"id"`
    RoomID *string `json:"room_id,omitempty"` // pointer — can be nil
    // ...
}
```
