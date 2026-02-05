-- Add missing indexes on token_hash columns for auth hot paths.
-- These columns are queried on every authenticated request (panels)
-- and every token refresh/logout (refresh_tokens).

CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_panels_token_hash ON panels(token_hash);
