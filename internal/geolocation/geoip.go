package geolocation

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeoLocation represents the geographic location of an IP address
type GeoLocation struct {
	IP          string  `json:"query"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Continent   string  `json:"continent"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Status      string  `json:"status"`
}

// GeoIPService handles IP geolocation lookups and caching
type GeoIPService struct {
	db          *sql.DB
	cache       map[string]*GeoLocation
	cacheMutex  sync.RWMutex
	httpClient  *http.Client
	rateLimiter chan struct{}
	lastRequest time.Time
	requestMu   sync.Mutex
}

// NewGeoIPService creates a new geolocation service
func NewGeoIPService(db *sql.DB) *GeoIPService {
	return &GeoIPService{
		db:    db,
		cache: make(map[string]*GeoLocation),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		rateLimiter: make(chan struct{}, 1), // Allow 1 concurrent request
	}
}

// LookupIP looks up the geographic location of an IP address
// Uses ip-api.com free tier (45 requests per minute)
func (s *GeoIPService) LookupIP(ipAddr string) (*GeoLocation, error) {
	// Extract IP from address (remove port if present)
	ip := extractIP(ipAddr)
	if ip == "" {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddr)
	}

	// Skip private/local IPs
	if isPrivateIP(ip) {
		return nil, fmt.Errorf("private IP address: %s", ip)
	}

	// Check cache first
	s.cacheMutex.RLock()
	if cached, ok := s.cache[ip]; ok {
		s.cacheMutex.RUnlock()
		return cached, nil
	}
	s.cacheMutex.RUnlock()

	// Rate limit: ip-api.com allows 45 requests per minute
	s.requestMu.Lock()
	elapsed := time.Since(s.lastRequest)
	if elapsed < 1400*time.Millisecond { // ~43 requests per minute to be safe
		time.Sleep(1400*time.Millisecond - elapsed)
	}
	s.lastRequest = time.Now()
	s.requestMu.Unlock()

	// Query ip-api.com (free, no API key needed)
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,continent,country,countryCode,city,lat,lon,isp,query", ip)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("geolocation API error: %w", err)
	}
	defer resp.Body.Close()

	var geo GeoLocation
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return nil, fmt.Errorf("failed to decode geolocation response: %w", err)
	}

	if geo.Status != "success" {
		return nil, fmt.Errorf("geolocation lookup failed for IP %s", ip)
	}

	// Cache the result
	s.cacheMutex.Lock()
	s.cache[ip] = &geo
	s.cacheMutex.Unlock()

	log.Printf("📍 Geolocated miner IP %s: %s, %s (%s)", ip, geo.City, geo.Country, geo.CountryCode)

	return &geo, nil
}

// UpdateMinerLocation updates the miner's location in the database
func (s *GeoIPService) UpdateMinerLocation(minerID int64, ipAddr string) error {
	geo, err := s.LookupIP(ipAddr)
	if err != nil {
		// Not a fatal error - just log and continue
		log.Printf("⚠️ Could not geolocate miner %d (%s): %v", minerID, ipAddr, err)
		return nil
	}

	ip := extractIP(ipAddr)

	_, err = s.db.Exec(`
		UPDATE miners SET 
			ip_address = $1,
			city = $2,
			country = $3,
			country_code = $4,
			continent = $5,
			latitude = $6,
			longitude = $7,
			location_updated_at = NOW()
		WHERE id = $8`,
		ip, geo.City, geo.Country, geo.CountryCode, geo.Continent,
		geo.Latitude, geo.Longitude, minerID,
	)

	if err != nil {
		return fmt.Errorf("failed to update miner location: %w", err)
	}

	log.Printf("✅ Updated location for miner %d: %s, %s", minerID, geo.City, geo.Country)
	return nil
}

// UpdateMinerLocationByUserAndName updates location using user_id and miner name
func (s *GeoIPService) UpdateMinerLocationByUserAndName(userID int64, minerName, ipAddr string) error {
	geo, err := s.LookupIP(ipAddr)
	if err != nil {
		// Not a fatal error - just log and continue
		return nil
	}

	ip := extractIP(ipAddr)

	_, err = s.db.Exec(`
		UPDATE miners SET 
			ip_address = $1,
			city = $2,
			country = $3,
			country_code = $4,
			continent = $5,
			latitude = $6,
			longitude = $7,
			location_updated_at = NOW()
		WHERE user_id = $8 AND name = $9`,
		ip, geo.City, geo.Country, geo.CountryCode, geo.Continent,
		geo.Latitude, geo.Longitude, userID, minerName,
	)

	if err != nil {
		return fmt.Errorf("failed to update miner location: %w", err)
	}

	return nil
}

// extractIP extracts the IP address from an address string (removes port)
func extractIP(addr string) string {
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(addr, "[") {
		if idx := strings.Index(addr, "]"); idx != -1 {
			return addr[1:idx]
		}
	}

	// Handle host:port format
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// Maybe it's just an IP without port
		if net.ParseIP(addr) != nil {
			return addr
		}
		return ""
	}
	return host
}

// isPrivateIP checks if an IP is private/local
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true // Invalid IP, treat as private
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for private ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",  // IPv6 private
		"fe80::/10", // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// GetCacheStats returns cache statistics
func (s *GeoIPService) GetCacheStats() (int, int) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return len(s.cache), 0 // cached count, misses not tracked
}
