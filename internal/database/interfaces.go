package database

import (
	"context"
	"time"
)

// =============================================================================
// ISP-COMPLIANT DATABASE INTERFACES
// Each interface is small and focused on a single responsibility
// Enables easy mocking, testing, and future optimizations
// =============================================================================

// -----------------------------------------------------------------------------
// Core Query Interfaces
// -----------------------------------------------------------------------------

// QueryExecutor executes database queries (read operations)
type QueryExecutor interface {
	QueryRow(ctx context.Context, query string, args ...interface{}) Scanner
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
}

// CommandExecutor executes database commands (write operations)
type CommandExecutor interface {
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
}

// TransactionExecutor combines query and command execution
type TransactionExecutor interface {
	QueryExecutor
	CommandExecutor
}

// Scanner wraps database row scanning
type Scanner interface {
	Scan(dest ...interface{}) error
}

// Rows wraps database result rows
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

// Result wraps command execution result
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// -----------------------------------------------------------------------------
// Transaction Interfaces
// -----------------------------------------------------------------------------

// TransactionManager manages database transactions
type TransactionManager interface {
	Begin(ctx context.Context) (Tx, error)
	BeginReadOnly(ctx context.Context) (Tx, error)
}

// Tx represents a database transaction interface
type Tx interface {
	TransactionExecutor
	Commit() error
	Rollback() error
}

// TransactionFunc is a function that runs within a transaction
type TransactionFunc func(tx Tx) error

// -----------------------------------------------------------------------------
// Repository Interfaces (Domain-specific)
// -----------------------------------------------------------------------------

// UserReader handles user read operations
type UserReader interface {
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}

// UserWriter handles user write operations
type UserWriter interface {
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id int64) error
}

// UserRepository combines read and write operations
type UserRepository interface {
	UserReader
	UserWriter
}

// MinerReader handles miner read operations
type MinerReader interface {
	GetMinerByID(ctx context.Context, id int64) (*Miner, error)
	GetMinersByUserID(ctx context.Context, userID int64) ([]*Miner, error)
	GetActiveMinerCount(ctx context.Context) (int64, error)
}

// MinerWriter handles miner write operations
type MinerWriter interface {
	CreateMiner(ctx context.Context, miner *Miner) error
	UpdateMiner(ctx context.Context, miner *Miner) error
	UpdateMinerLastSeen(ctx context.Context, minerID int64) error
	UpdateMinerHashrate(ctx context.Context, minerID int64, hashrate float64) error
}

// MinerRepository combines read and write operations
type MinerRepository interface {
	MinerReader
	MinerWriter
}

// ShareReader handles share read operations
type ShareReader interface {
	GetSharesByMinerID(ctx context.Context, minerID int64, limit int) ([]*Share, error)
	GetSharesByUserID(ctx context.Context, userID int64, since time.Time, limit int) ([]*Share, error)
	GetSharesForPayout(ctx context.Context, blockTime time.Time, windowSize int64) ([]*Share, error)
	GetShareCount(ctx context.Context, minerID int64, since time.Time) (int64, error)
}

// ShareWriter handles share write operations
type ShareWriter interface {
	CreateShare(ctx context.Context, share *Share) error
	CreateShareBatch(ctx context.Context, shares []*Share) error // Batch insert for performance
}

// ShareRepository combines read and write operations
type ShareRepository interface {
	ShareReader
	ShareWriter
}

// BlockReader handles block read operations
type BlockReader interface {
	GetBlockByID(ctx context.Context, id int64) (*Block, error)
	GetBlockByHash(ctx context.Context, hash string) (*Block, error)
	GetBlocksByStatus(ctx context.Context, status string, limit int) ([]*Block, error)
	GetRecentBlocks(ctx context.Context, limit int) ([]*Block, error)
}

// BlockWriter handles block write operations
type BlockWriter interface {
	CreateBlock(ctx context.Context, block *Block) error
	UpdateBlockStatus(ctx context.Context, id int64, status string) error
}

// BlockRepository combines read and write operations
type BlockRepository interface {
	BlockReader
	BlockWriter
}

// PayoutReader handles payout read operations
type PayoutReader interface {
	GetPayoutByID(ctx context.Context, id int64) (*Payout, error)
	GetPayoutsByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Payout, error)
	GetPendingPayouts(ctx context.Context, limit int) ([]*Payout, error)
}

// PayoutWriter handles payout write operations
type PayoutWriter interface {
	CreatePayout(ctx context.Context, payout *Payout) error
	CreatePayoutBatch(ctx context.Context, payouts []*Payout) error
	UpdatePayoutStatus(ctx context.Context, id int64, status string, txHash string) error
}

// PayoutRepository combines read and write operations
type PayoutRepository interface {
	PayoutReader
	PayoutWriter
}

// -----------------------------------------------------------------------------
// Health & Metrics Interfaces
// -----------------------------------------------------------------------------

// HealthChecker checks database health
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
	Ping(ctx context.Context) error
}

// MetricsProvider provides database metrics
type MetricsProvider interface {
	GetPoolStats() PoolStats
	GetQueryStats() QueryStats
}

// QueryStats tracks query performance
type QueryStats struct {
	TotalQueries     int64
	SlowQueries      int64 // > 100ms
	FailedQueries    int64
	AvgQueryTimeMs   float64
	MaxQueryTimeMs   float64
	QueriesPerSecond float64
}

// -----------------------------------------------------------------------------
// Batch Operations Interface
// -----------------------------------------------------------------------------

// BatchInserter handles high-performance batch inserts
type BatchInserter interface {
	// InsertBatch inserts multiple rows in a single statement
	// Returns the number of rows inserted
	InsertBatch(ctx context.Context, table string, columns []string, values [][]interface{}) (int64, error)

	// CopyFrom uses PostgreSQL COPY protocol for maximum throughput
	// Can handle 100k+ rows/second
	CopyFrom(ctx context.Context, table string, columns []string, values [][]interface{}) (int64, error)
}

// -----------------------------------------------------------------------------
// Cache Interface
// -----------------------------------------------------------------------------

// QueryCache caches frequently accessed data
type QueryCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// CachedReader wraps a reader with caching
type CachedReader interface {
	WithCache(cache QueryCache) CachedReader
	InvalidateCache(keys ...string)
}

// -----------------------------------------------------------------------------
// Read Replica Interface
// -----------------------------------------------------------------------------

// ReadReplicaRouter routes read queries to replicas
type ReadReplicaRouter interface {
	// Primary returns the primary database for writes
	Primary() TransactionExecutor

	// Replica returns a read replica for queries
	// Implements round-robin or least-connections load balancing
	Replica() QueryExecutor

	// PreferPrimary forces reads to primary (for consistency)
	PreferPrimary() QueryExecutor
}
