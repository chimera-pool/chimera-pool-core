-- Migration 013: Payout Execution Tables
-- Adds tables for pending payouts and user balances

-- User balances table
CREATE TABLE IF NOT EXISTS user_balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0,
    pending_balance BIGINT NOT NULL DEFAULT 0,
    total_paid BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT balance_non_negative CHECK (balance >= 0),
    CONSTRAINT pending_non_negative CHECK (pending_balance >= 0)
);

-- Pending payouts table
CREATE TABLE IF NOT EXISTS pending_payouts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    address VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payout_mode VARCHAR(20) NOT NULL DEFAULT 'pplns',
    block_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    tx_hash VARCHAR(100),
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    CONSTRAINT amount_positive CHECK (amount > 0),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'processed', 'failed', 'cancelled'))
);

-- Payout transactions table (for audit trail)
CREATE TABLE IF NOT EXISTS payout_transactions (
    id BIGSERIAL PRIMARY KEY,
    payout_id BIGINT NOT NULL REFERENCES pending_payouts(id),
    tx_hash VARCHAR(100) NOT NULL,
    amount BIGINT NOT NULL,
    fee BIGINT NOT NULL DEFAULT 0,
    confirmations INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_pending_payouts_status ON pending_payouts(status);
CREATE INDEX IF NOT EXISTS idx_pending_payouts_user ON pending_payouts(user_id);
CREATE INDEX IF NOT EXISTS idx_pending_payouts_created ON pending_payouts(created_at);
CREATE INDEX IF NOT EXISTS idx_payout_transactions_payout ON payout_transactions(payout_id);
CREATE INDEX IF NOT EXISTS idx_payout_transactions_tx ON payout_transactions(tx_hash);

-- Add trigger to update user_balances.updated_at
CREATE OR REPLACE FUNCTION update_balance_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_balance_timestamp ON user_balances;
CREATE TRIGGER trigger_update_balance_timestamp
    BEFORE UPDATE ON user_balances
    FOR EACH ROW
    EXECUTE FUNCTION update_balance_timestamp();

-- View for payout summary
CREATE OR REPLACE VIEW v_payout_summary AS
SELECT 
    u.id as user_id,
    u.username,
    ub.balance as current_balance,
    ub.pending_balance,
    ub.total_paid,
    COUNT(pp.id) FILTER (WHERE pp.status = 'pending') as pending_payouts,
    COUNT(pp.id) FILTER (WHERE pp.status = 'processed') as completed_payouts,
    COUNT(pp.id) FILTER (WHERE pp.status = 'failed') as failed_payouts,
    MAX(pp.processed_at) as last_payout_at
FROM users u
LEFT JOIN user_balances ub ON u.id = ub.user_id
LEFT JOIN pending_payouts pp ON u.id = pp.user_id
GROUP BY u.id, u.username, ub.balance, ub.pending_balance, ub.total_paid;

-- View for pool payout stats
CREATE OR REPLACE VIEW v_pool_payout_stats AS
SELECT 
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status = 'processed') as processed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COALESCE(SUM(amount) FILTER (WHERE status = 'processed'), 0) as total_paid,
    COALESCE(SUM(amount) FILTER (WHERE status = 'pending'), 0) as pending_amount,
    COUNT(DISTINCT user_id) as unique_users
FROM pending_payouts;

COMMENT ON TABLE user_balances IS 'Tracks user mining reward balances';
COMMENT ON TABLE pending_payouts IS 'Queue of payouts waiting to be processed';
COMMENT ON TABLE payout_transactions IS 'Audit trail of blockchain transactions';
