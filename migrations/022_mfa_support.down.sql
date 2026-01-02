-- Rollback MFA support migration

DROP TABLE IF EXISTS sensitive_operations_log;
DROP TABLE IF EXISTS mfa_backup_codes;
DROP TABLE IF EXISTS mfa_secrets;

-- Remove mfa_enabled column from users table
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled;
