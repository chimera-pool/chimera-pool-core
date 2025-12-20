-- Rollback: Remove roles and channel tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_channels_updated_at ON channels;
DROP TRIGGER IF EXISTS update_channel_categories_updated_at ON channel_categories;

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS moderation_log;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS channel_categories;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_role;

-- Remove role column from users
ALTER TABLE users DROP COLUMN IF EXISTS role;
