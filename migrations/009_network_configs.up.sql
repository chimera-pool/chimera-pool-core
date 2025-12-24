-- Migration: Add network configurations table for multi-coin support
-- This enables hot-swapping between different blockchain networks

CREATE TABLE IF NOT EXISTS network_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Network identification
    name VARCHAR(100) NOT NULL UNIQUE,
    symbol VARCHAR(20) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    
    -- Network status
    is_active BOOLEAN DEFAULT false,
    is_default BOOLEAN DEFAULT false,
    
    -- Algorithm configuration
    algorithm VARCHAR(50) NOT NULL,  -- scrypt, sha256, ethash, scrpy-variant, etc.
    algorithm_variant VARCHAR(50),    -- e.g., scrypt-n, sha256d
    algorithm_params JSONB DEFAULT '{}',  -- {"N": 1024, "r": 1, "p": 1}
    
    -- Network connection
    rpc_url VARCHAR(500) NOT NULL,
    rpc_user VARCHAR(100),
    rpc_password VARCHAR(255),
    rpc_timeout_ms INTEGER DEFAULT 30000,
    
    -- Secondary/fallback RPC
    rpc_url_fallback VARCHAR(500),
    
    -- Block explorer
    explorer_url VARCHAR(500),
    explorer_tx_path VARCHAR(100) DEFAULT '/tx/',
    explorer_block_path VARCHAR(100) DEFAULT '/block/',
    explorer_address_path VARCHAR(100) DEFAULT '/address/',
    
    -- Mining parameters
    stratum_port INTEGER NOT NULL DEFAULT 3333,
    vardiff_enabled BOOLEAN DEFAULT true,
    vardiff_min FLOAT DEFAULT 0.001,
    vardiff_max FLOAT DEFAULT 65536,
    vardiff_target_time INTEGER DEFAULT 15,  -- seconds
    
    -- Block parameters
    block_time_target INTEGER DEFAULT 150,  -- seconds (2.5 min for LTC)
    block_reward DECIMAL(20, 8) DEFAULT 0,
    min_confirmations INTEGER DEFAULT 100,
    
    -- Pool parameters
    pool_wallet_address VARCHAR(255) NOT NULL,
    pool_fee_percent DECIMAL(5, 2) DEFAULT 1.0,
    min_payout_threshold DECIMAL(20, 8) DEFAULT 0.01,
    payout_interval_seconds INTEGER DEFAULT 3600,
    
    -- Chain info
    chain_id INTEGER,
    network_type VARCHAR(20) DEFAULT 'mainnet',  -- mainnet, testnet, devnet
    address_prefix VARCHAR(10),
    
    -- Metadata
    logo_url VARCHAR(500),
    website_url VARCHAR(500),
    description TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    activated_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT unique_active_default CHECK (
        NOT (is_default = true AND is_active = false)
    )
);

-- Index for quick lookups
CREATE INDEX idx_network_configs_active ON network_configs(is_active);
CREATE INDEX idx_network_configs_symbol ON network_configs(symbol);
CREATE INDEX idx_network_configs_algorithm ON network_configs(algorithm);

-- Ensure only one default network
CREATE UNIQUE INDEX idx_network_configs_single_default 
ON network_configs(is_default) WHERE is_default = true;

-- Network switch history for audit trail
CREATE TABLE IF NOT EXISTS network_switch_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_network_id UUID REFERENCES network_configs(id),
    to_network_id UUID REFERENCES network_configs(id) NOT NULL,
    switched_by INTEGER REFERENCES users(id),
    switch_reason TEXT,
    switch_type VARCHAR(50) DEFAULT 'manual',  -- manual, scheduled, failover
    status VARCHAR(50) DEFAULT 'completed',     -- pending, in_progress, completed, failed, rolled_back
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT
);

-- Insert default BlockDAG configuration
INSERT INTO network_configs (
    name, symbol, display_name,
    is_active, is_default,
    algorithm, algorithm_variant, algorithm_params,
    rpc_url, explorer_url,
    stratum_port, block_time_target,
    pool_wallet_address, pool_fee_percent, min_payout_threshold,
    chain_id, network_type,
    description
) VALUES (
    'blockdag', 'BDAG', 'BlockDAG',
    true, true,
    'scrpy-variant', 'blockdag-v1', '{"N": 1024, "r": 1, "p": 1, "keyLen": 32}',
    'https://rpc.awakening.bdagscan.com', 'https://awakening.bdagscan.com',
    3333, 10,
    '0xD393798C098FFe3d64d4Ca531158D3562D00b66e', 1.0, 1.0,
    1043, 'mainnet',
    'BlockDAG custom Scrpy-variant algorithm mining network'
) ON CONFLICT (name) DO NOTHING;

-- Insert Litecoin configuration (for testing)
INSERT INTO network_configs (
    name, symbol, display_name,
    is_active, is_default,
    algorithm, algorithm_variant, algorithm_params,
    rpc_url, explorer_url,
    stratum_port, block_time_target, block_reward,
    pool_wallet_address, pool_fee_percent, min_payout_threshold,
    network_type, address_prefix,
    description
) VALUES (
    'litecoin', 'LTC', 'Litecoin',
    false, false,
    'scrypt', 'scrypt-n', '{"N": 1024, "r": 1, "p": 1, "keyLen": 32}',
    'http://localhost:9332', 'https://blockchair.com/litecoin',
    3334, 150, 6.25,
    '', 1.0, 0.01,
    'mainnet', 'L',
    'Litecoin - Scrypt algorithm cryptocurrency'
) ON CONFLICT (name) DO NOTHING;

-- Insert Bitcoin configuration (template)
INSERT INTO network_configs (
    name, symbol, display_name,
    is_active, is_default,
    algorithm, algorithm_variant,
    rpc_url, explorer_url,
    stratum_port, block_time_target, block_reward,
    pool_wallet_address, pool_fee_percent, min_payout_threshold,
    network_type, address_prefix,
    description
) VALUES (
    'bitcoin', 'BTC', 'Bitcoin',
    false, false,
    'sha256', 'sha256d',
    'http://localhost:8332', 'https://blockchair.com/bitcoin',
    3335, 600, 3.125,
    '', 1.0, 0.001,
    'mainnet', '1',
    'Bitcoin - SHA-256 algorithm cryptocurrency'
) ON CONFLICT (name) DO NOTHING;

-- Function to safely switch active network
CREATE OR REPLACE FUNCTION switch_active_network(
    p_new_network_name VARCHAR,
    p_switched_by INTEGER,
    p_reason TEXT DEFAULT 'Manual switch'
)
RETURNS UUID AS $$
DECLARE
    v_old_network_id UUID;
    v_new_network_id UUID;
    v_history_id UUID;
BEGIN
    -- Get current active network
    SELECT id INTO v_old_network_id FROM network_configs WHERE is_active = true AND is_default = true;
    
    -- Get new network ID
    SELECT id INTO v_new_network_id FROM network_configs WHERE name = p_new_network_name;
    
    IF v_new_network_id IS NULL THEN
        RAISE EXCEPTION 'Network not found: %', p_new_network_name;
    END IF;
    
    -- Create history record
    INSERT INTO network_switch_history (from_network_id, to_network_id, switched_by, switch_reason, status)
    VALUES (v_old_network_id, v_new_network_id, p_switched_by, p_reason, 'in_progress')
    RETURNING id INTO v_history_id;
    
    -- Deactivate old default
    UPDATE network_configs SET is_default = false WHERE is_default = true;
    
    -- Activate new network as default
    UPDATE network_configs 
    SET is_active = true, is_default = true, activated_at = NOW(), updated_at = NOW()
    WHERE id = v_new_network_id;
    
    -- Complete history record
    UPDATE network_switch_history 
    SET status = 'completed', completed_at = NOW()
    WHERE id = v_history_id;
    
    RETURN v_history_id;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update timestamps
CREATE OR REPLACE FUNCTION update_network_config_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_network_configs_updated
BEFORE UPDATE ON network_configs
FOR EACH ROW EXECUTE FUNCTION update_network_config_timestamp();

COMMENT ON TABLE network_configs IS 'Multi-coin network configurations for hot-swapping mining pools';
COMMENT ON TABLE network_switch_history IS 'Audit trail of network switches for debugging and compliance';
