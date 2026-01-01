-- Migration 020: User In-App Notifications System (Rollback)

DROP TRIGGER IF EXISTS trigger_notify_on_reply ON channel_messages;
DROP TRIGGER IF EXISTS trigger_notify_on_mention ON message_mentions;
DROP FUNCTION IF EXISTS notify_on_reply();
DROP FUNCTION IF EXISTS notify_on_mention();
DROP TABLE IF EXISTS admin_broadcasts;
DROP TABLE IF EXISTS message_mentions;
DROP TABLE IF EXISTS user_notifications;
