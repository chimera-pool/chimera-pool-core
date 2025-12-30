-- Migration 017: Add network_id to core tables for multi-coin support
-- This enables tracking shares, blocks, payouts, and miners per cryptocurrency network

-- =============================================================================
-- ADD NETWORK_ID TO SHARES TABLE
-- =============================================================================
ALTER TABLE shares ADD COLUMN IF NOT EXISTS network_id UUID REFERENCES network_configs(id);
CREATE INDEX IF NOT EXISTS idx_shares_network ON shares(network_id);
CREATE INDEX IF NOT EXISTS idx_shares_network_timestamp ON shares(network_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_shares_network_user ON shares(network_id, user_id);

-- =============================================================================
-- ADD NETWORK_ID TO BLOCKS TABLE
-- =============================================================================
ALTER TABLE blocks ADD COLUMN IF NOT EXISTS network_id UUID REFERENCES network_configs(id);
CREATE INDEX IF NOT EXISTS idx_blocks_network ON blocks(network_id);
CREATE INDEX IF NOT EXISTS idx_blocks_network_height ON blocks(network_id, height DESC);
CREATE INDEX IF NOT EXISTS idx_blocks_network_status ON blocks(network_id, status);

-- =============================================================================
-- ADD NETWORK_ID TO PAYOUTS TABLE
-- =============================================================================
ALTER TABLE payouts ADD COLUMN IF NOT EXISTS network_id UUID REFERENCES network_configs(id);
CREATE INDEX IF NOT EXISTS idx_payouts_network ON payouts(network_id);
CREATE INDEX IF NOT EXISTS idx_payouts_network_user ON payouts(network_id, user_id);
CREATE INDEX IF NOT EXISTS idx_payouts_network_status ON payouts(network_id, status);

-- =============================================================================
-- ADD NETWORK_ID TO MINERS TABLE
-- =============================================================================
ALTER TABLE miners ADD COLUMN IF NOT EXISTS network_id UUID REFERENCES network_configs(id);
CREATE INDEX IF NOT EXISTS idx_miners_network ON miners(network_id);
CREATE INDEX IF NOT EXISTS idx_miners_network_active ON miners(network_id, is_active);
CREATE INDEX IF NOT EXISTS idx_miners_network_user ON miners(network_id, user_id);

-- =============================================================================
-- BACKFILL EXISTING DATA WITH CURRENT ACTIVE NETWORK
-- =============================================================================
-- Get the current active network (Litecoin) and assign to all existing records
DO $$
DECLARE
    active_network_id UUID;
BEGIN
    -- Get the active network ID
    SELECT id INTO active_network_id FROM network_configs WHERE is_active = true AND is_default = true LIMIT 1;
    
    -- If no active network found, try to get any active network
    IF active_network_id IS NULL THEN
        SELECT id INTO active_network_id FROM network_configs WHERE is_active = true LIMIT 1;
    END IF;
    
    -- If still no network found, get the first network (fallback)
    IF active_network_id IS NULL THEN
        SELECT id INTO active_network_id FROM network_configs LIMIT 1;
    END IF;
    
    -- Update existing records that have NULL network_id
    IF active_network_id IS NOT NULL THEN
        UPDATE shares SET network_id = active_network_id WHERE network_id IS NULL;
        UPDATE blocks SET network_id = active_network_id WHERE network_id IS NULL;
        UPDATE payouts SET network_id = active_network_id WHERE network_id IS NULL;
        UPDATE miners SET network_id = active_network_id WHERE network_id IS NULL;
        
        RAISE NOTICE 'Backfilled existing records with network_id: %', active_network_id;
    ELSE
        RAISE WARNING 'No network_configs found - existing records not backfilled';
    END IF;
END $$;

-- =============================================================================
-- CREATE HELPER FUNCTIONS FOR MULTI-COIN OPERATIONS
-- =============================================================================

-- Function to get current active network ID
CREATE OR REPLACE FUNCTION get_active_network_id()
RETURNS UUID AS $$
DECLARE
    active_id UUID;
BEGIN
    SELECT id INTO active_id FROM network_configs 
    WHERE is_active = true AND is_default = true 
    LIMIT 1;
    
    IF active_id IS NULL THEN
        SELECT id INTO active_id FROM network_configs WHERE is_active = true LIMIT 1;
    END IF;
    
    RETURN active_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get pool stats for a specific network
CREATE OR REPLACE FUNCTION get_network_pool_stats(p_network_id UUID)
RETURNS TABLE (
    total_hashrate DECIMAL,
    active_miners BIGINT,
    total_shares_24h BIGINT,
    valid_shares_24h BIGINT,
    blocks_found_24h INTEGER,
    blocks_found_total INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(SUM(m.hashrate), 0)::DECIMAL as total_hashrate,
        COUNT(DISTINCT m.id) FILTER (WHERE m.is_active = true)::BIGINT as active_miners,
        COUNT(s.id) FILTER (WHERE s.timestamp > NOW() - INTERVAL '24 hours')::BIGINT as total_shares_24h,
        COUNT(s.id) FILTER (WHERE s.timestamp > NOW() - INTERVAL '24 hours' AND s.is_valid = true)::BIGINT as valid_shares_24h,
        COUNT(b.id) FILTER (WHERE b.timestamp > NOW() - INTERVAL '24 hours')::INTEGER as blocks_found_24h,
        COUNT(b.id)::INTEGER as blocks_found_total
    FROM miners m
    LEFT JOIN shares s ON s.miner_id = m.id AND s.network_id = p_network_id
    LEFT JOIN blocks b ON b.network_id = p_network_id
    WHERE m.network_id = p_network_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get user stats for a specific network
CREATE OR REPLACE FUNCTION get_user_network_mining_stats(p_user_id BIGINT, p_network_id UUID)
RETURNS TABLE (
    total_shares BIGINT,
    valid_shares BIGINT,
    total_hashrate DECIMAL,
    blocks_found INTEGER,
    last_share_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(s.id)::BIGINT as total_shares,
        COUNT(s.id) FILTER (WHERE s.is_valid = true)::BIGINT as valid_shares,
        COALESCE(SUM(m.hashrate), 0)::DECIMAL as total_hashrate,
        (SELECT COUNT(*)::INTEGER FROM blocks WHERE finder_id = p_user_id AND network_id = p_network_id) as blocks_found,
        MAX(s.timestamp) as last_share_at
    FROM shares s
    JOIN miners m ON m.id = s.miner_id
    WHERE s.user_id = p_user_id AND s.network_id = p_network_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- COMMENTS
-- =============================================================================
COMMENT ON COLUMN shares.network_id IS 'Reference to the network/coin this share was submitted for';
COMMENT ON COLUMN blocks.network_id IS 'Reference to the network/coin this block was found on';
COMMENT ON COLUMN payouts.network_id IS 'Reference to the network/coin this payout is for';
COMMENT ON COLUMN miners.network_id IS 'Reference to the network/coin this miner is currently mining on';
COMMENT ON FUNCTION get_active_network_id() IS 'Returns the UUID of the currently active mining network';
COMMENT ON FUNCTION get_network_pool_stats(UUID) IS 'Returns aggregated pool statistics for a specific network';
COMMENT ON FUNCTION get_user_network_mining_stats(BIGINT, UUID) IS 'Returns user mining statistics for a specific network';
