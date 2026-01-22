-- KNX Device Discovery Table
-- Version: 20260122_120000
--
-- Stores individual addresses of KNX devices discovered via passive bus monitoring.
-- Used for health checks (DeviceDescriptor_Read) without requiring manual configuration.
--
-- The bus monitor observes all traffic and records source addresses.
-- Over time, this builds a complete map of active devices on the bus.

CREATE TABLE knx_devices (
    -- Individual address in "area.line.device" format (e.g., "1.1.10")
    individual_address TEXT PRIMARY KEY,

    -- When this device was last seen sending a telegram (Unix timestamp)
    last_seen INTEGER NOT NULL,

    -- How many telegrams we've seen from this device
    message_count INTEGER NOT NULL DEFAULT 1
) STRICT;

-- Index for health check queries (most recently active devices first)
CREATE INDEX idx_knx_devices_last_seen ON knx_devices(last_seen DESC);
