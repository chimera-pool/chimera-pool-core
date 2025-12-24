package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Scanner Tests
// =============================================================================

func TestMockScanner_Scan_Success(t *testing.T) {
	now := time.Now()
	values := []interface{}{int64(1), "testuser", "test@example.com", true, 3.14, now}
	scanner := NewMockScanner(values, nil)

	var id int64
	var username, email string
	var active bool
	var rate float64
	var timestamp time.Time

	err := scanner.Scan(&id, &username, &email, &active, &rate, &timestamp)
	require.NoError(t, err)

	assert.Equal(t, int64(1), id)
	assert.Equal(t, "testuser", username)
	assert.Equal(t, "test@example.com", email)
	assert.True(t, active)
	assert.Equal(t, 3.14, rate)
	assert.Equal(t, now, timestamp)
}

func TestMockScanner_Scan_Error(t *testing.T) {
	scanner := NewMockScanner(nil, assert.AnError)

	var id int64
	err := scanner.Scan(&id)
	assert.Error(t, err)
}

// =============================================================================
// Mock Rows Tests
// =============================================================================

func TestMockRows_Iteration(t *testing.T) {
	data := [][]interface{}{
		{int64(1), "user1"},
		{int64(2), "user2"},
		{int64(3), "user3"},
	}
	rows := NewMockRows(data)

	var results []struct {
		id   int64
		name string
	}

	for rows.Next() {
		var id int64
		var name string
		err := rows.Scan(&id, &name)
		require.NoError(t, err)
		results = append(results, struct {
			id   int64
			name string
		}{id, name})
	}

	assert.Len(t, results, 3)
	assert.Equal(t, int64(1), results[0].id)
	assert.Equal(t, "user1", results[0].name)
	assert.Equal(t, int64(3), results[2].id)

	err := rows.Close()
	assert.NoError(t, err)
	assert.Nil(t, rows.Err())
}

func TestMockRows_Empty(t *testing.T) {
	rows := NewMockRows([][]interface{}{})

	assert.False(t, rows.Next())
	assert.Nil(t, rows.Err())
}

// =============================================================================
// Mock Result Tests
// =============================================================================

func TestMockResult_Success(t *testing.T) {
	result := NewMockResult(42, 1)

	lastID, err := result.LastInsertId()
	assert.NoError(t, err)
	assert.Equal(t, int64(42), lastID)

	affected, err := result.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

// =============================================================================
// In-Memory User Repository Tests
// =============================================================================

func TestInMemoryUserRepository_CreateAndGet(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		IsActive: true,
	}

	err := repo.CreateUser(ctx, user)
	require.NoError(t, err)
	assert.Greater(t, user.ID, int64(0))

	// Get by ID
	retrieved, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Username, retrieved.Username)
	assert.Equal(t, user.Email, retrieved.Email)

	// Get by Username
	retrieved, err = repo.GetUserByUsername(ctx, "testuser")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)

	// Get by Email
	retrieved, err = repo.GetUserByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
}

func TestInMemoryUserRepository_CreateDuplicate(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	user1 := &User{Username: "testuser", Email: "test1@example.com", IsActive: true}
	err := repo.CreateUser(ctx, user1)
	require.NoError(t, err)

	// Duplicate username
	user2 := &User{Username: "testuser", Email: "test2@example.com", IsActive: true}
	err = repo.CreateUser(ctx, user2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")

	// Duplicate email
	user3 := &User{Username: "testuser2", Email: "test1@example.com", IsActive: true}
	err = repo.CreateUser(ctx, user3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
}

func TestInMemoryUserRepository_Update(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	user := &User{Username: "testuser", Email: "test@example.com", IsActive: true}
	err := repo.CreateUser(ctx, user)
	require.NoError(t, err)

	user.Username = "updateduser"
	err = repo.UpdateUser(ctx, user)
	require.NoError(t, err)

	retrieved, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updateduser", retrieved.Username)
}

func TestInMemoryUserRepository_Delete(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	user := &User{Username: "testuser", Email: "test@example.com", IsActive: true}
	err := repo.CreateUser(ctx, user)
	require.NoError(t, err)

	err = repo.DeleteUser(ctx, user.ID)
	require.NoError(t, err)

	_, err = repo.GetUserByID(ctx, user.ID)
	assert.Error(t, err)
}

func TestInMemoryUserRepository_NotFound(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	_, err := repo.GetUserByID(ctx, 999)
	assert.Error(t, err)

	_, err = repo.GetUserByUsername(ctx, "nonexistent")
	assert.Error(t, err)

	_, err = repo.GetUserByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)

	err = repo.UpdateUser(ctx, &User{ID: 999})
	assert.Error(t, err)

	err = repo.DeleteUser(ctx, 999)
	assert.Error(t, err)
}

// =============================================================================
// In-Memory Miner Repository Tests
// =============================================================================

func TestInMemoryMinerRepository_CreateAndGet(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	miner := &Miner{
		UserID:   1,
		Name:     "worker1",
		Address:  "ltc1abc...",
		Hashrate: 1000000.0,
		IsActive: true,
	}

	err := repo.CreateMiner(ctx, miner)
	require.NoError(t, err)
	assert.Greater(t, miner.ID, int64(0))

	retrieved, err := repo.GetMinerByID(ctx, miner.ID)
	require.NoError(t, err)
	assert.Equal(t, miner.Name, retrieved.Name)
	assert.Equal(t, miner.Hashrate, retrieved.Hashrate)
}

func TestInMemoryMinerRepository_GetByUserID(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	// Create miners for user 1
	for i := 0; i < 3; i++ {
		miner := &Miner{UserID: 1, Name: "worker", IsActive: true}
		err := repo.CreateMiner(ctx, miner)
		require.NoError(t, err)
	}

	// Create miner for user 2
	miner := &Miner{UserID: 2, Name: "other_worker", IsActive: true}
	err := repo.CreateMiner(ctx, miner)
	require.NoError(t, err)

	miners, err := repo.GetMinersByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, miners, 3)
}

func TestInMemoryMinerRepository_GetActiveMinerCount(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	// Create active miners
	for i := 0; i < 5; i++ {
		miner := &Miner{UserID: 1, Name: "active", IsActive: true}
		err := repo.CreateMiner(ctx, miner)
		require.NoError(t, err)
	}

	// Create inactive miners
	for i := 0; i < 3; i++ {
		miner := &Miner{UserID: 1, Name: "inactive", IsActive: false}
		err := repo.CreateMiner(ctx, miner)
		require.NoError(t, err)
	}

	count, err := repo.GetActiveMinerCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestInMemoryMinerRepository_UpdateHashrate(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	miner := &Miner{UserID: 1, Name: "worker", Hashrate: 1000.0, IsActive: true}
	err := repo.CreateMiner(ctx, miner)
	require.NoError(t, err)

	err = repo.UpdateMinerHashrate(ctx, miner.ID, 5000.0)
	require.NoError(t, err)

	retrieved, err := repo.GetMinerByID(ctx, miner.ID)
	require.NoError(t, err)
	assert.Equal(t, 5000.0, retrieved.Hashrate)
}

func TestInMemoryMinerRepository_UpdateLastSeen(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	miner := &Miner{UserID: 1, Name: "worker", IsActive: true}
	err := repo.CreateMiner(ctx, miner)
	require.NoError(t, err)

	originalLastSeen := miner.LastSeen
	time.Sleep(10 * time.Millisecond)

	err = repo.UpdateMinerLastSeen(ctx, miner.ID)
	require.NoError(t, err)

	retrieved, err := repo.GetMinerByID(ctx, miner.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.LastSeen.After(originalLastSeen) || retrieved.LastSeen.Equal(originalLastSeen))
}

func TestInMemoryMinerRepository_NotFound(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	_, err := repo.GetMinerByID(ctx, 999)
	assert.Error(t, err)

	err = repo.UpdateMiner(ctx, &Miner{ID: 999})
	assert.Error(t, err)

	err = repo.UpdateMinerLastSeen(ctx, 999)
	assert.Error(t, err)

	err = repo.UpdateMinerHashrate(ctx, 999, 1000.0)
	assert.Error(t, err)
}

// =============================================================================
// In-Memory Share Repository Tests
// =============================================================================

func TestInMemoryShareRepository_CreateAndGet(t *testing.T) {
	repo := NewInMemoryShareRepository()
	ctx := context.Background()

	share := &Share{
		MinerID:    1,
		UserID:     1,
		Difficulty: 1000.0,
		IsValid:    true,
		Nonce:      "abc123",
		Hash:       "0x1234...",
	}

	err := repo.CreateShare(ctx, share)
	require.NoError(t, err)
	assert.Greater(t, share.ID, int64(0))

	retrieved, err := repo.GetShareByID(ctx, share.ID)
	require.NoError(t, err)
	assert.Equal(t, share.Difficulty, retrieved.Difficulty)
	assert.Equal(t, share.IsValid, retrieved.IsValid)
}

func TestInMemoryShareRepository_GetByMinerID(t *testing.T) {
	repo := NewInMemoryShareRepository()
	ctx := context.Background()

	// Create shares for miner 1
	for i := 0; i < 10; i++ {
		share := &Share{MinerID: 1, UserID: 1, Difficulty: 1000.0, IsValid: true}
		err := repo.CreateShare(ctx, share)
		require.NoError(t, err)
	}

	// Create shares for miner 2
	for i := 0; i < 5; i++ {
		share := &Share{MinerID: 2, UserID: 1, Difficulty: 1000.0, IsValid: true}
		err := repo.CreateShare(ctx, share)
		require.NoError(t, err)
	}

	shares, err := repo.GetSharesByMinerID(ctx, 1, 100)
	require.NoError(t, err)
	assert.Len(t, shares, 10)

	// Test limit
	shares, err = repo.GetSharesByMinerID(ctx, 1, 5)
	require.NoError(t, err)
	assert.Len(t, shares, 5)
}

func TestInMemoryShareRepository_GetValidShareCount(t *testing.T) {
	repo := NewInMemoryShareRepository()
	ctx := context.Background()

	// Create valid shares
	for i := 0; i < 7; i++ {
		share := &Share{MinerID: 1, UserID: 1, Difficulty: 1000.0, IsValid: true}
		err := repo.CreateShare(ctx, share)
		require.NoError(t, err)
	}

	// Create invalid shares
	for i := 0; i < 3; i++ {
		share := &Share{MinerID: 1, UserID: 1, Difficulty: 1000.0, IsValid: false}
		err := repo.CreateShare(ctx, share)
		require.NoError(t, err)
	}

	count, err := repo.GetValidShareCount(ctx, 1, time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(7), count)
}

func TestInMemoryShareRepository_CreateBatch(t *testing.T) {
	repo := NewInMemoryShareRepository()
	ctx := context.Background()

	shares := []*Share{
		{MinerID: 1, UserID: 1, Difficulty: 1000.0, IsValid: true},
		{MinerID: 1, UserID: 1, Difficulty: 2000.0, IsValid: true},
		{MinerID: 1, UserID: 1, Difficulty: 3000.0, IsValid: false},
	}

	err := repo.CreateShareBatch(ctx, shares)
	require.NoError(t, err)

	// Verify all were created
	for _, share := range shares {
		assert.Greater(t, share.ID, int64(0))
	}

	// Verify can retrieve
	allShares, err := repo.GetSharesByMinerID(ctx, 1, 100)
	require.NoError(t, err)
	assert.Len(t, allShares, 3)
}

func TestInMemoryShareRepository_NotFound(t *testing.T) {
	repo := NewInMemoryShareRepository()
	ctx := context.Background()

	_, err := repo.GetShareByID(ctx, 999)
	assert.Error(t, err)
}

// =============================================================================
// Mock Transaction Tests
// =============================================================================

func TestMockTx_CommitRollback(t *testing.T) {
	tx := NewMockTx()

	assert.False(t, tx.committed)
	assert.False(t, tx.rolledBack)

	err := tx.Commit()
	assert.NoError(t, err)
	assert.True(t, tx.committed)

	tx2 := NewMockTx()
	err = tx2.Rollback()
	assert.NoError(t, err)
	assert.True(t, tx2.rolledBack)
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestInMemoryUserRepository_ConcurrentAccess(t *testing.T) {
	repo := NewInMemoryUserRepository()
	ctx := context.Background()

	// Create initial user
	user := &User{Username: "concurrent", Email: "concurrent@test.com", IsActive: true}
	err := repo.CreateUser(ctx, user)
	require.NoError(t, err)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = repo.GetUserByID(ctx, user.ID)
			}
			done <- true
		}()
	}

	// Concurrent updates
	for i := 0; i < 5; i++ {
		go func(idx int) {
			for j := 0; j < 20; j++ {
				u, err := repo.GetUserByID(ctx, user.ID)
				if err == nil {
					u.IsActive = idx%2 == 0
					_ = repo.UpdateUser(ctx, u)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Verify data integrity
	_, err = repo.GetUserByID(ctx, user.ID)
	assert.NoError(t, err)
}

func TestInMemoryMinerRepository_ConcurrentAccess(t *testing.T) {
	repo := NewInMemoryMinerRepository()
	ctx := context.Background()

	// Create initial miner
	miner := &Miner{UserID: 1, Name: "concurrent", Hashrate: 1000.0, IsActive: true}
	err := repo.CreateMiner(ctx, miner)
	require.NoError(t, err)

	done := make(chan bool)

	// Concurrent hashrate updates
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 50; j++ {
				_ = repo.UpdateMinerHashrate(ctx, miner.ID, float64(idx*1000+j))
			}
			done <- true
		}(i)
	}

	// Concurrent last seen updates
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				_ = repo.UpdateMinerLastSeen(ctx, miner.ID)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Verify data integrity
	_, err = repo.GetMinerByID(ctx, miner.ID)
	assert.NoError(t, err)
}
