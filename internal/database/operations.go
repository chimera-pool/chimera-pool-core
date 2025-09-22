package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateUser creates a new user in the database
func CreateUser(db *sql.DB, user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO users (username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := db.QueryRowContext(ctx, query, user.Username, user.Email, user.Password, user.IsActive).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(db *sql.DB, id int64) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := &User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE id = $1
	`

	err := db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt, &user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// CreateMiner creates a new miner in the database
func CreateMiner(db *sql.DB, miner *Miner) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO miners (user_id, name, address, hashrate, is_active, created_at, updated_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), NOW())
		RETURNING id, created_at, updated_at, last_seen
	`

	err := db.QueryRowContext(ctx, query, miner.UserID, miner.Name, miner.Address, miner.Hashrate, miner.IsActive).
		Scan(&miner.ID, &miner.CreatedAt, &miner.UpdatedAt, &miner.LastSeen)
	
	if err != nil {
		return fmt.Errorf("failed to create miner: %w", err)
	}

	return nil
}

// CreateShare creates a new share in the database
func CreateShare(db *sql.DB, share *Share) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO shares (miner_id, user_id, difficulty, is_valid, nonce, hash, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, timestamp
	`

	err := db.QueryRowContext(ctx, query, share.MinerID, share.UserID, share.Difficulty, share.IsValid, share.Nonce, share.Hash).
		Scan(&share.ID, &share.Timestamp)
	
	if err != nil {
		return fmt.Errorf("failed to create share: %w", err)
	}

	return nil
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := &User{}
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE username = $1
	`

	err := db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt, &user.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetMinersByUserID retrieves all miners for a user
func GetMinersByUserID(db *sql.DB, userID int64) ([]*Miner, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, user_id, name, address, last_seen, hashrate, is_active, created_at, updated_at
		FROM miners
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query miners: %w", err)
	}
	defer rows.Close()

	var miners []*Miner
	for rows.Next() {
		miner := &Miner{}
		err := rows.Scan(
			&miner.ID, &miner.UserID, &miner.Name, &miner.Address,
			&miner.LastSeen, &miner.Hashrate, &miner.IsActive,
			&miner.CreatedAt, &miner.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan miner: %w", err)
		}
		miners = append(miners, miner)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating miners: %w", err)
	}

	return miners, nil
}

// UpdateMinerLastSeen updates the last seen timestamp for a miner
func UpdateMinerLastSeen(db *sql.DB, minerID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		UPDATE miners 
		SET last_seen = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, minerID)
	if err != nil {
		return fmt.Errorf("failed to update miner last seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("miner not found")
	}

	return nil
}

// GetSharesByMinerID retrieves shares for a specific miner
func GetSharesByMinerID(db *sql.DB, minerID int64, limit int) ([]*Share, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, miner_id, user_id, difficulty, is_valid, timestamp, nonce, hash
		FROM shares
		WHERE miner_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, minerID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query shares: %w", err)
	}
	defer rows.Close()

	var shares []*Share
	for rows.Next() {
		share := &Share{}
		err := rows.Scan(
			&share.ID, &share.MinerID, &share.UserID, &share.Difficulty,
			&share.IsValid, &share.Timestamp, &share.Nonce, &share.Hash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, share)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shares: %w", err)
	}

	return shares, nil
}