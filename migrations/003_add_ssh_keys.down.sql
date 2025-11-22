-- Drop user_ssh_keys table
DROP TRIGGER IF EXISTS update_user_ssh_keys_updated_at ON user_ssh_keys;
DROP TABLE IF EXISTS user_ssh_keys;

-- Drop SSH keys from users table
DROP INDEX IF EXISTS idx_users_ssh_keys;
ALTER TABLE users DROP COLUMN IF EXISTS ssh_public_keys;
