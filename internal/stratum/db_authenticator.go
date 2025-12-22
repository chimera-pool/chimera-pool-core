package stratum

import (
	"context"
	"database/sql"
	"time"
)

// =============================================================================
// DATABASE-BACKED AUTHENTICATOR
// Production implementation connecting stratum to database
// =============================================================================

// DBMinerLookup implements MinerLookup using database queries
type DBMinerLookup struct {
	db *sql.DB
}

// NewDBMinerLookup creates a new database-backed miner lookup
func NewDBMinerLookup(db *sql.DB) *DBMinerLookup {
	return &DBMinerLookup{db: db}
}

// GetUserByUsername retrieves user info by username
func (l *DBMinerLookup) GetUserByUsername(ctx context.Context, username string) (*UserInfo, error) {
	query := `
		SELECT id, username, password_hash, 
		       COALESCE(is_active, true) as is_active,
		       COALESCE(role, 'user') as role
		FROM users 
		WHERE username = $1
	`

	var user UserInfo
	err := l.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.IsActive,
		&user.Role,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetMinerByWorkerName retrieves miner info by worker name
func (l *DBMinerLookup) GetMinerByWorkerName(ctx context.Context, userID int64, workerName string) (*MinerInfo, error) {
	query := `
		SELECT id, user_id, name, 
		       COALESCE(ip_address, '') as ip_address,
		       last_seen,
		       COALESCE(is_active, true) as is_active
		FROM miners 
		WHERE user_id = $1 AND name = $2
	`

	var miner MinerInfo
	var lastSeen sql.NullTime

	err := l.db.QueryRowContext(ctx, query, userID, workerName).Scan(
		&miner.ID,
		&miner.UserID,
		&miner.WorkerName,
		&miner.IPAddress,
		&lastSeen,
		&miner.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, ErrMinerNotFound
	}
	if err != nil {
		return nil, err
	}

	if lastSeen.Valid {
		miner.LastSeen = lastSeen.Time
	}

	return &miner, nil
}

// DBMinerRegistrar implements MinerRegistrar using database operations
type DBMinerRegistrar struct {
	db *sql.DB
}

// NewDBMinerRegistrar creates a new database-backed miner registrar
func NewDBMinerRegistrar(db *sql.DB) *DBMinerRegistrar {
	return &DBMinerRegistrar{db: db}
}

// RegisterMiner creates a new miner for a user
func (r *DBMinerRegistrar) RegisterMiner(ctx context.Context, userID int64, workerName string, ipAddress string) (*MinerInfo, error) {
	query := `
		INSERT INTO miners (user_id, name, ip_address, last_seen, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, $4)
		RETURNING id
	`

	now := time.Now()
	var minerID int64

	err := r.db.QueryRowContext(ctx, query, userID, workerName, ipAddress, now).Scan(&minerID)
	if err != nil {
		return nil, err
	}

	return &MinerInfo{
		ID:         minerID,
		UserID:     userID,
		WorkerName: workerName,
		IPAddress:  ipAddress,
		LastSeen:   now,
		IsActive:   true,
	}, nil
}

// UpdateMinerLastSeen updates the last seen timestamp
func (r *DBMinerRegistrar) UpdateMinerLastSeen(ctx context.Context, minerID int64) error {
	query := `UPDATE miners SET last_seen = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), minerID)
	return err
}

// =============================================================================
// FACTORY FUNCTION
// =============================================================================

// NewDatabaseAuthenticator creates a production-ready authenticator backed by database
func NewDatabaseAuthenticator(db *sql.DB) *CachedAuthenticator {
	lookup := NewDBMinerLookup(db)
	registrar := NewDBMinerRegistrar(db)
	config := DefaultCachedAuthenticatorConfig()

	return NewCachedAuthenticator(lookup, registrar, config)
}
