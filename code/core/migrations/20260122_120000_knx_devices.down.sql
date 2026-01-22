-- Rollback: KNX Device Discovery Table
DROP INDEX IF EXISTS idx_knx_devices_last_seen;
DROP TABLE IF EXISTS knx_devices;
