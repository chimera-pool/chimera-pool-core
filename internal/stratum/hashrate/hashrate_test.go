package hashrate

import (
	"testing"
	"time"
)

// TestCalculateHashrate tests hashrate calculation from shares
func TestCalculateHashrate(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name       string
		shares     int64
		difficulty float64
		duration   time.Duration
		expected   float64 // Expected hashrate in H/s
		tolerance  float64 // Acceptable variance
	}{
		{
			name:       "basic calculation",
			shares:     10,
			difficulty: 1.0,
			duration:   10 * time.Second,
			expected:   4294967296, // 2^32 H/s at diff 1, 1 share/sec
			tolerance:  0.01,
		},
		{
			name:       "low difficulty high share rate",
			shares:     100,
			difficulty: 0.01,
			duration:   10 * time.Second,
			expected:   429496729.6, // 10 shares/sec * 0.01 diff * 2^32
			tolerance:  0.01,
		},
		{
			name:       "high difficulty low share rate",
			shares:     1,
			difficulty: 100.0,
			duration:   100 * time.Second,
			expected:   4294967296, // 0.01 shares/sec * 100 diff * 2^32
			tolerance:  0.01,
		},
		{
			name:       "zero shares",
			shares:     0,
			difficulty: 1.0,
			duration:   10 * time.Second,
			expected:   0,
			tolerance:  0,
		},
		{
			name:       "zero duration",
			shares:     10,
			difficulty: 1.0,
			duration:   0,
			expected:   0,
			tolerance:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.Calculate(tt.shares, tt.difficulty, tt.duration)

			if tt.expected == 0 {
				if result != 0 {
					t.Errorf("expected 0, got %f", result)
				}
				return
			}

			variance := (result - tt.expected) / tt.expected
			if variance < -tt.tolerance || variance > tt.tolerance {
				t.Errorf("expected ~%f H/s, got %f H/s (variance: %.2f%%)",
					tt.expected, result, variance*100)
			}
		})
	}
}

// TestFormatHashrate tests human-readable hashrate formatting
func TestFormatHashrate(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		hashrate float64
		expected string
	}{
		{0, "0.00 H/s"},
		{500, "500.00 H/s"},
		{1500, "1.50 KH/s"},
		{1500000, "1.50 MH/s"},
		{1500000000, "1.50 GH/s"},
		{1500000000000, "1.50 TH/s"},
		{1500000000000000, "1.50 PH/s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := calc.Format(tt.hashrate)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestHashrateWindow tests rolling window hashrate calculation
func TestHashrateWindow(t *testing.T) {
	window := NewWindow(5 * time.Minute)

	// Add shares
	now := time.Now()
	window.AddShare(1.0, now.Add(-4*time.Minute))
	window.AddShare(1.0, now.Add(-3*time.Minute))
	window.AddShare(1.0, now.Add(-2*time.Minute))
	window.AddShare(1.0, now.Add(-1*time.Minute))
	window.AddShare(1.0, now)

	// 5 shares at diff 1.0 over 5 minutes = 5 * 2^32 / 300 seconds
	hashrate := window.GetHashrate()
	expected := 5.0 * 4294967296.0 / 300.0

	variance := (hashrate - expected) / expected
	if variance < -0.1 || variance > 0.1 {
		t.Errorf("expected ~%f H/s, got %f H/s", expected, hashrate)
	}
}

// TestHashrateWindowExpiry tests that old shares are removed
func TestHashrateWindowExpiry(t *testing.T) {
	window := NewWindow(1 * time.Minute)

	// Add old share (should be expired)
	window.AddShare(1.0, time.Now().Add(-2*time.Minute))

	// Add recent share
	window.AddShare(1.0, time.Now())

	// Only recent share should count
	shares := window.GetShareCount()
	if shares != 1 {
		t.Errorf("expected 1 share (old one expired), got %d", shares)
	}
}

// TestHashrateWindowConcurrent tests thread safety
func TestHashrateWindowConcurrent(t *testing.T) {
	window := NewWindow(5 * time.Minute)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				window.AddShare(0.01, time.Now())
				window.GetHashrate()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 1000 shares
	if window.GetShareCount() != 1000 {
		t.Errorf("expected 1000 shares, got %d", window.GetShareCount())
	}
}
