-- Drop triggers
DROP TRIGGER IF EXISTS update_mastodon_tokens_updated_at ON mastodon_tokens;
DROP TRIGGER IF EXISTS update_mastodon_apps_updated_at ON mastodon_apps;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS mastodon_tokens;
DROP TABLE IF EXISTS device_codes;
DROP TABLE IF EXISTS mastodon_apps;
