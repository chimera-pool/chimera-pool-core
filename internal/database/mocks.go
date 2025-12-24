package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

// =============================================================================
// Mock Implementations for Database Interfaces
// Enables comprehensive unit testing without requiring a live database
// =============================================================================

// MockScanner implements the Scanner interface for testing
type MockScanner struct {
	mock.Mock
	values []interface{}
	err    error
}

func NewMockScanner(values []interface{}, err error) *MockScanner {
	return &MockScanner{values: values, err: err}
}

func (m *MockScanner) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}
	for i, d := range dest {
		if i < len(m.values) {
			switch v := d.(type) {
			case *int64:
				if val, ok := m.values[i].(int64); ok {
					*v = val
				}
			case *string:
				if val, ok := m.values[i].(string); ok {
					*v = val
				}
			case *bool:
				if val, ok := m.values[i].(bool); ok {
					*v = val
				}
			case *float64:
				if val, ok := m.values[i].(float64); ok {
					*v = val
				}
			case *time.Time:
				if val, ok := m.values[i].(time.Time); ok {
					*v = val
				}
			}
		}
	}
	return nil
}

// MockRows implements the Rows interface for testing
type MockRows struct {
	mock.Mock
	data    [][]interface{}
	current int
	closed  bool
	err     error
}

func NewMockRows(data [][]interface{}) *MockRows {
	return &MockRows{data: data, current: -1}
}

func (m *MockRows) Next() bool {
	m.current++
	return m.current < len(m.data)
}

func (m *MockRows) Scan(dest ...interface{}) error {
	if m.current >= len(m.data) {
		return fmt.Errorf("no more rows")
	}
	row := m.data[m.current]
	for i, d := range dest {
		if i < len(row) {
			switch v := d.(type) {
			case *int64:
				if val, ok := row[i].(int64); ok {
					*v = val
				}
			case *string:
				if val, ok := row[i].(string); ok {
					*v = val
				}
			case *bool:
				if val, ok := row[i].(bool); ok {
					*v = val
				}
			case *float64:
				if val, ok := row[i].(float64); ok {
					*v = val
				}
			case *time.Time:
				if val, ok := row[i].(time.Time); ok {
					*v = val
				}
			}
		}
	}
	return nil
}

func (m *MockRows) Close() error {
	m.closed = true
	return nil
}

func (m *MockRows) Err() error {
	return m.err
}

// MockResult implements the Result interface for testing
type MockResult struct {
	lastID       int64
	rowsAffected int64
	lastIDErr    error
	rowsErr      error
}

func NewMockResult(lastID, rowsAffected int64) *MockResult {
	return &MockResult{lastID: lastID, rowsAffected: rowsAffected}
}

func (m *MockResult) LastInsertId() (int64, error) {
	return m.lastID, m.lastIDErr
}

func (m *MockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, m.rowsErr
}

// MockQueryExecutor implements QueryExecutor for testing
type MockQueryExecutor struct {
	mock.Mock
}

func (m *MockQueryExecutor) QueryRow(ctx context.Context, query string, args ...interface{}) Scanner {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(Scanner)
}

func (m *MockQueryExecutor) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	callArgs := m.Called(ctx, query, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(Rows), callArgs.Error(1)
}

// MockCommandExecutor implements CommandExecutor for testing
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	callArgs := m.Called(ctx, query, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(Result), callArgs.Error(1)
}

// MockTx implements the Tx interface for testing
type MockTx struct {
	MockQueryExecutor
	MockCommandExecutor
	committed  bool
	rolledBack bool
}

func NewMockTx() *MockTx {
	return &MockTx{}
}

func (m *MockTx) Commit() error {
	m.committed = true
	return nil
}

func (m *MockTx) Rollback() error {
	m.rolledBack = true
	return nil
}

func (m *MockTx) QueryRow(ctx context.Context, query string, args ...interface{}) Scanner {
	return m.MockQueryExecutor.QueryRow(ctx, query, args...)
}

func (m *MockTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return m.MockQueryExecutor.Query(ctx, query, args...)
}

func (m *MockTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return m.MockCommandExecutor.Exec(ctx, query, args...)
}

// MockTransactionManager implements TransactionManager for testing
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) Begin(ctx context.Context) (Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(Tx), args.Error(1)
}

func (m *MockTransactionManager) BeginReadOnly(ctx context.Context) (Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(Tx), args.Error(1)
}

// =============================================================================
// In-Memory Repository Implementations
// Provides a complete in-memory database for testing
// =============================================================================

// InMemoryUserRepository implements UserRepository for testing
type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  map[int64]*User
	nextID int64
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

func (r *InMemoryUserRepository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	// Return a copy
	copy := *user
	return &copy, nil
}

func (r *InMemoryUserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Username == username {
			copy := *user
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *InMemoryUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Email == email {
			copy := *user
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *InMemoryUserRepository) CreateUser(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	for _, existing := range r.users {
		if existing.Username == user.Username {
			return fmt.Errorf("username already exists")
		}
		if existing.Email == user.Email {
			return fmt.Errorf("email already exists")
		}
	}

	user.ID = r.nextID
	r.nextID++
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	copy := *user
	r.users[user.ID] = &copy
	return nil
}

func (r *InMemoryUserRepository) UpdateUser(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}

	user.UpdatedAt = time.Now()
	copy := *user
	r.users[user.ID] = &copy
	return nil
}

func (r *InMemoryUserRepository) DeleteUser(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[id]; !exists {
		return fmt.Errorf("user not found")
	}

	delete(r.users, id)
	return nil
}

// InMemoryMinerRepository implements MinerRepository for testing
type InMemoryMinerRepository struct {
	mu     sync.RWMutex
	miners map[int64]*Miner
	nextID int64
}

func NewInMemoryMinerRepository() *InMemoryMinerRepository {
	return &InMemoryMinerRepository{
		miners: make(map[int64]*Miner),
		nextID: 1,
	}
}

func (r *InMemoryMinerRepository) GetMinerByID(ctx context.Context, id int64) (*Miner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	miner, exists := r.miners[id]
	if !exists {
		return nil, fmt.Errorf("miner not found")
	}
	copy := *miner
	return &copy, nil
}

func (r *InMemoryMinerRepository) GetMinersByUserID(ctx context.Context, userID int64) ([]*Miner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Miner
	for _, miner := range r.miners {
		if miner.UserID == userID {
			copy := *miner
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (r *InMemoryMinerRepository) GetActiveMinerCount(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, miner := range r.miners {
		if miner.IsActive {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryMinerRepository) CreateMiner(ctx context.Context, miner *Miner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	miner.ID = r.nextID
	r.nextID++
	miner.CreatedAt = time.Now()
	miner.UpdatedAt = time.Now()
	miner.LastSeen = time.Now()

	copy := *miner
	r.miners[miner.ID] = &copy
	return nil
}

func (r *InMemoryMinerRepository) UpdateMiner(ctx context.Context, miner *Miner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.miners[miner.ID]; !exists {
		return fmt.Errorf("miner not found")
	}

	miner.UpdatedAt = time.Now()
	copy := *miner
	r.miners[miner.ID] = &copy
	return nil
}

func (r *InMemoryMinerRepository) UpdateMinerLastSeen(ctx context.Context, minerID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	miner, exists := r.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found")
	}

	miner.LastSeen = time.Now()
	miner.UpdatedAt = time.Now()
	return nil
}

func (r *InMemoryMinerRepository) UpdateMinerHashrate(ctx context.Context, minerID int64, hashrate float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	miner, exists := r.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found")
	}

	miner.Hashrate = hashrate
	miner.UpdatedAt = time.Now()
	return nil
}

// InMemoryShareRepository implements ShareRepository for testing
type InMemoryShareRepository struct {
	mu     sync.RWMutex
	shares map[int64]*Share
	nextID int64
}

func NewInMemoryShareRepository() *InMemoryShareRepository {
	return &InMemoryShareRepository{
		shares: make(map[int64]*Share),
		nextID: 1,
	}
}

func (r *InMemoryShareRepository) GetShareByID(ctx context.Context, id int64) (*Share, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	share, exists := r.shares[id]
	if !exists {
		return nil, fmt.Errorf("share not found")
	}
	copy := *share
	return &copy, nil
}

func (r *InMemoryShareRepository) GetSharesByMinerID(ctx context.Context, minerID int64, limit int) ([]*Share, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Share
	for _, share := range r.shares {
		if share.MinerID == minerID {
			copy := *share
			result = append(result, &copy)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *InMemoryShareRepository) GetSharesByUserID(ctx context.Context, userID int64, limit int) ([]*Share, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Share
	for _, share := range r.shares {
		if share.UserID == userID {
			copy := *share
			result = append(result, &copy)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *InMemoryShareRepository) GetValidShareCount(ctx context.Context, minerID int64, since time.Time) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, share := range r.shares {
		if share.MinerID == minerID && share.IsValid && share.Timestamp.After(since) {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryShareRepository) CreateShare(ctx context.Context, share *Share) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	share.ID = r.nextID
	r.nextID++
	share.Timestamp = time.Now()

	copy := *share
	r.shares[share.ID] = &copy
	return nil
}

func (r *InMemoryShareRepository) CreateShareBatch(ctx context.Context, shares []*Share) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, share := range shares {
		share.ID = r.nextID
		r.nextID++
		share.Timestamp = time.Now()

		copy := *share
		r.shares[share.ID] = &copy
	}
	return nil
}
