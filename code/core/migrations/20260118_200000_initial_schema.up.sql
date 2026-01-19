-- Initial Schema for Gray Logic Core
-- Version: 20260118_200000
--
-- This migration creates the foundational tables for:
--   - Sites (installations)
--   - Areas (floors, buildings, zones)
--   - Rooms (physical spaces)
--   - Devices (controllable entities)
--   - Audit logs (change tracking)
--
-- Schema Rules (per database-schema.md):
--   - STRICT mode enforced for type safety
--   - All tables use TEXT for UUIDs
--   - Timestamps stored as TEXT in ISO 8601 format (UTC)
--   - Additive-only changes (no DROP/RENAME after production)

-- ============================================================================
-- SITES
-- ============================================================================
-- A site represents a single Gray Logic installation (home, office, building).

CREATE TABLE sites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,

    -- Location information
    address TEXT,
    latitude REAL,
    longitude REAL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    elevation_m REAL,

    -- Mode management
    modes_available TEXT NOT NULL DEFAULT '["home","away","night","holiday"]',  -- JSON array
    mode_current TEXT NOT NULL DEFAULT 'home',

    -- Settings (JSON object)
    settings TEXT NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
) STRICT;

CREATE INDEX idx_sites_slug ON sites(slug);

-- ============================================================================
-- AREAS
-- ============================================================================
-- Areas are logical groupings within a site (floors, buildings, zones).

CREATE TABLE areas (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,

    -- Classification
    type TEXT NOT NULL DEFAULT 'floor',  -- floor, building, wing, zone, outdoor, utility
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE (site_id, slug)
) STRICT;

CREATE INDEX idx_areas_site_id ON areas(site_id);
CREATE INDEX idx_areas_sort ON areas(site_id, sort_order);

-- ============================================================================
-- ROOMS
-- ============================================================================
-- Rooms are physical spaces within areas.

CREATE TABLE rooms (
    id TEXT PRIMARY KEY,
    area_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,

    -- Classification
    type TEXT NOT NULL DEFAULT 'other',  -- living, bedroom, bathroom, kitchen, etc.
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Zone associations (nullable)
    climate_zone_id TEXT,
    audio_zone_id TEXT,

    -- Room-specific settings (JSON object)
    settings TEXT NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE CASCADE,
    UNIQUE (area_id, slug)
) STRICT;

CREATE INDEX idx_rooms_area_id ON rooms(area_id);
CREATE INDEX idx_rooms_sort ON rooms(area_id, sort_order);
CREATE INDEX idx_rooms_climate_zone ON rooms(climate_zone_id);
CREATE INDEX idx_rooms_audio_zone ON rooms(audio_zone_id);

-- ============================================================================
-- DEVICES
-- ============================================================================
-- Devices are any controllable or monitorable entities.

CREATE TABLE devices (
    id TEXT PRIMARY KEY,

    -- Location (one of room_id or area_id should be set)
    room_id TEXT,
    area_id TEXT,

    -- Identity
    name TEXT NOT NULL,
    slug TEXT NOT NULL,

    -- Classification
    type TEXT NOT NULL,          -- light_switch, light_dimmer, thermostat, etc.
    domain TEXT NOT NULL,        -- lighting, climate, blinds, security, etc.

    -- Protocol information
    protocol TEXT NOT NULL,      -- knx, dali, modbus, mqtt, etc.
    address TEXT NOT NULL,       -- JSON: protocol-specific address
    gateway_id TEXT,             -- Reference to gateway device (if applicable)

    -- Capabilities (JSON array of capability identifiers)
    capabilities TEXT NOT NULL DEFAULT '[]',

    -- Device-specific configuration (JSON object)
    config TEXT NOT NULL DEFAULT '{}',

    -- Current state (JSON object)
    state TEXT NOT NULL DEFAULT '{}',
    state_updated_at TEXT,

    -- Health monitoring
    health_status TEXT NOT NULL DEFAULT 'unknown',  -- online, offline, degraded, unknown
    health_last_seen TEXT,
    phm_enabled INTEGER NOT NULL DEFAULT 0,         -- Boolean: PHM monitoring active
    phm_baseline TEXT,                              -- JSON: learned baseline data

    -- Metadata
    manufacturer TEXT,
    model TEXT,
    firmware_version TEXT,

    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE SET NULL,
    FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE SET NULL,
    FOREIGN KEY (gateway_id) REFERENCES devices(id) ON DELETE SET NULL
) STRICT;

CREATE INDEX idx_devices_room_id ON devices(room_id);
CREATE INDEX idx_devices_area_id ON devices(area_id);
CREATE INDEX idx_devices_domain ON devices(domain);
CREATE INDEX idx_devices_protocol ON devices(protocol);
CREATE INDEX idx_devices_type ON devices(type);
CREATE INDEX idx_devices_health ON devices(health_status);
CREATE INDEX idx_devices_gateway ON devices(gateway_id);

-- ============================================================================
-- AUDIT LOGS
-- ============================================================================
-- Audit logs track all changes for security and debugging.

CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,

    -- What happened
    action TEXT NOT NULL,        -- create, update, delete, command, login, etc.
    entity_type TEXT NOT NULL,   -- site, area, room, device, scene, user, etc.
    entity_id TEXT,              -- ID of affected entity (nullable for system events)

    -- Who did it
    user_id TEXT,                -- User who triggered action (nullable for system)
    source TEXT NOT NULL,        -- api, mqtt, scheduler, voice, system, etc.

    -- Details
    details TEXT,                -- JSON: additional context

    -- When
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
) STRICT;

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

-- ============================================================================
-- VIEWS
-- ============================================================================
-- Convenience views for common queries.

-- View: Devices with their location hierarchy
CREATE VIEW devices_with_location AS
SELECT
    d.*,
    r.name AS room_name,
    r.slug AS room_slug,
    a.id AS resolved_area_id,
    a.name AS area_name,
    a.slug AS area_slug,
    s.id AS site_id,
    s.name AS site_name,
    s.slug AS site_slug
FROM devices d
LEFT JOIN rooms r ON d.room_id = r.id
LEFT JOIN areas a ON COALESCE(d.area_id, r.area_id) = a.id
LEFT JOIN sites s ON a.site_id = s.id;
