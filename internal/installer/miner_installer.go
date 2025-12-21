package installer

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type MinerInstaller struct {
	hardwareDetector *HardwareDetector
	configGenerator  *MinerConfigGenerator
	driverManager    *DriverManager
}

type HardwareInfo struct {
	CPUs         []CPUInfo  `json:"cpus"`
	GPUs         []GPUInfo  `json:"gpus"`
	Memory       MemoryInfo `json:"memory"`
	OS           string     `json:"os"`
	Architecture string     `json:"architecture"`
}

type GPUInfo struct {
	Model    string `json:"model"`
	Memory   int64  `json:"memory"`
	Driver   string `json:"driver"`
	Vendor   string `json:"vendor"`
	PCIeSlot string `json:"pcie_slot"`
}

type MinerConfig struct {
	PoolURL       string `yaml:"pool_url"`
	WalletAddress string `yaml:"wallet_address"`
	MinerName     string `yaml:"miner_name"`
	Algorithm     string `yaml:"algorithm"`
	MiningMode    string `yaml:"mining_mode"` // cpu, gpu, hybrid
	CPUEnabled    bool   `yaml:"cpu_enabled"`
	GPUEnabled    bool   `yaml:"gpu_enabled"`
	CPUThreads    int    `yaml:"cpu_threads"`
	GPUThreads    int    `yaml:"gpu_threads"`
	PowerLimit    int    `yaml:"power_limit"` // Percentage
	TempLimit     int    `yaml:"temp_limit"`  // Celsius
	AutoStart     bool   `yaml:"auto_start"`
	LogLevel      string `yaml:"log_level"`
}

type WizardResponses struct {
	PoolURL       string `json:"pool_url"`
	WalletAddress string `json:"wallet_address"`
	MinerName     string `json:"miner_name"`
	AutoStart     bool   `json:"auto_start"`
	EnableGPU     bool   `json:"enable_gpu"`
	EnableCPU     bool   `json:"enable_cpu"`
}

type MinerInstallConfig struct {
	InstallPath   string `json:"install_path"`
	PoolURL       string `json:"pool_url"`
	WalletAddress string `json:"wallet_address"`
	MinerName     string `json:"miner_name"`
	AutoStart     bool   `json:"auto_start"`
	AutoDetect    bool   `json:"auto_detect"`
}

type MinerInstallResult struct {
	InstallationID  string   `json:"installation_id"`
	Status          string   `json:"status"`
	MinerExecutable string   `json:"miner_executable"`
	ConfigPath      string   `json:"config_path"`
	NextSteps       []string `json:"next_steps"`
	Errors          []string `json:"errors,omitempty"`
}

type DriverInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Required    bool   `json:"required"`
}

type UpdateInfo struct {
	Available   bool   `json:"available"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Changelog   string `json:"changelog"`
}

func NewMinerInstaller() *MinerInstaller {
	return &MinerInstaller{
		hardwareDetector: NewHardwareDetector(),
		configGenerator:  NewMinerConfigGenerator(),
		driverManager:    NewDriverManager(),
	}
}

func (mi *MinerInstaller) DetectHardware() (HardwareInfo, error) {
	return mi.hardwareDetector.DetectHardware()
}

func (mi *MinerInstaller) GenerateOptimalConfig(hardware HardwareInfo) (MinerConfig, error) {
	config := MinerConfig{
		Algorithm: "blake2s",
		LogLevel:  "info",
	}

	// Determine optimal mining mode
	if len(hardware.GPUs) > 0 {
		config.MiningMode = "gpu"
		config.GPUEnabled = true
		config.CPUEnabled = false
		config.GPUThreads = len(hardware.GPUs)
		config.PowerLimit = 80 // Conservative power limit
		config.TempLimit = 83  // Safe temperature limit
	} else if len(hardware.CPUs) > 0 {
		config.MiningMode = "cpu"
		config.GPUEnabled = false
		config.CPUEnabled = true

		// Use most cores but leave some for system
		totalThreads := 0
		for _, cpu := range hardware.CPUs {
			totalThreads += cpu.Threads
		}

		if totalThreads > 4 {
			config.CPUThreads = totalThreads - 2 // Leave 2 threads for system
		} else if totalThreads > 2 {
			config.CPUThreads = totalThreads / 2 // Use half on low-end systems
		} else {
			config.CPUThreads = 1 // Minimum viable
		}
	} else {
		return config, fmt.Errorf("no suitable mining hardware detected")
	}

	return config, nil
}

func (mi *MinerInstaller) ProcessWizardResponses(responses WizardResponses, hardware HardwareInfo) (MinerConfig, error) {
	config, err := mi.GenerateOptimalConfig(hardware)
	if err != nil {
		return config, err
	}

	// Apply user preferences
	config.PoolURL = responses.PoolURL
	config.WalletAddress = responses.WalletAddress
	config.MinerName = responses.MinerName
	config.AutoStart = responses.AutoStart

	// Override hardware detection with user preferences
	if responses.EnableGPU && len(hardware.GPUs) > 0 {
		config.GPUEnabled = true
	} else {
		config.GPUEnabled = false
	}

	if responses.EnableCPU && len(hardware.CPUs) > 0 {
		config.CPUEnabled = true
	} else if !config.GPUEnabled {
		// Force CPU mining if GPU is disabled and no other option
		config.CPUEnabled = true
	}

	// Determine mining mode based on enabled options
	if config.GPUEnabled && config.CPUEnabled {
		config.MiningMode = "hybrid"
	} else if config.GPUEnabled {
		config.MiningMode = "gpu"
	} else {
		config.MiningMode = "cpu"
	}

	return config, nil
}

func (mi *MinerInstaller) ValidateConfig(config MinerConfig) error {
	// Validate pool URL
	if config.PoolURL == "" {
		return fmt.Errorf("pool URL is required")
	}

	parsedURL, err := url.Parse(config.PoolURL)
	if err != nil {
		return fmt.Errorf("invalid pool URL: %w", err)
	}

	if parsedURL.Scheme != "stratum+tcp" && parsedURL.Scheme != "stratum+ssl" {
		return fmt.Errorf("pool URL must use stratum+tcp or stratum+ssl scheme")
	}

	// Validate wallet address
	if config.WalletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}

	// Basic wallet address validation (this would be more sophisticated in reality)
	if len(config.WalletAddress) < 10 {
		return fmt.Errorf("wallet address appears to be too short")
	}

	// Validate mining configuration
	if !config.CPUEnabled && !config.GPUEnabled {
		return fmt.Errorf("at least one mining method (CPU or GPU) must be enabled")
	}

	if config.CPUEnabled && config.CPUThreads <= 0 {
		return fmt.Errorf("CPU threads must be greater than 0 when CPU mining is enabled")
	}

	if config.GPUEnabled && config.GPUThreads <= 0 {
		return fmt.Errorf("GPU threads must be greater than 0 when GPU mining is enabled")
	}

	return nil
}

func (mi *MinerInstaller) OneClickInstall(ctx context.Context, installConfig MinerInstallConfig) (MinerInstallResult, error) {
	installationID := uuid.New().String()

	result := MinerInstallResult{
		InstallationID: installationID,
		Status:         "in_progress",
		NextSteps:      []string{},
		Errors:         []string{},
	}

	// Step 1: Detect hardware if auto-detect is enabled
	var hardware HardwareInfo
	var err error

	if installConfig.AutoDetect {
		hardware, err = mi.DetectHardware()
		if err != nil {
			result.Status = "failed"
			result.Errors = append(result.Errors, fmt.Sprintf("Hardware detection failed: %v", err))
			return result, err
		}
	}

	// Step 2: Generate configuration
	config := MinerConfig{
		PoolURL:       installConfig.PoolURL,
		WalletAddress: installConfig.WalletAddress,
		MinerName:     installConfig.MinerName,
		Algorithm:     "blake2s",
		AutoStart:     installConfig.AutoStart,
		LogLevel:      "info",
	}

	if installConfig.AutoDetect {
		optimalConfig, err := mi.GenerateOptimalConfig(hardware)
		if err != nil {
			result.Status = "failed"
			result.Errors = append(result.Errors, fmt.Sprintf("Configuration generation failed: %v", err))
			return result, err
		}

		// Merge optimal settings with user preferences
		config.MiningMode = optimalConfig.MiningMode
		config.CPUEnabled = optimalConfig.CPUEnabled
		config.GPUEnabled = optimalConfig.GPUEnabled
		config.CPUThreads = optimalConfig.CPUThreads
		config.GPUThreads = optimalConfig.GPUThreads
		config.PowerLimit = optimalConfig.PowerLimit
		config.TempLimit = optimalConfig.TempLimit
	} else {
		// Use safe defaults
		config.MiningMode = "cpu"
		config.CPUEnabled = true
		config.GPUEnabled = false
		config.CPUThreads = runtime.NumCPU() / 2
	}

	// Step 3: Validate configuration
	if err := mi.ValidateConfig(config); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Configuration validation failed: %v", err))
		return result, err
	}

	// Step 4: Create installation directory
	if err := os.MkdirAll(installConfig.InstallPath, 0755); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create install directory: %v", err))
		return result, err
	}

	// Step 5: Download and install miner executable
	minerExecutable, err := mi.downloadMinerExecutable(installConfig.InstallPath)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to download miner: %v", err))
		return result, err
	}

	// Step 6: Generate configuration file
	configData, err := yaml.Marshal(config)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to marshal config: %v", err))
		return result, err
	}

	configPath := filepath.Join(installConfig.InstallPath, "miner.yml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to write config file: %v", err))
		return result, err
	}

	// Step 7: Generate start/stop scripts
	if err := mi.generateMinerScripts(installConfig.InstallPath, minerExecutable); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate scripts: %v", err))
		return result, err
	}

	// Step 8: Install missing drivers if needed
	if installConfig.AutoDetect && len(hardware.GPUs) > 0 {
		missingDrivers, err := mi.DetectMissingDrivers()
		if err == nil && len(missingDrivers) > 0 {
			result.NextSteps = append(result.NextSteps, "Install missing GPU drivers for optimal performance")
		}
	}

	result.Status = "success"
	result.MinerExecutable = minerExecutable
	result.ConfigPath = configPath
	result.NextSteps = append(result.NextSteps,
		"Configuration saved to "+configPath,
		"Run './start-mining.sh' to begin mining",
		"Monitor your mining progress in the pool dashboard",
	)

	if config.AutoStart {
		result.NextSteps = append(result.NextSteps, "Miner will start automatically")
	}

	return result, nil
}

func (mi *MinerInstaller) downloadMinerExecutable(installPath string) (string, error) {
	// This would download the appropriate miner executable for the platform
	// For now, we'll create a placeholder

	var executableName string
	switch runtime.GOOS {
	case "windows":
		executableName = "chimera-miner.exe"
	default:
		executableName = "chimera-miner"
	}

	executablePath := filepath.Join(installPath, executableName)

	// Create a placeholder executable
	placeholder := `#!/bin/bash
echo "Chimera Miner v1.0.0"
echo "This is a placeholder executable for testing"
echo "In production, this would be the actual miner binary"
`

	if err := os.WriteFile(executablePath, []byte(placeholder), 0755); err != nil {
		return "", err
	}

	return executablePath, nil
}

func (mi *MinerInstaller) generateMinerScripts(installPath, minerExecutable string) error {
	// Generate start script
	var startScript string
	if runtime.GOOS == "windows" {
		startScript = fmt.Sprintf(`@echo off
echo Starting Chimera Miner...
"%s" --config miner.yml
pause
`, minerExecutable)
	} else {
		startScript = fmt.Sprintf(`#!/bin/bash
set -e

echo "Starting Chimera Miner..."
"%s" --config miner.yml

echo "Miner stopped."
`, minerExecutable)
	}

	startScriptName := "start-mining.sh"
	if runtime.GOOS == "windows" {
		startScriptName = "start-mining.bat"
	}

	startPath := filepath.Join(installPath, startScriptName)
	if err := os.WriteFile(startPath, []byte(startScript), 0755); err != nil {
		return err
	}

	// Generate stop script
	var stopScript string
	if runtime.GOOS == "windows" {
		stopScript = `@echo off
echo Stopping Chimera Miner...
taskkill /f /im chimera-miner.exe
echo Miner stopped.
pause
`
	} else {
		stopScript = `#!/bin/bash
echo "Stopping Chimera Miner..."
pkill -f chimera-miner || true
echo "Miner stopped."
`
	}

	stopScriptName := "stop-mining.sh"
	if runtime.GOOS == "windows" {
		stopScriptName = "stop-mining.bat"
	}

	stopPath := filepath.Join(installPath, stopScriptName)
	return os.WriteFile(stopPath, []byte(stopScript), 0755)
}

func (mi *MinerInstaller) DetectMissingDrivers() ([]DriverInfo, error) {
	return mi.driverManager.DetectMissingDrivers()
}

func (mi *MinerInstaller) GetDefaultInstallPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMFILES"), "ChimeraMiner"), nil
	case "darwin":
		return "/Applications/ChimeraMiner", nil
	case "linux":
		return "/opt/chimera-miner", nil
	default:
		return "./chimera-miner", nil
	}
}

func (mi *MinerInstaller) CheckForUpdates(currentVersion string) (UpdateInfo, error) {
	// This would check for updates from a remote server
	// For now, return no updates available
	return UpdateInfo{
		Available: false,
	}, nil
}
