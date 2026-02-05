-- Reverse: remove token_hash indexes
DROP INDEX IF EXISTS idx_refresh_tokens_hash;
DROP INDEX IF EXISTS idx_panels_token_hash;
