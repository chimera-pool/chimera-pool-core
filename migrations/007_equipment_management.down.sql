-- Rollback Equipment Management Migration
-- Migration 006 DOWN

DROP TRIGGER IF EXISTS trigger_update_pool_stats ON equipment;
DROP FUNCTION IF EXISTS update_pool_stats_cache();

DROP TABLE IF EXISTS equipment_alerts;
DROP TABLE IF EXISTS pool_stats_cache;
DROP TABLE IF EXISTS user_wallets;
DROP TABLE IF EXISTS payout_splits;
DROP TABLE IF EXISTS equipment_metrics_history;
DROP TABLE IF EXISTS equipment;
