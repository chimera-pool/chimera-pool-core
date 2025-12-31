-- ============================================================================
-- CHAT REACTIONS SYSTEM
-- Allows users to react to chat messages with emojis
-- ============================================================================

-- Predefined reaction types for mining/crypto community
CREATE TABLE IF NOT EXISTS reaction_types (
    id SERIAL PRIMARY KEY,
    emoji VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    category VARCHAR(20) DEFAULT 'general',
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true
);

-- Message reactions table
CREATE TABLE IF NOT EXISTS message_reactions (
    id SERIAL PRIMARY KEY,
    message_id INTEGER NOT NULL REFERENCES channel_messages(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction_type_id INTEGER NOT NULL REFERENCES reaction_types(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(message_id, user_id, reaction_type_id)  -- One reaction type per user per message
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_message_reactions_message_id ON message_reactions(message_id);
CREATE INDEX IF NOT EXISTS idx_message_reactions_user_id ON message_reactions(user_id);
CREATE INDEX IF NOT EXISTS idx_message_reactions_type ON message_reactions(reaction_type_id);

-- Insert predefined reaction types for mining community
INSERT INTO reaction_types (emoji, name, category, sort_order) VALUES
    -- Mining reactions
    ('⛏️', 'mining', 'mining', 1),
    ('💎', 'diamond_hands', 'mining', 2),
    ('🚀', 'to_the_moon', 'mining', 3),
    ('🔥', 'fire', 'mining', 4),
    ('⚡', 'lightning', 'mining', 5),
    -- Approval reactions
    ('👍', 'thumbs_up', 'approval', 10),
    ('👎', 'thumbs_down', 'approval', 11),
    ('❤️', 'heart', 'approval', 12),
    ('🎉', 'celebration', 'approval', 13),
    -- Crypto specific
    ('🐂', 'bull', 'crypto', 20),
    ('🐻', 'bear', 'crypto', 21),
    ('💰', 'money_bag', 'crypto', 22),
    ('📈', 'chart_up', 'crypto', 23),
    ('📉', 'chart_down', 'crypto', 24),
    -- Communication
    ('👀', 'eyes', 'communication', 30),
    ('🤔', 'thinking', 'communication', 31),
    ('💯', 'hundred', 'communication', 32)
ON CONFLICT (emoji) DO NOTHING;

-- View for aggregated reactions per message
CREATE OR REPLACE VIEW message_reaction_counts AS
SELECT 
    mr.message_id,
    rt.emoji,
    rt.name as reaction_name,
    COUNT(*) as count,
    ARRAY_AGG(u.username ORDER BY mr.created_at) as users
FROM message_reactions mr
JOIN reaction_types rt ON mr.reaction_type_id = rt.id
JOIN users u ON mr.user_id = u.id
GROUP BY mr.message_id, rt.id, rt.emoji, rt.name;

-- Function to toggle a reaction (add if not exists, remove if exists)
CREATE OR REPLACE FUNCTION toggle_message_reaction(
    p_message_id INTEGER,
    p_user_id INTEGER,
    p_emoji VARCHAR(10)
) RETURNS TABLE(action VARCHAR, reaction_count INTEGER) AS $$
DECLARE
    v_reaction_type_id INTEGER;
    v_existing_id INTEGER;
BEGIN
    -- Get reaction type id
    SELECT id INTO v_reaction_type_id FROM reaction_types WHERE emoji = p_emoji AND is_active = true;
    IF v_reaction_type_id IS NULL THEN
        RAISE EXCEPTION 'Invalid reaction emoji: %', p_emoji;
    END IF;
    
    -- Check if reaction exists
    SELECT id INTO v_existing_id FROM message_reactions 
    WHERE message_id = p_message_id AND user_id = p_user_id AND reaction_type_id = v_reaction_type_id;
    
    IF v_existing_id IS NOT NULL THEN
        -- Remove existing reaction
        DELETE FROM message_reactions WHERE id = v_existing_id;
        action := 'removed';
    ELSE
        -- Add new reaction
        INSERT INTO message_reactions (message_id, user_id, reaction_type_id)
        VALUES (p_message_id, p_user_id, v_reaction_type_id);
        action := 'added';
    END IF;
    
    -- Get updated count
    SELECT COUNT(*) INTO reaction_count 
    FROM message_reactions mr
    JOIN reaction_types rt ON mr.reaction_type_id = rt.id
    WHERE mr.message_id = p_message_id AND rt.emoji = p_emoji;
    
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE reaction_types IS 'Predefined reaction emojis for chat messages';
COMMENT ON TABLE message_reactions IS 'User reactions on chat messages';
COMMENT ON FUNCTION toggle_message_reaction IS 'Adds or removes a reaction from a message';
