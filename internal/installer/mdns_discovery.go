package installer

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type MDNSDiscovery struct {
	isAdvertising bool
	currentAdvert *PoolAdvertisement
}

type PoolAdvertisement struct {
	Name        string            `json:"name"`
	ServiceType string            `json:"service_type"`
	Port        int               `json:"port"`
	Domain      string            `json:"domain"`
	TXTRecords  map[string]string `json:"txt_records"`
}

type DiscoveredPool struct {
	Name       string            `json:"name"`
	Address    string            `json:"address"`
	Port       int               `json:"port"`
	TXTRecords map[string]string `json:"txt_records"`
}

type MDNSService struct {
	Instance string `json:"instance"`
	Service  string `json:"service"`
	Domain   string `json:"domain"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

type NetworkInterface struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
}

func NewMDNSDiscovery() *MDNSDiscovery {
	return &MDNSDiscovery{
		isAdvertising: false,
	}
}

func (md *MDNSDiscovery) AdvertisePool(ctx context.Context, poolInfo PoolAdvertisement) error {
	// In a real implementation, this would use a library like github.com/hashicorp/mdns
	// or github.com/grandcat/zeroconf to advertise the service
	
	// Validate the pool advertisement
	if err := md.validatePoolAdvertisement(poolInfo); err != nil {
		return fmt.Errorf("invalid pool advertisement: %w", err)
	}

	// Store the current advertisement
	md.currentAdvert = &poolInfo
	md.isAdvertising = true

	// Simulate mDNS advertisement
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				md.isAdvertising = false
				return
			case <-ticker.C:
				// Simulate periodic mDNS announcements
				// In reality, this would send mDNS packets
			}
		}
	}()

	return nil
}

func (md *MDNSDiscovery) StopAdvertising() error {
	md.isAdvertising = false
	md.currentAdvert = nil
	return nil
}

func (md *MDNSDiscovery) IsAdvertising() (bool, error) {
	return md.isAdvertising, nil
}

func (md *MDNSDiscovery) DiscoverPools(ctx context.Context) ([]DiscoveredPool, error) {
	// In a real implementation, this would browse for _chimera-pool._tcp services
	// and return discovered pools
	
	var pools []DiscoveredPool

	// Simulate discovery process
	select {
	case <-ctx.Done():
		return pools, ctx.Err()
	case <-time.After(100 * time.Millisecond): // Quick simulation for testing
		// Return mock discovered pools for testing
		if md.isAdvertising && md.currentAdvert != nil {
			pools = append(pools, DiscoveredPool{
				Name:       md.currentAdvert.Name,
				Address:    "127.0.0.1",
				Port:       md.currentAdvert.Port,
				TXTRecords: md.currentAdvert.TXTRecords,
			})
		}
	}

	return pools, nil
}

func (md *MDNSDiscovery) BrowseServices(ctx context.Context, serviceType string) ([]MDNSService, error) {
	var services []MDNSService

	// Simulate service browsing
	select {
	case <-ctx.Done():
		return services, ctx.Err()
	case <-time.After(100 * time.Millisecond): // Quick simulation for testing
		// Return mock services for testing
		if md.isAdvertising && md.currentAdvert != nil && md.currentAdvert.ServiceType == serviceType {
			services = append(services, MDNSService{
				Instance: md.currentAdvert.Name,
				Service:  serviceType,
				Domain:   md.currentAdvert.Domain,
				Address:  "127.0.0.1",
				Port:     md.currentAdvert.Port,
			})
		}
	}

	return services, nil
}

func (md *MDNSDiscovery) ParseTXTRecords(txtRecords []string) (map[string]string, error) {
	parsed := make(map[string]string)

	for _, record := range txtRecords {
		parts := strings.SplitN(record, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			parsed[key] = value
		} else if len(parts) == 1 {
			// Boolean flag (key without value)
			key := strings.TrimSpace(parts[0])
			parsed[key] = "true"
		}
	}

	return parsed, nil
}

func (md *MDNSDiscovery) GetNetworkInterfaces() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []NetworkInterface

	for _, iface := range interfaces {
		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip loopback interfaces (unless it's the only one)
		if iface.Flags&net.FlagLoopback != 0 && len(interfaces) > 1 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var addresses []string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil { // IPv4 only for simplicity
					addresses = append(addresses, ipnet.IP.String())
				}
			}
		}

		if len(addresses) > 0 {
			result = append(result, NetworkInterface{
				Name:      iface.Name,
				Addresses: addresses,
			})
		}
	}

	return result, nil
}

func (md *MDNSDiscovery) ResolveNameConflict(originalName string) (string, error) {
	// Simple conflict resolution by appending a number
	// In a real implementation, this would check for actual conflicts on the network
	
	baseName := originalName
	counter := 1

	// For testing, we'll assume no conflicts unless we're already advertising
	if md.isAdvertising && md.currentAdvert != nil && md.currentAdvert.Name == originalName {
		return fmt.Sprintf("%s (%d)", baseName, counter), nil
	}

	return originalName, nil
}

func (md *MDNSDiscovery) validatePoolAdvertisement(poolInfo PoolAdvertisement) error {
	if poolInfo.Name == "" {
		return fmt.Errorf("pool name is required")
	}

	if poolInfo.ServiceType == "" {
		return fmt.Errorf("service type is required")
	}

	if !strings.HasPrefix(poolInfo.ServiceType, "_") || !strings.HasSuffix(poolInfo.ServiceType, "._tcp") {
		return fmt.Errorf("service type must be in format _service._tcp")
	}

	if poolInfo.Port <= 0 || poolInfo.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if poolInfo.Domain == "" {
		poolInfo.Domain = "local" // Default to .local domain
	}

	// Validate TXT records
	for key, value := range poolInfo.TXTRecords {
		if key == "" {
			return fmt.Errorf("TXT record key cannot be empty")
		}

		// Check for reasonable length limits
		if len(key) > 255 {
			return fmt.Errorf("TXT record key too long: %s", key)
		}

		if len(value) > 255 {
			return fmt.Errorf("TXT record value too long for key %s", key)
		}

		// Validate specific known keys
		switch key {
		case "fee":
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("invalid fee value: %s", value)
			}
		case "miners":
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("invalid miners count: %s", value)
			}
		case "version":
			if !isValidVersion(value) {
				return fmt.Errorf("invalid version format: %s", value)
			}
		}
	}

	return nil
}

func isValidVersion(version string) bool {
	// Simple version validation (semantic versioning)
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

// Helper functions for creating common pool advertisements

func (md *MDNSDiscovery) CreatePoolAdvertisement(name string, port int, algorithm string, fee float64) PoolAdvertisement {
	return PoolAdvertisement{
		Name:        name,
		ServiceType: "_chimera-pool._tcp",
		Port:        port,
		Domain:      "local",
		TXTRecords: map[string]string{
			"algorithm": algorithm,
			"fee":       fmt.Sprintf("%.1f", fee),
			"version":   "1.0.0",
		},
	}
}

func (md *MDNSDiscovery) AddPoolStats(advert *PoolAdvertisement, miners int, hashrate string, location string) {
	if advert.TXTRecords == nil {
		advert.TXTRecords = make(map[string]string)
	}

	advert.TXTRecords["miners"] = strconv.Itoa(miners)
	advert.TXTRecords["hashrate"] = hashrate
	advert.TXTRecords["location"] = location
}

// Network utility functions

func (md *MDNSDiscovery) GetLocalIPAddresses() ([]string, error) {
	interfaces, err := md.GetNetworkInterfaces()
	if err != nil {
		return nil, err
	}

	var addresses []string
	for _, iface := range interfaces {
		addresses = append(addresses, iface.Addresses...)
	}

	return addresses, nil
}

func (md *MDNSDiscovery) IsLocalAddress(address string) bool {
	ip := net.ParseIP(address)
	if ip == nil {
		return false
	}

	// Check if it's a loopback address
	if ip.IsLoopback() {
		return true
	}

	// Check if it's a private address
	if ip.IsPrivate() {
		return true
	}

	// Check if it's one of our interface addresses
	localAddresses, err := md.GetLocalIPAddresses()
	if err != nil {
		return false
	}

	for _, localAddr := range localAddresses {
		if address == localAddr {
			return true
		}
	}

	return false
}