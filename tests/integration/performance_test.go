package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"chimera-pool-core/internal/testutil"
)

// PerformanceTestSuite validates system performance under production load
type PerformanceTestSuite struct {
	FinalIntegrationTestSuite
}

func TestPerformanceTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceTestSuite))
}

// TestHighConcurrencyMining tests the system under high concurrent load
func (s *PerformanceTestSuite) TestHighConcurrencyMining() {
	s.T().Log("Testing high concurrency mining performance")
	
	const (
		numMiners = 1000
		testDuration = 2 * time.Minute
		targetResponseTime = 100 * time.Millisecond
	)
	
	ctx, cancel := context.WithTimeout(s.ctx, testDuration+30*time.Second)
	defer cancel()
	
	// Create multiple miners
	miners := make([]*testutil.MockMiner, numMiners)
	var wg sync.WaitGroup
	
	// Performance metrics
	var (
		totalConnections    int64
		successfulConnections int64
		totalShares         int64
		acceptedShares      int64
		responseTimes       []time.Duration
		responseTimeMutex   sync.Mutex
	)
	
	// Start miners concurrently
	for i := 0; i < numMiners; i++ {
		wg.Add(1)
		go func(minerID int) {
			defer wg.Done()
			
			miner := testutil.NewMockMiner(fmt.Sprintf("miner_%d", minerID), "password123")
			miners[minerID] = miner
			
			// Measure connection time
			startTime := time.Now()
			err := miner.Connect("localhost:18332")
			connectionTime := time.Since(startTime)
			
			totalConnections++
			if err == nil {
				successfulConnections++
				
				// Record response time
				responseTimeMutex.Lock()
				responseTimes = append(responseTimes, connectionTime)
				responseTimeMutex.Unlock()
				
				// Subscribe and authorize
				if err := miner.Subscribe(); err == nil {
					if err := miner.Authorize(fmt.Sprintf("miner_%d", minerID), "password123"); err == nil {
						// Start mining
						miner.StartMining(ctx, 100) // 100 H/s per miner
					}
				}
			}
		}(i)
	}
	
	// Wait for all miners to connect
	wg.Wait()
	
	s.T().Logf("Connected %d/%d miners", successfulConnections, totalConnections)
	s.Assert().Greater(float64(successfulConnections)/float64(totalConnections), 0.95) // 95% success rate
	
	// Let miners run for the test duration
	time.Sleep(testDuration)
	
	// Collect performance metrics
	for _, miner := range miners {
		if miner != nil {
			stats := miner.GetStats()
			totalShares += int64(stats.SharesSubmitted)
			acceptedShares += int64(stats.SharesAccepted)
		}
	}
	
	// Calculate average response time
	var totalResponseTime time.Duration
	for _, rt := range responseTimes {
		totalResponseTime += rt
	}
	avgResponseTime := totalResponseTime / time.Duration(len(responseTimes))
	
	s.T().Logf("Performance Results:")
	s.T().Logf("- Successful connections: %d/%d (%.2f%%)", successfulConnections, totalConnections, 
		float64(successfulConnections)/float64(totalConnections)*100)
	s.T().Logf("- Average response time: %v", avgResponseTime)
	s.T().Logf("- Total shares submitted: %d", totalShares)
	s.T().Logf("- Shares accepted: %d (%.2f%%)", acceptedShares, float64(acceptedShares)/float64(totalShares)*100)
	
	// Performance assertions
	s.Assert().Less(avgResponseTime, targetResponseTime, "Average response time should be under %v", targetResponseTime)
	s.Assert().Greater(float64(acceptedShares)/float64(totalShares), 0.98, "Share acceptance rate should be > 98%")
	s.Assert().Greater(totalShares, int64(numMiners*10), "Should process significant number of shares")
}

// TestDatabasePerformance tests database performance under load
func (s *PerformanceTestSuite) TestDatabasePerformance() {
	s.T().Log("Testing database performance under load")
	
	const (
		numOperations = 10000
		concurrency = 100
	)
	
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()
	
	// Test concurrent database operations
	var wg sync.WaitGroup
	operationChan := make(chan int, numOperations)
	
	// Fill operation channel
	for i := 0; i < numOperations; i++ {
		operationChan <- i
	}
	close(operationChan)
	
	startTime := time.Now()
	
	// Start concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for opID := range operationChan {
				// Simulate database operations
				user := &testutil.TestUser{
					Username: fmt.Sprintf("perftest_user_%d_%d", workerID, opID),
					Email:    fmt.Sprintf("perftest_%d_%d@example.com", workerID, opID),
				}
				
				// Insert user
				err := s.db.CreateUser(ctx, user)
				if err != nil {
					s.T().Errorf("Failed to create user: %v", err)
					continue
				}
				
				// Query user
				_, err = s.db.GetUserByUsername(ctx, user.Username)
				if err != nil {
					s.T().Errorf("Failed to query user: %v", err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	opsPerSecond := float64(numOperations) / duration.Seconds()
	s.T().Logf("Database Performance: %.2f operations/second", opsPerSecond)
	
	// Should handle at least 1000 ops/second
	s.Assert().Greater(opsPerSecond, 1000.0, "Database should handle at least 1000 operations/second")
}

// TestMemoryUsage tests memory usage under load
func (s *PerformanceTestSuite) TestMemoryUsage() {
	s.T().Log("Testing memory usage under load")
	
	// Get initial memory stats
	initialStats := testutil.GetMemoryStats()
	
	// Create load
	const numMiners = 500
	miners := make([]*testutil.MockMiner, numMiners)
	
	for i := 0; i < numMiners; i++ {
		miner := testutil.NewMockMiner(fmt.Sprintf("mem_test_miner_%d", i), "password123")
		miners[i] = miner
		
		err := miner.Connect("localhost:18332")
		s.Require().NoError(err)
		
		err = miner.Subscribe()
		s.Require().NoError(err)
		
		err = miner.Authorize(fmt.Sprintf("mem_test_miner_%d", i), "password123")
		s.Require().NoError(err)
		
		miner.StartMining(s.ctx, 50) // 50 H/s per miner
	}
	
	// Let system run under load
	time.Sleep(30 * time.Second)
	
	// Get memory stats under load
	loadStats := testutil.GetMemoryStats()
	
	// Disconnect all miners
	for _, miner := range miners {
		miner.Disconnect()
	}
	
	// Wait for cleanup
	time.Sleep(10 * time.Second)
	
	// Get final memory stats
	finalStats := testutil.GetMemoryStats()
	
	s.T().Logf("Memory Usage:")
	s.T().Logf("- Initial: %d MB", initialStats.AllocMB)
	s.T().Logf("- Under load: %d MB", loadStats.AllocMB)
	s.T().Logf("- After cleanup: %d MB", finalStats.AllocMB)
	
	// Memory should not grow excessively
	memoryGrowth := loadStats.AllocMB - initialStats.AllocMB
	s.Assert().Less(memoryGrowth, int64(1000), "Memory growth should be less than 1GB under load")
	
	// Memory should be cleaned up after disconnection
	memoryAfterCleanup := finalStats.AllocMB - initialStats.AllocMB
	s.Assert().Less(memoryAfterCleanup, memoryGrowth/2, "Memory should be cleaned up after miner disconnection")
}

// TestAPIPerformance tests API endpoint performance
func (s *PerformanceTestSuite) TestAPIPerformance() {
	s.T().Log("Testing API performance")
	
	const (
		numRequests = 1000
		concurrency = 50
	)
	
	// Create test user and get auth token
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "api_perf_user",
		Email:    "api_perf@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: "api_perf_user",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Test different API endpoints
	endpoints := []struct {
		name string
		path string
	}{
		{"Pool Stats", "/api/pool/stats"},
		{"User Profile", "/api/user/profile"},
		{"Mining Stats", "/api/mining/stats"},
		{"Leaderboard", "/api/community/leaderboard"},
	}
	
	for _, endpoint := range endpoints {
		s.T().Logf("Testing %s endpoint performance", endpoint.name)
		
		var wg sync.WaitGroup
		requestChan := make(chan int, numRequests)
		responseTimes := make([]time.Duration, 0, numRequests)
		var responseTimeMutex sync.Mutex
		
		// Fill request channel
		for i := 0; i < numRequests; i++ {
			requestChan <- i
		}
		close(requestChan)
		
		startTime := time.Now()
		
		// Start concurrent workers
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				client := testutil.NewAPIClient("http://localhost:8080", token.AccessToken)
				
				for range requestChan {
					reqStart := time.Now()
					resp, err := client.Get(endpoint.path)
					reqDuration := time.Since(reqStart)
					
					if err == nil && resp.StatusCode == 200 {
						responseTimeMutex.Lock()
						responseTimes = append(responseTimes, reqDuration)
						responseTimeMutex.Unlock()
					}
				}
			}()
		}
		
		wg.Wait()
		totalDuration := time.Since(startTime)
		
		// Calculate metrics
		var totalResponseTime time.Duration
		for _, rt := range responseTimes {
			totalResponseTime += rt
		}
		
		avgResponseTime := totalResponseTime / time.Duration(len(responseTimes))
		requestsPerSecond := float64(len(responseTimes)) / totalDuration.Seconds()
		
		s.T().Logf("%s Results:", endpoint.name)
		s.T().Logf("- Successful requests: %d/%d", len(responseTimes), numRequests)
		s.T().Logf("- Average response time: %v", avgResponseTime)
		s.T().Logf("- Requests per second: %.2f", requestsPerSecond)
		
		// Performance assertions
		s.Assert().Greater(float64(len(responseTimes))/float64(numRequests), 0.95, 
			"%s should have >95%% success rate", endpoint.name)
		s.Assert().Less(avgResponseTime, 200*time.Millisecond, 
			"%s average response time should be <200ms", endpoint.name)
		s.Assert().Greater(requestsPerSecond, 100.0, 
			"%s should handle >100 requests/second", endpoint.name)
	}
}

// TestExtremeConcurrencyMining tests system under extreme concurrent load
func (s *PerformanceTestSuite) TestExtremeConcurrencyMining() {
	s.T().Log("Testing extreme concurrency mining performance")
	
	const (
		numMiners = 2000
		testDuration = 3 * time.Minute
		targetResponseTime = 150 * time.Millisecond
	)
	
	ctx, cancel := context.WithTimeout(s.ctx, testDuration+60*time.Second)
	defer cancel()
	
	// Create multiple miners in batches to avoid overwhelming the system
	batchSize := 100
	miners := make([]*testutil.MockMiner, 0, numMiners)
	var wg sync.WaitGroup
	
	// Performance metrics
	var (
		totalConnections      int64
		successfulConnections int64
		totalShares          int64
		acceptedShares       int64
		responseTimes        []time.Duration
		responseTimeMutex    sync.Mutex
	)
	
	// Connect miners in batches
	for batch := 0; batch < numMiners; batch += batchSize {
		batchEnd := batch + batchSize
		if batchEnd > numMiners {
			batchEnd = numMiners
		}
		
		s.T().Logf("Connecting batch %d-%d", batch, batchEnd-1)
		
		for i := batch; i < batchEnd; i++ {
			wg.Add(1)
			go func(minerID int) {
				defer wg.Done()
				
				miner := testutil.NewMockMiner(fmt.Sprintf("extreme_miner_%d", minerID), "password123")
				
				// Measure connection time
				startTime := time.Now()
				err := miner.Connect("localhost:18332")
				connectionTime := time.Since(startTime)
				
				totalConnections++
				if err == nil {
					successfulConnections++
					miners = append(miners, miner)
					
					// Record response time
					responseTimeMutex.Lock()
					responseTimes = append(responseTimes, connectionTime)
					responseTimeMutex.Unlock()
					
					// Subscribe and authorize
					if err := miner.Subscribe(); err == nil {
						if err := miner.Authorize(fmt.Sprintf("extreme_miner_%d", minerID), "password123"); err == nil {
							// Start mining with varied hashrates
							hashrate := uint64(50 + (minerID%100)) // 50-150 H/s
							miner.StartMining(ctx, hashrate)
						}
					}
				}
			}(i)
		}
		
		// Wait for batch to complete before starting next batch
		wg.Wait()
		
		// Small delay between batches
		time.Sleep(500 * time.Millisecond)
	}
	
	s.T().Logf("Connected %d/%d miners", successfulConnections, totalConnections)
	s.Assert().Greater(float64(successfulConnections)/float64(totalConnections), 0.90, 
		"At least 90% of miners should connect successfully under extreme load")
	
	// Let miners run for the test duration
	s.T().Log("Running extreme load test...")
	time.Sleep(testDuration)
	
	// Collect performance metrics
	for _, miner := range miners {
		if miner != nil {
			stats := miner.GetStats()
			totalShares += int64(stats.SharesSubmitted)
			acceptedShares += int64(stats.SharesAccepted)
		}
	}
	
	// Calculate average response time
	var totalResponseTime time.Duration
	for _, rt := range responseTimes {
		totalResponseTime += rt
	}
	avgResponseTime := totalResponseTime / time.Duration(len(responseTimes))
	
	s.T().Logf("Extreme Load Test Results:")
	s.T().Logf("- Successful connections: %d/%d (%.2f%%)", successfulConnections, totalConnections, 
		float64(successfulConnections)/float64(totalConnections)*100)
	s.T().Logf("- Average response time: %v", avgResponseTime)
	s.T().Logf("- Total shares submitted: %d", totalShares)
	s.T().Logf("- Shares accepted: %d (%.2f%%)", acceptedShares, float64(acceptedShares)/float64(totalShares)*100)
	
	// Performance assertions for extreme load
	s.Assert().Less(avgResponseTime, targetResponseTime, "Average response time should be under %v even under extreme load", targetResponseTime)
	s.Assert().Greater(float64(acceptedShares)/float64(totalShares), 0.95, "Share acceptance rate should be > 95% even under extreme load")
	s.Assert().Greater(totalShares, int64(numMiners*5), "Should process significant number of shares under extreme load")
}

// TestSystemRecoveryAfterOverload tests system recovery after overload
func (s *PerformanceTestSuite) TestSystemRecoveryAfterOverload() {
	s.T().Log("Testing system recovery after overload")
	
	// First, create overload condition
	const overloadMiners = 1500
	miners := make([]*testutil.MockMiner, 0, overloadMiners)
	
	// Create overload
	s.T().Log("Creating overload condition...")
	for i := 0; i < overloadMiners; i++ {
		miner := testutil.NewMockMiner(fmt.Sprintf("overload_miner_%d", i), "password123")
		err := miner.Connect("localhost:18332")
		if err == nil {
			miners = append(miners, miner)
			miner.Subscribe()
			miner.Authorize(fmt.Sprintf("overload_miner_%d", i), "password123")
			miner.StartMining(s.ctx, 100)
		}
	}
	
	s.T().Logf("Created overload with %d miners", len(miners))
	
	// Let system run under overload
	time.Sleep(30 * time.Second)
	
	// Measure system performance under overload
	overloadStart := time.Now()
	client := testutil.NewAPIClient("http://localhost:8080", "")
	resp, err := client.Get("/api/pool/stats")
	overloadResponseTime := time.Since(overloadStart)
	if err == nil {
		resp.Body.Close()
	}
	
	s.T().Logf("Response time under overload: %v", overloadResponseTime)
	
	// Now disconnect all miners to simulate load removal
	s.T().Log("Removing overload...")
	for _, miner := range miners {
		miner.Disconnect()
	}
	
	// Wait for system to recover
	time.Sleep(10 * time.Second)
	
	// Measure system performance after recovery
	recoveryStart := time.Now()
	resp, err = client.Get("/api/pool/stats")
	recoveryResponseTime := time.Since(recoveryStart)
	if err == nil {
		resp.Body.Close()
	}
	
	s.T().Logf("Response time after recovery: %v", recoveryResponseTime)
	
	// System should recover to normal performance
	s.Assert().Less(recoveryResponseTime, 100*time.Millisecond, 
		"System should recover to normal response times after overload removal")
	s.Assert().Less(recoveryResponseTime, overloadResponseTime, 
		"Recovery response time should be better than overload response time")
}

// TestLongRunningStabilityTest tests system stability over extended periods
func (s *PerformanceTestSuite) TestLongRunningStabilityTest() {
	s.T().Log("Testing long-running stability")
	
	const (
		numMiners = 500
		testDuration = 10 * time.Minute // Extended test duration
	)
	
	ctx, cancel := context.WithTimeout(s.ctx, testDuration+2*time.Minute)
	defer cancel()
	
	// Create stable mining load
	miners := make([]*testutil.MockMiner, 0, numMiners)
	
	for i := 0; i < numMiners; i++ {
		miner := testutil.NewMockMiner(fmt.Sprintf("stability_miner_%d", i), "password123")
		err := miner.Connect("localhost:18332")
		if err == nil {
			miners = append(miners, miner)
			miner.Subscribe()
			miner.Authorize(fmt.Sprintf("stability_miner_%d", i), "password123")
			miner.StartMining(ctx, 75) // Consistent hashrate
		}
	}
	
	s.T().Logf("Started stability test with %d miners for %v", len(miners), testDuration)
	
	// Monitor system health during the test
	healthCheckInterval := 1 * time.Minute
	healthTicker := time.NewTicker(healthCheckInterval)
	defer healthTicker.Stop()
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	healthCheckCount := 0
	healthCheckFailures := 0
	
	stabilityCtx, stabilityCancel := context.WithTimeout(ctx, testDuration)
	defer stabilityCancel()
	
	go func() {
		for {
			select {
			case <-stabilityCtx.Done():
				return
			case <-healthTicker.C:
				healthCheckCount++
				
				// Check API health
				resp, err := client.Get("/health")
				if err != nil || resp.StatusCode != http.StatusOK {
					healthCheckFailures++
					s.T().Logf("Health check %d failed", healthCheckCount)
				} else {
					resp.Body.Close()
				}
				
				// Check memory usage
				memStats := testutil.GetMemoryStats()
				s.T().Logf("Health check %d: Memory usage %d MB, GC count %d", 
					healthCheckCount, memStats.AllocMB, memStats.NumGC)
				
				// Memory should not grow excessively
				if memStats.AllocMB > 2000 { // 2GB limit
					s.T().Logf("WARNING: High memory usage detected: %d MB", memStats.AllocMB)
				}
			}
		}
	}()
	
	// Wait for test duration
	<-stabilityCtx.Done()
	
	// Collect final statistics
	var totalShares, acceptedShares int64
	for _, miner := range miners {
		if miner != nil {
			stats := miner.GetStats()
			totalShares += int64(stats.SharesSubmitted)
			acceptedShares += int64(stats.SharesAccepted)
		}
	}
	
	s.T().Logf("Stability Test Results:")
	s.T().Logf("- Test duration: %v", testDuration)
	s.T().Logf("- Health checks: %d/%d passed", healthCheckCount-healthCheckFailures, healthCheckCount)
	s.T().Logf("- Total shares: %d", totalShares)
	s.T().Logf("- Accepted shares: %d (%.2f%%)", acceptedShares, float64(acceptedShares)/float64(totalShares)*100)
	
	// Stability assertions
	healthCheckSuccessRate := float64(healthCheckCount-healthCheckFailures) / float64(healthCheckCount)
	s.Assert().Greater(healthCheckSuccessRate, 0.95, "Health check success rate should be > 95%")
	s.Assert().Greater(float64(acceptedShares)/float64(totalShares), 0.98, "Share acceptance rate should remain high during stability test")
	s.Assert().Greater(totalShares, int64(numMiners*30), "Should process significant shares during stability test")
}