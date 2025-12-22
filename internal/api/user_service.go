package api

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// USER SERVICE IMPLEMENTATIONS
// ISP-compliant services that implement the user interfaces
// =============================================================================

// -----------------------------------------------------------------------------
// User Profile Reader Implementation
// -----------------------------------------------------------------------------

// DBUserProfileReader implements UserProfileReader using database
type DBUserProfileReader struct {
	db *sql.DB
}

// NewDBUserProfileReader creates a new database user profile reader
func NewDBUserProfileReader(db *sql.DB) *DBUserProfileReader {
	return &DBUserProfileReader{db: db}
}

// GetProfile returns a user's profile data
func (r *DBUserProfileReader) GetProfile(userID int64) (*UserProfileData, error) {
	var username, email string
	var payoutAddress sql.NullString
	var isAdmin bool
	var createdAt time.Time

	err := r.db.QueryRow(
		"SELECT username, email, payout_address, is_admin, created_at FROM users WHERE id = $1",
		userID,
	).Scan(&username, &email, &payoutAddress, &isAdmin, &createdAt)

	if err != nil {
		return nil, errors.New("user not found")
	}

	return &UserProfileData{
		ID:            userID,
		Username:      username,
		Email:         email,
		PayoutAddress: payoutAddress.String,
		IsAdmin:       isAdmin,
		CreatedAt:     createdAt,
	}, nil
}

// -----------------------------------------------------------------------------
// User Profile Writer Implementation
// -----------------------------------------------------------------------------

// DBUserProfileWriter implements UserProfileWriter using database
type DBUserProfileWriter struct {
	db *sql.DB
}

// NewDBUserProfileWriter creates a new database user profile writer
func NewDBUserProfileWriter(db *sql.DB) *DBUserProfileWriter {
	return &DBUserProfileWriter{db: db}
}

// UpdateProfile updates a user's profile data
func (w *DBUserProfileWriter) UpdateProfile(userID int64, data *UpdateProfileData) (*UserProfileData, error) {
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if data.Username != "" {
		if len(data.Username) < 3 || len(data.Username) > 50 {
			return nil, errors.New("username must be between 3 and 50 characters")
		}

		// Check if username is taken
		var existingID int64
		err := w.db.QueryRow("SELECT id FROM users WHERE username = $1 AND id != $2", data.Username, userID).Scan(&existingID)
		if err == nil {
			return nil, errors.New("username is already taken")
		}

		updates = append(updates, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, data.Username)
		argIndex++
	}

	if data.PayoutAddress != "" {
		updates = append(updates, fmt.Sprintf("payout_address = $%d", argIndex))
		args = append(args, data.PayoutAddress)
		argIndex++
	}

	if len(updates) > 0 {
		args = append(args, userID)
		query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)
		_, err := w.db.Exec(query, args...)
		if err != nil {
			return nil, errors.New("failed to update profile")
		}
	}

	// Return updated profile
	reader := NewDBUserProfileReader(w.db)
	return reader.GetProfile(userID)
}

// -----------------------------------------------------------------------------
// User Password Changer Implementation
// -----------------------------------------------------------------------------

// DBUserPasswordChanger implements UserPasswordChanger using database
type DBUserPasswordChanger struct {
	db *sql.DB
}

// NewDBUserPasswordChanger creates a new database password changer
func NewDBUserPasswordChanger(db *sql.DB) *DBUserPasswordChanger {
	return &DBUserPasswordChanger{db: db}
}

// ChangePassword changes a user's password
func (c *DBUserPasswordChanger) ChangePassword(userID int64, currentPassword, newPassword string) error {
	// Get current password hash
	var passwordHash string
	err := c.db.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&passwordHash)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Validate new password
	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to process new password")
	}

	// Update password
	_, err = c.db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", string(newHash), userID)
	if err != nil {
		return errors.New("failed to update password")
	}

	return nil
}

// -----------------------------------------------------------------------------
// User Miner Reader Implementation
// -----------------------------------------------------------------------------

// DBUserMinerReader implements UserMinerReader using database
type DBUserMinerReader struct {
	db *sql.DB
}

// NewDBUserMinerReader creates a new database miner reader
func NewDBUserMinerReader(db *sql.DB) *DBUserMinerReader {
	return &DBUserMinerReader{db: db}
}

// GetMiners returns all miners for a user
func (r *DBUserMinerReader) GetMiners(userID int64) ([]*MinerData, error) {
	rows, err := r.db.Query(
		"SELECT id, worker_name, hashrate, last_share_at, is_active, COALESCE(share_count, 0) FROM miners WHERE user_id = $1 ORDER BY is_active DESC, last_share_at DESC",
		userID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch miners")
	}
	defer rows.Close()

	var miners []*MinerData
	for rows.Next() {
		var m MinerData
		var lastSeen sql.NullTime
		err := rows.Scan(&m.ID, &m.Name, &m.Hashrate, &lastSeen, &m.IsActive, &m.ShareCount)
		if err != nil {
			continue
		}
		if lastSeen.Valid {
			m.LastSeen = lastSeen.Time
		}
		miners = append(miners, &m)
	}

	return miners, nil
}

// -----------------------------------------------------------------------------
// User Payout Reader Implementation
// -----------------------------------------------------------------------------

// DBUserPayoutReader implements UserPayoutReader using database
type DBUserPayoutReader struct {
	db *sql.DB
}

// NewDBUserPayoutReader creates a new database payout reader
func NewDBUserPayoutReader(db *sql.DB) *DBUserPayoutReader {
	return &DBUserPayoutReader{db: db}
}

// GetPayouts returns payouts for a user
func (r *DBUserPayoutReader) GetPayouts(userID int64, limit, offset int) ([]*PayoutData, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.db.Query(
		"SELECT id, amount, tx_hash, status, created_at FROM payouts WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		userID, limit, offset,
	)
	if err != nil {
		return nil, errors.New("failed to fetch payouts")
	}
	defer rows.Close()

	var payouts []*PayoutData
	for rows.Next() {
		var p PayoutData
		var txHash sql.NullString
		err := rows.Scan(&p.ID, &p.Amount, &txHash, &p.Status, &p.CreatedAt)
		if err != nil {
			continue
		}
		if txHash.Valid {
			p.TxHash = txHash.String
		}
		payouts = append(payouts, &p)
	}

	return payouts, nil
}

// -----------------------------------------------------------------------------
// User Stats Reader Implementation
// -----------------------------------------------------------------------------

// DBUserStatsReader implements UserStatsReader using database
type DBUserStatsReader struct {
	db *sql.DB
}

// NewDBUserStatsReader creates a new database stats reader
func NewDBUserStatsReader(db *sql.DB) *DBUserStatsReader {
	return &DBUserStatsReader{db: db}
}

// GetHashrateHistory returns hashrate history for a user
func (r *DBUserStatsReader) GetHashrateHistory(userID int64, period string) ([]*HashratePoint, error) {
	interval := getPeriodIntervalSQL(period)

	rows, err := r.db.Query(
		fmt.Sprintf("SELECT timestamp, hashrate FROM hashrate_history WHERE user_id = $1 AND timestamp > NOW() - INTERVAL '%s' ORDER BY timestamp", interval),
		userID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch hashrate history")
	}
	defer rows.Close()

	var history []*HashratePoint
	for rows.Next() {
		var h HashratePoint
		err := rows.Scan(&h.Timestamp, &h.Hashrate)
		if err != nil {
			continue
		}
		history = append(history, &h)
	}

	return history, nil
}

// GetSharesHistory returns shares history for a user
func (r *DBUserStatsReader) GetSharesHistory(userID int64, period string) ([]*SharesPoint, error) {
	interval := getPeriodIntervalSQL(period)

	rows, err := r.db.Query(
		fmt.Sprintf("SELECT timestamp, valid_shares, invalid_shares FROM shares_history WHERE user_id = $1 AND timestamp > NOW() - INTERVAL '%s' ORDER BY timestamp", interval),
		userID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch shares history")
	}
	defer rows.Close()

	var history []*SharesPoint
	for rows.Next() {
		var s SharesPoint
		err := rows.Scan(&s.Timestamp, &s.ValidShares, &s.InvalidShares)
		if err != nil {
			continue
		}
		history = append(history, &s)
	}

	return history, nil
}

// GetEarningsHistory returns earnings history for a user
func (r *DBUserStatsReader) GetEarningsHistory(userID int64, period string) ([]*EarningsPoint, error) {
	interval := getPeriodIntervalSQL(period)

	rows, err := r.db.Query(
		fmt.Sprintf("SELECT timestamp, amount FROM earnings_history WHERE user_id = $1 AND timestamp > NOW() - INTERVAL '%s' ORDER BY timestamp", interval),
		userID,
	)
	if err != nil {
		return nil, errors.New("failed to fetch earnings history")
	}
	defer rows.Close()

	var history []*EarningsPoint
	for rows.Next() {
		var e EarningsPoint
		err := rows.Scan(&e.Timestamp, &e.Earnings)
		if err != nil {
			continue
		}
		history = append(history, &e)
	}

	return history, nil
}

func getPeriodIntervalSQL(period string) string {
	switch period {
	case "1h":
		return "1 hour"
	case "6h":
		return "6 hours"
	case "24h":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "24 hours"
	}
}

// =============================================================================
// USER SERVICE FACTORY
// Creates all user services with proper dependencies
// =============================================================================

// UserServices holds all user-related service implementations
type UserServices struct {
	ProfileReader   UserProfileReader
	ProfileWriter   UserProfileWriter
	PasswordChanger UserPasswordChanger
	MinerReader     UserMinerReader
	PayoutReader    UserPayoutReader
	StatsReader     UserStatsReader
}

// NewUserServices creates all user services
func NewUserServices(db *sql.DB) *UserServices {
	return &UserServices{
		ProfileReader:   NewDBUserProfileReader(db),
		ProfileWriter:   NewDBUserProfileWriter(db),
		PasswordChanger: NewDBUserPasswordChanger(db),
		MinerReader:     NewDBUserMinerReader(db),
		PayoutReader:    NewDBUserPayoutReader(db),
		StatsReader:     NewDBUserStatsReader(db),
	}
}

// CreateUserHandlers creates UserHandlers with all services wired up
func (s *UserServices) CreateUserHandlers() *UserHandlers {
	return NewUserHandlers(
		s.ProfileReader,
		s.ProfileWriter,
		s.PasswordChanger,
		s.MinerReader,
		s.PayoutReader,
		s.StatsReader,
	)
}
