-- MFA (Multi-Factor Authentication) Support
-- This migration adds tables for TOTP secrets and backup codes

-- MFA secrets table for storing TOTP secrets
CREATE TABLE IF NOT EXISTS mfa_secrets (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret_encrypted TEXT NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    enabled_at TIMESTAMP,
    last_used_at TIMESTAMP
);

-- MFA backup codes for account recovery
CREATE TABLE IF NOT EXISTS mfa_backup_codes (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster backup code lookups
CREATE INDEX IF NOT EXISTS idx_mfa_backup_codes_user_id ON mfa_backup_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_mfa_backup_codes_user_unused ON mfa_backup_codes(user_id) WHERE used = FALSE;

-- Sensitive operations audit log for financial security
CREATE TABLE IF NOT EXISTS sensitive_operations_log (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation_type VARCHAR(50) NOT NULL, -- 'wallet_address_change', 'password_change', 'mfa_disable', etc.
    ip_address VARCHAR(45),
    user_agent TEXT,
    mfa_verified BOOLEAN DEFAULT FALSE,
    old_value TEXT, -- encrypted or hashed
    new_value TEXT, -- encrypted or hashed
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'completed', 'rejected', 'expired'
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Index for audit queries
CREATE INDEX IF NOT EXISTS idx_sensitive_ops_user_id ON sensitive_operations_log(user_id);
CREATE INDEX IF NOT EXISTS idx_sensitive_ops_created_at ON sensitive_operations_log(created_at);
CREATE INDEX IF NOT EXISTS idx_sensitive_ops_type ON sensitive_operations_log(operation_type);

-- Add mfa_enabled column to users table if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'users' AND column_name = 'mfa_enabled') THEN
        ALTER TABLE users ADD COLUMN mfa_enabled BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

COMMENT ON TABLE mfa_secrets IS 'Stores encrypted TOTP secrets for multi-factor authentication';
COMMENT ON TABLE mfa_backup_codes IS 'Stores hashed backup codes for MFA recovery';
COMMENT ON TABLE sensitive_operations_log IS 'Audit log for sensitive financial and security operations';
