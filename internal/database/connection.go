package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
	MaxConns int
	MinConns int
}

// ConnectionPool wraps sql.DB with additional functionality
type ConnectionPool struct {
	db *sql.DB
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	MaxConns  int32
	OpenConns int32
	InUse     int32
	Idle      int32
}

// Transaction wraps sql.Tx with context support
type Transaction struct {
	tx *sql.Tx
}

// NewConnectionPool creates a new database connection pool
func NewConnectionPool(config *Config) (*ConnectionPool, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	if config.MaxConns > 0 {
		db.SetMaxOpenConns(config.MaxConns)
	} else {
		db.SetMaxOpenConns(25) // Default
	}

	if config.MinConns > 0 {
		db.SetMaxIdleConns(config.MinConns)
	} else {
		db.SetMaxIdleConns(5) // Default
	}

	// Set connection lifetime
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &ConnectionPool{db: db}, nil
}

// Close closes the database connection pool
func (p *ConnectionPool) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// HealthCheck performs a health check on the database connection
func (p *ConnectionPool) HealthCheck(ctx context.Context) bool {
	if p.db == nil {
		return false
	}

	if err := p.db.PingContext(ctx); err != nil {
		return false
	}

	// Test a simple query
	var result int
	err := p.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	return err == nil && result == 1
}

// Stats returns connection pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	if p.db == nil {
		return PoolStats{}
	}

	stats := p.db.Stats()
	return PoolStats{
		MaxConns:  int32(stats.MaxOpenConnections),
		OpenConns: int32(stats.OpenConnections),
		InUse:     int32(stats.InUse),
		Idle:      int32(stats.Idle),
	}
}

// DB returns the underlying database connection for testing purposes
func (p *ConnectionPool) DB() *sql.DB {
	return p.db
}

// QueryRow executes a query that is expected to return at most one row
func (p *ConnectionPool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// Query executes a query that returns rows
func (p *ConnectionPool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// Exec executes a query without returning any rows
func (p *ConnectionPool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

// Begin starts a transaction
func (p *ConnectionPool) Begin(ctx context.Context) (*Transaction, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx: tx}, nil
}

// Rollback rolls back the transaction
func (tx *Transaction) Rollback(ctx context.Context) error {
	return tx.tx.Rollback()
}

// Commit commits the transaction
func (tx *Transaction) Commit(ctx context.Context) error {
	return tx.tx.Commit()
}

// QueryRow executes a query within the transaction
func (tx *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return tx.tx.QueryRowContext(ctx, query, args...)
}

// Query executes a query within the transaction
func (tx *Transaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.tx.QueryContext(ctx, query, args...)
}

// Exec executes a query within the transaction
func (tx *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return tx.tx.ExecContext(ctx, query, args...)
}

// RunMigrations runs database migrations
func RunMigrations(config *Config, migrationsPath string) error {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode,
	)

	// Open database connection for migrations
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	// Create migration driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetMigrationStatus returns the current migration status
func GetMigrationStatus(config *Config) (interface{}, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create migration driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance (using empty path since we only need status)
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	return map[string]interface{}{
		"version": version,
		"dirty":   dirty,
	}, nil
}