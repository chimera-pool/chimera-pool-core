-- Rollback referral system

DROP FUNCTION IF EXISTS get_effective_pool_fee(INTEGER);
DROP FUNCTION IF EXISTS apply_referral(INTEGER, VARCHAR);
DROP FUNCTION IF EXISTS generate_referral_code(VARCHAR);

DROP INDEX IF EXISTS idx_users_referred_by;
DROP INDEX IF EXISTS idx_referrals_status;
DROP INDEX IF EXISTS idx_referrals_referee_id;
DROP INDEX IF EXISTS idx_referrals_referrer_id;
DROP INDEX IF EXISTS idx_referral_codes_code;
DROP INDEX IF EXISTS idx_referral_codes_user_id;

DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS referral_codes;

ALTER TABLE users DROP COLUMN IF EXISTS referrer_discount_percent;
ALTER TABLE users DROP COLUMN IF EXISTS total_referrals;
ALTER TABLE users DROP COLUMN IF EXISTS referral_discount_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS referral_discount_percent;
ALTER TABLE users DROP COLUMN IF EXISTS referral_code_used;
ALTER TABLE users DROP COLUMN IF EXISTS referred_by;
