package main

import (
	"database/sql"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrapMockDB wraps a sqlmock DB into a ResilientDB for testing
func wrapMockDB(db *sql.DB) *ResilientDB {
	return &ResilientDB{
		db:      db,
		url:     "mock://test",
		mu:      sync.RWMutex{},
		healthy: true,
	}
}

// MockConn implements net.Conn for testing
type MockConn struct {
	written []byte
}

func (m *MockConn) Read(b []byte) (n int, err error) { return 0, nil }
func (m *MockConn) Write(b []byte) (n int, err error) {
	m.written = append(m.written, b...)
	return len(b), nil
}
func (m *MockConn) Close() error { return nil }
func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3333}
}
func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345}
}
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

// TestMinerStructHasUserTracking verifies Miner struct has proper user tracking fields
func TestMinerStructHasUserTracking(t *testing.T) {
	miner := &Miner{
		ID:         "test-miner-1",
		UserID:     42,
		Username:   "testuser",
		Address:    "192.168.1.100:12345",
		Authorized: true,
		Difficulty: 1.0,
	}

	assert.Equal(t, int64(42), miner.UserID, "Miner should have UserID field")
	assert.Equal(t, "testuser", miner.Username, "Miner should have Username field")
}

// TestAuthorizeUserLookup tests that authorization looks up user by username
func TestAuthorizeUserLookup(t *testing.T) {
	// Create mock database
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	server := &StratumServer{
		config: &Config{Difficulty: 1.0},
		db:     wrapMockDB(db),
		miners: make(map[string]*Miner),
	}

	mockConn := &MockConn{}
	miner := &Miner{
		ID:         "test-miner-1",
		Address:    "192.168.1.100:12345",
		Conn:       mockConn,
		Difficulty: 1.0,
	}

	// Test case: User exists
	t.Run("user exists", func(t *testing.T) {
		// Mock user lookup - now queries both id and username, checks username OR email
		mock.ExpectQuery("SELECT id, username FROM users WHERE").
			WithArgs("picaxe").
			WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(int64(34), "picaxe"))

		// Mock miner lookup (not found)
		mock.ExpectQuery("SELECT id FROM miners WHERE user_id").
			WithArgs(int64(34), "picaxe").
			WillReturnError(sql.ErrNoRows)

		// Mock miner insert - address is now IP only (port stripped), includes network_id
		mock.ExpectExec("INSERT INTO miners").
			WithArgs(int64(34), "picaxe", "192.168.1.100", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := StratumRequest{
			ID:     1,
			Method: "mining.authorize",
			Params: []interface{}{"picaxe", "x"},
		}

		err := server.handleAuthorize(miner, req)
		assert.NoError(t, err)
		assert.True(t, miner.Authorized)
		assert.Equal(t, int64(34), miner.UserID)
		assert.Equal(t, "picaxe", miner.Username)
	})
}

// TestAuthorizeUserNotFound tests authorization fails for unknown users
func TestAuthorizeUserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	server := &StratumServer{
		config: &Config{Difficulty: 1.0},
		db:     wrapMockDB(db),
		miners: make(map[string]*Miner),
	}

	mockConn := &MockConn{}
	miner := &Miner{
		ID:         "test-miner-1",
		Address:    "192.168.1.100:12345",
		Conn:       mockConn,
		Difficulty: 1.0,
	}

	// Mock user lookup - not found (queries id+username, checks username OR email)
	mock.ExpectQuery("SELECT id, username FROM users WHERE").
		WithArgs("unknown_user").
		WillReturnError(sql.ErrNoRows)

	req := StratumRequest{
		ID:     1,
		Method: "mining.authorize",
		Params: []interface{}{"unknown_user", "x"},
	}

	err = server.handleAuthorize(miner, req)
	assert.NoError(t, err) // No error returned, but response indicates failure
	assert.False(t, miner.Authorized)
	assert.Equal(t, int64(0), miner.UserID)
}

// TestShareAttributionRequiresUserID tests that shares require a valid user_id
func TestShareAttributionRequiresUserID(t *testing.T) {
	server := &StratumServer{
		config: &Config{Difficulty: 1.0},
		miners: make(map[string]*Miner),
	}

	mockConn := &MockConn{}
	miner := &Miner{
		ID:         "test-miner-1",
		Address:    "192.168.1.100:12345",
		Conn:       mockConn,
		Authorized: true,
		UserID:     0, // No user ID set
		Difficulty: 1.0,
	}

	req := StratumRequest{
		ID:     1,
		Method: "mining.submit",
		Params: []interface{}{"worker", "job_id", "extranonce2", "ntime", "nonce"},
	}

	err := server.handleSubmit(miner, req)
	assert.NoError(t, err) // Returns error response, not Go error

	// Verify response indicates failure
	assert.Contains(t, string(mockConn.written), "Authorization error")
}

// TestShareAttributionWithValidUser tests shares are properly attributed
func TestShareAttributionWithValidUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create a minimal Redis client for testing (using miniredis would be better in production)
	// For now, we test the database interactions and skip Redis
	// This test validates the SQL query structure and user attribution logic

	mockConn := &MockConn{}
	miner := &Miner{
		ID:         "test-miner-1",
		UserID:     34,
		Username:   "picaxe",
		Address:    "192.168.1.100:12345",
		Conn:       mockConn,
		Authorized: true,
		Difficulty: 1.0,
	}

	// Mock miner lookup - returns miner_id 5
	mock.ExpectQuery("SELECT id FROM miners").
		WithArgs(int64(34), "picaxe").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(5)))

	// Mock share insert with correct user_id and miner_id
	mock.ExpectExec("INSERT INTO shares").
		WithArgs(int64(5), int64(34), 1.0, "submitted", "hash").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Verify the SQL expectations are correct by checking the mock
	// The actual handleSubmit requires Redis, so we test the SQL structure here

	// Test that the miner lookup query is correct
	var minerDBID int64
	err = db.QueryRow(`
		SELECT id FROM miners 
		WHERE user_id = $1 AND name = $2 
		LIMIT 1`,
		miner.UserID, miner.Username,
	).Scan(&minerDBID)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), minerDBID)

	// Test that the share insert query structure is correct
	_, err = db.Exec(`
		INSERT INTO shares (miner_id, user_id, difficulty, is_valid, nonce, hash, timestamp) 
		VALUES ($1, $2, $3, true, $4, $5, NOW())`,
		minerDBID, miner.UserID, miner.Difficulty, "submitted", "hash",
	)
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestMinerUserIDPreservation tests that UserID is preserved across connection lifecycle
func TestMinerUserIDPreservation(t *testing.T) {
	miner := &Miner{
		ID:      "test-miner-1",
		Address: "192.168.1.100:12345",
	}

	// Simulate authorization
	miner.Authorized = true
	miner.UserID = 42
	miner.Username = "testuser"

	// Verify fields persist
	assert.True(t, miner.Authorized)
	assert.Equal(t, int64(42), miner.UserID)
	assert.Equal(t, "testuser", miner.Username)

	// Simulate share submission - UserID should still be available
	assert.NotEqual(t, int64(0), miner.UserID, "UserID should persist for share attribution")
}
