package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// SERVICE FACTORY
// Central factory for creating all API services with proper dependency injection
// Follows ISP principles - each service is independently configurable
// =============================================================================

// ServiceConfig holds configuration for service creation
type ServiceConfig struct {
	JWTSecret        string
	JWTExpiration    int // hours
	ResetTokenExpiry int // hours
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig(jwtSecret string) *ServiceConfig {
	return &ServiceConfig{
		JWTSecret:        jwtSecret,
		JWTExpiration:    24,
		ResetTokenExpiry: 1,
	}
}

// Services holds all service implementations
type Services struct {
	Auth *AuthServices
	User *UserServices
	Pool *PoolServices
}

// NewServices creates all services with the given database and configuration
func NewServices(db *sql.DB, config *ServiceConfig) *Services {
	return &Services{
		Auth: NewAuthServices(db, config.JWTSecret),
		User: NewUserServices(db),
		Pool: NewPoolServices(db),
	}
}

// =============================================================================
// HANDLER FACTORY
// Creates all handlers using the service implementations
// =============================================================================

// Handlers holds all HTTP handler implementations
type Handlers struct {
	Auth *AuthHandlers
	User *UserHandlers
}

// NewHandlers creates all handlers from services
func NewHandlers(services *Services) *Handlers {
	return &Handlers{
		Auth: services.Auth.CreateAuthHandlers(),
		User: services.User.CreateUserHandlers(),
	}
}

// =============================================================================
// EXTENDED SERVER WITH SERVICES
// Server configuration that includes service-based handlers
// =============================================================================

// ServerWithServices extends Server with ISP-compliant service handlers
type ServerWithServices struct {
	*Server
	Services *Services
	Handlers *Handlers
}

// NewServerWithServices creates a server with all services wired up
func NewServerWithServices(config *ServerConfig, db *sql.DB) (*ServerWithServices, error) {
	// Create base server
	server := NewServer(config, db, nil)

	// Create services
	serviceConfig := DefaultServiceConfig(config.JWTSecret)
	services := NewServices(db, serviceConfig)

	// Create handlers
	handlers := NewHandlers(services)

	sws := &ServerWithServices{
		Server:   server,
		Services: services,
		Handlers: handlers,
	}

	// Register routes using service-based handlers
	sws.registerServiceRoutes()

	return sws, nil
}

// registerServiceRoutes registers all API routes using service-based handlers
func (s *ServerWithServices) registerServiceRoutes() {
	api := s.Router.Group("/api/v1")

	// Public auth routes
	api.POST("/auth/register", s.Handlers.Auth.Register)
	api.POST("/auth/login", s.Handlers.Auth.Login)
	api.POST("/auth/forgot-password", s.Handlers.Auth.ForgotPassword)
	api.POST("/auth/reset-password", s.Handlers.Auth.ResetPassword)

	// Pool stats (public)
	api.GET("/pool/stats", s.handlePoolStats)
	api.GET("/pool/blocks", s.handlePoolBlocks)

	// Protected routes
	protected := api.Group("/")
	protected.Use(s.AuthMiddleware())
	{
		// User profile
		protected.GET("/user/profile", s.Handlers.User.GetProfile)
		protected.PUT("/user/profile", s.Handlers.User.UpdateProfile)
		protected.PUT("/user/password", s.Handlers.User.ChangePassword)
		protected.GET("/user/miners", s.Handlers.User.GetMiners)
		protected.GET("/user/payouts", s.Handlers.User.GetPayouts)

		// User stats
		protected.GET("/user/stats/hashrate", s.Handlers.User.GetHashrateHistory)
		protected.GET("/user/stats/shares", s.Handlers.User.GetSharesHistory)
		protected.GET("/user/stats/earnings", s.Handlers.User.GetEarningsHistory)
	}
}

// handlePoolStats returns pool statistics using pool services
func (s *ServerWithServices) handlePoolStats(c *gin.Context) {
	stats, err := s.Services.Pool.StatsProvider.GetPoolStats()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get pool stats"})
		return
	}

	c.JSON(200, gin.H{
		"total_miners":     stats.ConnectedMiners,
		"total_hashrate":   stats.TotalHashrate,
		"blocks_found":     stats.BlocksFound,
		"pool_fee":         stats.PoolFee,
		"minimum_payout":   1.0,
		"payment_interval": "1 hour",
		"network":          "BlockDAG Awakening",
		"currency":         "BDAG",
	})
}

// handlePoolBlocks returns recent blocks using pool services
func (s *ServerWithServices) handlePoolBlocks(c *gin.Context) {
	blocks, err := s.Services.Pool.BlockReader.GetRecentBlocks(50)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get blocks"})
		return
	}

	c.JSON(200, gin.H{"blocks": blocks})
}
