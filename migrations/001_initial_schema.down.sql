-- Rollback initial schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_miners_updated_at ON miners;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS shares;
DROP TABLE IF EXISTS miners;
DROP TABLE IF EXISTS users;