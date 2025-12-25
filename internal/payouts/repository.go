package payouts

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// =============================================================================
// USER PAYOUT SETTINGS REPOSITORY
// =============================================================================

// UserPayoutSettingsRepository handles database operations for user payout settings
type UserPayoutSettingsRepository interface {
	// GetUserSettings retrieves payout settings for a user
	GetUserSettings(ctx context.Context, userID int64) (*UserPayoutSettings, error)

	// CreateUserSettings creates new payout settings for a user
	CreateUserSettings(ctx context.Context, settings *UserPayoutSettings) error

	// UpdateUserSettings updates existing payout settings
	UpdateUserSettings(ctx context.Context, settings *UserPayoutSettings) error

	// DeleteUserSettings removes payout settings for a user
	DeleteUserSettings(ctx context.Context, userID int64) error

	// GetUsersByPayoutMode retrieves all users with a specific payout mode
	GetUsersByPayoutMode(ctx context.Context, mode PayoutMode) ([]UserPayoutSettings, error)

	// GetUsersForAutoPayout retrieves users eligible for automatic payout
	GetUsersForAutoPayout(ctx context.Context, minBalance int64) ([]UserPayoutSettings, error)
}

// =============================================================================
// POOL FEE CONFIG REPOSITORY
// =============================================================================

// PoolFeeConfigRepository handles database operations for pool fee configuration
type PoolFeeConfigRepository interface {
	// GetFeeConfig retrieves fee configuration for a mode and coin
	GetFeeConfig(ctx context.Context, mode PayoutMode, coinSymbol string) (*PoolFeeConfig, error)

	// GetAllFeeConfigs retrieves all fee configurations
	GetAllFeeConfigs(ctx context.Context) ([]PoolFeeConfig, error)

	// UpdateFeeConfig updates fee configuration
	UpdateFeeConfig(ctx context.Context, config *PoolFeeConfig) error

	// GetEnabledModes retrieves all enabled payout modes for a coin
	GetEnabledModes(ctx context.Context, coinSymbol string) ([]PayoutMode, error)
}

// PoolFeeConfig represents pool fee configuration from the database
type PoolFeeConfig struct {
	ID         int64      `json:"id" db:"id"`
	PayoutMode PayoutMode `json:"payout_mode" db:"payout_mode"`
	CoinSymbol string     `json:"coin_symbol" db:"coin_symbol"`
	FeePercent float64    `json:"fee_percent" db:"fee_percent"`
	MinPayout  int64      `json:"min_payout" db:"min_payout"`
	IsEnabled  bool       `json:"is_enabled" db:"is_enabled"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// =============================================================================
// SQL IMPLEMENTATION
// =============================================================================

// SQLUserPayoutSettingsRepository implements UserPayoutSettingsRepository using SQL
type SQLUserPayoutSettingsRepository struct {
	db *sql.DB
}

// NewSQLUserPayoutSettingsRepository creates a new SQL-based repository
func NewSQLUserPayoutSettingsRepository(db *sql.DB) *SQLUserPayoutSettingsRepository {
	return &SQLUserPayoutSettingsRepository{db: db}
}

// GetUserSettings retrieves payout settings for a user
func (r *SQLUserPayoutSettingsRepository) GetUserSettings(ctx context.Context, userID int64) (*UserPayoutSettings, error) {
	query := `
		SELECT user_id, payout_mode, min_payout_amount, payout_address, 
		       auto_payout_enable, created_at, updated_at
		FROM user_payout_settings
		WHERE user_id = $1
	`

	settings := &UserPayoutSettings{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.PayoutMode,
		&settings.MinPayoutAmount,
		&settings.PayoutAddress,
		&settings.AutoPayoutEnable,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No settings found, return nil without error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	return settings, nil
}

// CreateUserSettings creates new payout settings for a user
func (r *SQLUserPayoutSettingsRepository) CreateUserSettings(ctx context.Context, settings *UserPayoutSettings) error {
	query := `
		INSERT INTO user_payout_settings 
		(user_id, payout_mode, min_payout_amount, payout_address, auto_payout_enable)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.UserID,
		settings.PayoutMode,
		settings.MinPayoutAmount,
		settings.PayoutAddress,
		settings.AutoPayoutEnable,
	)

	if err != nil {
		return fmt.Errorf("failed to create user settings: %w", err)
	}

	return nil
}

// UpdateUserSettings updates existing payout settings
func (r *SQLUserPayoutSettingsRepository) UpdateUserSettings(ctx context.Context, settings *UserPayoutSettings) error {
	query := `
		UPDATE user_payout_settings 
		SET payout_mode = $2, min_payout_amount = $3, payout_address = $4, 
		    auto_payout_enable = $5, updated_at = NOW()
		WHERE user_id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		settings.UserID,
		settings.PayoutMode,
		settings.MinPayoutAmount,
		settings.PayoutAddress,
		settings.AutoPayoutEnable,
	)

	if err != nil {
		return fmt.Errorf("failed to update user settings: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no settings found for user %d", settings.UserID)
	}

	return nil
}

// DeleteUserSettings removes payout settings for a user
func (r *SQLUserPayoutSettingsRepository) DeleteUserSettings(ctx context.Context, userID int64) error {
	query := `DELETE FROM user_payout_settings WHERE user_id = $1`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user settings: %w", err)
	}

	return nil
}

// GetUsersByPayoutMode retrieves all users with a specific payout mode
func (r *SQLUserPayoutSettingsRepository) GetUsersByPayoutMode(ctx context.Context, mode PayoutMode) ([]UserPayoutSettings, error) {
	query := `
		SELECT user_id, payout_mode, min_payout_amount, payout_address, 
		       auto_payout_enable, created_at, updated_at
		FROM user_payout_settings
		WHERE payout_mode = $1
		ORDER BY user_id
	`

	rows, err := r.db.QueryContext(ctx, query, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by payout mode: %w", err)
	}
	defer rows.Close()

	var settings []UserPayoutSettings
	for rows.Next() {
		var s UserPayoutSettings
		err := rows.Scan(
			&s.UserID,
			&s.PayoutMode,
			&s.MinPayoutAmount,
			&s.PayoutAddress,
			&s.AutoPayoutEnable,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user settings: %w", err)
		}
		settings = append(settings, s)
	}

	return settings, nil
}

// GetUsersForAutoPayout retrieves users eligible for automatic payout
func (r *SQLUserPayoutSettingsRepository) GetUsersForAutoPayout(ctx context.Context, minBalance int64) ([]UserPayoutSettings, error) {
	query := `
		SELECT ups.user_id, ups.payout_mode, ups.min_payout_amount, ups.payout_address, 
		       ups.auto_payout_enable, ups.created_at, ups.updated_at
		FROM user_payout_settings ups
		WHERE ups.auto_payout_enable = true
		  AND ups.payout_address IS NOT NULL
		  AND ups.payout_address != ''
		ORDER BY ups.user_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get users for auto payout: %w", err)
	}
	defer rows.Close()

	var settings []UserPayoutSettings
	for rows.Next() {
		var s UserPayoutSettings
		err := rows.Scan(
			&s.UserID,
			&s.PayoutMode,
			&s.MinPayoutAmount,
			&s.PayoutAddress,
			&s.AutoPayoutEnable,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user settings: %w", err)
		}
		settings = append(settings, s)
	}

	return settings, nil
}

// =============================================================================
// POOL FEE CONFIG SQL IMPLEMENTATION
// =============================================================================

// SQLPoolFeeConfigRepository implements PoolFeeConfigRepository using SQL
type SQLPoolFeeConfigRepository struct {
	db *sql.DB
}

// NewSQLPoolFeeConfigRepository creates a new SQL-based repository
func NewSQLPoolFeeConfigRepository(db *sql.DB) *SQLPoolFeeConfigRepository {
	return &SQLPoolFeeConfigRepository{db: db}
}

// GetFeeConfig retrieves fee configuration for a mode and coin
func (r *SQLPoolFeeConfigRepository) GetFeeConfig(ctx context.Context, mode PayoutMode, coinSymbol string) (*PoolFeeConfig, error) {
	query := `
		SELECT id, payout_mode, coin_symbol, fee_percent, min_payout, 
		       is_enabled, created_at, updated_at
		FROM pool_fee_config
		WHERE payout_mode = $1 AND coin_symbol = $2
	`

	config := &PoolFeeConfig{}
	err := r.db.QueryRowContext(ctx, query, mode, coinSymbol).Scan(
		&config.ID,
		&config.PayoutMode,
		&config.CoinSymbol,
		&config.FeePercent,
		&config.MinPayout,
		&config.IsEnabled,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get fee config: %w", err)
	}

	return config, nil
}

// GetAllFeeConfigs retrieves all fee configurations
func (r *SQLPoolFeeConfigRepository) GetAllFeeConfigs(ctx context.Context) ([]PoolFeeConfig, error) {
	query := `
		SELECT id, payout_mode, coin_symbol, fee_percent, min_payout, 
		       is_enabled, created_at, updated_at
		FROM pool_fee_config
		ORDER BY coin_symbol, payout_mode
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all fee configs: %w", err)
	}
	defer rows.Close()

	var configs []PoolFeeConfig
	for rows.Next() {
		var c PoolFeeConfig
		err := rows.Scan(
			&c.ID,
			&c.PayoutMode,
			&c.CoinSymbol,
			&c.FeePercent,
			&c.MinPayout,
			&c.IsEnabled,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fee config: %w", err)
		}
		configs = append(configs, c)
	}

	return configs, nil
}

// UpdateFeeConfig updates fee configuration
func (r *SQLPoolFeeConfigRepository) UpdateFeeConfig(ctx context.Context, config *PoolFeeConfig) error {
	query := `
		UPDATE pool_fee_config 
		SET fee_percent = $3, min_payout = $4, is_enabled = $5, updated_at = NOW()
		WHERE payout_mode = $1 AND coin_symbol = $2
	`

	result, err := r.db.ExecContext(ctx, query,
		config.PayoutMode,
		config.CoinSymbol,
		config.FeePercent,
		config.MinPayout,
		config.IsEnabled,
	)

	if err != nil {
		return fmt.Errorf("failed to update fee config: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no fee config found for %s/%s", config.PayoutMode, config.CoinSymbol)
	}

	return nil
}

// GetEnabledModes retrieves all enabled payout modes for a coin
func (r *SQLPoolFeeConfigRepository) GetEnabledModes(ctx context.Context, coinSymbol string) ([]PayoutMode, error) {
	query := `
		SELECT payout_mode
		FROM pool_fee_config
		WHERE coin_symbol = $1 AND is_enabled = true
		ORDER BY payout_mode
	`

	rows, err := r.db.QueryContext(ctx, query, coinSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled modes: %w", err)
	}
	defer rows.Close()

	var modes []PayoutMode
	for rows.Next() {
		var mode PayoutMode
		if err := rows.Scan(&mode); err != nil {
			return nil, fmt.Errorf("failed to scan payout mode: %w", err)
		}
		modes = append(modes, mode)
	}

	return modes, nil
}
