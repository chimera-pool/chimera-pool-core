-- Migration 020: User In-App Notifications System
-- Adds tables for notification bell, user mentions, and admin broadcasts

-- User notifications table (for notification bell)
CREATE TABLE IF NOT EXISTS user_notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- reply, mention, reaction, system, broadcast, payout, worker_alert
    title VARCHAR(255) NOT NULL,
    message TEXT,
    link VARCHAR(500), -- Optional link to navigate to
    metadata JSONB, -- Additional data (message_id, channel_id, etc.)
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE
);

-- User mentions in messages
CREATE TABLE IF NOT EXISTS message_mentions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES channel_messages(id) ON DELETE CASCADE,
    mentioned_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(message_id, mentioned_user_id)
);

-- Admin broadcast messages
CREATE TABLE IF NOT EXISTS admin_broadcasts (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL REFERENCES users(id),
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    broadcast_type VARCHAR(50) NOT NULL DEFAULT 'all', -- all, miners_only, admins_only
    send_email BOOLEAN NOT NULL DEFAULT false,
    send_notification BOOLEAN NOT NULL DEFAULT true,
    recipient_count INT DEFAULT 0,
    email_sent_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_notifications_user ON user_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_user_notifications_unread ON user_notifications(user_id, is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_user_notifications_created ON user_notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_notifications_type ON user_notifications(type);
CREATE INDEX IF NOT EXISTS idx_message_mentions_user ON message_mentions(mentioned_user_id);
CREATE INDEX IF NOT EXISTS idx_message_mentions_message ON message_mentions(message_id);
CREATE INDEX IF NOT EXISTS idx_admin_broadcasts_created ON admin_broadcasts(created_at DESC);

-- Function to create notification when user is mentioned
CREATE OR REPLACE FUNCTION notify_on_mention()
RETURNS TRIGGER AS $$
DECLARE
    msg_content TEXT;
    msg_author_name VARCHAR(100);
    channel_id BIGINT;
BEGIN
    -- Get message details
    SELECT cm.content, u.username, cm.channel_id 
    INTO msg_content, msg_author_name, channel_id
    FROM channel_messages cm
    JOIN users u ON cm.user_id = u.id
    WHERE cm.id = NEW.message_id;
    
    -- Create notification for mentioned user
    INSERT INTO user_notifications (user_id, type, title, message, link, metadata)
    VALUES (
        NEW.mentioned_user_id,
        'mention',
        msg_author_name || ' mentioned you',
        LEFT(msg_content, 100),
        '/community',
        jsonb_build_object('message_id', NEW.message_id, 'channel_id', channel_id, 'author', msg_author_name)
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_notify_on_mention ON message_mentions;
CREATE TRIGGER trigger_notify_on_mention
    AFTER INSERT ON message_mentions
    FOR EACH ROW
    EXECUTE FUNCTION notify_on_mention();

-- Function to create notification when someone replies to a message
CREATE OR REPLACE FUNCTION notify_on_reply()
RETURNS TRIGGER AS $$
DECLARE
    original_user_id BIGINT;
    replier_name VARCHAR(100);
BEGIN
    -- Only process if this is a reply
    IF NEW.reply_to_id IS NULL THEN
        RETURN NEW;
    END IF;
    
    -- Get original message author
    SELECT user_id INTO original_user_id
    FROM channel_messages
    WHERE id = NEW.reply_to_id;
    
    -- Don't notify if replying to own message
    IF original_user_id = NEW.user_id THEN
        RETURN NEW;
    END IF;
    
    -- Get replier name
    SELECT username INTO replier_name FROM users WHERE id = NEW.user_id;
    
    -- Create notification
    INSERT INTO user_notifications (user_id, type, title, message, link, metadata)
    VALUES (
        original_user_id,
        'reply',
        replier_name || ' replied to your message',
        LEFT(NEW.content, 100),
        '/community',
        jsonb_build_object('message_id', NEW.id, 'channel_id', NEW.channel_id, 'replier', replier_name)
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_notify_on_reply ON channel_messages;
CREATE TRIGGER trigger_notify_on_reply
    AFTER INSERT ON channel_messages
    FOR EACH ROW
    EXECUTE FUNCTION notify_on_reply();

COMMENT ON TABLE user_notifications IS 'In-app notifications for users (notification bell)';
COMMENT ON TABLE message_mentions IS 'Tracks @username mentions in messages';
COMMENT ON TABLE admin_broadcasts IS 'Admin broadcast messages to all users';
