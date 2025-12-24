package keepalive

import (
	"sync"
	"testing"
	"time"
)

// TestConfig_Validate tests configuration validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectValid bool
	}{
		{
			name:        "valid config",
			config:      DefaultConfig(),
			expectValid: true,
		},
		{
			name: "invalid - zero interval",
			config: Config{
				Interval:  0,
				Timeout:   5 * time.Second,
				MaxMissed: 3,
			},
			expectValid: false,
		},
		{
			name: "invalid - negative max missed",
			config: Config{
				Interval:  10 * time.Second,
				Timeout:   5 * time.Second,
				MaxMissed: -1,
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

// TestManager_Start tests starting keepalive for a miner
func TestManager_Start(t *testing.T) {
	config := DefaultConfig()
	config.Interval = 100 * time.Millisecond // Fast for testing

	manager := NewManager(config, nil)

	manager.Start("miner1")

	if !manager.IsAlive("miner1") {
		t.Error("miner should be alive after Start")
	}

	manager.Stop("miner1")
}

// TestManager_Stop tests stopping keepalive for a miner
func TestManager_Stop(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config, nil)

	manager.Start("miner1")
	manager.Stop("miner1")

	// After stop, miner should not be tracked
	if manager.IsAlive("miner1") {
		t.Error("miner should not be alive after Stop")
	}
}

// TestManager_RecordActivity tests activity recording
func TestManager_RecordActivity(t *testing.T) {
	config := DefaultConfig()
	config.Interval = 50 * time.Millisecond
	config.MaxMissed = 2

	manager := NewManager(config, nil)
	manager.Start("miner1")

	// Record activity should keep miner alive
	for i := 0; i < 5; i++ {
		time.Sleep(30 * time.Millisecond)
		manager.RecordActivity("miner1")
	}

	if !manager.IsAlive("miner1") {
		t.Error("miner should still be alive with regular activity")
	}

	manager.Stop("miner1")
}

// TestManager_Timeout tests miner timeout after missed keepalives
func TestManager_Timeout(t *testing.T) {
	config := DefaultConfig()
	config.Interval = 20 * time.Millisecond
	config.MaxMissed = 2
	config.Timeout = 10 * time.Millisecond

	var disconnectedMiner string
	var mu sync.Mutex

	onTimeout := func(minerID string) {
		mu.Lock()
		disconnectedMiner = minerID
		mu.Unlock()
	}

	manager := NewManager(config, onTimeout)
	manager.Start("miner1")

	// Wait for timeout (interval * maxMissed + buffer)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if disconnectedMiner != "miner1" {
		t.Errorf("expected timeout callback for miner1, got: %s", disconnectedMiner)
	}
	mu.Unlock()

	manager.Stop("miner1")
}

// TestManager_Concurrent tests thread safety
func TestManager_Concurrent(t *testing.T) {
	config := DefaultConfig()
	config.Interval = 10 * time.Millisecond

	manager := NewManager(config, nil)

	var wg sync.WaitGroup

	// Multiple goroutines starting/stopping/recording
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			minerID := string(rune('A' + id))

			manager.Start(minerID)
			for j := 0; j < 10; j++ {
				manager.RecordActivity(minerID)
				time.Sleep(5 * time.Millisecond)
			}
			manager.Stop(minerID)
		}(i)
	}

	wg.Wait()
}

// TestManager_GetConfig tests getting configuration
func TestManager_GetConfig(t *testing.T) {
	config := DefaultConfig()
	config.Interval = 42 * time.Second

	manager := NewManager(config, nil)

	got := manager.GetConfig()
	if got.Interval != 42*time.Second {
		t.Errorf("expected interval 42s, got %v", got.Interval)
	}
}

// TestManager_MultipleMiners tests managing multiple miners
func TestManager_MultipleMiners(t *testing.T) {
	config := DefaultConfig()
	config.Interval = 50 * time.Millisecond

	manager := NewManager(config, nil)

	// Start multiple miners
	manager.Start("miner1")
	manager.Start("miner2")
	manager.Start("miner3")

	if !manager.IsAlive("miner1") || !manager.IsAlive("miner2") || !manager.IsAlive("miner3") {
		t.Error("all miners should be alive")
	}

	// Stop one
	manager.Stop("miner2")

	if !manager.IsAlive("miner1") || manager.IsAlive("miner2") || !manager.IsAlive("miner3") {
		t.Error("miner2 should be stopped, others alive")
	}

	manager.Stop("miner1")
	manager.Stop("miner3")
}
