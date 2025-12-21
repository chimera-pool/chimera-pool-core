-- Equipment Management and Multi-Wallet Payout System
-- Migration 006

-- Equipment table for tracking all mining hardware
CREATE TABLE IF NOT EXISTS equipment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'gpu',
    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    worker_name VARCHAR(100),
    ip_address VARCHAR(45),
    
    -- Hardware specifications
    model VARCHAR(100),
    manufacturer VARCHAR(100),
    firmware_version VARCHAR(50),
    
    -- Performance metrics (updated in real-time)
    current_hashrate BIGINT DEFAULT 0,
    average_hashrate BIGINT DEFAULT 0,
    max_hashrate BIGINT DEFAULT 0,
    efficiency DECIMAL(10, 4) DEFAULT 0,
    power_usage DECIMAL(10, 2) DEFAULT 0,
    temperature DECIMAL(5, 2) DEFAULT 0,
    fan_speed INTEGER DEFAULT 0,
    
    -- Network metrics
    latency DECIMAL(10, 2) DEFAULT 0,
    connection_type VARCHAR(20) DEFAULT 'stratum_v1',
    last_seen TIMESTAMP WITH TIME ZONE,
    uptime BIGINT DEFAULT 0,
    
    -- Mining statistics
    shares_accepted BIGINT DEFAULT 0,
    shares_rejected BIGINT DEFAULT 0,
    shares_stale BIGINT DEFAULT 0,
    blocks_found INTEGER DEFAULT 0,
    total_earnings DECIMAL(20, 8) DEFAULT 0,
    
    -- Error tracking
    last_error TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE,
    error_count INTEGER DEFAULT 0,
    
    -- Timestamps
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_type CHECK (type IN ('asic', 'gpu', 'cpu', 'fpga', 'blockdag_x30', 'blockdag_x100')),
    CONSTRAINT valid_status CHECK (status IN ('online', 'offline', 'mining', 'idle', 'error', 'maintenance')),
    CONSTRAINT valid_connection CHECK (connection_type IN ('stratum_v1', 'stratum_v2'))
);

-- Index for fast lookups
CREATE INDEX idx_equipment_user_id ON equipment(user_id);
CREATE INDEX idx_equipment_status ON equipment(status);
CREATE INDEX idx_equipment_type ON equipment(type);
CREATE INDEX idx_equipment_worker_name ON equipment(user_id, worker_name);
CREATE INDEX idx_equipment_last_seen ON equipment(last_seen);

-- Equipment metrics history for time-series data
CREATE TABLE IF NOT EXISTS equipment_metrics_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipment_id UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    hashrate BIGINT DEFAULT 0,
    temperature DECIMAL(5, 2) DEFAULT 0,
    power_usage DECIMAL(10, 2) DEFAULT 0,
    latency DECIMAL(10, 2) DEFAULT 0,
    shares_accepted BIGINT DEFAULT 0,
    shares_rejected BIGINT DEFAULT 0
);

-- Index for time-series queries
CREATE INDEX idx_metrics_equipment_time ON equipment_metrics_history(equipment_id, timestamp DESC);

-- Payout splits for multi-wallet distribution
CREATE TABLE IF NOT EXISTS payout_splits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipment_id UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    wallet_address VARCHAR(100) NOT NULL,
    percentage DECIMAL(5, 2) NOT NULL,
    label VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT valid_percentage CHECK (percentage > 0 AND percentage <= 100)
);

-- Index for payout lookups
CREATE INDEX idx_payout_splits_equipment ON payout_splits(equipment_id);
CREATE INDEX idx_payout_splits_wallet ON payout_splits(wallet_address);

-- User wallets table for managing multiple wallet addresses
CREATE TABLE IF NOT EXISTS user_wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_address VARCHAR(100) NOT NULL,
    label VARCHAR(100),
    is_primary BOOLEAN DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    currency VARCHAR(10) DEFAULT 'BDAG',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT unique_user_wallet UNIQUE (user_id, wallet_address)
);

-- Index for wallet lookups
CREATE INDEX idx_user_wallets_user ON user_wallets(user_id);
CREATE INDEX idx_user_wallets_primary ON user_wallets(user_id, is_primary) WHERE is_primary = TRUE;

-- Pool-wide statistics cache (for public display)
CREATE TABLE IF NOT EXISTS pool_stats_cache (
    id INTEGER PRIMARY KEY DEFAULT 1,
    total_equipment INTEGER DEFAULT 0,
    online_equipment INTEGER DEFAULT 0,
    total_hashrate BIGINT DEFAULT 0,
    total_miners INTEGER DEFAULT 0,
    blocks_found_24h INTEGER DEFAULT 0,
    average_latency DECIMAL(10, 2) DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT single_row CHECK (id = 1)
);

-- Insert initial pool stats row
INSERT INTO pool_stats_cache (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- Function to update pool stats cache
CREATE OR REPLACE FUNCTION update_pool_stats_cache()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE pool_stats_cache SET
        total_equipment = (SELECT COUNT(*) FROM equipment),
        online_equipment = (SELECT COUNT(*) FROM equipment WHERE status IN ('online', 'mining')),
        total_hashrate = (SELECT COALESCE(SUM(current_hashrate), 0) FROM equipment WHERE status = 'mining'),
        total_miners = (SELECT COUNT(DISTINCT user_id) FROM equipment WHERE status IN ('online', 'mining')),
        average_latency = (SELECT COALESCE(AVG(latency), 0) FROM equipment WHERE status IN ('online', 'mining')),
        updated_at = NOW()
    WHERE id = 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update pool stats on equipment changes
CREATE TRIGGER trigger_update_pool_stats
AFTER INSERT OR UPDATE OR DELETE ON equipment
FOR EACH STATEMENT
EXECUTE FUNCTION update_pool_stats_cache();

-- Equipment alerts table for notifications
CREATE TABLE IF NOT EXISTS equipment_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    equipment_id UUID NOT NULL REFERENCES equipment(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT valid_alert_type CHECK (alert_type IN ('offline', 'error', 'performance_drop', 'high_temp', 'share_rejection'))
);

-- Index for alert queries
CREATE INDEX idx_equipment_alerts_user ON equipment_alerts(user_id, is_read);
CREATE INDEX idx_equipment_alerts_equipment ON equipment_alerts(equipment_id);
