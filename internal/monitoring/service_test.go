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
