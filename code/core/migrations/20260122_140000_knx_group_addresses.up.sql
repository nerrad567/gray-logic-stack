-- KNX Group Address Discovery Table
-- Version: 20260122_140000
--
-- Stores group addresses discovered via passive bus monitoring.
-- Used for Layer 3 health checks (GroupValue_Read) without requiring manual configuration.
--
-- The bus monitor observes all traffic and records destination group addresses.
-- Over time, this builds a map of active group addresses on the bus that can be
-- used for health checks when Layer 4 (DeviceDescriptor_Read) is not supported.

CREATE TABLE IF NOT EXISTS knx_group_addresses (
    -- Group address in "main/middle/sub" format (e.g., "1/2/3")
    group_address TEXT PRIMARY KEY,

    -- When this group address was last seen in a telegram (Unix timestamp)
    last_seen INTEGER NOT NULL,

    -- How many telegrams we've seen to/from this group address
    message_count INTEGER NOT NULL DEFAULT 1,

    -- Whether we've ever seen a read response from this address
    -- (indicates a device is bound to respond to reads on this GA)
    has_read_response INTEGER NOT NULL DEFAULT 0
) STRICT;

-- Index for health check queries (prefer addresses with known read responses)
CREATE INDEX IF NOT EXISTS idx_knx_group_addresses_health ON knx_group_addresses(has_read_response DESC, last_seen DESC);
