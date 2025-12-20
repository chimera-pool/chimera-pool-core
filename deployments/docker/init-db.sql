-- Chimera Pool Database Initialization Script
-- This script runs when the PostgreSQL container is first created

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    payout_address VARCHAR(255),
    pool_fee_percent DECIMAL(5,2) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    is_admin BOOLEAN DEFAULT false,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin', 'super_admin'))
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Create indexes for users
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);

-- Miners table
CREATE TABLE IF NOT EXISTS miners (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    address INET,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    hashrate DECIMAL(20,2) DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- Geolocation fields
    latitude DECIMAL(10,7),
    longitude DECIMAL(10,7),
    city VARCHAR(100),
    country VARCHAR(100),
    country_code CHAR(2),
    continent VARCHAR(50),
    geo_updated_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for miners
CREATE INDEX IF NOT EXISTS idx_miners_user_id ON miners(user_id);
CREATE INDEX IF NOT EXISTS idx_miners_active ON miners(is_active);
CREATE INDEX IF NOT EXISTS idx_miners_last_seen ON miners(last_seen);
CREATE INDEX IF NOT EXISTS idx_miners_country_code ON miners(country_code);
CREATE INDEX IF NOT EXISTS idx_miners_geo ON miners(latitude, longitude);

-- Shares table
CREATE TABLE IF NOT EXISTS shares (
    id BIGSERIAL PRIMARY KEY,
    miner_id BIGINT NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    difficulty DECIMAL(20,8) NOT NULL,
    is_valid BOOLEAN NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    nonce VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL
);

-- Create indexes for shares
CREATE INDEX IF NOT EXISTS idx_shares_miner_id ON shares(miner_id);
CREATE INDEX IF NOT EXISTS idx_shares_user_id ON shares(user_id);
CREATE INDEX IF NOT EXISTS idx_shares_timestamp ON shares(timestamp);
CREATE INDEX IF NOT EXISTS idx_shares_valid ON shares(is_valid);

-- Blocks table
CREATE TABLE IF NOT EXISTS blocks (
    id BIGSERIAL PRIMARY KEY,
    height BIGINT UNIQUE NOT NULL,
    hash VARCHAR(64) UNIQUE NOT NULL,
    finder_id BIGINT NOT NULL REFERENCES users(id),
    reward BIGINT NOT NULL,
    difficulty DECIMAL(20,8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'orphaned'))
);

-- Create indexes for blocks
CREATE INDEX IF NOT EXISTS idx_blocks_height ON blocks(height);
CREATE INDEX IF NOT EXISTS idx_blocks_finder_id ON blocks(finder_id);
CREATE INDEX IF NOT EXISTS idx_blocks_status ON blocks(status);
CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(timestamp);

-- Payouts table
CREATE TABLE IF NOT EXISTS payouts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    address VARCHAR(255) NOT NULL,
    tx_hash VARCHAR(64),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'confirmed', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for payouts
CREATE INDEX IF NOT EXISTS idx_payouts_user_id ON payouts(user_id);
CREATE INDEX IF NOT EXISTS idx_payouts_status ON payouts(status);
CREATE INDEX IF NOT EXISTS idx_payouts_created_at ON payouts(created_at);

-- Password reset tokens table
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for password reset tokens
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);

-- Wallet address history table
CREATE TABLE IF NOT EXISTS wallet_address_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address VARCHAR(255) NOT NULL,
    total_paid DECIMAL(20,8) DEFAULT 0,
    payout_count INTEGER DEFAULT 0,
    set_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    replaced_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for wallet address history
CREATE INDEX IF NOT EXISTS idx_wallet_history_user_id ON wallet_address_history(user_id);
CREATE INDEX IF NOT EXISTS idx_wallet_history_address ON wallet_address_history(address);
CREATE INDEX IF NOT EXISTS idx_wallet_history_set_at ON wallet_address_history(set_at);

-- Pool settings table
CREATE TABLE IF NOT EXISTS pool_settings (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default pool settings
INSERT INTO pool_settings (key, value, description) VALUES
    ('pool_fee', '1.0', 'Pool fee percentage'),
    ('min_payout', '1.0', 'Minimum payout threshold in BDAG'),
    ('payment_interval', '3600', 'Payment interval in seconds'),
    ('network', 'BlockDAG Awakening', 'Network name'),
    ('currency', 'BDAG', 'Currency symbol'),
    ('wallet_address', '0xD393798C098FFe3d64d4Ca531158D3562D00b66e', 'Pool wallet address'),
    ('algorithm', 'blake3', 'Current mining algorithm'),
    ('algorithm_variant', 'standard', 'Algorithm variant (standard, blockdag-custom, etc)'),
    ('difficulty_target', '1.0', 'Base difficulty target'),
    ('block_time', '10', 'Target block time in seconds'),
    ('stratum_port', '3333', 'Stratum server port'),
    ('algorithm_params', '{}', 'JSON parameters for algorithm customization')
ON CONFLICT (key) DO NOTHING;

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================
-- COMMUNITY SECTION TABLES
-- ============================================

-- Badges table
CREATE TABLE IF NOT EXISTS badges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    icon VARCHAR(10),
    color VARCHAR(7) DEFAULT '#00d4ff',
    badge_type VARCHAR(20) NOT NULL CHECK (badge_type IN ('hashrate', 'activity', 'special', 'moderator')),
    requirement_value BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User badges junction table
CREATE TABLE IF NOT EXISTS user_badges (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id INT NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
    earned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_primary BOOLEAN DEFAULT false,
    UNIQUE(user_id, badge_id)
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user ON user_badges(user_id);

-- User profiles extended
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    avatar_url VARCHAR(500),
    bio TEXT,
    country VARCHAR(100),
    country_code CHAR(2),
    show_earnings BOOLEAN DEFAULT true,
    show_country BOOLEAN DEFAULT true,
    reputation INT DEFAULT 0,
    forum_post_count INT DEFAULT 0,
    lifetime_hashrate DECIMAL(30,2) DEFAULT 0,
    online_status VARCHAR(20) DEFAULT 'offline' CHECK (online_status IN ('online', 'offline', 'away', 'mining')),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Channel categories
CREATE TABLE IF NOT EXISTS channel_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Channels (Discord-like)
CREATE TABLE IF NOT EXISTS channels (
    id SERIAL PRIMARY KEY,
    category_id INT REFERENCES channel_categories(id) ON DELETE SET NULL,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    channel_type VARCHAR(20) DEFAULT 'text' CHECK (channel_type IN ('text', 'announcement', 'regional')),
    is_read_only BOOLEAN DEFAULT false,
    admin_only_post BOOLEAN DEFAULT false,
    country_code CHAR(2),
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_channels_category ON channels(category_id);

-- Channel messages
CREATE TABLE IF NOT EXISTS channel_messages (
    id BIGSERIAL PRIMARY KEY,
    channel_id INT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    reply_to_id BIGINT REFERENCES channel_messages(id) ON DELETE SET NULL,
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_channel_messages_channel ON channel_messages(channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_user ON channel_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_created ON channel_messages(created_at DESC);

-- Message reactions
CREATE TABLE IF NOT EXISTS message_reactions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES channel_messages(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(10) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(message_id, user_id, emoji)
);
CREATE INDEX IF NOT EXISTS idx_reactions_message ON message_reactions(message_id);

-- Forum categories
CREATE TABLE IF NOT EXISTS forum_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(10),
    admin_only_post BOOLEAN DEFAULT false,
    sort_order INT DEFAULT 0,
    post_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Forum posts
CREATE TABLE IF NOT EXISTS forum_posts (
    id BIGSERIAL PRIMARY KEY,
    category_id INT NOT NULL REFERENCES forum_categories(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    tags VARCHAR(50)[],
    view_count INT DEFAULT 0,
    reply_count INT DEFAULT 0,
    upvotes INT DEFAULT 0,
    downvotes INT DEFAULT 0,
    is_pinned BOOLEAN DEFAULT false,
    is_locked BOOLEAN DEFAULT false,
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_forum_posts_category ON forum_posts(category_id);
CREATE INDEX IF NOT EXISTS idx_forum_posts_user ON forum_posts(user_id);
CREATE INDEX IF NOT EXISTS idx_forum_posts_created ON forum_posts(created_at DESC);

-- Forum replies
CREATE TABLE IF NOT EXISTS forum_replies (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_reply_id BIGINT REFERENCES forum_replies(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    upvotes INT DEFAULT 0,
    downvotes INT DEFAULT 0,
    is_solution BOOLEAN DEFAULT false,
    is_edited BOOLEAN DEFAULT false,
    is_deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_forum_replies_post ON forum_replies(post_id);
CREATE INDEX IF NOT EXISTS idx_forum_replies_user ON forum_replies(user_id);

-- Post/Reply votes
CREATE TABLE IF NOT EXISTS content_votes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_type VARCHAR(20) NOT NULL CHECK (content_type IN ('post', 'reply')),
    content_id BIGINT NOT NULL,
    vote_type SMALLINT NOT NULL CHECK (vote_type IN (-1, 1)),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, content_type, content_id)
);

-- Direct messages
CREATE TABLE IF NOT EXISTS direct_messages (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT false,
    is_deleted_sender BOOLEAN DEFAULT false,
    is_deleted_receiver BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dm_sender ON direct_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_dm_receiver ON direct_messages(receiver_id);
CREATE INDEX IF NOT EXISTS idx_dm_conversation ON direct_messages(sender_id, receiver_id);

-- Blocked users
CREATE TABLE IF NOT EXISTS blocked_users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, blocked_user_id)
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    title VARCHAR(200),
    content TEXT,
    link VARCHAR(500),
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = false;

-- User mentions
CREATE TABLE IF NOT EXISTS mentions (
    id BIGSERIAL PRIMARY KEY,
    mentioned_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mentioning_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_type VARCHAR(20) NOT NULL CHECK (content_type IN ('message', 'post', 'reply')),
    content_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mentions_user ON mentions(mentioned_user_id);

-- Moderation actions
CREATE TABLE IF NOT EXISTS moderation_actions (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL REFERENCES users(id),
    target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action_type VARCHAR(50) NOT NULL CHECK (action_type IN ('ban', 'unban', 'mute', 'unmute', 'delete_message', 'delete_post', 'warn')),
    reason TEXT,
    duration_minutes INT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mod_actions_target ON moderation_actions(target_user_id);

-- Community bans/mutes
CREATE TABLE IF NOT EXISTS community_restrictions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    restriction_type VARCHAR(20) NOT NULL CHECK (restriction_type IN ('ban', 'mute')),
    reason TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, restriction_type)
);
CREATE INDEX IF NOT EXISTS idx_restrictions_user ON community_restrictions(user_id);

-- Reports
CREATE TABLE IF NOT EXISTS content_reports (
    id BIGSERIAL PRIMARY KEY,
    reporter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_type VARCHAR(20) NOT NULL CHECK (content_type IN ('message', 'post', 'reply', 'user')),
    content_id BIGINT NOT NULL,
    reason VARCHAR(100) NOT NULL,
    details TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'reviewed', 'actioned', 'dismissed')),
    reviewed_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_reports_status ON content_reports(status);

-- Insert default badges
INSERT INTO badges (name, description, icon, color, badge_type, requirement_value) VALUES
    ('Newcomer', 'Welcome to the pool!', '🌱', '#4ade80', 'hashrate', 0),
    ('Miner', 'Contributing hash power', '⛏️', '#60a5fa', 'hashrate', 100000000),
    ('Contributor', 'Significant pool contributor', '💎', '#a78bfa', 'hashrate', 1000000000),
    ('Power Miner', 'Major mining force', '⚡', '#f59e0b', 'hashrate', 10000000000),
    ('Elite Miner', 'Elite tier contributor', '🔥', '#ef4444', 'hashrate', 100000000000),
    ('Legendary Miner', 'Legendary status achieved', '👑', '#fbbf24', 'hashrate', 1000000000000),
    ('Pool Champion', 'Top 3 all-time contributor', '🏆', '#fcd34d', 'special', NULL),
    ('First Block', 'Found your first block!', '🎯', '#34d399', 'activity', 1),
    ('Block Hunter', 'Found 10+ blocks', '🎪', '#f472b6', 'activity', 10),
    ('Loyal Miner', '1 year of continuous mining', '🎖️', '#c084fc', 'activity', 365),
    ('Early Adopter', 'Among first 100 users', '🚀', '#22d3ee', 'special', NULL),
    ('Helpful', '50+ helpful forum posts', '🤝', '#4ade80', 'activity', 50),
    ('Community Leader', 'Moderator status', '🛡️', '#f97316', 'moderator', NULL)
ON CONFLICT (name) DO NOTHING;

-- Insert default channel categories
INSERT INTO channel_categories (name, description, sort_order) VALUES
    ('General', 'Welcome and announcements', 1),
    ('Mining Talk', 'Technical mining discussions', 2),
    ('Regional', 'Connect with miners in your region', 3),
    ('Support', 'Get help and troubleshooting', 4),
    ('Off-Topic', 'Casual conversations', 5)
ON CONFLICT DO NOTHING;

-- Insert default channels
INSERT INTO channels (category_id, name, description, channel_type, is_read_only, admin_only_post, sort_order) VALUES
    (1, 'welcome', 'Welcome to Chimeria Pool! Read the rules here.', 'announcement', true, true, 1),
    (1, 'announcements', 'Official pool announcements', 'announcement', false, true, 2),
    (1, 'general-chat', 'General discussion for all topics', 'text', false, false, 3),
    (2, 'mining-help', 'Get help with mining setup and issues', 'text', false, false, 1),
    (2, 'hardware-setup', 'Discuss mining hardware and configurations', 'text', false, false, 2),
    (2, 'earnings-discussion', 'Talk about payouts and earnings', 'text', false, false, 3),
    (2, 'pool-suggestions', 'Suggest features and improvements', 'text', false, false, 4),
    (3, 'north-america', 'North American miners', 'regional', false, false, 1),
    (3, 'europe', 'European miners', 'regional', false, false, 2),
    (3, 'asia', 'Asian miners', 'regional', false, false, 3),
    (3, 'south-america', 'South American miners', 'regional', false, false, 4),
    (3, 'oceania', 'Oceania miners', 'regional', false, false, 5),
    (3, 'africa', 'African miners', 'regional', false, false, 6),
    (4, 'tech-support', 'Technical support and troubleshooting', 'text', false, false, 1),
    (4, 'bug-reports', 'Report bugs and issues', 'text', false, false, 2),
    (5, 'lounge', 'Casual off-topic chat', 'text', false, false, 1),
    (5, 'introductions', 'Introduce yourself to the community', 'text', false, false, 2)
ON CONFLICT DO NOTHING;

-- Insert default forum categories
INSERT INTO forum_categories (name, description, icon, admin_only_post, sort_order) VALUES
    ('Announcements', 'Official pool announcements and updates', '📢', true, 1),
    ('General Discussion', 'General mining and pool discussions', '💬', false, 2),
    ('Mining Guides & Tutorials', 'Helpful guides and how-tos', '📚', false, 3),
    ('Hardware Reviews', 'Share and discuss mining hardware', '🖥️', false, 4),
    ('Bug Reports', 'Report issues with the pool', '🐛', false, 5),
    ('Feature Requests', 'Suggest new features', '💡', false, 6),
    ('Marketplace', 'Buy, sell, or trade mining equipment', '🛒', false, 7),
    ('Success Stories', 'Share your mining achievements', '🎉', false, 8)
ON CONFLICT DO NOTHING;

-- Create triggers for updated_at
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_miners_updated_at ON miners;
CREATE TRIGGER update_miners_updated_at BEFORE UPDATE ON miners
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create admin user (password: Champions$1956)
-- bcrypt hash generated for 'Champions$1956'
INSERT INTO users (username, email, password_hash, is_active, is_admin) VALUES
    ('admin', 'reid@blockdaginvestors.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjqQQZo2WZK5KFYU4E4VQXBX/p.Nqm', true, true)
ON CONFLICT (username) DO NOTHING;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO chimera;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO chimera;

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'Chimera Pool database initialized successfully!';
END $$;
