-- Community and Monitoring Tables Migration

-- Teams table
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    total_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    member_count INTEGER NOT NULL DEFAULT 0,
    total_shares BIGINT NOT NULL DEFAULT 0,
    blocks_found INTEGER NOT NULL DEFAULT 0,
    
    CONSTRAINT teams_name_unique UNIQUE(name),
    CONSTRAINT teams_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- Team members table
CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY,
    team_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    CONSTRAINT team_members_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT team_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT team_members_unique UNIQUE(team_id, user_id)
);

-- Referrals table
CREATE TABLE IF NOT EXISTS referrals (
    id UUID PRIMARY KEY,
    referrer_id UUID NOT NULL,
    referred_id UUID,
    code VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    bonus_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    CONSTRAINT referrals_referrer_id_fkey FOREIGN KEY (referrer_id) REFERENCES users(id),
    CONSTRAINT referrals_referred_id_fkey FOREIGN KEY (referred_id) REFERENCES users(id)
);

-- Competitions table
CREATE TABLE IF NOT EXISTS competitions (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    prize_pool DOUBLE PRECISION NOT NULL DEFAULT 0,
    rules TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'upcoming',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Competition participants table
CREATE TABLE IF NOT EXISTS competition_participants (
    id UUID PRIMARY KEY,
    competition_id UUID NOT NULL,
    user_id UUID NOT NULL,
    team_id UUID,
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    total_shares BIGINT NOT NULL DEFAULT 0,
    total_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    rank INTEGER NOT NULL DEFAULT 0,
    prize DOUBLE PRECISION NOT NULL DEFAULT 0,
    
    CONSTRAINT competition_participants_competition_id_fkey FOREIGN KEY (competition_id) REFERENCES competitions(id),
    CONSTRAINT competition_participants_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT competition_participants_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id),
    CONSTRAINT competition_participants_unique UNIQUE(competition_id, user_id)
);

-- Social shares table
CREATE TABLE IF NOT EXISTS social_shares (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    platform VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    share_url VARCHAR(500),
    milestone VARCHAR(100) NOT NULL,
    shared_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    bonus_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    
    CONSTRAINT social_shares_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Team statistics table
CREATE TABLE IF NOT EXISTS team_statistics (
    team_id UUID NOT NULL,
    period VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    total_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_shares BIGINT NOT NULL DEFAULT 0,
    blocks_found INTEGER NOT NULL DEFAULT 0,
    earnings DOUBLE PRECISION NOT NULL DEFAULT 0,
    member_count INTEGER NOT NULL DEFAULT 0,
    
    PRIMARY KEY (team_id, period, date),
    CONSTRAINT team_statistics_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id)
);

-- Metrics table
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    labels JSONB,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    type VARCHAR(50) NOT NULL
);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    labels JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Alert rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    query TEXT NOT NULL,
    condition VARCHAR(10) NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    duration VARCHAR(20) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Dashboards table
CREATE TABLE IF NOT EXISTS dashboards (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT dashboards_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Performance metrics table
CREATE TABLE IF NOT EXISTS performance_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    disk_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
    network_in DOUBLE PRECISION NOT NULL DEFAULT 0,
    network_out DOUBLE PRECISION NOT NULL DEFAULT 0,
    active_miners INTEGER NOT NULL DEFAULT 0,
    total_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    shares_per_second DOUBLE PRECISION NOT NULL DEFAULT 0,
    blocks_found INTEGER NOT NULL DEFAULT 0,
    uptime DOUBLE PRECISION NOT NULL DEFAULT 0
);

-- Miner metrics table
CREATE TABLE IF NOT EXISTS miner_metrics (
    id SERIAL PRIMARY KEY,
    miner_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    shares_submitted BIGINT NOT NULL DEFAULT 0,
    shares_accepted BIGINT NOT NULL DEFAULT 0,
    shares_rejected BIGINT NOT NULL DEFAULT 0,
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_online BOOLEAN NOT NULL DEFAULT false,
    difficulty DOUBLE PRECISION NOT NULL DEFAULT 0,
    earnings DOUBLE PRECISION NOT NULL DEFAULT 0,
    
    CONSTRAINT miner_metrics_miner_id_fkey FOREIGN KEY (miner_id) REFERENCES miners(id)
);

-- Pool metrics table
CREATE TABLE IF NOT EXISTS pool_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    total_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    active_miners INTEGER NOT NULL DEFAULT 0,
    total_shares BIGINT NOT NULL DEFAULT 0,
    valid_shares BIGINT NOT NULL DEFAULT 0,
    invalid_shares BIGINT NOT NULL DEFAULT 0,
    blocks_found INTEGER NOT NULL DEFAULT 0,
    network_difficulty DOUBLE PRECISION NOT NULL DEFAULT 0,
    pool_difficulty DOUBLE PRECISION NOT NULL DEFAULT 0,
    network_hashrate DOUBLE PRECISION NOT NULL DEFAULT 0,
    pool_efficiency DOUBLE PRECISION NOT NULL DEFAULT 0,
    luck DOUBLE PRECISION NOT NULL DEFAULT 0
);

-- Alert channels table
CREATE TABLE IF NOT EXISTS alert_channels (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    alert_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    error TEXT,
    
    CONSTRAINT notifications_alert_id_fkey FOREIGN KEY (alert_id) REFERENCES alerts(id),
    CONSTRAINT notifications_channel_id_fkey FOREIGN KEY (channel_id) REFERENCES alert_channels(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_teams_owner_id ON teams(owner_id);
CREATE INDEX IF NOT EXISTS idx_teams_is_active ON teams(is_active);
CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON team_members(team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);
CREATE INDEX IF NOT EXISTS idx_team_members_is_active ON team_members(is_active);
CREATE INDEX IF NOT EXISTS idx_referrals_code ON referrals(code);
CREATE INDEX IF NOT EXISTS idx_referrals_referrer_id ON referrals(referrer_id);
CREATE INDEX IF NOT EXISTS idx_referrals_status ON referrals(status);
CREATE INDEX IF NOT EXISTS idx_competitions_status ON competitions(status);
CREATE INDEX IF NOT EXISTS idx_competitions_start_time ON competitions(start_time);
CREATE INDEX IF NOT EXISTS idx_competition_participants_competition_id ON competition_participants(competition_id);
CREATE INDEX IF NOT EXISTS idx_competition_participants_user_id ON competition_participants(user_id);
CREATE INDEX IF NOT EXISTS idx_social_shares_user_id ON social_shares(user_id);
CREATE INDEX IF NOT EXISTS idx_social_shares_platform ON social_shares(platform);
CREATE INDEX IF NOT EXISTS idx_social_shares_milestone ON social_shares(milestone);
CREATE INDEX IF NOT EXISTS idx_team_statistics_team_id ON team_statistics(team_id);
CREATE INDEX IF NOT EXISTS idx_team_statistics_period ON team_statistics(period);
CREATE INDEX IF NOT EXISTS idx_team_statistics_date ON team_statistics(date);
CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(type);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
CREATE INDEX IF NOT EXISTS idx_alert_rules_is_active ON alert_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_dashboards_created_by ON dashboards(created_by);
CREATE INDEX IF NOT EXISTS idx_dashboards_is_public ON dashboards(is_public);
CREATE INDEX IF NOT EXISTS idx_performance_metrics_timestamp ON performance_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_miner_metrics_miner_id ON miner_metrics(miner_id);
CREATE INDEX IF NOT EXISTS idx_miner_metrics_timestamp ON miner_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_pool_metrics_timestamp ON pool_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_alert_channels_type ON alert_channels(type);
CREATE INDEX IF NOT EXISTS idx_alert_channels_is_active ON alert_channels(is_active);
CREATE INDEX IF NOT EXISTS idx_notifications_alert_id ON notifications(alert_id);
CREATE INDEX IF NOT EXISTS idx_notifications_channel_id ON notifications(channel_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);