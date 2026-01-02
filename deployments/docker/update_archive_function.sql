-- Update archive_old_shares to be more aggressive (24 hours instead of 30 days)
CREATE OR REPLACE FUNCTION archive_old_shares()
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_archived_count INTEGER;
BEGIN
    -- Move shares older than 24 hours to archive (more aggressive to prevent slowdowns)
    WITH moved AS (
        DELETE FROM shares
        WHERE timestamp < NOW() - INTERVAL '24 hours'
        RETURNING *
    )
    INSERT INTO shares_archive
    SELECT * FROM moved;

    GET DIAGNOSTICS v_archived_count = ROW_COUNT;
    
    IF v_archived_count > 0 THEN
        INSERT INTO db_maintenance_log (operation, table_name, rows_affected, completed_at, notes)
        VALUES ('ARCHIVE', 'shares', v_archived_count, NOW(), 'Archived shares older than 24 hours');
    END IF;

    RETURN v_archived_count;
END;
$$;
