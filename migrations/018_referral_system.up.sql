-- ============================================================================
-- REFERRAL SYSTEM
-- Allows users to refer others and receive pool fee discounts
-- ============================================================================

-- Referral codes table
CREATE TABLE IF NOT EXISTS referral_codes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL UNIQUE,
    description VARCHAR(255),
    -- Discount settings
    referrer_discount_percent DECIMAL(5,2) DEFAULT 10.00,  -- Discount for the referrer (permanent)
    referee_discount_percent DECIMAL(5,2) DEFAULT 5.00,    -- Discount for new user (temporary)
    referee_discount_days INTEGER DEFAULT 30,               -- How long referee discount lasts
    -- Limits
    max_uses INTEGER DEFAULT NULL,                          -- NULL = unlimited
    times_used INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL        -- NULL = never expires
);

-- Referrals tracking table
CREATE TABLE IF NOT EXISTS referrals (
    id SERIAL PRIMARY KEY,
    referral_code_id INTEGER NOT NULL REFERENCES referral_codes(id) ON DELETE CASCADE,
    referrer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referee_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'expired', 'cancelled')),
    -- Discount tracking
    referee_discount_percent DECIMAL(5,2) NOT NULL,
    referee_discount_expires_at TIMESTAMP WITH TIME ZONE,
    -- Stats
    referee_total_shares BIGINT DEFAULT 0,
    referee_total_hashrate DECIMAL(20,2) DEFAULT 0,
    -- Clout bonus (added to referrer's engagement score)
    clout_bonus_awarded INTEGER DEFAULT 0,
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confirmed_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(referee_id)  -- Each user can only be referred once
);

-- Add referral fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by INTEGER REFERENCES users(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code_used VARCHAR(20);
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_discount_percent DECIMAL(5,2) DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_discount_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_referrals INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referrer_discount_percent DECIMAL(5,2) DEFAULT 0;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_referral_codes_user_id ON referral_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_referral_codes_code ON referral_codes(code);
CREATE INDEX IF NOT EXISTS idx_referrals_referrer_id ON referrals(referrer_id);
CREATE INDEX IF NOT EXISTS idx_referrals_referee_id ON referrals(referee_id);
CREATE INDEX IF NOT EXISTS idx_referrals_status ON referrals(status);
CREATE INDEX IF NOT EXISTS idx_users_referred_by ON users(referred_by);

-- Function to generate unique referral code
CREATE OR REPLACE FUNCTION generate_referral_code(username VARCHAR)
RETURNS VARCHAR AS $$
DECLARE
    base_code VARCHAR;
    final_code VARCHAR;
    suffix INTEGER := 0;
BEGIN
    -- Create base code from username (uppercase, first 6 chars + random)
    base_code := UPPER(LEFT(REGEXP_REPLACE(username, '[^a-zA-Z0-9]', '', 'g'), 6));
    final_code := base_code || LPAD(FLOOR(RANDOM() * 1000)::TEXT, 3, '0');
    
    -- Ensure uniqueness
    WHILE EXISTS (SELECT 1 FROM referral_codes WHERE code = final_code) LOOP
        suffix := suffix + 1;
        final_code := base_code || LPAD((FLOOR(RANDOM() * 1000) + suffix)::TEXT, 3, '0');
    END LOOP;
    
    RETURN final_code;
END;
$$ LANGUAGE plpgsql;

-- Function to apply referral on user registration
CREATE OR REPLACE FUNCTION apply_referral(
    p_referee_id INTEGER,
    p_referral_code VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    v_code_record RECORD;
    v_referrer_id INTEGER;
BEGIN
    -- Find the referral code
    SELECT rc.*, u.id as owner_id
    INTO v_code_record
    FROM referral_codes rc
    JOIN users u ON rc.user_id = u.id
    WHERE rc.code = UPPER(p_referral_code)
      AND rc.is_active = true
      AND (rc.expires_at IS NULL OR rc.expires_at > NOW())
      AND (rc.max_uses IS NULL OR rc.times_used < rc.max_uses);
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    v_referrer_id := v_code_record.owner_id;
    
    -- Don't allow self-referral
    IF v_referrer_id = p_referee_id THEN
        RETURN FALSE;
    END IF;
    
    -- Apply discount to referee
    UPDATE users SET
        referred_by = v_referrer_id,
        referral_code_used = UPPER(p_referral_code),
        referral_discount_percent = v_code_record.referee_discount_percent,
        referral_discount_expires_at = NOW() + (v_code_record.referee_discount_days || ' days')::INTERVAL
    WHERE id = p_referee_id;
    
    -- Update referrer's permanent discount (stacks up to 50%)
    UPDATE users SET
        total_referrals = total_referrals + 1,
        referrer_discount_percent = LEAST(referrer_discount_percent + v_code_record.referrer_discount_percent, 50.00)
    WHERE id = v_referrer_id;
    
    -- Create referral record
    INSERT INTO referrals (
        referral_code_id, referrer_id, referee_id, status,
        referee_discount_percent, referee_discount_expires_at
    ) VALUES (
        v_code_record.id, v_referrer_id, p_referee_id, 'confirmed',
        v_code_record.referee_discount_percent,
        NOW() + (v_code_record.referee_discount_days || ' days')::INTERVAL
    );
    
    -- Increment code usage
    UPDATE referral_codes SET times_used = times_used + 1 WHERE id = v_code_record.id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate effective pool fee for a user
CREATE OR REPLACE FUNCTION get_effective_pool_fee(p_user_id INTEGER)
RETURNS DECIMAL AS $$
DECLARE
    v_base_fee DECIMAL := 1.0;  -- Base 1% pool fee
    v_referrer_discount DECIMAL;
    v_referee_discount DECIMAL;
    v_discount_expires TIMESTAMP WITH TIME ZONE;
BEGIN
    SELECT 
        COALESCE(referrer_discount_percent, 0),
        COALESCE(referral_discount_percent, 0),
        referral_discount_expires_at
    INTO v_referrer_discount, v_referee_discount, v_discount_expires
    FROM users WHERE id = p_user_id;
    
    -- Check if referee discount is still valid
    IF v_discount_expires IS NOT NULL AND v_discount_expires < NOW() THEN
        v_referee_discount := 0;
    END IF;
    
    -- Calculate effective fee (base - discounts, minimum 0.1%)
    RETURN GREATEST(v_base_fee * (1 - (v_referrer_discount + v_referee_discount) / 100), 0.1);
END;
$$ LANGUAGE plpgsql;

-- Create default referral code for existing users
INSERT INTO referral_codes (user_id, code, description)
SELECT id, generate_referral_code(username), 'Auto-generated referral code'
FROM users
WHERE NOT EXISTS (SELECT 1 FROM referral_codes WHERE referral_codes.user_id = users.id)
ON CONFLICT DO NOTHING;

COMMENT ON TABLE referral_codes IS 'Referral codes that users can share to invite others';
COMMENT ON TABLE referrals IS 'Tracks who referred whom and their discounts';
COMMENT ON FUNCTION get_effective_pool_fee IS 'Calculates the effective pool fee after referral discounts';
