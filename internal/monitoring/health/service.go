package health

import (
	"context"
	"log"
	"os"
	"time"
)

// ServiceConfig contains configuration for the health monitor service.
type ServiceConfig struct {
	// Monitor configuration
	MonitorConfig *HealthMonitorConfig

	// Litecoin node configuration
	LitecoinRPCURL      string
	LitecoinRPCUser     string
	LitecoinRPCPassword string
	LitecoinContainer   string

	// BlockDAG node configuration (for future use)
	BlockDAGRPCURL    string
	BlockDAGContainer string

	// Recovery configuration
	DockerPath     string
	AlertWebhook   string
	CommandTimeout time.Duration

	// Prometheus configuration
	PrometheusEnabled bool
	PrometheusAddr    string

	// Logging
	Logger *log.Logger
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MonitorConfig:       DefaultHealthMonitorConfig(),
		LitecoinRPCURL:      "http://litecoind:9332",
		LitecoinRPCUser:     "chimera",
		LitecoinRPCPassword: "ChimeraLTC2024!",
		LitecoinContainer:   "docker-litecoind-1",
		BlockDAGRPCURL:      "https://rpc.awakening.bdagscan.com",
		BlockDAGContainer:   "",
		CommandTimeout:      60 * time.Second,
		PrometheusEnabled:   true,
		PrometheusAddr:      ":9091",
	}
}

// LoadServiceConfigFromEnv loads configuration from environment variables.
func LoadServiceConfigFromEnv() *ServiceConfig {
	config := DefaultServiceConfig()

	if url := os.Getenv("LITECOIN_RPC_URL"); url != "" {
		config.LitecoinRPCURL = url
	}
	if user := os.Getenv("LITECOIN_RPC_USER"); user != "" {
		config.LitecoinRPCUser = user
	}
	if pass := os.Getenv("LITECOIN_RPC_PASSWORD"); pass != "" {
		config.LitecoinRPCPassword = pass
	}
	if container := os.Getenv("LITECOIN_CONTAINER"); container != "" {
		config.LitecoinContainer = container
	}
	if url := os.Getenv("BLOCKDAG_RPC_URL"); url != "" {
		config.BlockDAGRPCURL = url
	}
	if container := os.Getenv("BLOCKDAG_CONTAINER"); container != "" {
		config.BlockDAGContainer = container
	}
	if webhook := os.Getenv("HEALTH_ALERT_WEBHOOK"); webhook != "" {
		config.AlertWebhook = webhook
	}
	if addr := os.Getenv("HEALTH_PROMETHEUS_ADDR"); addr != "" {
		config.PrometheusAddr = addr
	}
	if os.Getenv("HEALTH_PROMETHEUS_ENABLED") == "false" {
		config.PrometheusEnabled = false
	}
	if os.Getenv("HEALTH_AUTO_RESTART") == "false" {
		config.MonitorConfig.EnableAutoRestart = false
	}

	return config
}

// HealthService manages the complete health monitoring infrastructure.
type HealthService struct {
	config   *ServiceConfig
	monitor  *HealthMonitor
	exporter *PrometheusExporter
	recovery RecoveryAction
	logger   *log.Logger

	running bool
}

// NewHealthService creates a new health monitoring service.
func NewHealthService(config *ServiceConfig) *HealthService {
	if config == nil {
		config = LoadServiceConfigFromEnv()
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "[HealthService] ", log.LstdFlags)
	}

	// Create recovery action
	var recovery RecoveryAction
	if config.MonitorConfig.EnableAutoRestart {
		recovery = NewDockerRecoveryAction(&DockerRecoveryConfig{
			AlertWebhook:   config.AlertWebhook,
			CommandTimeout: config.CommandTimeout,
		})
	} else {
		recovery = NewLogRecoveryAction(func(format string, args ...interface{}) {
			logger.Printf(format, args...)
		})
	}

	// Create health monitor
	monitor := NewHealthMonitor(config.MonitorConfig, recovery, logger)

	// Add default rules
	monitor.AddRule(&MWEBFailureRule{})
	monitor.AddRule(&BlockTemplateFailureRule{})
	monitor.AddRule(&RPCDownRule{})

	service := &HealthService{
		config:   config,
		monitor:  monitor,
		recovery: recovery,
		logger:   logger,
	}

	// Create Prometheus exporter if enabled
	if config.PrometheusEnabled {
		service.exporter = NewPrometheusExporter(monitor, config.PrometheusAddr)
	}

	return service
}

// RegisterLitecoinNode registers the Litecoin node for monitoring.
func (s *HealthService) RegisterLitecoinNode() error {
	checker := NewLitecoinHealthChecker(&LitecoinNodeConfig{
		RPCURL:      s.config.LitecoinRPCURL,
		RPCUser:     s.config.LitecoinRPCUser,
		RPCPassword: s.config.LitecoinRPCPassword,
		Container:   s.config.LitecoinContainer,
	}, s.config.MonitorConfig.RPCTimeout)

	return s.monitor.RegisterNode("litecoin", checker, s.config.LitecoinContainer)
}

// RegisterBlockDAGNode registers the BlockDAG node for monitoring.
func (s *HealthService) RegisterBlockDAGNode() error {
	if s.config.BlockDAGContainer == "" {
		s.logger.Println("BlockDAG container not configured, skipping registration")
		return nil
	}

	checker := NewBlockDAGHealthChecker(&BlockDAGNodeConfig{
		RPCURL:    s.config.BlockDAGRPCURL,
		Container: s.config.BlockDAGContainer,
	}, s.config.MonitorConfig.RPCTimeout)

	return s.monitor.RegisterNode("blockdag", checker, s.config.BlockDAGContainer)
}

// Start starts the health monitoring service.
func (s *HealthService) Start(ctx context.Context) error {
	if s.running {
		return ErrMonitorAlreadyRunning
	}

	s.logger.Println("Starting health monitoring service...")

	// Register nodes
	if err := s.RegisterLitecoinNode(); err != nil {
		s.logger.Printf("Warning: Failed to register Litecoin node: %v", err)
	}

	if err := s.RegisterBlockDAGNode(); err != nil {
		s.logger.Printf("Warning: Failed to register BlockDAG node: %v", err)
	}

	// Start Prometheus exporter
	if s.exporter != nil {
		if err := s.exporter.Start(); err != nil {
			s.logger.Printf("Warning: Failed to start Prometheus exporter: %v", err)
		} else {
			s.logger.Printf("Prometheus metrics available at %s/metrics", s.config.PrometheusAddr)
		}
	}

	// Start health monitor
	if err := s.monitor.Start(ctx); err != nil {
		return err
	}

	s.running = true
	s.logger.Println("Health monitoring service started successfully")

	return nil
}

// Stop stops the health monitoring service.
func (s *HealthService) Stop(ctx context.Context) error {
	if !s.running {
		return ErrMonitorNotRunning
	}

	s.logger.Println("Stopping health monitoring service...")

	// Stop Prometheus exporter
	if s.exporter != nil {
		s.exporter.Stop()
	}

	// Stop health monitor
	if err := s.monitor.Stop(ctx); err != nil {
		return err
	}

	s.running = false
	s.logger.Println("Health monitoring service stopped")

	return nil
}

// GetMonitor returns the underlying health monitor.
func (s *HealthService) GetMonitor() *HealthMonitor {
	return s.monitor
}

// GetHealthStatus returns the current health status of all nodes.
func (s *HealthService) GetHealthStatus(ctx context.Context) (map[string]*NodeHealth, error) {
	return s.monitor.GetHealthStatus(ctx)
}

// ForceCheck forces an immediate health check on a specific node.
func (s *HealthService) ForceCheck(ctx context.Context, nodeName string) (*NodeDiagnostics, error) {
	return s.monitor.ForceCheck(ctx, nodeName)
}

// IsRunning returns whether the service is currently running.
func (s *HealthService) IsRunning() bool {
	return s.running
}

// GetStats returns monitoring statistics.
func (s *HealthService) GetStats() *MonitorStats {
	return s.monitor.GetStats()
}

// =============================================================================
// Global Health Service Instance (for easy integration)
// =============================================================================

var globalHealthService *HealthService
var globalHealthServiceOnce = false

// InitGlobalHealthService initializes the global health service.
func InitGlobalHealthService(config *ServiceConfig) *HealthService {
	if globalHealthServiceOnce {
		return globalHealthService
	}
	globalHealthService = NewHealthService(config)
	globalHealthServiceOnce = true
	return globalHealthService
}

// GetGlobalHealthService returns the global health service instance.
func GetGlobalHealthService() *HealthService {
	return globalHealthService
}

// StartGlobalHealthService starts the global health service.
func StartGlobalHealthService(ctx context.Context) error {
	if globalHealthService == nil {
		globalHealthService = NewHealthService(nil)
		globalHealthServiceOnce = true
	}
	return globalHealthService.Start(ctx)
}

// StopGlobalHealthService stops the global health service.
func StopGlobalHealthService(ctx context.Context) error {
	if globalHealthService != nil {
		return globalHealthService.Stop(ctx)
	}
	return nil
}
