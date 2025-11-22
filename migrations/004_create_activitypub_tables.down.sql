-- Drop ActivityPub tables in reverse order
DROP TRIGGER IF EXISTS update_following_updated_at ON following;
DROP TRIGGER IF EXISTS update_followers_updated_at ON followers;
DROP TRIGGER IF EXISTS update_posts_updated_at ON posts;

DROP TABLE IF EXISTS boosts;
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS following;
DROP TABLE IF EXISTS followers;
DROP TABLE IF EXISTS posts;
