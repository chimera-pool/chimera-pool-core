-- Migration 021: Login Security (Rollback)

DROP FUNCTION IF EXISTS cleanup_old_login_attempts();
DROP FUNCTION IF EXISTS record_successful_login(VARCHAR, VARCHAR, TEXT);
DROP FUNCTION IF EXISTS record_failed_login(VARCHAR, VARCHAR, TEXT, VARCHAR);
DROP FUNCTION IF EXISTS is_account_locked(VARCHAR);
DROP TABLE IF EXISTS account_lockouts;
DROP TABLE IF EXISTS login_attempts;
