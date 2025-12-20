-- Migration: Add roles to users and create channel tables
-- Requirements: Admin/Moderator role system, Channel management

-- Add role column to users table
ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user' 
    CHECK (role IN ('user', 'moderator', 'admin', 'super_admin'));

-- Create index for role-based queries
CREATE INDEX idx_users_role ON users(role);

-- Channel categories table
CREATE TABLE channel_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    position INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by BIGINT NOT NULL REFERENCES users(id)
);

CREATE INDEX idx_channel_categories_position ON channel_categories(position);

-- Channels table
CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES channel_categories(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(20) DEFAULT 'text' CHECK (type IN ('text', 'announcement', 'regional')),
    position INTEGER DEFAULT 0,
    is_read_only BOOLEAN DEFAULT false,
    admin_only_post BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by BIGINT NOT NULL REFERENCES users(id)
);

CREATE INDEX idx_channels_category_id ON channels(category_id);
CREATE INDEX idx_channels_type ON channels(type);
CREATE INDEX idx_channels_position ON channels(position);

-- Moderation log table
CREATE TABLE moderation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id BIGINT NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    details TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_moderation_log_actor_id ON moderation_log(actor_id);
CREATE INDEX idx_moderation_log_action ON moderation_log(action);
CREATE INDEX idx_moderation_log_created_at ON moderation_log(created_at);

-- Create triggers for updated_at on new tables
CREATE TRIGGER update_channel_categories_updated_at BEFORE UPDATE ON channel_categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default channel categories
INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'General',
    'General discussion and announcements',
    1,
    (SELECT id FROM users WHERE role = 'super_admin' LIMIT 1)
WHERE EXISTS (SELECT 1 FROM users WHERE role = 'super_admin');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Mining Talk',
    'Technical mining discussions',
    2,
    (SELECT id FROM users WHERE role = 'super_admin' LIMIT 1)
WHERE EXISTS (SELECT 1 FROM users WHERE role = 'super_admin');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Support',
    'Help and troubleshooting',
    3,
    (SELECT id FROM users WHERE role = 'super_admin' LIMIT 1)
WHERE EXISTS (SELECT 1 FROM users WHERE role = 'super_admin');
