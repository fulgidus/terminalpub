-- Create mastodon_apps table
-- Stores registered OAuth apps for each Mastodon instance
CREATE TABLE IF NOT EXISTS mastodon_apps (
    id SERIAL PRIMARY KEY,
    instance_url VARCHAR(255) NOT NULL UNIQUE,
    client_id VARCHAR(255) NOT NULL,
    client_secret VARCHAR(255) NOT NULL,
    redirect_uri VARCHAR(255) NOT NULL,
    scopes TEXT NOT NULL DEFAULT 'read write follow',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mastodon_apps_instance ON mastodon_apps(instance_url);

-- Create device_codes table
-- Tracks pending OAuth Device Flow authorizations
CREATE TABLE IF NOT EXISTS device_codes (
    id SERIAL PRIMARY KEY,
    user_code VARCHAR(10) NOT NULL UNIQUE,
    device_code VARCHAR(255) NOT NULL UNIQUE,
    instance_url VARCHAR(255) NOT NULL,
    ssh_session_id VARCHAR(255) NOT NULL,
    verification_uri VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    authorized BOOLEAN NOT NULL DEFAULT FALSE,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_device_codes_user_code ON device_codes(user_code);
CREATE INDEX idx_device_codes_device_code ON device_codes(device_code);
CREATE INDEX idx_device_codes_ssh_session ON device_codes(ssh_session_id);
CREATE INDEX idx_device_codes_expires_at ON device_codes(expires_at);

-- Create mastodon_tokens table
-- Stores OAuth access tokens for user's Mastodon accounts
CREATE TABLE IF NOT EXISTS mastodon_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    instance_url VARCHAR(255) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50) NOT NULL DEFAULT 'Bearer',
    scopes TEXT NOT NULL,
    expires_at TIMESTAMP,
    mastodon_id VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    avatar_url TEXT,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, instance_url, mastodon_id)
);

CREATE INDEX idx_mastodon_tokens_user_id ON mastodon_tokens(user_id);
CREATE INDEX idx_mastodon_tokens_instance ON mastodon_tokens(instance_url);
CREATE INDEX idx_mastodon_tokens_mastodon_id ON mastodon_tokens(mastodon_id);
CREATE INDEX idx_mastodon_tokens_primary ON mastodon_tokens(user_id, is_primary) WHERE is_primary = TRUE;

-- Create sessions table
-- Tracks active SSH sessions
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    ip_address INET NOT NULL,
    anonymous BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_ip_address ON sessions(ip_address);

-- Add updated_at trigger for mastodon_apps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_mastodon_apps_updated_at
    BEFORE UPDATE ON mastodon_apps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_mastodon_tokens_updated_at
    BEFORE UPDATE ON mastodon_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
