package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) StoreMetric(ctx context.Context, metric *Metric) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockRepository) GetMetrics(ctx context.Context, name string, start, end time.Time) ([]*Metric, error) {
	args := m.Called(ctx, name, start, end)
	return args.Get(0).([]*Metric), args.Error(1)
}

func (m *MockRepository) CreateAlert(ctx context.Context, alert *Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) UpdateAlert(ctx context.Context, alert *Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*Alert), args.Error(1)
}

func (m *MockRepository) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockRepository) GetAlertRules(ctx context.Context) ([]*AlertRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*AlertRule), args.Error(1)
}

func (m *MockRepository) CreateDashboard(ctx context.Context, dashboard *Dashboard) error {
	args := m.Called(ctx, dashboard)
	return args.Error(0)
}

func (m *MockRepository) GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*Dashboard), args.Error(1)
}

func (m *MockRepository) StorePerformanceMetrics(ctx context.Context, metrics *PerformanceMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockRepository) GetPerformanceMetrics(ctx context.Context, start, end time.Time) ([]*PerformanceMetrics, error) {
	args := m.Called(ctx, start, end)
	return args.Get(0).([]*PerformanceMetrics), args.Error(1)
}

func (m *MockRepository) StoreMinerMetrics(ctx context.Context, metrics *MinerMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockRepository) GetMinerMetrics(ctx context.Context, minerID uuid.UUID, start, end time.Time) ([]*MinerMetrics, error) {
	args := m.Called(ctx, minerID, start, end)
	return args.Get(0).([]*MinerMetrics), args.Error(1)
}

func (m *MockRepository) StorePoolMetrics(ctx context.Context, metrics *PoolMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockRepository) GetPoolMetrics(ctx context.Context, start, end time.Time) ([]*PoolMetrics, error) {
	args := m.Called(ctx, start, end)
	return args.Get(0).([]*PoolMetrics), args.Error(1)
}

func (m *MockRepository) CreateAlertChannel(ctx context.Context, channel *AlertChannel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockRepository) GetAlertChannels(ctx context.Context) ([]*AlertChannel, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*AlertChannel), args.Error(1)
}

func (m *MockRepository) RecordNotification(ctx context.Context, notification *Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

// MockPrometheusClient is a mock Prometheus client
type MockPrometheusClient struct {
	mock.Mock
}

func (m *MockPrometheusClient) RecordCounter(name string, labels map[string]string, value float64) error {
	args := m.Called(name, labels, value)
	return args.Error(0)
}

func (m *MockPrometheusClient) RecordGauge(name string, labels map[string]string, value float64) error {
	args := m.Called(name, labels, value)
	return args.Error(0)
}

func (m *MockPrometheusClient) RecordHistogram(name string, labels map[string]string, value float64) error {
	args := m.Called(name, labels, value)
	return args.Error(0)
}

func (m *MockPrometheusClient) Query(query string) (float64, error) {
	args := m.Called(query)
	return args.Get(0).(float64), args.Error(1)
}

func TestMonitoringService_RecordMetric(t *testing.T) {
	tests := []struct {
		name      string
		metric    *Metric
		setupMock func(*MockRepository, *MockPrometheusClient)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "should record counter metric successfully",
			metric: &Metric{
				Name:      "pool_shares_total",
				Value:     100.0,
				Labels:    map[string]string{"miner": "test"},
				Timestamp: time.Now(),
				Type:      "counter",
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				mr.On("StoreMetric", mock.Anything, mock.AnythingOfType("*monitoring.Metric")).Return(nil)
				mp.On("RecordCounter", "pool_shares_total", map[string]string{"miner": "test"}, 100.0).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "should record gauge metric successfully",
			metric: &Metric{
				Name:      "pool_hashrate",
				Value:     1000000.0,
				Labels:    map[string]string{"pool": "main"},
				Timestamp: time.Now(),
				Type:      "gauge",
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				mr.On("StoreMetric", mock.Anything, mock.AnythingOfType("*monitoring.Metric")).Return(nil)
				mp.On("RecordGauge", "pool_hashrate", map[string]string{"pool": "main"}, 1000000.0).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "should fail with invalid metric type",
			metric: &Metric{
				Name:      "test_metric",
				Value:     100.0,
				Labels:    map[string]string{},
				Timestamp: time.Now(),
				Type:      "invalid",
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {},
			wantErr:   true,
			errMsg:    "unsupported metric type",
		},
		{
			name: "should fail with empty metric name",
			metric: &Metric{
				Name:      "",
				Value:     100.0,
				Labels:    map[string]string{},
				Timestamp: time.Now(),
				Type:      "counter",
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {},
			wantErr:   true,
			errMsg:    "metric name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo, mockPrometheus)

			service := NewService(mockRepo, mockPrometheus)

			err := service.RecordMetric(context.Background(), tt.metric)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockPrometheus.AssertExpectations(t)
		})
	}
}

func TestMonitoringService_CreateAlert(t *testing.T) {
	tests := []struct {
		name        string
		alertName   string
		description string
		severity    string
		labels      map[string]string
		setupMock   func(*MockRepository)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "should create alert successfully",
			alertName:   "High CPU Usage",
			description: "CPU usage is above 90%",
			severity:    "warning",
			labels:      map[string]string{"component": "pool"},
			setupMock: func(m *MockRepository) {
				m.On("CreateAlert", mock.Anything, mock.AnythingOfType("*monitoring.Alert")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "should fail with empty alert name",
			alertName:   "",
			description: "CPU usage is above 90%",
			severity:    "warning",
			labels:      map[string]string{},
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "alert name cannot be empty",
		},
		{
			name:        "should fail with invalid severity",
			alertName:   "High CPU Usage",
			description: "CPU usage is above 90%",
			severity:    "invalid",
			labels:      map[string]string{},
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "invalid severity level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo, mockPrometheus)

			alert, err := service.CreateAlert(context.Background(), tt.alertName, tt.description, tt.severity, tt.labels)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, alert)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, alert)
				assert.Equal(t, tt.alertName, alert.Name)
				assert.Equal(t, tt.description, alert.Description)
				assert.Equal(t, tt.severity, alert.Severity)
				assert.Equal(t, "active", alert.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMonitoringService_CreateAlertRule(t *testing.T) {
	tests := []struct {
		name      string
		ruleName  string
		query     string
		condition string
		threshold float64
		duration  string
		severity  string
		setupMock func(*MockRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "should create alert rule successfully",
			ruleName:  "High CPU Alert",
			query:     "cpu_usage",
			condition: ">",
			threshold: 90.0,
			duration:  "5m",
			severity:  "warning",
			setupMock: func(m *MockRepository) {
				m.On("CreateAlertRule", mock.Anything, mock.AnythingOfType("*monitoring.AlertRule")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "should fail with invalid condition",
			ruleName:  "High CPU Alert",
			query:     "cpu_usage",
			condition: "invalid",
			threshold: 90.0,
			duration:  "5m",
			severity:  "warning",
			setupMock: func(m *MockRepository) {},
			wantErr:   true,
			errMsg:    "invalid condition",
		},
		{
			name:      "should fail with invalid duration",
			ruleName:  "High CPU Alert",
			query:     "cpu_usage",
			condition: ">",
			threshold: 90.0,
			duration:  "invalid",
			severity:  "warning",
			setupMock: func(m *MockRepository) {},
			wantErr:   true,
			errMsg:    "invalid duration format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo, mockPrometheus)

			rule, err := service.CreateAlertRule(context.Background(), tt.ruleName, tt.query, tt.condition, tt.threshold, tt.duration, tt.severity)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, rule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rule)
				assert.Equal(t, tt.ruleName, rule.Name)
				assert.Equal(t, tt.query, rule.Query)
				assert.Equal(t, tt.condition, rule.Condition)
				assert.Equal(t, tt.threshold, rule.Threshold)
				assert.Equal(t, tt.duration, rule.Duration)
				assert.Equal(t, tt.severity, rule.Severity)
				assert.True(t, rule.IsActive)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMonitoringService_EvaluateAlertRules(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func(*MockRepository, *MockPrometheusClient)
		expectAlerts int
		wantErr      bool
	}{
		{
			name: "should evaluate rules and create alerts",
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				rules := []*AlertRule{
					{
						ID:        uuid.New(),
						Name:      "High CPU",
						Query:     "cpu_usage",
						Condition: ">",
						Threshold: 90.0,
						Duration:  "5m",
						Severity:  "warning",
						IsActive:  true,
					},
				}
				mr.On("GetAlertRules", mock.Anything).Return(rules, nil)
				mp.On("Query", "cpu_usage").Return(95.0, nil)
				mr.On("CreateAlert", mock.Anything, mock.AnythingOfType("*monitoring.Alert")).Return(nil)
			},
			expectAlerts: 1,
			wantErr:      false,
		},
		{
			name: "should not create alerts when conditions not met",
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				rules := []*AlertRule{
					{
						ID:        uuid.New(),
						Name:      "High CPU",
						Query:     "cpu_usage",
						Condition: ">",
						Threshold: 90.0,
						Duration:  "5m",
						Severity:  "warning",
						IsActive:  true,
					},
				}
				mr.On("GetAlertRules", mock.Anything).Return(rules, nil)
				mp.On("Query", "cpu_usage").Return(80.0, nil)
			},
			expectAlerts: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo, mockPrometheus)

			service := NewService(mockRepo, mockPrometheus)

			alerts, err := service.EvaluateAlertRules(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, alerts, tt.expectAlerts)
			}

			mockRepo.AssertExpectations(t)
			mockPrometheus.AssertExpectations(t)
		})
	}
}

func TestMonitoringService_CreateDashboard(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name        string
		dashName    string
		description string
		config      string
		isPublic    bool
		createdBy   uuid.UUID
		setupMock   func(*MockRepository)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "should create dashboard successfully",
			dashName:    "Pool Overview",
			description: "Main pool monitoring dashboard",
			config:      `{"panels": [{"title": "Hashrate", "type": "graph"}]}`,
			isPublic:    true,
			createdBy:   userID,
			setupMock: func(m *MockRepository) {
				m.On("CreateDashboard", mock.Anything, mock.AnythingOfType("*monitoring.Dashboard")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "should fail with empty dashboard name",
			dashName:    "",
			description: "Main pool monitoring dashboard",
			config:      `{"panels": []}`,
			isPublic:    true,
			createdBy:   userID,
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "dashboard name cannot be empty",
		},
		{
			name:        "should fail with invalid JSON config",
			dashName:    "Pool Overview",
			description: "Main pool monitoring dashboard",
			config:      `invalid json`,
			isPublic:    true,
			createdBy:   userID,
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "invalid JSON configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo, mockPrometheus)

			dashboard, err := service.CreateDashboard(context.Background(), tt.dashName, tt.description, tt.config, tt.isPublic, tt.createdBy)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, dashboard)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dashboard)
				assert.Equal(t, tt.dashName, dashboard.Name)
				assert.Equal(t, tt.description, dashboard.Description)
				assert.Equal(t, tt.config, dashboard.Config)
				assert.Equal(t, tt.isPublic, dashboard.IsPublic)
				assert.Equal(t, tt.createdBy, dashboard.CreatedBy)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// ADDITIONAL COMPREHENSIVE TESTS FOR 60%+ COVERAGE
// =============================================================================

func TestMonitoringService_RecordHistogramMetric(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	metric := &Metric{
		Name:      "request_latency",
		Value:     0.5,
		Labels:    map[string]string{"endpoint": "/api"},
		Timestamp: time.Now(),
		Type:      "histogram",
	}

	mockRepo.On("StoreMetric", mock.Anything, mock.AnythingOfType("*monitoring.Metric")).Return(nil)
	mockPrometheus.On("RecordHistogram", "request_latency", map[string]string{"endpoint": "/api"}, 0.5).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	err := service.RecordMetric(context.Background(), metric)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockPrometheus.AssertExpectations(t)
}

func TestMonitoringService_ResolveAlert(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	alertID := uuid.New()
	mockRepo.On("UpdateAlert", mock.Anything, mock.AnythingOfType("*monitoring.Alert")).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	err := service.ResolveAlert(context.Background(), alertID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_RecordPerformanceMetrics(t *testing.T) {
	tests := []struct {
		name      string
		metrics   *PerformanceMetrics
		setupMock func(*MockRepository, *MockPrometheusClient)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "should record performance metrics successfully",
			metrics: &PerformanceMetrics{
				Timestamp:       time.Now(),
				CPUUsage:        75.5,
				MemoryUsage:     60.0,
				DiskUsage:       40.0,
				ActiveMiners:    100,
				TotalHashrate:   1000000,
				SharesPerSecond: 50.0,
				BlocksFound:     5,
				Uptime:          86400,
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				mr.On("StorePerformanceMetrics", mock.Anything, mock.AnythingOfType("*monitoring.PerformanceMetrics")).Return(nil)
				mp.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mp.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "should fail with nil metrics",
			metrics:   nil,
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {},
			wantErr:   true,
			errMsg:    "cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo, mockPrometheus)

			service := NewService(mockRepo, mockPrometheus)
			err := service.RecordPerformanceMetrics(context.Background(), tt.metrics)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMonitoringService_RecordMinerMetrics(t *testing.T) {
	minerID := uuid.New()

	tests := []struct {
		name      string
		metrics   *MinerMetrics
		setupMock func(*MockRepository, *MockPrometheusClient)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "should record miner metrics successfully when online",
			metrics: &MinerMetrics{
				MinerID:         minerID,
				Timestamp:       time.Now(),
				Hashrate:        50000,
				SharesSubmitted: 1000,
				SharesAccepted:  980,
				SharesRejected:  20,
				IsOnline:        true,
				Difficulty:      100,
				Earnings:        1.5,
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				mr.On("StoreMinerMetrics", mock.Anything, mock.AnythingOfType("*monitoring.MinerMetrics")).Return(nil)
				mp.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mp.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "should record miner metrics when offline",
			metrics: &MinerMetrics{
				MinerID:  minerID,
				IsOnline: false,
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				mr.On("StoreMinerMetrics", mock.Anything, mock.AnythingOfType("*monitoring.MinerMetrics")).Return(nil)
				mp.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mp.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "should fail with nil metrics",
			metrics:   nil,
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {},
			wantErr:   true,
			errMsg:    "cannot be nil",
		},
		{
			name: "should fail with empty miner ID",
			metrics: &MinerMetrics{
				MinerID: uuid.Nil,
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {},
			wantErr:   true,
			errMsg:    "miner ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo, mockPrometheus)

			service := NewService(mockRepo, mockPrometheus)
			err := service.RecordMinerMetrics(context.Background(), tt.metrics)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMonitoringService_RecordPoolMetrics(t *testing.T) {
	tests := []struct {
		name      string
		metrics   *PoolMetrics
		setupMock func(*MockRepository, *MockPrometheusClient)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "should record pool metrics successfully",
			metrics: &PoolMetrics{
				Timestamp:         time.Now(),
				TotalHashrate:     5000000,
				ActiveMiners:      200,
				TotalShares:       100000,
				ValidShares:       99000,
				InvalidShares:     1000,
				BlocksFound:       10,
				NetworkDifficulty: 1000000,
				PoolDifficulty:    50000,
				NetworkHashrate:   100000000,
				PoolEfficiency:    98.5,
				Luck:              105.0,
			},
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {
				mr.On("StorePoolMetrics", mock.Anything, mock.AnythingOfType("*monitoring.PoolMetrics")).Return(nil)
				mp.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mp.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "should fail with nil metrics",
			metrics:   nil,
			setupMock: func(mr *MockRepository, mp *MockPrometheusClient) {},
			wantErr:   true,
			errMsg:    "cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo, mockPrometheus)

			service := NewService(mockRepo, mockPrometheus)
			err := service.RecordPoolMetrics(context.Background(), tt.metrics)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMonitoringService_GetMetrics(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		start      time.Time
		end        time.Time
		setupMock  func(*MockRepository)
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "should get metrics successfully",
			metricName: "cpu_usage",
			start:      time.Now().Add(-time.Hour),
			end:        time.Now(),
			setupMock: func(mr *MockRepository) {
				mr.On("GetMetrics", mock.Anything, "cpu_usage", mock.Anything, mock.Anything).Return([]*Metric{
					{Name: "cpu_usage", Value: 50},
					{Name: "cpu_usage", Value: 60},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:       "should fail with empty metric name",
			metricName: "",
			start:      time.Now().Add(-time.Hour),
			end:        time.Now(),
			setupMock:  func(mr *MockRepository) {},
			wantErr:    true,
			errMsg:     "name cannot be empty",
		},
		{
			name:       "should fail when end before start",
			metricName: "cpu_usage",
			start:      time.Now(),
			end:        time.Now().Add(-time.Hour),
			setupMock:  func(mr *MockRepository) {},
			wantErr:    true,
			errMsg:     "end time must be after start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo, mockPrometheus)
			metrics, err := service.GetMetrics(context.Background(), tt.metricName, tt.start, tt.end)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metrics)
			}
		})
	}
}

func TestMonitoringService_CreateDashboard_EmptyUserID(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)

	_, err := service.CreateDashboard(context.Background(), "Dashboard", "Desc", "{}", true, uuid.Nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "created by user ID cannot be empty")
}

func TestMonitoringService_CreateAlertRule_EmptyName(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)

	_, err := service.CreateAlertRule(context.Background(), "", "query", ">", 90.0, "5m", "warning")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestMonitoringService_CreateAlertRule_EmptyQuery(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)

	_, err := service.CreateAlertRule(context.Background(), "Rule", "", ">", 90.0, "5m", "warning")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

func TestMonitoringService_CreateAlertRule_InvalidSeverity(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)

	_, err := service.CreateAlertRule(context.Background(), "Rule", "query", ">", 90.0, "5m", "invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid severity level")
}

func TestMonitoringService_CreateAlert_AllSeverities(t *testing.T) {
	severities := []string{"info", "warning", "error", "critical"}

	for _, severity := range severities {
		t.Run(severity, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			mockRepo.On("CreateAlert", mock.Anything, mock.AnythingOfType("*monitoring.Alert")).Return(nil)

			service := NewService(mockRepo, mockPrometheus)
			alert, err := service.CreateAlert(context.Background(), "Test", "Desc", severity, nil)

			assert.NoError(t, err)
			assert.Equal(t, severity, alert.Severity)
		})
	}
}

func TestMonitoringService_CreateAlertRule_AllConditions(t *testing.T) {
	conditions := []string{">", "<", ">=", "<=", "==", "!="}

	for _, cond := range conditions {
		t.Run(cond, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			mockRepo.On("CreateAlertRule", mock.Anything, mock.AnythingOfType("*monitoring.AlertRule")).Return(nil)

			service := NewService(mockRepo, mockPrometheus)
			rule, err := service.CreateAlertRule(context.Background(), "Rule", "query", cond, 90.0, "5m", "warning")

			assert.NoError(t, err)
			assert.Equal(t, cond, rule.Condition)
		})
	}
}

func TestMonitoringService_CreateAlertRule_ValidDurations(t *testing.T) {
	durations := []string{"5m", "10s", "1h", "30m"}

	for _, dur := range durations {
		t.Run(dur, func(t *testing.T) {
			mockRepo := &MockRepository{}
			mockPrometheus := &MockPrometheusClient{}
			mockRepo.On("CreateAlertRule", mock.Anything, mock.AnythingOfType("*monitoring.AlertRule")).Return(nil)

			service := NewService(mockRepo, mockPrometheus)
			rule, err := service.CreateAlertRule(context.Background(), "Rule", "query", ">", 90.0, dur, "warning")

			assert.NoError(t, err)
			assert.Equal(t, dur, rule.Duration)
		})
	}
}

func TestMonitoringService_EvaluateAlertRules_InactiveRule(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	rules := []*AlertRule{
		{
			ID:        uuid.New(),
			Name:      "High CPU",
			Query:     "cpu_usage",
			Condition: ">",
			Threshold: 90.0,
			IsActive:  false, // Inactive
		},
	}
	mockRepo.On("GetAlertRules", mock.Anything).Return(rules, nil)

	service := NewService(mockRepo, mockPrometheus)
	alerts, err := service.EvaluateAlertRules(context.Background())

	assert.NoError(t, err)
	assert.Len(t, alerts, 0)
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestMonitoringService_CreateDashboard_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	mockRepo.On("CreateDashboard", mock.Anything, mock.AnythingOfType("*monitoring.Dashboard")).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	userID := uuid.New()
	dashboard, err := service.CreateDashboard(context.Background(), "Test Dashboard", "Description", `{"panels":[]}`, true, userID)

	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.Equal(t, "Test Dashboard", dashboard.Name)
	assert.True(t, dashboard.IsPublic)
}

func TestMonitoringService_CreateDashboard_EmptyName(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	userID := uuid.New()
	_, err := service.CreateDashboard(context.Background(), "", "Description", `{}`, true, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestMonitoringService_CreateDashboard_InvalidJSON(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	userID := uuid.New()
	_, err := service.CreateDashboard(context.Background(), "Dashboard", "Description", `{invalid}`, true, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON configuration")
}

func TestMonitoringService_RecordPerformanceMetrics_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	mockRepo.On("StorePerformanceMetrics", mock.Anything, mock.AnythingOfType("*monitoring.PerformanceMetrics")).Return(nil)
	mockPrometheus.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockPrometheus.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	metrics := &PerformanceMetrics{
		CPUUsage:    50.0,
		MemoryUsage: 60.0,
		DiskUsage:   70.0,
	}
	err := service.RecordPerformanceMetrics(context.Background(), metrics)

	assert.NoError(t, err)
}

func TestMonitoringService_RecordPerformanceMetrics_NilMetrics(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	err := service.RecordPerformanceMetrics(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metrics cannot be nil")
}

func TestMonitoringService_RecordMinerMetrics_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	mockRepo.On("StoreMinerMetrics", mock.Anything, mock.AnythingOfType("*monitoring.MinerMetrics")).Return(nil)
	mockPrometheus.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockPrometheus.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	minerID := uuid.New()
	metrics := &MinerMetrics{
		MinerID:        minerID,
		Hashrate:       1000000.0,
		SharesAccepted: 100,
	}
	err := service.RecordMinerMetrics(context.Background(), metrics)

	assert.NoError(t, err)
}

func TestMonitoringService_RecordMinerMetrics_NilMetrics(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	err := service.RecordMinerMetrics(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metrics cannot be nil")
}

func TestMonitoringService_RecordMinerMetrics_EmptyMinerID(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	metrics := &MinerMetrics{
		MinerID:  uuid.Nil,
		Hashrate: 1000000.0,
	}
	err := service.RecordMinerMetrics(context.Background(), metrics)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "miner ID cannot be empty")
}

func TestMonitoringService_RecordPoolMetrics_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	mockRepo.On("StorePoolMetrics", mock.Anything, mock.AnythingOfType("*monitoring.PoolMetrics")).Return(nil)
	// Pool metrics calls both gauge and counter for various metrics
	mockPrometheus.On("RecordGauge", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockPrometheus.On("RecordCounter", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	metrics := &PoolMetrics{
		TotalHashrate: 5000000.0,
		ActiveMiners:  50,
		TotalShares:   10000,
		ValidShares:   9900,
	}
	err := service.RecordPoolMetrics(context.Background(), metrics)

	assert.NoError(t, err)
}

func TestMonitoringService_RecordPoolMetrics_NilMetrics(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	err := service.RecordPoolMetrics(context.Background(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metrics cannot be nil")
}

func TestMonitoringService_GetMetrics_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	expectedMetrics := []*Metric{
		{Name: "test_metric", Value: 100.0, Type: "gauge"},
		{Name: "test_metric", Value: 150.0, Type: "gauge"},
	}
	mockRepo.On("GetMetrics", mock.Anything, "test_metric", mock.Anything, mock.Anything).Return(expectedMetrics, nil)

	service := NewService(mockRepo, mockPrometheus)
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	metrics, err := service.GetMetrics(context.Background(), "test_metric", start, end)

	assert.NoError(t, err)
	assert.Len(t, metrics, 2)
}

func TestMonitoringService_GetMetrics_EmptyName(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	_, err := service.GetMetrics(context.Background(), "", start, end)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metric name cannot be empty")
}

func TestMonitoringService_GetMetrics_InvalidTimeRange(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	service := NewService(mockRepo, mockPrometheus)
	start := time.Now()
	end := time.Now().Add(-1 * time.Hour) // End before start
	_, err := service.GetMetrics(context.Background(), "test_metric", start, end)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end time must be after start time")
}

func TestMonitoringService_EvaluateAlertRules_WithActiveRule(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	ruleID := uuid.New()
	rules := []*AlertRule{
		{
			ID:        ruleID,
			Name:      "High CPU",
			Query:     "cpu_usage",
			Condition: ">",
			Threshold: 90.0,
			Severity:  "critical",
			IsActive:  true,
		},
	}
	mockRepo.On("GetAlertRules", mock.Anything).Return(rules, nil)
	mockPrometheus.On("Query", "cpu_usage").Return(95.0, nil) // Above threshold
	mockRepo.On("CreateAlert", mock.Anything, mock.AnythingOfType("*monitoring.Alert")).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	alerts, err := service.EvaluateAlertRules(context.Background())

	assert.NoError(t, err)
	assert.Len(t, alerts, 1)
}

func TestMonitoringService_EvaluateAlertRules_BelowThreshold(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}

	rules := []*AlertRule{
		{
			ID:        uuid.New(),
			Name:      "High CPU",
			Query:     "cpu_usage",
			Condition: ">",
			Threshold: 90.0,
			Severity:  "critical",
			IsActive:  true,
		},
	}
	mockRepo.On("GetAlertRules", mock.Anything).Return(rules, nil)
	mockPrometheus.On("Query", "cpu_usage").Return(50.0, nil) // Below threshold

	service := NewService(mockRepo, mockPrometheus)
	alerts, err := service.EvaluateAlertRules(context.Background())

	assert.NoError(t, err)
	assert.Len(t, alerts, 0)
}

func TestMonitoringService_ResolveAlert_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	mockRepo.On("UpdateAlert", mock.Anything, mock.AnythingOfType("*monitoring.Alert")).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	alertID := uuid.New()
	err := service.ResolveAlert(context.Background(), alertID)

	assert.NoError(t, err)
}

func TestMonitoringService_RecordMetric_HistogramType(t *testing.T) {
	mockRepo := &MockRepository{}
	mockPrometheus := &MockPrometheusClient{}
	mockRepo.On("StoreMetric", mock.Anything, mock.AnythingOfType("*monitoring.Metric")).Return(nil)
	mockPrometheus.On("RecordHistogram", "request_latency", mock.Anything, 0.5).Return(nil)

	service := NewService(mockRepo, mockPrometheus)
	metric := &Metric{
		Name:      "request_latency",
		Value:     0.5,
		Labels:    map[string]string{},
		Timestamp: time.Now(),
		Type:      "histogram",
	}
	err := service.RecordMetric(context.Background(), metric)

	assert.NoError(t, err)
}
