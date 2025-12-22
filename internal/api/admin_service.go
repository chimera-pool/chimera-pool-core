package api

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// =============================================================================
// ADMIN SERVICE IMPLEMENTATIONS
// ISP-compliant services for admin operations
// =============================================================================

// -----------------------------------------------------------------------------
// Admin Data Types
// -----------------------------------------------------------------------------

// AdminUserData represents a user in admin context
type AdminUserData struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminStatsData represents admin dashboard statistics
type AdminStatsData struct {
	TotalUsers    int64   `json:"total_users"`
	TotalMiners   int64   `json:"total_miners"`
	TotalBlocks   int64   `json:"total_blocks"`
	TotalHashrate float64 `json:"total_hashrate"`
	TotalEarnings float64 `json:"total_earnings"`
}

// SettingData represents a pool setting
type SettingData struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReportData represents a content report
type ReportData struct {
	ID          int64     `json:"id"`
	ReporterID  int64     `json:"reporter_id"`
	ContentType string    `json:"content_type"`
	ContentID   int64     `json:"content_id"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// -----------------------------------------------------------------------------
// Admin Service Interfaces (ISP)
// -----------------------------------------------------------------------------

// AdminStatsReader reads admin statistics
type AdminStatsReader interface {
	GetStats() (*AdminStatsData, error)
	GetHashrateHistory(period string) ([]map[string]interface{}, error)
}

// AdminUserManager manages users
type AdminUserManager interface {
	ListUsers(page, limit int, search string) ([]*AdminUserData, int64, error)
	GetUser(userID int64) (*AdminUserData, error)
	UpdateUser(userID int64, isAdmin, isActive *bool) error
	DeleteUser(userID int64) error
}

// AdminSettingsManager manages pool settings
type AdminSettingsManager interface {
	GetSettings() (map[string]*SettingData, error)
	UpdateSettings(settings map[string]string) error
}

// AdminModerationManager manages content moderation
type AdminModerationManager interface {
	BanUser(userID int64) error
	UnbanUser(userID int64) error
	MuteUser(userID int64) error
	UnmuteUser(userID int64) error
	GetReports() ([]*ReportData, error)
	ReviewReport(reportID int64, status string) error
}

// -----------------------------------------------------------------------------
// Admin Stats Service Implementation
// -----------------------------------------------------------------------------

// DBAdminStatsService implements admin stats operations
type DBAdminStatsService struct {
	db *sql.DB
}

// NewDBAdminStatsService creates a new admin stats service
func NewDBAdminStatsService(db *sql.DB) *DBAdminStatsService {
	return &DBAdminStatsService{db: db}
}

// GetStats returns admin dashboard statistics
func (s *DBAdminStatsService) GetStats() (*AdminStatsData, error) {
	var stats AdminStatsData

	s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	s.db.QueryRow("SELECT COUNT(*) FROM miners WHERE is_active = true").Scan(&stats.TotalMiners)
	s.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&stats.TotalBlocks)
	s.db.QueryRow("SELECT COALESCE(SUM(hashrate), 0) FROM miners WHERE is_active = true").Scan(&stats.TotalHashrate)
	s.db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE status = 'completed'").Scan(&stats.TotalEarnings)

	return &stats, nil
}

// GetHashrateHistory returns hashrate history
func (s *DBAdminStatsService) GetHashrateHistory(period string) ([]map[string]interface{}, error) {
	interval := getPeriodIntervalSQL(period)

	rows, err := s.db.Query(
		fmt.Sprintf("SELECT timestamp, hashrate FROM pool_hashrate_history WHERE timestamp > NOW() - INTERVAL '%s' ORDER BY timestamp", interval),
	)
	if err != nil {
		return nil, errors.New("failed to fetch hashrate history")
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var timestamp time.Time
		var hashrate float64
		if err := rows.Scan(&timestamp, &hashrate); err != nil {
			continue
		}
		history = append(history, map[string]interface{}{
			"timestamp": timestamp,
			"hashrate":  hashrate,
		})
	}

	return history, nil
}

// -----------------------------------------------------------------------------
// Admin User Service Implementation
// -----------------------------------------------------------------------------

// DBAdminUserService implements admin user management
type DBAdminUserService struct {
	db *sql.DB
}

// NewDBAdminUserService creates a new admin user service
func NewDBAdminUserService(db *sql.DB) *DBAdminUserService {
	return &DBAdminUserService{db: db}
}

// ListUsers returns paginated user list
func (s *DBAdminUserService) ListUsers(page, limit int, search string) ([]*AdminUserData, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var rows *sql.Rows
	var err error
	var totalCount int64

	if search != "" {
		searchPattern := "%" + search + "%"
		s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username ILIKE $1 OR email ILIKE $1", searchPattern).Scan(&totalCount)
		rows, err = s.db.Query(
			"SELECT id, username, email, is_admin, is_active, COALESCE(role, 'user'), created_at FROM users WHERE username ILIKE $1 OR email ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
			searchPattern, limit, offset,
		)
	} else {
		s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalCount)
		rows, err = s.db.Query(
			"SELECT id, username, email, is_admin, is_active, COALESCE(role, 'user'), created_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2",
			limit, offset,
		)
	}

	if err != nil {
		return nil, 0, errors.New("failed to fetch users")
	}
	defer rows.Close()

	var users []*AdminUserData
	for rows.Next() {
		var u AdminUserData
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.IsActive, &u.Role, &u.CreatedAt)
		if err != nil {
			continue
		}
		users = append(users, &u)
	}

	return users, totalCount, nil
}

// GetUser returns a single user
func (s *DBAdminUserService) GetUser(userID int64) (*AdminUserData, error) {
	var u AdminUserData
	err := s.db.QueryRow(
		"SELECT id, username, email, is_admin, is_active, COALESCE(role, 'user'), created_at FROM users WHERE id = $1",
		userID,
	).Scan(&u.ID, &u.Username, &u.Email, &u.IsAdmin, &u.IsActive, &u.Role, &u.CreatedAt)

	if err != nil {
		return nil, errors.New("user not found")
	}

	return &u, nil
}

// UpdateUser updates a user's admin status
func (s *DBAdminUserService) UpdateUser(userID int64, isAdmin, isActive *bool) error {
	if isAdmin != nil {
		_, err := s.db.Exec("UPDATE users SET is_admin = $1 WHERE id = $2", *isAdmin, userID)
		if err != nil {
			return errors.New("failed to update admin status")
		}
	}
	if isActive != nil {
		_, err := s.db.Exec("UPDATE users SET is_active = $1 WHERE id = $2", *isActive, userID)
		if err != nil {
			return errors.New("failed to update active status")
		}
	}
	return nil
}

// DeleteUser deletes a user
func (s *DBAdminUserService) DeleteUser(userID int64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		return errors.New("failed to delete user")
	}
	return nil
}

// -----------------------------------------------------------------------------
// Admin Settings Service Implementation
// -----------------------------------------------------------------------------

// DBAdminSettingsService implements pool settings management
type DBAdminSettingsService struct {
	db *sql.DB
}

// NewDBAdminSettingsService creates a new admin settings service
func NewDBAdminSettingsService(db *sql.DB) *DBAdminSettingsService {
	return &DBAdminSettingsService{db: db}
}

// GetSettings returns all pool settings
func (s *DBAdminSettingsService) GetSettings() (map[string]*SettingData, error) {
	rows, err := s.db.Query("SELECT key, value, COALESCE(description, ''), updated_at FROM pool_settings")
	if err != nil {
		return nil, errors.New("failed to fetch settings")
	}
	defer rows.Close()

	settings := make(map[string]*SettingData)
	for rows.Next() {
		var sd SettingData
		err := rows.Scan(&sd.Key, &sd.Value, &sd.Description, &sd.UpdatedAt)
		if err != nil {
			continue
		}
		settings[sd.Key] = &sd
	}

	return settings, nil
}

// UpdateSettings updates pool settings
func (s *DBAdminSettingsService) UpdateSettings(settings map[string]string) error {
	for key, value := range settings {
		if value != "" {
			_, err := s.db.Exec(
				"INSERT INTO pool_settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()",
				key, value,
			)
			if err != nil {
				return fmt.Errorf("failed to update setting: %s", key)
			}
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Admin Moderation Service Implementation
// -----------------------------------------------------------------------------

// DBAdminModerationService implements content moderation
type DBAdminModerationService struct {
	db *sql.DB
}

// NewDBAdminModerationService creates a new admin moderation service
func NewDBAdminModerationService(db *sql.DB) *DBAdminModerationService {
	return &DBAdminModerationService{db: db}
}

// BanUser bans a user
func (s *DBAdminModerationService) BanUser(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET is_banned = true WHERE id = $1", userID)
	return err
}

// UnbanUser unbans a user
func (s *DBAdminModerationService) UnbanUser(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET is_banned = false WHERE id = $1", userID)
	return err
}

// MuteUser mutes a user
func (s *DBAdminModerationService) MuteUser(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET is_muted = true WHERE id = $1", userID)
	return err
}

// UnmuteUser unmutes a user
func (s *DBAdminModerationService) UnmuteUser(userID int64) error {
	_, err := s.db.Exec("UPDATE users SET is_muted = false WHERE id = $1", userID)
	return err
}

// GetReports returns all content reports
func (s *DBAdminModerationService) GetReports() ([]*ReportData, error) {
	rows, err := s.db.Query(
		"SELECT id, reporter_id, content_type, content_id, reason, status, created_at FROM reports ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, errors.New("failed to fetch reports")
	}
	defer rows.Close()

	var reports []*ReportData
	for rows.Next() {
		var r ReportData
		err := rows.Scan(&r.ID, &r.ReporterID, &r.ContentType, &r.ContentID, &r.Reason, &r.Status, &r.CreatedAt)
		if err != nil {
			continue
		}
		reports = append(reports, &r)
	}

	return reports, nil
}

// ReviewReport updates a report's status
func (s *DBAdminModerationService) ReviewReport(reportID int64, status string) error {
	_, err := s.db.Exec("UPDATE reports SET status = $1, reviewed_at = NOW() WHERE id = $2", status, reportID)
	return err
}

// =============================================================================
// ADMIN SERVICE FACTORY
// =============================================================================

// AdminServices holds all admin-related service implementations
type AdminServices struct {
	Stats      *DBAdminStatsService
	Users      *DBAdminUserService
	Settings   *DBAdminSettingsService
	Moderation *DBAdminModerationService
}

// NewAdminServices creates all admin services
func NewAdminServices(db *sql.DB) *AdminServices {
	return &AdminServices{
		Stats:      NewDBAdminStatsService(db),
		Users:      NewDBAdminUserService(db),
		Settings:   NewDBAdminSettingsService(db),
		Moderation: NewDBAdminModerationService(db),
	}
}
