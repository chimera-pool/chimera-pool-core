-- Rollback chat reactions

DROP FUNCTION IF EXISTS toggle_message_reaction(INTEGER, INTEGER, VARCHAR);
DROP VIEW IF EXISTS message_reaction_counts;
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS reaction_types;
