-- Create posts table for storing user posts
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    content_type VARCHAR(50) DEFAULT 'text/plain',
    in_reply_to_id INTEGER REFERENCES posts(id) ON DELETE SET NULL,
    visibility VARCHAR(50) DEFAULT 'public', -- public, unlisted, followers, direct
    published_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- ActivityPub fields
    ap_id VARCHAR(512) UNIQUE, -- ActivityPub URI
    ap_type VARCHAR(50) DEFAULT 'Note', -- Note, Article, etc.
    ap_object JSONB -- Full ActivityPub object
);

CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_published_at ON posts(published_at DESC);
CREATE INDEX idx_posts_ap_id ON posts(ap_id);
CREATE INDEX idx_posts_in_reply_to ON posts(in_reply_to_id);

-- Create followers table for tracking follow relationships
CREATE TABLE IF NOT EXISTS followers (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    follower_actor_id VARCHAR(512) NOT NULL, -- ActivityPub actor URI
    follower_username VARCHAR(255), -- e.g., user@domain.com
    follower_inbox VARCHAR(512), -- Follower's inbox URL
    follower_shared_inbox VARCHAR(512), -- Shared inbox URL
    accepted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, follower_actor_id)
);

CREATE INDEX idx_followers_user_id ON followers(user_id);
CREATE INDEX idx_followers_actor_id ON followers(follower_actor_id);

-- Create following table for tracking who we follow
CREATE TABLE IF NOT EXISTS following (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_actor_id VARCHAR(512) NOT NULL, -- ActivityPub actor URI
    target_username VARCHAR(255), -- e.g., user@domain.com
    target_inbox VARCHAR(512), -- Target's inbox URL
    target_shared_inbox VARCHAR(512), -- Shared inbox URL
    accepted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, target_actor_id)
);

CREATE INDEX idx_following_user_id ON following(user_id);
CREATE INDEX idx_following_actor_id ON following(target_actor_id);

-- Create activities table for incoming/outgoing ActivityPub activities
CREATE TABLE IF NOT EXISTS activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL, -- Create, Update, Delete, Follow, Accept, Reject, Like, Announce, etc.
    actor_id VARCHAR(512) NOT NULL, -- ActivityPub actor URI
    object_id VARCHAR(512), -- ActivityPub object URI
    target_id VARCHAR(512), -- ActivityPub target URI
    activity_json JSONB NOT NULL, -- Full ActivityPub activity
    direction VARCHAR(10) NOT NULL, -- 'inbound' or 'outbound'
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_activities_user_id ON activities(user_id);
CREATE INDEX idx_activities_type ON activities(activity_type);
CREATE INDEX idx_activities_actor_id ON activities(actor_id);
CREATE INDEX idx_activities_direction ON activities(direction);
CREATE INDEX idx_activities_processed ON activities(processed);
CREATE INDEX idx_activities_created_at ON activities(created_at DESC);

-- Create likes table
CREATE TABLE IF NOT EXISTS likes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    actor_id VARCHAR(512) NOT NULL, -- Who liked (for remote likes)
    ap_id VARCHAR(512) UNIQUE, -- ActivityPub Like activity URI
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, post_id, actor_id)
);

CREATE INDEX idx_likes_user_id ON likes(user_id);
CREATE INDEX idx_likes_post_id ON likes(post_id);
CREATE INDEX idx_likes_actor_id ON likes(actor_id);

-- Create boosts/reblogs table
CREATE TABLE IF NOT EXISTS boosts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
    actor_id VARCHAR(512) NOT NULL, -- Who boosted (for remote boosts)
    ap_id VARCHAR(512) UNIQUE, -- ActivityPub Announce activity URI
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, post_id, actor_id)
);

CREATE INDEX idx_boosts_user_id ON boosts(user_id);
CREATE INDEX idx_boosts_post_id ON boosts(post_id);
CREATE INDEX idx_boosts_actor_id ON boosts(actor_id);

-- Add triggers for updated_at columns
CREATE TRIGGER update_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_followers_updated_at
    BEFORE UPDATE ON followers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_following_updated_at
    BEFORE UPDATE ON following
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
