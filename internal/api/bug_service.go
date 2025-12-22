package api

import (
	"database/sql"
	"errors"
	"time"
)

// =============================================================================
// BUG SERVICE IMPLEMENTATIONS
// ISP-compliant services for bug report management
// =============================================================================

// -----------------------------------------------------------------------------
// Bug Data Types
// -----------------------------------------------------------------------------

// BugReportData represents a bug report
type BugReportData struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	StepsToReproduce string    `json:"steps_to_reproduce,omitempty"`
	ExpectedBehavior string    `json:"expected_behavior,omitempty"`
	ActualBehavior   string    `json:"actual_behavior,omitempty"`
	Category         string    `json:"category,omitempty"`
	Status           string    `json:"status"`
	Priority         string    `json:"priority"`
	AssigneeID       *int64    `json:"assignee_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// BugCommentData represents a bug comment
type BugCommentData struct {
	ID        int64     `json:"id"`
	BugID     int64     `json:"bug_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateBugRequest represents bug creation request
type CreateBugRequest struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	StepsToReproduce string `json:"steps_to_reproduce"`
	ExpectedBehavior string `json:"expected_behavior"`
	ActualBehavior   string `json:"actual_behavior"`
	Category         string `json:"category"`
}

// -----------------------------------------------------------------------------
// Bug Service Interfaces (ISP)
// -----------------------------------------------------------------------------

// BugReader reads bug reports
type BugReader interface {
	GetUserBugs(userID int64) ([]*BugReportData, error)
	GetBug(bugID int64) (*BugReportData, error)
	GetBugComments(bugID int64) ([]*BugCommentData, error)
}

// BugWriter writes bug reports
type BugWriter interface {
	CreateBug(userID int64, req *CreateBugRequest) (*BugReportData, error)
	AddComment(bugID, userID int64, content string, isAdmin bool) (*BugCommentData, error)
}

// BugSubscriber manages bug subscriptions
type BugSubscriber interface {
	Subscribe(bugID, userID int64) error
	Unsubscribe(bugID, userID int64) error
	IsSubscribed(bugID, userID int64) bool
}

// AdminBugManager manages bugs (admin only)
type AdminBugManager interface {
	GetAllBugs() ([]*BugReportData, error)
	UpdateStatus(bugID int64, status string) error
	UpdatePriority(bugID int64, priority string) error
	AssignBug(bugID, assigneeID int64) error
	DeleteBug(bugID int64) error
}

// -----------------------------------------------------------------------------
// Bug Service Implementation
// -----------------------------------------------------------------------------

// DBBugService implements bug report operations
type DBBugService struct {
	db *sql.DB
}

// NewDBBugService creates a new bug service
func NewDBBugService(db *sql.DB) *DBBugService {
	return &DBBugService{db: db}
}

// GetUserBugs returns bug reports for a user
func (s *DBBugService) GetUserBugs(userID int64) ([]*BugReportData, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, title, status, priority, created_at FROM bug_reports WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch bug reports")
	}
	defer rows.Close()

	var bugs []*BugReportData
	for rows.Next() {
		var b BugReportData
		err := rows.Scan(&b.ID, &b.UserID, &b.Title, &b.Status, &b.Priority, &b.CreatedAt)
		if err != nil {
			continue
		}
		bugs = append(bugs, &b)
	}

	return bugs, nil
}

// GetBug returns a single bug report
func (s *DBBugService) GetBug(bugID int64) (*BugReportData, error) {
	var b BugReportData
	var stepsToReproduce, expectedBehavior, actualBehavior, category sql.NullString
	var assigneeID sql.NullInt64

	err := s.db.QueryRow(
		"SELECT id, user_id, title, description, steps_to_reproduce, expected_behavior, actual_behavior, category, status, priority, assignee_id, created_at FROM bug_reports WHERE id = $1",
		bugID,
	).Scan(&b.ID, &b.UserID, &b.Title, &b.Description, &stepsToReproduce, &expectedBehavior, &actualBehavior, &category, &b.Status, &b.Priority, &assigneeID, &b.CreatedAt)

	if err != nil {
		return nil, errors.New("bug report not found")
	}

	if stepsToReproduce.Valid {
		b.StepsToReproduce = stepsToReproduce.String
	}
	if expectedBehavior.Valid {
		b.ExpectedBehavior = expectedBehavior.String
	}
	if actualBehavior.Valid {
		b.ActualBehavior = actualBehavior.String
	}
	if category.Valid {
		b.Category = category.String
	}
	if assigneeID.Valid {
		b.AssigneeID = &assigneeID.Int64
	}

	return &b, nil
}

// GetBugComments returns comments for a bug
func (s *DBBugService) GetBugComments(bugID int64) ([]*BugCommentData, error) {
	rows, err := s.db.Query(
		"SELECT c.id, c.bug_id, c.user_id, u.username, c.content, COALESCE(c.is_admin, false), c.created_at FROM bug_comments c JOIN users u ON c.user_id = u.id WHERE c.bug_id = $1 ORDER BY c.created_at",
		bugID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch comments")
	}
	defer rows.Close()

	var comments []*BugCommentData
	for rows.Next() {
		var c BugCommentData
		err := rows.Scan(&c.ID, &c.BugID, &c.UserID, &c.Username, &c.Content, &c.IsAdmin, &c.CreatedAt)
		if err != nil {
			continue
		}
		comments = append(comments, &c)
	}

	return comments, nil
}

// CreateBug creates a new bug report
func (s *DBBugService) CreateBug(userID int64, req *CreateBugRequest) (*BugReportData, error) {
	var id int64
	var createdAt time.Time

	err := s.db.QueryRow(
		"INSERT INTO bug_reports (user_id, title, description, steps_to_reproduce, expected_behavior, actual_behavior, category, status, priority) VALUES ($1, $2, $3, $4, $5, $6, $7, 'open', 'medium') RETURNING id, created_at",
		userID, req.Title, req.Description, req.StepsToReproduce, req.ExpectedBehavior, req.ActualBehavior, req.Category,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, errors.New("failed to create bug report")
	}

	return &BugReportData{
		ID:               id,
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		StepsToReproduce: req.StepsToReproduce,
		ExpectedBehavior: req.ExpectedBehavior,
		ActualBehavior:   req.ActualBehavior,
		Category:         req.Category,
		Status:           "open",
		Priority:         "medium",
		CreatedAt:        createdAt,
	}, nil
}

// AddComment adds a comment to a bug
func (s *DBBugService) AddComment(bugID, userID int64, content string, isAdmin bool) (*BugCommentData, error) {
	var id int64
	var createdAt time.Time

	err := s.db.QueryRow(
		"INSERT INTO bug_comments (bug_id, user_id, content, is_admin) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		bugID, userID, content, isAdmin,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, errors.New("failed to add comment")
	}

	var username string
	s.db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)

	return &BugCommentData{
		ID:        id,
		BugID:     bugID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		IsAdmin:   isAdmin,
		CreatedAt: createdAt,
	}, nil
}

// Subscribe subscribes to a bug
func (s *DBBugService) Subscribe(bugID, userID int64) error {
	_, err := s.db.Exec(
		"INSERT INTO bug_subscriptions (bug_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		bugID, userID,
	)
	return err
}

// Unsubscribe unsubscribes from a bug
func (s *DBBugService) Unsubscribe(bugID, userID int64) error {
	_, err := s.db.Exec("DELETE FROM bug_subscriptions WHERE bug_id = $1 AND user_id = $2", bugID, userID)
	return err
}

// IsSubscribed checks if user is subscribed
func (s *DBBugService) IsSubscribed(bugID, userID int64) bool {
	var exists bool
	s.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM bug_subscriptions WHERE bug_id = $1 AND user_id = $2)",
		bugID, userID,
	).Scan(&exists)
	return exists
}

// GetAllBugs returns all bug reports (admin)
func (s *DBBugService) GetAllBugs() ([]*BugReportData, error) {
	rows, err := s.db.Query(
		"SELECT id, user_id, title, status, priority, created_at FROM bug_reports ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, errors.New("failed to fetch bug reports")
	}
	defer rows.Close()

	var bugs []*BugReportData
	for rows.Next() {
		var b BugReportData
		err := rows.Scan(&b.ID, &b.UserID, &b.Title, &b.Status, &b.Priority, &b.CreatedAt)
		if err != nil {
			continue
		}
		bugs = append(bugs, &b)
	}

	return bugs, nil
}

// UpdateStatus updates a bug's status
func (s *DBBugService) UpdateStatus(bugID int64, status string) error {
	_, err := s.db.Exec("UPDATE bug_reports SET status = $1, updated_at = NOW() WHERE id = $2", status, bugID)
	return err
}

// UpdatePriority updates a bug's priority
func (s *DBBugService) UpdatePriority(bugID int64, priority string) error {
	_, err := s.db.Exec("UPDATE bug_reports SET priority = $1, updated_at = NOW() WHERE id = $2", priority, bugID)
	return err
}

// AssignBug assigns a bug to a user
func (s *DBBugService) AssignBug(bugID, assigneeID int64) error {
	_, err := s.db.Exec("UPDATE bug_reports SET assignee_id = $1, updated_at = NOW() WHERE id = $2", assigneeID, bugID)
	return err
}

// DeleteBug deletes a bug report
func (s *DBBugService) DeleteBug(bugID int64) error {
	_, err := s.db.Exec("DELETE FROM bug_reports WHERE id = $1", bugID)
	return err
}

// =============================================================================
// BUG SERVICE FACTORY
// =============================================================================

// BugServices holds all bug-related service implementations
type BugServices struct {
	Bug *DBBugService
}

// NewBugServices creates all bug services
func NewBugServices(db *sql.DB) *BugServices {
	return &BugServices{
		Bug: NewDBBugService(db),
	}
}
