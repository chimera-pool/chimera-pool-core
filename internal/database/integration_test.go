//go:build integration
// +build integration

package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDatabaseIntegrationWithContainer(t *testing.T) {
	// Start PostgreSQL container
	ctx := context.Background()
	
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "chimera_pool_test",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer postgresContainer.Terminate(ctx)

	// Get container connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Create database config
	config := &Config{
		Host:     host,
		Port:     port.Int(),
		Database: "chimera_pool_test",
		Username: "test_user",
		Password: "test_password",
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}

	t.Run("E2E_DatabaseOperations", func(t *testing.T) {
		// Test connection pool creation
		pool, err := NewConnectionPool(config)
		require.NoError(t, err, "should create connection pool")
		defer pool.Close()

		// Test health check
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		healthy := pool.HealthCheck(ctx)
		assert.True(t, healthy, "connection pool should be healthy")

		// Create tables manually for this test (since we can't run migrations easily in container)
		err = createTestTables(pool.DB())
		require.NoError(t, err, "should create test tables")

		// Test complete user workflow
		user := &User{
			Username: "integration_user",
			Email:    "integration@test.com",
			Password: "hashed_password",
			IsActive: true,
		}

		err = CreateUser(pool.DB(), user)
		require.NoError(t, err, "should create user")
		assert.Greater(t, user.ID, int64(0), "user ID should be set")

		// Test user retrieval
		retrievedUser, err := GetUserByID(pool.DB(), user.ID)
		require.NoError(t, err, "should retrieve user")
		assert.Equal(t, user.Username, retrievedUser.Username)
		assert.Equal(t, user.Email, retrievedUser.Email)

		// Test miner creation
		miner := &Miner{
			UserID:   user.ID,
			Name:     "integration_miner",
			Address:  "192.168.1.200",
			Hashrate: 5000.0,
			IsActive: true,
		}

		err = CreateMiner(pool.DB(), miner)
		require.NoError(t, err, "should create miner")
		assert.Greater(t, miner.ID, int64(0), "miner ID should be set")

		// Test share creation
		share := &Share{
			MinerID:    miner.ID,
			UserID:     user.ID,
			Difficulty: 2000.0,
			IsValid:    true,
			Nonce:      "integration_nonce",
			Hash:       "integration_hash",
		}

		err = CreateShare(pool.DB(), share)
		require.NoError(t, err, "should create share")
		assert.Greater(t, share.ID, int64(0), "share ID should be set")

		// Test transaction support
		tx, err := pool.Begin(ctx)
		require.NoError(t, err, "should begin transaction")

		// Create another user in transaction
		txUser := &User{
			Username: "tx_user",
			Email:    "tx@test.com",
			Password: "tx_password",
			IsActive: true,
		}

		// Use transaction to create user
		query := `
			INSERT INTO users (username, email, password_hash, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`
		err = tx.QueryRow(ctx, query, txUser.Username, txUser.Email, txUser.Password, txUser.IsActive).
			Scan(&txUser.ID, &txUser.CreatedAt, &txUser.UpdatedAt)
		require.NoError(t, err, "should create user in transaction")

		// Rollback transaction
		err = tx.Rollback(ctx)
		require.NoError(t, err, "should rollback transaction")

		// Verify user was not created (due to rollback)
		_, err = GetUserByID(pool.DB(), txUser.ID)
		assert.Error(t, err, "user should not exist after rollback")
	})

	t.Run("E2E_ConnectionPoolStats", func(t *testing.T) {
		pool, err := NewConnectionPool(config)
		require.NoError(t, err)
		defer pool.Close()

		stats := pool.Stats()
		assert.Greater(t, stats.MaxConns, int32(0), "should have max connections configured")
		assert.GreaterOrEqual(t, stats.OpenConns, int32(0), "should have open connections")
	})

	t.Run("E2E_ConcurrentOperations", func(t *testing.T) {
		pool, err := NewConnectionPool(config)
		require.NoError(t, err)
		defer pool.Close()

		// Create tables
		err = createTestTables(pool.DB())
		require.NoError(t, err)

		// Test concurrent operations
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				user := &User{
					Username: fmt.Sprintf("concurrent_user_%d", id),
					Email:    fmt.Sprintf("concurrent_%d@test.com", id),
					Password: "concurrent_password",
					IsActive: true,
				}

				if err := CreateUser(pool.DB(), user); err != nil {
					errors <- err
					return
				}

				// Verify user was created
				_, err := GetUserByID(pool.DB(), user.ID)
				if err != nil {
					errors <- err
					return
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("Concurrent operation failed: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}
	})
}

// Helper function to create test tables
func createTestTables(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			is_active BOOLEAN DEFAULT true
		);

		CREATE TABLE IF NOT EXISTS miners (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			address INET,
			last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			hashrate DECIMAL(20,2) DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS shares (
			id BIGSERIAL PRIMARY KEY,
			miner_id BIGINT NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			difficulty DECIMAL(20,8) NOT NULL,
			is_valid BOOLEAN NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			nonce VARCHAR(64) NOT NULL,
			hash VARCHAR(64) NOT NULL
		);

		CREATE TABLE IF NOT EXISTS blocks (
			id BIGSERIAL PRIMARY KEY,
			height BIGINT UNIQUE NOT NULL,
			hash VARCHAR(64) UNIQUE NOT NULL,
			finder_id BIGINT NOT NULL REFERENCES users(id),
			reward BIGINT NOT NULL,
			difficulty DECIMAL(20,8) NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'orphaned'))
		);

		CREATE TABLE IF NOT EXISTS payouts (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			amount BIGINT NOT NULL,
			address VARCHAR(255) NOT NULL,
			tx_hash VARCHAR(64),
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'confirmed', 'failed')),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			processed_at TIMESTAMP WITH TIME ZONE
		);
	`

	_, err := db.Exec(schema)
	return err
}