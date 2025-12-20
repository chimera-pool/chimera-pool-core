-- Migration: Seed default community categories for mining pool
-- These categories are essential for community features to work

-- Insert default categories only if they don't already exist
-- Using the first admin user as creator, or first user if no admin

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Announcements',
    'Official pool announcements and news',
    1,
    COALESCE(
        (SELECT id FROM users WHERE is_admin = true LIMIT 1),
        (SELECT id FROM users ORDER BY id LIMIT 1),
        1
    )
WHERE NOT EXISTS (SELECT 1 FROM channel_categories WHERE name = 'Announcements');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'General',
    'General discussion about the pool',
    2,
    COALESCE(
        (SELECT id FROM users WHERE is_admin = true LIMIT 1),
        (SELECT id FROM users ORDER BY id LIMIT 1),
        1
    )
WHERE NOT EXISTS (SELECT 1 FROM channel_categories WHERE name = 'General');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Setup & Configuration',
    'Help with miner setup and configuration',
    3,
    COALESCE(
        (SELECT id FROM users WHERE is_admin = true LIMIT 1),
        (SELECT id FROM users ORDER BY id LIMIT 1),
        1
    )
WHERE NOT EXISTS (SELECT 1 FROM channel_categories WHERE name = 'Setup & Configuration');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Troubleshooting',
    'Report issues and get help solving problems',
    4,
    COALESCE(
        (SELECT id FROM users WHERE is_admin = true LIMIT 1),
        (SELECT id FROM users ORDER BY id LIMIT 1),
        1
    )
WHERE NOT EXISTS (SELECT 1 FROM channel_categories WHERE name = 'Troubleshooting');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Mining Hardware',
    'Discuss mining hardware, ASICs, GPUs, and optimization',
    5,
    COALESCE(
        (SELECT id FROM users WHERE is_admin = true LIMIT 1),
        (SELECT id FROM users ORDER BY id LIMIT 1),
        1
    )
WHERE NOT EXISTS (SELECT 1 FROM channel_categories WHERE name = 'Mining Hardware');

INSERT INTO channel_categories (id, name, description, position, created_by)
SELECT 
    gen_random_uuid(),
    'Off-Topic',
    'Casual conversation and community chat',
    6,
    COALESCE(
        (SELECT id FROM users WHERE is_admin = true LIMIT 1),
        (SELECT id FROM users ORDER BY id LIMIT 1),
        1
    )
WHERE NOT EXISTS (SELECT 1 FROM channel_categories WHERE name = 'Off-Topic');

-- Create default channels for each category
-- Announcements category
INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'pool-updates',
    'Pool status updates and maintenance notices',
    'announcement',
    1,
    false,
    true,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Announcements'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'pool-updates' AND c.category_id = cc.id);

-- General category
INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'general-chat',
    'General discussion about mining and the pool',
    'text',
    1,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'General'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'general-chat' AND c.category_id = cc.id);

INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'introductions',
    'Introduce yourself to the community',
    'text',
    2,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'General'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'introductions' AND c.category_id = cc.id);

-- Setup & Configuration category
INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'getting-started',
    'Help for new miners getting set up',
    'text',
    1,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Setup & Configuration'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'getting-started' AND c.category_id = cc.id);

INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'miner-config',
    'Share and discuss miner configurations',
    'text',
    2,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Setup & Configuration'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'miner-config' AND c.category_id = cc.id);

-- Troubleshooting category
INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'help-desk',
    'Get help with pool-related issues',
    'text',
    1,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Troubleshooting'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'help-desk' AND c.category_id = cc.id);

-- Mining Hardware category
INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'asic-miners',
    'Discussion about ASIC miners including BlockDAG X30/X100',
    'text',
    1,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Mining Hardware'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'asic-miners' AND c.category_id = cc.id);

INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'gpu-mining',
    'GPU mining discussion and optimization',
    'text',
    2,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Mining Hardware'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'gpu-mining' AND c.category_id = cc.id);

-- Off-Topic category
INSERT INTO channels (id, category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
SELECT 
    gen_random_uuid(),
    cc.id,
    'random',
    'Random off-topic conversations',
    'text',
    1,
    false,
    false,
    COALESCE((SELECT id FROM users WHERE is_admin = true LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
FROM channel_categories cc
WHERE cc.name = 'Off-Topic'
AND NOT EXISTS (SELECT 1 FROM channels c WHERE c.name = 'random' AND c.category_id = cc.id);
