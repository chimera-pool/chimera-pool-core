package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PerformanceTestSuite provides performance testing for the API
type PerformanceTestSuite struct {
	router   *gin.Engine
	handlers *APIHandlers
	mockAuth *MockAuthService
	mockPool *MockPoolStatsService
	mockUser *MockUserService
}

// setupPerformanceTest initializes the performance test environment
func setupPerformanceTest() *PerformanceTestSuite {
	gin.SetMode(gin.TestMode)
	
	// Create mock services
	mockAuth := &MockAuthService{}
	mockPool := &MockPoolStatsService{}
	mockUser := &MockUserService{}
	
	// Create handlers
	handlers := NewAPIHandlers(mockAuth, mockPool, mockUser)
	
	// Setup router
	router := gin.New()
	SetupAPIRoutes(router, handlers)
	
	return &PerformanceTestSuite{
		router:   router,
		handlers: handlers,
		mockAuth: mockAuth,
		mockPool: mockPool,
		mockUser: mockUser,
	}
}

// TestAPIPerformance_ConcurrentRequests tests API performance under concurrent load
func TestAPIPerformance_ConcurrentRequests(t *testing.T) {
	suite := setupPerformanceTest()
	
	// Setup mock expectations for pool stats
	poolStats := &PoolStats{
		TotalHashrate:     1000000.0,
		ConnectedMiners:   150,
		TotalShares:       50000,
		ValidShares:       49500,
		BlocksFound:       25,
		LastBlockTime:     time.Now().Add(-10 * time.Minute),
		NetworkHashrate:   50000000.0,
		NetworkDifficulty: 1000000.0,
		PoolFee:           1.0,
	}
	
	// Setup expectations for concurrent requests
	for i := 0; i < 1000; i++ {
		suite.mockPool.On("GetPoolStats").Return(poolStats, nil)
	}
	
	// Test concurrent requests
	concurrency := 100
	requestsPerWorker := 10
	totalRequests := concurrency * requestsPerWorker
	
	var wg sync.WaitGroup
	results := make(chan time.Duration, totalRequests)
	errors := make(chan error, totalRequests)
	
	startTime := time.Now()
	
	// Launch concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < requestsPerWorker; j++ {
				requestStart := time.Now()
				
				req := httptest.NewRequest("GET", "/api/v1/pool/stats", nil)
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				
				requestDuration := time.Since(requestStart)
				results <- requestDuration
				
				if w.Code != http.StatusOK {
					errors <- assert.AnError
				}
			}
		}()
	}
	
	wg.Wait()
	close(results)
	close(errors)
	
	totalDuration := time.Since(startTime)
	
	// Collect results
	var responseTimes []time.Duration
	for duration := range results {
		responseTimes = append(responseTimes, duration)
	}
	
	// Check for errors
	errorCount := len(errors)
	assert.Equal(t, 0, errorCount, "Expected no errors during concurrent requests")
	
	// Calculate performance metrics
	var totalResponseTime time.Duration
	maxResponseTime := time.Duration(0)
	minResponseTime := time.Hour // Start with a large value
	
	for _, duration := range responseTimes {
		totalResponseTime += duration
		if duration > maxResponseTime {
			maxResponseTime = duration
		}
		if duration < minResponseTime {
			minResponseTime = duration
		}
	}
	
	averageResponseTime := totalResponseTime / time.Duration(len(responseTimes))
	requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
	
	// Performance assertions (Requirement 3.2: sub-100ms response times)
	assert.Less(t, averageResponseTime, 100*time.Millisecond, 
		"Average response time should be less than 100ms")
	assert.Less(t, maxResponseTime, 200*time.Millisecond, 
		"Maximum response time should be less than 200ms")
	assert.Greater(t, requestsPerSecond, 100.0, 
		"Should handle at least 100 requests per second")
	
	t.Logf("Performance Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Average Response Time: %v", averageResponseTime)
	t.Logf("  Min Response Time: %v", minResponseTime)
	t.Logf("  Max Response Time: %v", maxResponseTime)
	t.Logf("  Requests per Second: %.2f", requestsPerSecond)
}

// TestAPIPerformance_AuthenticationOverhead tests authentication performance
func TestAPIPerformance_AuthenticationOverhead(t *testing.T) {
	suite := setupPerformanceTest()
	
	userID := int64(123)
	token := "valid-jwt-token"
	
	// Setup mock expectations
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	userProfile := &UserProfile{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		JoinedAt: time.Now().Add(-30 * 24 * time.Hour),
		IsActive: true,
	}
	
	// Setup expectations for concurrent authenticated requests
	for i := 0; i < 500; i++ {
		suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
		suite.mockUser.On("GetUserProfile", userID).Return(userProfile, nil)
	}
	
	// Test authenticated endpoint performance
	concurrency := 50
	requestsPerWorker := 10
	totalRequests := concurrency * requestsPerWorker
	
	var wg sync.WaitGroup
	results := make(chan time.Duration, totalRequests)
	
	startTime := time.Now()
	
	// Launch concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for j := 0; j < requestsPerWorker; j++ {
				requestStart := time.Now()
				
				req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				w := httptest.NewRecorder()
				suite.router.ServeHTTP(w, req)
				
				requestDuration := time.Since(requestStart)
				results <- requestDuration
				
				assert.Equal(t, http.StatusOK, w.Code)
			}
		}()
	}
	
	wg.Wait()
	close(results)
	
	totalDuration := time.Since(startTime)
	
	// Collect results
	var responseTimes []time.Duration
	for duration := range results {
		responseTimes = append(responseTimes, duration)
	}
	
	// Calculate performance metrics
	var totalResponseTime time.Duration
	for _, duration := range responseTimes {
		totalResponseTime += duration
	}
	
	averageResponseTime := totalResponseTime / time.Duration(len(responseTimes))
	requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
	
	// Performance assertions for authenticated endpoints
	assert.Less(t, averageResponseTime, 150*time.Millisecond, 
		"Average response time for authenticated endpoints should be less than 150ms")
	assert.Greater(t, requestsPerSecond, 50.0, 
		"Should handle at least 50 authenticated requests per second")
	
	t.Logf("Authentication Performance Results:")
	t.Logf("  Total Authenticated Requests: %d", totalRequests)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Average Response Time: %v", averageResponseTime)
	t.Logf("  Requests per Second: %.2f", requestsPerSecond)
}

// TestAPIPerformance_RealTimeStatsEndpoint tests real-time stats endpoint performance
func TestAPIPerformance_RealTimeStatsEndpoint(t *testing.T) {
	suite := setupPerformanceTest()
	
	// Setup mock expectations for real-time stats
	realTimeStats := &RealTimeStats{
		CurrentHashrate:   1500000.0,
		AverageHashrate:   1200000.0,
		ActiveMiners:      175,
		SharesPerSecond:   25.5,
		LastBlockFound:    time.Now().Add(-5 * time.Minute),
		NetworkDifficulty: 1500000.0,
		PoolEfficiency:    99.2,
	}
	
	// Setup expectations for high-frequency requests (simulating real-time updates)
	for i := 0; i < 1000; i++ {
		suite.mockPool.On("GetRealTimeStats").Return(realTimeStats, nil)
	}
	
	// Test high-frequency requests to real-time endpoint
	requestCount := 1000
	requestInterval := 10 * time.Millisecond // Simulate 100 requests per second
	
	var responseTimes []time.Duration
	var errors []error
	
	startTime := time.Now()
	
	for i := 0; i < requestCount; i++ {
		requestStart := time.Now()
		
		req := httptest.NewRequest("GET", "/api/v1/pool/realtime", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		requestDuration := time.Since(requestStart)
		responseTimes = append(responseTimes, requestDuration)
		
		if w.Code != http.StatusOK {
			errors = append(errors, assert.AnError)
		}
		
		// Verify response structure
		var response RealTimeStatsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			errors = append(errors, err)
		}
		
		// Small delay to simulate real-time polling
		if i < requestCount-1 {
			time.Sleep(requestInterval)
		}
	}
	
	totalDuration := time.Since(startTime)
	
	// Check for errors
	assert.Empty(t, errors, "Expected no errors during real-time stats requests")
	
	// Calculate performance metrics
	var totalResponseTime time.Duration
	maxResponseTime := time.Duration(0)
	
	for _, duration := range responseTimes {
		totalResponseTime += duration
		if duration > maxResponseTime {
			maxResponseTime = duration
		}
	}
	
	averageResponseTime := totalResponseTime / time.Duration(len(responseTimes))
	actualRequestsPerSecond := float64(requestCount) / totalDuration.Seconds()
	
	// Performance assertions for real-time endpoint
	assert.Less(t, averageResponseTime, 50*time.Millisecond, 
		"Real-time stats endpoint should respond in less than 50ms on average")
	assert.Less(t, maxResponseTime, 100*time.Millisecond, 
		"Real-time stats endpoint should never exceed 100ms response time")
	assert.Greater(t, actualRequestsPerSecond, 80.0, 
		"Should handle at least 80 real-time requests per second")
	
	t.Logf("Real-Time Stats Performance Results:")
	t.Logf("  Total Requests: %d", requestCount)
	t.Logf("  Total Duration: %v", totalDuration)
	t.Logf("  Average Response Time: %v", averageResponseTime)
	t.Logf("  Max Response Time: %v", maxResponseTime)
	t.Logf("  Actual Requests per Second: %.2f", actualRequestsPerSecond)
}

// BenchmarkPoolStatsEndpoint benchmarks the pool stats endpoint
func BenchmarkPoolStatsEndpoint(b *testing.B) {
	suite := setupPerformanceTest()
	
	poolStats := &PoolStats{
		TotalHashrate:     1000000.0,
		ConnectedMiners:   150,
		TotalShares:       50000,
		ValidShares:       49500,
		BlocksFound:       25,
		LastBlockTime:     time.Now().Add(-10 * time.Minute),
		NetworkHashrate:   50000000.0,
		NetworkDifficulty: 1000000.0,
		PoolFee:           1.0,
	}
	
	// Setup expectations for benchmark
	for i := 0; i < b.N; i++ {
		suite.mockPool.On("GetPoolStats").Return(poolStats, nil)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/pool/stats", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkAuthenticatedEndpoint benchmarks authenticated endpoints
func BenchmarkAuthenticatedEndpoint(b *testing.B) {
	suite := setupPerformanceTest()
	
	userID := int64(123)
	token := "valid-jwt-token"
	
	claims := &JWTClaims{
		UserID:    userID,
		Username:  "testuser",
		Email:     "test@example.com",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	userProfile := &UserProfile{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		JoinedAt: time.Now().Add(-30 * 24 * time.Hour),
		IsActive: true,
	}
	
	// Setup expectations for benchmark
	for i := 0; i < b.N; i++ {
		suite.mockAuth.On("ValidateJWT", token).Return(claims, nil)
		suite.mockUser.On("GetUserProfile", userID).Return(userProfile, nil)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}