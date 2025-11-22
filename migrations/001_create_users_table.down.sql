-- Drop indexes
DROP INDEX IF EXISTS idx_users_mastodon_acct;
DROP INDEX IF EXISTS idx_users_actor_url;
DROP INDEX IF EXISTS idx_users_username;

-- Drop table
DROP TABLE IF EXISTS users;
