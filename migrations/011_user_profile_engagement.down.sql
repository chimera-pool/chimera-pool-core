-- Rollback Migration 011: Remove engagement tracking columns

DROP INDEX IF EXISTS idx_user_profiles_lifetime_hashrate;
DROP INDEX IF EXISTS idx_user_profiles_reputation;

ALTER TABLE user_profiles DROP COLUMN IF EXISTS lifetime_hashrate;
ALTER TABLE user_profiles DROP COLUMN IF EXISTS reputation;
