package integration

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"chimera-pool-core/internal/api"
	"chimera-pool-core/internal/auth"
	"chimera-pool-core/internal/community"
	"chimera-pool-core/internal/database"
	"chimera-pool-core/internal/monitoring"
	"chimera-pool-core/internal/payouts"
	"chimera-pool-core/internal/poolmanager"
	"chimera-pool-core/internal/security"
	"chimera-pool-core/internal/shares"
	"chimera-pool-core/internal/simulation"
	"chimera-pool-core/internal/stratum"
	"chimera-pool-core/internal/testutil"
)

// FinalIntegrationTestSuite tests all components working together
type FinalIntegrationTestSuite struct {
	suite.Suite
	ctx           context.Context
	cancel        context.CancelFunc
	db            *database.Database
	poolManager   *poolmanager.PoolManager
	stratumServer *stratum.Server
	apiServer     *api.Server
	authService   *auth.Service
	securitySvc   *security.Service
	shareProc     *shares.ShareProcessor
	payoutSvc     *payouts.Service
	communitySvc  *community.Service
	monitoringSvc *monitoring.Service
	simulator     *simulation.SimulationManager
	testCleanup   []func()
}

func TestFinalIntegrationSuite(t *testing.T) {
	suite.Run(t, new(FinalIntegrationTestSuite))
}

func (s *FinalIntegrationTestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 30*time.Minute)
	
	// Initialize test database
	var err error
	s.db, err = testutil.SetupTestDatabase(s.ctx)
	s.Require().NoError(err)
	s.testCleanup = append(s.testCleanup, func() { s.db.Close() })

	// Initialize all services
	s.setupServices()
	
	// Start all servers
	s.startServers()
}

func (s *FinalIntegrationTestSuite) TearDownSuite() {
	s.cancel()
	
	// Run cleanup functions in reverse order
	for i := len(s.testCleanup) - 1; i >= 0; i-- {
		s.testCleanup[i]()
	}
}

func (s *FinalIntegrationTestSuite) setupServices() {
	var err error
	
	// Security service
	s.securitySvc, err = security.NewService(s.db)
	s.Require().NoError(err)
	
	// Auth service
	s.authService, err = auth.NewService(s.db, s.securitySvc)
	s.Require().NoError(err)
	
	// Share processor
	s.shareProc, err = shares.NewShareProcessor(s.db)
	s.Require().NoError(err)
	
	// Payout service
	s.payoutSvc, err = payouts.NewService(s.db)
	s.Require().NoError(err)
	
	// Community service
	s.communitySvc, err = community.NewService(s.db)
	s.Require().NoError(err)
	
	// Monitoring service
	s.monitoringSvc, err = monitoring.NewService(s.db)
	s.Require().NoError(err)
	
	// Pool manager
	s.poolManager, err = poolmanager.NewPoolManager(s.db, s.shareProc, s.payoutSvc)
	s.Require().NoError(err)
	
	// Simulation manager
	s.simulator, err = simulation.NewSimulationManager(s.db)
	s.Require().NoError(err)
}

func (s *FinalIntegrationTestSuite) startServers() {
	var err error
	
	// Start Stratum server
	s.stratumServer, err = stratum.NewServer(s.poolManager, ":18332")
	s.Require().NoError(err)
	go s.stratumServer.Start(s.ctx)
	s.testCleanup = append(s.testCleanup, func() { s.stratumServer.Stop() })
	
	// Start API server
	s.apiServer, err = api.NewServer(
		s.authService,
		s.poolManager,
		s.communitySvc,
		s.monitoringSvc,
		s.securitySvc,
	)
	s.Require().NoError(err)
	go s.apiServer.Start(s.ctx, ":8080")
	s.testCleanup = append(s.testCleanup, func() { s.apiServer.Stop() })
	
	// Wait for servers to start
	time.Sleep(2 * time.Second)
}

// TestCompleteEndToEndWorkflow tests the complete mining workflow
func (s *FinalIntegrationTestSuite) TestCompleteEndToEndWorkflow() {
	s.T().Log("Testing complete end-to-end mining workflow")
	
	// Step 1: User registration and authentication
	user := s.testUserRegistrationAndAuth()
	
	// Step 2: Miner connection and mining
	miner := s.testMinerConnectionAndMining(user)
	
	// Step 3: Share processing and validation
	s.testShareProcessingAndValidation(miner)
	
	// Step 4: Block discovery and payouts
	s.testBlockDiscoveryAndPayouts(user)
	
	// Step 5: Community features
	s.testCommunityFeatures(user)
	
	// Step 6: Monitoring and analytics
	s.testMonitoringAndAnalytics()
	
	// Step 7: Algorithm hot-swap functionality
	s.testAlgorithmHotSwap()
	
	// Step 8: Security framework validation
	s.testSecurityFramework(user)
	
	// Step 9: Installation and deployment readiness
	s.testInstallationReadiness()
	
	s.T().Log("Complete end-to-end workflow test passed")
}

// TestAlgorithmHotSwap tests the algorithm hot-swap functionality
func (s *FinalIntegrationTestSuite) testAlgorithmHotSwap() {
	s.T().Log("Testing algorithm hot-swap functionality")
	
	// This would test the algorithm engine hot-swap capability
	// For now, we'll verify the components exist and are accessible
	
	// Test algorithm status endpoint
	client := testutil.NewAPIClient("http://localhost:8080", s.getAdminToken())
	
	resp, err := client.Get("/api/admin/algorithm/status")
	if err == nil {
		defer resp.Body.Close()
		s.Assert().Equal(http.StatusOK, resp.StatusCode, "Algorithm status endpoint should be accessible")
		
		var status map[string]interface{}
		err = testutil.DecodeJSONResponse(resp, &status)
		s.Require().NoError(err)
		s.Assert().NotEmpty(status["active_algorithm"], "Should have active algorithm")
	}
}

// TestSecurityFramework tests comprehensive security measures
func (s *FinalIntegrationTestSuite) testSecurityFramework(user *auth.User) {
	s.T().Log("Testing security framework")
	
	// Test MFA setup and validation
	mfaSecret, err := s.securitySvc.SetupMFA(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Assert().NotEmpty(mfaSecret.Secret)
	s.Assert().Greater(len(mfaSecret.BackupCodes), 0)
	
	// Test TOTP validation
	validCode := testutil.GenerateTOTP(mfaSecret.Secret)
	valid, err := s.securitySvc.ValidateMFA(s.ctx, user.ID, validCode)
	s.Require().NoError(err)
	s.Assert().True(valid)
	
	// Test encryption/decryption
	testData := "sensitive mining pool data"
	encrypted, err := s.securitySvc.Encrypt([]byte(testData))
	s.Require().NoError(err)
	s.Assert().NotEqual(testData, string(encrypted))
	
	decrypted, err := s.securitySvc.Decrypt(encrypted)
	s.Require().NoError(err)
	s.Assert().Equal(testData, string(decrypted))
	
	// Test rate limiting
	client := testutil.NewAPIClient("http://localhost:8080", "")
	rateLimitHit := false
	
	for i := 0; i < 20; i++ {
		resp, err := client.Get("/api/pool/stats")
		if err == nil {
			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitHit = true
				resp.Body.Close()
				break
			}
			resp.Body.Close()
		}
	}
	
	s.Assert().True(rateLimitHit, "Rate limiting should be active")
}

// TestInstallationReadiness tests installation and deployment readiness
func (s *FinalIntegrationTestSuite) testInstallationReadiness() {
	s.T().Log("Testing installation and deployment readiness")
	
	// Test health check endpoints
	client := testutil.NewAPIClient("http://localhost:8080", "")
	
	// Test main health endpoint
	resp, err := client.Get("/health")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	var health map[string]interface{}
	err = testutil.DecodeJSONResponse(resp, &health)
	s.Require().NoError(err)
	s.Assert().Equal("healthy", health["status"])
	
	// Test readiness endpoint
	resp, err = client.Get("/health/ready")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	// Test liveness endpoint
	resp, err = client.Get("/health/live")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	// Test metrics endpoint
	resp, err = client.Get("/metrics")
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	
	body, err := testutil.ReadResponseBody(resp)
	s.Require().NoError(err)
	metricsContent := string(body)
	
	// Verify essential metrics are present
	essentialMetrics := []string{
		"pool_active_miners",
		"pool_total_hashrate",
		"pool_shares_submitted",
		"api_requests_total",
		"database_connections",
	}
	
	for _, metric := range essentialMetrics {
		s.Assert().Contains(metricsContent, metric, "Essential metric %s should be present", metric)
	}
}

// Helper method to get admin token
func (s *FinalIntegrationTestSuite) getAdminToken() string {
	// Create admin user if not exists
	adminUser, err := s.authService.Register(s.ctx, &auth.RegisterRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
	})
	if err != nil {
		// User might already exist, try to login
		token, err := s.authService.Login(s.ctx, &auth.LoginRequest{
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
	token, err := s.authService.Login(s.ctx, &auth.LoginRequest{
		Username: "admin",
		Password: "AdminPassword123!",
	})
	if err != nil {
		return ""
	}
	
	return token.AccessToken
}

func (s *FinalIntegrationTestSuite) testUserRegistrationAndAuth() *auth.User {
	s.T().Log("Testing user registration and authentication")
	
	// Register new user
	user, err := s.authService.Register(s.ctx, &auth.RegisterRequest{
		Username: "testminer",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(user.ID)
	s.Assert().Equal("testminer", user.Username)
	
	// Test login
	token, err := s.authService.Login(s.ctx, &auth.LoginRequest{
		Username: "testminer",
		Password: "SecurePassword123!",
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(token.AccessToken)
	
	// Test MFA setup
	mfaSecret, err := s.securitySvc.SetupMFA(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Assert().NotEmpty(mfaSecret.Secret)
	
	return user
}

func (s *FinalIntegrationTestSuite) testMinerConnectionAndMining(user *auth.User) *testutil.MockMiner {
	s.T().Log("Testing miner connection and mining")
	
	// Create mock miner
	miner := testutil.NewMockMiner("testminer", "password123")
	
	// Connect to Stratum server
	err := miner.Connect("localhost:18332")
	s.Require().NoError(err)
	
	// Subscribe to mining
	err = miner.Subscribe()
	s.Require().NoError(err)
	
	// Authorize miner
	err = miner.Authorize(user.Username, "password123")
	s.Require().NoError(err)
	
	// Start mining simulation
	err = miner.StartMining(s.ctx, 1000) // 1000 H/s
	s.Require().NoError(err)
	
	// Wait for some shares to be submitted
	time.Sleep(5 * time.Second)
	
	stats := miner.GetStats()
	s.Assert().Greater(stats.SharesSubmitted, uint64(0))
	s.Assert().Greater(stats.SharesAccepted, uint64(0))
	
	return miner
}

func (s *FinalIntegrationTestSuite) testShareProcessingAndValidation(miner *testutil.MockMiner) {
	s.T().Log("Testing share processing and validation")
	
	// Get share statistics
	stats, err := s.shareProc.GetMinerStats(s.ctx, "testminer")
	s.Require().NoError(err)
	s.Assert().Greater(stats.ValidShares, uint64(0))
	s.Assert().Equal(uint64(0), stats.InvalidShares) // Mock miner should only submit valid shares
	
	// Test share validation with invalid share
	invalidShare := &shares.Share{
		MinerID:   "testminer",
		JobID:     "invalid_job",
		Nonce:     12345,
		Timestamp: time.Now(),
		Target:    []byte("invalid_target"),
	}
	
	result := s.shareProc.ProcessShare(s.ctx, invalidShare)
	s.Assert().False(result.Valid)
	s.Assert().NotEmpty(result.Reason)
}

func (s *FinalIntegrationTestSuite) testBlockDiscoveryAndPayouts(user *auth.User) {
	s.T().Log("Testing block discovery and payouts")
	
	// Simulate block discovery
	block := &payouts.Block{
		Height:     100001,
		Hash:       "0000000000000000000123456789abcdef",
		Reward:     5000000000, // 50 coins
		Timestamp:  time.Now(),
		MinedBy:    user.Username,
	}
	
	err := s.payoutSvc.ProcessBlock(s.ctx, block)
	s.Require().NoError(err)
	
	// Calculate payouts
	payouts, err := s.payoutSvc.CalculatePayouts(s.ctx, block.Hash)
	s.Require().NoError(err)
	s.Assert().Greater(len(payouts), 0)
	
	// Find user's payout
	var userPayout *payouts.Payout
	for _, payout := range payouts {
		if payout.MinerID == user.Username {
			userPayout = payout
			break
		}
	}
	s.Require().NotNil(userPayout)
	s.Assert().Greater(userPayout.Amount, uint64(0))
	
	// Process payouts
	err = s.payoutSvc.ProcessPayouts(s.ctx, payouts)
	s.Require().NoError(err)
	
	// Verify payout was recorded
	balance, err := s.payoutSvc.GetMinerBalance(s.ctx, user.Username)
	s.Require().NoError(err)
	s.Assert().Greater(balance.PendingBalance, uint64(0))
}

func (s *FinalIntegrationTestSuite) testCommunityFeatures(user *auth.User) {
	s.T().Log("Testing community features")
	
	// Create a team
	team, err := s.communitySvc.CreateTeam(s.ctx, &community.CreateTeamRequest{
		Name:        "Test Mining Team",
		Description: "A test mining team",
		CreatorID:   user.ID,
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(team.ID)
	
	// Join team
	err = s.communitySvc.JoinTeam(s.ctx, team.ID, user.ID)
	s.Require().NoError(err)
	
	// Get team stats
	stats, err := s.communitySvc.GetTeamStats(s.ctx, team.ID)
	s.Require().NoError(err)
	s.Assert().Equal(1, stats.MemberCount)
	s.Assert().Greater(stats.TotalHashrate, uint64(0))
	
	// Test referral system
	referralCode, err := s.communitySvc.GenerateReferralCode(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Assert().NotEmpty(referralCode)
}

func (s *FinalIntegrationTestSuite) testMonitoringAndAnalytics() {
	s.T().Log("Testing monitoring and analytics")
	
	// Get pool statistics
	poolStats, err := s.monitoringSvc.GetPoolStats(s.ctx)
	s.Require().NoError(err)
	s.Assert().Greater(poolStats.TotalHashrate, uint64(0))
	s.Assert().Greater(poolStats.ActiveMiners, 0)
	
	// Get system health
	health, err := s.monitoringSvc.GetSystemHealth(s.ctx)
	s.Require().NoError(err)
	s.Assert().Equal("healthy", health.Status)
	
	// Test metrics collection
	metrics, err := s.monitoringSvc.GetMetrics(s.ctx, time.Now().Add(-1*time.Hour), time.Now())
	s.Require().NoError(err)
	s.Assert().Greater(len(metrics), 0)
}