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

type HardwareDetector struct{}

func NewHardwareDetector() *HardwareDetector {
	return &HardwareDetector{}
}

func (hd *HardwareDetector) DetectHardware() (HardwareInfo, error) {
	hardware := HardwareInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	var err error

	// Detect CPU information
	hardware.CPUs, err = hd.detectCPUs()
	if err != nil {
		return hardware, fmt.Errorf("failed to detect CPUs: %w", err)
	}

	// Detect GPU information
	hardware.GPUs, err = hd.detectGPUs()
	if err != nil {
		// GPU detection failure is not critical
		hardware.GPUs = []GPUInfo{}
	}

	// Detect memory information
	hardware.Memory, err = hd.detectMemory()
	if err != nil {
		return hardware, fmt.Errorf("failed to detect memory: %w", err)
	}

	return hardware, nil
}

func (hd *HardwareDetector) detectCPUs() ([]CPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return hd.detectCPUsLinux()
	case "darwin":
		return hd.detectCPUsDarwin()
	case "windows":
		return hd.detectCPUsWindows()
	default:
		// Fallback
		return []CPUInfo{{
			Cores:        runtime.NumCPU(),
			Threads:      runtime.NumCPU(),
			Architecture: runtime.GOARCH,
			Model:        "Unknown CPU",
		}}, nil
	}
}

func (hd *HardwareDetector) detectCPUsLinux() ([]CPUInfo, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cpuMap := make(map[int]*CPUInfo)
	scanner := bufio.NewScanner(file)
	currentProcessor := -1

	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "processor") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if proc, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					currentProcessor = proc
					if cpuMap[currentProcessor] == nil {
						cpuMap[currentProcessor] = &CPUInfo{
							Architecture: runtime.GOARCH,
							Threads:      1,
						}
					}
				}
			}
		} else if currentProcessor >= 0 {
			if strings.HasPrefix(line, "model name") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					cpuMap[currentProcessor].Model = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "cpu cores") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					if cores, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
						cpuMap[currentProcessor].Cores = cores
					}
				}
			} else if strings.HasPrefix(line, "siblings") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					if threads, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
						cpuMap[currentProcessor].Threads = threads
					}
				}
			}
		}
	}

	// Convert map to slice and deduplicate
	seenCPUs := make(map[string]*CPUInfo)
	for _, cpu := range cpuMap {
		if cpu.Model != "" {
			key := fmt.Sprintf("%s_%d_%d", cpu.Model, cpu.Cores, cpu.Threads)
			if existing, exists := seenCPUs[key]; exists {
				// This is a duplicate, skip
				continue
			} else {
				seenCPUs[key] = cpu
			}
		}
	}

	var cpus []CPUInfo
	for _, cpu := range seenCPUs {
		cpus = append(cpus, *cpu)
	}

	if len(cpus) == 0 {
		// Fallback
		cpus = append(cpus, CPUInfo{
			Cores:        runtime.NumCPU(),
			Threads:      runtime.NumCPU(),
			Architecture: runtime.GOARCH,
			Model:        "Unknown CPU",
		})
	}

	return cpus, scanner.Err()
}

func (hd *HardwareDetector) detectCPUsDarwin() ([]CPUInfo, error) {
	cpu := CPUInfo{
		Architecture: runtime.GOARCH,
	}

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

	return []CPUInfo{cpu}, nil
}

func (hd *HardwareDetector) detectCPUsWindows() ([]CPUInfo, error) {
	cmd := exec.Command("wmic", "cpu", "get", "Name,NumberOfCores,NumberOfLogicalProcessors", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var cpus []CPUInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				cpu := CPUInfo{
					Architecture: runtime.GOARCH,
				}
				
				cpu.Model = strings.TrimSpace(parts[1])
				if cores, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
					cpu.Cores = cores
				}
				if threads, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
					cpu.Threads = threads
				}
				
				if cpu.Model != "" {
					cpus = append(cpus, cpu)
				}
			}
		}
	}

	if len(cpus) == 0 {
		// Fallback
		cpus = append(cpus, CPUInfo{
			Cores:        runtime.NumCPU(),
			Threads:      runtime.NumCPU(),
			Architecture: runtime.GOARCH,
			Model:        "Unknown CPU",
		})
	}

	return cpus, nil
}

func (hd *HardwareDetector) detectGPUs() ([]GPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return hd.detectGPUsLinux()
	case "darwin":
		return hd.detectGPUsDarwin()
	case "windows":
		return hd.detectGPUsWindows()
	default:
		return []GPUInfo{}, nil
	}
}

func (hd *HardwareDetector) detectGPUsLinux() ([]GPUInfo, error) {
	var gpus []GPUInfo

	// Try nvidia-smi first
	nvidiaGPUs, err := hd.detectNVIDIAGPUs()
	if err == nil {
		gpus = append(gpus, nvidiaGPUs...)
	}

	// Try AMD GPUs
	amdGPUs, err := hd.detectAMDGPUs()
	if err == nil {
		gpus = append(gpus, amdGPUs...)
	}

	// Try Intel GPUs
	intelGPUs, err := hd.detectIntelGPUs()
	if err == nil {
		gpus = append(gpus, intelGPUs...)
	}

	return gpus, nil
}

func (hd *HardwareDetector) detectNVIDIAGPUs() ([]GPUInfo, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version,pci.bus_id", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ",")
		if len(parts) >= 4 {
			gpu := GPUInfo{
				Model:    strings.TrimSpace(parts[0]),
				Driver:   strings.TrimSpace(parts[2]),
				Vendor:   "NVIDIA",
				PCIeSlot: strings.TrimSpace(parts[3]),
			}
			
			// Parse memory (in MB)
			if memMB, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
				gpu.Memory = memMB * 1024 * 1024 // Convert to bytes
			}
			
			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

func (hd *HardwareDetector) detectAMDGPUs() ([]GPUInfo, error) {
	// Try rocm-smi
	cmd := exec.Command("rocm-smi", "--showproductname", "--showmeminfo", "vram")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "GPU") && strings.Contains(line, ":") {
			// This is a simplified parser - real implementation would be more robust
			gpu := GPUInfo{
				Vendor: "AMD",
				Driver: "amdgpu",
			}
			
			// Extract GPU name (simplified)
			if strings.Contains(line, "Radeon") || strings.Contains(line, "RX") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					gpu.Model = strings.TrimSpace(parts[1])
				}
			}
			
			if gpu.Model != "" {
				gpus = append(gpus, gpu)
			}
		}
	}

	return gpus, nil
}

func (hd *HardwareDetector) detectIntelGPUs() ([]GPUInfo, error) {
	// Intel GPU detection is more complex and varies by generation
	// This is a placeholder implementation
	return []GPUInfo{}, nil
}

func (hd *HardwareDetector) detectGPUsDarwin() ([]GPUInfo, error) {
	// Use system_profiler to get GPU information
	cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// This would parse the JSON output to extract GPU information
	// For now, return empty slice
	return []GPUInfo{}, nil
}

func (hd *HardwareDetector) detectGPUsWindows() ([]GPUInfo, error) {
	cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "Name,AdapterRAM,DriverVersion,PNPDeviceID", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, ",") && !strings.Contains(line, "Node") {
			parts := strings.Split(line, ",")
			if len(parts) >= 5 {
				gpu := GPUInfo{
					Model:  strings.TrimSpace(parts[3]),
					Driver: strings.TrimSpace(parts[2]),
				}
				
				// Parse memory
				if memBytes, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					gpu.Memory = memBytes
				}
				
				// Determine vendor from model name
				modelLower := strings.ToLower(gpu.Model)
				if strings.Contains(modelLower, "nvidia") || strings.Contains(modelLower, "geforce") || strings.Contains(modelLower, "quadro") {
					gpu.Vendor = "NVIDIA"
				} else if strings.Contains(modelLower, "amd") || strings.Contains(modelLower, "radeon") {
					gpu.Vendor = "AMD"
				} else if strings.Contains(modelLower, "intel") {
					gpu.Vendor = "Intel"
				}
				
				if gpu.Model != "" && gpu.Memory > 0 {
					gpus = append(gpus, gpu)
				}
			}
		}
	}

	return gpus, nil
}

func (hd *HardwareDetector) detectMemory() (MemoryInfo, error) {
	// Reuse the memory detection from SystemDetector
	detector := NewSystemDetector()
	return detector.detectMemory()
}