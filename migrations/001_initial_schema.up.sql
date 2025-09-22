-- Initial schema for Chimera Pool
-- Requirements: 6.1, 6.2

-- Users table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

-- Create index for faster lookups
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active);

-- Miners table
CREATE TABLE miners (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    address INET,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    hashrate DECIMAL(20,2) DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for miners
CREATE INDEX idx_miners_user_id ON miners(user_id);
CREATE INDEX idx_miners_active ON miners(is_active);
CREATE INDEX idx_miners_last_seen ON miners(last_seen);

-- Shares table
CREATE TABLE shares (
    id BIGSERIAL PRIMARY KEY,
    miner_id BIGINT NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    difficulty DECIMAL(20,8) NOT NULL,
    is_valid BOOLEAN NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    nonce VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL
);

-- Create indexes for shares (important for performance)
CREATE INDEX idx_shares_miner_id ON shares(miner_id);
CREATE INDEX idx_shares_user_id ON shares(user_id);
CREATE INDEX idx_shares_timestamp ON shares(timestamp);
CREATE INDEX idx_shares_valid ON shares(is_valid);

-- Blocks table
CREATE TABLE blocks (
    id BIGSERIAL PRIMARY KEY,
    height BIGINT UNIQUE NOT NULL,
    hash VARCHAR(64) UNIQUE NOT NULL,
    finder_id BIGINT NOT NULL REFERENCES users(id),
    reward BIGINT NOT NULL,
    difficulty DECIMAL(20,8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'orphaned'))
);

-- Create indexes for blocks
CREATE INDEX idx_blocks_height ON blocks(height);
CREATE INDEX idx_blocks_finder_id ON blocks(finder_id);
CREATE INDEX idx_blocks_status ON blocks(status);
CREATE INDEX idx_blocks_timestamp ON blocks(timestamp);

-- Payouts table
CREATE TABLE payouts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    address VARCHAR(255) NOT NULL,
    tx_hash VARCHAR(64),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'confirmed', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for payouts
CREATE INDEX idx_payouts_user_id ON payouts(user_id);
CREATE INDEX idx_payouts_status ON payouts(status);
CREATE INDEX idx_payouts_created_at ON payouts(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_miners_updated_at BEFORE UPDATE ON miners
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();