package geoip

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Location represents geographic location data from GeoIP lookup
type Location struct {
	IP          string  `json:"ip"`
	City        string  `json:"city"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Continent   string  `json:"continent"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// GeoIPService interface for location lookups (ISP-compliant)
type GeoIPService interface {
	Lookup(ip string) (*Location, error)
	LookupWithCache(ip string) (*Location, error)
}

// ipAPIResponse represents the response from ip-api.com
type ipAPIResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Query       string  `json:"query"`
}

// Service implements GeoIPService using ip-api.com (free, no API key required)
type Service struct {
	client    *http.Client
	cache     map[string]*cachedLocation
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration
	rateLimit *rateLimiter
}

type cachedLocation struct {
	location  *Location
	expiresAt time.Time
}

type rateLimiter struct {
	mu        sync.Mutex
	tokens    int
	maxTokens int
	refillAt  time.Time
}

// NewService creates a new GeoIP service
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:    make(map[string]*cachedLocation),
		cacheTTL: 24 * time.Hour, // Cache locations for 24 hours
		rateLimit: &rateLimiter{
			tokens:    45, // ip-api.com allows 45 requests per minute
			maxTokens: 45,
			refillAt:  time.Now().Add(time.Minute),
		},
	}
}

// Lookup performs a GeoIP lookup for the given IP address
func (s *Service) Lookup(ip string) (*Location, error) {
	// Validate IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// Skip private/local IPs
	if isPrivateIP(parsedIP) {
		return &Location{
			IP:          ip,
			City:        "Local",
			Country:     "Local Network",
			CountryCode: "LO",
			Continent:   "Unknown",
			Latitude:    0,
			Longitude:   0,
		}, nil
	}

	// Check rate limit
	if !s.checkRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded, please wait")
	}

	// Query ip-api.com
	resp, err := s.client.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
	if err != nil {
		return nil, fmt.Errorf("failed to query GeoIP API: %w", err)
	}
	defer resp.Body.Close()

	var apiResp ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode GeoIP response: %w", err)
	}

	if apiResp.Status != "success" {
		return nil, fmt.Errorf("GeoIP lookup failed for IP: %s", ip)
	}

	return &Location{
		IP:          ip,
		City:        apiResp.City,
		Country:     apiResp.Country,
		CountryCode: apiResp.CountryCode,
		Continent:   getContinent(apiResp.CountryCode),
		Latitude:    apiResp.Lat,
		Longitude:   apiResp.Lon,
	}, nil
}

// LookupWithCache performs a cached GeoIP lookup
func (s *Service) LookupWithCache(ip string) (*Location, error) {
	// Check cache first
	s.cacheMu.RLock()
	if cached, ok := s.cache[ip]; ok && time.Now().Before(cached.expiresAt) {
		s.cacheMu.RUnlock()
		return cached.location, nil
	}
	s.cacheMu.RUnlock()

	// Perform lookup
	location, err := s.Lookup(ip)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cacheMu.Lock()
	s.cache[ip] = &cachedLocation{
		location:  location,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMu.Unlock()

	return location, nil
}

func (s *Service) checkRateLimit() bool {
	s.rateLimit.mu.Lock()
	defer s.rateLimit.mu.Unlock()

	// Refill tokens if needed
	if time.Now().After(s.rateLimit.refillAt) {
		s.rateLimit.tokens = s.rateLimit.maxTokens
		s.rateLimit.refillAt = time.Now().Add(time.Minute)
	}

	if s.rateLimit.tokens > 0 {
		s.rateLimit.tokens--
		return true
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	// Check for private IPv4 ranges
	private := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
	}

	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}

	// Check for loopback
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}

	return false
}

// getContinent maps country codes to continents
func getContinent(countryCode string) string {
	continents := map[string]string{
		// North America
		"US": "North America", "CA": "North America", "MX": "North America",
		"GT": "North America", "BZ": "North America", "HN": "North America",
		"SV": "North America", "NI": "North America", "CR": "North America",
		"PA": "North America", "CU": "North America", "JM": "North America",
		"HT": "North America", "DO": "North America", "PR": "North America",
		// South America
		"BR": "South America", "AR": "South America", "CO": "South America",
		"PE": "South America", "VE": "South America", "CL": "South America",
		"EC": "South America", "BO": "South America", "PY": "South America",
		"UY": "South America", "GY": "South America", "SR": "South America",
		// Europe
		"GB": "Europe", "DE": "Europe", "FR": "Europe", "IT": "Europe",
		"ES": "Europe", "PT": "Europe", "NL": "Europe", "BE": "Europe",
		"CH": "Europe", "AT": "Europe", "PL": "Europe", "SE": "Europe",
		"NO": "Europe", "DK": "Europe", "FI": "Europe", "IE": "Europe",
		"CZ": "Europe", "RO": "Europe", "HU": "Europe", "GR": "Europe",
		"UA": "Europe", "RU": "Europe", "BY": "Europe", "SK": "Europe",
		"BG": "Europe", "HR": "Europe", "RS": "Europe", "SI": "Europe",
		"LT": "Europe", "LV": "Europe", "EE": "Europe", "IS": "Europe",
		"LU": "Europe", "MT": "Europe", "CY": "Europe", "AL": "Europe",
		// Asia
		"CN": "Asia", "JP": "Asia", "KR": "Asia", "IN": "Asia",
		"ID": "Asia", "TH": "Asia", "VN": "Asia", "PH": "Asia",
		"MY": "Asia", "SG": "Asia", "TW": "Asia", "HK": "Asia",
		"PK": "Asia", "BD": "Asia", "NP": "Asia", "LK": "Asia",
		"MM": "Asia", "KH": "Asia", "LA": "Asia", "MN": "Asia",
		"KZ": "Asia", "UZ": "Asia", "AE": "Asia", "SA": "Asia",
		"IL": "Asia", "TR": "Asia", "IR": "Asia", "IQ": "Asia",
		// Africa
		"ZA": "Africa", "EG": "Africa", "NG": "Africa", "KE": "Africa",
		"MA": "Africa", "DZ": "Africa", "TN": "Africa", "GH": "Africa",
		"ET": "Africa", "TZ": "Africa", "UG": "Africa", "ZW": "Africa",
		"SN": "Africa", "CI": "Africa", "CM": "Africa", "AO": "Africa",
		// Oceania
		"AU": "Oceania", "NZ": "Oceania", "PG": "Oceania", "FJ": "Oceania",
		"NC": "Oceania", "PF": "Oceania", "WS": "Oceania", "VU": "Oceania",
	}

	if continent, ok := continents[countryCode]; ok {
		return continent
	}
	return "Unknown"
}
