-- Migration 013: Rollback Payout Execution Tables

DROP VIEW IF EXISTS v_pool_payout_stats;
DROP VIEW IF EXISTS v_payout_summary;
DROP TRIGGER IF EXISTS trigger_update_balance_timestamp ON user_balances;
DROP FUNCTION IF EXISTS update_balance_timestamp();
DROP TABLE IF EXISTS payout_transactions;
DROP TABLE IF EXISTS pending_payouts;
DROP TABLE IF EXISTS user_balances;
