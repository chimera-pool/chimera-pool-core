-- Rollback performance optimization indexes

DROP INDEX IF EXISTS idx_shares_user_timestamp;
DROP INDEX IF EXISTS idx_miners_active_hashrate;
DROP INDEX IF EXISTS idx_blocks_status_timestamp;
DROP INDEX IF EXISTS idx_payouts_user_status;
DROP INDEX IF EXISTS idx_equipment_metrics_time;
DROP INDEX IF EXISTS idx_messages_channel_time;
DROP INDEX IF EXISTS idx_posts_forum_time;
DROP INDEX IF EXISTS idx_bugs_status_priority;
