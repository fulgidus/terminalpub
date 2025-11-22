-- Fix permissions for terminalpub database user
-- Run this as the postgres superuser

-- Grant usage on schema
GRANT USAGE ON SCHEMA public TO terminalpub;

-- Grant permissions on all existing tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO terminalpub;

-- Grant permissions on all sequences (for auto-increment IDs)
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO terminalpub;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO terminalpub;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO terminalpub;

-- Explicitly grant on specific tables that might have been missed
GRANT SELECT, INSERT, UPDATE, DELETE ON users TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON mastodon_apps TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON device_codes TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON mastodon_tokens TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_ssh_keys TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON sessions TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON posts TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON followers TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON following TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON activities TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON likes TO terminalpub;
GRANT SELECT, INSERT, UPDATE, DELETE ON boosts TO terminalpub;

-- Grant on sequences
GRANT USAGE, SELECT ON SEQUENCE users_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE mastodon_apps_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE device_codes_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE mastodon_tokens_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE user_ssh_keys_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE sessions_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE posts_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE followers_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE following_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE activities_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE likes_id_seq TO terminalpub;
GRANT USAGE, SELECT ON SEQUENCE boosts_id_seq TO terminalpub;

-- Verify permissions
SELECT grantee, privilege_type, table_name 
FROM information_schema.role_table_grants 
WHERE grantee = 'terminalpub'
ORDER BY table_name, privilege_type;
