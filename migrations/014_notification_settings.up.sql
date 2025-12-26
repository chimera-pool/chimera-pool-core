-- Migration 014: User Notification Settings
-- Adds tables for user notification preferences and alert history

-- User notification settings table
CREATE TABLE IF NOT EXISTS user_notification_settings (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    discord_webhook VARCHAR(500),
    phone_number VARCHAR(20),
    
    -- Per-alert-type settings
    worker_offline_enabled BOOLEAN NOT NULL DEFAULT true,
    worker_offline_delay INT NOT NULL DEFAULT 5, -- minutes
    hashrate_drop_enabled BOOLEAN NOT NULL DEFAULT true,
    hashrate_drop_percent INT NOT NULL DEFAULT 50,
    block_found_enabled BOOLEAN NOT NULL DEFAULT true,
    payout_enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Channel preferences
    email_enabled BOOLEAN NOT NULL DEFAULT true,
    discord_enabled BOOLEAN NOT NULL DEFAULT false,
    sms_enabled BOOLEAN NOT NULL DEFAULT false,
    
    -- Rate limiting
    max_alerts_per_hour INT NOT NULL DEFAULT 10,
    quiet_hours_start INT, -- Hour (0-23) when to stop sending alerts
    quiet_hours_end INT,   -- Hour (0-23) when to resume alerts
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Alert history table (for audit and deduplication)
CREATE TABLE IF NOT EXISTS alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_id VARCHAR(50) NOT NULL UNIQUE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    worker_id BIGINT,
    worker_name VARCHAR(100),
    metadata JSONB,
    is_resolved BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Notification delivery log
CREATE TABLE IF NOT EXISTS notification_delivery_log (
    id BIGSERIAL PRIMARY KEY,
    alert_id VARCHAR(50) NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    channel VARCHAR(20) NOT NULL, -- email, discord, sms
    destination VARCHAR(500) NOT NULL,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_alert_history_user ON alert_history(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_type ON alert_history(alert_type);
CREATE INDEX IF NOT EXISTS idx_alert_history_created ON alert_history(created_at);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_alert ON notification_delivery_log(alert_id);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_user ON notification_delivery_log(user_id);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_notification_settings_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_notification_settings_timestamp ON user_notification_settings;
CREATE TRIGGER trigger_update_notification_settings_timestamp
    BEFORE UPDATE ON user_notification_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_notification_settings_timestamp();

-- Initialize settings for existing users (from their profile email)
INSERT INTO user_notification_settings (user_id, email)
SELECT id, email FROM users
WHERE id NOT IN (SELECT user_id FROM user_notification_settings)
ON CONFLICT DO NOTHING;

COMMENT ON TABLE user_notification_settings IS 'User preferences for pool notifications';
COMMENT ON TABLE alert_history IS 'History of all alerts sent';
COMMENT ON TABLE notification_delivery_log IS 'Log of notification delivery attempts';
