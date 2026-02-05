-- Rollback auth users migration
DROP TABLE IF EXISTS user_room_access;
DROP TABLE IF EXISTS panel_room_access;
DROP TABLE IF EXISTS panels;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
