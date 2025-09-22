package installer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinerInstaller_HardwareDetection(t *testing.T) {
	installer := NewMinerInstaller()
	
	hardware, err := installer.DetectHardware()
	require.NoError(t, err)
	
	// Verify basic hardware detection
	assert.Greater(t, len(hardware.CPUs), 0)
	assert.Greater(t, hardware.Memory.Total, int64(0))
	assert.NotEmpty(t, hardware.OS)
	assert.NotEmpty(t, hardware.Architecture)
	
	// Verify CPU information
	for _, cpu := range hardware.CPUs {
		assert.Greater(t, cpu.Cores, 0)
		assert.Greater(t, cpu.Threads, 0)
		assert.NotEmpty(t, cpu.Model)
	}
}

func TestMinerInstaller_GPUDetection(t *testing.T) {
	installer := NewMinerInstaller()
	
	hardware, err := installer.DetectHardware()
	require.NoError(t, err)
	
	// GPU detection might not find GPUs in CI environment
	// but should not error
	for _, gpu := range hardware.GPUs {
		assert.NotEmpty(t, gpu.Model)
		assert.Greater(t, gpu.Memory, int64(0))
		assert.NotEmpty(t, gpu.Driver)
	}
}

func TestMinerInstaller_OptimalConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		hardware HardwareInfo
		expected MinerConfig
	}{
		{
			name: "high_end_gpu_system",
			hardware: HardwareInfo{
				CPUs: []CPUInfo{{Cores: 8, Threads: 16, Model: "Intel i7-9700K"}},
				GPUs: []GPUInfo{
					{Model: "RTX 3080", Memory: 10 * 1024 * 1024 * 1024, Driver: "nvidia"},
					{Model: "RTX 3070", Memory: 8 * 1024 * 1024 * 1024, Driver: "nvidia"},
				},
				Memory:       MemoryInfo{Total: 32 * 1024 * 1024 * 1024},
				OS:           "linux",
				Architecture: "x86_64",
			},
			expected: MinerConfig{
				MiningMode:    "gpu",
				GPUEnabled:    true,
				CPUEnabled:    false,
				GPUThreads:    2,
				CPUThreads:    0,
				PowerLimit:    80,
				TempLimit:     83,
				Algorithm:     "blake2s",
				PoolURL:       "",
				WalletAddress: "",
			},
		},
		{
			name: "cpu_only_system",
			hardware: HardwareInfo{
				CPUs:         []CPUInfo{{Cores: 16, Threads: 32, Model: "AMD Ryzen 9 5950X"}},
				GPUs:         []GPUInfo{},
				Memory:       MemoryInfo{Total: 64 * 1024 * 1024 * 1024},
				OS:           "linux",
				Architecture: "x86_64",
			},
			expected: MinerConfig{
				MiningMode:    "cpu",
				GPUEnabled:    false,
				CPUEnabled:    true,
				GPUThreads:    0,
				CPUThreads:    30, // Leave 2 threads for system
				PowerLimit:    0,
				TempLimit:     0,
				Algorithm:     "blake2s",
				PoolURL:       "",
				WalletAddress: "",
			},
		},
		{
			name: "low_end_system",
			hardware: HardwareInfo{
				CPUs:         []CPUInfo{{Cores: 2, Threads: 4, Model: "Intel Celeron"}},
				GPUs:         []GPUInfo{},
				Memory:       MemoryInfo{Total: 4 * 1024 * 1024 * 1024},
				OS:           "windows",
				Architecture: "x86_64",
			},
			expected: MinerConfig{
				MiningMode:    "cpu",
				GPUEnabled:    false,
				CPUEnabled:    true,
				GPUThreads:    0,
				CPUThreads:    2, // Use half the threads
				PowerLimit:    0,
				TempLimit:     0,
				Algorithm:     "blake2s",
				PoolURL:       "",
				WalletAddress: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := NewMinerInstaller()
			
			config, err := installer.GenerateOptimalConfig(tt.hardware)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expected.MiningMode, config.MiningMode)
			assert.Equal(t, tt.expected.GPUEnabled, config.GPUEnabled)
			assert.Equal(t, tt.expected.CPUEnabled, config.CPUEnabled)
			assert.Equal(t, tt.expected.GPUThreads, config.GPUThreads)
			assert.Equal(t, tt.expected.CPUThreads, config.CPUThreads)
		})
	}
}

func TestMinerInstaller_WizardFlow(t *testing.T) {
	installer := NewMinerInstaller()
	
	// Simulate wizard responses
	responses := WizardResponses{
		PoolURL:       "stratum+tcp://pool.example.com:4444",
		WalletAddress: "test_wallet_address_123",
		MinerName:     "test_miner",
		AutoStart:     true,
		EnableGPU:     true,
		EnableCPU:     false,
	}
	
	hardware := HardwareInfo{
		CPUs: []CPUInfo{{Cores: 8, Threads: 16, Model: "Test CPU"}},
		GPUs: []GPUInfo{{Model: "Test GPU", Memory: 8 * 1024 * 1024 * 1024, Driver: "nvidia"}},
		Memory: MemoryInfo{Total: 16 * 1024 * 1024 * 1024},
		OS: runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	
	config, err := installer.ProcessWizardResponses(responses, hardware)
	require.NoError(t, err)
	
	assert.Equal(t, responses.PoolURL, config.PoolURL)
	assert.Equal(t, responses.WalletAddress, config.WalletAddress)
	assert.Equal(t, responses.MinerName, config.MinerName)
	assert.Equal(t, responses.EnableGPU, config.GPUEnabled)
	assert.Equal(t, responses.EnableCPU, config.CPUEnabled)
}

func TestMinerInstaller_OneClickInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	installer := NewMinerInstaller()
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	
	installConfig := MinerInstallConfig{
		InstallPath:   tempDir,
		PoolURL:       "stratum+tcp://test.pool.com:4444",
		WalletAddress: "test_wallet_123",
		MinerName:     "test_miner",
		AutoStart:     false, // Don't start mining in test
		AutoDetect:    true,
	}
	
	result, err := installer.OneClickInstall(ctx, installConfig)
	require.NoError(t, err)
	
	// Verify installation artifacts
	assert.FileExists(t, filepath.Join(tempDir, "miner.yml"))
	assert.FileExists(t, filepath.Join(tempDir, "start-mining.sh"))
	assert.FileExists(t, filepath.Join(tempDir, "stop-mining.sh"))
	
	// Verify result
	assert.NotEmpty(t, result.InstallationID)
	assert.Equal(t, "success", result.Status)
	assert.NotEmpty(t, result.MinerExecutable)
}

func TestMinerInstaller_DriverInstallation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Driver installation test not supported on Windows in CI")
	}

	installer := NewMinerInstaller()
	
	// Test driver detection
	drivers, err := installer.DetectMissingDrivers()
	require.NoError(t, err)
	
	// Should not error even if no drivers are missing
	for _, driver := range drivers {
		assert.NotEmpty(t, driver.Name)
		assert.NotEmpty(t, driver.Version)
		assert.NotEmpty(t, driver.DownloadURL)
	}
}

func TestMinerInstaller_ConfigValidation(t *testing.T) {
	installer := NewMinerInstaller()
	
	tests := []struct {
		name        string
		config      MinerConfig
		shouldError bool
	}{
		{
			name: "valid_config",
			config: MinerConfig{
				PoolURL:       "stratum+tcp://pool.example.com:4444",
				WalletAddress: "valid_wallet_address",
				MinerName:     "test_miner",
				Algorithm:     "blake2s",
				CPUEnabled:    true,
				CPUThreads:    4,
			},
			shouldError: false,
		},
		{
			name: "invalid_pool_url",
			config: MinerConfig{
				PoolURL:       "invalid_url",
				WalletAddress: "valid_wallet_address",
				MinerName:     "test_miner",
				Algorithm:     "blake2s",
			},
			shouldError: true,
		},
		{
			name: "empty_wallet_address",
			config: MinerConfig{
				PoolURL:       "stratum+tcp://pool.example.com:4444",
				WalletAddress: "",
				MinerName:     "test_miner",
				Algorithm:     "blake2s",
			},
			shouldError: true,
		},
		{
			name: "no_mining_enabled",
			config: MinerConfig{
				PoolURL:       "stratum+tcp://pool.example.com:4444",
				WalletAddress: "valid_wallet_address",
				MinerName:     "test_miner",
				Algorithm:     "blake2s",
				CPUEnabled:    false,
				GPUEnabled:    false,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := installer.ValidateConfig(tt.config)
			
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMinerInstaller_PlatformSpecificInstall(t *testing.T) {
	installer := NewMinerInstaller()
	
	// Test platform-specific installation paths
	installPath, err := installer.GetDefaultInstallPath()
	require.NoError(t, err)
	
	switch runtime.GOOS {
	case "windows":
		assert.Contains(t, installPath, "Program Files")
	case "darwin":
		assert.Contains(t, installPath, "/Applications")
	case "linux":
		assert.Contains(t, installPath, "/opt")
	}
}

func TestMinerInstaller_AutoUpdate(t *testing.T) {
	installer := NewMinerInstaller()
	
	// Test update check
	updateInfo, err := installer.CheckForUpdates("1.0.0")
	require.NoError(t, err)
	
	// Should not error even if no updates available
	if updateInfo.Available {
		assert.NotEmpty(t, updateInfo.Version)
		assert.NotEmpty(t, updateInfo.DownloadURL)
		assert.NotEmpty(t, updateInfo.Changelog)
	}
}