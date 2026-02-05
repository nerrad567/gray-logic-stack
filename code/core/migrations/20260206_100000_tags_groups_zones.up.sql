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
