-- Rollback Migration 015: Remove miner location tracking

DROP VIEW IF EXISTS miner_location_markers;
DROP VIEW IF EXISTS miner_location_stats;

DROP INDEX IF EXISTS idx_miners_location;
DROP INDEX IF EXISTS idx_miners_continent;
DROP INDEX IF EXISTS idx_miners_country;

ALTER TABLE miners DROP COLUMN IF EXISTS location_updated_at;
ALTER TABLE miners DROP COLUMN IF EXISTS longitude;
ALTER TABLE miners DROP COLUMN IF EXISTS latitude;
ALTER TABLE miners DROP COLUMN IF EXISTS continent;
ALTER TABLE miners DROP COLUMN IF EXISTS country_code;
ALTER TABLE miners DROP COLUMN IF EXISTS country;
ALTER TABLE miners DROP COLUMN IF EXISTS city;
ALTER TABLE miners DROP COLUMN IF EXISTS ip_address;
