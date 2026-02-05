-- Rollback: State History Schema for Gray Logic Core
-- Version: 20260205_090000
--
-- WARNING: This will DELETE ALL state history data.

DROP INDEX IF EXISTS idx_state_history_time;
DROP INDEX IF EXISTS idx_state_history_device;
DROP TABLE IF EXISTS state_history;
