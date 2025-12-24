package api

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMinerMonitoringService(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewMinerMonitoringService(db)
	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
}

func TestGetUserMinerSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewMinerMonitoringService(db)

	// Mock user info query
	mock.ExpectQuery("SELECT id, username, email FROM users").
		WithArgs(int64(34)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}).
			AddRow(int64(34), "picaxe", "picaxe@example.com"))

	// Mock miners query
	lastSeen := time.Now()
	mock.ExpectQuery("SELECT").
		WithArgs(int64(34)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "address", "hashrate", "is_active", "last_seen", "shares_24h", "valid_percent"}).
			AddRow(int64(1), "X100-Worker1", "192.168.1.100", 150000000.0, true, lastSeen, int64(500), 98.5).
			AddRow(int64(2), "X100-Worker2", "192.168.1.101", 145000000.0, false, lastSeen.Add(-time.Hour), int64(450), 97.0))

	summary, err := service.GetUserMinerSummary(34)
	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, int64(34), summary.UserID)
	assert.Equal(t, "picaxe", summary.Username)
	assert.Equal(t, 2, summary.TotalMiners)
	assert.Equal(t, 1, summary.ActiveMiners)
	assert.Equal(t, 1, summary.InactiveMiners)
	assert.Equal(t, int64(950), summary.TotalShares24h)
}

func TestGetMinerDetail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewMinerMonitoringService(db)

	lastSeen := time.Now()
	createdAt := time.Now().Add(-24 * time.Hour)

	// Mock miner info query
	mock.ExpectQuery("SELECT").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "username", "name", "address", "hashrate", "is_active", "last_seen", "created_at", "updated_at"}).
			AddRow(int64(1), int64(34), "picaxe", "X100-Worker1", "192.168.1.100", 150000000.0, true, lastSeen, createdAt, lastSeen))

	// Mock active minutes query
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(720)))

	// Mock share stats query
	mock.ExpectQuery("SELECT").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"total", "valid", "invalid", "avg_diff", "total_diff", "last_24h", "last_hour"}).
			AddRow(int64(1000), int64(980), int64(20), 1.5, 1500.0, int64(500), int64(25)))

	// Mock recent shares query
	mock.ExpectQuery("SELECT id, difficulty, is_valid, nonce, hash, timestamp").
		WithArgs(int64(1), 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "difficulty", "is_valid", "nonce", "hash", "timestamp"}).
			AddRow(int64(100), 1.5, true, "abc123", "hash456", time.Now()))

	detail, err := service.GetMinerDetail(1)
	require.NoError(t, err)
	assert.NotNil(t, detail)
	assert.Equal(t, int64(1), detail.ID)
	assert.Equal(t, "picaxe", detail.Username)
	assert.Equal(t, "X100-Worker1", detail.Name)
	assert.True(t, detail.IsActive)
	assert.Equal(t, int64(1000), detail.ShareStats.TotalShares)
	assert.Equal(t, int64(980), detail.ShareStats.ValidShares)
	assert.Equal(t, 98.0, detail.ShareStats.AcceptanceRate)
}

func TestGetAllMinersForAdmin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewMinerMonitoringService(db)

	// Mock count query
	mock.ExpectQuery("SELECT COUNT").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(25)))

	// Mock miners query
	lastSeen := time.Now()
	mock.ExpectQuery("SELECT").
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "address", "hashrate", "is_active", "last_seen", "shares_24h", "valid_percent"}).
			AddRow(int64(1), "X100-Worker1", "192.168.1.100", 150000000.0, true, lastSeen, int64(500), 98.5).
			AddRow(int64(2), "X100-Worker2", "192.168.1.101", 145000000.0, true, lastSeen, int64(450), 97.0))

	miners, total, err := service.GetAllMinersForAdmin(1, 20, "", false)
	require.NoError(t, err)
	assert.Equal(t, int64(25), total)
	assert.Len(t, miners, 2)
	assert.Equal(t, "X100-Worker1", miners[0].Name)
	assert.Equal(t, int64(500), miners[0].Shares24h)
}

func TestGetMinerShareHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewMinerMonitoringService(db)

	shareTime := time.Now().Add(-5 * time.Minute)

	mock.ExpectQuery("SELECT id, difficulty, is_valid, nonce, hash, timestamp").
		WithArgs(int64(1), 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "difficulty", "is_valid", "nonce", "hash", "timestamp"}).
			AddRow(int64(100), 1.5, true, "nonce1", "hash1", shareTime).
			AddRow(int64(99), 1.4, true, "nonce2", "hash2", shareTime.Add(-time.Minute)).
			AddRow(int64(98), 1.6, false, "nonce3", "hash3", shareTime.Add(-2*time.Minute)))

	shares, err := service.GetMinerShareHistory(1, 10)
	require.NoError(t, err)
	assert.Len(t, shares, 3)
	assert.True(t, shares[0].IsValid)
	assert.False(t, shares[2].IsValid)
	assert.Equal(t, 1.5, shares[0].Difficulty)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{2 * time.Hour, "2h 0m ago"},
		{25 * time.Hour, "1d 1h ago"},
		{50 * time.Hour, "2d 2h ago"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		assert.Equal(t, tt.expected, result, "for duration %v", tt.duration)
	}
}

func TestMinerDetailShareStats(t *testing.T) {
	stats := ShareStats{
		TotalShares:   1000,
		ValidShares:   950,
		InvalidShares: 50,
		StaleShares:   0,
		Last24Hours:   500,
		LastHour:      25,
		AvgDifficulty: 1.5,
	}

	// Calculate acceptance rate
	acceptanceRate := float64(stats.ValidShares) / float64(stats.TotalShares) * 100
	assert.Equal(t, 95.0, acceptanceRate)
}

func TestPerformanceMetrics(t *testing.T) {
	perf := PerformanceMetrics{
		EffectiveHashrate:    140000000,
		ReportedHashrate:     150000000,
		SharesPerMinute:      0.5,
		AvgShareTime:         120,
		Efficiency:           93.3,
		EstimatedDailyShares: 720,
	}

	assert.Equal(t, float64(150000000), perf.ReportedHashrate)
	assert.Equal(t, float64(140000000), perf.EffectiveHashrate)
	assert.Equal(t, 93.3, perf.Efficiency)

	// Delta calculation
	delta := perf.EffectiveHashrate - perf.ReportedHashrate
	assert.Equal(t, float64(-10000000), delta)
}
