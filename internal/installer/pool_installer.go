package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type PoolInstaller struct {
	systemDetector *SystemDetector
	configGenerator *ConfigGenerator
	dockerComposer *DockerComposer
	cloudTemplater *CloudTemplater
}

type SystemSpecs struct {
	CPU        CPUInfo        `json:"cpu"`
	Memory     MemoryInfo     `json:"memory"`
	Storage    StorageInfo    `json:"storage"`
	Network    NetworkInfo    `json:"network"`
	OS         string         `json:"os"`
	Containers ContainerSupport `json:"containers"`
}

type CPUInfo struct {
	Cores        int    `json:"cores"`
	Threads      int    `json:"threads"`
	Architecture string `json:"architecture"`
	Model        string `json:"model"`
}

type MemoryInfo struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
}

type StorageInfo struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
	Type      string `json:"type"` // ssd, hdd, nvme
}

type NetworkInfo struct {
	Bandwidth int    `json:"bandwidth"` // Mbps
	Latency   int    `json:"latency"`   // ms
	Type      string `json:"type"`      // ethernet, wifi
}

type ContainerSupport struct {
	Docker bool `json:"docker"`
	Podman bool `json:"podman"`
}

type PoolConfig struct {
	MaxMiners        int              `yaml:"max_miners"`
	DatabaseConfig   DatabaseConfig   `yaml:"database"`
	RedisConfig      RedisConfig      `yaml:"redis"`
	StratumConfig    StratumConfig    `yaml:"stratum"`
	SecurityConfig   SecurityConfig   `yaml:"security"`
	MonitoringConfig MonitoringConfig `yaml:"monitoring"`
}

type DatabaseConfig struct {
	MaxConnections int    `yaml:"max_connections"`
	PoolSize       int    `yaml:"pool_size"`
	InstanceType   string `yaml:"instance_type,omitempty"`
	Storage        int    `yaml:"storage,omitempty"`
}

type RedisConfig struct {
	MaxConnections int `yaml:"max_connections"`
}

type StratumConfig struct {
	MaxConnections int `yaml:"max_connections"`
	WorkerThreads  int `yaml:"worker_threads"`
	Port           int `yaml:"port"`
}

type SecurityConfig struct {
	RateLimitEnabled     bool     `yaml:"rate_limit_enabled"`
	MFARequired          bool     `yaml:"mfa_required"`
	EnableSSL            bool     `yaml:"enable_ssl,omitempty"`
	AllowedCIDRs         []string `yaml:"allowed_cidrs,omitempty"`
	EnableWAF            bool     `yaml:"enable_waf,omitempty"`
	EnableDDoSProtection bool     `yaml:"enable_ddos_protection,omitempty"`
}

type MonitoringConfig struct {
	MetricsEnabled bool   `yaml:"metrics_enabled"`
	LogLevel       string `yaml:"log_level"`
}

type InstallConfig struct {
	InstallPath    string `json:"install_path"`
	AutoStart      bool   `json:"auto_start"`
	EnableSSL      bool   `json:"enable_ssl"`
	DomainName     string `json:"domain_name"`
	AdminEmail     string `json:"admin_email"`
	WalletAddress  string `json:"wallet_address"`
}

type InstallResult struct {
	InstallationID string   `json:"installation_id"`
	Status         string   `json:"status"`
	ConfigPath     string   `json:"config_path"`
	NextSteps      []string `json:"next_steps"`
	Errors         []string `json:"errors,omitempty"`
}

type CloudProvider string

const (
	CloudProviderAWS   CloudProvider = "aws"
	CloudProviderGCP   CloudProvider = "gcp"
	CloudProviderAzure CloudProvider = "azure"
)

func NewPoolInstaller() *PoolInstaller {
	return &PoolInstaller{
		systemDetector:  NewSystemDetector(),
		configGenerator: NewConfigGenerator(),
		dockerComposer:  NewDockerComposer(),
		cloudTemplater:  NewCloudTemplater(),
	}
}

func (pi *PoolInstaller) DetectSystemSpecs() (SystemSpecs, error) {
	return pi.systemDetector.DetectSpecs()
}

func (pi *PoolInstaller) GenerateAutoConfiguration(specs SystemSpecs) (PoolConfig, error) {
	// Validate minimum requirements
	if err := pi.validateMinimumRequirements(specs); err != nil {
		return PoolConfig{}, fmt.Errorf("system does not meet minimum requirements: %w", err)
	}

	config := PoolConfig{}

	// Configure based on system resources
	config.MaxMiners = pi.calculateMaxMiners(specs)
	config.DatabaseConfig = pi.configureDatabaseForSpecs(specs)
	config.RedisConfig = pi.configureRedisForSpecs(specs)
	config.StratumConfig = pi.configureStratumForSpecs(specs)
	config.SecurityConfig = pi.configureSecurityForSpecs(specs)
	config.MonitoringConfig = pi.configureMonitoringForSpecs(specs)

	return config, nil
}

func (pi *PoolInstaller) validateMinimumRequirements(specs SystemSpecs) error {
	// Minimum requirements
	minCores := 1
	minMemoryGB := 1
	minStorageGB := 10

	if specs.CPU.Cores < minCores {
		return fmt.Errorf("insufficient CPU cores: need %d, have %d", minCores, specs.CPU.Cores)
	}

	memoryGB := specs.Memory.Total / (1024 * 1024 * 1024)
	if memoryGB < int64(minMemoryGB) {
		return fmt.Errorf("insufficient memory: need %dGB, have %dGB", minMemoryGB, memoryGB)
	}

	storageGB := specs.Storage.Available / (1024 * 1024 * 1024)
	if storageGB < int64(minStorageGB) {
		return fmt.Errorf("insufficient storage: need %dGB, have %dGB", minStorageGB, storageGB)
	}

	if !specs.Containers.Docker && !specs.Containers.Podman {
		return fmt.Errorf("no container runtime available (Docker or Podman required)")
	}

	return nil
}

func (pi *PoolInstaller) calculateMaxMiners(specs SystemSpecs) int {
	// Base calculation on CPU cores and memory
	coreBasedLimit := specs.CPU.Cores * 100
	memoryGB := specs.Memory.Total / (1024 * 1024 * 1024)
	memoryBasedLimit := int(memoryGB) * 50

	// Use the lower of the two limits
	maxMiners := coreBasedLimit
	if memoryBasedLimit < maxMiners {
		maxMiners = memoryBasedLimit
	}

	// Apply network bandwidth constraints
	if specs.Network.Bandwidth < 100 {
		maxMiners = min(maxMiners, 100)
	} else if specs.Network.Bandwidth < 1000 {
		maxMiners = min(maxMiners, 1000)
	}

	// Ensure minimum viable pool size
	if maxMiners < 10 {
		maxMiners = 10
	}

	return maxMiners
}

func (pi *PoolInstaller) configureDatabaseForSpecs(specs SystemSpecs) DatabaseConfig {
	config := DatabaseConfig{
		MaxConnections: 10,
		PoolSize:       5,
	}

	// Scale based on expected load
	memoryGB := specs.Memory.Total / (1024 * 1024 * 1024)
	
	if memoryGB >= 32 {
		config.MaxConnections = 100
		config.PoolSize = 50
	} else if memoryGB >= 16 {
		config.MaxConnections = 50
		config.PoolSize = 25
	} else if memoryGB >= 8 {
		config.MaxConnections = 25
		config.PoolSize = 12
	}

	return config
}

func (pi *PoolInstaller) configureRedisForSpecs(specs SystemSpecs) RedisConfig {
	config := RedisConfig{
		MaxConnections: 20,
	}

	memoryGB := specs.Memory.Total / (1024 * 1024 * 1024)
	
	if memoryGB >= 32 {
		config.MaxConnections = 200
	} else if memoryGB >= 16 {
		config.MaxConnections = 100
	} else if memoryGB >= 8 {
		config.MaxConnections = 50
	}

	return config
}

func (pi *PoolInstaller) configureStratumForSpecs(specs SystemSpecs) StratumConfig {
	config := StratumConfig{
		MaxConnections: 100,
		WorkerThreads:  specs.CPU.Cores,
		Port:           4444,
	}

	// Scale connections based on system capacity
	maxMiners := pi.calculateMaxMiners(specs)
	config.MaxConnections = maxMiners

	// Ensure reasonable thread count
	if config.WorkerThreads > 32 {
		config.WorkerThreads = 32
	}
	if config.WorkerThreads < 2 {
		config.WorkerThreads = 2
	}

	return config
}

func (pi *PoolInstaller) configureSecurityForSpecs(specs SystemSpecs) SecurityConfig {
	config := SecurityConfig{
		RateLimitEnabled: true,
		MFARequired:      false,
	}

	// Enable MFA for high-capacity pools
	if pi.calculateMaxMiners(specs) > 1000 {
		config.MFARequired = true
	}

	return config
}

func (pi *PoolInstaller) configureMonitoringForSpecs(specs SystemSpecs) MonitoringConfig {
	config := MonitoringConfig{
		MetricsEnabled: false,
		LogLevel:       "warn",
	}

	// Enable metrics for systems with sufficient resources
	memoryGB := specs.Memory.Total / (1024 * 1024 * 1024)
	if memoryGB >= 8 {
		config.MetricsEnabled = true
		config.LogLevel = "info"
	}

	return config
}

func (pi *PoolInstaller) GenerateDockerCompose(config PoolConfig) (string, error) {
	return pi.dockerComposer.Generate(config)
}

func (pi *PoolInstaller) OneClickInstall(ctx context.Context, installConfig InstallConfig) (InstallResult, error) {
	installationID := uuid.New().String()
	
	result := InstallResult{
		InstallationID: installationID,
		Status:         "in_progress",
		NextSteps:      []string{},
		Errors:         []string{},
	}

	// Step 1: Detect system specifications
	specs, err := pi.DetectSystemSpecs()
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to detect system specs: %v", err))
		return result, err
	}

	// Step 2: Generate optimal configuration
	poolConfig, err := pi.GenerateAutoConfiguration(specs)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate configuration: %v", err))
		return result, err
	}

	// Step 3: Create installation directory
	if err := os.MkdirAll(installConfig.InstallPath, 0755); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create install directory: %v", err))
		return result, err
	}

	// Step 4: Generate Docker Compose file
	dockerCompose, err := pi.GenerateDockerCompose(poolConfig)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate Docker Compose: %v", err))
		return result, err
	}

	dockerComposePath := filepath.Join(installConfig.InstallPath, "docker-compose.yml")
	if err := os.WriteFile(dockerComposePath, []byte(dockerCompose), 0644); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to write Docker Compose file: %v", err))
		return result, err
	}

	// Step 5: Generate pool configuration
	configDir := filepath.Join(installConfig.InstallPath, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create config directory: %v", err))
		return result, err
	}

	configData, err := yaml.Marshal(poolConfig)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to marshal config: %v", err))
		return result, err
	}

	configPath := filepath.Join(configDir, "pool.yml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to write config file: %v", err))
		return result, err
	}

	// Step 6: Generate management scripts
	if err := pi.generateManagementScripts(installConfig.InstallPath); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate scripts: %v", err))
		return result, err
	}

	// Step 7: Set up SSL if requested
	if installConfig.EnableSSL {
		if err := pi.setupSSL(installConfig); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("SSL setup failed: %v", err))
		}
	}

	result.Status = "success"
	result.ConfigPath = configPath
	result.NextSteps = []string{
		"Review the generated configuration in " + configPath,
		"Run 'cd " + installConfig.InstallPath + " && ./start.sh' to start the pool",
		"Access the web dashboard at http://localhost:8080",
		"Configure your wallet address and pool settings",
	}

	if installConfig.AutoStart {
		result.NextSteps = append(result.NextSteps, "Pool services are starting automatically")
	}

	return result, nil
}

func (pi *PoolInstaller) generateManagementScripts(installPath string) error {
	scriptsDir := filepath.Join(installPath, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return err
	}

	// Generate start script
	startScript := `#!/bin/bash
set -e

echo "Starting Chimera Mining Pool..."
docker-compose up -d

echo "Waiting for services to be ready..."
sleep 10

echo "Pool started successfully!"
echo "Web dashboard: http://localhost:8080"
echo "Stratum server: stratum+tcp://localhost:4444"
`

	startPath := filepath.Join(scriptsDir, "start.sh")
	if err := os.WriteFile(startPath, []byte(startScript), 0755); err != nil {
		return err
	}

	// Generate stop script
	stopScript := `#!/bin/bash
set -e

echo "Stopping Chimera Mining Pool..."
docker-compose down

echo "Pool stopped successfully!"
`

	stopPath := filepath.Join(scriptsDir, "stop.sh")
	if err := os.WriteFile(stopPath, []byte(stopScript), 0755); err != nil {
		return err
	}

	return nil
}

func (pi *PoolInstaller) setupSSL(config InstallConfig) error {
	// This would integrate with Let's Encrypt or similar
	// For now, just create placeholder files
	sslDir := filepath.Join(config.InstallPath, "ssl")
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return err
	}

	// Create placeholder SSL configuration
	sslConfig := fmt.Sprintf(`# SSL Configuration for %s
# This file would contain SSL certificate paths and configuration
domain: %s
email: %s
`, config.DomainName, config.DomainName, config.AdminEmail)

	sslConfigPath := filepath.Join(sslDir, "ssl.conf")
	return os.WriteFile(sslConfigPath, []byte(sslConfig), 0644)
}

func (pi *PoolInstaller) GenerateCloudTemplate(provider CloudProvider, config PoolConfig) (string, error) {
	return pi.cloudTemplater.Generate(provider, config)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}