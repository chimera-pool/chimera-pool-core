-- Migration 022: Wallet Enhancements
-- Adds wallet types, per-miner assignments, and enhanced allocation functions
-- Note: user_wallets table already has is_active, percentage, label columns

-- Add new columns to user_wallets table
ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS wallet_type VARCHAR(20) DEFAULT 'hot';
ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS is_locked BOOLEAN DEFAULT false;
ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS min_payout_threshold DECIMAL(18,8) DEFAULT 0;
ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS network VARCHAR(50) DEFAULT 'litecoin';

-- Update percentage check constraint to allow 0
ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS user_wallets_percentage_check;
ALTER TABLE user_wallets ADD CONSTRAINT user_wallets_percentage_check CHECK (percentage >= 0 AND percentage <= 100);

-- Create miner-wallet assignments table for per-miner wallet allocation
CREATE TABLE IF NOT EXISTS miner_wallet_assignments (
    id SERIAL PRIMARY KEY,
    miner_id BIGINT REFERENCES miners(id) ON DELETE CASCADE,
    wallet_id BIGINT REFERENCES user_wallets(id) ON DELETE CASCADE,
    allocation_percent DECIMAL(5,2) NOT NULL DEFAULT 100 CHECK (allocation_percent >= 0 AND allocation_percent <= 100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(miner_id, wallet_id)
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_miner_wallet_assignments_miner ON miner_wallet_assignments(miner_id);
CREATE INDEX IF NOT EXISTS idx_miner_wallet_assignments_wallet ON miner_wallet_assignments(wallet_id);
CREATE INDEX IF NOT EXISTS idx_user_wallets_network ON user_wallets(network);

-- Function to toggle wallet active status and redistribute allocation
CREATE OR REPLACE FUNCTION toggle_wallet_active(
    p_wallet_id BIGINT,
    p_is_active BOOLEAN
) RETURNS TABLE(wallet_id BIGINT, is_active BOOLEAN, new_percentage DECIMAL) AS $$
DECLARE
    v_user_id BIGINT;
    v_network VARCHAR;
    v_old_percentage DECIMAL(5,2);
    v_remaining_wallets INTEGER;
BEGIN
    -- Get wallet info
    SELECT user_id, network, percentage 
    INTO v_user_id, v_network, v_old_percentage
    FROM user_wallets WHERE id = p_wallet_id;
    
    IF NOT p_is_active THEN
        -- Deactivating: redistribute this wallet's allocation to others
        UPDATE user_wallets
        SET is_active = false,
            percentage = 0,
            updated_at = NOW()
        WHERE id = p_wallet_id;
        
        -- Count remaining active wallets
        SELECT COUNT(*) INTO v_remaining_wallets
        FROM user_wallets
        WHERE user_id = v_user_id 
          AND network = v_network 
          AND is_active = true;
        
        IF v_remaining_wallets > 0 AND v_old_percentage > 0 THEN
            -- Distribute the freed allocation equally
            UPDATE user_wallets
            SET percentage = percentage + (v_old_percentage / v_remaining_wallets),
                updated_at = NOW()
            WHERE user_id = v_user_id 
              AND network = v_network 
              AND is_active = true;
        END IF;
    ELSE
        -- Activating: set to equal share with others
        SELECT COUNT(*) INTO v_remaining_wallets
        FROM user_wallets
        WHERE user_id = v_user_id 
          AND network = v_network 
          AND is_active = true;
        
        -- Calculate equal share
        v_old_percentage := 100.0 / (v_remaining_wallets + 1);
        
        -- Reduce existing wallets proportionally
        IF v_remaining_wallets > 0 THEN
            UPDATE user_wallets
            SET percentage = percentage * (v_remaining_wallets::DECIMAL / (v_remaining_wallets + 1)),
                updated_at = NOW()
            WHERE user_id = v_user_id 
              AND network = v_network 
              AND is_active = true;
        ELSE
            v_old_percentage := 100.0;
        END IF;
        
        -- Activate this wallet with its share
        UPDATE user_wallets
        SET is_active = true,
            percentage = v_old_percentage,
            updated_at = NOW()
        WHERE id = p_wallet_id;
    END IF;
    
    -- Return all affected wallets
    RETURN QUERY
    SELECT w.id, w.is_active, w.percentage
    FROM user_wallets w
    WHERE w.user_id = v_user_id AND w.network = v_network;
END;
$$ LANGUAGE plpgsql;

-- Function to update wallet allocation with auto-balance
CREATE OR REPLACE FUNCTION update_wallet_percentage(
    p_wallet_id BIGINT,
    p_new_percentage DECIMAL(5,2)
) RETURNS TABLE(wallet_id BIGINT, new_percentage DECIMAL) AS $$
DECLARE
    v_user_id BIGINT;
    v_network VARCHAR;
    v_old_percentage DECIMAL(5,2);
    v_diff DECIMAL(5,2);
    v_other_total DECIMAL(5,2);
BEGIN
    -- Get wallet info
    SELECT user_id, network, percentage 
    INTO v_user_id, v_network, v_old_percentage
    FROM user_wallets WHERE id = p_wallet_id;
    
    -- Calculate difference
    v_diff := p_new_percentage - v_old_percentage;
    
    -- Get total of other active wallets
    SELECT COALESCE(SUM(percentage), 0) INTO v_other_total
    FROM user_wallets
    WHERE user_id = v_user_id 
      AND network = v_network 
      AND is_active = true
      AND id != p_wallet_id;
    
    -- Update the target wallet
    UPDATE user_wallets
    SET percentage = p_new_percentage,
        updated_at = NOW()
    WHERE id = p_wallet_id;
    
    -- Adjust other wallets proportionally
    IF v_other_total > 0 AND v_diff != 0 THEN
        UPDATE user_wallets
        SET percentage = GREATEST(0, LEAST(100, percentage - (percentage / v_other_total) * v_diff)),
            updated_at = NOW()
        WHERE user_id = v_user_id 
          AND network = v_network 
          AND is_active = true
          AND id != p_wallet_id;
    END IF;
    
    -- Return all affected wallets
    RETURN QUERY
    SELECT w.id, w.percentage
    FROM user_wallets w
    WHERE w.user_id = v_user_id AND w.network = v_network AND w.is_active = true;
END;
$$ LANGUAGE plpgsql;

-- Function to assign a wallet to a specific miner
CREATE OR REPLACE FUNCTION assign_miner_wallet(
    p_miner_id BIGINT,
    p_wallet_id BIGINT,
    p_allocation_percent DECIMAL(5,2) DEFAULT 100
) RETURNS VOID AS $$
BEGIN
    INSERT INTO miner_wallet_assignments (miner_id, wallet_id, allocation_percent)
    VALUES (p_miner_id, p_wallet_id, p_allocation_percent)
    ON CONFLICT (miner_id, wallet_id) 
    DO UPDATE SET allocation_percent = p_allocation_percent, updated_at = NOW();
END;
$$ LANGUAGE plpgsql;
