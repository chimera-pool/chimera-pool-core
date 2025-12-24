-- Rollback: Remove network configurations tables

DROP TRIGGER IF EXISTS trg_network_configs_updated ON network_configs;
DROP FUNCTION IF EXISTS update_network_config_timestamp();
DROP FUNCTION IF EXISTS switch_active_network(VARCHAR, INTEGER, TEXT);
DROP INDEX IF EXISTS idx_network_configs_single_default;
DROP INDEX IF EXISTS idx_network_configs_algorithm;
DROP INDEX IF EXISTS idx_network_configs_symbol;
DROP INDEX IF EXISTS idx_network_configs_active;
DROP TABLE IF EXISTS network_switch_history;
DROP TABLE IF EXISTS network_configs;
