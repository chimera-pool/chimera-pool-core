-- Rollback migration: Multi-coin user statistics

DROP TRIGGER IF EXISTS trg_user_network_stats_updated ON user_network_stats;
DROP TRIGGER IF EXISTS trg_miner_network_assignments_updated ON miner_network_assignments;

DROP FUNCTION IF EXISTS get_user_aggregated_stats(BIGINT);
DROP FUNCTION IF EXISTS get_user_all_network_stats(BIGINT);
DROP FUNCTION IF EXISTS update_user_network_stats(BIGINT, UUID, DECIMAL, BIGINT, BIGINT);

DROP TABLE IF EXISTS network_stats_history;
DROP TABLE IF EXISTS network_pool_stats;
DROP TABLE IF EXISTS miner_network_assignments;
DROP TABLE IF EXISTS user_network_stats;
