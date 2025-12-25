-- Advanced Payout Options Migration
-- Adds support for multiple payout modes: PPLNS, PPS, PPS+, FPPS, SCORE, SOLO, SLICE
-- Requirements: Configurable fees, minimum thresholds, merged mining hooks, V2 Job Declaration support

-- =============================================================================
-- PAYOUT MODE ENUM TYPE
-- =============================================================================

CREATE TYPE payout_mode AS ENUM (
    'pplns',      -- Pay Per Last N Shares
    'pps',        -- Pay Per Share
    'pps_plus',   -- PPS for block reward + PPLNS for tx fees
    'fpps',       -- Full Pay Per Share (includes expected tx fees)
    'score',      -- Time-weighted PPLNS (discourages pool hopping)
    'solo',       -- Solo mining through pool
    'slice'       -- V2 Job Declaration enhanced PPLNS with sliced windows
);

-- =============================================================================
-- USER PAYOUT SETTINGS TABLE
-- =============================================================================

CREATE TABLE user_payout_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payout_mode payout_mode NOT NULL DEFAULT 'pplns',
    min_payout_amount BIGINT NOT NULL DEFAULT 1000000, -- 0.01 LTC in litoshis
    payout_address VARCHAR(255),
    auto_payout_enable BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_user_payout_settings UNIQUE (user_id)
);

CREATE INDEX idx_user_payout_settings_user_id ON user_payout_settings(user_id);
CREATE INDEX idx_user_payout_settings_mode ON user_payout_settings(payout_mode);
CREATE INDEX idx_user_payout_settings_auto ON user_payout_settings(auto_payout_enable);

-- Trigger for updated_at
CREATE TRIGGER update_user_payout_settings_updated_at
    BEFORE UPDATE ON user_payout_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- POOL FEE CONFIGURATION TABLE
-- =============================================================================

CREATE TABLE pool_fee_config (
    id BIGSERIAL PRIMARY KEY,
    payout_mode payout_mode NOT NULL,
    coin_symbol VARCHAR(10) NOT NULL DEFAULT 'LTC',
    fee_percent DECIMAL(5,2) NOT NULL CHECK (fee_percent >= 0 AND fee_percent <= 100),
    min_payout BIGINT NOT NULL DEFAULT 1000000,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_mode_coin UNIQUE (payout_mode, coin_symbol)
);

-- Insert default fee configurations
INSERT INTO pool_fee_config (payout_mode, coin_symbol, fee_percent, min_payout, is_enabled) VALUES
    ('pplns', 'LTC', 1.0, 1000000, true),      -- 1% fee, 0.01 LTC min
    ('pps', 'LTC', 2.0, 1000000, false),       -- 2% fee (disabled by default - high pool risk)
    ('pps_plus', 'LTC', 1.5, 1000000, true),   -- 1.5% fee
    ('fpps', 'LTC', 2.0, 1000000, false),      -- 2% fee (disabled by default - highest pool risk)
    ('score', 'LTC', 1.0, 1000000, true),      -- 1% fee
    ('solo', 'LTC', 0.5, 1000000, true),       -- 0.5% fee
    ('slice', 'LTC', 0.8, 1000000, true),      -- 0.8% fee - V2 enhanced, lower fee
    ('pplns', 'BDAG', 1.0, 1000000000, true),  -- 1% fee, 10 BDAG min
    ('pps', 'BDAG', 2.0, 1000000000, false),
    ('pps_plus', 'BDAG', 1.5, 1000000000, true),
    ('fpps', 'BDAG', 2.0, 1000000000, false),
    ('score', 'BDAG', 1.0, 1000000000, true),
    ('solo', 'BDAG', 0.5, 1000000000, true),
    ('slice', 'BDAG', 0.8, 1000000000, true);  -- V2 enhanced for BlockDAG

CREATE INDEX idx_pool_fee_config_mode ON pool_fee_config(payout_mode);
CREATE INDEX idx_pool_fee_config_coin ON pool_fee_config(coin_symbol);
CREATE INDEX idx_pool_fee_config_enabled ON pool_fee_config(is_enabled);

-- Trigger for updated_at
CREATE TRIGGER update_pool_fee_config_updated_at
    BEFORE UPDATE ON pool_fee_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- PAYOUT CALCULATION HISTORY (for auditing and debugging)
-- =============================================================================

CREATE TABLE payout_calculations (
    id BIGSERIAL PRIMARY KEY,
    block_id BIGINT NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    payout_mode payout_mode NOT NULL,
    total_shares BIGINT NOT NULL,
    total_difficulty DECIMAL(30,8) NOT NULL,
    block_reward BIGINT NOT NULL,
    tx_fees BIGINT NOT NULL DEFAULT 0,
    pool_fee_amount BIGINT NOT NULL,
    net_reward BIGINT NOT NULL,
    window_size BIGINT,
    calculation_params JSONB,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payout_calculations_block_id ON payout_calculations(block_id);
CREATE INDEX idx_payout_calculations_mode ON payout_calculations(payout_mode);
CREATE INDEX idx_payout_calculations_date ON payout_calculations(calculated_at);

-- =============================================================================
-- USER PAYOUT RECORDS (detailed per-block payouts by mode)
-- =============================================================================

ALTER TABLE payouts ADD COLUMN IF NOT EXISTS payout_mode payout_mode DEFAULT 'pplns';
ALTER TABLE payouts ADD COLUMN IF NOT EXISTS block_id BIGINT REFERENCES blocks(id);
ALTER TABLE payouts ADD COLUMN IF NOT EXISTS share_difficulty DECIMAL(30,8);
ALTER TABLE payouts ADD COLUMN IF NOT EXISTS fee_amount BIGINT DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_payouts_mode ON payouts(payout_mode);
CREATE INDEX IF NOT EXISTS idx_payouts_block_id ON payouts(block_id);

-- =============================================================================
-- MERGED MINING PLACEHOLDER TABLES
-- Future-proofing for auxiliary chain support
-- =============================================================================

CREATE TABLE aux_chains (
    id BIGSERIAL PRIMARY KEY,
    chain_id VARCHAR(50) UNIQUE NOT NULL,
    chain_name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    rpc_url VARCHAR(500),
    wallet_address VARCHAR(255),
    fee_percent DECIMAL(5,2) NOT NULL DEFAULT 1.0,
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    block_time_seconds INTEGER DEFAULT 60,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_aux_chains_enabled ON aux_chains(is_enabled);
CREATE INDEX idx_aux_chains_symbol ON aux_chains(symbol);

-- Trigger for updated_at
CREATE TRIGGER update_aux_chains_updated_at
    BEFORE UPDATE ON aux_chains
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Aux chain blocks found
CREATE TABLE aux_blocks (
    id BIGSERIAL PRIMARY KEY,
    aux_chain_id BIGINT NOT NULL REFERENCES aux_chains(id) ON DELETE CASCADE,
    parent_block_id BIGINT REFERENCES blocks(id), -- Primary chain block that found this
    height BIGINT NOT NULL,
    hash VARCHAR(128) NOT NULL,
    reward BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'orphaned')),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_aux_blocks_chain ON aux_blocks(aux_chain_id);
CREATE INDEX idx_aux_blocks_parent ON aux_blocks(parent_block_id);
CREATE INDEX idx_aux_blocks_status ON aux_blocks(status);

-- Aux chain payouts
CREATE TABLE aux_payouts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    aux_chain_id BIGINT NOT NULL REFERENCES aux_chains(id) ON DELETE CASCADE,
    aux_block_id BIGINT NOT NULL REFERENCES aux_blocks(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    payout_mode payout_mode NOT NULL DEFAULT 'pplns',
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'confirmed', 'failed')),
    tx_hash VARCHAR(128),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_aux_payouts_user ON aux_payouts(user_id);
CREATE INDEX idx_aux_payouts_chain ON aux_payouts(aux_chain_id);
CREATE INDEX idx_aux_payouts_status ON aux_payouts(status);

-- =============================================================================
-- PPLNS WINDOW CONFIGURATION
-- =============================================================================

CREATE TABLE pplns_config (
    id BIGSERIAL PRIMARY KEY,
    coin_symbol VARCHAR(10) NOT NULL,
    window_size BIGINT NOT NULL DEFAULT 200000, -- Total difficulty window
    score_decay_factor DECIMAL(5,4) DEFAULT 0.5, -- For SCORE mode: 50% per hour
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_pplns_config_coin UNIQUE (coin_symbol)
);

INSERT INTO pplns_config (coin_symbol, window_size, score_decay_factor) VALUES
    ('LTC', 200000, 0.5),
    ('BDAG', 100000, 0.6);

CREATE TRIGGER update_pplns_config_updated_at
    BEFORE UPDATE ON pplns_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- VIEWS FOR REPORTING
-- =============================================================================

-- View: User payout summary by mode
CREATE OR REPLACE VIEW v_user_payout_summary AS
SELECT 
    u.id AS user_id,
    u.username,
    COALESCE(ups.payout_mode, 'pplns') AS payout_mode,
    COALESCE(ups.min_payout_amount, 1000000) AS min_payout_amount,
    COALESCE(ups.auto_payout_enable, true) AS auto_payout_enable,
    COALESCE(SUM(p.amount), 0) AS total_earned,
    COUNT(p.id) AS payout_count,
    MAX(p.created_at) AS last_payout
FROM users u
LEFT JOIN user_payout_settings ups ON u.id = ups.user_id
LEFT JOIN payouts p ON u.id = p.user_id AND p.status = 'confirmed'
GROUP BY u.id, u.username, ups.payout_mode, ups.min_payout_amount, ups.auto_payout_enable;

-- View: Pool fee summary by mode
CREATE OR REPLACE VIEW v_pool_fee_summary AS
SELECT 
    payout_mode,
    coin_symbol,
    fee_percent,
    min_payout,
    is_enabled,
    CASE 
        WHEN payout_mode = 'pplns' THEN 'Miners share block variance'
        WHEN payout_mode = 'pps' THEN 'Pool absorbs all variance'
        WHEN payout_mode = 'pps_plus' THEN 'PPS + PPLNS for tx fees'
        WHEN payout_mode = 'fpps' THEN 'Pool pays expected tx fees'
        WHEN payout_mode = 'score' THEN 'Time-weighted shares'
        WHEN payout_mode = 'solo' THEN 'Full block to finder'
        WHEN payout_mode = 'slice' THEN 'V2 Job Declaration enhanced PPLNS'
    END AS description
FROM pool_fee_config
ORDER BY coin_symbol, payout_mode;
