package bugs

import (
	"context"
	"time"
)

// BugReport represents a user-submitted bug report
type BugReport struct {
	ID               int64      `json:"id"`
	ReportNumber     string     `json:"report_number"`
	UserID           int64      `json:"user_id"`
	Username         string     `json:"username,omitempty"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	StepsToReproduce string     `json:"steps_to_reproduce,omitempty"`
	ExpectedBehavior string     `json:"expected_behavior,omitempty"`
	ActualBehavior   string     `json:"actual_behavior,omitempty"`
	Category         string     `json:"category"`
	Status           string     `json:"status"`
	Priority         string     `json:"priority"`
	AssignedTo       *int64     `json:"assigned_to,omitempty"`
	AssigneeName     string     `json:"assignee_name,omitempty"`
	BrowserInfo      string     `json:"browser_info,omitempty"`
	OSInfo           string     `json:"os_info,omitempty"`
	PageURL          string     `json:"page_url,omitempty"`
	ConsoleErrors    string     `json:"console_errors,omitempty"`
	AttachmentCount  int        `json:"attachment_count"`
	CommentCount     int        `json:"comment_count"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

// BugComment represents a comment on a bug report
type BugComment struct {
	ID             int64     `json:"id"`
	BugReportID    int64     `json:"bug_report_id"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Content        string    `json:"content"`
	IsInternal     bool      `json:"is_internal"`
	IsStatusChange bool      `json:"is_status_change"`
	CreatedAt      time.Time `json:"created_at"`
}

// BugAttachment represents a file attachment on a bug report
type BugAttachment struct {
	ID               int64     `json:"id"`
	BugReportID      int64     `json:"bug_report_id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	FileType         string    `json:"file_type"`
	FileSize         int64     `json:"file_size"`
	IsScreenshot     bool      `json:"is_screenshot"`
	CreatedAt        time.Time `json:"created_at"`
}

// BugFilter defines filtering options for bug queries
type BugFilter struct {
	Status     string
	Priority   string
	Category   string
	AssignedTo *int64
	UserID     *int64
	Search     string
	Limit      int
	Offset     int
}

// Interface Segregation Principle (ISP) - Small, focused interfaces

// BugReader provides read-only access to bug reports
type BugReader interface {
	GetBugByID(ctx context.Context, id int64) (*BugReport, error)
	GetBugByReportNumber(ctx context.Context, reportNumber string) (*BugReport, error)
	ListBugs(ctx context.Context, filter BugFilter) ([]BugReport, int, error)
	GetBugComments(ctx context.Context, bugID int64, includeInternal bool) ([]BugComment, error)
	GetBugAttachments(ctx context.Context, bugID int64) ([]BugAttachment, error)
}

// BugWriter provides write access to bug reports
type BugWriter interface {
	CreateBug(ctx context.Context, bug *BugReport) error
	UpdateBugStatus(ctx context.Context, bugID int64, status string, updatedBy int64) error
	UpdateBugPriority(ctx context.Context, bugID int64, priority string) error
	AssignBug(ctx context.Context, bugID int64, assigneeID *int64, assignedBy int64) error
	DeleteBug(ctx context.Context, bugID int64) error
}

// BugCommenter provides comment functionality
type BugCommenter interface {
	AddComment(ctx context.Context, comment *BugComment) error
	GetCommentByID(ctx context.Context, id int64) (*BugComment, error)
}

// BugAttacher provides attachment functionality
type BugAttacher interface {
	AddAttachment(ctx context.Context, attachment *BugAttachment, data []byte) error
	GetAttachmentData(ctx context.Context, attachmentID int64) ([]byte, error)
}

// BugSubscriber provides subscription functionality
type BugSubscriber interface {
	Subscribe(ctx context.Context, bugID int64, userID int64) error
	Unsubscribe(ctx context.Context, bugID int64, userID int64) error
	GetSubscribers(ctx context.Context, bugID int64) ([]int64, error)
	IsSubscribed(ctx context.Context, bugID int64, userID int64) (bool, error)
}

// BugNotifier provides notification functionality
type BugNotifier interface {
	NotifyNewBug(ctx context.Context, bug *BugReport) error
	NotifyStatusChange(ctx context.Context, bug *BugReport, oldStatus, newStatus string) error
	NotifyNewComment(ctx context.Context, bug *BugReport, comment *BugComment) error
	NotifyAssignment(ctx context.Context, bug *BugReport, assigneeID int64) error
}

// BugService combines all bug-related interfaces for full functionality
type BugService interface {
	BugReader
	BugWriter
	BugCommenter
	BugAttacher
	BugSubscriber
}

// AdminBugService is a convenience interface for admin operations
type AdminBugService interface {
	BugReader
	BugWriter
	BugCommenter
}
