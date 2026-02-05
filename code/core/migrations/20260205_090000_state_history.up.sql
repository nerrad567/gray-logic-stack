-- State History Schema for Gray Logic Core
-- Version: 20260205_090000
--
-- This migration creates the state history audit trail table for
-- recording device state snapshots over time.
--
-- Schema Rules (per database-schema.md):
--   - STRICT mode enforced for type safety
--   - Timestamps stored as TEXT in ISO 8601 format (UTC)
--   - Additive-only changes (no DROP/RENAME after production)

-- ============================================================================
-- STATE HISTORY
-- ============================================================================
-- Stores each device state change for local audit and history queries.

CREATE TABLE state_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,
    state TEXT NOT NULL,                 -- JSON-encoded state snapshot
    source TEXT NOT NULL DEFAULT 'mqtt', -- mqtt, command, scene
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_state_history_device ON state_history(device_id, created_at DESC);
CREATE INDEX idx_state_history_time ON state_history(created_at DESC);
