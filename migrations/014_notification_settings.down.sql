-- Migration 014: Rollback User Notification Settings

DROP TRIGGER IF EXISTS trigger_update_notification_settings_timestamp ON user_notification_settings;
DROP FUNCTION IF EXISTS update_notification_settings_timestamp();
DROP TABLE IF EXISTS notification_delivery_log;
DROP TABLE IF EXISTS alert_history;
DROP TABLE IF EXISTS user_notification_settings;
