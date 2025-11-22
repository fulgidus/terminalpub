-- Add SSH keys tracking to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS ssh_public_keys TEXT[] DEFAULT '{}';

-- Create index for SSH key lookups
CREATE INDEX IF NOT EXISTS idx_users_ssh_keys ON users USING GIN (ssh_public_keys);

-- Create user_ssh_keys table for better management (alternative approach)
CREATE TABLE IF NOT EXISTS user_ssh_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL UNIQUE,
    fingerprint VARCHAR(255) NOT NULL,
    key_type VARCHAR(50) NOT NULL,
    comment VARCHAR(255),
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_ssh_keys_user_id ON user_ssh_keys(user_id);
CREATE INDEX idx_user_ssh_keys_fingerprint ON user_ssh_keys(fingerprint);
CREATE INDEX idx_user_ssh_keys_public_key ON user_ssh_keys(public_key);

-- Add trigger for updated_at
CREATE TRIGGER update_user_ssh_keys_updated_at
    BEFORE UPDATE ON user_ssh_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
