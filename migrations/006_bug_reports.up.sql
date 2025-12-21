-- Bug Reporting System for Chimera Pool
-- Migration 006: Bug reports with attachments, comments, and email tracking

-- Bug report status enum values
-- 'open', 'in_progress', 'resolved', 'closed', 'wont_fix'

-- Bug report priority enum values
-- 'low', 'medium', 'high', 'critical'

-- Bug report category enum values
-- 'ui', 'performance', 'security', 'feature_request', 'crash', 'other'

-- Main bug reports table
CREATE TABLE bug_reports (
    id BIGSERIAL PRIMARY KEY,
    report_number VARCHAR(20) UNIQUE NOT NULL, -- e.g., BUG-000001
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    steps_to_reproduce TEXT,
    expected_behavior TEXT,
    actual_behavior TEXT,
    category VARCHAR(50) DEFAULT 'other' CHECK (category IN ('ui', 'performance', 'security', 'feature_request', 'crash', 'other')),
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed', 'wont_fix')),
    browser_info TEXT,
    os_info TEXT,
    page_url TEXT,
    console_errors TEXT,
    assigned_to BIGINT REFERENCES users(id) ON DELETE SET NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Bug report attachments (screenshots, files)
CREATE TABLE bug_attachments (
    id BIGSERIAL PRIMARY KEY,
    bug_report_id BIGINT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    file_data BYTEA, -- Store small files directly, or use file path for large files
    file_path TEXT, -- Alternative: path to file on disk/S3
    is_screenshot BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Bug report comments (conversation thread)
CREATE TABLE bug_comments (
    id BIGSERIAL PRIMARY KEY,
    bug_report_id BIGINT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_internal BOOLEAN DEFAULT false, -- Admin-only internal notes
    is_status_change BOOLEAN DEFAULT false, -- System-generated status change comment
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Email notifications tracking
CREATE TABLE bug_email_notifications (
    id BIGSERIAL PRIMARY KEY,
    bug_report_id BIGINT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_type VARCHAR(50) NOT NULL CHECK (email_type IN ('new_report', 'status_change', 'new_comment', 'assigned', 'resolved')),
    email_address VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE,
    is_sent BOOLEAN DEFAULT false,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Bug report subscribers (users who want updates)
CREATE TABLE bug_subscribers (
    id BIGSERIAL PRIMARY KEY,
    bug_report_id BIGINT NOT NULL REFERENCES bug_reports(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(bug_report_id, user_id)
);

-- Sequence for bug report numbers
CREATE SEQUENCE bug_report_number_seq START 1;

-- Function to generate bug report number
CREATE OR REPLACE FUNCTION generate_bug_report_number()
RETURNS TRIGGER AS $$
BEGIN
    NEW.report_number := 'BUG-' || LPAD(nextval('bug_report_number_seq')::TEXT, 6, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-generate report number
CREATE TRIGGER set_bug_report_number
    BEFORE INSERT ON bug_reports
    FOR EACH ROW
    EXECUTE FUNCTION generate_bug_report_number();

-- Create indexes for performance
CREATE INDEX idx_bug_reports_user_id ON bug_reports(user_id);
CREATE INDEX idx_bug_reports_status ON bug_reports(status);
CREATE INDEX idx_bug_reports_priority ON bug_reports(priority);
CREATE INDEX idx_bug_reports_category ON bug_reports(category);
CREATE INDEX idx_bug_reports_assigned_to ON bug_reports(assigned_to);
CREATE INDEX idx_bug_reports_created_at ON bug_reports(created_at);
CREATE INDEX idx_bug_reports_report_number ON bug_reports(report_number);

CREATE INDEX idx_bug_attachments_bug_report_id ON bug_attachments(bug_report_id);
CREATE INDEX idx_bug_comments_bug_report_id ON bug_comments(bug_report_id);
CREATE INDEX idx_bug_comments_user_id ON bug_comments(user_id);
CREATE INDEX idx_bug_email_notifications_bug_report_id ON bug_email_notifications(bug_report_id);
CREATE INDEX idx_bug_email_notifications_is_sent ON bug_email_notifications(is_sent);
CREATE INDEX idx_bug_subscribers_bug_report_id ON bug_subscribers(bug_report_id);

-- Triggers for updated_at
CREATE TRIGGER update_bug_reports_updated_at 
    BEFORE UPDATE ON bug_reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bug_comments_updated_at 
    BEFORE UPDATE ON bug_comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
