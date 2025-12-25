-- Rollback Advanced Payout Options Migration

-- Drop views
DROP VIEW IF EXISTS v_pool_fee_summary;
DROP VIEW IF EXISTS v_user_payout_summary;

-- Drop aux chain tables
DROP TABLE IF EXISTS aux_payouts;
DROP TABLE IF EXISTS aux_blocks;
DROP TABLE IF EXISTS aux_chains;

-- Drop pplns config
DROP TABLE IF EXISTS pplns_config;

-- Drop payout calculation history
DROP TABLE IF EXISTS payout_calculations;

-- Remove added columns from payouts
ALTER TABLE payouts DROP COLUMN IF EXISTS payout_mode;
ALTER TABLE payouts DROP COLUMN IF EXISTS block_id;
ALTER TABLE payouts DROP COLUMN IF EXISTS share_difficulty;
ALTER TABLE payouts DROP COLUMN IF EXISTS fee_amount;

-- Drop pool fee config
DROP TABLE IF EXISTS pool_fee_config;

-- Drop user payout settings
DROP TABLE IF EXISTS user_payout_settings;

-- Drop payout mode enum
DROP TYPE IF EXISTS payout_mode;
