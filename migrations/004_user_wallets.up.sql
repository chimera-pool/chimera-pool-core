-- Migration: Add multiple wallet support with payment splitting
-- Users can have multiple wallets with percentage-based allocation

-- User wallets table
CREATE TABLE user_wallets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address VARCHAR(255) NOT NULL,
    label VARCHAR(100),
    percentage DECIMAL(5,2) NOT NULL DEFAULT 100.00 CHECK (percentage > 0 AND percentage <= 100),
    is_primary BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure unique address per user
    UNIQUE(user_id, address)
);

-- Create indexes for user_wallets
CREATE INDEX idx_user_wallets_user_id ON user_wallets(user_id);
CREATE INDEX idx_user_wallets_active ON user_wallets(is_active);
CREATE INDEX idx_user_wallets_primary ON user_wallets(is_primary);

-- Add trigger for updated_at
CREATE TRIGGER update_user_wallets_updated_at BEFORE UPDATE ON user_wallets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to validate wallet percentages sum to 100% for active wallets
CREATE OR REPLACE FUNCTION validate_wallet_percentages()
RETURNS TRIGGER AS $$
DECLARE
    total_percentage DECIMAL(5,2);
BEGIN
    -- Calculate total percentage for active wallets of this user
    SELECT COALESCE(SUM(percentage), 0) INTO total_percentage
    FROM user_wallets
    WHERE user_id = NEW.user_id 
      AND is_active = true
      AND id != COALESCE(NEW.id, 0);
    
    -- Add the new/updated wallet's percentage
    IF NEW.is_active THEN
        total_percentage := total_percentage + NEW.percentage;
    END IF;
    
    -- Allow if total is exactly 100% or if this is the only active wallet
    IF total_percentage > 100.00 THEN
        RAISE EXCEPTION 'Total wallet percentages cannot exceed 100%%. Current total would be: %%', total_percentage;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for percentage validation
CREATE TRIGGER validate_wallet_percentages_trigger
    BEFORE INSERT OR UPDATE ON user_wallets
    FOR EACH ROW EXECUTE FUNCTION validate_wallet_percentages();

-- Function to ensure at least one primary wallet when others exist
CREATE OR REPLACE FUNCTION ensure_primary_wallet()
RETURNS TRIGGER AS $$
BEGIN
    -- If setting this wallet as primary, unset others
    IF NEW.is_primary = true THEN
        UPDATE user_wallets 
        SET is_primary = false 
        WHERE user_id = NEW.user_id 
          AND id != COALESCE(NEW.id, 0)
          AND is_primary = true;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for primary wallet management
CREATE TRIGGER ensure_primary_wallet_trigger
    BEFORE INSERT OR UPDATE ON user_wallets
    FOR EACH ROW EXECUTE FUNCTION ensure_primary_wallet();

-- Migrate existing payout_address from users to user_wallets
-- This preserves existing wallet addresses
DO $$
DECLARE
    user_record RECORD;
BEGIN
    FOR user_record IN 
        SELECT id, payout_address 
        FROM users 
        WHERE payout_address IS NOT NULL AND payout_address != ''
    LOOP
        INSERT INTO user_wallets (user_id, address, label, percentage, is_primary, is_active)
        VALUES (user_record.id, user_record.payout_address, 'Primary Wallet', 100.00, true, true)
        ON CONFLICT (user_id, address) DO NOTHING;
    END LOOP;
END $$;

-- Add payout_wallet_id to payouts table to track which wallet received payment
ALTER TABLE payouts ADD COLUMN wallet_id BIGINT REFERENCES user_wallets(id);

-- Create index for wallet_id in payouts
CREATE INDEX idx_payouts_wallet_id ON payouts(wallet_id);
