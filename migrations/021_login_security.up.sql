-- Migration 021: Login Security - Failed attempt tracking and account lockout
-- Financial services grade security for brute force protection

-- Track failed login attempts per IP and email
CREATE TABLE IF NOT EXISTS login_attempts (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL DEFAULT false,
    failure_reason VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Account lockout tracking
CREATE TABLE IF NOT EXISTS account_lockouts (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    locked_until TIMESTAMP WITH TIME ZONE NOT NULL,
    lockout_count INT NOT NULL DEFAULT 1,
    last_failed_attempt TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_login_attempts_email ON login_attempts(email);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip ON login_attempts(ip_address);
CREATE INDEX IF NOT EXISTS idx_login_attempts_created ON login_attempts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_attempts_email_ip_recent ON login_attempts(email, ip_address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_account_lockouts_email ON account_lockouts(email);
CREATE INDEX IF NOT EXISTS idx_account_lockouts_locked_until ON account_lockouts(locked_until);

-- Function to check if account is locked
CREATE OR REPLACE FUNCTION is_account_locked(check_email VARCHAR(255))
RETURNS BOOLEAN AS $$
DECLARE
    lock_time TIMESTAMP WITH TIME ZONE;
BEGIN
    SELECT locked_until INTO lock_time
    FROM account_lockouts
    WHERE email = check_email;
    
    IF lock_time IS NULL THEN
        RETURN false;
    END IF;
    
    IF lock_time > NOW() THEN
        RETURN true;
    END IF;
    
    -- Lock expired, clean up
    DELETE FROM account_lockouts WHERE email = check_email;
    RETURN false;
END;
$$ LANGUAGE plpgsql;

-- Function to record failed login and potentially lock account
CREATE OR REPLACE FUNCTION record_failed_login(
    p_email VARCHAR(255),
    p_ip VARCHAR(45),
    p_user_agent TEXT,
    p_reason VARCHAR(100)
)
RETURNS TABLE(is_locked BOOLEAN, locked_until TIMESTAMP WITH TIME ZONE, attempts_remaining INT) AS $$
DECLARE
    recent_failures INT;
    lockout_duration INTERVAL;
    current_lockout_count INT;
    new_locked_until TIMESTAMP WITH TIME ZONE;
    max_attempts INT := 5;  -- Lock after 5 failed attempts
    attempt_window INTERVAL := '15 minutes';  -- Within 15 minute window
BEGIN
    -- Record the failed attempt
    INSERT INTO login_attempts (email, ip_address, user_agent, success, failure_reason)
    VALUES (p_email, p_ip, p_user_agent, false, p_reason);
    
    -- Count recent failures for this email
    SELECT COUNT(*) INTO recent_failures
    FROM login_attempts
    WHERE email = p_email
      AND success = false
      AND created_at > NOW() - attempt_window;
    
    -- Check if we need to lock the account
    IF recent_failures >= max_attempts THEN
        -- Get current lockout count for progressive lockout
        SELECT lockout_count INTO current_lockout_count
        FROM account_lockouts
        WHERE email = p_email;
        
        IF current_lockout_count IS NULL THEN
            current_lockout_count := 0;
        END IF;
        
        -- Progressive lockout: 15min, 30min, 1hr, 2hr, 4hr, 24hr
        CASE current_lockout_count
            WHEN 0 THEN lockout_duration := '15 minutes';
            WHEN 1 THEN lockout_duration := '30 minutes';
            WHEN 2 THEN lockout_duration := '1 hour';
            WHEN 3 THEN lockout_duration := '2 hours';
            WHEN 4 THEN lockout_duration := '4 hours';
            ELSE lockout_duration := '24 hours';
        END CASE;
        
        new_locked_until := NOW() + lockout_duration;
        
        -- Upsert lockout record
        INSERT INTO account_lockouts (email, locked_until, lockout_count, last_failed_attempt)
        VALUES (p_email, new_locked_until, current_lockout_count + 1, NOW())
        ON CONFLICT (email) DO UPDATE SET
            locked_until = new_locked_until,
            lockout_count = account_lockouts.lockout_count + 1,
            last_failed_attempt = NOW();
        
        RETURN QUERY SELECT true, new_locked_until, 0;
    ELSE
        RETURN QUERY SELECT false, NULL::TIMESTAMP WITH TIME ZONE, max_attempts - recent_failures;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to record successful login (clears lockout)
CREATE OR REPLACE FUNCTION record_successful_login(
    p_email VARCHAR(255),
    p_ip VARCHAR(45),
    p_user_agent TEXT
)
RETURNS VOID AS $$
BEGIN
    -- Record successful login
    INSERT INTO login_attempts (email, ip_address, user_agent, success)
    VALUES (p_email, p_ip, p_user_agent, true);
    
    -- Clear any lockout for this email
    DELETE FROM account_lockouts WHERE email = p_email;
END;
$$ LANGUAGE plpgsql;

-- Cleanup old login attempts (keep 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_login_attempts()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM login_attempts WHERE created_at < NOW() - INTERVAL '30 days';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE login_attempts IS 'Tracks all login attempts for security auditing';
COMMENT ON TABLE account_lockouts IS 'Tracks account lockouts for brute force protection';
