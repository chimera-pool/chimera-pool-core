-- Migration 022: Wallet Enhancements - Rollback

-- Drop functions
DROP FUNCTION IF EXISTS update_wallet_allocation(INTEGER, DECIMAL);
DROP FUNCTION IF EXISTS toggle_wallet_status(INTEGER, VARCHAR);
DROP FUNCTION IF EXISTS auto_balance_wallet_allocations();

-- Drop indexes
DROP INDEX IF EXISTS idx_wallets_status;
DROP INDEX IF EXISTS idx_miner_groups_user;
DROP INDEX IF EXISTS idx_miner_wallet_assignments_wallet;
DROP INDEX IF EXISTS idx_miner_wallet_assignments_miner;

-- Drop miner-wallet assignments table
DROP TABLE IF EXISTS miner_wallet_assignments;

-- Remove columns from miners
ALTER TABLE miners DROP COLUMN IF EXISTS custom_name;
ALTER TABLE miners DROP COLUMN IF EXISTS miner_group_id;

-- Drop miner groups table
DROP TABLE IF EXISTS miner_groups;

-- Remove columns from wallets
ALTER TABLE wallets DROP COLUMN IF EXISTS min_payout_threshold;
ALTER TABLE wallets DROP COLUMN IF EXISTS is_locked;
ALTER TABLE wallets DROP COLUMN IF EXISTS label;
ALTER TABLE wallets DROP COLUMN IF EXISTS wallet_type;
ALTER TABLE wallets DROP COLUMN IF EXISTS status;
