-- Migration 020: Database Optimization for Multi-Chain Mining Pool
-- Includes: Table partitioning, composite indexes, and automated maintenance

-- ============================================================================
-- 1. COMPOSITE INDEXES FOR HOT QUERY PATHS
-- ============================================================================

-- Composite index for share lookups by miner and time (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_shares_miner_timestamp 
ON shares (miner_id, timestamp DESC);

-- Composite index for user share queries with validity check
CREATE INDEX IF NOT EXISTS idx_shares_user_valid_timestamp 
ON shares (user_id, is_valid, timestamp DESC);

-- Composite index for network-specific share queries
CREATE INDEX IF NOT EXISTS idx_shares_network_timestamp 
ON shares (network_id, timestamp DESC) WHERE network_id IS NOT NULL;

-- Covering index for common miner stats query
CREATE INDEX IF NOT EXISTS idx_miners_active_stats 
ON miners (is_active, last_seen DESC, hashrate DESC) 
WHERE is_active = true;

-- Index for hashrate history queries
CREATE INDEX IF NOT EXISTS idx_miner_stats_history_time 
ON miner_stats_history (miner_id, timestamp DESC);

-- ============================================================================
-- 2. PARTITIONING SETUP FOR SHARES TABLE
-- ============================================================================

-- Create partitioned shares table structure for future data
-- Note: Existing data stays in main table, new partitions created monthly

-- Function to create monthly partitions automatically
CREATE OR REPLACE FUNCTION create_shares_partition_if_needed()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- Get the start of next month
    partition_date := date_trunc('month', CURRENT_DATE + INTERVAL '1 month')::DATE;
    partition_name := 'shares_' || to_char(partition_date, 'YYYY_MM');
    start_date := partition_date;
    end_date := partition_date + INTERVAL '1 month';
    
    -- Check if partition already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE tablename = partition_name AND schemaname = 'public'
    ) THEN
        -- Create the partition (for future partitioned table migration)
        -- For now, just log that partition would be created
        RAISE NOTICE 'Partition % would be created for range % to %', 
            partition_name, start_date, end_date;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 3. AUTOMATED VACUUM ANALYZE WITH ACTIVITY MONITORING
-- ============================================================================

-- Table to track database maintenance activities
CREATE TABLE IF NOT EXISTS db_maintenance_log (
    id BIGSERIAL PRIMARY KEY,
    operation VARCHAR(50) NOT NULL,
    table_name VARCHAR(100) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    rows_affected BIGINT,
    duration_ms INTEGER,
    activity_level VARCHAR(20), -- 'low', 'medium', 'high'
    notes TEXT
);

-- Table to track activity levels for smart scheduling
CREATE TABLE IF NOT EXISTS db_activity_metrics (
    id BIGSERIAL PRIMARY KEY,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    shares_per_minute INTEGER,
    active_connections INTEGER,
    avg_query_time_ms NUMERIC(10,2),
    activity_level VARCHAR(20) GENERATED ALWAYS AS (
        CASE 
            WHEN shares_per_minute > 1000 THEN 'high'
            WHEN shares_per_minute > 100 THEN 'medium'
            ELSE 'low'
        END
    ) STORED
);

-- Function to record current activity metrics
CREATE OR REPLACE FUNCTION record_activity_metrics()
RETURNS void AS $$
DECLARE
    v_shares_per_minute INTEGER;
    v_active_connections INTEGER;
BEGIN
    -- Count shares in last minute
    SELECT COUNT(*) INTO v_shares_per_minute
    FROM shares 
    WHERE timestamp > NOW() - INTERVAL '1 minute';
    
    -- Count active connections
    SELECT COUNT(*) INTO v_active_connections
    FROM pg_stat_activity 
    WHERE state = 'active';
    
    INSERT INTO db_activity_metrics (shares_per_minute, active_connections, avg_query_time_ms)
    VALUES (v_shares_per_minute, v_active_connections, 0);
    
    -- Keep only last 24 hours of metrics
    DELETE FROM db_activity_metrics 
    WHERE recorded_at < NOW() - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;

-- Function to check if maintenance should run (during low activity)
CREATE OR REPLACE FUNCTION should_run_maintenance()
RETURNS BOOLEAN AS $$
DECLARE
    v_avg_activity VARCHAR(20);
    v_current_hour INTEGER;
BEGIN
    -- Get average activity level from last 15 minutes
    SELECT activity_level INTO v_avg_activity
    FROM db_activity_metrics
    WHERE recorded_at > NOW() - INTERVAL '15 minutes'
    ORDER BY recorded_at DESC
    LIMIT 1;
    
    -- Get current hour (UTC)
    v_current_hour := EXTRACT(HOUR FROM NOW());
    
    -- Run maintenance if:
    -- 1. Activity is low, OR
    -- 2. It's during off-peak hours (2-6 AM UTC) regardless of activity
    RETURN (v_avg_activity = 'low' OR v_avg_activity IS NULL) 
        OR (v_current_hour >= 2 AND v_current_hour <= 6);
END;
$$ LANGUAGE plpgsql;

-- Function to perform smart vacuum analyze
CREATE OR REPLACE FUNCTION perform_smart_maintenance()
RETURNS void AS $$
DECLARE
    v_start_time TIMESTAMP;
    v_dead_tuples BIGINT;
    v_table_name TEXT;
    v_log_id BIGINT;
BEGIN
    -- Only run if activity is low
    IF NOT should_run_maintenance() THEN
        RAISE NOTICE 'Skipping maintenance - activity too high';
        RETURN;
    END IF;
    
    -- Check shares table for dead tuples
    SELECT n_dead_tup INTO v_dead_tuples
    FROM pg_stat_user_tables
    WHERE relname = 'shares';
    
    -- If more than 10000 dead tuples, vacuum analyze
    IF v_dead_tuples > 10000 THEN
        v_start_time := clock_timestamp();
        
        -- Log start
        INSERT INTO db_maintenance_log (operation, table_name, activity_level, notes)
        VALUES ('VACUUM ANALYZE', 'shares', 
                (SELECT activity_level FROM db_activity_metrics ORDER BY recorded_at DESC LIMIT 1),
                'Dead tuples: ' || v_dead_tuples)
        RETURNING id INTO v_log_id;
        
        -- Perform vacuum analyze
        VACUUM ANALYZE shares;
        
        -- Log completion
        UPDATE db_maintenance_log
        SET completed_at = clock_timestamp(),
            duration_ms = EXTRACT(MILLISECONDS FROM clock_timestamp() - v_start_time)::INTEGER,
            rows_affected = v_dead_tuples
        WHERE id = v_log_id;
        
        RAISE NOTICE 'Vacuumed shares table, % dead tuples cleaned', v_dead_tuples;
    END IF;
    
    -- Also check miners table
    SELECT n_dead_tup INTO v_dead_tuples
    FROM pg_stat_user_tables
    WHERE relname = 'miners';
    
    IF v_dead_tuples > 500 THEN
        v_start_time := clock_timestamp();
        
        INSERT INTO db_maintenance_log (operation, table_name, activity_level, notes)
        VALUES ('VACUUM ANALYZE', 'miners', 
                (SELECT activity_level FROM db_activity_metrics ORDER BY recorded_at DESC LIMIT 1),
                'Dead tuples: ' || v_dead_tuples)
        RETURNING id INTO v_log_id;
        
        VACUUM ANALYZE miners;
        
        UPDATE db_maintenance_log
        SET completed_at = clock_timestamp(),
            duration_ms = EXTRACT(MILLISECONDS FROM clock_timestamp() - v_start_time)::INTEGER,
            rows_affected = v_dead_tuples
        WHERE id = v_log_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 4. SHARES TABLE ARCHIVAL FOR LONG-TERM STORAGE
-- ============================================================================

-- Archive table for old shares (older than 30 days)
CREATE TABLE IF NOT EXISTS shares_archive (
    LIKE shares INCLUDING ALL
);

-- Function to archive old shares (run during low activity)
CREATE OR REPLACE FUNCTION archive_old_shares()
RETURNS INTEGER AS $$
DECLARE
    v_archived_count INTEGER;
BEGIN
    IF NOT should_run_maintenance() THEN
        RETURN 0;
    END IF;
    
    -- Move shares older than 30 days to archive
    WITH moved AS (
        DELETE FROM shares
        WHERE timestamp < NOW() - INTERVAL '30 days'
        RETURNING *
    )
    INSERT INTO shares_archive
    SELECT * FROM moved;
    
    GET DIAGNOSTICS v_archived_count = ROW_COUNT;
    
    IF v_archived_count > 0 THEN
        INSERT INTO db_maintenance_log (operation, table_name, rows_affected, completed_at, notes)
        VALUES ('ARCHIVE', 'shares', v_archived_count, NOW(), 
                'Archived shares older than 30 days');
    END IF;
    
    RETURN v_archived_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 5. INDEX FOR SHARES ARCHIVE
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_shares_archive_timestamp 
ON shares_archive (timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_shares_archive_user 
ON shares_archive (user_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_shares_archive_miner 
ON shares_archive (miner_id, timestamp DESC);

-- ============================================================================
-- 6. UPDATE STATISTICS
-- ============================================================================

ANALYZE shares;
ANALYZE miners;
ANALYZE miner_stats_history;

-- Log migration completion
INSERT INTO db_maintenance_log (operation, table_name, completed_at, notes)
VALUES ('MIGRATION', '020_database_optimization', NOW(), 'Added composite indexes, partitioning functions, and automated maintenance');
