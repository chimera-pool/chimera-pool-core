// Package database provides database connectivity and operations for the Chimera Pool
package database

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Database represents the main database service
type Database struct {
	Pool   *ConnectionPool
	Config *Config
}

// New creates a new database instance with connection pool
func New(config *Config) (*Database, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Create connection pool
	pool, err := NewConnectionPool(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !pool.HealthCheck(ctx) {
		pool.Close()
		return nil, fmt.Errorf("database health check failed")
	}

	log.Printf("Database connection established successfully")

	return &Database{
		Pool:   pool,
		Config: config,
	}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.Pool != nil {
		return db.Pool.Close()
	}
	return nil
}

// HealthCheck performs a comprehensive health check
func (db *Database) HealthCheck(ctx context.Context) error {
	if db.Pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	if !db.Pool.HealthCheck(ctx) {
		return fmt.Errorf("database health check failed")
	}

	// Check if we can perform basic operations
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query users table: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func (db *Database) GetStats() PoolStats {
	if db.Pool == nil {
		return PoolStats{}
	}
	return db.Pool.Stats()
}

// RunMigrations runs database migrations
func (db *Database) RunMigrations(migrationsPath string) error {
	return RunMigrations(db.Config, migrationsPath)
}

// GetMigrationStatus returns current migration status
func (db *Database) GetMigrationStatus() (interface{}, error) {
	return GetMigrationStatus(db.Config)
}

// validateConfig validates the database configuration
func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if config.Database == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	if config.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if config.Password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if config.MaxConns < 0 {
		return fmt.Errorf("max connections cannot be negative")
	}

	if config.MinConns < 0 {
		return fmt.Errorf("min connections cannot be negative")
	}

	if config.MinConns > config.MaxConns && config.MaxConns > 0 {
		return fmt.Errorf("min connections cannot be greater than max connections")
	}

	return nil
}

// DefaultConfig returns a default database configuration
func DefaultConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     5432,
		Database: "chimera_pool_dev",
		Username: "chimera",
		Password: "dev_password",
		SSLMode:  "disable",
		MaxConns: 25,
		MinConns: 5,
	}
}