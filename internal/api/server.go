package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// SERVER - Unified API Server with Route Registration
// Provides structure for organizing routes while maintaining backward compatibility
// with existing main.go handler patterns
// =============================================================================

// ServerConfig holds all configuration for the API server
type ServerConfig struct {
	Port         string
	Environment  string
	JWTSecret    string
	DatabaseURL  string
	RedisURL     string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	FrontendURL  string
}

// LoadServerConfig loads configuration from environment variables
func LoadServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:         getEnvOrDefault("PORT", "8080"),
		Environment:  getEnvOrDefault("ENVIRONMENT", "development"),
		JWTSecret:    getEnvOrDefault("JWT_SECRET", "default-secret-change-me"),
		DatabaseURL:  getEnvOrDefault("DATABASE_URL", "postgres://chimera:password@localhost:5432/chimera_pool?sslmode=disable"),
		RedisURL:     getEnvOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		SMTPHost:     getEnvOrDefault("SMTP_HOST", "smtp.example.com"),
		SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUser:     getEnvOrDefault("SMTP_USER", ""),
		SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnvOrDefault("SMTP_FROM", "noreply@chimerapool.com"),
		FrontendURL:  getEnvOrDefault("FRONTEND_URL", "http://localhost:3000"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Server represents the API server with all dependencies
type Server struct {
	Config     *ServerConfig
	DB         *sql.DB
	Redis      *redis.Client
	Router     *gin.Engine
	HTTPServer *http.Server
}

// NewServer creates a new API server with basic setup
func NewServer(config *ServerConfig, db *sql.DB, redisClient *redis.Client) *Server {
	// Set Gin mode based on environment
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	s := &Server{
		Config: config,
		DB:     db,
		Redis:  redisClient,
		Router: router,
	}

	// Register health check
	router.GET("/health", s.handleHealth)

	// Create HTTP server
	s.HTTPServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// APIGroup returns the /api/v1 router group for registering routes
func (s *Server) APIGroup() *gin.RouterGroup {
	return s.Router.Group("/api/v1")
}

// handleHealth returns server health status
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "chimera-pool-api",
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("✅ API Server listening on port %s", s.Config.Port)
	return s.HTTPServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.HTTPServer.Shutdown(ctx)
}

// Run starts the server and handles graceful shutdown
func (s *Server) Run() error {
	// Start server in goroutine
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
	return nil
}
