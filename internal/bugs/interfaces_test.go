package bugs

import (
	"context"
	"testing"
	"time"
)

// Mock implementations for testing

type MockBugReader struct {
	bugs     map[int64]*BugReport
	comments map[int64][]BugComment
}

func NewMockBugReader() *MockBugReader {
	return &MockBugReader{
		bugs:     make(map[int64]*BugReport),
		comments: make(map[int64][]BugComment),
	}
}

func (m *MockBugReader) GetBugByID(ctx context.Context, id int64) (*BugReport, error) {
	if bug, ok := m.bugs[id]; ok {
		return bug, nil
	}
	return nil, nil
}

func (m *MockBugReader) GetBugByReportNumber(ctx context.Context, reportNumber string) (*BugReport, error) {
	for _, bug := range m.bugs {
		if bug.ReportNumber == reportNumber {
			return bug, nil
		}
	}
	return nil, nil
}

func (m *MockBugReader) ListBugs(ctx context.Context, filter BugFilter) ([]BugReport, int, error) {
	var result []BugReport
	for _, bug := range m.bugs {
		if filter.Status != "" && bug.Status != filter.Status {
			continue
		}
		if filter.Priority != "" && bug.Priority != filter.Priority {
			continue
		}
		if filter.Category != "" && bug.Category != filter.Category {
			continue
		}
		result = append(result, *bug)
	}
	return result, len(result), nil
}

func (m *MockBugReader) GetBugComments(ctx context.Context, bugID int64, includeInternal bool) ([]BugComment, error) {
	comments := m.comments[bugID]
	if !includeInternal {
		var filtered []BugComment
		for _, c := range comments {
			if !c.IsInternal {
				filtered = append(filtered, c)
			}
		}
		return filtered, nil
	}
	return comments, nil
}

func (m *MockBugReader) GetBugAttachments(ctx context.Context, bugID int64) ([]BugAttachment, error) {
	return nil, nil
}

// Test BugReport struct
func TestBugReport_Fields(t *testing.T) {
	now := time.Now()
	bug := BugReport{
		ID:               1,
		ReportNumber:     "BUG-000001",
		UserID:           100,
		Username:         "testuser",
		Title:            "Test Bug",
		Description:      "This is a test bug description",
		StepsToReproduce: "1. Do this\n2. Do that",
		ExpectedBehavior: "Should work",
		ActualBehavior:   "Doesn't work",
		Category:         "ui",
		Status:           "open",
		Priority:         "medium",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if bug.ID != 1 {
		t.Errorf("Expected ID 1, got %d", bug.ID)
	}
	if bug.ReportNumber != "BUG-000001" {
		t.Errorf("Expected report number BUG-000001, got %s", bug.ReportNumber)
	}
	if bug.Status != "open" {
		t.Errorf("Expected status open, got %s", bug.Status)
	}
	if bug.Priority != "medium" {
		t.Errorf("Expected priority medium, got %s", bug.Priority)
	}
}

// Test BugFilter
func TestBugFilter_Defaults(t *testing.T) {
	filter := BugFilter{}

	if filter.Status != "" {
		t.Errorf("Expected empty status, got %s", filter.Status)
	}
	if filter.Limit != 0 {
		t.Errorf("Expected 0 limit, got %d", filter.Limit)
	}
}

// Test MockBugReader.ListBugs with filters
func TestMockBugReader_ListBugs_StatusFilter(t *testing.T) {
	reader := NewMockBugReader()
	now := time.Now()

	reader.bugs[1] = &BugReport{ID: 1, Status: "open", Priority: "high", CreatedAt: now}
	reader.bugs[2] = &BugReport{ID: 2, Status: "resolved", Priority: "low", CreatedAt: now}
	reader.bugs[3] = &BugReport{ID: 3, Status: "open", Priority: "critical", CreatedAt: now}

	ctx := context.Background()

	// Filter by status
	bugs, count, err := reader.ListBugs(ctx, BugFilter{Status: "open"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 open bugs, got %d", count)
	}
	for _, bug := range bugs {
		if bug.Status != "open" {
			t.Errorf("Expected all bugs to have status open, got %s", bug.Status)
		}
	}
}

func TestMockBugReader_ListBugs_PriorityFilter(t *testing.T) {
	reader := NewMockBugReader()
	now := time.Now()

	reader.bugs[1] = &BugReport{ID: 1, Status: "open", Priority: "high", CreatedAt: now}
	reader.bugs[2] = &BugReport{ID: 2, Status: "resolved", Priority: "low", CreatedAt: now}
	reader.bugs[3] = &BugReport{ID: 3, Status: "open", Priority: "critical", CreatedAt: now}

	ctx := context.Background()

	// Filter by priority
	bugs, count, err := reader.ListBugs(ctx, BugFilter{Priority: "critical"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 critical bug, got %d", count)
	}
	if len(bugs) > 0 && bugs[0].Priority != "critical" {
		t.Errorf("Expected priority critical, got %s", bugs[0].Priority)
	}
}

func TestMockBugReader_GetBugComments_ExcludesInternal(t *testing.T) {
	reader := NewMockBugReader()
	now := time.Now()

	reader.comments[1] = []BugComment{
		{ID: 1, BugReportID: 1, Content: "Public comment", IsInternal: false, CreatedAt: now},
		{ID: 2, BugReportID: 1, Content: "Internal note", IsInternal: true, CreatedAt: now},
		{ID: 3, BugReportID: 1, Content: "Another public", IsInternal: false, CreatedAt: now},
	}

	ctx := context.Background()

	// Without internal comments
	comments, err := reader.GetBugComments(ctx, 1, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("Expected 2 public comments, got %d", len(comments))
	}
	for _, c := range comments {
		if c.IsInternal {
			t.Error("Expected no internal comments when includeInternal=false")
		}
	}

	// With internal comments
	allComments, err := reader.GetBugComments(ctx, 1, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(allComments) != 3 {
		t.Errorf("Expected 3 total comments, got %d", len(allComments))
	}
}

// Test valid status values
func TestBugReport_ValidStatuses(t *testing.T) {
	validStatuses := []string{"open", "in_progress", "resolved", "closed", "wont_fix"}

	for _, status := range validStatuses {
		bug := BugReport{Status: status}
		if bug.Status != status {
			t.Errorf("Status %s should be valid", status)
		}
	}
}

// Test valid priority values
func TestBugReport_ValidPriorities(t *testing.T) {
	validPriorities := []string{"low", "medium", "high", "critical"}

	for _, priority := range validPriorities {
		bug := BugReport{Priority: priority}
		if bug.Priority != priority {
			t.Errorf("Priority %s should be valid", priority)
		}
	}
}

// Test valid category values
func TestBugReport_ValidCategories(t *testing.T) {
	validCategories := []string{"ui", "performance", "security", "feature_request", "crash", "other"}

	for _, category := range validCategories {
		bug := BugReport{Category: category}
		if bug.Category != category {
			t.Errorf("Category %s should be valid", category)
		}
	}
}

// Test BugComment struct
func TestBugComment_Fields(t *testing.T) {
	now := time.Now()
	comment := BugComment{
		ID:             1,
		BugReportID:    100,
		UserID:         50,
		Username:       "admin",
		Content:        "Working on this issue",
		IsInternal:     true,
		IsStatusChange: false,
		CreatedAt:      now,
	}

	if comment.ID != 1 {
		t.Errorf("Expected ID 1, got %d", comment.ID)
	}
	if !comment.IsInternal {
		t.Error("Expected IsInternal to be true")
	}
	if comment.IsStatusChange {
		t.Error("Expected IsStatusChange to be false")
	}
}

// Test interface compliance
func TestInterfaceCompliance(t *testing.T) {
	// Ensure MockBugReader implements BugReader
	var _ BugReader = (*MockBugReader)(nil)
}
