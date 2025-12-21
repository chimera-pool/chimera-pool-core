package database

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseSchema(t *testing.T) {
	skipIfNoDatabase(t)

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	t.Run("UsersTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'users'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "users table should exist")
	})

	t.Run("MinersTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'miners'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "miners table should exist")
	})

	t.Run("SharesTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'shares'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "shares table should exist")
	})

	t.Run("BlocksTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'blocks'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "blocks table should exist")
	})

	t.Run("PayoutsTableExists", func(t *testing.T) {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'payouts'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "payouts table should exist")
	})
}

func TestBasicDatabaseOperations(t *testing.T) {
	skipIfNoDatabase(t)

	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	t.Run("CreateUser", func(t *testing.T) {
		user := &User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "hashedpassword",
			IsActive: true,
		}

		err := CreateUser(db, user)
		require.NoError(t, err)
		assert.Greater(t, user.ID, int64(0), "user ID should be set after creation")
		assert.False(t, user.CreatedAt.IsZero(), "created_at should be set")
		assert.False(t, user.UpdatedAt.IsZero(), "updated_at should be set")
	})

	t.Run("GetUserByID", func(t *testing.T) {
		// First create a user
		user := &User{
			Username: "getuser",
			Email:    "get@example.com",
			Password: "hashedpassword",
			IsActive: true,
		}
		err := CreateUser(db, user)
		require.NoError(t, err)

		// Then retrieve it
		retrieved, err := GetUserByID(db, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Username, retrieved.Username)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.IsActive, retrieved.IsActive)
	})

	t.Run("CreateMiner", func(t *testing.T) {
		// First create a user
		user := &User{
			Username: "minerowner",
			Email:    "miner@example.com",
			Password: "hashedpassword",
			IsActive: true,
		}
		err := CreateUser(db, user)
		require.NoError(t, err)

		// Then create a miner
		miner := &Miner{
			UserID:   user.ID,
			Name:     "test-miner-1",
			Address:  "192.168.1.100",
			Hashrate: 1000.0,
			IsActive: true,
		}

		err = CreateMiner(db, miner)
		require.NoError(t, err)
		assert.Greater(t, miner.ID, int64(0), "miner ID should be set after creation")
		assert.False(t, miner.CreatedAt.IsZero(), "created_at should be set")
	})

	t.Run("CreateShare", func(t *testing.T) {
		// Setup user and miner
		user := &User{
			Username: "shareuser",
			Email:    "share@example.com",
			Password: "hashedpassword",
			IsActive: true,
		}
		err := CreateUser(db, user)
		require.NoError(t, err)

		miner := &Miner{
			UserID:   user.ID,
			Name:     "share-miner",
			Address:  "192.168.1.101",
			Hashrate: 2000.0,
			IsActive: true,
		}
		err = CreateMiner(db, miner)
		require.NoError(t, err)

		// Create share
		share := &Share{
			MinerID:    miner.ID,
			UserID:     user.ID,
			Difficulty: 1000.0,
			IsValid:    true,
			Nonce:      "123456789",
			Hash:       "abcdef1234567890",
		}

		err = CreateShare(db, share)
		require.NoError(t, err)
		assert.Greater(t, share.ID, int64(0), "share ID should be set after creation")
		assert.False(t, share.Timestamp.IsZero(), "timestamp should be set")
	})
}

// Helper functions for testing
func setupTestDB(t *testing.T) *sql.DB {
	config := getTestConfig()

	// Run migrations first
	err := RunMigrations(config, "../../migrations")
	require.NoError(t, err, "should run migrations successfully")

	// Create connection pool
	pool, err := NewConnectionPool(config)
	require.NoError(t, err, "should create connection pool")

	return pool.DB()
}

func teardownTestDB(t *testing.T, db *sql.DB) {
	// Clean up test data
	db.Exec("TRUNCATE TABLE payouts, blocks, shares, miners, users RESTART IDENTITY CASCADE")
	db.Close()
}
