-- Migration 011: Add engagement/clout tracking to user profiles
-- These columns support the enhanced leaderboard and badge system

-- Add lifetime hashrate and reputation to user_profiles
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS lifetime_hashrate NUMERIC(20,2) DEFAULT 0;
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS reputation INTEGER DEFAULT 0;

-- Create index for faster leaderboard queries
CREATE INDEX IF NOT EXISTS idx_user_profiles_lifetime_hashrate ON user_profiles(lifetime_hashrate DESC);
CREATE INDEX IF NOT EXISTS idx_user_profiles_reputation ON user_profiles(reputation DESC);

-- Ensure all users have a profile entry for leaderboard queries
INSERT INTO user_profiles (user_id, forum_post_count, lifetime_hashrate, reputation)
SELECT id, 0, 0, 0 FROM users 
WHERE id NOT IN (SELECT user_id FROM user_profiles)
ON CONFLICT (user_id) DO NOTHING;
