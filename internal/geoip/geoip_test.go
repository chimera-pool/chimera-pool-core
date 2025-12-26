package geoip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	svc := NewService()
	require.NotNil(t, svc)
	assert.NotNil(t, svc.client)
	assert.NotNil(t, svc.cache)
	assert.NotNil(t, svc.rateLimit)
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"private 10.x", "10.0.0.1", true},
		{"private 172.x", "172.16.0.1", true},
		{"private 192.x", "192.168.1.1", true},
		{"loopback", "127.0.0.1", true},
		{"public google", "8.8.8.8", false},
		{"public cloudflare", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			result := isPrivateIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetContinent(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"US", "North America"},
		{"GB", "Europe"},
		{"JP", "Asia"},
		{"AU", "Oceania"},
		{"BR", "South America"},
		{"ZA", "Africa"},
		{"XX", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := getContinent(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLookupPrivateIP(t *testing.T) {
	svc := NewService()

	location, err := svc.Lookup("192.168.1.1")
	require.NoError(t, err)
	assert.Equal(t, "Local", location.City)
	assert.Equal(t, "Local Network", location.Country)
}

func TestLookupInvalidIP(t *testing.T) {
	svc := NewService()

	_, err := svc.Lookup("invalid-ip")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid IP address")
}

func parseIP(s string) net.IP {
	return net.ParseIP(s)
}
