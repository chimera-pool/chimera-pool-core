-- Performance optimization indexes
-- Based on deep-dive audit findings

-- Composite index for share queries by user and time
-- Improves: PPLNS calculations, user stats queries
CREATE INDEX IF NOT EXISTS idx_shares_user_timestamp ON shares(user_id, timestamp DESC);

-- Partial index for active miners with hashrate
-- Improves: Pool stats, active miner counts
CREATE INDEX IF NOT EXISTS idx_miners_active_hashrate ON miners(hashrate) WHERE is_active = true;

-- Composite index for blocks by status and time
-- Improves: Recent block queries, confirmed block lookups
CREATE INDEX IF NOT EXISTS idx_blocks_status_timestamp ON blocks(status, timestamp DESC);

-- Composite index for payouts by user and status
-- Improves: User payout history, pending payout queries
CREATE INDEX IF NOT EXISTS idx_payouts_user_status ON payouts(user_id, status);

-- Index for equipment metrics time-series queries
-- Improves: Dashboard charts, equipment history
CREATE INDEX IF NOT EXISTS idx_equipment_metrics_time ON equipment_metrics_history(equipment_id, recorded_at DESC);

-- Index for community messages by channel and time
-- Improves: Message loading, chat scrolling
CREATE INDEX IF NOT EXISTS idx_messages_channel_time ON channel_messages(channel_id, created_at DESC);

-- Index for forum posts by forum and time
-- Improves: Forum listing, recent posts
CREATE INDEX IF NOT EXISTS idx_posts_forum_time ON forum_posts(forum_id, created_at DESC);

-- Index for bug reports by status
-- Improves: Admin bug listing, status filtering
CREATE INDEX IF NOT EXISTS idx_bugs_status_priority ON bug_reports(status, priority);
