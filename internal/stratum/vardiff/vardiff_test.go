package vardiff

import (
	"testing"
	"time"
)

// TestVardiffConfig tests configuration validation
func TestVardiffConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectValid bool
	}{
		{
			name: "valid config",
			config: Config{
				TargetShareTime:  10 * time.Second,
				RetargetInterval: 30 * time.Second,
				VariancePercent:  30,
				MinDifficulty:    0.001,
				MaxDifficulty:    1000000,
			},
			expectValid: true,
		},
		{
			name: "invalid - zero target time",
			config: Config{
				TargetShareTime:  0,
				RetargetInterval: 30 * time.Second,
				VariancePercent:  30,
				MinDifficulty:    0.001,
				MaxDifficulty:    1000000,
			},
			expectValid: false,
		},
		{
			name: "invalid - min > max difficulty",
			config: Config{
				TargetShareTime:  10 * time.Second,
				RetargetInterval: 30 * time.Second,
				VariancePercent:  30,
				MinDifficulty:    1000,
				MaxDifficulty:    100,
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectValid && err != nil {
				t.Errorf("expected valid config, got error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Errorf("expected invalid config, got no error")
			}
		})
	}
}

// TestVardiffManager_InitialDifficulty tests getting initial difficulty
func TestVardiffManager_InitialDifficulty(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// New miner should get initial difficulty
	diff := manager.GetDifficulty("miner1")
	if diff != config.InitialDifficulty {
		t.Errorf("expected initial difficulty %f, got %f", config.InitialDifficulty, diff)
	}
}

// TestVardiffManager_SetDifficulty tests setting difficulty manually
func TestVardiffManager_SetDifficulty(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// Set custom difficulty
	err := manager.SetDifficulty("miner1", 0.5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	diff := manager.GetDifficulty("miner1")
	if diff != 0.5 {
		t.Errorf("expected difficulty 0.5, got %f", diff)
	}
}

// TestVardiffManager_SetDifficulty_Bounds tests difficulty bounds
func TestVardiffManager_SetDifficulty_Bounds(t *testing.T) {
	config := DefaultConfig()
	config.MinDifficulty = 0.01
	config.MaxDifficulty = 100
	manager := NewManager(config)

	// Try to set below minimum
	err := manager.SetDifficulty("miner1", 0.001)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	diff := manager.GetDifficulty("miner1")
	if diff < config.MinDifficulty {
		t.Errorf("difficulty %f below minimum %f", diff, config.MinDifficulty)
	}

	// Try to set above maximum
	err = manager.SetDifficulty("miner2", 1000)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	diff = manager.GetDifficulty("miner2")
	if diff > config.MaxDifficulty {
		t.Errorf("difficulty %f above maximum %f", diff, config.MaxDifficulty)
	}
}

// TestVardiffManager_AdjustDifficulty_TooFast tests difficulty increase when shares are too fast
func TestVardiffManager_AdjustDifficulty_TooFast(t *testing.T) {
	config := DefaultConfig()
	config.TargetShareTime = 10 * time.Second
	config.VariancePercent = 30 // 30% variance allowed
	config.InitialDifficulty = 1.0
	config.RetargetInterval = 1 * time.Millisecond // Fast retarget for testing
	config.ShareWindow = 3                         // Small window for testing
	manager := NewManager(config)

	// Set initial difficulty
	manager.SetDifficulty("miner1", 1.0)

	// Record shares that came too fast (2 seconds instead of 10)
	for i := 0; i < 5; i++ {
		manager.RecordShare("miner1", 2*time.Second)
		time.Sleep(2 * time.Millisecond) // Allow retarget interval to pass
	}

	newDiff := manager.GetDifficulty("miner1")
	if newDiff <= config.InitialDifficulty {
		t.Errorf("expected difficulty to increase from %f, got %f", config.InitialDifficulty, newDiff)
	}
}

// TestVardiffManager_AdjustDifficulty_TooSlow tests difficulty decrease when shares are too slow
func TestVardiffManager_AdjustDifficulty_TooSlow(t *testing.T) {
	config := DefaultConfig()
	config.TargetShareTime = 10 * time.Second
	config.VariancePercent = 30
	config.InitialDifficulty = 1.0
	manager := NewManager(config)

	// Set initial difficulty
	manager.SetDifficulty("miner1", 1.0)

	// Record shares that came too slow (30 seconds instead of 10)
	for i := 0; i < 5; i++ {
		manager.RecordShare("miner1", 30*time.Second)
	}

	newDiff := manager.GetDifficulty("miner1")
	if newDiff >= 1.0 {
		t.Errorf("expected difficulty to decrease from 1.0, got %f", newDiff)
	}
}

// TestVardiffManager_AdjustDifficulty_WithinVariance tests no change when within variance
func TestVardiffManager_AdjustDifficulty_WithinVariance(t *testing.T) {
	config := DefaultConfig()
	config.TargetShareTime = 10 * time.Second
	config.VariancePercent = 30 // 7-13 seconds is acceptable
	config.InitialDifficulty = 1.0
	manager := NewManager(config)

	// Set initial difficulty
	manager.SetDifficulty("miner1", 1.0)

	// Record shares within acceptable variance (8-12 seconds)
	for i := 0; i < 10; i++ {
		manager.RecordShare("miner1", 9*time.Second)
	}

	newDiff := manager.GetDifficulty("miner1")
	// Difficulty should stay close to initial (within small tolerance)
	if newDiff < 0.9 || newDiff > 1.1 {
		t.Errorf("expected difficulty to stay near 1.0, got %f", newDiff)
	}
}

// TestVardiffManager_GetTargetShareTime tests getting target share time
func TestVardiffManager_GetTargetShareTime(t *testing.T) {
	config := DefaultConfig()
	config.TargetShareTime = 15 * time.Second
	manager := NewManager(config)

	target := manager.GetTargetShareTime()
	if target != 15*time.Second {
		t.Errorf("expected target share time 15s, got %v", target)
	}
}

// TestVardiffManager_Concurrent tests thread safety
func TestVardiffManager_Concurrent(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	done := make(chan bool)

	// Multiple goroutines accessing the manager
	for i := 0; i < 10; i++ {
		go func(id int) {
			minerID := "miner" + string(rune('0'+id))
			for j := 0; j < 100; j++ {
				manager.GetDifficulty(minerID)
				manager.SetDifficulty(minerID, float64(j)*0.01)
				manager.RecordShare(minerID, time.Duration(j)*time.Second)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestVardiffManager_RemoveMiner tests removing miner data
func TestVardiffManager_RemoveMiner(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// Set custom difficulty
	manager.SetDifficulty("miner1", 5.0)
	if manager.GetDifficulty("miner1") != 5.0 {
		t.Fatal("failed to set difficulty")
	}

	// Remove miner
	manager.RemoveMiner("miner1")

	// Should get initial difficulty again
	diff := manager.GetDifficulty("miner1")
	if diff != config.InitialDifficulty {
		t.Errorf("expected initial difficulty after removal, got %f", diff)
	}
}
