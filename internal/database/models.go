package database

import (
	"time"
	"github.com/google/uuid"
)

// User represents a mining pool user account
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
}

// Miner represents a mining device/worker
type Miner struct {
	ID         int64      `json:"id" db:"id"`
	UserID     int64      `json:"user_id" db:"user_id"`
	Name       string     `json:"name" db:"name"`
	Address    string     `json:"address" db:"address"`
	LastSeen   time.Time  `json:"last_seen" db:"last_seen"`
	Hashrate   float64    `json:"hashrate" db:"hashrate"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	NetworkID  *uuid.UUID `json:"network_id" db:"network_id"`
}

// Share represents a submitted mining share
type Share struct {
	ID         int64      `json:"id" db:"id"`
	MinerID    int64      `json:"miner_id" db:"miner_id"`
	UserID     int64      `json:"user_id" db:"user_id"`
	Difficulty float64    `json:"difficulty" db:"difficulty"`
	IsValid    bool       `json:"is_valid" db:"is_valid"`
	Timestamp  time.Time  `json:"timestamp" db:"timestamp"`
	Nonce      string     `json:"nonce" db:"nonce"`
	Hash       string     `json:"hash" db:"hash"`
	NetworkID  *uuid.UUID `json:"network_id" db:"network_id"`
}

// Block represents a found block
type Block struct {
	ID         int64      `json:"id" db:"id"`
	Height     int64      `json:"height" db:"height"`
	Hash       string     `json:"hash" db:"hash"`
	FinderID   int64      `json:"finder_id" db:"finder_id"`
	Reward     int64      `json:"reward" db:"reward"`
	Difficulty float64    `json:"difficulty" db:"difficulty"`
	Timestamp  time.Time  `json:"timestamp" db:"timestamp"`
	Status     string     `json:"status" db:"status"` // pending, confirmed, orphaned
	NetworkID  *uuid.UUID `json:"network_id" db:"network_id"`
}

// Payout represents a payout to a user
type Payout struct {
	ID          int64      `json:"id" db:"id"`
	UserID      int64      `json:"user_id" db:"user_id"`
	Amount      int64      `json:"amount" db:"amount"`
	Address     string     `json:"address" db:"address"`
	TxHash      string     `json:"tx_hash" db:"tx_hash"`
	Status      string     `json:"status" db:"status"` // pending, sent, confirmed, failed
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ProcessedAt *time.Time `json:"processed_at" db:"processed_at"`
	NetworkID   *uuid.UUID `json:"network_id" db:"network_id"`
}