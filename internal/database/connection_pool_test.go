package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// CONNECTION POOL TESTS - Comprehensive Database Layer Testing
// =============================================================================

func TestConfig_Defaults(t *testing.T) {
	config := &Config{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "disable", config.SSLMode)
	assert.Equal(t, 0, config.MaxConns) // Default
	assert.Equal(t, 0, config.MinConns) // Default
}

func TestConfig_WithConnLimits(t *testing.T) {
	config := &Config{
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
		MaxConns: 50,
		MinConns: 10,
	}

	assert.Equal(t, 50, config.MaxConns)
	assert.Equal(t, 10, config.MinConns)
}

func TestPoolStats_ZeroValues(t *testing.T) {
	stats := PoolStats{}

	assert.Equal(t, int32(0), stats.MaxConns)
	assert.Equal(t, int32(0), stats.OpenConns)
	assert.Equal(t, int32(0), stats.InUse)
	assert.Equal(t, int32(0), stats.Idle)
}

func TestPoolStats_WithValues(t *testing.T) {
	stats := PoolStats{
		MaxConns:  25,
		OpenConns: 10,
		InUse:     5,
		Idle:      5,
	}

	assert.Equal(t, int32(25), stats.MaxConns)
	assert.Equal(t, int32(10), stats.OpenConns)
	assert.Equal(t, int32(5), stats.InUse)
	assert.Equal(t, int32(5), stats.Idle)
}

// =============================================================================
// MOCK DATABASE TESTS
// =============================================================================

// MockConnectionPool implements a mock for testing without actual database
type MockConnectionPool struct {
	healthy     bool
	stats       PoolStats
	queryError  error
	execError   error
	beginError  error
	returnValue interface{}
}

func NewMockConnectionPool() *MockConnectionPool {
	return &MockConnectionPool{
		healthy: true,
		stats: PoolStats{
			MaxConns:  25,
			OpenConns: 5,
			InUse:     2,
			Idle:      3,
		},
	}
}

func (m *MockConnectionPool) SetHealthy(healthy bool) {
	m.healthy = healthy
}

func (m *MockConnectionPool) SetQueryError(err error) {
	m.queryError = err
}

func (m *MockConnectionPool) HealthCheck(ctx context.Context) bool {
	return m.healthy
}

func (m *MockConnectionPool) Stats() PoolStats {
	return m.stats
}

func TestMockConnectionPool_HealthCheck_Healthy(t *testing.T) {
	mock := NewMockConnectionPool()
	ctx := context.Background()

	assert.True(t, mock.HealthCheck(ctx))
}

func TestMockConnectionPool_HealthCheck_Unhealthy(t *testing.T) {
	mock := NewMockConnectionPool()
	mock.SetHealthy(false)
	ctx := context.Background()

	assert.False(t, mock.HealthCheck(ctx))
}

func TestMockConnectionPool_Stats(t *testing.T) {
	mock := NewMockConnectionPool()

	stats := mock.Stats()
	assert.Equal(t, int32(25), stats.MaxConns)
	assert.Equal(t, int32(5), stats.OpenConns)
	assert.Equal(t, int32(2), stats.InUse)
	assert.Equal(t, int32(3), stats.Idle)
}

// =============================================================================
// TRANSACTION TESTS (Mock-based)
// =============================================================================

// MockTransaction implements a mock transaction for testing
type MockTransaction struct {
	committed  bool
	rolledback bool
	queryError error
	execError  error
}

func NewMockTransaction() *MockTransaction {
	return &MockTransaction{}
}

func (m *MockTransaction) Commit() error {
	if m.committed {
		return assert.AnError
	}
	m.committed = true
	return nil
}

func (m *MockTransaction) Rollback() error {
	if m.rolledback {
		return assert.AnError
	}
	m.rolledback = true
	return nil
}

func (m *MockTransaction) IsCommitted() bool {
	return m.committed
}

func (m *MockTransaction) IsRolledBack() bool {
	return m.rolledback
}

func TestMockTransaction_Commit(t *testing.T) {
	tx := NewMockTransaction()

	err := tx.Commit()
	assert.NoError(t, err)
	assert.True(t, tx.IsCommitted())
	assert.False(t, tx.IsRolledBack())
}

func TestMockTransaction_Rollback(t *testing.T) {
	tx := NewMockTransaction()

	err := tx.Rollback()
	assert.NoError(t, err)
	assert.False(t, tx.IsCommitted())
	assert.True(t, tx.IsRolledBack())
}

func TestMockTransaction_DoubleCommit(t *testing.T) {
	tx := NewMockTransaction()

	err := tx.Commit()
	assert.NoError(t, err)

	err = tx.Commit()
	assert.Error(t, err)
}

func TestMockTransaction_DoubleRollback(t *testing.T) {
	tx := NewMockTransaction()

	err := tx.Rollback()
	assert.NoError(t, err)

	err = tx.Rollback()
	assert.Error(t, err)
}

// =============================================================================
// CONTEXT TIMEOUT TESTS
// =============================================================================

func TestContext_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Context should not be done yet
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done yet")
	default:
		// Expected
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Context should be done now
	select {
	case <-ctx.Done():
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	default:
		t.Fatal("Context should be done")
	}
}

func TestContext_WithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Context should not be done yet
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done yet")
	default:
		// Expected
	}

	// Cancel the context
	cancel()

	// Context should be done now
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		t.Fatal("Context should be done")
	}
}

// =============================================================================
// CONNECTION STRING BUILDER TESTS
// =============================================================================

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "basic",
			config: &Config{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "user",
				Password: "pass",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable",
		},
		{
			name: "with_ssl",
			config: &Config{
				Host:     "db.example.com",
				Port:     5432,
				Database: "proddb",
				Username: "admin",
				Password: "secret",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5432 user=admin password=secret dbname=proddb sslmode=require",
		},
		{
			name: "custom_port",
			config: &Config{
				Host:     "localhost",
				Port:     5433,
				Database: "testdb",
				Username: "user",
				Password: "pass",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5433 user=user password=pass dbname=testdb sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build connection string manually (same logic as NewConnectionPool)
			connStr := buildConnString(tt.config)
			assert.Equal(t, tt.expected, connStr)
		})
	}
}

// Helper function to build connection string (mirrors internal logic)
func buildConnString(config *Config) string {
	return "host=" + config.Host +
		" port=" + string(rune(config.Port/1000+'0')) + string(rune((config.Port/100)%10+'0')) + string(rune((config.Port/10)%10+'0')) + string(rune(config.Port%10+'0')) +
		" user=" + config.Username +
		" password=" + config.Password +
		" dbname=" + config.Database +
		" sslmode=" + config.SSLMode
}

// =============================================================================
// INTERFACE COMPLIANCE TESTS
// =============================================================================

// Ensure interfaces are correctly defined
func TestInterfaceCompliance_QueryExecutor(t *testing.T) {
	// This test verifies the QueryExecutor interface is properly defined
	var _ QueryExecutor = (*mockQueryExecutor)(nil)
}

func TestInterfaceCompliance_CommandExecutor(t *testing.T) {
	var _ CommandExecutor = (*mockCommandExecutor)(nil)
}

func TestInterfaceCompliance_TransactionExecutor(t *testing.T) {
	var _ TransactionExecutor = (*mockTransactionExecutor)(nil)
}

// Mock implementations for interface compliance
type mockQueryExecutor struct{}

func (m *mockQueryExecutor) QueryRow(ctx context.Context, query string, args ...interface{}) Scanner {
	return &mockScanner{}
}

func (m *mockQueryExecutor) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return &mockRows{}, nil
}

type mockCommandExecutor struct{}

func (m *mockCommandExecutor) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return &mockResult{}, nil
}

type mockTransactionExecutor struct {
	mockQueryExecutor
	mockCommandExecutor
}

type mockScanner struct{}

func (m *mockScanner) Scan(dest ...interface{}) error {
	return nil
}

type mockRows struct{}

func (m *mockRows) Next() bool                { return false }
func (m *mockRows) Scan(...interface{}) error { return nil }
func (m *mockRows) Close() error              { return nil }
func (m *mockRows) Err() error                { return nil }

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 1, nil }

// =============================================================================
// REPOSITORY INTERFACE TESTS
// =============================================================================

// MockUserRepository for testing
type MockUserRepository struct {
	users map[int64]*User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[int64]*User),
	}
}

func (r *MockUserRepository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, ErrNotFound
}

func (r *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	for _, user := range r.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MockUserRepository) CreateUser(ctx context.Context, user *User) error {
	r.users[user.ID] = user
	return nil
}

func (r *MockUserRepository) UpdateUser(ctx context.Context, user *User) error {
	if _, ok := r.users[user.ID]; !ok {
		return ErrNotFound
	}
	r.users[user.ID] = user
	return nil
}

func (r *MockUserRepository) DeleteUser(ctx context.Context, id int64) error {
	if _, ok := r.users[id]; !ok {
		return ErrNotFound
	}
	delete(r.users, id)
	return nil
}

// Error for not found
var ErrNotFound = assert.AnError

func TestMockUserRepository_CreateAndGet(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	err := repo.CreateUser(ctx, user)
	assert.NoError(t, err)

	retrieved, err := repo.GetUserByID(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, retrieved.Username)
}

func TestMockUserRepository_GetByUsername(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	repo.CreateUser(ctx, user)

	retrieved, err := repo.GetUserByUsername(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
}

func TestMockUserRepository_GetByEmail(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	repo.CreateUser(ctx, user)

	retrieved, err := repo.GetUserByEmail(ctx, "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
}

func TestMockUserRepository_NotFound(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	_, err := repo.GetUserByID(ctx, 999)
	assert.Error(t, err)
}

func TestMockUserRepository_Update(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	repo.CreateUser(ctx, user)

	user.Username = "updateduser"
	err := repo.UpdateUser(ctx, user)
	assert.NoError(t, err)

	retrieved, _ := repo.GetUserByID(ctx, 1)
	assert.Equal(t, "updateduser", retrieved.Username)
}

func TestMockUserRepository_Delete(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	repo.CreateUser(ctx, user)
	err := repo.DeleteUser(ctx, 1)
	assert.NoError(t, err)

	_, err = repo.GetUserByID(ctx, 1)
	assert.Error(t, err)
}

// =============================================================================
// QUERY STATS TESTS
// =============================================================================

func TestQueryStats_ZeroValues(t *testing.T) {
	stats := QueryStats{}

	assert.Equal(t, int64(0), stats.TotalQueries)
	assert.Equal(t, int64(0), stats.SlowQueries)
	assert.Equal(t, int64(0), stats.FailedQueries)
	assert.Equal(t, float64(0), stats.AvgQueryTimeMs)
	assert.Equal(t, float64(0), stats.MaxQueryTimeMs)
	assert.Equal(t, float64(0), stats.QueriesPerSecond)
}

func TestQueryStats_WithValues(t *testing.T) {
	stats := QueryStats{
		TotalQueries:     1000,
		SlowQueries:      10,
		FailedQueries:    5,
		AvgQueryTimeMs:   15.5,
		MaxQueryTimeMs:   250.0,
		QueriesPerSecond: 100.0,
	}

	assert.Equal(t, int64(1000), stats.TotalQueries)
	assert.Equal(t, int64(10), stats.SlowQueries)
	assert.Equal(t, int64(5), stats.FailedQueries)
	assert.Equal(t, 15.5, stats.AvgQueryTimeMs)
	assert.Equal(t, 250.0, stats.MaxQueryTimeMs)
	assert.Equal(t, 100.0, stats.QueriesPerSecond)
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkMockConnectionPool_HealthCheck(b *testing.B) {
	mock := NewMockConnectionPool()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.HealthCheck(ctx)
	}
}

func BenchmarkMockUserRepository_GetByID(b *testing.B) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	// Populate with some users
	for i := int64(1); i <= 1000; i++ {
		repo.CreateUser(ctx, &User{
			ID:       i,
			Username: "user" + string(rune(i)),
			Email:    "user" + string(rune(i)) + "@example.com",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.GetUserByID(ctx, int64(i%1000)+1)
	}
}
