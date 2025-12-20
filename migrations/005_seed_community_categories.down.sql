-- Rollback: Remove seeded community categories and channels
-- Note: This will delete all default channels and categories

DELETE FROM channels WHERE name IN (
    'pool-updates', 'general-chat', 'introductions', 
    'getting-started', 'miner-config', 'help-desk',
    'asic-miners', 'gpu-mining', 'random'
);

DELETE FROM channel_categories WHERE name IN (
    'Announcements', 'General', 'Setup & Configuration',
    'Troubleshooting', 'Mining Hardware', 'Off-Topic'
);
