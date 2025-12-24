-- Rollback statistics optimization

DROP TABLE IF EXISTS miner_stats_history;
DROP TABLE IF EXISTS pool_stats_history;

DROP INDEX IF EXISTS idx_shares_timestamp_valid;
DROP INDEX IF EXISTS idx_shares_timestamp_difficulty;
DROP INDEX IF EXISTS idx_shares_miner_timestamp_valid;
DROP INDEX IF EXISTS idx_pool_stats_bucket;
DROP INDEX IF EXISTS idx_miner_stats_bucket;
