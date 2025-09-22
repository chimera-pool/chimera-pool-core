package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"chimera-pool-core/internal/testutil"
)

// ProductionReadinessTestSuite validates production readiness
type ProductionReadinessTestSuite struct {
	FinalIntegrationTestSuite
}

func TestProductionReadinessTestSuite(t *testing.T) {
	suite.Run(t, new(ProductionReadinessTestSuite))
}

// TestSystemHealthChecks validates all health check endpoints
func (s *ProductionReadinessTestSuite) TestSystemHealthChecks() {
	s.T().Log("Testing system health checks")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test main health endpoint
	resp, err := client.Get("/health")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	var healthResponse struct {
		Status    string            `json:"status"`
		Timestamp time.Time         `json:"timestamp"`
		Services  map[string]string `json:"services"`
		Version   string            `json:"version"`
	}
	
	err = testutil.DecodeJSONResponse(resp, &healthResponse)
	s.Require().NoError(err)
	
	s.Assert().Equal("healthy", healthResponse.Status)
	s.Assert().NotEmpty(healthResponse.Version)
	s.Assert().NotEmpty(healthResponse.Services)
	
	// Validate individual service health
	requiredServices := []string{
		"database",
		"redis",
		"stratum_server",
		"algorithm_engine",
		"pool_manager",
	}
	
	for _, service := range requiredServices {
		status, exists := healthResponse.Services[service]
		s.Assert().True(exists, "Service %s should be present in health check", service)
		s.Assert().Equal("healthy", status, "Service %s should be healthy", service)
	}
}

// TestDatabaseHealth validates database connectivity and performance
func (s *ProductionReadinessTestSuite) TestDatabaseHealth() {
	s.T().Log("Testing database health and readiness")
	
	// Test basic connectivity
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	
	err := s.db.Ping(ctx)
	s.Require().NoError(err, "Database should be reachable")
	
	// Test connection pool health
	stats := s.db.Stats()
	s.Assert().Greater(stats.MaxOpenConnections, 0, "Connection pool should be configured")
	s.Assert().GreaterOrEqual(stats.OpenConnections, 0, "Should have valid connection count")
	
	// Test transaction capability
	tx, err := s.db.BeginTx(ctx, nil)
	s.Require().NoError(err)
	
	err = tx.Rollback()
	s.Require().NoError(err)
	
	// Test schema integrity
	tables := []string{
		"users",
		"miners",
		"shares",
		"blocks",
		"payouts",
		"teams",
		"monitoring_metrics",
	}
	
	for _, table := range tables {
		var count int
		err := s.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		s.Assert().NoError(err, "Table %s should exist and be queryable", table)
	}
}

// TestConfigurationValidation validates all configuration settings
func (s *ProductionReadinessTestSuite) TestConfigurationValidation() {
	s.T().Log("Testing configuration validation")
	
	// Test environment variables
	requiredEnvVars := []string{
		"DATABASE_URL",
		"REDIS_URL",
		"JWT_SECRET",
		"ENCRYPTION_KEY",
	}
	
	for _, envVar := range requiredEnvVars {
		value := os.Getenv(envVar)
		s.Assert().NotEmpty(value, "Environment variable %s should be set", envVar)
		
		// Basic validation for sensitive values
		if strings.Contains(envVar, "SECRET") || strings.Contains(envVar, "KEY") {
			s.Assert().GreaterOrEqual(len(value), 32, 
				"Secret/key %s should be at least 32 characters", envVar)
		}
	}
	
	// Test configuration endpoint (admin only)
	adminToken := s.getAdminToken()
	client := testutil.NewAPIClient("http://localhost:8080", adminToken)
	
	resp, err := client.Get("/api/admin/config")
	if err == nil {
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode)
		
		var config map[string]interface{}
		err = testutil.DecodeJSONResponse(resp, &config)
		s.Require().NoError(err)
		
		// Validate critical configuration values
		s.Assert().NotEmpty(config["pool_name"], "Pool name should be configured")
		s.Assert().NotEmpty(config["stratum_port"], "Stratum port should be configured")
		s.Assert().NotEmpty(config["api_port"], "API port should be configured")
	}
}

// TestLoggingAndMonitoring validates logging and monitoring setup
func (s *ProductionReadinessTestSuite) TestLoggingAndMonitoring() {
	s.T().Log("Testing logging and monitoring")
	
	// Test metrics endpoint
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/metrics")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	// Read metrics content
	body, err := testutil.ReadResponseBody(resp)
	s.Require().NoError(err)
	
	// Validate Prometheus metrics format
	metricsContent := string(body)
	s.Assert().Contains(metricsContent, "# HELP", "Should contain Prometheus help comments")
	s.Assert().Contains(metricsContent, "# TYPE", "Should contain Prometheus type comments")
	
	// Check for essential metrics
	essentialMetrics := []string{
		"pool_active_miners",
		"pool_total_hashrate",
		"pool_shares_submitted",
		"pool_shares_accepted",
		"pool_blocks_found",
		"api_requests_total",
		"database_connections",
		"system_memory_usage",
		"system_cpu_usage",
	}
	
	for _, metric := range essentialMetrics {
		s.Assert().Contains(metricsContent, metric, 
			"Essential metric %s should be present", metric)
	}
	
	// Test structured logging
	s.testStructuredLogging()
}

func (s *ProductionReadinessTestSuite) testStructuredLogging() {
	s.T().Log("Testing structured logging")
	
	// Generate some log entries by performing operations
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "log_test_user",
		Email:    "logtest@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Login to generate auth logs
	_, err = s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Check if logs are being written (this would depend on log configuration)
	// In a real implementation, you might check log files or log aggregation systems
}

// TestSecurityCompliance validates security compliance measures
func (s *ProductionReadinessTestSuite) TestSecurityCompliance() {
	s.T().Log("Testing security compliance")
	
	// Test HTTPS enforcement (if configured)
	s.testHTTPSEnforcement()
	
	// Test security headers
	s.testSecurityHeaders()
	
	// Test audit logging
	s.testAuditLogging()
}

func (s *ProductionReadinessTestSuite) testHTTPSEnforcement() {
	s.T().Log("Testing HTTPS enforcement")
	
	// This test would be more relevant in a production environment with HTTPS
	// For now, we'll test that the server responds appropriately to security requirements
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/api/pool/stats")
	s.Require().NoError(err)
	defer resp.Body.Close()
	
	// In production, this should redirect to HTTPS or have appropriate security headers
	securityHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
	}
	
	for _, header := range securityHeaders {
		value := resp.Header.Get(header)
		s.Assert().NotEmpty(value, "Security header %s should be present", header)
	}
}

func (s *ProductionReadinessTestSuite) testSecurityHeaders() {
	s.T().Log("Testing security headers compliance")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/")
	s.Require().NoError(err)
	defer resp.Body.Close()
	
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}
	
	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		s.Assert().Equal(expectedValue, actualValue, 
			"Security header %s should have value %s", header, expectedValue)
	}
}

func (s *ProductionReadinessTestSuite) testAuditLogging() {
	s.T().Log("Testing audit logging")
	
	// Perform auditable actions
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "audit_test_user",
		Email:    "audit@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Login (should be audited)
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: user.Username,
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// Perform sensitive operation (should be audited)
	client := testutil.NewAPIClient("http://localhost:8080", token.AccessToken)
	resp, err := client.PostJSON("/api/user/profile", map[string]interface{}{
		"email": "newemail@example.com",
	})
	if err == nil {
		resp.Body.Close()
	}
	
	// In a real implementation, you would verify that audit logs were created
	// This might involve checking a database table, log files, or external audit system
}

// TestBackupAndRecovery validates backup and recovery procedures
func (s *ProductionReadinessTestSuite) TestBackupAndRecovery() {
	s.T().Log("Testing backup and recovery readiness")
	
	// Test database backup capability
	s.testDatabaseBackup()
	
	// Test configuration backup
	s.testConfigurationBackup()
}

func (s *ProductionReadinessTestSuite) testDatabaseBackup() {
	s.T().Log("Testing database backup capability")
	
	// Create some test data
	user, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "backup_test_user",
		Email:    "backup@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	
	// In a real implementation, you would:
	// 1. Trigger a backup process
	// 2. Verify backup file creation
	// 3. Test backup integrity
	// 4. Test restore process
	
	// For this test, we'll verify that backup-related endpoints exist
	adminToken := s.getAdminToken()
	client := testutil.NewAPIClient("http://localhost:8080", adminToken)
	
	// Test backup status endpoint
	resp, err := client.Get("/api/admin/backup/status")
	if err == nil {
		defer resp.Body.Close()
		// Backup functionality should be available
		s.Assert().NotEqual(http.StatusNotFound, resp.StatusCode, 
			"Backup status endpoint should exist")
	}
}

func (s *ProductionReadinessTestSuite) testConfigurationBackup() {
	s.T().Log("Testing configuration backup")
	
	// Test that configuration can be exported
	adminToken := s.getAdminToken()
	client := testutil.NewAPIClient("http://localhost:8080", adminToken)
	
	resp, err := client.Get("/api/admin/config/export")
	if err == nil {
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode, 
			"Configuration export should be available")
		
		var config map[string]interface{}
		err = testutil.DecodeJSONResponse(resp, &config)
		s.Require().NoError(err)
		s.Assert().NotEmpty(config, "Exported configuration should not be empty")
	}
}

// TestScalabilityReadiness validates scalability preparations
func (s *ProductionReadinessTestSuite) TestScalabilityReadiness() {
	s.T().Log("Testing scalability readiness")
	
	// Test horizontal scaling readiness
	s.testHorizontalScalingReadiness()
	
	// Test load balancer health checks
	s.testLoadBalancerHealthChecks()
}

func (s *ProductionReadinessTestSuite) testHorizontalScalingReadiness() {
	s.T().Log("Testing horizontal scaling readiness")
	
	// Test that the application is stateless (no local state dependencies)
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Make multiple requests and verify consistent responses
	for i := 0; i < 10; i++ {
		resp, err := client.Get("/api/pool/stats")
		s.Require().NoError(err)
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode)
		
		var stats map[string]interface{}
		err = testutil.DecodeJSONResponse(resp, &stats)
		s.Require().NoError(err)
		s.Assert().NotEmpty(stats, "Pool stats should be available")
	}
}

func (s *ProductionReadinessTestSuite) testLoadBalancerHealthChecks() {
	s.T().Log("Testing load balancer health checks")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test health check endpoint that load balancers would use
	resp, err := client.Get("/health/ready")
	s.Require().NoError(err)
	defer resp.Body.Close()
	
	// Should return 200 OK when ready to serve traffic
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	// Test liveness probe
	resp, err = client.Get("/health/live")
	s.Require().NoError(err)
	defer resp.Body.Close()
	
	// Should return 200 OK when application is alive
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
}

// TestDocumentationAndSupport validates documentation and support readiness
func (s *ProductionReadinessTestSuite) TestDocumentationAndSupport() {
	s.T().Log("Testing documentation and support readiness")
	
	// Test API documentation availability
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/docs")
	if err == nil {
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode, 
			"API documentation should be available")
	}
	
	// Test OpenAPI/Swagger specification
	resp, err = client.Get("/api/swagger.json")
	if err == nil {
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode, 
			"OpenAPI specification should be available")
		
		var swagger map[string]interface{}
		err = testutil.DecodeJSONResponse(resp, &swagger)
		s.Require().NoError(err)
		s.Assert().NotEmpty(swagger["paths"], "API paths should be documented")
	}
}

// TestProductionDeploymentReadiness validates deployment readiness
func (s *ProductionReadinessTestSuite) TestProductionDeploymentReadiness() {
	s.T().Log("Testing production deployment readiness")
	
	// Test Docker deployment readiness
	s.testDockerDeploymentReadiness()
	
	// Test cloud deployment readiness
	s.testCloudDeploymentReadiness()
	
	// Test monitoring readiness
	s.testMonitoringReadiness()
	
	// Test backup and disaster recovery readiness
	s.testDisasterRecoveryReadiness()
}

func (s *ProductionReadinessTestSuite) testDockerDeploymentReadiness() {
	s.T().Log("Testing Docker deployment readiness")
	
	// Check for Docker Compose files
	dockerFiles := []string{
		"deployments/docker/docker-compose.yml",
		"deployments/docker/docker-compose.prod.yml",
	}
	
	for _, file := range dockerFiles {
		if _, err := os.Stat(file); err == nil {
			s.T().Logf("✅ Docker file exists: %s", file)
		} else {
			s.T().Logf("⚠️  Docker file missing: %s", file)
		}
	}
	
	// Test container health checks
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/health")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Health check should work for container orchestration")
	
	// Test graceful shutdown capability
	resp, err = client.Get("/health/ready")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Readiness check should work for rolling deployments")
}

func (s *ProductionReadinessTestSuite) testCloudDeploymentReadiness() {
	s.T().Log("Testing cloud deployment readiness")
	
	// Check for cloud deployment templates
	cloudFiles := []string{
		"deployments/terraform/main.tf",
		"deployments/kubernetes/deployment.yaml",
	}
	
	for _, file := range cloudFiles {
		if _, err := os.Stat(file); err == nil {
			s.T().Logf("✅ Cloud deployment file exists: %s", file)
		} else {
			s.T().Logf("⚠️  Cloud deployment file missing: %s", file)
		}
	}
	
	// Test environment variable configuration
	requiredEnvVars := []string{
		"DATABASE_URL",
		"REDIS_URL",
		"JWT_SECRET",
		"ENCRYPTION_KEY",
	}
	
	missingVars := 0
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missingVars++
			s.T().Logf("⚠️  Environment variable not set: %s", envVar)
		}
	}
	
	s.Assert().LessOrEqual(missingVars, 2, "Most environment variables should be configured")
}

func (s *ProductionReadinessTestSuite) testMonitoringReadiness() {
	s.T().Log("Testing monitoring readiness")
	
	// Test Prometheus metrics endpoint
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	resp, err := client.Get("/metrics")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Prometheus metrics should be available")
	
	body, err := testutil.ReadResponseBody(resp)
	s.Require().NoError(err)
	metricsContent := string(body)
	
	// Check for essential business metrics
	businessMetrics := []string{
		"pool_active_miners",
		"pool_total_hashrate",
		"pool_shares_submitted_total",
		"pool_shares_accepted_total",
		"pool_blocks_found_total",
		"pool_payouts_processed_total",
	}
	
	for _, metric := range businessMetrics {
		s.Assert().Contains(metricsContent, metric, "Business metric %s should be available", metric)
	}
	
	// Check for system metrics
	systemMetrics := []string{
		"go_memstats_alloc_bytes",
		"go_goroutines",
		"process_cpu_seconds_total",
		"http_requests_total",
	}
	
	for _, metric := range systemMetrics {
		s.Assert().Contains(metricsContent, metric, "System metric %s should be available", metric)
	}
	
	// Test alerting configuration
	if _, err := os.Stat("configs/prometheus/alert_rules.yml"); err == nil {
		s.T().Log("✅ Prometheus alert rules configured")
	} else {
		s.T().Log("⚠️  Prometheus alert rules not found")
	}
	
	// Test Grafana dashboard configuration
	if _, err := os.Stat("configs/grafana/dashboards/pool-overview.json"); err == nil {
		s.T().Log("✅ Grafana dashboards configured")
	} else {
		s.T().Log("⚠️  Grafana dashboards not found")
	}
}

func (s *ProductionReadinessTestSuite) testDisasterRecoveryReadiness() {
	s.T().Log("Testing disaster recovery readiness")
	
	// Test database backup capability
	adminToken := s.getAdminToken()
	client := testutil.NewAPIClient("http://localhost:8080", adminToken)
	
	// Test backup endpoint
	resp, err := client.Get("/api/admin/backup/status")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			s.T().Log("✅ Database backup system is available")
		} else {
			s.T().Log("⚠️  Database backup system not ready")
		}
	}
	
	// Test configuration export
	resp, err = client.Get("/api/admin/config/export")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			s.T().Log("✅ Configuration export is available")
		} else {
			s.T().Log("⚠️  Configuration export not ready")
		}
	}
	
	// Test data integrity checks
	err = s.db.Ping(s.ctx)
	s.Assert().NoError(err, "Database should be accessible for disaster recovery")
	
	// Test that critical data can be queried
	var userCount int
	err = s.db.QueryRow(s.ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	s.Assert().NoError(err, "Should be able to query critical data")
}

// TestPerformanceUnderLoad validates performance under production-like load
func (s *ProductionReadinessTestSuite) TestPerformanceUnderLoad() {
	s.T().Log("Testing performance under production-like load")
	
	const (
		numMiners = 1000
		testDuration = 2 * time.Minute
		maxResponseTime = 200 * time.Millisecond
	)
	
	ctx, cancel := context.WithTimeout(s.ctx, testDuration+30*time.Second)
	defer cancel()
	
	// Create production-like mining load
	miners := make([]*testutil.MockMiner, 0, numMiners)
	var wg sync.WaitGroup
	
	// Connect miners
	for i := 0; i < numMiners; i++ {
		wg.Add(1)
		go func(minerID int) {
			defer wg.Done()
			
			miner := testutil.NewMockMiner(fmt.Sprintf("prod_miner_%d", minerID), "password123")
			err := miner.Connect("localhost:18332")
			if err == nil {
				miners = append(miners, miner)
				miner.Subscribe()
				miner.Authorize(fmt.Sprintf("prod_miner_%d", minerID), "password123")
				miner.StartMining(ctx, uint64(100+minerID%200)) // Varied hashrates
			}
		}(i)
	}
	
	wg.Wait()
	s.T().Logf("Connected %d miners for production load test", len(miners))
	
	// Monitor API performance during load
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	performanceCtx, performanceCancel := context.WithTimeout(ctx, testDuration)
	defer performanceCancel()
	
	responseTimes := make([]time.Duration, 0)
	var responseTimeMutex sync.Mutex
	
	// Monitor API performance
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-performanceCtx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				resp, err := client.Get("/api/pool/stats")
				duration := time.Since(start)
				
				if err == nil && resp.StatusCode == http.StatusOK {
					responseTimeMutex.Lock()
					responseTimes = append(responseTimes, duration)
					responseTimeMutex.Unlock()
					resp.Body.Close()
				}
			}
		}
	}()
	
	// Wait for test duration
	<-performanceCtx.Done()
	
	// Calculate performance metrics
	if len(responseTimes) > 0 {
		var totalTime time.Duration
		maxTime := time.Duration(0)
		
		for _, rt := range responseTimes {
			totalTime += rt
			if rt > maxTime {
				maxTime = rt
			}
		}
		
		avgResponseTime := totalTime / time.Duration(len(responseTimes))
		
		s.T().Logf("Production Load Test Results:")
		s.T().Logf("- Average API response time: %v", avgResponseTime)
		s.T().Logf("- Maximum API response time: %v", maxTime)
		s.T().Logf("- API requests tested: %d", len(responseTimes))
		
		// Performance assertions
		s.Assert().Less(avgResponseTime, maxResponseTime, 
			"Average response time should be under %v during production load", maxResponseTime)
		s.Assert().Less(maxTime, maxResponseTime*2, 
			"Maximum response time should be reasonable during production load")
	}
	
	// Check system health after load test
	resp, err := client.Get("/health")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode, "System should remain healthy after load test")
}

// TestFailoverAndRecovery tests system failover and recovery capabilities
func (s *ProductionReadinessTestSuite) TestFailoverAndRecovery() {
	s.T().Log("Testing failover and recovery capabilities")
	
	// Test graceful degradation
	s.testGracefulDegradation()
	
	// Test service recovery
	s.testServiceRecovery()
}

func (s *ProductionReadinessTestSuite) testGracefulDegradation() {
	s.T().Log("Testing graceful degradation")
	
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test that system continues to function even if some components are stressed
	// This is a simplified test - in production you might simulate actual component failures
	
	// Create some load
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				resp, err := client.Get("/api/pool/stats")
				if err == nil {
					resp.Body.Close()
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
	
	// System should still respond to health checks
	time.Sleep(2 * time.Second)
	
	resp, err := client.Get("/health")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Health check should work during load")
}

func (s *ProductionReadinessTestSuite) testServiceRecovery() {
	s.T().Log("Testing service recovery")
	
	// Test that services can recover from temporary issues
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Baseline health check
	resp, err := client.Get("/health")
	s.Require().NoError(err)
	resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode, "System should be healthy initially")
	
	// Simulate recovery by checking health multiple times
	healthChecks := 0
	healthyChecks := 0
	
	for i := 0; i < 10; i++ {
		resp, err := client.Get("/health")
		healthChecks++
		
		if err == nil && resp.StatusCode == http.StatusOK {
			healthyChecks++
			resp.Body.Close()
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	healthRatio := float64(healthyChecks) / float64(healthChecks)
	s.Assert().Greater(healthRatio, 0.8, "System should maintain high availability during recovery tests")
}

// Helper method to get admin token
func (s *ProductionReadinessTestSuite) getAdminToken() string {
	// Create admin user if not exists
	adminUser, err := s.authService.Register(s.ctx, &testutil.RegisterRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
	})
	if err != nil {
		// User might already exist, try to login
		token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
			Username: "admin",
			Password: "AdminPassword123!",
		})
		if err == nil {
			return token.AccessToken
		}
		return ""
	}
	
	// Set admin role
	err = s.authService.SetUserRole(s.ctx, adminUser.ID, "admin")
	if err != nil {
		return ""
	}
	
	// Login and get token
	token, err := s.authService.Login(s.ctx, &testutil.LoginRequest{
		Username: "admin",
		Password: "AdminPassword123!",
	})
	if err != nil {
		return ""
	}
	
	return token.AccessToken
}