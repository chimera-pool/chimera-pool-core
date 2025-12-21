-- Rollback bug reporting system

DROP TRIGGER IF EXISTS update_bug_comments_updated_at ON bug_comments;
DROP TRIGGER IF EXISTS update_bug_reports_updated_at ON bug_reports;
DROP TRIGGER IF EXISTS set_bug_report_number ON bug_reports;

DROP FUNCTION IF EXISTS generate_bug_report_number();

DROP TABLE IF EXISTS bug_subscribers CASCADE;
DROP TABLE IF EXISTS bug_email_notifications CASCADE;
DROP TABLE IF EXISTS bug_comments CASCADE;
DROP TABLE IF EXISTS bug_attachments CASCADE;
DROP TABLE IF EXISTS bug_reports CASCADE;

DROP SEQUENCE IF EXISTS bug_report_number_seq;
