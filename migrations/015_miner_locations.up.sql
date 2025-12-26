-- Migration 015: Add miner location tracking
-- Enables GeoIP-based location tracking for miners on the world map

-- Add location fields to miners table
ALTER TABLE miners ADD COLUMN IF NOT EXISTS ip_address INET;
ALTER TABLE miners ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS country VARCHAR(100);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS country_code VARCHAR(3);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS continent VARCHAR(50);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS latitude DECIMAL(10, 7);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS longitude DECIMAL(10, 7);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS location_updated_at TIMESTAMP WITH TIME ZONE;

-- Create index for location queries
CREATE INDEX IF NOT EXISTS idx_miners_country ON miners(country);
CREATE INDEX IF NOT EXISTS idx_miners_continent ON miners(continent);
CREATE INDEX IF NOT EXISTS idx_miners_location ON miners(latitude, longitude) WHERE latitude IS NOT NULL;

-- Create aggregated location stats view for efficient querying
CREATE OR REPLACE VIEW miner_location_stats AS
SELECT 
    country,
    country_code,
    continent,
    COUNT(*) as miner_count,
    COUNT(*) FILTER (WHERE is_active AND last_seen > NOW() - INTERVAL '15 minutes') as active_count,
    SUM(hashrate) as total_hashrate
FROM miners
WHERE country IS NOT NULL
GROUP BY country, country_code, continent;

-- Create view for location markers (aggregated by city)
CREATE OR REPLACE VIEW miner_location_markers AS
SELECT 
    city,
    country,
    country_code,
    continent,
    AVG(latitude) as lat,
    AVG(longitude) as lng,
    COUNT(*) as miner_count,
    COUNT(*) FILTER (WHERE is_active AND last_seen > NOW() - INTERVAL '15 minutes') as active_count,
    SUM(hashrate) as total_hashrate,
    BOOL_OR(is_active AND last_seen > NOW() - INTERVAL '15 minutes') as is_active
FROM miners
WHERE latitude IS NOT NULL AND longitude IS NOT NULL
GROUP BY city, country, country_code, continent;
