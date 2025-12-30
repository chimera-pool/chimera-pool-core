-- Migration 017 DOWN: Remove network_id from core tables
-- WARNING: This will remove network tracking - use with caution

-- Drop helper functions
DROP FUNCTION IF EXISTS get_user_network_mining_stats(BIGINT, UUID);
DROP FUNCTION IF EXISTS get_network_pool_stats(UUID);
DROP FUNCTION IF EXISTS get_active_network_id();

-- Drop indexes
DROP INDEX IF EXISTS idx_shares_network;
DROP INDEX IF EXISTS idx_shares_network_timestamp;
DROP INDEX IF EXISTS idx_shares_network_user;

DROP INDEX IF EXISTS idx_blocks_network;
DROP INDEX IF EXISTS idx_blocks_network_height;
DROP INDEX IF EXISTS idx_blocks_network_status;

DROP INDEX IF EXISTS idx_payouts_network;
DROP INDEX IF EXISTS idx_payouts_network_user;
DROP INDEX IF EXISTS idx_payouts_network_status;

DROP INDEX IF EXISTS idx_miners_network;
DROP INDEX IF EXISTS idx_miners_network_active;
DROP INDEX IF EXISTS idx_miners_network_user;

-- Remove network_id columns
ALTER TABLE shares DROP COLUMN IF EXISTS network_id;
ALTER TABLE blocks DROP COLUMN IF EXISTS network_id;
ALTER TABLE payouts DROP COLUMN IF EXISTS network_id;
ALTER TABLE miners DROP COLUMN IF EXISTS network_id;
