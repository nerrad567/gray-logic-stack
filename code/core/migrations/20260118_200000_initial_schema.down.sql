-- Rollback: Initial Schema for Gray Logic Core
-- Version: 20260118_200000
--
-- WARNING: This will DELETE ALL DATA in these tables.
-- Only use for development/testing rollback.

-- Drop views first
DROP VIEW IF EXISTS devices_with_location;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS areas;
DROP TABLE IF EXISTS sites;
