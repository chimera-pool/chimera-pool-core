-- Drop Community and Monitoring Tables Migration

-- Drop indexes first
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_channel_id;
DROP INDEX IF EXISTS idx_notifications_alert_id;
DROP INDEX IF EXISTS idx_alert_channels_is_active;
DROP INDEX IF EXISTS idx_alert_channels_type;
DROP INDEX IF EXISTS idx_pool_metrics_timestamp;
DROP INDEX IF EXISTS idx_miner_metrics_timestamp;
DROP INDEX IF EXISTS idx_miner_metrics_miner_id;
DROP INDEX IF EXISTS idx_performance_metrics_timestamp;
DROP INDEX IF EXISTS idx_dashboards_is_public;
DROP INDEX IF EXISTS idx_dashboards_created_by;
DROP INDEX IF EXISTS idx_alert_rules_is_active;
DROP INDEX IF EXISTS idx_alerts_created_at;
DROP INDEX IF EXISTS idx_alerts_severity;
DROP INDEX IF EXISTS idx_alerts_status;
DROP INDEX IF EXISTS idx_metrics_type;
DROP INDEX IF EXISTS idx_metrics_timestamp;
DROP INDEX IF EXISTS idx_metrics_name;
DROP INDEX IF EXISTS idx_team_statistics_date;
DROP INDEX IF EXISTS idx_team_statistics_period;
DROP INDEX IF EXISTS idx_team_statistics_team_id;
DROP INDEX IF EXISTS idx_social_shares_milestone;
DROP INDEX IF EXISTS idx_social_shares_platform;
DROP INDEX IF EXISTS idx_social_shares_user_id;
DROP INDEX IF EXISTS idx_competition_participants_user_id;
DROP INDEX IF EXISTS idx_competition_participants_competition_id;
DROP INDEX IF EXISTS idx_competitions_start_time;
DROP INDEX IF EXISTS idx_competitions_status;
DROP INDEX IF EXISTS idx_referrals_status;
DROP INDEX IF EXISTS idx_referrals_referrer_id;
DROP INDEX IF EXISTS idx_referrals_code;
DROP INDEX IF EXISTS idx_team_members_is_active;
DROP INDEX IF EXISTS idx_team_members_user_id;
DROP INDEX IF EXISTS idx_team_members_team_id;
DROP INDEX IF EXISTS idx_teams_is_active;
DROP INDEX IF EXISTS idx_teams_owner_id;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS alert_channels;
DROP TABLE IF EXISTS pool_metrics;
DROP TABLE IF EXISTS miner_metrics;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS dashboards;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS team_statistics;
DROP TABLE IF EXISTS social_shares;
DROP TABLE IF EXISTS competition_participants;
DROP TABLE IF EXISTS competitions;
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;