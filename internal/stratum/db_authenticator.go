package stratum

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/geoip"
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
	db    *sql.DB
	geoIP geoip.GeoIPService
}

// NewDBMinerRegistrar creates a new database-backed miner registrar
func NewDBMinerRegistrar(db *sql.DB) *DBMinerRegistrar {
	return &DBMinerRegistrar{
		db:    db,
		geoIP: geoip.NewService(),
	}
}

// RegisterMiner creates a new miner for a user with GeoIP lookup
func (r *DBMinerRegistrar) RegisterMiner(ctx context.Context, userID int64, workerName string, ipAddress string) (*MinerInfo, error) {
	// Perform GeoIP lookup for miner location
	var city, country, countryCode, continent string
	var lat, lng float64

	if r.geoIP != nil && ipAddress != "" {
		location, err := r.geoIP.LookupWithCache(ipAddress)
		if err == nil && location != nil {
			city = location.City
			country = location.Country
			countryCode = location.CountryCode
			continent = location.Continent
			lat = location.Latitude
			lng = location.Longitude
			log.Printf("[GeoIP] Miner %s located: %s, %s (%s)", workerName, city, country, ipAddress)
		} else if err != nil {
			log.Printf("[GeoIP] Lookup failed for %s: %v", ipAddress, err)
		}
	}

	query := `
		INSERT INTO miners (user_id, name, ip_address, city, country, country_code, continent, latitude, longitude, location_updated_at, last_seen, is_active, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, 0), NULLIF($9, 0), $10, $10, true, $10)
		RETURNING id
	`

	now := time.Now()
	var minerID int64

	err := r.db.QueryRowContext(ctx, query, userID, workerName, ipAddress, city, country, countryCode, continent, lat, lng, now).Scan(&minerID)
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

// UpdateMinerLocation updates the miner's location using GeoIP lookup
func (r *DBMinerRegistrar) UpdateMinerLocation(ctx context.Context, minerID int64, ipAddress string) error {
	if r.geoIP == nil || ipAddress == "" {
		return nil
	}

	location, err := r.geoIP.LookupWithCache(ipAddress)
	if err != nil || location == nil {
		return err
	}

	query := `
		UPDATE miners SET 
			ip_address = $1,
			city = NULLIF($2, ''),
			country = NULLIF($3, ''),
			country_code = NULLIF($4, ''),
			continent = NULLIF($5, ''),
			latitude = NULLIF($6, 0),
			longitude = NULLIF($7, 0),
			location_updated_at = $8
		WHERE id = $9
	`

	_, err = r.db.ExecContext(ctx, query,
		ipAddress,
		location.City,
		location.Country,
		location.CountryCode,
		location.Continent,
		location.Latitude,
		location.Longitude,
		time.Now(),
		minerID,
	)

	if err == nil {
		log.Printf("[GeoIP] Updated miner %d location: %s, %s", minerID, location.City, location.Country)
	}

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
