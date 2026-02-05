# Task: M2.1 — Device Tags, Device Groups & Infrastructure Zones (Phases 1-3)

## Context

Gray Logic is a building automation platform. The Go core (`code/core/`) manages devices, scenes, locations (Site → Area → Room), and protocol bridges. Year 1 foundation is complete. This task implements three new data-layer features for M2.1.

**GLCore is an orchestrator** — it doesn't directly control hardware. It sends commands to protocol bridges (KNX, DALI, Modbus) via MQTT. Every controllable thing is modelled as a `Device` with a protocol address. This task adds metadata and grouping capabilities on top of the existing device and location models.

**Codex owns Phases 1-3 only** (repository/domain layer). Claude will handle Phase 4 (API handlers, routing, wiring) separately after this work and the M1.7 follow-up task both complete.

This is a Go codebase using SQLite, the standard library `database/sql`, and `github.com/mattn/go-sqlite3`. UK English throughout (colour, not color; authorisation, not authorization).

## Objective

Three deliverables, in dependency order:

### Deliverable 1: Device Tags (Phase 1)

Simple string labels on devices enabling filtering and exception-based operations. Example: tag emergency fittings as `escape_lighting`, then exclude them from a "power down the wing" group.

### Deliverable 2: Device Groups (Phase 2)

Named, resolvable collections of devices. Three types:
- `static` — Explicit device list ("Cinema System" = [projector, avr, screen-motor])
- `dynamic` — Location scope + domain + capability + tag filters ("West Wing Lighting" = area:west-wing + domain:lighting)
- `hybrid` — Dynamic base + explicit includes/excludes

Groups are **resolved at runtime** — membership changes as devices are added/removed without re-saving anything.

### Deliverable 3: Unified Infrastructure Zones (Phase 3)

A single generic `infrastructure_zones` table covering ALL domains (climate, audio, lighting, power, security, video). Domain-specific config lives in a JSON `settings` column. Zones group **rooms** (not devices) that share a physical resource — e.g., rooms sharing a thermostat (climate), rooms sharing an amplifier channel (audio).

Replaces the original plan of separate `climate_zones` / `audio_zones` tables. The existing `rooms.climate_zone_id` and `rooms.audio_zone_id` columns become deprecated (stop reading/writing them — zone membership moves to the new `infrastructure_zone_rooms` junction table).

## Key Files

### Must Read (understand patterns before writing code)

| File | Purpose |
|------|---------|
| `code/core/internal/device/types.go` | Device struct — add `Tags []string` field here |
| `code/core/internal/device/registry.go` | Device cache with `RefreshCache`, deep-copy pattern, `RLock`/`Lock` usage |
| `code/core/internal/device/repository.go` | `Repository` interface + `SQLiteRepository` — follow this pattern exactly |
| `code/core/internal/device/repository_test.go` | Test patterns: `setupTestDB`, in-memory SQLite, `t.Helper()`, `t.Cleanup` |
| `code/core/internal/device/registry_test.go` | Registry test patterns with mock repository |
| `code/core/internal/location/types.go` | Site, Area, Room structs |
| `code/core/internal/location/repository.go` | Location `Repository` interface + `SQLiteRepository` — follow this pattern for zones |
| `code/core/internal/location/repository_test.go` | Location test patterns: `setupTestDB` with inline schema + seed data |
| `code/core/internal/location/errors.go` | Error sentinel pattern — add zone errors here |
| `code/core/.golangci.yml` | Linter config — strict, UK English (misspell locale: UK), test files exempt from some linters |
| `code/core/internal/device/validation.go` | `GenerateID()`, `GenerateSlug()` helpers — reuse these |

### Must Create

| File | Deliverable | Purpose |
|------|-------------|---------|
| `migrations/20260206_100000_tags_groups_zones.up.sql` | All | Migration creating all new tables |
| `migrations/20260206_100000_tags_groups_zones.down.sql` | All | Rollback migration |
| `internal/device/tags.go` | 1 | TagRepository interface + SQLite implementation |
| `internal/device/tags_test.go` | 1 | Tag CRUD tests |
| `internal/device/group_types.go` | 2 | DeviceGroup, GroupType, FilterRules structs |
| `internal/device/group_repository.go` | 2 | GroupRepository interface + SQLite implementation |
| `internal/device/group_repository_test.go` | 2 | Group CRUD + member management tests |
| `internal/device/group_resolver.go` | 2 | ResolveGroup function — expands groups to device lists |
| `internal/device/group_resolver_test.go` | 2 | Resolver tests with scope, domain, tag, exclusion filters |
| `internal/location/zone_types.go` | 3 | InfrastructureZone struct, ZoneDomain constants |
| `internal/location/zone_repository.go` | 3 | ZoneRepository interface + SQLite implementation |
| `internal/location/zone_repository_test.go` | 3 | Zone CRUD + room assignment tests |

### Must Modify

| File | Deliverable | Changes |
|------|-------------|---------|
| `internal/device/types.go` | 1 | Add `Tags []string` field to Device struct, update DeepCopy |
| `internal/device/registry.go` | 1 | Load tags in RefreshCache, include Tags in cache operations |
| `internal/location/errors.go` | 3 | Add zone error sentinels |

### Do NOT Modify

These files are being modified by another Codex task (M1.7 follow-up) or are Claude's responsibility:

- `internal/api/*.go` — ALL API handler files
- `cmd/graylogic/main.go` — Wiring
- `internal/auth/*.go` — Auth package
- `internal/automation/*.go` — Scene engine

---

## Migration SQL

### `migrations/20260206_100000_tags_groups_zones.up.sql`

```sql
-- ============================================================
-- Device tags: simple labels for filtering and exceptions
-- ============================================================
CREATE TABLE device_tags (
    device_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (device_id, tag),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_device_tags_tag ON device_tags(tag);

-- ============================================================
-- Device groups: named, resolvable collections of devices
-- ============================================================
CREATE TABLE device_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,

    -- Group type: static | dynamic | hybrid
    type TEXT NOT NULL DEFAULT 'static',

    -- Dynamic filter rules (JSON, null for static groups)
    -- {
    --   "scope_type": "site"|"area"|"room",
    --   "scope_id": "area-id" (null for site),
    --   "domains": ["lighting", "audio"],
    --   "capabilities": ["dim", "on_off"],
    --   "tags": ["accent"],
    --   "exclude_tags": ["escape_lighting"],
    --   "device_types": ["light_dimmer", "light_rgb"]
    -- }
    filter_rules TEXT,

    -- UI metadata
    icon TEXT,
    colour TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
) STRICT;

-- Explicit group members (static + hybrid groups)
CREATE TABLE device_group_members (
    group_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (group_id, device_id),
    FOREIGN KEY (group_id) REFERENCES device_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_device_group_members_device ON device_group_members(device_id);

-- ============================================================
-- Infrastructure zones: unified physical resource groupings
-- One table for all domains (climate, audio, lighting, power, security, video).
-- Domain-specific config lives in the JSON settings column.
-- ============================================================
CREATE TABLE infrastructure_zones (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    domain TEXT NOT NULL,

    -- Domain-specific configuration (JSON)
    -- Climate:  {"thermostat_id": "dev-123", "mode": "heating"}
    -- Audio:    {"matrix_zone": 4, "max_volume": 80}
    -- Lighting: {"gateway_id": "dev-456", "dali_line": 1}
    -- Security: {"panel_id": "dev-789", "partition": 1}
    -- Power:    {"priority": "flexible"}
    settings TEXT NOT NULL DEFAULT '{}',

    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE (site_id, slug)
) STRICT;

CREATE INDEX idx_infra_zones_site ON infrastructure_zones(site_id);
CREATE INDEX idx_infra_zones_domain ON infrastructure_zones(domain);

-- Room membership in infrastructure zones.
-- A room can belong to one zone per domain (e.g., one climate zone + one audio zone).
CREATE TABLE infrastructure_zone_rooms (
    zone_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (zone_id, room_id),
    FOREIGN KEY (zone_id) REFERENCES infrastructure_zones(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_infra_zone_rooms_room ON infrastructure_zone_rooms(room_id);
```

### `migrations/20260206_100000_tags_groups_zones.down.sql`

```sql
DROP TABLE IF EXISTS infrastructure_zone_rooms;
DROP TABLE IF EXISTS infrastructure_zones;
DROP TABLE IF EXISTS device_group_members;
DROP TABLE IF EXISTS device_groups;
DROP TABLE IF EXISTS device_tags;
```

---

## Deliverable 1: Device Tags

### Types — Modify `internal/device/types.go`

Add to the `Device` struct:

```go
// Tags are free-form string labels for filtering and exception-based operations.
// Example: ["escape_lighting", "accent", "entertainment"]
Tags []string `json:"tags,omitempty"`
```

Update `DeepCopy()` to clone the Tags slice:

```go
if d.Tags != nil {
    cpy.Tags = make([]string, len(d.Tags))
    copy(cpy.Tags, d.Tags)
}
```

### TagRepository — New file `internal/device/tags.go`

```go
// TagRepository manages device tag associations in SQLite.
type TagRepository interface {
    // SetTags replaces all tags for a device (delete + insert in transaction).
    SetTags(ctx context.Context, deviceID string, tags []string) error

    // GetTags returns all tags for a device.
    GetTags(ctx context.Context, deviceID string) ([]string, error)

    // AddTag adds a single tag to a device (idempotent).
    AddTag(ctx context.Context, deviceID, tag string) error

    // RemoveTag removes a single tag from a device.
    RemoveTag(ctx context.Context, deviceID, tag string) error

    // ListDevicesByTag returns all device IDs that have the given tag.
    ListDevicesByTag(ctx context.Context, tag string) ([]string, error)

    // ListAllTags returns all unique tags across all devices, sorted alphabetically.
    ListAllTags(ctx context.Context) ([]string, error)

    // GetTagsForDevices returns a map of deviceID → []tags for bulk loading.
    // Used by Registry.RefreshCache to populate device tags efficiently.
    GetTagsForDevices(ctx context.Context, deviceIDs []string) (map[string][]string, error)
}
```

Implement `SQLiteTagRepository` with `*sql.DB` field. Follow the same constructor pattern as `NewSQLiteRepository`.

**Key implementation details:**
- `SetTags` uses a transaction: `DELETE FROM device_tags WHERE device_id = ?` then batch `INSERT`
- `AddTag` uses `INSERT OR IGNORE` for idempotency
- `GetTagsForDevices` uses `WHERE device_id IN (?)` with placeholder expansion for bulk loading. If the deviceIDs list is empty, return an empty map (don't query).
- Tags are always stored lowercase and trimmed. Normalise in the repository before insert.

### Registry Integration — Modify `internal/device/registry.go`

Add a `tagRepo TagRepository` field to the `Registry` struct. Add a `SetTagRepository(repo TagRepository)` method (same pattern as `SetLogger`).

In `RefreshCache`:
1. After loading devices from repo, collect all device IDs
2. Call `tagRepo.GetTagsForDevices(ctx, deviceIDs)` to bulk-load tags
3. Assign tags to each cached device before storing

In `CreateDevice`, `UpdateDevice`, `DeleteDevice`: tags are managed separately via TagRepository, not through the device CRUD. The cache just needs refreshing after tag changes.

Add a convenience method:

```go
// GetDevicesByTag returns all cached devices that have the given tag.
func (r *Registry) GetDevicesByTag(ctx context.Context, tag string) ([]Device, error) {
    r.cacheMu.RLock()
    defer r.cacheMu.RUnlock()

    var devices []Device
    for _, d := range r.cache {
        for _, t := range d.Tags {
            if t == tag {
                devices = append(devices, *d.DeepCopy())
                break
            }
        }
    }
    sortDevicesByName(devices)
    return devices, nil
}
```

### Tests — New file `internal/device/tags_test.go`

Follow the existing `repository_test.go` pattern: `setupTestDB` with in-memory SQLite, inline schema creation. The test DB needs the `devices` table AND the `device_tags` table. Seed a few test devices.

**Required test cases:**

1. `TestTagRepository_SetTags` — Set tags, verify retrieval, replace with new set
2. `TestTagRepository_SetTags_Empty` — Set empty slice clears all tags
3. `TestTagRepository_AddTag` — Add single tag, verify idempotency (add same tag twice)
4. `TestTagRepository_RemoveTag` — Remove tag, verify it's gone
5. `TestTagRepository_ListDevicesByTag` — Multiple devices with same tag, verify all returned
6. `TestTagRepository_ListAllTags` — Multiple devices, multiple tags, verify unique sorted list
7. `TestTagRepository_GetTagsForDevices` — Bulk load for multiple devices, verify correct mapping
8. `TestTagRepository_GetTagsForDevices_Empty` — Empty device list returns empty map
9. `TestTagRepository_Normalisation` — Tags with mixed case and whitespace are normalised to lowercase/trimmed
10. `TestRegistry_GetDevicesByTag` — Registry cache returns correct devices filtered by tag

---

## Deliverable 2: Device Groups

### Types — New file `internal/device/group_types.go`

```go
package device

import "time"

// GroupType defines how a device group resolves its members.
type GroupType string

const (
    // GroupTypeStatic resolves from explicit member list only.
    GroupTypeStatic GroupType = "static"
    // GroupTypeDynamic resolves from filter rules only (location + domain + tag filters).
    GroupTypeDynamic GroupType = "dynamic"
    // GroupTypeHybrid resolves from filter rules unioned with explicit members.
    GroupTypeHybrid GroupType = "hybrid"
)

// FilterRules defines dynamic resolution criteria for a device group.
// All non-empty fields are ANDed together. Empty fields are ignored.
type FilterRules struct {
    // Location scope
    ScopeType string `json:"scope_type,omitempty"` // "site", "area", or "room"
    ScopeID   string `json:"scope_id,omitempty"`   // area or room ID (empty for site scope)

    // Device filters (all ANDed)
    Domains      []string `json:"domains,omitempty"`      // e.g., ["lighting", "audio"]
    Capabilities []string `json:"capabilities,omitempty"` // e.g., ["dim", "on_off"]
    Tags         []string `json:"tags,omitempty"`         // include devices with ANY of these tags
    ExcludeTags  []string `json:"exclude_tags,omitempty"` // exclude devices with ANY of these tags
    DeviceTypes  []string `json:"device_types,omitempty"` // e.g., ["light_dimmer", "light_rgb"]
}

// DeviceGroup is a named, resolvable collection of devices.
type DeviceGroup struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Slug        string     `json:"slug"`
    Description *string    `json:"description,omitempty"`
    Type        GroupType  `json:"type"`
    FilterRules *FilterRules `json:"filter_rules,omitempty"` // nil for static groups
    Icon        *string    `json:"icon,omitempty"`
    Colour      *string    `json:"colour,omitempty"`
    SortOrder   int        `json:"sort_order"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// GroupMember represents an explicit device in a static or hybrid group.
type GroupMember struct {
    GroupID   string    `json:"group_id"`
    DeviceID  string    `json:"device_id"`
    SortOrder int       `json:"sort_order"`
    CreatedAt time.Time `json:"created_at"`
}
```

### GroupRepository — New file `internal/device/group_repository.go`

```go
type GroupRepository interface {
    Create(ctx context.Context, group *DeviceGroup) error
    GetByID(ctx context.Context, id string) (*DeviceGroup, error)
    List(ctx context.Context) ([]DeviceGroup, error)
    Update(ctx context.Context, group *DeviceGroup) error
    Delete(ctx context.Context, id string) error

    // Member management (for static and hybrid groups)
    SetMembers(ctx context.Context, groupID string, deviceIDs []string) error
    GetMembers(ctx context.Context, groupID string) ([]GroupMember, error)
    GetMemberDeviceIDs(ctx context.Context, groupID string) ([]string, error)
}
```

Implement `SQLiteGroupRepository`. Key details:
- `FilterRules` is serialised/deserialised as JSON TEXT in the `filter_rules` column
- `SetMembers` uses a transaction: delete existing + bulk insert with sort_order = insertion index
- `List` returns groups ordered by `sort_order, name`
- Use `GenerateID()` and `GenerateSlug()` from `validation.go` if ID/slug not provided
- Error sentinel: add `ErrGroupNotFound` to a new error var block at the top of the file (or in a separate `group_errors.go` if you prefer — keep it in the `device` package either way)

### GroupResolver — New file `internal/device/group_resolver.go`

```go
// ResolveGroup expands a DeviceGroup into a concrete list of devices.
// It combines explicit members (static/hybrid) with dynamic filter results,
// then applies exclude_tags to remove exceptions.
//
// The registry parameter provides cached device access.
// The tagRepo parameter provides tag lookups for filtering.
func ResolveGroup(
    ctx context.Context,
    group *DeviceGroup,
    registry *Registry,
    tagRepo TagRepository,
) ([]Device, error)
```

**Resolution algorithm:**

```
1. Collect explicit members (if static or hybrid):
   - Get member device IDs from GroupRepository
   - Look up each device from Registry

2. Collect dynamic results (if dynamic or hybrid):
   a. Get base device list based on scope:
      - scope_type "site" → registry.ListDevices()
      - scope_type "area" → registry.GetDevicesByArea(scope_id)
      - scope_type "room" → registry.GetDevicesByRoom(scope_id)
   b. Filter by domains (if non-empty): keep only devices where device.Domain is in the list
   c. Filter by device_types (if non-empty): keep only devices where device.Type is in the list
   d. Filter by capabilities (if non-empty): keep only devices that have ALL specified capabilities
   e. Filter by tags (if non-empty): keep only devices that have ANY of the specified tags

3. Union: merge explicit + dynamic (deduplicate by device ID)

4. Apply exclude_tags (if non-empty): remove any device that has ANY of the exclude tags

5. Sort by name for deterministic output

6. Return final device list
```

**Important:** The resolver takes a `*Registry` not a `Repository` — it works from the cache for performance. Tags are read from the device's `Tags` field (populated by cache refresh), not by querying the tag repository for each device. The `tagRepo` parameter is available if needed for edge cases but the primary path uses cached tags.

### Tests — `internal/device/group_repository_test.go`

1. `TestGroupRepository_Create` — Create group, verify fields persisted
2. `TestGroupRepository_Create_WithFilterRules` — Dynamic group with JSON filter rules
3. `TestGroupRepository_GetByID` — Retrieve group, verify all fields including deserialised FilterRules
4. `TestGroupRepository_GetByID_NotFound` — Returns ErrGroupNotFound
5. `TestGroupRepository_List` — Multiple groups, verify sort order
6. `TestGroupRepository_Update` — Modify name, type, filter_rules
7. `TestGroupRepository_Delete` — Delete group, verify cascade deletes members
8. `TestGroupRepository_SetMembers` — Set members, verify retrieval
9. `TestGroupRepository_SetMembers_Replace` — Set members replaces previous set
10. `TestGroupRepository_GetMemberDeviceIDs` — Verify correct device IDs returned

### Tests — `internal/device/group_resolver_test.go`

These tests need a populated Registry (use the mock from `registry_test.go` pattern — create a `mockRepository` implementing the `Repository` interface, populate it, call `RefreshCache`). Also need a `SQLiteTagRepository` with tags set on test devices.

1. `TestResolveGroup_Static` — Static group with 3 explicit members → returns those 3 devices
2. `TestResolveGroup_Dynamic_ByArea` — Dynamic group scoped to area → returns only devices in that area
3. `TestResolveGroup_Dynamic_ByDomain` — Dynamic group filtered to lighting domain → returns only lighting devices
4. `TestResolveGroup_Dynamic_ByTag` — Dynamic group filtered by tag "accent" → returns only tagged devices
5. `TestResolveGroup_Dynamic_ExcludeTags` — Dynamic group with exclude_tags ["escape_lighting"] → removes tagged devices
6. `TestResolveGroup_Dynamic_MultipleFilters` — Area + domain + capability filters ANDed together
7. `TestResolveGroup_Hybrid` — Dynamic results + explicit members merged, no duplicates
8. `TestResolveGroup_Dynamic_ByDeviceType` — Filter by device_types
9. `TestResolveGroup_Dynamic_ByCapability` — Filter by capabilities (device must have ALL)
10. `TestResolveGroup_Empty` — Group resolves to zero devices → returns empty slice (not nil)
11. `TestResolveGroup_ExcludeRemovesExplicitMembers` — Exclude tags apply to explicit members too

---

## Deliverable 3: Unified Infrastructure Zones

### Types — New file `internal/location/zone_types.go`

```go
package location

import "time"

// ZoneDomain represents the infrastructure domain a zone belongs to.
type ZoneDomain string

const (
    ZoneDomainClimate  ZoneDomain = "climate"
    ZoneDomainAudio    ZoneDomain = "audio"
    ZoneDomainLighting ZoneDomain = "lighting"
    ZoneDomainPower    ZoneDomain = "power"
    ZoneDomainSecurity ZoneDomain = "security"
    ZoneDomainVideo    ZoneDomain = "video"
)

// AllZoneDomains returns all valid zone domain values.
func AllZoneDomains() []ZoneDomain {
    return []ZoneDomain{
        ZoneDomainClimate, ZoneDomainAudio, ZoneDomainLighting,
        ZoneDomainPower, ZoneDomainSecurity, ZoneDomainVideo,
    }
}

// ValidZoneDomain checks whether the given string is a valid zone domain.
func ValidZoneDomain(s string) bool {
    for _, d := range AllZoneDomains() {
        if ZoneDomain(s) == d {
            return true
        }
    }
    return false
}

// InfrastructureZone represents a physical resource grouping that spans one or more rooms.
// All domains use this same table — domain-specific config lives in Settings.
//
// Examples:
//   - Climate zone: rooms sharing a thermostat. Settings: {"thermostat_id": "dev-123"}
//   - Audio zone: rooms sharing an amplifier channel. Settings: {"matrix_zone": 4}
//   - Lighting zone: rooms on a DALI bus. Settings: {"gateway_id": "dev-456", "dali_line": 1}
//   - Security zone: rooms in an alarm partition. Settings: {"panel_id": "dev-789", "partition": 1}
type InfrastructureZone struct {
    ID        string     `json:"id"`
    SiteID    string     `json:"site_id"`
    Name      string     `json:"name"`
    Slug      string     `json:"slug"`
    Domain    ZoneDomain `json:"domain"`
    Settings  Settings   `json:"settings"` // Reuse location.Settings (map[string]any)
    SortOrder int        `json:"sort_order"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

### Error Sentinels — Modify `internal/location/errors.go`

Add:

```go
// ErrZoneNotFound is returned when an infrastructure zone ID does not exist.
ErrZoneNotFound = errors.New("infrastructure zone not found")

// ErrRoomAlreadyInZoneDomain is returned when assigning a room to a zone
// but the room is already in another zone of the same domain.
ErrRoomAlreadyInZoneDomain = errors.New("room already assigned to a zone in this domain")
```

### ZoneRepository — New file `internal/location/zone_repository.go`

```go
type ZoneRepository interface {
    // Zone CRUD
    CreateZone(ctx context.Context, zone *InfrastructureZone) error
    GetZone(ctx context.Context, id string) (*InfrastructureZone, error)
    ListZones(ctx context.Context, siteID string) ([]InfrastructureZone, error)
    ListZonesByDomain(ctx context.Context, siteID string, domain ZoneDomain) ([]InfrastructureZone, error)
    UpdateZone(ctx context.Context, zone *InfrastructureZone) error
    DeleteZone(ctx context.Context, id string) error

    // Room membership
    SetZoneRooms(ctx context.Context, zoneID string, roomIDs []string) error
    GetZoneRooms(ctx context.Context, zoneID string) ([]Room, error)
    GetZoneRoomIDs(ctx context.Context, zoneID string) ([]string, error)

    // Reverse lookup: which zone is this room in for a given domain?
    GetZoneForRoom(ctx context.Context, roomID string, domain ZoneDomain) (*InfrastructureZone, error)

    // List all zones a room belongs to (across all domains)
    GetZonesForRoom(ctx context.Context, roomID string) ([]InfrastructureZone, error)
}
```

Implement `SQLiteZoneRepository` with `*sql.DB` field.

**Key implementation details:**

- `Settings` is serialised/deserialised as JSON TEXT (same pattern as Room.Settings in `repository.go`)
- `SetZoneRooms` must enforce the one-zone-per-domain constraint:
  1. Get the zone to find its domain
  2. For each room being assigned, check if it's already in another zone of the same domain
  3. If so, return `ErrRoomAlreadyInZoneDomain`
  4. Transaction: delete existing room assignments for this zone, then insert new ones
- `GetZoneForRoom` queries: `SELECT z.* FROM infrastructure_zones z JOIN infrastructure_zone_rooms r ON z.id = r.zone_id WHERE r.room_id = ? AND z.domain = ?`
- `ListZones` returns zones ordered by `domain, sort_order, name`
- `DeleteZone` cascades to `infrastructure_zone_rooms` (handled by FK ON DELETE CASCADE)
- Slugs auto-generated from name if not provided (use a local slugify — same pattern as `repository.go` in the location package, or import from device package if accessible)

### Tests — New file `internal/location/zone_repository_test.go`

Extend the existing `setupTestDB` pattern. The test DB schema needs `sites`, `areas`, `rooms` (already in the existing test setup) PLUS the new `infrastructure_zones` and `infrastructure_zone_rooms` tables.

**Important:** Either extend the existing `setupTestDB` or create a `setupZoneTestDB` that includes all tables. Do NOT modify the existing `setupTestDB` function — create a new one so existing tests aren't affected.

**Required test cases:**

1. `TestZoneRepository_CreateZone` — Create climate zone, verify all fields
2. `TestZoneRepository_CreateZone_AudioDomain` — Create audio zone with domain-specific settings
3. `TestZoneRepository_GetZone` — Retrieve zone, verify Settings deserialised correctly
4. `TestZoneRepository_GetZone_NotFound` — Returns ErrZoneNotFound
5. `TestZoneRepository_ListZones` — Multiple zones, verify ordering by domain then sort_order
6. `TestZoneRepository_ListZonesByDomain` — Filter by domain, verify only matching zones returned
7. `TestZoneRepository_UpdateZone` — Update name, settings
8. `TestZoneRepository_DeleteZone` — Delete zone, verify cascade removes room assignments
9. `TestZoneRepository_SetZoneRooms` — Assign 3 rooms to a zone, verify retrieval
10. `TestZoneRepository_SetZoneRooms_Replace` — Replace room assignments
11. `TestZoneRepository_GetZoneRooms` — Verify Room structs returned (not just IDs)
12. `TestZoneRepository_GetZoneForRoom` — Room in climate zone → returns that zone
13. `TestZoneRepository_GetZoneForRoom_NoDomain` — Room not in any climate zone → returns ErrZoneNotFound
14. `TestZoneRepository_GetZonesForRoom` — Room in climate + audio zones → returns both
15. `TestZoneRepository_OneZonePerDomain` — Assign room to climate zone A, then try assigning to climate zone B → returns ErrRoomAlreadyInZoneDomain
16. `TestZoneRepository_DifferentDomainsAllowed` — Room in climate zone A AND audio zone B → succeeds (different domains)

---

## Constraints

- Must pass `golangci-lint run` with zero warnings
- Must pass `go test -race -count=1 ./...` — all packages pass
- Must pass `go build ./...`
- Use UK English throughout (colour, authorisation, etc.)
- Follow existing patterns: `*sql.DB` field, `context.Context` first param, `error` last return, `t.Helper()` in test helpers
- Do NOT create any files in `internal/api/` — that's Claude's responsibility
- Do NOT modify `cmd/graylogic/main.go` — that's Claude's responsibility
- Do NOT modify existing test helper functions (`setupTestDB`) — create new ones
- Keep all code in existing packages (`device`, `location`) — no new packages
- When adding `//nolint` directives, always include a justification comment
- JSON serialisation/deserialisation: use `encoding/json` (stdlib), not third-party

## Acceptance Criteria

### Deliverable 1: Device Tags
- [ ] Migration creates `device_tags` table
- [ ] `SQLiteTagRepository` implements all `TagRepository` interface methods
- [ ] Tags are normalised to lowercase + trimmed before storage
- [ ] `SetTags` is transactional (delete + insert)
- [ ] `AddTag` is idempotent (INSERT OR IGNORE)
- [ ] `GetTagsForDevices` handles empty input gracefully
- [ ] Device struct has `Tags []string` field with DeepCopy support
- [ ] Registry loads tags on `RefreshCache` via bulk `GetTagsForDevices`
- [ ] Registry has `GetDevicesByTag` method
- [ ] All 10 tag tests pass

### Deliverable 2: Device Groups
- [ ] Migration creates `device_groups` and `device_group_members` tables
- [ ] `SQLiteGroupRepository` implements all `GroupRepository` interface methods
- [ ] `FilterRules` serialises/deserialises to/from JSON
- [ ] `SetMembers` is transactional (delete + insert)
- [ ] `ResolveGroup` correctly handles static, dynamic, and hybrid groups
- [ ] Dynamic resolution: scope filtering works (site/area/room)
- [ ] Dynamic resolution: domain, capability, tag, device_type filters AND correctly
- [ ] Dynamic resolution: exclude_tags removes matching devices (including explicit members)
- [ ] Dynamic resolution: results deduplicated and sorted by name
- [ ] All 10 repository tests + 11 resolver tests pass

### Deliverable 3: Unified Infrastructure Zones
- [ ] Migration creates `infrastructure_zones` and `infrastructure_zone_rooms` tables
- [ ] `SQLiteZoneRepository` implements all `ZoneRepository` interface methods
- [ ] Settings serialise/deserialise as JSON
- [ ] `SetZoneRooms` enforces one-zone-per-domain constraint
- [ ] `GetZoneForRoom` does reverse lookup correctly
- [ ] `GetZonesForRoom` returns zones across all domains
- [ ] Error sentinels added to `errors.go`
- [ ] `ZoneDomain` constants and validation function exist
- [ ] All 16 zone tests pass

### Quality Gates
- [ ] `cd code/core && go build ./...` succeeds
- [ ] `cd code/core && go test -race -count=1 ./...` — all packages pass
- [ ] `cd code/core && golangci-lint run` — zero warnings
- [ ] No new `//nolint` directives without justification comments

## Additional Guidance

Use your own judgement to identify and fix any additional issues you discover while working through the codebase. If you find edge cases not covered by the specified tests, add them. Document anything extra you find and fix in a summary at the end.
