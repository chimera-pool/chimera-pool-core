package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolInstaller_AutoConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		systemSpecs    SystemSpecs
		expectedConfig PoolConfig
		shouldFail     bool
	}{
		{
			name: "high_performance_server",
			systemSpecs: SystemSpecs{
				CPU:        CPUInfo{Cores: 16, Architecture: "x86_64"},
				Memory:     MemoryInfo{Total: 32 * 1024 * 1024 * 1024}, // 32GB
				Storage:    StorageInfo{Available: 1024 * 1024 * 1024 * 1024}, // 1TB
				Network:    NetworkInfo{Bandwidth: 1000}, // 1Gbps
				OS:         "linux",
				Containers: ContainerSupport{Docker: true, Podman: false},
			},
			expectedConfig: PoolConfig{
				MaxMiners:        10000,
				DatabaseConfig:   DatabaseConfig{MaxConnections: 100, PoolSize: 50},
				RedisConfig:      RedisConfig{MaxConnections: 200},
				StratumConfig:    StratumConfig{MaxConnections: 10000, WorkerThreads: 16},
				SecurityConfig:   SecurityConfig{RateLimitEnabled: true, MFARequired: true},
				MonitoringConfig: MonitoringConfig{MetricsEnabled: true, LogLevel: "info"},
			},
			shouldFail: false,
		},
		{
			name: "low_resource_system",
			systemSpecs: SystemSpecs{
				CPU:        CPUInfo{Cores: 2, Architecture: "x86_64"},
				Memory:     MemoryInfo{Total: 2 * 1024 * 1024 * 1024}, // 2GB
				Storage:    StorageInfo{Available: 50 * 1024 * 1024 * 1024}, // 50GB
				Network:    NetworkInfo{Bandwidth: 100}, // 100Mbps
				OS:         "linux",
				Containers: ContainerSupport{Docker: true, Podman: false},
			},
			expectedConfig: PoolConfig{
				MaxMiners:        100,
				DatabaseConfig:   DatabaseConfig{MaxConnections: 10, PoolSize: 5},
				RedisConfig:      RedisConfig{MaxConnections: 20},
				StratumConfig:    StratumConfig{MaxConnections: 100, WorkerThreads: 2},
				SecurityConfig:   SecurityConfig{RateLimitEnabled: true, MFARequired: false},
				MonitoringConfig: MonitoringConfig{MetricsEnabled: false, LogLevel: "warn"},
			},
			shouldFail: false,
		},
		{
			name: "insufficient_resources",
			systemSpecs: SystemSpecs{
				CPU:        CPUInfo{Cores: 1, Architecture: "x86_64"},
				Memory:     MemoryInfo{Total: 512 * 1024 * 1024}, // 512MB
				Storage:    StorageInfo{Available: 1 * 1024 * 1024 * 1024}, // 1GB
				Network:    NetworkInfo{Bandwidth: 10}, // 10Mbps
				OS:         "linux",
				Containers: ContainerSupport{Docker: false, Podman: false},
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := NewPoolInstaller()
			
			config, err := installer.GenerateAutoConfiguration(tt.systemSpecs)
			
			if tt.shouldFail {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.expectedConfig.MaxMiners, config.MaxMiners)
			assert.Equal(t, tt.expectedConfig.DatabaseConfig.MaxConnections, config.DatabaseConfig.MaxConnections)
			assert.Equal(t, tt.expectedConfig.StratumConfig.WorkerThreads, config.StratumConfig.WorkerThreads)
		})
	}
}

func TestPoolInstaller_DockerComposeGeneration(t *testing.T) {
	installer := NewPoolInstaller()
	config := PoolConfig{
		MaxMiners:      1000,
		DatabaseConfig: DatabaseConfig{MaxConnections: 50, PoolSize: 25},
		RedisConfig:    RedisConfig{MaxConnections: 100},
	}

	dockerCompose, err := installer.GenerateDockerCompose(config)
	require.NoError(t, err)
	
	// Verify essential services are present
	assert.Contains(t, dockerCompose, "chimera-pool-core")
	assert.Contains(t, dockerCompose, "postgresql")
	assert.Contains(t, dockerCompose, "redis")
	assert.Contains(t, dockerCompose, "nginx")
	
	// Verify configuration is applied
	assert.Contains(t, dockerCompose, "POSTGRES_MAX_CONNECTIONS=50")
	assert.Contains(t, dockerCompose, "REDIS_MAX_CONNECTIONS=100")
}

func TestPoolInstaller_SystemDetection(t *testing.T) {
	installer := NewPoolInstaller()
	
	specs, err := installer.DetectSystemSpecs()
	require.NoError(t, err)
	
	// Basic validation that detection works
	assert.Greater(t, specs.CPU.Cores, 0)
	assert.Greater(t, specs.Memory.Total, int64(0))
	assert.Greater(t, specs.Storage.Available, int64(0))
	assert.NotEmpty(t, specs.OS)
}

func TestPoolInstaller_OneClickInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	installer := NewPoolInstaller()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	installConfig := InstallConfig{
		InstallPath:    tempDir,
		AutoStart:      false, // Don't start services in test
		EnableSSL:      false,
		DomainName:     "test.local",
		AdminEmail:     "admin@test.local",
		WalletAddress:  "test_wallet_address",
	}
	
	result, err := installer.OneClickInstall(ctx, installConfig)
	require.NoError(t, err)
	
	// Verify installation artifacts
	assert.FileExists(t, filepath.Join(tempDir, "docker-compose.yml"))
	assert.FileExists(t, filepath.Join(tempDir, "config", "pool.yml"))
	assert.FileExists(t, filepath.Join(tempDir, "scripts", "start.sh"))
	assert.FileExists(t, filepath.Join(tempDir, "scripts", "stop.sh"))
	
	// Verify result contains expected information
	assert.NotEmpty(t, result.InstallationID)
	assert.Equal(t, "success", result.Status)
	assert.Contains(t, result.NextSteps, "start the pool")
}

func TestPoolInstaller_CloudTemplateGeneration(t *testing.T) {
	installer := NewPoolInstaller()
	config := PoolConfig{
		MaxMiners: 5000,
		DatabaseConfig: DatabaseConfig{
			MaxConnections: 100,
			InstanceType:   "db.t3.medium",
		},
	}

	tests := []struct {
		name     string
		provider CloudProvider
	}{
		{"aws", CloudProviderAWS},
		{"gcp", CloudProviderGCP},
		{"azure", CloudProviderAzure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := installer.GenerateCloudTemplate(tt.provider, config)
			require.NoError(t, err)
			assert.NotEmpty(t, template)
			
			// Verify provider-specific elements
			switch tt.provider {
			case CloudProviderAWS:
				assert.Contains(t, template, "AWS::EC2::Instance")
				assert.Contains(t, template, "AWS::RDS::DBInstance")
			case CloudProviderGCP:
				assert.Contains(t, template, "compute.v1.instance")
				assert.Contains(t, template, "sqladmin.v1beta4.instance")
			case CloudProviderAzure:
				assert.Contains(t, template, "Microsoft.Compute/virtualMachines")
				assert.Contains(t, template, "Microsoft.Sql/servers")
			}
		})
	}
}