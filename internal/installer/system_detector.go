package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type SystemDetector struct{}

func NewSystemDetector() *SystemDetector {
	return &SystemDetector{}
}

func (sd *SystemDetector) DetectSpecs() (SystemSpecs, error) {
	specs := SystemSpecs{
		OS: runtime.GOOS,
	}

	var err error

	// Detect CPU information
	specs.CPU, err = sd.detectCPU()
	if err != nil {
		return specs, fmt.Errorf("failed to detect CPU: %w", err)
	}

	// Detect memory information
	specs.Memory, err = sd.detectMemory()
	if err != nil {
		return specs, fmt.Errorf("failed to detect memory: %w", err)
	}

	// Detect storage information
	specs.Storage, err = sd.detectStorage()
	if err != nil {
		return specs, fmt.Errorf("failed to detect storage: %w", err)
	}

	// Detect network information
	specs.Network, err = sd.detectNetwork()
	if err != nil {
		return specs, fmt.Errorf("failed to detect network: %w", err)
	}

	// Detect container support
	specs.Containers, err = sd.detectContainerSupport()
	if err != nil {
		return specs, fmt.Errorf("failed to detect container support: %w", err)
	}

	return specs, nil
}

func (sd *SystemDetector) detectCPU() (CPUInfo, error) {
	cpu := CPUInfo{
		Architecture: runtime.GOARCH,
		Cores:        runtime.NumCPU(),
		Threads:      runtime.NumCPU(), // Default assumption
	}

	switch runtime.GOOS {
	case "linux":
		return sd.detectCPULinux(cpu)
	case "darwin":
		return sd.detectCPUDarwin(cpu)
	case "windows":
		return sd.detectCPUWindows(cpu)
	default:
		// Fallback to basic detection
		cpu.Model = "Unknown"
		return cpu, nil
	}
}

func (sd *SystemDetector) detectCPULinux(cpu CPUInfo) (CPUInfo, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return cpu, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	coreCount := 0
	threadCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				cpu.Model = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "processor") {
			threadCount++
		} else if strings.HasPrefix(line, "cpu cores") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if cores, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					coreCount = cores
				}
			}
		}
	}

	if coreCount > 0 {
		cpu.Cores = coreCount
	}
	if threadCount > 0 {
		cpu.Threads = threadCount
	}

	return cpu, scanner.Err()
}

func (sd *SystemDetector) detectCPUDarwin(cpu CPUInfo) (CPUInfo, error) {
	// Get CPU model
	cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
	if output, err := cmd.Output(); err == nil {
		cpu.Model = strings.TrimSpace(string(output))
	}

	// Get core count
	cmd = exec.Command("sysctl", "-n", "hw.physicalcpu")
	if output, err := cmd.Output(); err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			cpu.Cores = cores
		}
	}

	// Get thread count
	cmd = exec.Command("sysctl", "-n", "hw.logicalcpu")
	if output, err := cmd.Output(); err == nil {
		if threads, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			cpu.Threads = threads
		}
	}

	return cpu, nil
}

func (sd *SystemDetector) detectCPUWindows(cpu CPUInfo) (CPUInfo, error) {
	// Use WMIC to get CPU information
	cmd := exec.Command("wmic", "cpu", "get", "Name,NumberOfCores,NumberOfLogicalProcessors", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return cpu, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				cpu.Model = strings.TrimSpace(parts[1])
				if cores, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
					cpu.Cores = cores
				}
				if threads, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
					cpu.Threads = threads
				}
				break
			}
		}
	}

	return cpu, nil
}

func (sd *SystemDetector) detectMemory() (MemoryInfo, error) {
	memory := MemoryInfo{}

	switch runtime.GOOS {
	case "linux":
		return sd.detectMemoryLinux()
	case "darwin":
		return sd.detectMemoryDarwin()
	case "windows":
		return sd.detectMemoryWindows()
	default:
		// Fallback - assume 4GB
		memory.Total = 4 * 1024 * 1024 * 1024
		memory.Available = memory.Total / 2
		return memory, nil
	}
}

func (sd *SystemDetector) detectMemoryLinux() (MemoryInfo, error) {
	memory := MemoryInfo{}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return memory, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					memory.Total = kb * 1024 // Convert KB to bytes
				}
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					memory.Available = kb * 1024 // Convert KB to bytes
				}
			}
		}
	}

	// If MemAvailable is not available, estimate as 50% of total
	if memory.Available == 0 {
		memory.Available = memory.Total / 2
	}

	return memory, scanner.Err()
}

func (sd *SystemDetector) detectMemoryDarwin() (MemoryInfo, error) {
	memory := MemoryInfo{}

	// Get total memory
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	if output, err := cmd.Output(); err == nil {
		if total, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64); err == nil {
			memory.Total = total
		}
	}

	// Estimate available memory as 70% of total (macOS typically uses more for system)
	memory.Available = memory.Total * 7 / 10

	return memory, nil
}

func (sd *SystemDetector) detectMemoryWindows() (MemoryInfo, error) {
	memory := MemoryInfo{}

	// Get total physical memory
	cmd := exec.Command("wmic", "computersystem", "get", "TotalPhysicalMemory", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return memory, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				if total, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					memory.Total = total
				}
				break
			}
		}
	}

	// Estimate available memory as 60% of total
	memory.Available = memory.Total * 6 / 10

	return memory, nil
}

func (sd *SystemDetector) detectStorage() (StorageInfo, error) {
	storage := StorageInfo{
		Type: "unknown",
	}

	switch runtime.GOOS {
	case "linux":
		return sd.detectStorageLinux()
	case "darwin":
		return sd.detectStorageDarwin()
	case "windows":
		return sd.detectStorageWindows()
	default:
		// Fallback
		storage.Total = 100 * 1024 * 1024 * 1024      // 100GB
		storage.Available = 50 * 1024 * 1024 * 1024   // 50GB
		return storage, nil
	}
}

func (sd *SystemDetector) detectStorageLinux() (StorageInfo, error) {
	storage := StorageInfo{}

	cmd := exec.Command("df", "-B1", "/")
	output, err := cmd.Output()
	if err != nil {
		return storage, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			if total, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				storage.Total = total
			}
			if available, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
				storage.Available = available
			}
		}
	}

	// Try to detect storage type
	if _, err := os.Stat("/sys/block"); err == nil {
		storage.Type = "ssd" // Assume SSD for modern systems
	}

	return storage, nil
}

func (sd *SystemDetector) detectStorageDarwin() (StorageInfo, error) {
	storage := StorageInfo{}

	cmd := exec.Command("df", "-b", "/")
	output, err := cmd.Output()
	if err != nil {
		return storage, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			if total, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				storage.Total = total
			}
			if available, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
				storage.Available = available
			}
		}
	}

	storage.Type = "ssd" // Most modern Macs have SSDs

	return storage, nil
}

func (sd *SystemDetector) detectStorageWindows() (StorageInfo, error) {
	storage := StorageInfo{}

	cmd := exec.Command("wmic", "logicaldisk", "where", "caption=\"C:\"", "get", "size,freespace", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return storage, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				if available, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					storage.Available = available
				}
				if total, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64); err == nil {
					storage.Total = total
				}
				break
			}
		}
	}

	storage.Type = "ssd" // Assume SSD for modern systems

	return storage, nil
}

func (sd *SystemDetector) detectNetwork() (NetworkInfo, error) {
	network := NetworkInfo{
		Bandwidth: 100,  // Default to 100 Mbps
		Latency:   10,   // Default to 10ms
		Type:      "ethernet",
	}

	// This is a simplified implementation
	// In a real system, you would query network interfaces and test bandwidth
	return network, nil
}

func (sd *SystemDetector) detectContainerSupport() (ContainerSupport, error) {
	support := ContainerSupport{}

	// Check for Docker
	if _, err := exec.LookPath("docker"); err == nil {
		// Verify Docker is running
		cmd := exec.Command("docker", "version")
		if err := cmd.Run(); err == nil {
			support.Docker = true
		}
	}

	// Check for Podman
	if _, err := exec.LookPath("podman"); err == nil {
		// Verify Podman is working
		cmd := exec.Command("podman", "version")
		if err := cmd.Run(); err == nil {
			support.Podman = true
		}
	}

	return support, nil
}