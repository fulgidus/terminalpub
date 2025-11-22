-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    email VARCHAR(255) UNIQUE,
    
    -- Mastodon primary account
    primary_mastodon_instance VARCHAR(255),
    primary_mastodon_id VARCHAR(100),
    primary_mastodon_acct VARCHAR(200),
    
    -- ActivityPub fields
    private_key TEXT,
    public_key TEXT,
    actor_url VARCHAR(500) UNIQUE,
    inbox_url VARCHAR(500),
    outbox_url VARCHAR(500),
    followers_url VARCHAR(500),
    following_url VARCHAR(500),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    bio TEXT,
    avatar_url TEXT
);

-- Create indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_actor_url ON users(actor_url);
CREATE INDEX idx_users_mastodon_acct ON users(primary_mastodon_acct);
