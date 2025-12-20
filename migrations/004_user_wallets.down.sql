-- Rollback: Remove multiple wallet support

-- Remove wallet_id from payouts
ALTER TABLE payouts DROP COLUMN IF EXISTS wallet_id;

-- Drop triggers
DROP TRIGGER IF EXISTS ensure_primary_wallet_trigger ON user_wallets;
DROP TRIGGER IF EXISTS validate_wallet_percentages_trigger ON user_wallets;
DROP TRIGGER IF EXISTS update_user_wallets_updated_at ON user_wallets;

-- Drop functions
DROP FUNCTION IF EXISTS ensure_primary_wallet();
DROP FUNCTION IF EXISTS validate_wallet_percentages();

-- Drop table
DROP TABLE IF EXISTS user_wallets;
