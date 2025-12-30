-- Migration: Multi-coin user statistics
-- Tracks user mining metrics per network for universal platform dashboard

-- User network stats - per-user, per-network mining metrics
CREATE TABLE IF NOT EXISTS user_network_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    network_id UUID NOT NULL REFERENCES network_configs(id) ON DELETE CASCADE,
    
    -- Mining metrics
    total_hashrate DECIMAL(30, 8) DEFAULT 0,
    average_hashrate DECIMAL(30, 8) DEFAULT 0,
    peak_hashrate DECIMAL(30, 8) DEFAULT 0,
    
    -- Share statistics
    total_shares BIGINT DEFAULT 0,
    valid_shares BIGINT DEFAULT 0,
    invalid_shares BIGINT DEFAULT 0,
    stale_shares BIGINT DEFAULT 0,
    share_efficiency DECIMAL(5, 2) DEFAULT 100.00,
    
    -- Blocks found
    blocks_found INTEGER DEFAULT 0,
    last_block_found_at TIMESTAMP WITH TIME ZONE,
    
    -- Earnings
    total_earned DECIMAL(30, 8) DEFAULT 0,
    pending_balance DECIMAL(30, 8) DEFAULT 0,
    total_paid_out DECIMAL(30, 8) DEFAULT 0,
    
    -- Active workers on this network
    active_workers INTEGER DEFAULT 0,
    total_workers INTEGER DEFAULT 0,
    
    -- Time tracking
    first_connected_at TIMESTAMP WITH TIME ZONE,
    last_active_at TIMESTAMP WITH TIME ZONE,
    total_mining_time_seconds BIGINT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Unique constraint per user per network
    CONSTRAINT unique_user_network UNIQUE (user_id, network_id)
);

-- Indexes for fast lookups
CREATE INDEX idx_user_network_stats_user ON user_network_stats(user_id);
CREATE INDEX idx_user_network_stats_network ON user_network_stats(network_id);
CREATE INDEX idx_user_network_stats_hashrate ON user_network_stats(total_hashrate DESC);
CREATE INDEX idx_user_network_stats_active ON user_network_stats(last_active_at DESC);

-- Miner network assignments - tracks which network each miner is on
CREATE TABLE IF NOT EXISTS miner_network_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    miner_id BIGINT NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
    network_id UUID NOT NULL REFERENCES network_configs(id) ON DELETE CASCADE,
    
    -- Current status
    is_active BOOLEAN DEFAULT true,
    current_hashrate DECIMAL(30, 8) DEFAULT 0,
    current_difficulty DECIMAL(20, 8) DEFAULT 1,
    
    -- Session tracking
    connected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_share_at TIMESTAMP WITH TIME ZONE,
    shares_this_session BIGINT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Unique constraint per miner per network
    CONSTRAINT unique_miner_network UNIQUE (miner_id, network_id)
);

CREATE INDEX idx_miner_network_miner ON miner_network_assignments(miner_id);
CREATE INDEX idx_miner_network_network ON miner_network_assignments(network_id);
CREATE INDEX idx_miner_network_active ON miner_network_assignments(is_active, network_id);

-- Network pool stats - aggregated stats per network
CREATE TABLE IF NOT EXISTS network_pool_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID NOT NULL REFERENCES network_configs(id) ON DELETE CASCADE,
    
    -- Pool-wide metrics
    total_hashrate DECIMAL(30, 8) DEFAULT 0,
    active_miners INTEGER DEFAULT 0,
    active_workers INTEGER DEFAULT 0,
    
    -- Share statistics
    shares_per_second DECIMAL(20, 4) DEFAULT 0,
    total_shares_24h BIGINT DEFAULT 0,
    
    -- Blocks
    blocks_found_24h INTEGER DEFAULT 0,
    blocks_found_total INTEGER DEFAULT 0,
    last_block_at TIMESTAMP WITH TIME ZONE,
    
    -- Network info (from RPC)
    network_difficulty DECIMAL(30, 8) DEFAULT 0,
    network_hashrate DECIMAL(30, 8) DEFAULT 0,
    current_block_height BIGINT DEFAULT 0,
    
    -- Pool percentage of network
    pool_percentage DECIMAL(10, 6) DEFAULT 0,
    
    -- Connection status
    rpc_connected BOOLEAN DEFAULT false,
    rpc_last_check TIMESTAMP WITH TIME ZONE,
    rpc_latency_ms INTEGER DEFAULT 0,
    
    -- Timestamps
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Unique constraint for latest stats per network
    CONSTRAINT unique_network_stats UNIQUE (network_id)
);

CREATE INDEX idx_network_pool_stats_network ON network_pool_stats(network_id);

-- Historical network stats for charts
CREATE TABLE IF NOT EXISTS network_stats_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID NOT NULL REFERENCES network_configs(id) ON DELETE CASCADE,
    
    -- Metrics at this point in time
    total_hashrate DECIMAL(30, 8) DEFAULT 0,
    active_miners INTEGER DEFAULT 0,
    active_workers INTEGER DEFAULT 0,
    network_difficulty DECIMAL(30, 8) DEFAULT 0,
    blocks_found INTEGER DEFAULT 0,
    
    -- Time bucket (hourly aggregation)
    bucket_time TIMESTAMP WITH TIME ZONE NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_network_stats_history_network ON network_stats_history(network_id);
CREATE INDEX idx_network_stats_history_time ON network_stats_history(bucket_time DESC);
CREATE INDEX idx_network_stats_history_lookup ON network_stats_history(network_id, bucket_time DESC);

-- Function to update user network stats
CREATE OR REPLACE FUNCTION update_user_network_stats(
    p_user_id BIGINT,
    p_network_id UUID,
    p_hashrate DECIMAL DEFAULT NULL,
    p_valid_shares BIGINT DEFAULT 0,
    p_invalid_shares BIGINT DEFAULT 0
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO user_network_stats (user_id, network_id, total_hashrate, valid_shares, invalid_shares, last_active_at, first_connected_at)
    VALUES (p_user_id, p_network_id, COALESCE(p_hashrate, 0), p_valid_shares, p_invalid_shares, NOW(), NOW())
    ON CONFLICT (user_id, network_id) DO UPDATE SET
        total_hashrate = COALESCE(p_hashrate, user_network_stats.total_hashrate),
        valid_shares = user_network_stats.valid_shares + p_valid_shares,
        invalid_shares = user_network_stats.invalid_shares + p_invalid_shares,
        total_shares = user_network_stats.total_shares + p_valid_shares + p_invalid_shares,
        share_efficiency = CASE 
            WHEN (user_network_stats.valid_shares + p_valid_shares + user_network_stats.invalid_shares + p_invalid_shares) > 0 
            THEN ((user_network_stats.valid_shares + p_valid_shares)::DECIMAL / 
                  (user_network_stats.valid_shares + p_valid_shares + user_network_stats.invalid_shares + p_invalid_shares)::DECIMAL) * 100
            ELSE 100
        END,
        peak_hashrate = GREATEST(user_network_stats.peak_hashrate, COALESCE(p_hashrate, 0)),
        last_active_at = NOW(),
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to get user stats across all networks
CREATE OR REPLACE FUNCTION get_user_all_network_stats(p_user_id BIGINT)
RETURNS TABLE (
    network_name VARCHAR,
    network_symbol VARCHAR,
    network_display_name VARCHAR,
    is_network_active BOOLEAN,
    user_hashrate DECIMAL,
    user_shares BIGINT,
    user_blocks INTEGER,
    user_earned DECIMAL,
    user_pending DECIMAL,
    user_workers INTEGER,
    last_active TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        nc.name,
        nc.symbol,
        nc.display_name,
        nc.is_active,
        COALESCE(uns.total_hashrate, 0),
        COALESCE(uns.total_shares, 0),
        COALESCE(uns.blocks_found, 0),
        COALESCE(uns.total_earned, 0),
        COALESCE(uns.pending_balance, 0),
        COALESCE(uns.active_workers, 0),
        uns.last_active_at
    FROM network_configs nc
    LEFT JOIN user_network_stats uns ON nc.id = uns.network_id AND uns.user_id = p_user_id
    ORDER BY nc.is_active DESC, uns.last_active_at DESC NULLS LAST;
END;
$$ LANGUAGE plpgsql;

-- Function to get aggregated stats across all networks for a user
CREATE OR REPLACE FUNCTION get_user_aggregated_stats(p_user_id BIGINT)
RETURNS TABLE (
    total_networks_mined INTEGER,
    active_networks INTEGER,
    combined_hashrate DECIMAL,
    total_shares_all BIGINT,
    total_blocks_all INTEGER,
    total_earned_all DECIMAL,
    total_pending_all DECIMAL,
    total_workers_all INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(DISTINCT uns.network_id)::INTEGER,
        COUNT(DISTINCT CASE WHEN nc.is_active THEN uns.network_id END)::INTEGER,
        COALESCE(SUM(uns.total_hashrate), 0),
        COALESCE(SUM(uns.total_shares), 0),
        COALESCE(SUM(uns.blocks_found), 0),
        COALESCE(SUM(uns.total_earned), 0),
        COALESCE(SUM(uns.pending_balance), 0),
        COALESCE(SUM(uns.active_workers), 0)
    FROM user_network_stats uns
    JOIN network_configs nc ON nc.id = uns.network_id
    WHERE uns.user_id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Trigger for updated_at
CREATE TRIGGER trg_user_network_stats_updated
BEFORE UPDATE ON user_network_stats
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_miner_network_assignments_updated
BEFORE UPDATE ON miner_network_assignments
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE user_network_stats IS 'Per-user, per-network mining statistics for multi-coin dashboard';
COMMENT ON TABLE miner_network_assignments IS 'Tracks which network each miner/worker is currently mining on';
COMMENT ON TABLE network_pool_stats IS 'Aggregated pool-wide statistics per network';
COMMENT ON TABLE network_stats_history IS 'Historical stats for charting network performance over time';
