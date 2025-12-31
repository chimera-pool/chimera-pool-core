-- Migration 020 Down: Rollback database optimizations

-- Drop archive indexes
DROP INDEX IF EXISTS idx_shares_archive_miner;
DROP INDEX IF EXISTS idx_shares_archive_user;
DROP INDEX IF EXISTS idx_shares_archive_timestamp;

-- Drop archive table
DROP TABLE IF EXISTS shares_archive;

-- Drop functions
DROP FUNCTION IF EXISTS archive_old_shares();
DROP FUNCTION IF EXISTS perform_smart_maintenance();
DROP FUNCTION IF EXISTS should_run_maintenance();
DROP FUNCTION IF EXISTS record_activity_metrics();
DROP FUNCTION IF EXISTS create_shares_partition_if_needed();

-- Drop maintenance tables
DROP TABLE IF EXISTS db_activity_metrics;
DROP TABLE IF EXISTS db_maintenance_log;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_miner_stats_history_time;
DROP INDEX IF EXISTS idx_miners_active_stats;
DROP INDEX IF EXISTS idx_shares_network_timestamp;
DROP INDEX IF EXISTS idx_shares_user_valid_timestamp;
DROP INDEX IF EXISTS idx_shares_miner_timestamp;
