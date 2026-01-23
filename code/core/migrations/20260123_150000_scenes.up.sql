-- Scene Schema for Gray Logic Core
-- Version: 20260123_150000
--
-- This migration creates the tables for:
--   - Scenes (predefined device action collections)
--   - Scene executions (execution history and tracking)
--
-- Schema Rules (per database-schema.md):
--   - STRICT mode enforced for type safety
--   - All tables use TEXT for UUIDs
--   - Timestamps stored as TEXT in ISO 8601 format (UTC)
--   - Additive-only changes (no DROP/RENAME after production)

-- ============================================================================
-- SCENES
-- ============================================================================
-- A scene is a named collection of device commands that can be activated
-- together. Actions within a scene can execute in parallel or sequentially,
-- with optional delays and fade transitions.

CREATE TABLE scenes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,

    -- Scope (optional location binding)
    room_id TEXT,
    area_id TEXT,

    -- Configuration
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 50,   -- 1-100; higher = more important

    -- UI metadata
    icon TEXT,
    colour TEXT,                             -- Hex colour (#RRGGBB)
    category TEXT,                           -- comfort, entertainment, daily, etc.

    -- Actions stored as JSON array of SceneAction objects
    actions TEXT NOT NULL DEFAULT '[]',

    -- Sort order for UI display
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE SET NULL,
    FOREIGN KEY (area_id) REFERENCES areas(id) ON DELETE SET NULL
) STRICT;

CREATE INDEX idx_scenes_slug ON scenes(slug);
CREATE INDEX idx_scenes_room_id ON scenes(room_id);
CREATE INDEX idx_scenes_area_id ON scenes(area_id);
CREATE INDEX idx_scenes_category ON scenes(category);
CREATE INDEX idx_scenes_enabled ON scenes(enabled);

-- ============================================================================
-- SCENE EXECUTIONS
-- ============================================================================
-- Tracks each activation of a scene for auditing and diagnostics.

CREATE TABLE scene_executions (
    id TEXT PRIMARY KEY,
    scene_id TEXT NOT NULL,

    -- Timing
    triggered_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    started_at TEXT,
    completed_at TEXT,

    -- Trigger information
    trigger_type TEXT NOT NULL DEFAULT 'manual',   -- manual, schedule, event, voice, automation
    trigger_source TEXT,                           -- api, wall_panel, mobile, etc.

    -- Status tracking
    status TEXT NOT NULL DEFAULT 'pending',        -- pending, running, completed, partial, failed, cancelled

    -- Action counts
    actions_total INTEGER NOT NULL DEFAULT 0,
    actions_completed INTEGER NOT NULL DEFAULT 0,
    actions_failed INTEGER NOT NULL DEFAULT 0,
    actions_skipped INTEGER NOT NULL DEFAULT 0,

    -- Failure details (JSON array of ActionFailure objects)
    failures TEXT,

    -- Duration in milliseconds
    duration_ms INTEGER,

    FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_scene_executions_scene_id ON scene_executions(scene_id);
CREATE INDEX idx_scene_executions_status ON scene_executions(status);
CREATE INDEX idx_scene_executions_triggered ON scene_executions(triggered_at);
