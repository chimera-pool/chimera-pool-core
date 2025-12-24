-- Statistics and graph optimization indexes
-- Supports time-range queries for dashboard graphs (1H, 6H, 24H, 7D, 30D, 3M, 6M, 1Y, All)

-- Composite index for time-based share aggregation queries
-- Improves: Hashrate history graphs, shares submitted graphs
CREATE INDEX IF NOT EXISTS idx_shares_timestamp_valid ON shares(timestamp DESC, is_valid) 
WHERE is_valid = true;

-- Composite index for difficulty-weighted hashrate calculations
-- Improves: Pool hashrate calculation from shares
CREATE INDEX IF NOT EXISTS idx_shares_timestamp_difficulty ON shares(timestamp DESC, difficulty) 
WHERE is_valid = true;

-- Index for miner-specific share time queries
-- Improves: Per-miner hashrate calculation, vardiff analysis
CREATE INDEX IF NOT EXISTS idx_shares_miner_timestamp_valid ON shares(miner_id, timestamp DESC) 
WHERE is_valid = true;

-- Add pool_stats_history table for time-series statistics caching
CREATE TABLE IF NOT EXISTS pool_stats_history (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    time_bucket TIMESTAMP WITH TIME ZONE NOT NULL,
    bucket_type VARCHAR(20) NOT NULL CHECK (bucket_type IN ('minute', 'hour', 'day', 'week', 'month')),
    total_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    active_miners INTEGER NOT NULL DEFAULT 0,
    total_shares BIGINT NOT NULL DEFAULT 0,
    valid_shares BIGINT NOT NULL DEFAULT 0,
    invalid_shares BIGINT NOT NULL DEFAULT 0,
    blocks_found INTEGER NOT NULL DEFAULT 0,
    UNIQUE(time_bucket, bucket_type)
);

-- Index for fast time-bucket lookups
CREATE INDEX IF NOT EXISTS idx_pool_stats_bucket ON pool_stats_history(bucket_type, time_bucket DESC);

-- Add miner_stats_history table for per-miner time-series data
CREATE TABLE IF NOT EXISTS miner_stats_history (
    id BIGSERIAL PRIMARY KEY,
    miner_id BIGINT NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    time_bucket TIMESTAMP WITH TIME ZONE NOT NULL,
    bucket_type VARCHAR(20) NOT NULL CHECK (bucket_type IN ('minute', 'hour', 'day', 'week', 'month')),
    hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    shares_submitted BIGINT NOT NULL DEFAULT 0,
    shares_accepted BIGINT NOT NULL DEFAULT 0,
    shares_rejected BIGINT NOT NULL DEFAULT 0,
    difficulty DOUBLE PRECISION NOT NULL DEFAULT 0,
    UNIQUE(miner_id, time_bucket, bucket_type)
);

-- Index for miner-specific stats lookups
CREATE INDEX IF NOT EXISTS idx_miner_stats_bucket ON miner_stats_history(miner_id, bucket_type, time_bucket DESC);

COMMENT ON TABLE pool_stats_history IS 'Time-series cache of pool-wide statistics for dashboard graphs';
COMMENT ON TABLE miner_stats_history IS 'Time-series cache of per-miner statistics for graphs';
