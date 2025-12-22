package stratum

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkerName(t *testing.T) {
	tests := []struct {
		name          string
		workerName    string
		wantUsername  string
		wantMinerName string
		wantErr       error
	}{
		{
			name:          "username.worker format",
			workerName:    "alice.rig1",
			wantUsername:  "alice",
			wantMinerName: "rig1",
			wantErr:       nil,
		},
		{
			name:          "username only",
			workerName:    "bob",
			wantUsername:  "bob",
			wantMinerName: "default",
			wantErr:       nil,
		},
		{
			name:          "username with multiple dots",
			workerName:    "charlie.worker.extra",
			wantUsername:  "charlie",
			wantMinerName: "worker.extra",
			wantErr:       nil,
		},
		{
			name:          "empty string",
			workerName:    "",
			wantUsername:  "",
			wantMinerName: "",
			wantErr:       ErrInvalidWorkerName,
		},
		{
			name:          "single character username",
			workerName:    "a",
			wantUsername:  "",
			wantMinerName: "",
			wantErr:       ErrInvalidWorkerName,
		},
		{
			name:          "whitespace only",
			workerName:    "   ",
			wantUsername:  "",
			wantMinerName: "",
			wantErr:       ErrInvalidWorkerName,
		},
		{
			name:          "username with trailing dot",
			workerName:    "dave.",
			wantUsername:  "dave",
			wantMinerName: "default",
			wantErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, minerName, err := ParseWorkerName(tt.workerName)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantUsername, username)
			assert.Equal(t, tt.wantMinerName, minerName)
		})
	}
}

func TestMockAuthenticator_Authenticate(t *testing.T) {
	t.Run("AllowAll mode accepts any worker", func(t *testing.T) {
		auth := NewMockAuthenticator()
		auth.AllowAll = true

		result, err := auth.Authenticate(context.Background(), "testuser.rig1", "")
		require.NoError(t, err)
		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "rig1", result.WorkerName)
		assert.True(t, result.IsNewMiner)
		assert.True(t, result.Permissions.CanSubmitShares)
	})

	t.Run("strict mode rejects unknown user", func(t *testing.T) {
		auth := NewMockAuthenticator()
		auth.AllowAll = false

		_, err := auth.Authenticate(context.Background(), "unknown.rig1", "")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("strict mode accepts known user", func(t *testing.T) {
		auth := NewMockAuthenticator()
		auth.AllowAll = false

		userID := int64(12345)
		auth.AddUser(&UserInfo{
			ID:       userID,
			Username: "validuser",
			IsActive: true,
		})

		result, err := auth.Authenticate(context.Background(), "validuser.miner1", "")
		require.NoError(t, err)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, "validuser", result.Username)
		assert.Equal(t, "miner1", result.WorkerName)
	})

	t.Run("returns existing miner info", func(t *testing.T) {
		auth := NewMockAuthenticator()
		auth.AllowAll = false

		userID := int64(12345)
		minerID := int64(67890)
		auth.AddUser(&UserInfo{
			ID:       userID,
			Username: "miner",
			IsActive: true,
		})
		auth.AddMiner(&MinerInfo{
			ID:         minerID,
			UserID:     userID,
			WorkerName: "existing",
			LastSeen:   time.Now(),
			IsActive:   true,
		})

		result, err := auth.Authenticate(context.Background(), "miner.existing", "")
		require.NoError(t, err)
		assert.Equal(t, minerID, result.MinerID)
		assert.False(t, result.IsNewMiner)
	})

	t.Run("rejects invalid worker name", func(t *testing.T) {
		auth := NewMockAuthenticator()

		_, err := auth.Authenticate(context.Background(), "", "")
		assert.ErrorIs(t, err, ErrInvalidWorkerName)
	})
}

func TestDefaultMinerPermissions(t *testing.T) {
	perms := DefaultMinerPermissions()

	assert.True(t, perms.CanSubmitShares)
	assert.True(t, perms.CanReceiveJobs)
	assert.Greater(t, perms.MaxDifficulty, float64(0))
	assert.Greater(t, perms.MinDifficulty, float64(0))
}

func TestCachedAuthenticator_Config(t *testing.T) {
	config := DefaultCachedAuthenticatorConfig()

	assert.Greater(t, config.UserCacheTTL, time.Duration(0))
	assert.Greater(t, config.MinerCacheTTL, time.Duration(0))
}

// mockLookup implements MinerLookup for testing
type mockLookup struct {
	users  map[string]*UserInfo
	miners map[string]*MinerInfo
}

func (m *mockLookup) GetMinerByWorkerName(ctx context.Context, userID int64, workerName string) (*MinerInfo, error) {
	key := fmt.Sprintf("%d:%s", userID, workerName)
	if miner, ok := m.miners[key]; ok {
		return miner, nil
	}
	return nil, ErrMinerNotFound
}

func (m *mockLookup) GetUserByUsername(ctx context.Context, username string) (*UserInfo, error) {
	if user, ok := m.users[username]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

// mockRegistrar implements MinerRegistrar for testing
type mockRegistrar struct {
	registeredMiners []*MinerInfo
}

func (m *mockRegistrar) RegisterMiner(ctx context.Context, userID int64, workerName string, ipAddress string) (*MinerInfo, error) {
	miner := &MinerInfo{
		ID:         time.Now().UnixNano(),
		UserID:     userID,
		WorkerName: workerName,
		IPAddress:  ipAddress,
		LastSeen:   time.Now(),
		IsActive:   true,
	}
	m.registeredMiners = append(m.registeredMiners, miner)
	return miner, nil
}

func (m *mockRegistrar) UpdateMinerLastSeen(ctx context.Context, minerID int64) error {
	return nil
}

func TestCachedAuthenticator_Authenticate(t *testing.T) {
	t.Run("authenticates valid user and creates miner", func(t *testing.T) {
		userID := int64(100001)
		lookup := &mockLookup{
			users: map[string]*UserInfo{
				"testuser": {
					ID:       userID,
					Username: "testuser",
					IsActive: true,
				},
			},
			miners: make(map[string]*MinerInfo),
		}
		registrar := &mockRegistrar{}

		auth := NewCachedAuthenticator(lookup, registrar, DefaultCachedAuthenticatorConfig())

		result, err := auth.Authenticate(context.Background(), "testuser.worker1", "")
		require.NoError(t, err)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "worker1", result.WorkerName)
		assert.True(t, result.IsNewMiner)
		assert.Len(t, registrar.registeredMiners, 1)
	})

	t.Run("returns existing miner without registration", func(t *testing.T) {
		userID := int64(100002)
		minerID := int64(200002)
		lookup := &mockLookup{
			users: map[string]*UserInfo{
				"existing": {
					ID:       userID,
					Username: "existing",
					IsActive: true,
				},
			},
			miners: map[string]*MinerInfo{
				fmt.Sprintf("%d:miner1", userID): {
					ID:         minerID,
					UserID:     userID,
					WorkerName: "miner1",
					IsActive:   true,
				},
			},
		}
		registrar := &mockRegistrar{}

		auth := NewCachedAuthenticator(lookup, registrar, DefaultCachedAuthenticatorConfig())

		result, err := auth.Authenticate(context.Background(), "existing.miner1", "")
		require.NoError(t, err)
		assert.Equal(t, minerID, result.MinerID)
		assert.False(t, result.IsNewMiner)
		assert.Len(t, registrar.registeredMiners, 0)
	})

	t.Run("rejects disabled user", func(t *testing.T) {
		lookup := &mockLookup{
			users: map[string]*UserInfo{
				"disabled": {
					ID:       int64(100003),
					Username: "disabled",
					IsActive: false,
				},
			},
			miners: make(map[string]*MinerInfo),
		}
		registrar := &mockRegistrar{}

		auth := NewCachedAuthenticator(lookup, registrar, DefaultCachedAuthenticatorConfig())

		_, err := auth.Authenticate(context.Background(), "disabled.worker", "")
		assert.ErrorIs(t, err, ErrUserDisabled)
	})

	t.Run("caches user lookups", func(t *testing.T) {
		userID := int64(100004)
		lookupCount := 0
		lookup := &countingLookup{
			users: map[string]*UserInfo{
				"cached": {
					ID:       userID,
					Username: "cached",
					IsActive: true,
				},
			},
			miners:      make(map[string]*MinerInfo),
			lookupCount: &lookupCount,
		}
		registrar := &mockRegistrar{}

		auth := NewCachedAuthenticator(lookup, registrar, DefaultCachedAuthenticatorConfig())

		// First call - should hit database
		_, err := auth.Authenticate(context.Background(), "cached.w1", "")
		require.NoError(t, err)
		assert.Equal(t, 1, lookupCount)

		// Second call - should use cache
		_, err = auth.Authenticate(context.Background(), "cached.w2", "")
		require.NoError(t, err)
		assert.Equal(t, 1, lookupCount) // Still 1, cache hit
	})
}

// countingLookup tracks the number of lookups
type countingLookup struct {
	users       map[string]*UserInfo
	miners      map[string]*MinerInfo
	lookupCount *int
}

func (m *countingLookup) GetMinerByWorkerName(ctx context.Context, userID int64, workerName string) (*MinerInfo, error) {
	key := fmt.Sprintf("%d:%s", userID, workerName)
	if miner, ok := m.miners[key]; ok {
		return miner, nil
	}
	return nil, ErrMinerNotFound
}

func (m *countingLookup) GetUserByUsername(ctx context.Context, username string) (*UserInfo, error) {
	*m.lookupCount++
	if user, ok := m.users[username]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func TestAuthResult_Permissions(t *testing.T) {
	result := &AuthResult{
		UserID:      int64(99999),
		MinerID:     int64(88888),
		Username:    "test",
		WorkerName:  "worker",
		IsNewMiner:  true,
		Permissions: DefaultMinerPermissions(),
	}

	assert.True(t, result.Permissions.CanSubmitShares)
	assert.True(t, result.Permissions.CanReceiveJobs)
}
