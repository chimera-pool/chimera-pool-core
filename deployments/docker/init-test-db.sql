-- Initialize test database with required extensions and test data

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create test database if it doesn't exist
SELECT 'CREATE DATABASE chimera_pool_test'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'chimera_pool_test');

-- Connect to test database
\c chimera_pool_test;

-- Create test schema
CREATE SCHEMA IF NOT EXISTS test_data;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE chimera_pool_test TO test;
GRANT ALL PRIVILEGES ON SCHEMA public TO test;
GRANT ALL PRIVILEGES ON SCHEMA test_data TO test;

-- Create test data cleanup function
CREATE OR REPLACE FUNCTION test_data.cleanup_test_data()
RETURNS void AS $$
BEGIN
    -- Clean up test data older than 1 hour
    DELETE FROM payouts WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM shares WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM blocks WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM team_members WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM teams WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM user_sessions WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM users WHERE created_at < NOW() - INTERVAL '1 hour' AND username LIKE 'test_%';
END;
$$ LANGUAGE plpgsql;

-- Create test data generation function
CREATE OR REPLACE FUNCTION test_data.generate_test_data()
RETURNS void AS $$
DECLARE
    test_user_id UUID;
    test_team_id UUID;
BEGIN
    -- Create test users
    INSERT INTO users (id, username, email, password_hash, created_at)
    VALUES 
        (uuid_generate_v4(), 'test_user_1', 'test1@example.com', crypt('TestPassword123!', gen_salt('bf')), NOW()),
        (uuid_generate_v4(), 'test_user_2', 'test2@example.com', crypt('TestPassword123!', gen_salt('bf')), NOW()),
        (uuid_generate_v4(), 'test_user_3', 'test3@example.com', crypt('TestPassword123!', gen_salt('bf')), NOW())
    ON CONFLICT (username) DO NOTHING;
    
    -- Get a test user ID
    SELECT id INTO test_user_id FROM users WHERE username = 'test_user_1' LIMIT 1;
    
    -- Create test team
    INSERT INTO teams (id, name, description, creator_id, created_at)
    VALUES (uuid_generate_v4(), 'Test Mining Team', 'A team for integration testing', test_user_id, NOW())
    ON CONFLICT (name) DO NOTHING;
    
    -- Get test team ID
    SELECT id INTO test_team_id FROM teams WHERE name = 'Test Mining Team' LIMIT 1;
    
    -- Add user to team
    INSERT INTO team_members (team_id, user_id, joined_at)
    VALUES (test_team_id, test_user_id, NOW())
    ON CONFLICT (team_id, user_id) DO NOTHING;
    
    -- Create test shares
    INSERT INTO shares (id, miner_id, job_id, nonce, target, difficulty, valid, created_at)
    SELECT 
        uuid_generate_v4(),
        'test_user_1',
        'job_' || generate_series,
        generate_series * 12345,
        decode('0000ffff00000000000000000000000000000000000000000000000000000000', 'hex'),
        1000,
        true,
        NOW() - (generate_series || ' seconds')::interval
    FROM generate_series(1, 100);
    
    -- Create test blocks
    INSERT INTO blocks (id, height, hash, reward, timestamp, mined_by, created_at)
    VALUES 
        (uuid_generate_v4(), 100001, '0000000000000000000123456789abcdef', 5000000000, NOW() - INTERVAL '1 hour', 'test_user_1', NOW()),
        (uuid_generate_v4(), 100002, '0000000000000000000fedcba987654321', 5000000000, NOW() - INTERVAL '30 minutes', 'test_user_2', NOW());
    
    RAISE NOTICE 'Test data generated successfully';
END;
$$ LANGUAGE plpgsql;

-- Create indexes for better test performance
CREATE INDEX IF NOT EXISTS idx_users_username_test ON users(username) WHERE username LIKE 'test_%';
CREATE INDEX IF NOT EXISTS idx_shares_miner_test ON shares(miner_id) WHERE miner_id LIKE 'test_%';
CREATE INDEX IF NOT EXISTS idx_shares_created_test ON shares(created_at) WHERE created_at > NOW() - INTERVAL '1 day';

-- Set up test database configuration
ALTER DATABASE chimera_pool_test SET timezone = 'UTC';
ALTER DATABASE chimera_pool_test SET log_statement = 'all';
ALTER DATABASE chimera_pool_test SET log_min_duration_statement = 100;