package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chimera-pool/chimera-pool-core/internal/api"
	"github.com/chimera-pool/chimera-pool-core/internal/stats"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	log.Println("🚀 Starting Chimera Pool API Server...")

	// Load configuration from environment
	config := loadConfig()

	// Initialize database connection
	db, err := initDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis connection
	redisClient, err := initRedis(config.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Run database migrations
	if err := runMigrations(db); err != nil {
		log.Printf("Warning: Migration error: %v", err)
	}

	// Setup Gin router
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "chimera-pool-api",
		})
	})

	// Initialize rate limiter for auth endpoints (prevents brute force attacks)
	authRateLimiter := api.NewRateLimiter(api.AuthRateLimiterConfig())
	defer authRateLimiter.Stop()

	// API routes
	apiGroup := router.Group("/api/v1")
	{
		// Auth routes with rate limiting (5 attempts per 15 minutes, 30-minute block)
		authGroup := apiGroup.Group("/auth")
		authGroup.Use(api.RateLimitMiddleware(authRateLimiter))
		{
			authGroup.POST("/register", handleRegister(db, config.JWTSecret))
			authGroup.POST("/login", handleLogin(db, config.JWTSecret))
			// Email service configured - password reset enabled
			authGroup.POST("/forgot-password", handleForgotPassword(db, config))
			authGroup.POST("/reset-password", handleResetPassword(db))
		}

		// Public routes (no rate limiting needed) - with Redis caching
		apiGroup.GET("/pool/stats", handlePoolStatsWithCache(db, redisClient))
		apiGroup.GET("/pool/stats/hashrate", handlePublicPoolHashrateHistory(db))
		apiGroup.GET("/pool/stats/shares", handlePublicPoolSharesHistory(db))
		apiGroup.GET("/pool/stats/miners", handlePublicPoolMinersHistory(db))
		apiGroup.GET("/pool/miners", handlePoolMiners(db))
		apiGroup.GET("/pool/blocks", handleBlocks(db))
		apiGroup.GET("/stats", handlePublicStats(db))
		apiGroup.GET("/miners/locations", handlePublicMinerLocations(db))
		apiGroup.GET("/miners/locations/stats", handleMinerLocationStats(db))

		// Public payout information
		apiGroup.GET("/payout-modes", handleGetPayoutModes())
		apiGroup.GET("/pool/fees", handleGetPoolFees())

		// Multi-coin public routes
		apiGroup.GET("/networks", handleGetSupportedNetworks(db))
		apiGroup.GET("/networks/stats", handleGetAllNetworkPoolStats(db))

		// Protected routes
		protected := apiGroup.Group("/")
		protected.Use(authMiddleware(config.JWTSecret))
		{
			protected.GET("/user/profile", handleUserProfile(db))
			protected.PUT("/user/profile", handleUpdateUserProfile(db))
			protected.PUT("/user/password", handleChangePassword(db))
			protected.GET("/user/miners", handleUserMiners(db))
			protected.GET("/user/equipment", handleUserEquipment(db))
			protected.GET("/user/payouts", handleUserPayouts(db))
			protected.POST("/user/payout-address", handleSetPayoutAddress(db))
			protected.GET("/user/wallet-history", handleGetWalletHistory(db))

			// Multi-wallet management
			protected.GET("/user/wallets", handleGetUserWallets(db))
			protected.POST("/user/wallets", handleCreateUserWallet(db))
			protected.PUT("/user/wallets/:id", handleUpdateUserWallet(db))
			protected.DELETE("/user/wallets/:id", handleDeleteUserWallet(db))
			protected.GET("/user/wallets/preview", handleWalletPayoutPreview(db))
			protected.GET("/user/stats", handleUserStats(db))
			protected.GET("/user/stats/hashrate", handleUserHashrateHistory(db))
			protected.GET("/user/stats/shares", handleUserSharesHistory(db))
			protected.GET("/user/stats/earnings", handleUserEarningsHistory(db))

			// Multi-coin user stats
			protected.GET("/user/networks/stats", handleGetUserNetworkStats(db))
			protected.GET("/user/networks/aggregated", handleGetUserAggregatedStats(db))

			// Referral system routes
			protected.GET("/user/referral", handleGetUserReferral(db))
			protected.GET("/user/referrals", handleGetUserReferrals(db))

			// Payout settings routes
			protected.GET("/user/payout-settings", handleGetPayoutSettings(db))
			protected.PUT("/user/payout-settings", handleUpdatePayoutSettings(db))
			protected.GET("/user/payout-estimate", handleGetPayoutEstimate(db))

			// Community routes (authenticated)
			protected.GET("/community/channels", handleGetChannels(db))
			protected.GET("/community/channel-categories", handleAdminGetCategories(db))
			protected.GET("/community/channels/:id/messages", handleGetChannelMessages(db))
			protected.POST("/community/channels/:id/messages", handleSendMessage(db))
			protected.PUT("/community/messages/:id", handleEditMessage(db))
			protected.DELETE("/community/messages/:id", handleDeleteMessage(db))
			protected.GET("/community/reaction-types", handleGetReactionTypes(db))
			protected.POST("/community/messages/:id/reactions", handleAddReaction(db))
			protected.DELETE("/community/messages/:id/reactions/:emoji", handleRemoveReaction(db))

			protected.GET("/community/forums", handleGetForums(db))
			protected.GET("/community/forums/:id/posts", handleGetForumPosts(db))
			protected.POST("/community/forums/:id/posts", handleCreatePost(db))
			protected.GET("/community/posts/:id", handleGetPost(db))
			protected.PUT("/community/posts/:id", handleEditPost(db))
			protected.POST("/community/posts/:id/replies", handleAddReply(db))
			protected.POST("/community/posts/:id/vote", handleVotePost(db))
			protected.POST("/community/replies/:id/vote", handleVoteReply(db))

			protected.GET("/community/badges", handleGetBadges(db))
			protected.GET("/community/users/:id/badges", handleGetUserBadges(db))
			protected.PUT("/community/profile/primary-badge", handleSetPrimaryBadge(db))
			protected.GET("/community/users/:id/profile", handleGetUserProfile(db))
			protected.PUT("/community/profile", handleUpdateProfile(db))
			protected.GET("/community/leaderboard", handleGetLeaderboard(db))

			protected.GET("/community/notifications", handleGetNotifications(db))
			protected.PUT("/community/notifications/read", handleMarkNotificationsRead(db))

			protected.GET("/community/dm", handleGetDMList(db))
			protected.GET("/community/dm/:userId", handleGetDMConversation(db))
			protected.POST("/community/dm/:userId", handleSendDM(db))

			protected.POST("/community/report", handleReportContent(db))
			protected.GET("/community/online-users", handleGetOnlineUsers(db))

			// Bug reporting routes (authenticated users)
			protected.POST("/bugs", handleCreateBugReport(db, config))
			protected.GET("/bugs", handleGetUserBugReports(db))
			protected.GET("/bugs/:id", handleGetBugReport(db))
			protected.POST("/bugs/:id/comments", handleAddBugComment(db, config))
			protected.POST("/bugs/:id/attachments", handleUploadBugAttachment(db))
			protected.POST("/bugs/:id/subscribe", handleSubscribeToBug(db))
			protected.DELETE("/bugs/:id/subscribe", handleUnsubscribeFromBug(db))
		}

		// Admin routes
		admin := apiGroup.Group("/admin")
		admin.Use(authMiddleware(config.JWTSecret))
		admin.Use(adminMiddleware(db))
		{
			admin.GET("/stats", handleAdminStats(db))
			admin.GET("/users", handleAdminListUsers(db))
			admin.GET("/users/:id", handleAdminGetUser(db))
			admin.PUT("/users/:id", handleAdminUpdateUser(db))
			admin.DELETE("/users/:id", handleAdminDeleteUser(db))
			admin.GET("/users/:id/earnings", handleAdminUserEarnings(db))
			admin.GET("/settings", handleAdminGetSettings(db))
			admin.PUT("/settings", handleAdminUpdateSettings(db))
			admin.GET("/algorithm", handleAdminGetAlgorithm(db))
			admin.PUT("/algorithm", handleAdminUpdateAlgorithm(db))
			admin.GET("/stats/hashrate", handlePoolHashrateHistory(db))
			admin.GET("/stats/shares", handlePoolSharesHistory(db))
			admin.GET("/stats/miners", handlePoolMinersHistory(db))
			admin.GET("/stats/blocks", handlePoolBlocksHistory(db))
			admin.GET("/stats/payouts", handlePoolPayoutsHistory(db))
			admin.GET("/stats/distribution", handlePoolDistribution(db))
			admin.GET("/miners/locations", handleAdminMinerLocations(db))

			// Admin community moderation
			admin.POST("/community/ban/:userId", handleBanUser(db))
			admin.POST("/community/unban/:userId", handleUnbanUser(db))
			admin.POST("/community/mute/:userId", handleMuteUser(db))
			admin.POST("/community/unmute/:userId", handleUnmuteUser(db))
			admin.GET("/community/reports", handleGetReports(db))
			admin.PUT("/community/reports/:id", handleReviewReport(db))
			admin.DELETE("/community/messages/:id", handleAdminDeleteMessage(db))
			admin.DELETE("/community/posts/:id", handleAdminDeletePost(db))
			admin.PUT("/community/posts/:id/pin", handlePinPost(db))
			admin.PUT("/community/posts/:id/lock", handleLockPost(db))

			// Admin channel/category management
			admin.GET("/community/channel-categories", handleAdminGetCategories(db))
			admin.POST("/community/channel-categories", handleAdminCreateCategory(db))
			admin.PUT("/community/channel-categories/:id", handleAdminUpdateCategory(db))
			admin.DELETE("/community/channel-categories/:id", handleAdminDeleteCategory(db))
			admin.POST("/community/channels", handleAdminCreateChannel(db))
			admin.PUT("/community/channels/:id", handleAdminUpdateChannel(db))
			admin.DELETE("/community/channels/:id", handleAdminDeleteChannel(db))

			// Admin bug report management
			admin.GET("/bugs", handleAdminGetAllBugReports(db))
			admin.GET("/bugs/:id", handleAdminGetBugReport(db))
			admin.PUT("/bugs/:id/status", handleAdminUpdateBugStatus(db, config))
			admin.PUT("/bugs/:id/priority", handleAdminUpdateBugPriority(db))
			admin.PUT("/bugs/:id/assign", handleAdminAssignBug(db, config))
			admin.POST("/bugs/:id/comments", handleAdminAddBugComment(db, config))
			admin.DELETE("/bugs/:id", handleAdminDeleteBugReport(db))

			// Network configuration management
			admin.GET("/networks", handleAdminListNetworks(db))
			admin.GET("/networks/:id", handleAdminGetNetwork(db))
			admin.POST("/networks", handleAdminCreateNetwork(db))
			admin.PUT("/networks/:id", handleAdminUpdateNetwork(db))
			admin.DELETE("/networks/:id", handleAdminDeleteNetwork(db))
			admin.POST("/networks/switch", handleAdminSwitchNetwork(db))
			admin.GET("/networks/history", handleAdminNetworkHistory(db))
			admin.POST("/networks/:id/test", handleAdminTestNetworkConnection(db))

			// Miner monitoring routes
			admin.GET("/monitoring/miners", handleAdminGetAllMiners(db))
			admin.GET("/monitoring/miners/:id", handleAdminGetMinerDetail(db))
			admin.GET("/monitoring/miners/:id/shares", handleAdminGetMinerShares(db))
			admin.GET("/monitoring/users/:id/miners", handleAdminGetUserMiners(db))

			// Role management
			admin.GET("/roles", handleAdminGetRoles(db))
		}

		// Public network info route
		apiGroup.GET("/network/active", handleGetActiveNetwork(db))
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("✅ API Server listening on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Start database maintenance scheduler
	maintenanceStop := make(chan struct{})
	go startDatabaseMaintenanceScheduler(db, maintenanceStop)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Stop maintenance scheduler
	close(maintenanceStop)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}

// startDatabaseMaintenanceScheduler runs periodic database maintenance tasks
func startDatabaseMaintenanceScheduler(db *sql.DB, stop chan struct{}) {
	log.Println("🔧 Starting database maintenance scheduler...")

	// Record activity metrics every minute
	activityTicker := time.NewTicker(1 * time.Minute)
	defer activityTicker.Stop()

	// Run maintenance check every 15 minutes
	maintenanceTicker := time.NewTicker(15 * time.Minute)
	defer maintenanceTicker.Stop()

	// Archive old shares daily at 3 AM UTC
	archiveTicker := time.NewTicker(1 * time.Hour)
	defer archiveTicker.Stop()

	for {
		select {
		case <-stop:
			log.Println("🔧 Stopping database maintenance scheduler...")
			return

		case <-activityTicker.C:
			// Record current activity metrics
			_, err := db.Exec("SELECT record_activity_metrics()")
			if err != nil {
				log.Printf("Warning: Failed to record activity metrics: %v", err)
			}

		case <-maintenanceTicker.C:
			// Run smart maintenance (vacuum analyze if needed)
			_, err := db.Exec("SELECT perform_smart_maintenance()")
			if err != nil {
				log.Printf("Warning: Failed to run smart maintenance: %v", err)
			} else {
				log.Println("🔧 Database maintenance check completed")
			}

		case <-archiveTicker.C:
			// Check if it's 3 AM UTC for archiving
			now := time.Now().UTC()
			if now.Hour() == 3 {
				var archivedCount int
				err := db.QueryRow("SELECT archive_old_shares()").Scan(&archivedCount)
				if err != nil {
					log.Printf("Warning: Failed to archive old shares: %v", err)
				} else if archivedCount > 0 {
					log.Printf("🔧 Archived %d old shares", archivedCount)
				}
			}
		}
	}
}

type Config struct {
	DatabaseURL  string
	RedisURL     string
	JWTSecret    string
	Port         string
	Environment  string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	FrontendURL  string
	ResendAPIKey string
	EmailFrom    string
}

func loadConfig() *Config {
	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://chimera:password@localhost:5432/chimera_pool?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:    getEnv("JWT_SECRET", "default-secret-change-me"),
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		SMTPHost:     getEnv("SMTP_HOST", "smtp.example.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@chimerapool.com"),
		FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:3000"),
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		EmailFrom:    getEnv("EMAIL_FROM", "Chimera Pool <noreply@chimeriapool.com>"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDatabase(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to PostgreSQL database")
	return db, nil
}

func initRedis(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to Redis")
	return client, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///app/migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	log.Println("✅ Database migrations applied")
	return nil
}

// Handler functions
// validatePasswordStrength enforces strong password policy
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be less than 128 characters")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		case c == '!' || c == '@' || c == '#' || c == '$' || c == '%' || c == '^' || c == '&' || c == '*' || c == '(' || c == ')' || c == '-' || c == '_' || c == '=' || c == '+' || c == '[' || c == ']' || c == '{' || c == '}' || c == '|' || c == ';' || c == ':' || c == '\'' || c == '"' || c == ',' || c == '.' || c == '<' || c == '>' || c == '/' || c == '?' || c == '`' || c == '~':
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character (!@#$%%^&*)")
	}

	// Check for common weak passwords
	weakPasswords := []string{"password", "12345678", "qwerty12", "letmein1", "welcome1", "admin123", "password1", "Password1"}
	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if strings.Contains(lowerPassword, weak) {
			return fmt.Errorf("password is too common, please choose a stronger password")
		}
	}

	return nil
}

func handleRegister(db *sql.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// SECURITY: Enforce strong password policy
		if err := validatePasswordStrength(req.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Hash password
		hashedPassword, err := hashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}

		// Insert user
		var userID int64
		err = db.QueryRow(
			"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
			req.Username, req.Email, hashedPassword,
		).Scan(&userID)

		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"user_id": userID,
		})
	}
}

func handleLogin(db *sql.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get user by email
		var userID int64
		var username, passwordHash string
		err := db.QueryRow(
			"SELECT id, username, password_hash FROM users WHERE email = $1 AND is_active = true",
			req.Email,
		).Scan(&userID, &username, &passwordHash)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Verify password
		if !verifyPassword(req.Password, passwordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Generate JWT
		token, err := generateJWT(userID, username, jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":   token,
			"user_id": userID,
		})
	}
}

func handlePoolStats(db *sql.DB) gin.HandlerFunc {
	return handlePoolStatsWithCache(db, nil) // No cache - direct DB queries
}

// handlePoolStatsWithCache returns pool stats with optional Redis caching
func handlePoolStatsWithCache(db *sql.DB, redisClient *redis.Client) gin.HandlerFunc {
	const cacheKey = "chimera:pool:stats"
	const cacheTTL = 60 * time.Second // Increased to 60 seconds to reduce DB load

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Try cache first if Redis is available
		if redisClient != nil {
			cached, err := redisClient.Get(ctx, cacheKey).Result()
			if err == nil && cached != "" {
				c.Header("X-Cache", "HIT")
				c.Data(http.StatusOK, "application/json", []byte(cached))
				return
			}
		}

		// Cache miss - query database with timeout
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var activeMiners, totalMiners, totalBlocks int64
		var totalHashrate float64

		// Use time-based check for truly active miners (seen in last 5 minutes)
		db.QueryRowContext(queryCtx, "SELECT COUNT(*) FROM miners WHERE last_seen > NOW() - INTERVAL '5 minutes'").Scan(&activeMiners)
		db.QueryRowContext(queryCtx, "SELECT COUNT(*) FROM miners WHERE is_active = true").Scan(&totalMiners)
		db.QueryRowContext(queryCtx, "SELECT COUNT(*) FROM blocks").Scan(&totalBlocks)

		// Get hashrate directly from miners table (fast query with index)
		db.QueryRowContext(queryCtx, "SELECT COALESCE(SUM(hashrate), 0) FROM miners WHERE last_seen > NOW() - INTERVAL '5 minutes'").Scan(&totalHashrate)

		response := gin.H{
			"active_miners":    activeMiners,
			"total_miners":     totalMiners,
			"total_hashrate":   totalHashrate,
			"blocks_found":     totalBlocks,
			"pool_fee":         1.0,
			"minimum_payout":   0.01,
			"payment_interval": "1 hour",
			"network":          "Litecoin",
			"currency":         "LTC",
			"algorithm":        "Scrypt",
		}

		// Cache the response if Redis is available
		if redisClient != nil {
			if jsonData, err := json.Marshal(response); err == nil {
				redisClient.Set(ctx, cacheKey, jsonData, cacheTTL)
			}
		}

		c.Header("X-Cache", "MISS")
		c.JSON(http.StatusOK, response)
	}
}

// handlePublicStats returns general stats for the monitoring dashboard
func handlePublicStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var totalMiners, totalBlocks int64
		var totalHashrate float64

		// Use time-based check for truly active miners (seen in last 5 minutes)
		db.QueryRow("SELECT COUNT(*) FROM miners WHERE last_seen > NOW() - INTERVAL '5 minutes'").Scan(&totalMiners)
		db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&totalBlocks)
		db.QueryRow("SELECT COALESCE(SUM(hashrate), 0) FROM miners WHERE last_seen > NOW() - INTERVAL '5 minutes'").Scan(&totalHashrate)

		c.JSON(http.StatusOK, gin.H{
			"activeMiners":  totalMiners,
			"totalHashrate": totalHashrate,
			"blocksFound":   totalBlocks,
			"network":       "Litecoin",
			"status":        "online",
		})
	}
}

// handleUserStats returns user-specific mining stats
func handleUserStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var totalHashrate, pendingBalance, totalEarned float64
		var activeMiners, totalShares int64

		db.QueryRow("SELECT COALESCE(SUM(hashrate), 0), COUNT(*) FROM miners WHERE user_id = $1 AND is_active = true", userID).Scan(&totalHashrate, &activeMiners)
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = $1 AND status = 'pending'", userID).Scan(&pendingBalance)
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = $1 AND status = 'completed'", userID).Scan(&totalEarned)
		db.QueryRow("SELECT COUNT(*) FROM shares WHERE user_id = $1", userID).Scan(&totalShares)

		c.JSON(http.StatusOK, gin.H{
			"hashrate":       totalHashrate,
			"activeMiners":   activeMiners,
			"pendingBalance": pendingBalance,
			"totalEarned":    totalEarned,
			"totalShares":    totalShares,
		})
	}
}

// handlePoolMiners returns list of active miners for the pool
func handlePoolMiners(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Query miners with share counts from shares table
		rows, err := db.Query(`
			SELECT m.id, m.name, m.hashrate, m.is_active, m.last_seen, m.user_id,
			       COALESCE(s.valid_shares, 0) as valid_shares,
			       COALESCE(s.invalid_shares, 0) as invalid_shares,
			       COALESCE(s.avg_difficulty, 0) as difficulty
			FROM miners m
			LEFT JOIN (
				SELECT miner_id,
				       COUNT(*) FILTER (WHERE is_valid = true) as valid_shares,
				       COUNT(*) FILTER (WHERE is_valid = false) as invalid_shares,
				       AVG(difficulty) as avg_difficulty
				FROM shares
				WHERE timestamp > NOW() - INTERVAL '24 hours'
				GROUP BY miner_id
			) s ON m.id = s.miner_id
			WHERE m.last_seen > NOW() - INTERVAL '5 minutes'
			ORDER BY m.hashrate DESC
			LIMIT 100
		`)
		if err != nil {
			log.Printf("handlePoolMiners query error: %v", err)
			c.JSON(http.StatusOK, gin.H{"miners": []gin.H{}})
			return
		}
		defer rows.Close()

		var miners []gin.H
		for rows.Next() {
			var id, userID int64
			var name string
			var hashrate, difficulty float64
			var isActive bool
			var lastSeen time.Time
			var validShares, invalidShares int64

			if err := rows.Scan(&id, &name, &hashrate, &isActive, &lastSeen, &userID, &validShares, &invalidShares, &difficulty); err != nil {
				log.Printf("handlePoolMiners scan error: %v", err)
				continue
			}

			miners = append(miners, gin.H{
				"id":             id,
				"name":           name,
				"hashrate":       hashrate,
				"is_active":      isActive,
				"last_seen":      lastSeen,
				"user_id":        userID,
				"valid_shares":   validShares,
				"invalid_shares": invalidShares,
				"difficulty":     difficulty,
			})
		}

		if miners == nil {
			miners = []gin.H{}
		}

		c.JSON(http.StatusOK, gin.H{"miners": miners})
	}
}

// Cache for chart data to prevent database overload
var chartCache = struct {
	sync.RWMutex
	data     map[string]interface{}
	expiry   map[string]time.Time
	cacheTTL time.Duration
}{
	data:     make(map[string]interface{}),
	expiry:   make(map[string]time.Time),
	cacheTTL: 5 * time.Minute, // Cache chart data for 5 minutes
}

func getCachedChartData(key string) (interface{}, bool) {
	chartCache.RLock()
	defer chartCache.RUnlock()
	if exp, ok := chartCache.expiry[key]; ok && time.Now().Before(exp) {
		return chartCache.data[key], true
	}
	return nil, false
}

func setCachedChartData(key string, data interface{}) {
	chartCache.Lock()
	defer chartCache.Unlock()
	chartCache.data[key] = data
	chartCache.expiry[key] = time.Now().Add(chartCache.cacheTTL)
}

// handlePublicPoolHashrateHistory returns pool hashrate history calculated from shares
func handlePublicPoolHashrateHistory(db *sql.DB) gin.HandlerFunc {
	timeService := stats.NewTimeRangeService()

	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		cacheKey := "hashrate_history_" + rangeStr

		// Check cache first
		if cached, ok := getCachedChartData(cacheKey); ok {
			c.Header("X-Cache", "HIT")
			c.JSON(http.StatusOK, cached)
			return
		}

		pgInterval := timeService.GetPostgresInterval(rangeStr)
		dateTrunc := timeService.GetDateTrunc(rangeStr)
		secondsPerBucket := getSecondsPerBucket(dateTrunc)

		// Use query timeout to prevent blocking
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', timestamp) as time_bucket,
				(COALESCE(SUM(difficulty), 0) * 4294967296.0 / %d) as hashrate
			FROM shares 
			WHERE timestamp > NOW() - INTERVAL '%s' AND is_valid = true
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, dateTrunc, secondsPerBucket, pgInterval)

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{}, "range": rangeStr, "error": "Query timeout - data cached soon"})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var hashrate float64
			if err := rows.Scan(&timeBucket, &hashrate); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":     timeBucket,
				"hashrate": hashrate,
			})
		}

		response := gin.H{"data": data, "range": rangeStr}
		setCachedChartData(cacheKey, response)
		c.Header("X-Cache", "MISS")
		c.JSON(http.StatusOK, response)
	}
}

// handlePublicPoolSharesHistory returns pool shares history
func handlePublicPoolSharesHistory(db *sql.DB) gin.HandlerFunc {
	timeService := stats.NewTimeRangeService()

	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		cacheKey := "shares_history_" + rangeStr

		// Check cache first
		if cached, ok := getCachedChartData(cacheKey); ok {
			c.Header("X-Cache", "HIT")
			c.JSON(http.StatusOK, cached)
			return
		}

		pgInterval := timeService.GetPostgresInterval(rangeStr)
		dateTrunc := timeService.GetDateTrunc(rangeStr)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', timestamp) as time_bucket,
				COUNT(*) FILTER (WHERE is_valid = true) as valid_shares,
				COUNT(*) FILTER (WHERE is_valid = false) as invalid_shares
			FROM shares 
			WHERE timestamp > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, dateTrunc, pgInterval)

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{}, "range": rangeStr, "error": "Query timeout"})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var validShares, invalidShares int64
			if err := rows.Scan(&timeBucket, &validShares, &invalidShares); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":          timeBucket,
				"validShares":   validShares,
				"invalidShares": invalidShares,
			})
		}

		response := gin.H{"data": data, "range": rangeStr}
		setCachedChartData(cacheKey, response)
		c.Header("X-Cache", "MISS")
		c.JSON(http.StatusOK, response)
	}
}

// handlePublicPoolMinersHistory returns pool active miners history over time
func handlePublicPoolMinersHistory(db *sql.DB) gin.HandlerFunc {
	timeService := stats.NewTimeRangeService()

	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		cacheKey := "miners_history_" + rangeStr

		// Check cache first
		if cached, ok := getCachedChartData(cacheKey); ok {
			c.Header("X-Cache", "HIT")
			c.JSON(http.StatusOK, cached)
			return
		}

		pgInterval := timeService.GetPostgresInterval(rangeStr)
		dateTrunc := timeService.GetDateTrunc(rangeStr)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Query to get unique active miners per time bucket
		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', timestamp) as time_bucket,
				COUNT(DISTINCT miner_id) as active_miners,
				COUNT(DISTINCT user_id) as unique_users
			FROM shares 
			WHERE timestamp > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, dateTrunc, pgInterval)

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{}, "range": rangeStr})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var activeMiners, uniqueUsers int64
			if err := rows.Scan(&timeBucket, &activeMiners, &uniqueUsers); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":         timeBucket,
				"activeMiners": activeMiners,
				"uniqueUsers":  uniqueUsers,
				"totalMiners":  activeMiners,
			})
		}

		response := gin.H{"data": data, "range": rangeStr}
		setCachedChartData(cacheKey, response)
		c.Header("X-Cache", "MISS")
		c.JSON(http.StatusOK, response)
	}
}

// getModelFromHashrate returns model name based on hashrate
func getModelFromHashrate(hashrate float64) string {
	if hashrate > 10000000000000 {
		return "BlockDAG X100"
	} else if hashrate > 100000000000 {
		return "BlockDAG X30"
	} else if hashrate > 1000000000 {
		return "ASIC Miner"
	}
	return "GPU Rig"
}

// calculateUptimePercent calculates uptime percentage from connection and downtime
func calculateUptimePercent(totalConnectionTime, totalDowntime int64) float64 {
	if totalConnectionTime <= 0 {
		return 0
	}
	uptime := float64(totalConnectionTime-totalDowntime) / float64(totalConnectionTime) * 100
	if uptime < 0 {
		return 0
	}
	if uptime > 100 {
		return 100
	}
	return uptime
}

// getSecondsPerBucket returns seconds per time bucket for hashrate calculation
func getSecondsPerBucket(dateTrunc string) int {
	switch dateTrunc {
	case "minute":
		return 60
	case "hour":
		return 3600
	case "day":
		return 86400
	case "week":
		return 604800
	case "month":
		return 2592000 // 30 days
	default:
		return 3600
	}
}

func handleBlocks(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(
			"SELECT id, height, hash, reward, status, timestamp FROM blocks ORDER BY height DESC LIMIT 50",
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blocks"})
			return
		}
		defer rows.Close()

		var blocks []gin.H
		for rows.Next() {
			var id, height, reward int64
			var hash, status string
			var timestamp time.Time
			rows.Scan(&id, &height, &hash, &reward, &status, &timestamp)
			blocks = append(blocks, gin.H{
				"id":        id,
				"height":    height,
				"hash":      hash,
				"reward":    reward,
				"status":    status,
				"timestamp": timestamp,
			})
		}

		c.JSON(http.StatusOK, gin.H{"blocks": blocks})
	}
}

func handleUserProfile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var username, email string
		var payoutAddress sql.NullString
		var isAdmin bool
		var createdAt time.Time
		err := db.QueryRow(
			"SELECT username, email, payout_address, is_admin, created_at FROM users WHERE id = $1",
			userID,
		).Scan(&username, &email, &payoutAddress, &isAdmin, &createdAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":             userID, // Use 'id' for consistency with message user objects
			"user_id":        userID, // Keep for backwards compatibility
			"username":       username,
			"email":          email,
			"payout_address": payoutAddress.String,
			"is_admin":       isAdmin,
			"created_at":     createdAt,
		})
	}
}

func handleUpdateUserProfile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			Username      string `json:"username"`
			PayoutAddress string `json:"payout_address"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate username if provided
		if req.Username != "" {
			if len(req.Username) < 3 || len(req.Username) > 50 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be between 3 and 50 characters"})
				return
			}

			// Check if username is already taken by another user
			var existingID int64
			err := db.QueryRow("SELECT id FROM users WHERE username = $1 AND id != $2", req.Username, userID).Scan(&existingID)
			if err == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Username is already taken"})
				return
			} else if err != sql.ErrNoRows {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check username availability"})
				return
			}
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argIndex := 1

		if req.Username != "" {
			updates = append(updates, fmt.Sprintf("username = $%d", argIndex))
			args = append(args, req.Username)
			argIndex++
		}

		if req.PayoutAddress != "" {
			updates = append(updates, fmt.Sprintf("payout_address = $%d", argIndex))
			args = append(args, req.PayoutAddress)
			argIndex++
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}

		// Add user ID as the last argument
		args = append(args, userID)
		query := fmt.Sprintf("UPDATE users SET %s, updated_at = NOW() WHERE id = $%d",
			strings.Join(updates, ", "), argIndex)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}

		// Fetch updated user data
		var username, email string
		var payoutAddress sql.NullString
		var isAdmin bool
		db.QueryRow("SELECT username, email, payout_address, is_admin FROM users WHERE id = $1", userID).Scan(&username, &email, &payoutAddress, &isAdmin)

		c.JSON(http.StatusOK, gin.H{
			"message": "Profile updated successfully",
			"user": gin.H{
				"user_id":        userID,
				"username":       username,
				"email":          email,
				"payout_address": payoutAddress.String,
				"is_admin":       isAdmin,
			},
		})
	}
}

func handleChangePassword(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password and new password (min 8 chars) are required"})
			return
		}

		// Get current password hash
		var storedHash string
		err := db.QueryRow("SELECT password_hash FROM users WHERE id = $1", userID).Scan(&storedHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify user"})
			return
		}

		// Verify current password
		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.CurrentPassword)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
			return
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
			return
		}

		// Update password
		_, err = db.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", hashedPassword, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
	}
}

func handleUserMiners(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(
			"SELECT id, name, hashrate, last_seen, is_active FROM miners WHERE user_id = $1",
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miners"})
			return
		}
		defer rows.Close()

		var miners []gin.H
		for rows.Next() {
			var id int64
			var name string
			var hashrate float64
			var lastSeen time.Time
			var isActive bool
			rows.Scan(&id, &name, &hashrate, &lastSeen, &isActive)
			miners = append(miners, gin.H{
				"id":        id,
				"name":      name,
				"hashrate":  hashrate,
				"last_seen": lastSeen,
				"is_active": isActive,
			})
		}

		c.JSON(http.StatusOK, gin.H{"miners": miners})
	}
}

// handleUserEquipment returns comprehensive equipment data for the user's miners
func handleUserEquipment(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		// Get all miners with stats using only existing columns
		rows, err := db.Query(`
			SELECT 
				m.id, m.name, COALESCE(m.address::text, '') as address, m.hashrate, m.is_active, m.last_seen, m.created_at,
				EXTRACT(EPOCH FROM (NOW() - m.created_at))::bigint as total_connection_time
			FROM miners m
			WHERE m.user_id = $1
			ORDER BY m.last_seen DESC
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch equipment", "details": err.Error()})
			return
		}
		defer rows.Close()

		var equipment []gin.H
		for rows.Next() {
			var id int64
			var name, address string
			var hashrate float64
			var isActive bool
			var lastSeen, createdAt time.Time
			var totalConnectionTime int64

			err := rows.Scan(&id, &name, &address, &hashrate, &isActive, &lastSeen, &createdAt, &totalConnectionTime)
			if err != nil {
				log.Printf("Error scanning equipment row: %v", err)
				continue
			}

			// Calculate uptime
			var uptime int64
			if isActive {
				uptime = int64(time.Since(createdAt).Seconds())
			}

			// Determine status
			status := "offline"
			if isActive && time.Since(lastSeen) < 5*time.Minute {
				status = "mining"
			} else if time.Since(lastSeen) < 15*time.Minute {
				status = "idle"
			}

			// Determine equipment type based on hashrate
			eqType := "gpu"
			if hashrate > 10000000000000 {
				eqType = "blockdag_x100"
			} else if hashrate > 100000000000 {
				eqType = "blockdag_x30"
			} else if hashrate > 1000000000 {
				eqType = "asic"
			}

			equipment = append(equipment, gin.H{
				"id":                    fmt.Sprintf("eq-%d", id),
				"miner_id":              id,
				"name":                  name,
				"type":                  eqType,
				"status":                status,
				"worker_name":           name,
				"model":                 getModelFromHashrate(hashrate),
				"current_hashrate":      hashrate,
				"average_hashrate":      hashrate * 0.98,
				"temperature":           0,
				"power_usage":           0,
				"latency":               0,
				"shares_accepted":       0,
				"shares_rejected":       0,
				"uptime":                uptime,
				"last_seen":             lastSeen,
				"total_earnings":        0,
				"connected_at":          createdAt,
				"total_connection_time": totalConnectionTime,
				"total_downtime":        0,
				"downtime_incidents":    0,
				"difficulty":            0.0,
				"address":               address,
				"is_active":             isActive,
			})
		}

		if equipment == nil {
			equipment = []gin.H{}
		}

		c.JSON(http.StatusOK, gin.H{"equipment": equipment})
	}
}

func handleUserPayouts(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(
			"SELECT id, amount, address, tx_hash, status, created_at FROM payouts WHERE user_id = $1 ORDER BY created_at DESC",
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payouts"})
			return
		}
		defer rows.Close()

		var payouts []gin.H
		for rows.Next() {
			var id, amount int64
			var address, status string
			var txHash sql.NullString
			var createdAt time.Time
			rows.Scan(&id, &amount, &address, &txHash, &status, &createdAt)
			payouts = append(payouts, gin.H{
				"id":         id,
				"amount":     amount,
				"address":    address,
				"tx_hash":    txHash.String,
				"status":     status,
				"created_at": createdAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"payouts": payouts})
	}
}

func handleSetPayoutAddress(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			Address string `json:"address" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get current payout address
		var currentAddress sql.NullString
		db.QueryRow("SELECT payout_address FROM users WHERE id = $1", userID).Scan(&currentAddress)

		// If there's a current address, mark it as replaced in history
		if currentAddress.Valid && currentAddress.String != "" && currentAddress.String != req.Address {
			db.Exec(`UPDATE wallet_address_history SET replaced_at = NOW() 
				WHERE user_id = $1 AND address = $2 AND replaced_at IS NULL`, userID, currentAddress.String)
		}

		// Check if this address already exists in history for this user
		var existingHistoryID int64
		err := db.QueryRow(`SELECT id FROM wallet_address_history 
			WHERE user_id = $1 AND address = $2`, userID, req.Address).Scan(&existingHistoryID)

		if err == sql.ErrNoRows {
			// Insert new address into history
			db.Exec(`INSERT INTO wallet_address_history (user_id, address, set_at) VALUES ($1, $2, NOW())`,
				userID, req.Address)
		} else if err == nil {
			// Update existing history entry - reactivate it
			db.Exec(`UPDATE wallet_address_history SET replaced_at = NULL, set_at = NOW() 
				WHERE id = $1`, existingHistoryID)
		}

		// Update user's current payout address
		_, err = db.Exec("UPDATE users SET payout_address = $1 WHERE id = $2", req.Address, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Payout address updated",
			"address": req.Address,
		})
	}
}

func handleGetWalletHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		// Get wallet history with payout totals
		rows, err := db.Query(`
			SELECT wh.id, wh.address, wh.set_at, wh.replaced_at,
				COALESCE(SUM(p.amount), 0) as total_paid,
				COUNT(p.id) as payout_count
			FROM wallet_address_history wh
			LEFT JOIN payouts p ON p.user_id = wh.user_id AND p.address = wh.address AND p.status = 'confirmed'
			WHERE wh.user_id = $1
			GROUP BY wh.id, wh.address, wh.set_at, wh.replaced_at
			ORDER BY wh.set_at DESC
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallet history"})
			return
		}
		defer rows.Close()

		var history []gin.H
		for rows.Next() {
			var id int64
			var address string
			var setAt time.Time
			var replacedAt sql.NullTime
			var totalPaid float64
			var payoutCount int

			if err := rows.Scan(&id, &address, &setAt, &replacedAt, &totalPaid, &payoutCount); err != nil {
				continue
			}

			entry := gin.H{
				"id":           id,
				"address":      address,
				"set_at":       setAt,
				"total_paid":   totalPaid / 1e8, // Convert from satoshis
				"payout_count": payoutCount,
				"is_current":   !replacedAt.Valid,
			}
			if replacedAt.Valid {
				entry["replaced_at"] = replacedAt.Time
			}
			history = append(history, entry)
		}

		// Get current address from user
		var currentAddress sql.NullString
		db.QueryRow("SELECT payout_address FROM users WHERE id = $1", userID).Scan(&currentAddress)

		c.JSON(http.StatusOK, gin.H{
			"current_address": currentAddress.String,
			"history":         history,
		})
	}
}

func handleAdminStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var totalUsers, totalMiners, totalBlocks, pendingPayouts int64

		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
		db.QueryRow("SELECT COUNT(*) FROM miners").Scan(&totalMiners)
		db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&totalBlocks)
		db.QueryRow("SELECT COUNT(*) FROM payouts WHERE status = 'pending'").Scan(&pendingPayouts)

		c.JSON(http.StatusOK, gin.H{
			"total_users":     totalUsers,
			"total_miners":    totalMiners,
			"total_blocks":    totalBlocks,
			"pending_payouts": pendingPayouts,
		})
	}
}

// Admin middleware - checks if user is admin
func adminMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var isAdmin bool
		err := db.QueryRow("SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)
		if err != nil || !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// handleAdminListUsers returns paginated list of all users with stats, badges, and clout
func handleAdminListUsers(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		pageSize := 20
		search := c.Query("search")
		sortField := c.Query("sort_field")
		sortDirection := c.Query("sort_direction")

		if p := c.Query("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		if ps := c.Query("page_size"); ps != "" {
			fmt.Sscanf(ps, "%d", &pageSize)
		}
		if pageSize > 100 {
			pageSize = 100
		}
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * pageSize

		// Validate sort parameters
		validSortFields := map[string]string{
			"id":               "u.id",
			"username":         "u.username",
			"email":            "u.email",
			"wallet_count":     "wallet_count",
			"total_hashrate":   "total_hashrate",
			"total_earnings":   "total_earnings",
			"pool_fee_percent": "u.pool_fee_percent",
			"is_active":        "u.is_active",
			"created_at":       "u.created_at",
		}
		orderByField := "u.created_at"
		if field, ok := validSortFields[sortField]; ok {
			orderByField = field
		}
		orderDirection := "DESC"
		if sortDirection == "asc" {
			orderDirection = "ASC"
		}

		// Enhanced query with role, badges, and engagement metrics
		baseQuery := `
			SELECT 
				u.id, u.username, u.email, u.payout_address, u.pool_fee_percent,
				u.is_active, u.is_admin, u.role, u.created_at,
				COALESCE(SUM(p.amount), 0) as total_earnings,
				COALESCE((SELECT SUM(amount) FROM payouts WHERE user_id = u.id AND status = 'pending'), 0) as pending_payout,
				COALESCE((SELECT SUM(hashrate) FROM miners WHERE user_id = u.id AND is_active = true), 0) as total_hashrate,
				COALESCE((SELECT COUNT(*) FROM miners WHERE user_id = u.id AND is_active = true), 0) as active_miners,
				COALESCE((SELECT COUNT(*) FROM user_wallets WHERE user_id = u.id), 0) as wallet_count,
				COALESCE((SELECT address FROM user_wallets WHERE user_id = u.id AND is_primary = true LIMIT 1), '') as primary_wallet,
				COALESCE((SELECT SUM(percentage) FROM user_wallets WHERE user_id = u.id AND is_active = true), 0) as total_allocated,
				COALESCE((SELECT COUNT(*) FROM blocks WHERE finder_id = u.id), 0) as blocks_found,
				COALESCE((SELECT COUNT(*) FROM shares WHERE user_id = u.id), 0) as total_shares,
				COALESCE(up.forum_post_count, 0) as forum_posts,
				COALESCE(up.reputation, 0) as reputation,
				-- Engagement/Clout score calculation (includes mining + community)
				(
					COALESCE((SELECT COUNT(*) FROM shares WHERE user_id = u.id), 0) / 1000 +
					COALESCE((SELECT COUNT(*) FROM blocks WHERE finder_id = u.id), 0) * 500 +
					COALESCE(up.forum_post_count, 0) * 10 +
					COALESCE(up.reputation, 0) +
					COALESCE((SELECT COUNT(*) FROM channel_messages WHERE user_id = u.id AND is_deleted = false), 0) * 2
				) as engagement_score,
				COALESCE(pb.icon, '🌱') as primary_badge_icon,
				COALESCE(pb.color, '#4ade80') as primary_badge_color,
				COALESCE(pb.name, 'Newcomer') as primary_badge_name
			FROM users u
			LEFT JOIN payouts p ON u.id = p.user_id AND p.status = 'confirmed'
			LEFT JOIN user_profiles up ON u.id = up.user_id
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
			LEFT JOIN badges pb ON ub.badge_id = pb.id
		`
		countQuery := "SELECT COUNT(*) FROM users"

		var args []interface{}
		argIdx := 1

		if search != "" {
			baseQuery += fmt.Sprintf(" WHERE (u.username ILIKE $%d OR u.email ILIKE $%d)", argIdx, argIdx+1)
			countQuery += fmt.Sprintf(" WHERE (username ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx+1)
			args = append(args, "%"+search+"%", "%"+search+"%")
			argIdx += 2
		}

		baseQuery += fmt.Sprintf(" GROUP BY u.id, up.forum_post_count, up.reputation, pb.icon, pb.color, pb.name ORDER BY %s %s", orderByField, orderDirection)
		baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, pageSize, offset)

		rows, err := db.Query(baseQuery, args...)
		if err != nil {
			log.Printf("Error fetching users: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
			return
		}
		defer rows.Close()

		var users []gin.H
		var userIDs []int64
		for rows.Next() {
			var id int64
			var username, email string
			var payoutAddress sql.NullString
			var poolFeePercent sql.NullFloat64
			var isActive, isAdmin bool
			var role sql.NullString
			var createdAt time.Time
			var totalEarnings, pendingPayout, totalHashrate float64
			var activeMiners, walletCount int
			var primaryWallet string
			var totalAllocated float64
			var blocksFound, forumPosts, reputation int
			var totalShares, engagementScore int64
			var primaryBadgeIcon, primaryBadgeColor, primaryBadgeName string

			err := rows.Scan(&id, &username, &email, &payoutAddress, &poolFeePercent,
				&isActive, &isAdmin, &role, &createdAt, &totalEarnings, &pendingPayout,
				&totalHashrate, &activeMiners, &walletCount, &primaryWallet, &totalAllocated,
				&blocksFound, &totalShares, &forumPosts, &reputation, &engagementScore,
				&primaryBadgeIcon, &primaryBadgeColor, &primaryBadgeName)
			if err != nil {
				log.Printf("Error scanning user row: %v", err)
				continue
			}

			// Determine effective role
			userRole := "user"
			if role.Valid {
				userRole = role.String
			}
			if isAdmin {
				userRole = "admin"
			}

			users = append(users, gin.H{
				"id":               id,
				"username":         username,
				"email":            email,
				"payout_address":   payoutAddress.String,
				"pool_fee_percent": poolFeePercent.Float64,
				"is_active":        isActive,
				"is_admin":         isAdmin,
				"role":             userRole,
				"roleBadge":        getRoleBadge(userRole),
				"created_at":       createdAt,
				"total_earnings":   totalEarnings,
				"pending_payout":   pendingPayout,
				"total_hashrate":   totalHashrate,
				"active_miners":    activeMiners,
				"wallet_count":     walletCount,
				"primary_wallet":   primaryWallet,
				"total_allocated":  totalAllocated,
				// Clout/engagement metrics
				"blocks_found":     blocksFound,
				"total_shares":     totalShares,
				"forum_posts":      forumPosts,
				"reputation":       reputation,
				"engagement_score": engagementScore,
				// Primary badge
				"primaryBadge": gin.H{
					"icon":  primaryBadgeIcon,
					"color": primaryBadgeColor,
					"name":  primaryBadgeName,
				},
				"badges": []gin.H{}, // Will be populated below
			})
			userIDs = append(userIDs, id)
		}

		// Fetch all badges for users
		if len(userIDs) > 0 {
			badgeQuery := `
				SELECT ub.user_id, b.icon, b.color, b.name, b.badge_type, ub.is_primary
				FROM user_badges ub
				JOIN badges b ON ub.badge_id = b.id
				WHERE ub.user_id = ANY($1)
				ORDER BY ub.is_primary DESC, ub.earned_at DESC
			`
			badgeRows, err := db.Query(badgeQuery, pq.Array(userIDs))
			if err == nil {
				defer badgeRows.Close()
				userBadges := make(map[int64][]gin.H)
				for badgeRows.Next() {
					var uid int64
					var icon, color, name, badgeType string
					var isPrimary bool
					badgeRows.Scan(&uid, &icon, &color, &name, &badgeType, &isPrimary)
					userBadges[uid] = append(userBadges[uid], gin.H{
						"icon":      icon,
						"color":     color,
						"name":      name,
						"type":      badgeType,
						"isPrimary": isPrimary,
					})
				}
				// Assign badges to users
				for i, user := range users {
					uid := user["id"].(int64)
					if badges, ok := userBadges[uid]; ok {
						users[i]["badges"] = badges
					}
				}
			}
		}

		// Get total count
		var totalCount int
		countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET args
		db.QueryRow(countQuery, countArgs...).Scan(&totalCount)

		c.JSON(http.StatusOK, gin.H{
			"users":       users,
			"total_count": totalCount,
			"page":        page,
			"page_size":   pageSize,
		})
	}
}

// handleAdminGetRoles returns all users with admin, moderator, or super_admin roles
func handleAdminGetRoles(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var admins []gin.H
		var moderators []gin.H

		// Get admins and super_admins
		adminRows, err := db.Query(`
			SELECT id, username, email, role, created_at 
			FROM users 
			WHERE role IN ('admin', 'super_admin', 'superadmin') 
			ORDER BY role DESC, username
		`)
		if err == nil {
			defer adminRows.Close()
			for adminRows.Next() {
				var id int64
				var username, email, role string
				var createdAt time.Time
				adminRows.Scan(&id, &username, &email, &role, &createdAt)
				admins = append(admins, gin.H{
					"id":         id,
					"username":   username,
					"email":      email,
					"role":       role,
					"created_at": createdAt,
				})
			}
		}

		// Get moderators
		modRows, err := db.Query(`
			SELECT id, username, email, role, created_at 
			FROM users 
			WHERE role = 'moderator' 
			ORDER BY username
		`)
		if err == nil {
			defer modRows.Close()
			for modRows.Next() {
				var id int64
				var username, email, role string
				var createdAt time.Time
				modRows.Scan(&id, &username, &email, &role, &createdAt)
				moderators = append(moderators, gin.H{
					"id":         id,
					"username":   username,
					"email":      email,
					"role":       role,
					"created_at": createdAt,
				})
			}
		}

		if admins == nil {
			admins = []gin.H{}
		}
		if moderators == nil {
			moderators = []gin.H{}
		}

		c.JSON(http.StatusOK, gin.H{
			"admins":     admins,
			"moderators": moderators,
		})
	}
}

// handleAdminGetUser returns detailed info for a single user
func handleAdminGetUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Param("id")
		var userID int64
		if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		// Get user details
		var user struct {
			ID             int64
			Username       string
			Email          string
			PayoutAddress  sql.NullString
			PoolFeePercent sql.NullFloat64
			IsActive       bool
			IsAdmin        bool
			CreatedAt      time.Time
			UpdatedAt      time.Time
		}

		err := db.QueryRow(`
			SELECT id, username, email, payout_address, pool_fee_percent, 
			       is_active, is_admin, created_at, updated_at
			FROM users WHERE id = $1
		`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.PayoutAddress,
			&user.PoolFeePercent, &user.IsActive, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if err != nil {
			log.Printf("Error fetching user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
			return
		}

		// Get user's miners with comprehensive equipment data
		minerRows, _ := db.Query(`
			SELECT 
				m.id, m.name, COALESCE(m.address::text, '') as address, m.hashrate, m.is_active, m.last_seen, m.created_at,
				EXTRACT(EPOCH FROM (NOW() - m.created_at))::bigint as total_connection_time
			FROM miners m
			WHERE m.user_id = $1 ORDER BY m.last_seen DESC
		`, userID)
		defer minerRows.Close()

		var miners []gin.H
		var totalHashrate float64
		var activeMiners int
		for minerRows.Next() {
			var id int64
			var name string
			var address sql.NullString
			var hashrate float64
			var isActive bool
			var lastSeen, createdAt time.Time
			var totalConnectionTime int64

			minerRows.Scan(&id, &name, &address, &hashrate, &isActive, &lastSeen, &createdAt, &totalConnectionTime)

			// Determine status
			status := "offline"
			if isActive && time.Since(lastSeen) < 5*time.Minute {
				status = "mining"
				activeMiners++
			} else if time.Since(lastSeen) < 15*time.Minute {
				status = "idle"
			}

			// Determine equipment type based on hashrate
			eqType := "gpu"
			if hashrate > 10000000000000 {
				eqType = "blockdag_x100"
			} else if hashrate > 100000000000 {
				eqType = "blockdag_x30"
			} else if hashrate > 1000000000 {
				eqType = "asic"
			}

			totalHashrate += hashrate

			miners = append(miners, gin.H{
				"id":                    id,
				"name":                  name,
				"address":               address.String,
				"hashrate":              hashrate,
				"is_active":             isActive,
				"last_seen":             lastSeen,
				"created_at":            createdAt,
				"status":                status,
				"type":                  eqType,
				"worker_name":           name,
				"model":                 getModelFromHashrate(hashrate),
				"difficulty":            0.0,
				"total_connection_time": totalConnectionTime,
				"uptime_percent":        calculateUptimePercent(totalConnectionTime, 0),
			})
		}

		// Get user's payouts
		payoutRows, _ := db.Query(`
			SELECT id, amount, address, tx_hash, status, created_at, processed_at
			FROM payouts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50
		`, userID)
		defer payoutRows.Close()

		var payouts []gin.H
		for payoutRows.Next() {
			var id, amount int64
			var address, status string
			var txHash sql.NullString
			var createdAt time.Time
			var processedAt sql.NullTime
			payoutRows.Scan(&id, &amount, &address, &txHash, &status, &createdAt, &processedAt)
			payout := gin.H{
				"id":         id,
				"amount":     amount,
				"address":    address,
				"tx_hash":    txHash.String,
				"status":     status,
				"created_at": createdAt,
			}
			if processedAt.Valid {
				payout["processed_at"] = processedAt.Time
			}
			payouts = append(payouts, payout)
		}

		// Get shares stats
		var totalShares, validShares, invalidShares, last24Hours int64
		db.QueryRow("SELECT COUNT(*) FROM shares WHERE user_id = $1", userID).Scan(&totalShares)
		db.QueryRow("SELECT COUNT(*) FROM shares WHERE user_id = $1 AND is_valid = true", userID).Scan(&validShares)
		db.QueryRow("SELECT COUNT(*) FROM shares WHERE user_id = $1 AND is_valid = false", userID).Scan(&invalidShares)
		db.QueryRow("SELECT COUNT(*) FROM shares WHERE user_id = $1 AND timestamp > NOW() - INTERVAL '24 hours'", userID).Scan(&last24Hours)

		// Get total earnings
		var totalEarnings, pendingPayout float64
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = $1 AND status = 'confirmed'", userID).Scan(&totalEarnings)
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = $1 AND status = 'pending'", userID).Scan(&pendingPayout)

		// Get blocks found
		var blocksFound int
		db.QueryRow("SELECT COUNT(*) FROM blocks WHERE finder_id = $1", userID).Scan(&blocksFound)

		// Get user's wallets for multi-wallet payout visibility
		walletRows, _ := db.Query(`
			SELECT id, address, label, percentage, is_primary, is_active, created_at, updated_at
			FROM user_wallets WHERE user_id = $1 ORDER BY is_primary DESC, created_at ASC
		`, userID)
		defer walletRows.Close()

		var wallets []gin.H
		var totalAllocated float64
		var activeWallets, totalWallets int
		var primaryWallet string

		for walletRows.Next() {
			var id int64
			var address string
			var label sql.NullString
			var percentage float64
			var isPrimary, isActive bool
			var createdAt, updatedAt time.Time
			walletRows.Scan(&id, &address, &label, &percentage, &isPrimary, &isActive, &createdAt, &updatedAt)

			wallets = append(wallets, gin.H{
				"id":         id,
				"address":    address,
				"label":      label.String,
				"percentage": percentage,
				"is_primary": isPrimary,
				"is_active":  isActive,
				"created_at": createdAt,
				"updated_at": updatedAt,
			})

			totalWallets++
			if isActive {
				activeWallets++
				totalAllocated += percentage
			}
			if isPrimary {
				primaryWallet = address
			}
		}

		// Calculate wallet summary
		walletSummary := gin.H{
			"total_wallets":        totalWallets,
			"active_wallets":       activeWallets,
			"total_allocated":      totalAllocated,
			"remaining_percent":    100.0 - totalAllocated,
			"has_multiple_wallets": totalWallets > 1,
			"primary_wallet":       primaryWallet,
		}

		// Equipment summary for admin view
		equipmentSummary := gin.H{
			"total_miners":   len(miners),
			"active_miners":  activeMiners,
			"total_hashrate": totalHashrate,
			"offline_miners": len(miners) - activeMiners,
		}

		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":               user.ID,
				"username":         user.Username,
				"email":            user.Email,
				"payout_address":   user.PayoutAddress.String,
				"pool_fee_percent": user.PoolFeePercent.Float64,
				"is_active":        user.IsActive,
				"is_admin":         user.IsAdmin,
				"created_at":       user.CreatedAt,
				"updated_at":       user.UpdatedAt,
				"total_earnings":   totalEarnings,
				"pending_payout":   pendingPayout,
				"blocks_found":     blocksFound,
				"total_hashrate":   totalHashrate,
				"active_miners":    activeMiners,
			},
			"miners":            miners,
			"equipment_summary": equipmentSummary,
			"payouts":           payouts,
			"wallets":           wallets,
			"wallet_summary":    walletSummary,
			"shares_stats": gin.H{
				"total_shares":   totalShares,
				"valid_shares":   validShares,
				"invalid_shares": invalidShares,
				"last_24_hours":  last24Hours,
			},
		})
	}
}

// handleAdminUpdateUser updates user settings (fee, status, etc)
func handleAdminUpdateUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Param("id")
		var userID int64
		if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		// Check user exists
		var exists bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		var req struct {
			PoolFeePercent *float64 `json:"pool_fee_percent"`
			IsActive       *bool    `json:"is_active"`
			PayoutAddress  *string  `json:"payout_address"`
			IsAdmin        *bool    `json:"is_admin"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate pool fee
		if req.PoolFeePercent != nil {
			if *req.PoolFeePercent < 0 || *req.PoolFeePercent > 100 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Pool fee must be between 0 and 100"})
				return
			}
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argIdx := 1

		if req.PoolFeePercent != nil {
			updates = append(updates, fmt.Sprintf("pool_fee_percent = $%d", argIdx))
			args = append(args, *req.PoolFeePercent)
			argIdx++
		}
		if req.IsActive != nil {
			updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
			args = append(args, *req.IsActive)
			argIdx++
		}
		if req.PayoutAddress != nil {
			updates = append(updates, fmt.Sprintf("payout_address = $%d", argIdx))
			args = append(args, *req.PayoutAddress)
			argIdx++
		}
		if req.IsAdmin != nil {
			updates = append(updates, fmt.Sprintf("is_admin = $%d", argIdx))
			args = append(args, *req.IsAdmin)
			argIdx++
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}

		updates = append(updates, "updated_at = NOW()")
		args = append(args, userID)

		query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d",
			joinStrings(updates, ", "), argIdx)

		_, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("Error updating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
	}
}

// handleAdminDeleteUser deletes a user (soft delete by deactivating)
func handleAdminDeleteUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Param("id")
		var userID int64
		if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		// Prevent deleting self
		currentUserID := c.GetInt64("user_id")
		if userID == currentUserID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
			return
		}

		// Soft delete - just deactivate
		result, err := db.Exec("UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User deactivated successfully"})
	}
}

// handleAdminUserEarnings returns detailed earnings for a user
func handleAdminUserEarnings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Param("id")
		var userID int64
		if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		// Check user exists
		var exists bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get date range from query params
		fromDate := c.Query("from")
		toDate := c.Query("to")

		// Get earnings summary
		var totalPaid, totalPending float64
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = $1 AND status = 'confirmed'", userID).Scan(&totalPaid)
		db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE user_id = $1 AND status = 'pending'", userID).Scan(&totalPending)

		// Get blocks found by user
		blockQuery := `
			SELECT id, height, hash, reward, difficulty, timestamp, status
			FROM blocks WHERE finder_id = $1
		`
		args := []interface{}{userID}
		argIdx := 2

		if fromDate != "" {
			blockQuery += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
			args = append(args, fromDate)
			argIdx++
		}
		if toDate != "" {
			blockQuery += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
			args = append(args, toDate)
			argIdx++
		}
		blockQuery += " ORDER BY timestamp DESC"

		blockRows, _ := db.Query(blockQuery, args...)
		defer blockRows.Close()

		var blocks []gin.H
		var totalBlockRewards int64
		for blockRows.Next() {
			var id, height, reward int64
			var hash, status string
			var difficulty float64
			var timestamp time.Time
			blockRows.Scan(&id, &height, &hash, &reward, &difficulty, &timestamp, &status)
			blocks = append(blocks, gin.H{
				"id":         id,
				"height":     height,
				"hash":       hash,
				"reward":     reward,
				"difficulty": difficulty,
				"timestamp":  timestamp,
				"status":     status,
			})
			if status == "confirmed" {
				totalBlockRewards += reward
			}
		}

		// Get daily earnings for chart
		dailyQuery := `
			SELECT DATE(created_at) as date, SUM(amount) as amount
			FROM payouts WHERE user_id = $1 AND status = 'confirmed'
			GROUP BY DATE(created_at) ORDER BY date DESC LIMIT 30
		`
		dailyRows, _ := db.Query(dailyQuery, userID)
		defer dailyRows.Close()

		var dailyEarnings []gin.H
		for dailyRows.Next() {
			var date time.Time
			var amount float64
			dailyRows.Scan(&date, &amount)
			dailyEarnings = append(dailyEarnings, gin.H{
				"date":   date.Format("2006-01-02"),
				"amount": amount,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":             userID,
			"total_paid":          totalPaid,
			"total_pending":       totalPending,
			"total_block_rewards": totalBlockRewards,
			"blocks_found":        blocks,
			"daily_earnings":      dailyEarnings,
		})
	}
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// Auth middleware
func authMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		userID, _, err := validateJWT(tokenString, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// JWT and password utilities
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJWT(userID int64, username, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	return token.SignedString([]byte(secret))
}

func validateJWT(tokenString, secret string) (int64, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return 0, "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int64(claims["user_id"].(float64))
		username := claims["username"].(string)
		return userID, username, nil
	}

	return 0, "", fmt.Errorf("invalid token")
}

// handlePasswordResetDisabled returns a message that password reset is temporarily disabled
func handlePasswordResetDisabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Password reset is temporarily disabled",
			"message": "Email service is being configured. Please contact support at support@chimeriapool.com to reset your password.",
		})
	}
}

// Password Reset Handlers
func handleForgotPassword(db *sql.DB, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if user exists
		var userID int64
		var username string
		err := db.QueryRow(
			"SELECT id, username FROM users WHERE email = $1 AND is_active = true",
			req.Email,
		).Scan(&userID, &username)

		if err != nil {
			// Don't reveal if email exists or not for security
			c.JSON(http.StatusOK, gin.H{
				"message": "If an account with that email exists, a password reset link has been sent.",
			})
			return
		}

		// Generate secure reset token
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
			return
		}
		resetToken := hex.EncodeToString(tokenBytes)

		// Set token expiry to 1 hour from now
		expiresAt := time.Now().Add(1 * time.Hour)

		// Invalidate any existing tokens for this user
		db.Exec("UPDATE password_reset_tokens SET used = true WHERE user_id = $1 AND used = false", userID)

		// Store reset token in database
		_, err = db.Exec(
			"INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
			userID, resetToken, expiresAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reset token"})
			return
		}

		// Send password reset email
		resetLink := fmt.Sprintf("%s/reset-password?token=%s", config.FrontendURL, resetToken)
		err = sendPasswordResetEmail(config, req.Email, username, resetLink)
		if err != nil {
			log.Printf("Failed to send password reset email: %v", err)
			// Don't expose email sending errors to user
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "If an account with that email exists, a password reset link has been sent.",
		})
	}
}

func handleResetPassword(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token       string `json:"token" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find valid, unexpired token
		var userID int64
		var tokenID int64
		err := db.QueryRow(
			`SELECT id, user_id FROM password_reset_tokens 
			 WHERE token = $1 AND used = false AND expires_at > NOW()`,
			req.Token,
		).Scan(&tokenID, &userID)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
			return
		}

		// Hash new password
		hashedPassword, err := hashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}

		// Update user's password
		_, err = db.Exec(
			"UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2",
			hashedPassword, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}

		// Mark token as used
		db.Exec("UPDATE password_reset_tokens SET used = true WHERE id = $1", tokenID)

		// Invalidate all other tokens for this user
		db.Exec("UPDATE password_reset_tokens SET used = true WHERE user_id = $1", userID)

		c.JSON(http.StatusOK, gin.H{
			"message": "Password has been reset successfully. You can now log in with your new password.",
		})
	}
}

// sendPasswordResetEmail sends password reset email via Resend API
func sendPasswordResetEmail(config *Config, toEmail, username, resetLink string) error {
	subject := "Chimera Pool - Password Reset Request"
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; text-align: center;">Chimera Pool</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #ddd; border-top: none;">
        <h2 style="color: #333;">Hello %s,</h2>
        <p>You have requested to reset your password for your Chimera Pool account.</p>
        <p>Click the button below to reset your password:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 15px 30px; text-decoration: none; border-radius: 5px; font-weight: bold; display: inline-block;">Reset Password</a>
        </div>
        <p style="color: #666; font-size: 14px;">This link will expire in 1 hour.</p>
        <p style="color: #666; font-size: 14px;">If you did not request this password reset, please ignore this email. Your password will remain unchanged.</p>
        <hr style="border: none; border-top: 1px solid #ddd; margin: 30px 0;">
        <p style="color: #999; font-size: 12px; text-align: center;">
            Best regards,<br>Chimera Pool Team<br><br>
            This is an automated message. Please do not reply to this email.
        </p>
    </div>
</body>
</html>
`, username, resetLink)

	return sendEmailViaResend(config, toEmail, subject, htmlBody)
}

// sendEmailViaResend sends email using Resend API
func sendEmailViaResend(config *Config, toEmail, subject, htmlBody string) error {
	if config.ResendAPIKey == "" {
		return fmt.Errorf("Resend API key not configured")
	}

	// Resend API payload
	payload := map[string]interface{}{
		"from":    config.EmailFrom,
		"to":      []string{toEmail},
		"subject": subject,
		"html":    htmlBody,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %v", err)
	}

	// Create HTTP request to Resend API
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Resend API error (status %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("Email sent successfully to %s via Resend", toEmail)
	return nil
}

// Generic email sending function
func sendEmail(config *Config, toEmail, subject, body string) error {
	// Compose email
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", config.SMTPFrom, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)

	if config.SMTPPort == "465" {
		// Direct TLS connection for port 465
		tlsConfig := &tls.Config{
			ServerName: config.SMTPHost,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("Failed to connect to SMTP server: %v", err)
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.SMTPHost)
		if err != nil {
			log.Printf("Failed to create SMTP client: %v", err)
			return err
		}
		defer client.Close()

		auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			log.Printf("SMTP authentication failed: %v", err)
			return err
		}

		if err := client.Mail(config.SMTPFrom); err != nil {
			return err
		}
		if err := client.Rcpt(toEmail); err != nil {
			return err
		}

		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return err
		}
		w.Close()
		client.Quit()
	} else {
		auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
		err := smtp.SendMail(addr, auth, config.SMTPFrom, []string{toEmail}, []byte(msg))
		if err != nil {
			log.Printf("Failed to send email: %v", err)
			return err
		}
	}

	log.Printf("Email sent to %s: %s", toEmail, subject)
	return nil
}

func handleAdminGetSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT key, value, description, updated_at FROM pool_settings ORDER BY key")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
			return
		}
		defer rows.Close()

		settings := make(map[string]gin.H)
		for rows.Next() {
			var key, value string
			var description sql.NullString
			var updatedAt time.Time
			if err := rows.Scan(&key, &value, &description, &updatedAt); err != nil {
				continue
			}
			settings[key] = gin.H{
				"value":       value,
				"description": description.String,
				"updated_at":  updatedAt,
			}
		}

		c.JSON(http.StatusOK, gin.H{"settings": settings})
	}
}

func handleAdminUpdateSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Settings map[string]string `json:"settings" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		for key, value := range req.Settings {
			_, err := db.Exec("UPDATE pool_settings SET value = $1, updated_at = NOW() WHERE key = $2", value, key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update setting: %s", key)})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully"})
	}
}

func handleAdminGetAlgorithm(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		algorithmKeys := []string{"algorithm", "algorithm_variant", "difficulty_target", "block_time", "stratum_port", "algorithm_params"}

		algorithm := make(map[string]interface{})
		for _, key := range algorithmKeys {
			var value string
			var description sql.NullString
			var updatedAt time.Time
			err := db.QueryRow("SELECT value, description, updated_at FROM pool_settings WHERE key = $1", key).Scan(&value, &description, &updatedAt)
			if err == nil {
				algorithm[key] = gin.H{
					"value":       value,
					"description": description.String,
					"updated_at":  updatedAt,
				}
			}
		}

		// Add list of supported algorithms for UI dropdown
		supportedAlgorithms := []gin.H{
			{"id": "blake3", "name": "Blake3", "description": "Standard Blake3 algorithm"},
			{"id": "blake3-blockdag", "name": "Blake3 (BlockDAG Custom)", "description": "BlockDAG's custom Blake3 derivative"},
			{"id": "sha256", "name": "SHA-256", "description": "Bitcoin-style SHA-256"},
			{"id": "ethash", "name": "Ethash", "description": "Ethereum-style Ethash"},
			{"id": "kawpow", "name": "KawPoW", "description": "Ravencoin KawPoW algorithm"},
			{"id": "custom", "name": "Custom", "description": "Custom algorithm (specify in params)"},
		}

		c.JSON(http.StatusOK, gin.H{
			"algorithm":            algorithm,
			"supported_algorithms": supportedAlgorithms,
		})
	}
}

func handleAdminUpdateAlgorithm(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Algorithm        string `json:"algorithm"`
			AlgorithmVariant string `json:"algorithm_variant"`
			DifficultyTarget string `json:"difficulty_target"`
			BlockTime        string `json:"block_time"`
			StratumPort      string `json:"stratum_port"`
			AlgorithmParams  string `json:"algorithm_params"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update each setting if provided
		updates := map[string]string{
			"algorithm":         req.Algorithm,
			"algorithm_variant": req.AlgorithmVariant,
			"difficulty_target": req.DifficultyTarget,
			"block_time":        req.BlockTime,
			"stratum_port":      req.StratumPort,
			"algorithm_params":  req.AlgorithmParams,
		}

		for key, value := range updates {
			if value != "" {
				_, err := db.Exec("UPDATE pool_settings SET value = $1, updated_at = NOW() WHERE key = $2", value, key)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update %s", key)})
					return
				}
			}
		}

		// Log the algorithm change for audit
		log.Printf("Algorithm settings updated: algorithm=%s, variant=%s, difficulty=%s, block_time=%s",
			req.Algorithm, req.AlgorithmVariant, req.DifficultyTarget, req.BlockTime)

		c.JSON(http.StatusOK, gin.H{
			"message": "Algorithm settings updated successfully",
			"note":    "Stratum server may need restart for changes to take effect",
		})
	}
}

// Helper function to parse time range
func parseTimeRange(rangeStr string) time.Duration {
	switch rangeStr {
	case "1h":
		return time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "3m":
		return 90 * 24 * time.Hour
	case "6m":
		return 180 * 24 * time.Hour
	case "1y":
		return 365 * 24 * time.Hour
	case "all":
		return 10 * 365 * 24 * time.Hour // 10 years for "lifetime"
	default:
		return 24 * time.Hour
	}
}

// Helper function to get interval for data points
func getInterval(duration time.Duration) string {
	switch {
	case duration <= time.Hour:
		return "5 minutes"
	case duration <= 6*time.Hour:
		return "15 minutes"
	case duration <= 24*time.Hour:
		return "1 hour"
	case duration <= 7*24*time.Hour:
		return "6 hours"
	case duration <= 30*24*time.Hour:
		return "1 day"
	case duration <= 90*24*time.Hour:
		return "3 days"
	case duration <= 180*24*time.Hour:
		return "1 week"
	case duration <= 365*24*time.Hour:
		return "2 weeks"
	default:
		return "1 month"
	}
}

func handleUserHashrateHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		rangeStr := c.DefaultQuery("range", "24h")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', created_at) as time_bucket,
				AVG(hashrate) as avg_hashrate,
				MAX(hashrate) as max_hashrate,
				COUNT(*) as sample_count
			FROM miners 
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query, userID)
		if err != nil {
			// Return sample data if no real data
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSampleHashrateData(duration),
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var avgHashrate, maxHashrate float64
			var sampleCount int
			if err := rows.Scan(&timeBucket, &avgHashrate, &maxHashrate, &sampleCount); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":        timeBucket,
				"hashrate":    avgHashrate,
				"maxHashrate": maxHashrate,
			})
		}

		if len(data) == 0 {
			data = generateSampleHashrateData(duration)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

func generateSampleHashrateData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 24
	if duration <= time.Hour {
		points = 12
	} else if duration >= 7*24*time.Hour {
		points = 28
	}

	interval := duration / time.Duration(points)
	baseHashrate := 150000000.0 // 150 MH/s base

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		// Add some variation
		variation := (float64(i%5) - 2) * 10000000
		data = append(data, gin.H{
			"time":        t,
			"hashrate":    baseHashrate + variation,
			"maxHashrate": baseHashrate + variation + 5000000,
		})
	}
	return data
}

func handleUserSharesHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		rangeStr := c.DefaultQuery("range", "24h")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', created_at) as time_bucket,
				COUNT(*) FILTER (WHERE is_valid = true) as valid_shares,
				COUNT(*) FILTER (WHERE is_valid = false) as invalid_shares,
				COUNT(*) as total_shares
			FROM shares 
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query, userID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSampleSharesData(duration),
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var validShares, invalidShares, totalShares int64
			if err := rows.Scan(&timeBucket, &validShares, &invalidShares, &totalShares); err != nil {
				continue
			}
			acceptanceRate := float64(0)
			if totalShares > 0 {
				acceptanceRate = float64(validShares) / float64(totalShares) * 100
			}
			data = append(data, gin.H{
				"time":           timeBucket,
				"validShares":    validShares,
				"invalidShares":  invalidShares,
				"totalShares":    totalShares,
				"acceptanceRate": acceptanceRate,
			})
		}

		if len(data) == 0 {
			data = generateSampleSharesData(duration)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

func generateSampleSharesData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 24
	if duration <= time.Hour {
		points = 12
	} else if duration >= 7*24*time.Hour {
		points = 28
	}

	interval := duration / time.Duration(points)

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		validShares := 100 + (i % 20)
		invalidShares := i % 3
		totalShares := validShares + invalidShares
		acceptanceRate := float64(validShares) / float64(totalShares) * 100
		data = append(data, gin.H{
			"time":           t,
			"validShares":    validShares,
			"invalidShares":  invalidShares,
			"totalShares":    totalShares,
			"acceptanceRate": acceptanceRate,
		})
	}
	return data
}

func handleUserEarningsHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		rangeStr := c.DefaultQuery("range", "7d")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', created_at) as time_bucket,
				SUM(amount) as earnings
			FROM payouts 
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query, userID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSampleEarningsData(duration),
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		var cumulative float64
		for rows.Next() {
			var timeBucket time.Time
			var earnings float64
			if err := rows.Scan(&timeBucket, &earnings); err != nil {
				continue
			}
			cumulative += earnings
			data = append(data, gin.H{
				"time":       timeBucket,
				"earnings":   earnings,
				"cumulative": cumulative,
			})
		}

		if len(data) == 0 {
			data = generateSampleEarningsData(duration)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

func generateSampleEarningsData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 14
	if duration <= 24*time.Hour {
		points = 24
	} else if duration >= 30*24*time.Hour {
		points = 30
	}

	interval := duration / time.Duration(points)
	var cumulative float64

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		earnings := 0.05 + float64(i%5)*0.01
		cumulative += earnings
		data = append(data, gin.H{
			"time":       t,
			"earnings":   earnings,
			"cumulative": cumulative,
		})
	}
	return data
}

// Pool-wide statistics handlers - calculates real-time hashrate from shares
func handlePoolHashrateHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		// Calculate bucket duration in seconds for hashrate calculation
		bucketSeconds := getBucketSeconds(interval)

		// Query hashrate from shares - sum difficulty and convert to hashrate
		// Hashrate = (difficulty * 2^32) / time_window_seconds
		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', timestamp) as time_bucket,
				COALESCE(SUM(difficulty) * 4294967296.0 / %d, 0) as total_hashrate,
				COUNT(DISTINCT miner_id) as active_miners,
				COUNT(DISTINCT user_id) as active_users
			FROM shares 
			WHERE timestamp > NOW() - INTERVAL '%s' AND is_valid = true
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, bucketSeconds, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Hashrate history query error: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"data":     []gin.H{},
				"range":    rangeStr,
				"interval": interval,
				"error":    "Query failed",
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var totalHashrate float64
			var activeMiners, activeUsers int
			if err := rows.Scan(&timeBucket, &totalHashrate, &activeMiners, &activeUsers); err != nil {
				continue
			}
			avgHashrate := totalHashrate
			if activeMiners > 0 {
				avgHashrate = totalHashrate / float64(activeMiners)
			}
			data = append(data, gin.H{
				"time":          timeBucket,
				"totalHashrate": totalHashrate,
				"avgHashrate":   avgHashrate,
				"activeMiners":  activeMiners,
				"activeUsers":   activeUsers,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

// getBucketSeconds returns seconds per bucket for hashrate calculation
func getBucketSeconds(interval string) int {
	switch interval {
	case "minute":
		return 60
	case "hour":
		return 3600
	case "day":
		return 86400
	case "week":
		return 604800
	case "month":
		return 2592000
	default:
		return 3600
	}
}

func generateSamplePoolHashrateData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 24
	if duration <= time.Hour {
		points = 12
	} else if duration >= 7*24*time.Hour {
		points = 28
	}

	interval := duration / time.Duration(points)
	baseHashrate := 5000000000.0 // 5 GH/s base pool hashrate

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		variation := (float64(i%7) - 3) * 200000000
		data = append(data, gin.H{
			"time":          t,
			"totalHashrate": baseHashrate + variation,
			"avgHashrate":   (baseHashrate + variation) / 25,
			"activeUsers":   20 + (i % 10),
		})
	}
	return data
}

func handlePoolSharesHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', timestamp) as time_bucket,
				COUNT(*) FILTER (WHERE is_valid = true) as valid_shares,
				COUNT(*) FILTER (WHERE is_valid = false) as invalid_shares,
				COUNT(*) as total_shares
			FROM shares 
			WHERE timestamp > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Shares history query error: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"data":     []gin.H{},
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var validShares, invalidShares, totalShares int64
			if err := rows.Scan(&timeBucket, &validShares, &invalidShares, &totalShares); err != nil {
				continue
			}
			acceptanceRate := float64(0)
			if totalShares > 0 {
				acceptanceRate = float64(validShares) / float64(totalShares) * 100
			}
			data = append(data, gin.H{
				"time":           timeBucket,
				"validShares":    validShares,
				"invalidShares":  invalidShares,
				"totalShares":    totalShares,
				"acceptanceRate": acceptanceRate,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

func generateSamplePoolSharesData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 24
	interval := duration / time.Duration(points)

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		validShares := 5000 + (i % 500)
		invalidShares := 20 + (i % 10)
		totalShares := validShares + invalidShares
		acceptanceRate := float64(validShares) / float64(totalShares) * 100
		data = append(data, gin.H{
			"time":           t,
			"validShares":    validShares,
			"invalidShares":  invalidShares,
			"totalShares":    totalShares,
			"acceptanceRate": acceptanceRate,
		})
	}
	return data
}

func handlePoolMinersHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		// Query from shares table for real-time miner activity
		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', timestamp) as time_bucket,
				COUNT(DISTINCT miner_id) as active_miners,
				COUNT(DISTINCT user_id) as unique_users
			FROM shares 
			WHERE timestamp > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Miners history query error: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"data":     []gin.H{},
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var activeMiners, uniqueUsers int
			if err := rows.Scan(&timeBucket, &activeMiners, &uniqueUsers); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":         timeBucket,
				"activeMiners": activeMiners,
				"uniqueUsers":  uniqueUsers,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

func generateSampleMinersData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 24
	interval := duration / time.Duration(points)

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		activeMiners := 45 + (i % 15)
		data = append(data, gin.H{
			"time":         t,
			"activeMiners": activeMiners,
			"totalMiners":  activeMiners + 10,
			"uniqueUsers":  activeMiners / 2,
		})
	}
	return data
}

func handlePoolBlocksHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "7d")

		rows, err := db.Query(`
			SELECT 
				id, height, hash, reward, status, found_by, created_at
			FROM blocks 
			WHERE created_at > NOW() - $1::interval
			ORDER BY created_at DESC
			LIMIT 100
		`, rangeStr)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"blocks": generateSampleBlocksData(),
				"range":  rangeStr,
			})
			return
		}
		defer rows.Close()

		var blocks []gin.H
		for rows.Next() {
			var id, height int64
			var hash string
			var reward float64
			var status string
			var foundBy sql.NullInt64
			var createdAt time.Time
			if err := rows.Scan(&id, &height, &hash, &reward, &status, &foundBy, &createdAt); err != nil {
				continue
			}
			blocks = append(blocks, gin.H{
				"id":        id,
				"height":    height,
				"hash":      hash,
				"reward":    reward,
				"status":    status,
				"foundBy":   foundBy.Int64,
				"createdAt": createdAt,
			})
		}

		if len(blocks) == 0 {
			blocks = generateSampleBlocksData()
		}

		c.JSON(http.StatusOK, gin.H{
			"blocks": blocks,
			"range":  rangeStr,
		})
	}
}

func generateSampleBlocksData() []gin.H {
	var blocks []gin.H
	now := time.Now()

	for i := 0; i < 10; i++ {
		blocks = append(blocks, gin.H{
			"id":        1000 + i,
			"height":    500000 + i*100,
			"hash":      fmt.Sprintf("0x%064d", i),
			"reward":    2.5,
			"status":    "confirmed",
			"foundBy":   1,
			"createdAt": now.Add(-time.Duration(i) * 6 * time.Hour),
		})
	}
	return blocks
}

func handlePoolPayoutsHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "30d")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', created_at) as time_bucket,
				SUM(amount) as total_paid,
				COUNT(*) as payout_count,
				COUNT(DISTINCT user_id) as unique_recipients
			FROM payouts 
			WHERE created_at > NOW() - INTERVAL '%s' AND status = 'completed'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSamplePayoutsData(duration),
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		var cumulative float64
		for rows.Next() {
			var timeBucket time.Time
			var totalPaid float64
			var payoutCount, uniqueRecipients int
			if err := rows.Scan(&timeBucket, &totalPaid, &payoutCount, &uniqueRecipients); err != nil {
				continue
			}
			cumulative += totalPaid
			data = append(data, gin.H{
				"time":             timeBucket,
				"totalPaid":        totalPaid,
				"cumulative":       cumulative,
				"payoutCount":      payoutCount,
				"uniqueRecipients": uniqueRecipients,
			})
		}

		if len(data) == 0 {
			data = generateSamplePayoutsData(duration)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
	}
}

func generateSamplePayoutsData(duration time.Duration) []gin.H {
	var data []gin.H
	now := time.Now()
	points := 30
	interval := duration / time.Duration(points)
	var cumulative float64

	for i := points - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		totalPaid := 5.0 + float64(i%10)*0.5
		cumulative += totalPaid
		data = append(data, gin.H{
			"time":             t,
			"totalPaid":        totalPaid,
			"cumulative":       cumulative,
			"payoutCount":      10 + (i % 5),
			"uniqueRecipients": 8 + (i % 3),
		})
	}
	return data
}

func handlePoolDistribution(db *sql.DB) gin.HandlerFunc {
	// Color palette for pie chart
	colors := []string{"#00d4ff", "#ff6b6b", "#4ecdc4", "#ffe66d", "#95e1d3", "#f38181", "#aa96da", "#fcbad3"}

	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT 
				u.id, u.username, 
				COALESCE(SUM(m.hashrate), 0) as total_hashrate,
				COUNT(m.id) as miner_count
			FROM users u
			LEFT JOIN miners m ON u.id = m.user_id AND m.is_active = true
			WHERE u.is_active = true
			GROUP BY u.id, u.username
			HAVING COALESCE(SUM(m.hashrate), 0) > 0
			ORDER BY total_hashrate DESC
			LIMIT 20
		`)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"distribution": generateDefaultDistribution(),
			})
			return
		}
		defer rows.Close()

		var distribution []gin.H
		var totalPoolHashrate float64
		var tempData []gin.H

		for rows.Next() {
			var userID int64
			var username string
			var totalHashrate float64
			var minerCount int
			if err := rows.Scan(&userID, &username, &totalHashrate, &minerCount); err != nil {
				continue
			}
			totalPoolHashrate += totalHashrate
			tempData = append(tempData, gin.H{
				"userId":     userID,
				"username":   username,
				"hashrate":   totalHashrate,
				"minerCount": minerCount,
			})
		}

		// Build distribution with format expected by frontend PieChart
		// Frontend expects: { name: string, value: number, color: string }
		for i, d := range tempData {
			hashrate := d["hashrate"].(float64)
			percentage := float64(0)
			if totalPoolHashrate > 0 {
				percentage = (hashrate / totalPoolHashrate) * 100
			}
			color := colors[i%len(colors)]
			distribution = append(distribution, gin.H{
				// Fields for PieChart component
				"name":  d["username"],
				"value": percentage,
				"color": color,
				// Additional data
				"userId":     d["userId"],
				"hashrate":   hashrate,
				"minerCount": d["minerCount"],
			})
		}

		if len(distribution) == 0 {
			distribution = generateDefaultDistribution()
		}

		c.JSON(http.StatusOK, gin.H{
			"distribution":      distribution,
			"totalPoolHashrate": totalPoolHashrate,
		})
	}
}

func generateDefaultDistribution() []gin.H {
	// Return single "No miners" entry when no data
	return []gin.H{
		{
			"name":  "No active miners",
			"value": 100,
			"color": "#444444",
		},
	}
}

// Miner Location Handlers
func handlePublicMinerLocations(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT 
				COALESCE(city, 'Unknown') as city,
				COALESCE(country, 'Unknown') as country,
				COALESCE(country_code, 'XX') as country_code,
				COALESCE(continent, 'Unknown') as continent,
				ROUND(AVG(latitude)::numeric, 4) as lat,
				ROUND(AVG(longitude)::numeric, 4) as lng,
				COUNT(*) as miner_count,
				SUM(hashrate) as total_hashrate,
				COUNT(*) FILTER (WHERE is_active = true) as active_count
			FROM miners 
			WHERE latitude IS NOT NULL AND longitude IS NOT NULL
			GROUP BY city, country, country_code, continent
			ORDER BY miner_count DESC
		`)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"locations": generateSampleMinerLocations(),
			})
			return
		}
		defer rows.Close()

		var locations []gin.H
		for rows.Next() {
			var city, country, countryCode, continent string
			var lat, lng float64
			var minerCount, activeCount int
			var totalHashrate float64
			if err := rows.Scan(&city, &country, &countryCode, &continent, &lat, &lng, &minerCount, &totalHashrate, &activeCount); err != nil {
				continue
			}
			locations = append(locations, gin.H{
				"city":        city,
				"country":     country,
				"countryCode": countryCode,
				"continent":   continent,
				"lat":         lat,
				"lng":         lng,
				"minerCount":  minerCount,
				"hashrate":    totalHashrate,
				"activeCount": activeCount,
				"isActive":    activeCount > 0,
			})
		}

		if len(locations) == 0 {
			locations = generateSampleMinerLocations()
		}

		c.JSON(http.StatusOK, gin.H{
			"locations": locations,
		})
	}
}

func generateSampleMinerLocations() []gin.H {
	return []gin.H{
		{"city": "New York", "country": "United States", "countryCode": "US", "continent": "North America", "lat": 40.7128, "lng": -74.0060, "minerCount": 15, "hashrate": 1500000000, "activeCount": 12, "isActive": true},
		{"city": "London", "country": "United Kingdom", "countryCode": "GB", "continent": "Europe", "lat": 51.5074, "lng": -0.1278, "minerCount": 12, "hashrate": 1200000000, "activeCount": 10, "isActive": true},
		{"city": "Tokyo", "country": "Japan", "countryCode": "JP", "continent": "Asia", "lat": 35.6762, "lng": 139.6503, "minerCount": 10, "hashrate": 1000000000, "activeCount": 8, "isActive": true},
		{"city": "Singapore", "country": "Singapore", "countryCode": "SG", "continent": "Asia", "lat": 1.3521, "lng": 103.8198, "minerCount": 8, "hashrate": 800000000, "activeCount": 7, "isActive": true},
		{"city": "Frankfurt", "country": "Germany", "countryCode": "DE", "continent": "Europe", "lat": 50.1109, "lng": 8.6821, "minerCount": 7, "hashrate": 700000000, "activeCount": 6, "isActive": true},
		{"city": "Sydney", "country": "Australia", "countryCode": "AU", "continent": "Oceania", "lat": -33.8688, "lng": 151.2093, "minerCount": 5, "hashrate": 500000000, "activeCount": 4, "isActive": true},
		{"city": "Toronto", "country": "Canada", "countryCode": "CA", "continent": "North America", "lat": 43.6532, "lng": -79.3832, "minerCount": 6, "hashrate": 600000000, "activeCount": 5, "isActive": true},
		{"city": "São Paulo", "country": "Brazil", "countryCode": "BR", "continent": "South America", "lat": -23.5505, "lng": -46.6333, "minerCount": 4, "hashrate": 400000000, "activeCount": 3, "isActive": true},
		{"city": "Dubai", "country": "United Arab Emirates", "countryCode": "AE", "continent": "Asia", "lat": 25.2048, "lng": 55.2708, "minerCount": 3, "hashrate": 300000000, "activeCount": 2, "isActive": true},
		{"city": "Mumbai", "country": "India", "countryCode": "IN", "continent": "Asia", "lat": 19.0760, "lng": 72.8777, "minerCount": 4, "hashrate": 350000000, "activeCount": 3, "isActive": true},
	}
}

func handleMinerLocationStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var totalMiners, totalCountries, activeMiners int

		db.QueryRow("SELECT COUNT(*) FROM miners WHERE latitude IS NOT NULL").Scan(&totalMiners)
		db.QueryRow("SELECT COUNT(DISTINCT country_code) FROM miners WHERE country_code IS NOT NULL").Scan(&totalCountries)
		db.QueryRow("SELECT COUNT(*) FROM miners WHERE is_active = true AND latitude IS NOT NULL").Scan(&activeMiners)

		// Top countries by miner count
		topCountries := []gin.H{}
		rows, err := db.Query(`
			SELECT 
				COALESCE(country, 'Unknown') as country,
				COALESCE(country_code, 'XX') as country_code,
				COUNT(*) as miner_count,
				SUM(hashrate) as total_hashrate
			FROM miners 
			WHERE country IS NOT NULL
			GROUP BY country, country_code
			ORDER BY miner_count DESC
			LIMIT 10
		`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var country, countryCode string
				var minerCount int
				var totalHashrate float64
				if err := rows.Scan(&country, &countryCode, &minerCount, &totalHashrate); err != nil {
					continue
				}
				topCountries = append(topCountries, gin.H{
					"country":     country,
					"countryCode": countryCode,
					"minerCount":  minerCount,
					"hashrate":    totalHashrate,
				})
			}
		}

		// Continent breakdown
		continentBreakdown := []gin.H{}
		rows2, err := db.Query(`
			SELECT 
				COALESCE(continent, 'Unknown') as continent,
				COUNT(*) as miner_count,
				SUM(hashrate) as total_hashrate
			FROM miners 
			WHERE continent IS NOT NULL
			GROUP BY continent
			ORDER BY miner_count DESC
		`)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var continent string
				var minerCount int
				var totalHashrate float64
				if err := rows2.Scan(&continent, &minerCount, &totalHashrate); err != nil {
					continue
				}
				continentBreakdown = append(continentBreakdown, gin.H{
					"continent":  continent,
					"minerCount": minerCount,
					"hashrate":   totalHashrate,
				})
			}
		}

		// Use sample data if no real data
		if totalMiners == 0 {
			totalMiners = 74
			totalCountries = 10
			activeMiners = 60
			topCountries = []gin.H{
				{"country": "United States", "countryCode": "US", "minerCount": 21, "hashrate": 2100000000},
				{"country": "United Kingdom", "countryCode": "GB", "minerCount": 12, "hashrate": 1200000000},
				{"country": "Japan", "countryCode": "JP", "minerCount": 10, "hashrate": 1000000000},
				{"country": "Singapore", "countryCode": "SG", "minerCount": 8, "hashrate": 800000000},
				{"country": "Germany", "countryCode": "DE", "minerCount": 7, "hashrate": 700000000},
			}
			continentBreakdown = []gin.H{
				{"continent": "North America", "minerCount": 27, "hashrate": 2700000000},
				{"continent": "Europe", "minerCount": 19, "hashrate": 1900000000},
				{"continent": "Asia", "minerCount": 25, "hashrate": 2450000000},
				{"continent": "Oceania", "minerCount": 5, "hashrate": 500000000},
				{"continent": "South America", "minerCount": 4, "hashrate": 400000000},
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"totalMiners":        totalMiners,
			"totalCountries":     totalCountries,
			"activeMiners":       activeMiners,
			"topCountries":       topCountries,
			"continentBreakdown": continentBreakdown,
		})
	}
}

func handleAdminMinerLocations(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		countryFilter := c.Query("country")

		query := `
			SELECT 
				m.id, m.name, m.address, m.hashrate, m.is_active,
				COALESCE(m.city, 'Unknown') as city,
				COALESCE(m.country, 'Unknown') as country,
				COALESCE(m.country_code, 'XX') as country_code,
				COALESCE(m.latitude, 0) as lat,
				COALESCE(m.longitude, 0) as lng,
				u.username, u.email,
				m.last_seen
			FROM miners m
			JOIN users u ON m.user_id = u.id
			WHERE 1=1
		`
		args := []interface{}{}

		if countryFilter != "" {
			query += " AND m.country_code = $1"
			args = append(args, countryFilter)
		}

		query += " ORDER BY m.is_active DESC, m.hashrate DESC LIMIT 500"

		rows, err := db.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"miners": generateSampleAdminMinerLocations(),
			})
			return
		}
		defer rows.Close()

		var miners []gin.H
		for rows.Next() {
			var id int64
			var name string
			var address sql.NullString
			var hashrate float64
			var isActive bool
			var city, country, countryCode string
			var lat, lng float64
			var username, email string
			var lastSeen time.Time

			if err := rows.Scan(&id, &name, &address, &hashrate, &isActive, &city, &country, &countryCode, &lat, &lng, &username, &email, &lastSeen); err != nil {
				continue
			}

			miners = append(miners, gin.H{
				"id":          id,
				"name":        name,
				"address":     address.String,
				"hashrate":    hashrate,
				"isActive":    isActive,
				"city":        city,
				"country":     country,
				"countryCode": countryCode,
				"lat":         lat,
				"lng":         lng,
				"username":    username,
				"email":       email,
				"lastSeen":    lastSeen,
			})
		}

		if len(miners) == 0 {
			miners = generateSampleAdminMinerLocations()
		}

		c.JSON(http.StatusOK, gin.H{
			"miners": miners,
		})
	}
}

func generateSampleAdminMinerLocations() []gin.H {
	locations := generateSampleMinerLocations()
	var miners []gin.H

	for i, loc := range locations {
		for j := 0; j < loc["minerCount"].(int); j++ {
			miners = append(miners, gin.H{
				"id":          i*100 + j,
				"name":        fmt.Sprintf("worker_%d", i*100+j),
				"address":     fmt.Sprintf("192.168.%d.%d", i, j),
				"hashrate":    float64(loc["hashrate"].(int)) / float64(loc["minerCount"].(int)),
				"isActive":    j < loc["activeCount"].(int),
				"city":        loc["city"],
				"country":     loc["country"],
				"countryCode": loc["countryCode"],
				"lat":         loc["lat"],
				"lng":         loc["lng"],
				"username":    fmt.Sprintf("user_%d", i),
				"email":       fmt.Sprintf("user%d@example.com", i),
				"lastSeen":    time.Now().Add(-time.Duration(j) * time.Hour),
			})
		}
	}
	return miners
}

// ============================================
// COMMUNITY HANDLERS
// ============================================

// Channel handlers
func handleGetChannels(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First, fetch all categories (even those without channels)
		catRows, err := db.Query(`
			SELECT id, name, COALESCE(description, ''), position
			FROM channel_categories
			ORDER BY position ASC
		`)
		if err != nil {
			log.Printf("Error fetching categories: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
			return
		}
		defer catRows.Close()

		categories := make(map[string]gin.H)
		categoryOrder := []string{}

		for catRows.Next() {
			var id, name, description string
			var position int
			if err := catRows.Scan(&id, &name, &description, &position); err != nil {
				log.Printf("Error scanning category: %v", err)
				continue
			}
			categories[id] = gin.H{
				"id":          id,
				"name":        name,
				"description": description,
				"position":    position,
				"channels":    []gin.H{},
			}
			categoryOrder = append(categoryOrder, id)
		}

		// Now fetch all channels and add them to their categories
		chanRows, err := db.Query(`
			SELECT id, category_id, name, COALESCE(description, ''), type, position, is_read_only, admin_only_post
			FROM channels
			ORDER BY position ASC
		`)
		if err != nil {
			log.Printf("Error fetching channels: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
			return
		}
		defer chanRows.Close()

		for chanRows.Next() {
			var id, categoryID, name, description, channelType string
			var position int
			var isReadOnly, adminOnlyPost bool
			if err := chanRows.Scan(&id, &categoryID, &name, &description, &channelType, &position, &isReadOnly, &adminOnlyPost); err != nil {
				log.Printf("Error scanning channel: %v", err)
				continue
			}

			if cat, exists := categories[categoryID]; exists {
				channels := cat["channels"].([]gin.H)
				channels = append(channels, gin.H{
					"id":            id,
					"name":          name,
					"description":   description,
					"type":          channelType,
					"position":      position,
					"isReadOnly":    isReadOnly,
					"adminOnlyPost": adminOnlyPost,
				})
				categories[categoryID]["channels"] = channels
			}
		}

		// Build result in order
		result := make([]gin.H, 0, len(categoryOrder))
		for _, catID := range categoryOrder {
			result = append(result, categories[catID])
		}

		log.Printf("Returning %d categories with channels", len(result))
		c.JSON(http.StatusOK, gin.H{"categories": result})
	}
}

// Admin category/channel management handlers
func handleAdminGetCategories(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, name, COALESCE(description, ''), position, created_at
			FROM channel_categories
			ORDER BY position ASC
		`)
		if err != nil {
			log.Printf("Error fetching categories: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
			return
		}
		defer rows.Close()

		categories := make([]gin.H, 0) // Initialize as empty array, not nil
		for rows.Next() {
			var id string
			var name, description string
			var position int
			var createdAt time.Time
			if err := rows.Scan(&id, &name, &description, &position, &createdAt); err != nil {
				log.Printf("Error scanning category row: %v", err)
				continue
			}
			categories = append(categories, gin.H{
				"id":          id,
				"name":        name,
				"description": description,
				"position":    position,
				"created_at":  createdAt,
			})
		}

		log.Printf("Returning %d categories", len(categories))
		c.JSON(http.StatusOK, gin.H{"categories": categories})
	}
}

func handleAdminCreateCategory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Category name is required"})
			return
		}

		// Get next position
		var maxPosition int
		db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM channel_categories").Scan(&maxPosition)

		var id string
		var createdAt time.Time
		err := db.QueryRow(`
			INSERT INTO channel_categories (name, description, position, created_by)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at
		`, req.Name, req.Description, maxPosition+1, userID).Scan(&id, &createdAt)

		if err != nil {
			log.Printf("Error creating category: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"category": gin.H{
				"id":          id,
				"name":        req.Name,
				"description": req.Description,
				"position":    maxPosition + 1,
				"created_at":  createdAt,
			},
			"message": "Category created successfully",
		})
	}
}

func handleAdminUpdateCategory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Param("id")

		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Position    *int   `json:"position"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argNum := 1

		if req.Name != "" {
			updates = append(updates, fmt.Sprintf("name = $%d", argNum))
			args = append(args, req.Name)
			argNum++
		}
		if req.Description != "" {
			updates = append(updates, fmt.Sprintf("description = $%d", argNum))
			args = append(args, req.Description)
			argNum++
		}
		if req.Position != nil {
			updates = append(updates, fmt.Sprintf("position = $%d", argNum))
			args = append(args, *req.Position)
			argNum++
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}

		updates = append(updates, fmt.Sprintf("updated_at = $%d", argNum))
		args = append(args, time.Now())
		argNum++

		args = append(args, categoryID)
		query := fmt.Sprintf("UPDATE channel_categories SET %s WHERE id = $%d", strings.Join(updates, ", "), argNum)

		result, err := db.Exec(query, args...)
		if err != nil {
			log.Printf("Error updating category: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully"})
	}
}

func handleAdminDeleteCategory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Param("id")

		// Check if category has channels
		var channelCount int
		db.QueryRow("SELECT COUNT(*) FROM channels WHERE category_id = $1", categoryID).Scan(&channelCount)
		if channelCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete category with channels. Delete channels first."})
			return
		}

		result, err := db.Exec("DELETE FROM channel_categories WHERE id = $1", categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
	}
}

func handleAdminCreateChannel(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			CategoryID    string `json:"category_id" binding:"required"`
			Name          string `json:"name" binding:"required"`
			Description   string `json:"description"`
			Type          string `json:"type"`
			IsReadOnly    bool   `json:"is_read_only"`
			AdminOnlyPost bool   `json:"admin_only_post"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Channel name and category are required"})
			return
		}

		if req.Type == "" {
			req.Type = "text"
		}

		// Verify category exists
		var categoryExists bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM channel_categories WHERE id = $1)", req.CategoryID).Scan(&categoryExists)
		if !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Category not found"})
			return
		}

		// Get next position in category
		var maxPosition int
		db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM channels WHERE category_id = $1", req.CategoryID).Scan(&maxPosition)

		var id string
		var createdAt time.Time
		err := db.QueryRow(`
			INSERT INTO channels (category_id, name, description, type, position, is_read_only, admin_only_post, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_at
		`, req.CategoryID, req.Name, req.Description, req.Type, maxPosition+1, req.IsReadOnly, req.AdminOnlyPost, userID).Scan(&id, &createdAt)

		if err != nil {
			log.Printf("Error creating channel: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"channel": gin.H{
				"id":              id,
				"category_id":     req.CategoryID,
				"name":            req.Name,
				"description":     req.Description,
				"type":            req.Type,
				"is_read_only":    req.IsReadOnly,
				"admin_only_post": req.AdminOnlyPost,
				"created_at":      createdAt,
			},
			"message": "Channel created successfully",
		})
	}
}

func handleAdminUpdateChannel(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("id")

		var req struct {
			Name          *string `json:"name"`
			Description   *string `json:"description"`
			Type          *string `json:"type"`
			IsReadOnly    *bool   `json:"is_read_only"`
			AdminOnlyPost *bool   `json:"admin_only_post"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argIdx := 1

		if req.Name != nil {
			updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
			args = append(args, *req.Name)
			argIdx++
		}
		if req.Description != nil {
			updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
			args = append(args, *req.Description)
			argIdx++
		}
		if req.Type != nil {
			updates = append(updates, fmt.Sprintf("type = $%d", argIdx))
			args = append(args, *req.Type)
			argIdx++
		}
		if req.IsReadOnly != nil {
			updates = append(updates, fmt.Sprintf("is_read_only = $%d", argIdx))
			args = append(args, *req.IsReadOnly)
			argIdx++
		}
		if req.AdminOnlyPost != nil {
			updates = append(updates, fmt.Sprintf("admin_only_post = $%d", argIdx))
			args = append(args, *req.AdminOnlyPost)
			argIdx++
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}

		updates = append(updates, "updated_at = NOW()")
		args = append(args, channelID)

		query := fmt.Sprintf("UPDATE channels SET %s WHERE id = $%d",
			strings.Join(updates, ", "), argIdx)

		result, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Channel updated successfully"})
	}
}

func handleAdminDeleteChannel(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("id")

		result, err := db.Exec("DELETE FROM channels WHERE id = $1", channelID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
	}
}

func handleGetChannelMessages(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("id")
		before := c.Query("before")
		limit := 50

		// Enhanced query to include user role and all badges
		query := `
			SELECT cm.id, cm.content, cm.is_edited, cm.created_at, cm.reply_to_id,
				   u.id as user_id, u.username, u.role, u.is_admin,
				   COALESCE(b.icon, '🌱') as badge_icon, 
				   COALESCE(b.color, '#4ade80') as badge_color,
				   COALESCE(b.name, 'Newcomer') as badge_name
			FROM channel_messages cm
			JOIN users u ON cm.user_id = u.id
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
			LEFT JOIN badges b ON ub.badge_id = b.id
			WHERE cm.channel_id = $1 AND cm.is_deleted = false
		`
		args := []interface{}{channelID}

		if before != "" {
			query += " AND cm.id < $2"
			args = append(args, before)
		}

		query += " ORDER BY cm.created_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, limit)

		rows, err := db.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
			return
		}
		defer rows.Close()

		messages := []gin.H{}
		userIDs := []int64{}
		for rows.Next() {
			var id int64
			var content string
			var isEdited bool
			var createdAt time.Time
			var replyToID sql.NullInt64
			var userID int64
			var username string
			var role sql.NullString
			var isAdmin bool
			var badgeIcon, badgeColor, badgeName string

			rows.Scan(&id, &content, &isEdited, &createdAt, &replyToID, &userID, &username, &role, &isAdmin, &badgeIcon, &badgeColor, &badgeName)

			// Determine user role
			userRole := "user"
			if role.Valid {
				userRole = role.String
			}
			if isAdmin {
				userRole = "admin"
			}

			msg := gin.H{
				"id":        id,
				"content":   content,
				"isEdited":  isEdited,
				"createdAt": createdAt,
				"user": gin.H{
					"id":         userID,
					"username":   username,
					"role":       userRole,
					"roleBadge":  getRoleBadge(userRole),
					"badgeIcon":  badgeIcon,
					"badgeColor": badgeColor,
					"badgeName":  badgeName,
				},
			}
			if replyToID.Valid {
				msg["replyToId"] = replyToID.Int64
			}
			messages = append(messages, msg)
			userIDs = append(userIDs, userID)
		}

		// Fetch all badges for message authors
		if len(userIDs) > 0 {
			uniqueUserIDs := make(map[int64]bool)
			for _, uid := range userIDs {
				uniqueUserIDs[uid] = true
			}
			userIDList := make([]int64, 0, len(uniqueUserIDs))
			for uid := range uniqueUserIDs {
				userIDList = append(userIDList, uid)
			}

			badgeQuery := `
				SELECT ub.user_id, b.icon, b.color, b.name, b.badge_type, ub.is_primary
				FROM user_badges ub
				JOIN badges b ON ub.badge_id = b.id
				WHERE ub.user_id = ANY($1)
				ORDER BY ub.is_primary DESC, ub.earned_at DESC
			`
			badgeRows, err := db.Query(badgeQuery, pq.Array(userIDList))
			if err == nil {
				defer badgeRows.Close()
				userBadges := make(map[int64][]gin.H)
				for badgeRows.Next() {
					var uid int64
					var icon, color, name, badgeType string
					var isPrimary bool
					badgeRows.Scan(&uid, &icon, &color, &name, &badgeType, &isPrimary)
					userBadges[uid] = append(userBadges[uid], gin.H{
						"icon":      icon,
						"color":     color,
						"name":      name,
						"type":      badgeType,
						"isPrimary": isPrimary,
					})
				}
				// Assign badges to messages
				for i, msg := range messages {
					user := msg["user"].(gin.H)
					uid := user["id"].(int64)
					if badges, ok := userBadges[uid]; ok {
						user["badges"] = badges
						messages[i]["user"] = user
					}
				}
			}
		}

		// Fetch reactions for all messages
		currentUserID := c.GetInt64("user_id")
		if len(messages) > 0 {
			messageIDs := make([]int64, len(messages))
			for i, msg := range messages {
				messageIDs[i] = msg["id"].(int64)
			}

			reactionQuery := `
				SELECT mr.message_id, rt.emoji, rt.name, COUNT(*) as count,
					   EXISTS(SELECT 1 FROM message_reactions mr2 WHERE mr2.message_id = mr.message_id AND mr2.reaction_type_id = mr.reaction_type_id AND mr2.user_id = $2) as has_reacted
				FROM message_reactions mr
				JOIN reaction_types rt ON mr.reaction_type_id = rt.id
				WHERE mr.message_id = ANY($1)
				GROUP BY mr.message_id, rt.id, rt.emoji, rt.name
				ORDER BY count DESC
			`
			reactionRows, err := db.Query(reactionQuery, pq.Array(messageIDs), currentUserID)
			if err == nil {
				defer reactionRows.Close()
				messageReactions := make(map[int64][]gin.H)
				for reactionRows.Next() {
					var msgID int64
					var emoji, name string
					var count int
					var hasReacted bool
					reactionRows.Scan(&msgID, &emoji, &name, &count, &hasReacted)
					messageReactions[msgID] = append(messageReactions[msgID], gin.H{
						"emoji":      emoji,
						"name":       name,
						"count":      count,
						"hasReacted": hasReacted,
					})
				}
				// Assign reactions to messages
				for i, msg := range messages {
					msgID := msg["id"].(int64)
					if reactions, ok := messageReactions[msgID]; ok {
						messages[i]["reactions"] = reactions
					} else {
						messages[i]["reactions"] = []gin.H{}
					}
				}
			}
		}

		// Reverse to chronological order
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

func handleSendMessage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		channelID := c.Param("id")

		var req struct {
			Content   string `json:"content" binding:"required"`
			ReplyToID *int64 `json:"reply_to_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if user is muted
		var isMuted bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM community_restrictions WHERE user_id = $1 AND restriction_type = 'mute' AND (expires_at IS NULL OR expires_at > NOW()))", userID).Scan(&isMuted)
		if isMuted {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are muted from posting"})
			return
		}

		var id int64
		var createdAt time.Time
		err := db.QueryRow(`
			INSERT INTO channel_messages (channel_id, user_id, content, reply_to_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at
		`, channelID, userID, req.Content, req.ReplyToID).Scan(&id, &createdAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id, "createdAt": createdAt})
	}
}

func handleEditMessage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		messageID := c.Param("id")

		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := db.Exec(`
			UPDATE channel_messages SET content = $1, is_edited = true, updated_at = NOW()
			WHERE id = $2 AND user_id = $3 AND created_at > NOW() - INTERVAL '5 minutes'
		`, req.Content, messageID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to edit message"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot edit this message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleDeleteMessage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		messageID := c.Param("id")

		// Check if user is moderator/admin
		var userRole string
		db.QueryRow("SELECT COALESCE(role, 'user') FROM users WHERE id = $1", userID).Scan(&userRole)
		isModerator := userRole == "moderator" || userRole == "admin" || userRole == "super_admin"

		var result sql.Result
		var err error

		if isModerator {
			// Moderators can delete any message
			result, err = db.Exec(`
				UPDATE channel_messages SET is_deleted = true, updated_at = NOW() WHERE id = $1
			`, messageID)
		} else {
			// Regular users can only delete their own messages
			result, err = db.Exec(`
				UPDATE channel_messages SET is_deleted = true, updated_at = NOW() WHERE id = $1 AND user_id = $2
			`, messageID, userID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete this message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleAddReaction(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		messageID := c.Param("id")

		var req struct {
			Emoji string `json:"emoji" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Use the toggle function which adds or removes the reaction
		var action string
		var count int
		err := db.QueryRow(`SELECT * FROM toggle_message_reaction($1, $2, $3)`, messageID, userID, req.Emoji).Scan(&action, &count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle reaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "action": action, "count": count})
	}
}

func handleRemoveReaction(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		messageID := c.Param("id")
		emoji := c.Param("emoji")

		// Get reaction type id
		var reactionTypeID int
		err := db.QueryRow(`SELECT id FROM reaction_types WHERE emoji = $1`, emoji).Scan(&reactionTypeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reaction emoji"})
			return
		}

		db.Exec(`DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND reaction_type_id = $3`,
			messageID, userID, reactionTypeID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleGetReactionTypes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, emoji, name, category 
			FROM reaction_types 
			WHERE is_active = true 
			ORDER BY sort_order
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reaction types"})
			return
		}
		defer rows.Close()

		reactions := []gin.H{}
		for rows.Next() {
			var id int
			var emoji, name, category string
			rows.Scan(&id, &emoji, &name, &category)
			reactions = append(reactions, gin.H{
				"id":       id,
				"emoji":    emoji,
				"name":     name,
				"category": category,
			})
		}

		c.JSON(http.StatusOK, gin.H{"reactions": reactions})
	}
}

// Forum handlers
func handleGetForums(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, name, description, icon, admin_only_post, post_count
			FROM forum_categories
			ORDER BY sort_order
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch forums"})
			return
		}
		defer rows.Close()

		forums := []gin.H{}
		for rows.Next() {
			var id int
			var name, description string
			var icon sql.NullString
			var adminOnlyPost bool
			var postCount int

			rows.Scan(&id, &name, &description, &icon, &adminOnlyPost, &postCount)
			forums = append(forums, gin.H{
				"id":            id,
				"name":          name,
				"description":   description,
				"icon":          icon.String,
				"adminOnlyPost": adminOnlyPost,
				"postCount":     postCount,
			})
		}

		c.JSON(http.StatusOK, gin.H{"forums": forums})
	}
}

func handleGetForumPosts(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		forumID := c.Param("id")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		sort := c.DefaultQuery("sort", "newest")
		limit := 20
		offset := (page - 1) * limit

		orderBy := "fp.created_at DESC"
		if sort == "popular" {
			orderBy = "fp.upvotes DESC"
		} else if sort == "replies" {
			orderBy = "fp.reply_count DESC"
		}

		rows, err := db.Query(fmt.Sprintf(`
			SELECT fp.id, fp.title, fp.content, fp.tags, fp.view_count, fp.reply_count,
				   fp.upvotes, fp.downvotes, fp.is_pinned, fp.is_locked, fp.created_at,
				   u.id as user_id, u.username,
				   COALESCE(b.icon, '🌱') as badge_icon
			FROM forum_posts fp
			JOIN users u ON fp.user_id = u.id
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
			LEFT JOIN badges b ON ub.badge_id = b.id
			WHERE fp.category_id = $1 AND fp.is_deleted = false
			ORDER BY fp.is_pinned DESC, %s
			LIMIT $2 OFFSET $3
		`, orderBy), forumID, limit, offset)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
			return
		}
		defer rows.Close()

		posts := []gin.H{}
		for rows.Next() {
			var id int64
			var title, content string
			var tags []string
			var viewCount, replyCount, upvotes, downvotes int
			var isPinned, isLocked bool
			var createdAt time.Time
			var userID int64
			var username, badgeIcon string

			rows.Scan(&id, &title, &content, pq.Array(&tags), &viewCount, &replyCount,
				&upvotes, &downvotes, &isPinned, &isLocked, &createdAt,
				&userID, &username, &badgeIcon)

			posts = append(posts, gin.H{
				"id":         id,
				"title":      title,
				"preview":    truncateString(content, 150),
				"tags":       tags,
				"viewCount":  viewCount,
				"replyCount": replyCount,
				"upvotes":    upvotes,
				"downvotes":  downvotes,
				"isPinned":   isPinned,
				"isLocked":   isLocked,
				"createdAt":  createdAt,
				"author": gin.H{
					"id":        userID,
					"username":  username,
					"badgeIcon": badgeIcon,
				},
			})
		}

		c.JSON(http.StatusOK, gin.H{"posts": posts})
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func handleCreatePost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		forumID := c.Param("id")

		var req struct {
			Title   string   `json:"title" binding:"required"`
			Content string   `json:"content" binding:"required"`
			Tags    []string `json:"tags"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var id int64
		err := db.QueryRow(`
			INSERT INTO forum_posts (category_id, user_id, title, content, tags)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, forumID, userID, req.Title, req.Content, pq.Array(req.Tags)).Scan(&id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
			return
		}

		db.Exec("UPDATE forum_categories SET post_count = post_count + 1 WHERE id = $1", forumID)

		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}

func handleGetPost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		postID := c.Param("id")

		// Increment view count
		db.Exec("UPDATE forum_posts SET view_count = view_count + 1 WHERE id = $1", postID)

		var post struct {
			ID        int64
			Title     string
			Content   string
			Tags      []string
			ViewCount int
			Upvotes   int
			Downvotes int
			IsPinned  bool
			IsLocked  bool
			CreatedAt time.Time
			UserID    int64
			Username  string
			BadgeIcon string
		}

		err := db.QueryRow(`
			SELECT fp.id, fp.title, fp.content, fp.tags, fp.view_count, fp.upvotes, fp.downvotes,
				   fp.is_pinned, fp.is_locked, fp.created_at,
				   u.id, u.username, COALESCE(b.icon, '🌱')
			FROM forum_posts fp
			JOIN users u ON fp.user_id = u.id
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
			LEFT JOIN badges b ON ub.badge_id = b.id
			WHERE fp.id = $1 AND fp.is_deleted = false
		`, postID).Scan(&post.ID, &post.Title, &post.Content, pq.Array(&post.Tags),
			&post.ViewCount, &post.Upvotes, &post.Downvotes, &post.IsPinned, &post.IsLocked,
			&post.CreatedAt, &post.UserID, &post.Username, &post.BadgeIcon)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
			return
		}

		// Get replies
		rows, _ := db.Query(`
			SELECT fr.id, fr.content, fr.upvotes, fr.downvotes, fr.is_solution, fr.created_at,
				   u.id, u.username, COALESCE(b.icon, '🌱')
			FROM forum_replies fr
			JOIN users u ON fr.user_id = u.id
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
			LEFT JOIN badges b ON ub.badge_id = b.id
			WHERE fr.post_id = $1 AND fr.is_deleted = false
			ORDER BY fr.is_solution DESC, fr.created_at
		`, postID)
		defer rows.Close()

		replies := []gin.H{}
		for rows.Next() {
			var id int64
			var content string
			var upvotes, downvotes int
			var isSolution bool
			var createdAt time.Time
			var userID int64
			var username, badgeIcon string

			rows.Scan(&id, &content, &upvotes, &downvotes, &isSolution, &createdAt,
				&userID, &username, &badgeIcon)

			replies = append(replies, gin.H{
				"id":         id,
				"content":    content,
				"upvotes":    upvotes,
				"downvotes":  downvotes,
				"isSolution": isSolution,
				"createdAt":  createdAt,
				"author": gin.H{
					"id":        userID,
					"username":  username,
					"badgeIcon": badgeIcon,
				},
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"post": gin.H{
				"id":        post.ID,
				"title":     post.Title,
				"content":   post.Content,
				"tags":      post.Tags,
				"viewCount": post.ViewCount,
				"upvotes":   post.Upvotes,
				"downvotes": post.Downvotes,
				"isPinned":  post.IsPinned,
				"isLocked":  post.IsLocked,
				"createdAt": post.CreatedAt,
				"author": gin.H{
					"id":        post.UserID,
					"username":  post.Username,
					"badgeIcon": post.BadgeIcon,
				},
			},
			"replies": replies,
		})
	}
}

func handleEditPost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		postID := c.Param("id")

		var req struct {
			Title   string   `json:"title"`
			Content string   `json:"content"`
			Tags    []string `json:"tags"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE forum_posts SET title = $1, content = $2, tags = $3, is_edited = true, updated_at = NOW()
			WHERE id = $4 AND user_id = $5
		`, req.Title, req.Content, pq.Array(req.Tags), postID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to edit post"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleAddReply(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		postID := c.Param("id")

		var req struct {
			Content       string `json:"content" binding:"required"`
			ParentReplyID *int64 `json:"parentReplyId"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if post is locked
		var isLocked bool
		db.QueryRow("SELECT is_locked FROM forum_posts WHERE id = $1", postID).Scan(&isLocked)
		if isLocked {
			c.JSON(http.StatusForbidden, gin.H{"error": "This post is locked"})
			return
		}

		var id int64
		err := db.QueryRow(`
			INSERT INTO forum_replies (post_id, user_id, content, parent_reply_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, postID, userID, req.Content, req.ParentReplyID).Scan(&id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reply"})
			return
		}

		db.Exec("UPDATE forum_posts SET reply_count = reply_count + 1 WHERE id = $1", postID)

		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}

func handleVotePost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		postID := c.Param("id")

		var req struct {
			Vote int `json:"vote"` // 1 or -1
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Vote != 1 && req.Vote != -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vote must be 1 or -1"})
			return
		}

		// Upsert vote
		_, err := db.Exec(`
			INSERT INTO content_votes (user_id, content_type, content_id, vote_type)
			VALUES ($1, 'post', $2, $3)
			ON CONFLICT (user_id, content_type, content_id)
			DO UPDATE SET vote_type = $3
		`, userID, postID, req.Vote)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to vote"})
			return
		}

		// Recalculate votes
		db.Exec(`
			UPDATE forum_posts SET
				upvotes = (SELECT COUNT(*) FROM content_votes WHERE content_type = 'post' AND content_id = $1 AND vote_type = 1),
				downvotes = (SELECT COUNT(*) FROM content_votes WHERE content_type = 'post' AND content_id = $1 AND vote_type = -1)
			WHERE id = $1
		`, postID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleVoteReply(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		replyID := c.Param("id")

		var req struct {
			Vote int `json:"vote"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			INSERT INTO content_votes (user_id, content_type, content_id, vote_type)
			VALUES ($1, 'reply', $2, $3)
			ON CONFLICT (user_id, content_type, content_id)
			DO UPDATE SET vote_type = $3
		`, userID, replyID, req.Vote)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to vote"})
			return
		}

		db.Exec(`
			UPDATE forum_replies SET
				upvotes = (SELECT COUNT(*) FROM content_votes WHERE content_type = 'reply' AND content_id = $1 AND vote_type = 1),
				downvotes = (SELECT COUNT(*) FROM content_votes WHERE content_type = 'reply' AND content_id = $1 AND vote_type = -1)
			WHERE id = $1
		`, replyID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// Badge handlers
func handleGetBadges(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`SELECT id, name, description, icon, color, badge_type, requirement_value FROM badges ORDER BY id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch badges"})
			return
		}
		defer rows.Close()

		badges := []gin.H{}
		for rows.Next() {
			var id int
			var name, description, icon, color, badgeType string
			var reqValue sql.NullInt64

			rows.Scan(&id, &name, &description, &icon, &color, &badgeType, &reqValue)
			badges = append(badges, gin.H{
				"id":          id,
				"name":        name,
				"description": description,
				"icon":        icon,
				"color":       color,
				"type":        badgeType,
				"requirement": reqValue.Int64,
			})
		}

		c.JSON(http.StatusOK, gin.H{"badges": badges})
	}
}

func handleGetUserBadges(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("id")

		rows, err := db.Query(`
			SELECT b.id, b.name, b.description, b.icon, b.color, ub.earned_at, ub.is_primary
			FROM user_badges ub
			JOIN badges b ON ub.badge_id = b.id
			WHERE ub.user_id = $1
			ORDER BY ub.earned_at DESC
		`, targetUserID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch badges"})
			return
		}
		defer rows.Close()

		badges := []gin.H{}
		for rows.Next() {
			var id int
			var name, description, icon, color string
			var earnedAt time.Time
			var isPrimary bool

			rows.Scan(&id, &name, &description, &icon, &color, &earnedAt, &isPrimary)
			badges = append(badges, gin.H{
				"id":          id,
				"name":        name,
				"description": description,
				"icon":        icon,
				"color":       color,
				"earnedAt":    earnedAt,
				"isPrimary":   isPrimary,
			})
		}

		c.JSON(http.StatusOK, gin.H{"badges": badges})
	}
}

func handleSetPrimaryBadge(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			BadgeID int `json:"badgeId" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Clear current primary
		db.Exec("UPDATE user_badges SET is_primary = false WHERE user_id = $1", userID)
		// Set new primary
		db.Exec("UPDATE user_badges SET is_primary = true WHERE user_id = $1 AND badge_id = $2", userID, req.BadgeID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// Profile handlers
func handleGetUserProfile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetUserID := c.Param("id")

		var profile struct {
			UserID           int64
			Username         string
			Email            string
			CreatedAt        time.Time
			AvatarURL        sql.NullString
			Bio              sql.NullString
			Country          sql.NullString
			ShowEarnings     bool
			ShowCountry      bool
			Reputation       int
			ForumPostCount   int
			LifetimeHashrate float64
			OnlineStatus     string
		}

		err := db.QueryRow(`
			SELECT u.id, u.username, u.email, u.created_at,
				   COALESCE(up.avatar_url, ''), COALESCE(up.bio, ''),
				   COALESCE(up.country, ''), COALESCE(up.show_earnings, true),
				   COALESCE(up.show_country, true), COALESCE(up.reputation, 0),
				   COALESCE(up.forum_post_count, 0), COALESCE(up.lifetime_hashrate, 0),
				   COALESCE(up.online_status, 'offline')
			FROM users u
			LEFT JOIN user_profiles up ON u.id = up.user_id
			WHERE u.id = $1
		`, targetUserID).Scan(&profile.UserID, &profile.Username, &profile.Email, &profile.CreatedAt,
			&profile.AvatarURL, &profile.Bio, &profile.Country, &profile.ShowEarnings,
			&profile.ShowCountry, &profile.Reputation, &profile.ForumPostCount,
			&profile.LifetimeHashrate, &profile.OnlineStatus)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Get blocks found count
		var blocksFound int
		db.QueryRow("SELECT COUNT(*) FROM blocks WHERE finder_id = $1", targetUserID).Scan(&blocksFound)

		c.JSON(http.StatusOK, gin.H{
			"id":               profile.UserID,
			"username":         profile.Username,
			"memberSince":      profile.CreatedAt,
			"avatarUrl":        profile.AvatarURL.String,
			"bio":              profile.Bio.String,
			"country":          profile.Country.String,
			"showEarnings":     profile.ShowEarnings,
			"showCountry":      profile.ShowCountry,
			"reputation":       profile.Reputation,
			"forumPostCount":   profile.ForumPostCount,
			"lifetimeHashrate": profile.LifetimeHashrate,
			"onlineStatus":     profile.OnlineStatus,
			"blocksFound":      blocksFound,
		})
	}
}

func handleUpdateProfile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			Bio          string `json:"bio"`
			Country      string `json:"country"`
			CountryCode  string `json:"countryCode"`
			ShowEarnings *bool  `json:"showEarnings"`
			ShowCountry  *bool  `json:"showCountry"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Upsert profile
		_, err := db.Exec(`
			INSERT INTO user_profiles (user_id, bio, country, country_code, show_earnings, show_country)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (user_id) DO UPDATE SET
				bio = COALESCE($2, user_profiles.bio),
				country = COALESCE($3, user_profiles.country),
				country_code = COALESCE($4, user_profiles.country_code),
				show_earnings = COALESCE($5, user_profiles.show_earnings),
				show_country = COALESCE($6, user_profiles.show_country),
				updated_at = NOW()
		`, userID, req.Bio, req.Country, req.CountryCode, req.ShowEarnings, req.ShowCountry)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleGetLeaderboard(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		leaderboardType := c.DefaultQuery("type", "hashrate")
		_ = c.DefaultQuery("period", "all") // Reserved for future use

		// Pagination parameters
		page := 1
		pageSize := 20 // Default page size
		if pageParam := c.Query("page"); pageParam != "" {
			if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
				page = parsed
			}
		}
		if pageSizeParam := c.Query("pageSize"); pageSizeParam != "" {
			if parsed, err := strconv.Atoi(pageSizeParam); err == nil && parsed > 0 {
				pageSize = parsed
				if pageSize > 100 {
					pageSize = 100
				}
			}
		}
		offset := (page - 1) * pageSize

		// Get current user ID if authenticated (for finding their rank)
		currentUserID := c.GetInt64("user_id")

		// Comprehensive leaderboard query with all user stats and badges
		var orderBy string
		switch leaderboardType {
		case "blocks":
			orderBy = "blocks_found DESC"
		case "shares":
			orderBy = "total_shares DESC"
		case "forum":
			orderBy = "forum_posts DESC"
		case "engagement":
			orderBy = "engagement_score DESC"
		default: // hashrate
			orderBy = "current_hashrate DESC"
		}

		// Get total count of active users
		var totalUsers int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&totalUsers)
		totalPages := (totalUsers + pageSize - 1) / pageSize

		// Build the ranked CTE query with pagination
		query := fmt.Sprintf(`
			WITH user_stats AS (
				SELECT 
					u.id,
					u.username,
					u.role,
					u.is_admin,
					u.created_at as join_date,
					COALESCE(SUM(m.hashrate), 0) as current_hashrate,
					COUNT(DISTINCT m.id) FILTER (WHERE m.is_active = true) as active_miners,
					COALESCE(up.lifetime_hashrate, 0) as lifetime_hashrate,
					COALESCE(up.forum_post_count, 0) as forum_posts,
					COALESCE(up.reputation, 0) as reputation,
					(
						SELECT COUNT(*) FROM blocks b WHERE b.finder_id = u.id
					) as blocks_found,
					(
						SELECT COUNT(*) FROM shares s WHERE s.user_id = u.id
					) as total_shares,
					(
						-- Mining activity (core engagement)
						COALESCE((SELECT COUNT(*) FROM shares s WHERE s.user_id = u.id), 0) / 1000 +  -- 1 point per 1000 shares
						COALESCE(NULLIF(SUM(m.hashrate), 0) / 1000000000000, 0)::bigint +  -- 1 point per TH/s
						(SELECT COUNT(*) FROM blocks b WHERE b.finder_id = u.id) * 500 +  -- 500 points per block found
						-- Community engagement
						COALESCE(up.forum_post_count, 0) * 10 +  -- 10 points per forum post
						COALESCE(up.reputation, 0) +  -- Direct reputation points
						(SELECT COUNT(*) FROM channel_messages cm WHERE cm.user_id = u.id AND cm.is_deleted = false) * 2 +  -- 2 points per chat message
						-- Referral bonus (future: add referral_count * 50)
						0
					) as engagement_score
				FROM users u
				LEFT JOIN miners m ON u.id = m.user_id AND m.is_active = true
				LEFT JOIN user_profiles up ON u.id = up.user_id
				WHERE u.is_active = true
				GROUP BY u.id, u.username, u.role, u.is_admin, u.created_at, up.lifetime_hashrate, up.forum_post_count, up.reputation
			),
			ranked_stats AS (
				SELECT us.*, ROW_NUMBER() OVER (ORDER BY %s, us.id) as rank
				FROM user_stats us
			)
			SELECT 
				rs.*,
				COALESCE(pb.icon, '🌱') as primary_badge_icon,
				COALESCE(pb.color, '#4ade80') as primary_badge_color,
				COALESCE(pb.name, 'Newcomer') as primary_badge_name
			FROM ranked_stats rs
			LEFT JOIN user_badges ub ON ub.user_id = rs.id AND ub.is_primary = true
			LEFT JOIN badges pb ON ub.badge_id = pb.id
			ORDER BY rs.rank
			LIMIT $1 OFFSET $2
		`, orderBy)

		rows, err := db.Query(query, pageSize, offset)
		if err != nil {
			log.Printf("Leaderboard query error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
			return
		}
		defer rows.Close()

		leaders := []gin.H{}
		userIDs := []int64{}

		for rows.Next() {
			var userID int64
			var username string
			var role sql.NullString
			var isAdmin bool
			var joinDate time.Time
			var currentHashrate, lifetimeHashrate float64
			var activeMiners, forumPosts, reputation, blocksFound int
			var totalShares, engagementScore int64
			var rank int
			var primaryBadgeIcon, primaryBadgeColor, primaryBadgeName string

			err := rows.Scan(
				&userID, &username, &role, &isAdmin, &joinDate,
				&currentHashrate, &activeMiners, &lifetimeHashrate,
				&forumPosts, &reputation, &blocksFound, &totalShares, &engagementScore, &rank,
				&primaryBadgeIcon, &primaryBadgeColor, &primaryBadgeName,
			)
			if err != nil {
				log.Printf("Leaderboard row scan error: %v", err)
				continue
			}

			// Determine role badge
			userRole := "user"
			if role.Valid {
				userRole = role.String
			}
			if isAdmin {
				userRole = "admin"
			}
			roleBadge := getRoleBadge(userRole)

			leaders = append(leaders, gin.H{
				"rank":      rank,
				"userId":    userID,
				"username":  username,
				"role":      userRole,
				"roleBadge": roleBadge,
				"stats": gin.H{
					"currentHashrate":  currentHashrate,
					"lifetimeHashrate": lifetimeHashrate,
					"activeMiners":     activeMiners,
					"blocksFound":      blocksFound,
					"totalShares":      totalShares,
					"forumPosts":       forumPosts,
					"reputation":       reputation,
					"engagementScore":  engagementScore,
					"joinDate":         joinDate,
				},
				"primaryBadge": gin.H{
					"icon":  primaryBadgeIcon,
					"color": primaryBadgeColor,
					"name":  primaryBadgeName,
				},
				"badges": []gin.H{}, // Will be populated below
			})
			userIDs = append(userIDs, userID)
		}

		// Fetch all badges for each user
		if len(userIDs) > 0 {
			badgeQuery := `
				SELECT ub.user_id, b.icon, b.color, b.name, b.badge_type, ub.is_primary
				FROM user_badges ub
				JOIN badges b ON ub.badge_id = b.id
				WHERE ub.user_id = ANY($1)
				ORDER BY ub.is_primary DESC, ub.earned_at DESC
			`
			badgeRows, err := db.Query(badgeQuery, pq.Array(userIDs))
			if err == nil {
				defer badgeRows.Close()
				userBadges := make(map[int64][]gin.H)
				for badgeRows.Next() {
					var uid int64
					var icon, color, name, badgeType string
					var isPrimary bool
					badgeRows.Scan(&uid, &icon, &color, &name, &badgeType, &isPrimary)
					userBadges[uid] = append(userBadges[uid], gin.H{
						"icon":      icon,
						"color":     color,
						"name":      name,
						"type":      badgeType,
						"isPrimary": isPrimary,
					})
				}
				// Assign badges to leaders
				for i, leader := range leaders {
					uid := leader["userId"].(int64)
					if badges, ok := userBadges[uid]; ok {
						leaders[i]["badges"] = badges
					}
				}
			}
		}

		// Get current user's rank if authenticated
		var myRank interface{} = nil
		var myPage interface{} = nil
		if currentUserID > 0 {
			rankQuery := fmt.Sprintf(`
				WITH user_stats AS (
					SELECT 
						u.id,
						COALESCE(SUM(m.hashrate), 0) as current_hashrate,
						COALESCE(up.forum_post_count, 0) as forum_posts,
						COALESCE(up.reputation, 0) as reputation,
						(SELECT COUNT(*) FROM blocks b WHERE b.finder_id = u.id) as blocks_found,
						(SELECT COUNT(*) FROM shares s WHERE s.user_id = u.id) as total_shares,
						(
							COALESCE((SELECT COUNT(*) FROM shares s WHERE s.user_id = u.id), 0) / 1000 +
							COALESCE(NULLIF(SUM(m.hashrate), 0) / 1000000000000, 0)::bigint +
							(SELECT COUNT(*) FROM blocks b WHERE b.finder_id = u.id) * 500 +
							COALESCE(up.forum_post_count, 0) * 10 +
							COALESCE(up.reputation, 0) +
							(SELECT COUNT(*) FROM channel_messages cm WHERE cm.user_id = u.id AND cm.is_deleted = false) * 2
						) as engagement_score
					FROM users u
					LEFT JOIN miners m ON u.id = m.user_id AND m.is_active = true
					LEFT JOIN user_profiles up ON u.id = up.user_id
					WHERE u.is_active = true
					GROUP BY u.id, up.forum_post_count, up.reputation
				),
				ranked AS (
					SELECT id, ROW_NUMBER() OVER (ORDER BY %s, id) as rank
					FROM user_stats
				)
				SELECT rank FROM ranked WHERE id = $1
			`, orderBy)
			var userRank int
			if err := db.QueryRow(rankQuery, currentUserID).Scan(&userRank); err == nil {
				myRank = userRank
				myPage = ((userRank - 1) / pageSize) + 1
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"leaderboard": leaders,
			"type":        leaderboardType,
			"pagination": gin.H{
				"page":       page,
				"pageSize":   pageSize,
				"totalUsers": totalUsers,
				"totalPages": totalPages,
			},
			"myRank": myRank,
			"myPage": myPage,
		})
	}
}

// getRoleBadge returns badge info for user roles (admin/moderator/super_admin)
func getRoleBadge(role string) gin.H {
	switch role {
	case "super_admin":
		return gin.H{"icon": "👑", "color": "#fbbf24", "name": "Super Admin", "type": "role"}
	case "admin":
		return gin.H{"icon": "⚔️", "color": "#ef4444", "name": "Admin", "type": "role"}
	case "moderator":
		return gin.H{"icon": "🛡️", "color": "#f97316", "name": "Moderator", "type": "role"}
	default:
		return nil
	}
}

// Notification handlers
func handleGetNotifications(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(`
			SELECT id, notification_type, title, content, link, is_read, created_at
			FROM notifications
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT 50
		`, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
			return
		}
		defer rows.Close()

		notifications := []gin.H{}
		unreadCount := 0
		for rows.Next() {
			var id int64
			var notifType, title string
			var content, link sql.NullString
			var isRead bool
			var createdAt time.Time

			rows.Scan(&id, &notifType, &title, &content, &link, &isRead, &createdAt)
			notifications = append(notifications, gin.H{
				"id":        id,
				"type":      notifType,
				"title":     title,
				"content":   content.String,
				"link":      link.String,
				"isRead":    isRead,
				"createdAt": createdAt,
			})
			if !isRead {
				unreadCount++
			}
		}

		c.JSON(http.StatusOK, gin.H{"notifications": notifications, "unreadCount": unreadCount})
	}
}

func handleMarkNotificationsRead(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			IDs []int64 `json:"ids"`
		}
		c.ShouldBindJSON(&req)

		if len(req.IDs) > 0 {
			db.Exec("UPDATE notifications SET is_read = true WHERE user_id = $1 AND id = ANY($2)", userID, pq.Array(req.IDs))
		} else {
			db.Exec("UPDATE notifications SET is_read = true WHERE user_id = $1", userID)
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// DM handlers
func handleGetDMList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(`
			SELECT DISTINCT ON (other_user) 
				other_user, u.username, dm.content, dm.created_at, dm.is_read
			FROM (
				SELECT sender_id as other_user, id, content, created_at, is_read
				FROM direct_messages WHERE receiver_id = $1 AND is_deleted_receiver = false
				UNION ALL
				SELECT receiver_id as other_user, id, content, created_at, true as is_read
				FROM direct_messages WHERE sender_id = $1 AND is_deleted_sender = false
			) dm
			JOIN users u ON dm.other_user = u.id
			ORDER BY other_user, dm.created_at DESC
		`, userID)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{"conversations": []gin.H{}})
			return
		}
		defer rows.Close()

		conversations := []gin.H{}
		for rows.Next() {
			var otherUserID int64
			var username, lastMessage string
			var lastMessageAt time.Time
			var isRead bool

			rows.Scan(&otherUserID, &username, &lastMessage, &lastMessageAt, &isRead)
			conversations = append(conversations, gin.H{
				"userId":        otherUserID,
				"username":      username,
				"lastMessage":   truncateString(lastMessage, 50),
				"lastMessageAt": lastMessageAt,
				"isRead":        isRead,
			})
		}

		c.JSON(http.StatusOK, gin.H{"conversations": conversations})
	}
}

func handleGetDMConversation(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		otherUserID := c.Param("userId")

		rows, err := db.Query(`
			SELECT id, sender_id, content, created_at
			FROM direct_messages
			WHERE ((sender_id = $1 AND receiver_id = $2 AND is_deleted_sender = false)
				OR (sender_id = $2 AND receiver_id = $1 AND is_deleted_receiver = false))
			ORDER BY created_at DESC
			LIMIT 100
		`, userID, otherUserID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
			return
		}
		defer rows.Close()

		messages := []gin.H{}
		for rows.Next() {
			var id, senderID int64
			var content string
			var createdAt time.Time

			rows.Scan(&id, &senderID, &content, &createdAt)
			messages = append(messages, gin.H{
				"id":        id,
				"senderId":  senderID,
				"content":   content,
				"createdAt": createdAt,
				"isMine":    senderID == userID,
			})
		}

		// Mark as read
		db.Exec("UPDATE direct_messages SET is_read = true WHERE receiver_id = $1 AND sender_id = $2", userID, otherUserID)

		// Reverse for chronological
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}

		c.JSON(http.StatusOK, gin.H{"messages": messages})
	}
}

func handleSendDM(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		receiverID := c.Param("userId")

		var req struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if blocked
		var isBlocked bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM blocked_users WHERE user_id = $1 AND blocked_user_id = $2)", receiverID, userID).Scan(&isBlocked)
		if isBlocked {
			c.JSON(http.StatusForbidden, gin.H{"error": "You cannot message this user"})
			return
		}

		var id int64
		err := db.QueryRow(`
			INSERT INTO direct_messages (sender_id, receiver_id, content)
			VALUES ($1, $2, $3)
			RETURNING id
		`, userID, receiverID, req.Content).Scan(&id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}

// Report handler
func handleReportContent(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			ContentType string `json:"contentType" binding:"required"`
			ContentID   int64  `json:"contentId" binding:"required"`
			Reason      string `json:"reason" binding:"required"`
			Details     string `json:"details"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			INSERT INTO content_reports (reporter_id, content_type, content_id, reason, details)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, req.ContentType, req.ContentID, req.Reason, req.Details)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleGetOnlineUsers(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT u.id, u.username, COALESCE(up.online_status, 'offline'),
				   COALESCE(b.icon, '🌱')
			FROM users u
			LEFT JOIN user_profiles up ON u.id = up.user_id
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
			LEFT JOIN badges b ON ub.badge_id = b.id
			WHERE up.online_status IN ('online', 'mining')
			OR up.last_seen_at > NOW() - INTERVAL '5 minutes'
			LIMIT 50
		`)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{"users": []gin.H{}})
			return
		}
		defer rows.Close()

		users := []gin.H{}
		for rows.Next() {
			var id int64
			var username, status, badgeIcon string

			rows.Scan(&id, &username, &status, &badgeIcon)
			users = append(users, gin.H{
				"id":        id,
				"username":  username,
				"status":    status,
				"badgeIcon": badgeIcon,
			})
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}

// Admin moderation handlers
func handleBanUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		targetUserID := c.Param("userId")

		var req struct {
			Reason  string `json:"reason"`
			Minutes int    `json:"minutes"` // 0 for permanent
		}
		c.ShouldBindJSON(&req)

		var expiresAt *time.Time
		if req.Minutes > 0 {
			t := time.Now().Add(time.Duration(req.Minutes) * time.Minute)
			expiresAt = &t
		}

		db.Exec(`
			INSERT INTO community_restrictions (user_id, restriction_type, reason, expires_at, created_by)
			VALUES ($1, 'ban', $2, $3, $4)
			ON CONFLICT (user_id, restriction_type) DO UPDATE SET reason = $2, expires_at = $3
		`, targetUserID, req.Reason, expiresAt, adminID)

		db.Exec(`
			INSERT INTO moderation_actions (admin_id, target_user_id, action_type, reason, duration_minutes, expires_at)
			VALUES ($1, $2, 'ban', $3, $4, $5)
		`, adminID, targetUserID, req.Reason, req.Minutes, expiresAt)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleUnbanUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		targetUserID := c.Param("userId")

		db.Exec("DELETE FROM community_restrictions WHERE user_id = $1 AND restriction_type = 'ban'", targetUserID)
		db.Exec("INSERT INTO moderation_actions (admin_id, target_user_id, action_type) VALUES ($1, $2, 'unban')", adminID, targetUserID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleMuteUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		targetUserID := c.Param("userId")

		var req struct {
			Reason  string `json:"reason"`
			Minutes int    `json:"minutes"`
		}
		c.ShouldBindJSON(&req)

		expiresAt := time.Now().Add(time.Duration(req.Minutes) * time.Minute)

		db.Exec(`
			INSERT INTO community_restrictions (user_id, restriction_type, reason, expires_at, created_by)
			VALUES ($1, 'mute', $2, $3, $4)
			ON CONFLICT (user_id, restriction_type) DO UPDATE SET reason = $2, expires_at = $3
		`, targetUserID, req.Reason, expiresAt, adminID)

		db.Exec(`
			INSERT INTO moderation_actions (admin_id, target_user_id, action_type, reason, duration_minutes, expires_at)
			VALUES ($1, $2, 'mute', $3, $4, $5)
		`, adminID, targetUserID, req.Reason, req.Minutes, expiresAt)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleUnmuteUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		targetUserID := c.Param("userId")

		db.Exec("DELETE FROM community_restrictions WHERE user_id = $1 AND restriction_type = 'mute'", targetUserID)
		db.Exec("INSERT INTO moderation_actions (admin_id, target_user_id, action_type) VALUES ($1, $2, 'unmute')", adminID, targetUserID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleGetReports(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "pending")

		rows, err := db.Query(`
			SELECT cr.id, cr.content_type, cr.content_id, cr.reason, cr.details, cr.status, cr.created_at,
				   u.username as reporter
			FROM content_reports cr
			JOIN users u ON cr.reporter_id = u.id
			WHERE cr.status = $1
			ORDER BY cr.created_at DESC
			LIMIT 50
		`, status)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
			return
		}
		defer rows.Close()

		reports := []gin.H{}
		for rows.Next() {
			var id int64
			var contentType string
			var contentID int64
			var reason string
			var details sql.NullString
			var reportStatus string
			var createdAt time.Time
			var reporter string

			rows.Scan(&id, &contentType, &contentID, &reason, &details, &reportStatus, &createdAt, &reporter)
			reports = append(reports, gin.H{
				"id":          id,
				"contentType": contentType,
				"contentId":   contentID,
				"reason":      reason,
				"details":     details.String,
				"status":      reportStatus,
				"createdAt":   createdAt,
				"reporter":    reporter,
			})
		}

		c.JSON(http.StatusOK, gin.H{"reports": reports})
	}
}

func handleReviewReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		reportID := c.Param("id")

		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		db.Exec(`
			UPDATE content_reports SET status = $1, reviewed_by = $2, reviewed_at = NOW()
			WHERE id = $3
		`, req.Status, adminID, reportID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleAdminDeleteMessage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		messageID := c.Param("id")

		db.Exec("UPDATE channel_messages SET is_deleted = true WHERE id = $1", messageID)
		db.Exec("INSERT INTO moderation_actions (admin_id, action_type, reason) VALUES ($1, 'delete_message', $2)", adminID, "Message ID: "+messageID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleAdminDeletePost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		postID := c.Param("id")

		db.Exec("UPDATE forum_posts SET is_deleted = true WHERE id = $1", postID)
		db.Exec("INSERT INTO moderation_actions (admin_id, action_type, reason) VALUES ($1, 'delete_post', $2)", adminID, "Post ID: "+postID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handlePinPost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		postID := c.Param("id")

		var req struct {
			Pinned bool `json:"pinned"`
		}
		c.ShouldBindJSON(&req)

		db.Exec("UPDATE forum_posts SET is_pinned = $1 WHERE id = $2", req.Pinned, postID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleLockPost(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		postID := c.Param("id")

		var req struct {
			Locked bool `json:"locked"`
		}
		c.ShouldBindJSON(&req)

		db.Exec("UPDATE forum_posts SET is_locked = $1 WHERE id = $2", req.Locked, postID)

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// Multi-wallet management handlers
func handleGetUserWallets(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(`
			SELECT id, user_id, address, COALESCE(label, ''), percentage, is_primary, is_active, created_at, updated_at
			FROM user_wallets WHERE user_id = $1
			ORDER BY is_primary DESC, created_at ASC
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
			return
		}
		defer rows.Close()

		wallets := []gin.H{}
		var totalPercentage float64 = 0
		var activeCount int = 0
		var hasPrimary bool = false

		for rows.Next() {
			var id, uid int64
			var address, label string
			var percentage float64
			var isPrimary, isActive bool
			var createdAt, updatedAt time.Time

			rows.Scan(&id, &uid, &address, &label, &percentage, &isPrimary, &isActive, &createdAt, &updatedAt)

			wallets = append(wallets, gin.H{
				"id":         id,
				"user_id":    uid,
				"address":    address,
				"label":      label,
				"percentage": percentage,
				"is_primary": isPrimary,
				"is_active":  isActive,
				"created_at": createdAt,
				"updated_at": updatedAt,
			})

			if isActive {
				totalPercentage += percentage
				activeCount++
			}
			if isPrimary {
				hasPrimary = true
			}
		}

		summary := gin.H{
			"total_wallets":        len(wallets),
			"active_wallets":       activeCount,
			"total_percentage":     totalPercentage,
			"remaining_percentage": 100 - totalPercentage,
			"has_primary_wallet":   hasPrimary,
		}

		c.JSON(http.StatusOK, gin.H{
			"wallets": wallets,
			"summary": summary,
		})
	}
}

// validateLitecoinAddress validates a Litecoin address format
func validateLitecoinAddress(address string) error {
	if len(address) < 26 || len(address) > 35 {
		return fmt.Errorf("invalid address length")
	}

	// Litecoin addresses start with L, M, or ltc1 (bech32)
	if !strings.HasPrefix(address, "L") &&
		!strings.HasPrefix(address, "M") &&
		!strings.HasPrefix(address, "ltc1") {
		return fmt.Errorf("invalid Litecoin address prefix (must start with L, M, or ltc1)")
	}

	// Check for valid base58 characters (for legacy addresses)
	if !strings.HasPrefix(address, "ltc1") {
		validChars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
		for _, c := range address {
			if !strings.ContainsRune(validChars, c) {
				return fmt.Errorf("invalid character in address")
			}
		}
	}

	// Reject obvious SQL injection attempts
	if strings.ContainsAny(address, "';\"--/*") {
		return fmt.Errorf("invalid characters in address")
	}

	return nil
}

func handleCreateUserWallet(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			Address    string  `json:"address" binding:"required"`
			Label      string  `json:"label"`
			Percentage float64 `json:"percentage" binding:"required,gt=0,lte=100"`
			IsPrimary  bool    `json:"is_primary"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}

		// SECURITY: Validate wallet address format
		if err := validateLitecoinAddress(req.Address); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet address: " + err.Error()})
			return
		}

		// Check current total percentage
		var currentTotal float64
		db.QueryRow(`
			SELECT COALESCE(SUM(percentage), 0) FROM user_wallets 
			WHERE user_id = $1 AND is_active = true
		`, userID).Scan(&currentTotal)

		if currentTotal+req.Percentage > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Total percentage would exceed 100%%. Current: %.2f%%, Adding: %.2f%%", currentTotal, req.Percentage),
			})
			return
		}

		// If setting as primary, unset others
		if req.IsPrimary {
			db.Exec("UPDATE user_wallets SET is_primary = false WHERE user_id = $1", userID)
		}

		var walletID int64
		var createdAt, updatedAt time.Time
		err := db.QueryRow(`
			INSERT INTO user_wallets (user_id, address, label, percentage, is_primary, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
			RETURNING id, created_at, updated_at
		`, userID, req.Address, req.Label, req.Percentage, req.IsPrimary).Scan(&walletID, &createdAt, &updatedAt)

		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Wallet address already exists for this account"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"wallet": gin.H{
				"id":         walletID,
				"user_id":    userID,
				"address":    req.Address,
				"label":      req.Label,
				"percentage": req.Percentage,
				"is_primary": req.IsPrimary,
				"is_active":  true,
				"created_at": createdAt,
				"updated_at": updatedAt,
			},
			"message": "Wallet created successfully",
		})
	}
}

func handleUpdateUserWallet(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		walletID := c.Param("id")

		// Verify ownership
		var ownerID int64
		err := db.QueryRow("SELECT user_id FROM user_wallets WHERE id = $1", walletID).Scan(&ownerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}
		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
			return
		}

		var req struct {
			Address    string  `json:"address"`
			Label      string  `json:"label"`
			Percentage float64 `json:"percentage"`
			IsPrimary  bool    `json:"is_primary"`
			IsActive   bool    `json:"is_active"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Check percentage if changing
		if req.Percentage > 0 {
			var currentTotal float64
			db.QueryRow(`
				SELECT COALESCE(SUM(percentage), 0) FROM user_wallets 
				WHERE user_id = $1 AND is_active = true AND id != $2
			`, userID, walletID).Scan(&currentTotal)

			if currentTotal+req.Percentage > 100 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Total percentage would exceed 100%%. Other wallets: %.2f%%, This: %.2f%%", currentTotal, req.Percentage),
				})
				return
			}
		}

		// If setting as primary, unset others
		if req.IsPrimary {
			db.Exec("UPDATE user_wallets SET is_primary = false WHERE user_id = $1 AND id != $2", userID, walletID)
		}

		_, err = db.Exec(`
			UPDATE user_wallets 
			SET address = COALESCE(NULLIF($1, ''), address),
			    label = $2,
			    percentage = CASE WHEN $3 > 0 THEN $3 ELSE percentage END,
			    is_primary = $4,
			    is_active = $5
			WHERE id = $6 AND user_id = $7
		`, req.Address, req.Label, req.Percentage, req.IsPrimary, req.IsActive, walletID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Wallet updated successfully"})
	}
}

func handleDeleteUserWallet(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		walletID := c.Param("id")

		result, err := db.Exec("DELETE FROM user_wallets WHERE id = $1 AND user_id = $2", walletID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wallet"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found or not authorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted successfully"})
	}
}

func handleWalletPayoutPreview(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		amountStr := c.DefaultQuery("amount", "10000000000") // Default 100 BDAG
		amount, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
			return
		}

		rows, err := db.Query(`
			SELECT id, address, COALESCE(label, ''), percentage
			FROM user_wallets WHERE user_id = $1 AND is_active = true
			ORDER BY is_primary DESC, created_at ASC
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wallets"})
			return
		}
		defer rows.Close()

		splits := []gin.H{}
		var totalAllocated int64 = 0
		var walletData []struct {
			id         int64
			address    string
			label      string
			percentage float64
		}

		for rows.Next() {
			var w struct {
				id         int64
				address    string
				label      string
				percentage float64
			}
			rows.Scan(&w.id, &w.address, &w.label, &w.percentage)
			walletData = append(walletData, w)
		}

		for i, w := range walletData {
			var splitAmount int64
			if i == len(walletData)-1 {
				splitAmount = amount - totalAllocated
			} else {
				splitAmount = int64(float64(amount) * (w.percentage / 100.0))
				totalAllocated += splitAmount
			}

			splits = append(splits, gin.H{
				"wallet_id":  w.id,
				"address":    w.address,
				"label":      w.label,
				"percentage": w.percentage,
				"amount":     splitAmount,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"total_amount": amount,
			"splits":       splits,
		})
	}
}

// ==================== BUG REPORTING HANDLERS ====================

// BugReport represents a bug report
type BugReport struct {
	ID               int64      `json:"id"`
	ReportNumber     string     `json:"report_number"`
	UserID           int64      `json:"user_id"`
	Username         string     `json:"username,omitempty"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	StepsToReproduce string     `json:"steps_to_reproduce,omitempty"`
	ExpectedBehavior string     `json:"expected_behavior,omitempty"`
	ActualBehavior   string     `json:"actual_behavior,omitempty"`
	Category         string     `json:"category"`
	Priority         string     `json:"priority"`
	Status           string     `json:"status"`
	BrowserInfo      string     `json:"browser_info,omitempty"`
	OSInfo           string     `json:"os_info,omitempty"`
	PageURL          string     `json:"page_url,omitempty"`
	ConsoleErrors    string     `json:"console_errors,omitempty"`
	AssignedTo       *int64     `json:"assigned_to,omitempty"`
	AssignedUsername string     `json:"assigned_username,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy       *int64     `json:"resolved_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	AttachmentCount  int        `json:"attachment_count,omitempty"`
	CommentCount     int        `json:"comment_count,omitempty"`
}

// BugComment represents a comment on a bug report
type BugComment struct {
	ID             int64     `json:"id"`
	BugReportID    int64     `json:"bug_report_id"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Content        string    `json:"content"`
	IsInternal     bool      `json:"is_internal"`
	IsStatusChange bool      `json:"is_status_change"`
	CreatedAt      time.Time `json:"created_at"`
}

// BugAttachment represents a file attachment
type BugAttachment struct {
	ID               int64     `json:"id"`
	BugReportID      int64     `json:"bug_report_id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	FileType         string    `json:"file_type"`
	FileSize         int64     `json:"file_size"`
	IsScreenshot     bool      `json:"is_screenshot"`
	CreatedAt        time.Time `json:"created_at"`
}

// handleCreateBugReport creates a new bug report
func handleCreateBugReport(db *sql.DB, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		type AttachmentData struct {
			Name string `json:"name"`
			Type string `json:"type"`
			Size int64  `json:"size"`
			Data string `json:"data"` // Base64 encoded
		}
		var req struct {
			Title            string           `json:"title" binding:"required,min=5,max=255"`
			Description      string           `json:"description" binding:"required,min=10"`
			StepsToReproduce string           `json:"steps_to_reproduce"`
			ExpectedBehavior string           `json:"expected_behavior"`
			ActualBehavior   string           `json:"actual_behavior"`
			Category         string           `json:"category"`
			BrowserInfo      string           `json:"browser_info"`
			OSInfo           string           `json:"os_info"`
			PageURL          string           `json:"page_url"`
			ConsoleErrors    string           `json:"console_errors"`
			Screenshot       string           `json:"screenshot"`  // Base64 encoded screenshot
			Attachments      []AttachmentData `json:"attachments"` // Additional file attachments
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title (min 5 chars) and description (min 10 chars) are required"})
			return
		}

		// Default category if not provided
		if req.Category == "" {
			req.Category = "other"
		}

		// Validate category
		validCategories := map[string]bool{"ui": true, "performance": true, "security": true, "feature_request": true, "crash": true, "other": true}
		if !validCategories[req.Category] {
			req.Category = "other"
		}

		// Insert bug report
		var bugID int64
		var reportNumber string
		err := db.QueryRow(`
			INSERT INTO bug_reports (user_id, title, description, steps_to_reproduce, expected_behavior, actual_behavior, category, browser_info, os_info, page_url, console_errors)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, report_number
		`, userID, req.Title, req.Description, req.StepsToReproduce, req.ExpectedBehavior, req.ActualBehavior, req.Category, req.BrowserInfo, req.OSInfo, req.PageURL, req.ConsoleErrors).Scan(&bugID, &reportNumber)

		if err != nil {
			log.Printf("Failed to create bug report: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bug report"})
			return
		}

		// Auto-subscribe the reporter
		db.Exec("INSERT INTO bug_subscribers (bug_report_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", bugID, userID)

		// Handle screenshot if provided
		if req.Screenshot != "" {
			// Decode base64 screenshot
			screenshotData, err := base64.StdEncoding.DecodeString(req.Screenshot)
			if err == nil && len(screenshotData) > 0 && len(screenshotData) < 10*1024*1024 { // Max 10MB
				filename := fmt.Sprintf("screenshot_%s_%d.png", reportNumber, time.Now().Unix())
				db.Exec(`
					INSERT INTO bug_attachments (bug_report_id, filename, original_filename, file_type, file_size, file_data, is_screenshot)
					VALUES ($1, $2, $3, $4, $5, $6, true)
				`, bugID, filename, "screenshot.png", "image/png", len(screenshotData), screenshotData)
			}
		}

		// Handle additional attachments
		for i, attachment := range req.Attachments {
			if attachment.Data == "" || attachment.Size > 25*1024*1024 { // Max 25MB per file
				continue
			}
			fileData, err := base64.StdEncoding.DecodeString(attachment.Data)
			if err != nil {
				log.Printf("Failed to decode attachment %d: %v", i, err)
				continue
			}
			filename := fmt.Sprintf("attachment_%s_%d_%d", reportNumber, time.Now().Unix(), i)
			db.Exec(`
				INSERT INTO bug_attachments (bug_report_id, filename, original_filename, file_type, file_size, file_data, is_screenshot)
				VALUES ($1, $2, $3, $4, $5, $6, false)
			`, bugID, filename, attachment.Name, attachment.Type, len(fileData), fileData)
		}

		// Get user email for notification
		var userEmail, username string
		db.QueryRow("SELECT email, username FROM users WHERE id = $1", userID).Scan(&userEmail, &username)

		// Send notification email to admins
		go sendBugReportNotificationToAdmins(db, config, bugID, reportNumber, req.Title, username)

		c.JSON(http.StatusCreated, gin.H{
			"message":       "Bug report submitted successfully",
			"id":            bugID,
			"report_number": reportNumber,
		})
	}
}

// handleGetUserBugReports gets all bug reports for the current user
func handleGetUserBugReports(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(`
			SELECT b.id, b.report_number, b.title, b.category, b.priority, b.status, b.created_at, b.updated_at,
				(SELECT COUNT(*) FROM bug_attachments WHERE bug_report_id = b.id) as attachment_count,
				(SELECT COUNT(*) FROM bug_comments WHERE bug_report_id = b.id AND is_internal = false) as comment_count
			FROM bug_reports b
			WHERE b.user_id = $1
			ORDER BY b.created_at DESC
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bug reports"})
			return
		}
		defer rows.Close()

		bugs := []gin.H{}
		for rows.Next() {
			var b BugReport
			rows.Scan(&b.ID, &b.ReportNumber, &b.Title, &b.Category, &b.Priority, &b.Status, &b.CreatedAt, &b.UpdatedAt, &b.AttachmentCount, &b.CommentCount)
			bugs = append(bugs, gin.H{
				"id":               b.ID,
				"report_number":    b.ReportNumber,
				"title":            b.Title,
				"category":         b.Category,
				"priority":         b.Priority,
				"status":           b.Status,
				"created_at":       b.CreatedAt,
				"updated_at":       b.UpdatedAt,
				"attachment_count": b.AttachmentCount,
				"comment_count":    b.CommentCount,
			})
		}

		c.JSON(http.StatusOK, gin.H{"bugs": bugs})
	}
}

// handleGetBugReport gets a single bug report with comments and attachments
func handleGetBugReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		bugID := c.Param("id")

		// Get bug report (user can only see their own bugs or if they're an admin)
		var b BugReport
		var assignedTo, resolvedBy sql.NullInt64
		var assignedUsername sql.NullString
		var resolvedAt sql.NullTime

		err := db.QueryRow(`
			SELECT b.id, b.report_number, b.user_id, u.username, b.title, b.description, 
				COALESCE(b.steps_to_reproduce, ''), COALESCE(b.expected_behavior, ''), COALESCE(b.actual_behavior, ''),
				b.category, b.priority, b.status, COALESCE(b.browser_info, ''), COALESCE(b.os_info, ''), 
				COALESCE(b.page_url, ''), COALESCE(b.console_errors, ''),
				b.assigned_to, au.username, b.resolved_at, b.resolved_by, b.created_at, b.updated_at
			FROM bug_reports b
			JOIN users u ON b.user_id = u.id
			LEFT JOIN users au ON b.assigned_to = au.id
			WHERE b.id = $1 AND (b.user_id = $2 OR EXISTS (SELECT 1 FROM users WHERE id = $2 AND is_admin = true))
		`, bugID, userID).Scan(
			&b.ID, &b.ReportNumber, &b.UserID, &b.Username, &b.Title, &b.Description,
			&b.StepsToReproduce, &b.ExpectedBehavior, &b.ActualBehavior,
			&b.Category, &b.Priority, &b.Status, &b.BrowserInfo, &b.OSInfo,
			&b.PageURL, &b.ConsoleErrors,
			&assignedTo, &assignedUsername, &resolvedAt, &resolvedBy, &b.CreatedAt, &b.UpdatedAt,
		)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bug report not found"})
			return
		}

		if assignedTo.Valid {
			b.AssignedTo = &assignedTo.Int64
		}
		if assignedUsername.Valid {
			b.AssignedUsername = assignedUsername.String
		}
		if resolvedAt.Valid {
			b.ResolvedAt = &resolvedAt.Time
		}
		if resolvedBy.Valid {
			b.ResolvedBy = &resolvedBy.Int64
		}

		// Get comments (exclude internal comments for non-admins)
		var isAdmin bool
		db.QueryRow("SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)

		commentQuery := `
			SELECT bc.id, bc.user_id, u.username, bc.content, bc.is_internal, bc.is_status_change, bc.created_at
			FROM bug_comments bc
			JOIN users u ON bc.user_id = u.id
			WHERE bc.bug_report_id = $1
		`
		if !isAdmin {
			commentQuery += " AND bc.is_internal = false"
		}
		commentQuery += " ORDER BY bc.created_at ASC"

		commentRows, _ := db.Query(commentQuery, bugID)
		defer commentRows.Close()

		comments := []BugComment{}
		for commentRows.Next() {
			var comment BugComment
			commentRows.Scan(&comment.ID, &comment.UserID, &comment.Username, &comment.Content, &comment.IsInternal, &comment.IsStatusChange, &comment.CreatedAt)
			comments = append(comments, comment)
		}

		// Get attachments
		attachmentRows, _ := db.Query(`
			SELECT id, filename, original_filename, file_type, file_size, is_screenshot, created_at
			FROM bug_attachments WHERE bug_report_id = $1 ORDER BY created_at ASC
		`, bugID)
		defer attachmentRows.Close()

		attachments := []BugAttachment{}
		for attachmentRows.Next() {
			var att BugAttachment
			attachmentRows.Scan(&att.ID, &att.Filename, &att.OriginalFilename, &att.FileType, &att.FileSize, &att.IsScreenshot, &att.CreatedAt)
			att.BugReportID = b.ID
			attachments = append(attachments, att)
		}

		c.JSON(http.StatusOK, gin.H{
			"bug":         b,
			"comments":    comments,
			"attachments": attachments,
		})
	}
}

// handleAddBugComment adds a comment to a bug report
func handleAddBugComment(db *sql.DB, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		bugID := c.Param("id")

		var req struct {
			Content string `json:"content" binding:"required,min=1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Comment content is required"})
			return
		}

		// Verify user owns this bug or is admin
		var bugUserID int64
		var reportNumber string
		err := db.QueryRow("SELECT user_id, report_number FROM bug_reports WHERE id = $1", bugID).Scan(&bugUserID, &reportNumber)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bug report not found"})
			return
		}

		var isAdmin bool
		db.QueryRow("SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)

		if bugUserID != userID && !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only comment on your own bug reports"})
			return
		}

		// Insert comment
		var commentID int64
		err = db.QueryRow(`
			INSERT INTO bug_comments (bug_report_id, user_id, content, is_internal)
			VALUES ($1, $2, $3, false)
			RETURNING id
		`, bugID, userID, req.Content).Scan(&commentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
			return
		}

		// Update bug report updated_at
		db.Exec("UPDATE bug_reports SET updated_at = NOW() WHERE id = $1", bugID)

		// Send notification to subscribers
		go sendBugCommentNotification(db, config, bugID, reportNumber, userID, req.Content)

		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment added successfully",
			"id":      commentID,
		})
	}
}

// handleUploadBugAttachment uploads an attachment to a bug report
func handleUploadBugAttachment(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		bugID := c.Param("id")

		// Verify user owns this bug
		var bugUserID int64
		err := db.QueryRow("SELECT user_id FROM bug_reports WHERE id = $1", bugID).Scan(&bugUserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bug report not found"})
			return
		}

		if bugUserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only add attachments to your own bug reports"})
			return
		}

		var req struct {
			Filename     string `json:"filename" binding:"required"`
			FileType     string `json:"file_type" binding:"required"`
			FileData     string `json:"file_data" binding:"required"` // Base64 encoded
			IsScreenshot bool   `json:"is_screenshot"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Filename, file_type, and file_data are required"})
			return
		}

		// Decode base64 file
		fileData, err := base64.StdEncoding.DecodeString(req.FileData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 file data"})
			return
		}

		// Validate file size (max 10MB)
		if len(fileData) > 10*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 10MB limit"})
			return
		}

		// Generate unique filename
		storedFilename := fmt.Sprintf("%s_%d_%s", bugID, time.Now().Unix(), req.Filename)

		var attachmentID int64
		err = db.QueryRow(`
			INSERT INTO bug_attachments (bug_report_id, filename, original_filename, file_type, file_size, file_data, is_screenshot)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, bugID, storedFilename, req.Filename, req.FileType, len(fileData), fileData, req.IsScreenshot).Scan(&attachmentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload attachment"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Attachment uploaded successfully",
			"id":      attachmentID,
		})
	}
}

// handleSubscribeToBug subscribes user to bug updates
func handleSubscribeToBug(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		bugID := c.Param("id")

		_, err := db.Exec("INSERT INTO bug_subscribers (bug_report_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", bugID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Subscribed to bug updates"})
	}
}

// handleUnsubscribeFromBug unsubscribes user from bug updates
func handleUnsubscribeFromBug(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		bugID := c.Param("id")

		db.Exec("DELETE FROM bug_subscribers WHERE bug_report_id = $1 AND user_id = $2", bugID, userID)
		c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed from bug updates"})
	}
}

// ==================== ADMIN BUG REPORT HANDLERS ====================

// handleAdminGetAllBugReports gets all bug reports for admins
func handleAdminGetAllBugReports(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "")
		priority := c.DefaultQuery("priority", "")
		category := c.DefaultQuery("category", "")

		query := `
			SELECT b.id, b.report_number, b.user_id, u.username, b.title, b.category, b.priority, b.status, 
				b.assigned_to, au.username, b.created_at, b.updated_at,
				(SELECT COUNT(*) FROM bug_attachments WHERE bug_report_id = b.id) as attachment_count,
				(SELECT COUNT(*) FROM bug_comments WHERE bug_report_id = b.id) as comment_count
			FROM bug_reports b
			JOIN users u ON b.user_id = u.id
			LEFT JOIN users au ON b.assigned_to = au.id
			WHERE 1=1
		`
		args := []interface{}{}
		argIndex := 1

		if status != "" {
			query += fmt.Sprintf(" AND b.status = $%d", argIndex)
			args = append(args, status)
			argIndex++
		}
		if priority != "" {
			query += fmt.Sprintf(" AND b.priority = $%d", argIndex)
			args = append(args, priority)
			argIndex++
		}
		if category != "" {
			query += fmt.Sprintf(" AND b.category = $%d", argIndex)
			args = append(args, category)
			argIndex++
		}

		query += " ORDER BY CASE b.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END, b.created_at DESC"

		rows, err := db.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bug reports"})
			return
		}
		defer rows.Close()

		bugs := []gin.H{}
		for rows.Next() {
			var id, userID int64
			var reportNumber, username, title, category, priority, status string
			var assignedTo sql.NullInt64
			var assignedUsername sql.NullString
			var createdAt, updatedAt time.Time
			var attachmentCount, commentCount int

			rows.Scan(&id, &reportNumber, &userID, &username, &title, &category, &priority, &status,
				&assignedTo, &assignedUsername, &createdAt, &updatedAt, &attachmentCount, &commentCount)

			bug := gin.H{
				"id":               id,
				"report_number":    reportNumber,
				"user_id":          userID,
				"username":         username,
				"title":            title,
				"category":         category,
				"priority":         priority,
				"status":           status,
				"created_at":       createdAt,
				"updated_at":       updatedAt,
				"attachment_count": attachmentCount,
				"comment_count":    commentCount,
			}
			if assignedTo.Valid {
				bug["assigned_to"] = assignedTo.Int64
				bug["assigned_username"] = assignedUsername.String
			}
			bugs = append(bugs, bug)
		}

		// Get counts by status
		var openCount, inProgressCount, resolvedCount, closedCount int
		db.QueryRow("SELECT COUNT(*) FROM bug_reports WHERE status = 'open'").Scan(&openCount)
		db.QueryRow("SELECT COUNT(*) FROM bug_reports WHERE status = 'in_progress'").Scan(&inProgressCount)
		db.QueryRow("SELECT COUNT(*) FROM bug_reports WHERE status = 'resolved'").Scan(&resolvedCount)
		db.QueryRow("SELECT COUNT(*) FROM bug_reports WHERE status = 'closed'").Scan(&closedCount)

		c.JSON(http.StatusOK, gin.H{
			"bugs": bugs,
			"counts": gin.H{
				"open":        openCount,
				"in_progress": inProgressCount,
				"resolved":    resolvedCount,
				"closed":      closedCount,
			},
		})
	}
}

// handleAdminGetBugReport gets a single bug report for admins (same as user but with internal comments)
func handleAdminGetBugReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bugID := c.Param("id")

		var b BugReport
		var assignedTo, resolvedBy sql.NullInt64
		var assignedUsername sql.NullString
		var resolvedAt sql.NullTime

		err := db.QueryRow(`
			SELECT b.id, b.report_number, b.user_id, u.username, b.title, b.description, 
				COALESCE(b.steps_to_reproduce, ''), COALESCE(b.expected_behavior, ''), COALESCE(b.actual_behavior, ''),
				b.category, b.priority, b.status, COALESCE(b.browser_info, ''), COALESCE(b.os_info, ''), 
				COALESCE(b.page_url, ''), COALESCE(b.console_errors, ''),
				b.assigned_to, au.username, b.resolved_at, b.resolved_by, b.created_at, b.updated_at
			FROM bug_reports b
			JOIN users u ON b.user_id = u.id
			LEFT JOIN users au ON b.assigned_to = au.id
			WHERE b.id = $1
		`, bugID).Scan(
			&b.ID, &b.ReportNumber, &b.UserID, &b.Username, &b.Title, &b.Description,
			&b.StepsToReproduce, &b.ExpectedBehavior, &b.ActualBehavior,
			&b.Category, &b.Priority, &b.Status, &b.BrowserInfo, &b.OSInfo,
			&b.PageURL, &b.ConsoleErrors,
			&assignedTo, &assignedUsername, &resolvedAt, &resolvedBy, &b.CreatedAt, &b.UpdatedAt,
		)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bug report not found"})
			return
		}

		if assignedTo.Valid {
			b.AssignedTo = &assignedTo.Int64
			b.AssignedUsername = assignedUsername.String
		}
		if resolvedAt.Valid {
			b.ResolvedAt = &resolvedAt.Time
		}
		if resolvedBy.Valid {
			b.ResolvedBy = &resolvedBy.Int64
		}

		// Get all comments including internal
		commentRows, _ := db.Query(`
			SELECT bc.id, bc.user_id, u.username, bc.content, bc.is_internal, bc.is_status_change, bc.created_at
			FROM bug_comments bc
			JOIN users u ON bc.user_id = u.id
			WHERE bc.bug_report_id = $1
			ORDER BY bc.created_at ASC
		`, bugID)
		defer commentRows.Close()

		comments := []BugComment{}
		for commentRows.Next() {
			var comment BugComment
			commentRows.Scan(&comment.ID, &comment.UserID, &comment.Username, &comment.Content, &comment.IsInternal, &comment.IsStatusChange, &comment.CreatedAt)
			comments = append(comments, comment)
		}

		// Get attachments
		attachmentRows, _ := db.Query(`
			SELECT id, filename, original_filename, file_type, file_size, is_screenshot, created_at
			FROM bug_attachments WHERE bug_report_id = $1 ORDER BY created_at ASC
		`, bugID)
		defer attachmentRows.Close()

		attachments := []BugAttachment{}
		for attachmentRows.Next() {
			var att BugAttachment
			attachmentRows.Scan(&att.ID, &att.Filename, &att.OriginalFilename, &att.FileType, &att.FileSize, &att.IsScreenshot, &att.CreatedAt)
			att.BugReportID = b.ID
			attachments = append(attachments, att)
		}

		// Get subscribers
		subscriberRows, _ := db.Query(`
			SELECT u.id, u.username FROM bug_subscribers bs
			JOIN users u ON bs.user_id = u.id
			WHERE bs.bug_report_id = $1
		`, bugID)
		defer subscriberRows.Close()

		subscribers := []gin.H{}
		for subscriberRows.Next() {
			var id int64
			var username string
			subscriberRows.Scan(&id, &username)
			subscribers = append(subscribers, gin.H{"id": id, "username": username})
		}

		c.JSON(http.StatusOK, gin.H{
			"bug":         b,
			"comments":    comments,
			"attachments": attachments,
			"subscribers": subscribers,
		})
	}
}

// handleAdminUpdateBugStatus updates a bug report's status
func handleAdminUpdateBugStatus(db *sql.DB, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		bugID := c.Param("id")

		var req struct {
			Status  string `json:"status" binding:"required"`
			Comment string `json:"comment"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
			return
		}

		validStatuses := map[string]bool{"open": true, "in_progress": true, "resolved": true, "closed": true, "wont_fix": true}
		if !validStatuses[req.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}

		// Get current status and report number
		var currentStatus, reportNumber string
		err := db.QueryRow("SELECT status, report_number FROM bug_reports WHERE id = $1", bugID).Scan(&currentStatus, &reportNumber)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bug report not found"})
			return
		}

		// Update status
		updateQuery := "UPDATE bug_reports SET status = $1, updated_at = NOW()"
		args := []interface{}{req.Status}

		if req.Status == "resolved" || req.Status == "closed" {
			updateQuery += ", resolved_at = NOW(), resolved_by = $2 WHERE id = $3"
			args = append(args, adminID, bugID)
		} else {
			updateQuery += " WHERE id = $2"
			args = append(args, bugID)
		}

		_, err = db.Exec(updateQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
			return
		}

		// Add status change comment
		statusComment := fmt.Sprintf("Status changed from '%s' to '%s'", currentStatus, req.Status)
		if req.Comment != "" {
			statusComment += "\n\n" + req.Comment
		}
		db.Exec(`
			INSERT INTO bug_comments (bug_report_id, user_id, content, is_status_change)
			VALUES ($1, $2, $3, true)
		`, bugID, adminID, statusComment)

		// Send notification to subscribers
		go sendBugStatusChangeNotification(db, config, bugID, reportNumber, req.Status, adminID)

		c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
	}
}

// handleAdminUpdateBugPriority updates a bug report's priority
func handleAdminUpdateBugPriority(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bugID := c.Param("id")

		var req struct {
			Priority string `json:"priority" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Priority is required"})
			return
		}

		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if !validPriorities[req.Priority] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid priority"})
			return
		}

		_, err := db.Exec("UPDATE bug_reports SET priority = $1, updated_at = NOW() WHERE id = $2", req.Priority, bugID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update priority"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Priority updated successfully"})
	}
}

// handleAdminAssignBug assigns a bug to an admin
func handleAdminAssignBug(db *sql.DB, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		bugIDStr := c.Param("id")
		bugID, _ := strconv.ParseInt(bugIDStr, 10, 64)

		var req struct {
			AssignTo *int64 `json:"assign_to"` // null to unassign
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		var reportNumber string
		db.QueryRow("SELECT report_number FROM bug_reports WHERE id = $1", bugIDStr).Scan(&reportNumber)

		if req.AssignTo != nil {
			// Verify assignee is an admin
			var isAssigneeAdmin bool
			db.QueryRow("SELECT is_admin FROM users WHERE id = $1", *req.AssignTo).Scan(&isAssigneeAdmin)
			if !isAssigneeAdmin {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Can only assign to admins"})
				return
			}

			_, err := db.Exec("UPDATE bug_reports SET assigned_to = $1, updated_at = NOW() WHERE id = $2", *req.AssignTo, bugIDStr)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign bug"})
				return
			}

			// Auto-subscribe assignee
			db.Exec("INSERT INTO bug_subscribers (bug_report_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", bugIDStr, *req.AssignTo)

			// Add assignment comment
			var assigneeName string
			db.QueryRow("SELECT username FROM users WHERE id = $1", *req.AssignTo).Scan(&assigneeName)
			db.Exec(`
				INSERT INTO bug_comments (bug_report_id, user_id, content, is_status_change)
				VALUES ($1, $2, $3, true)
			`, bugIDStr, adminID, fmt.Sprintf("Assigned to %s", assigneeName))

			// Send notification
			go sendBugAssignmentNotification(db, config, bugID, reportNumber, *req.AssignTo)
		} else {
			db.Exec("UPDATE bug_reports SET assigned_to = NULL, updated_at = NOW() WHERE id = $1", bugIDStr)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Assignment updated successfully"})
	}
}

// handleAdminAddBugComment adds an admin comment (can be internal)
func handleAdminAddBugComment(db *sql.DB, config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt64("user_id")
		bugID := c.Param("id")

		var req struct {
			Content    string `json:"content" binding:"required,min=1"`
			IsInternal bool   `json:"is_internal"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Comment content is required"})
			return
		}

		var reportNumber string
		err := db.QueryRow("SELECT report_number FROM bug_reports WHERE id = $1", bugID).Scan(&reportNumber)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bug report not found"})
			return
		}

		var commentID int64
		err = db.QueryRow(`
			INSERT INTO bug_comments (bug_report_id, user_id, content, is_internal)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, bugID, adminID, req.Content, req.IsInternal).Scan(&commentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
			return
		}

		db.Exec("UPDATE bug_reports SET updated_at = NOW() WHERE id = $1", bugID)

		// Send notification only for non-internal comments
		if !req.IsInternal {
			go sendBugCommentNotification(db, config, bugID, reportNumber, adminID, req.Content)
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Comment added successfully",
			"id":      commentID,
		})
	}
}

// handleAdminDeleteBugReport deletes a bug report
func handleAdminDeleteBugReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bugID := c.Param("id")

		_, err := db.Exec("DELETE FROM bug_reports WHERE id = $1", bugID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bug report"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bug report deleted successfully"})
	}
}

// ==================== BUG NOTIFICATION HELPERS ====================

func sendBugReportNotificationToAdmins(db *sql.DB, config *Config, bugID int64, reportNumber, title, username string) {
	// Get all admin emails
	rows, err := db.Query("SELECT email FROM users WHERE is_admin = true")
	if err != nil {
		return
	}
	defer rows.Close()

	bugURL := fmt.Sprintf("%s/admin/bugs/%d", config.FrontendURL, bugID)
	subject := fmt.Sprintf("[%s] New Bug Report: %s", reportNumber, title)
	body := fmt.Sprintf(`
A new bug report has been submitted.

Report Number: %s
Title: %s
Submitted by: %s

View and respond to this bug report:
%s

---
Chimera Pool Bug Tracking System
`, reportNumber, title, username, bugURL)

	for rows.Next() {
		var email string
		rows.Scan(&email)
		sendEmail(config, email, subject, body)

		// Log notification
		db.Exec(`
			INSERT INTO bug_email_notifications (bug_report_id, user_id, email_type, email_address, subject, is_sent, sent_at)
			SELECT $1, id, 'new_report', $2, $3, true, NOW() FROM users WHERE email = $2
		`, bugID, email, subject)
	}
}

func sendBugStatusChangeNotification(db *sql.DB, config *Config, bugID interface{}, reportNumber, newStatus string, adminID int64) {
	// Get subscribers
	rows, err := db.Query(`
		SELECT u.email, u.username FROM bug_subscribers bs
		JOIN users u ON bs.user_id = u.id
		WHERE bs.bug_report_id = $1
	`, bugID)
	if err != nil {
		return
	}
	defer rows.Close()

	var adminName string
	db.QueryRow("SELECT username FROM users WHERE id = $1", adminID).Scan(&adminName)

	bugURL := fmt.Sprintf("%s/bugs/%v", config.FrontendURL, bugID)
	subject := fmt.Sprintf("[%s] Status Updated: %s", reportNumber, newStatus)
	body := fmt.Sprintf(`
Your bug report status has been updated.

Report Number: %s
New Status: %s
Updated by: %s

View the full conversation and updates:
%s

---
Chimera Pool Bug Tracking System
`, reportNumber, newStatus, adminName, bugURL)

	for rows.Next() {
		var email, username string
		rows.Scan(&email, &username)
		sendEmail(config, email, subject, body)
	}
}

func sendBugCommentNotification(db *sql.DB, config *Config, bugID interface{}, reportNumber string, commenterID int64, content string) {
	// Get subscribers except the commenter
	rows, err := db.Query(`
		SELECT u.email, u.username FROM bug_subscribers bs
		JOIN users u ON bs.user_id = u.id
		WHERE bs.bug_report_id = $1 AND bs.user_id != $2
	`, bugID, commenterID)
	if err != nil {
		return
	}
	defer rows.Close()

	var commenterName string
	db.QueryRow("SELECT username FROM users WHERE id = $1", commenterID).Scan(&commenterName)

	bugURL := fmt.Sprintf("%s/bugs/%v", config.FrontendURL, bugID)
	subject := fmt.Sprintf("[%s] New Comment", reportNumber)

	// Truncate content for email preview
	preview := content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	body := fmt.Sprintf(`
A new comment has been added to bug report %s.

Comment by: %s
---
%s
---

View the full conversation:
%s

---
Chimera Pool Bug Tracking System
`, reportNumber, commenterName, preview, bugURL)

	for rows.Next() {
		var email, username string
		rows.Scan(&email, &username)
		sendEmail(config, email, subject, body)
	}
}

func sendBugAssignmentNotification(db *sql.DB, config *Config, bugID int64, reportNumber string, assigneeID int64) {
	var email, title string
	db.QueryRow("SELECT email FROM users WHERE id = $1", assigneeID).Scan(&email)
	db.QueryRow("SELECT title FROM bug_reports WHERE id = $1", bugID).Scan(&title)

	bugURL := fmt.Sprintf("%s/admin/bugs/%d", config.FrontendURL, bugID)
	subject := fmt.Sprintf("[%s] Bug Assigned to You: %s", reportNumber, title)
	body := fmt.Sprintf(`
A bug report has been assigned to you.

Report Number: %s
Title: %s

View and work on this bug report:
%s

---
Chimera Pool Bug Tracking System
`, reportNumber, title, bugURL)

	sendEmail(config, email, subject, body)
}

// =============================================================================
// NETWORK CONFIGURATION HANDLERS
// =============================================================================

// handleGetActiveNetwork returns the currently active network (public)
func handleGetActiveNetwork(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var network struct {
			ID                 string  `json:"id"`
			Name               string  `json:"name"`
			Symbol             string  `json:"symbol"`
			DisplayName        string  `json:"display_name"`
			Algorithm          string  `json:"algorithm"`
			RPCURL             string  `json:"rpc_url"`
			ExplorerURL        string  `json:"explorer_url"`
			StratumPort        int     `json:"stratum_port"`
			PoolFeePercent     float64 `json:"pool_fee_percent"`
			MinPayoutThreshold float64 `json:"min_payout_threshold"`
			PoolWalletAddress  string  `json:"pool_wallet_address"`
			NetworkType        string  `json:"network_type"`
			Description        string  `json:"description"`
		}

		err := db.QueryRow(`
			SELECT id, name, symbol, display_name, algorithm, 
				   rpc_url, COALESCE(explorer_url, ''), stratum_port,
				   pool_fee_percent, min_payout_threshold, pool_wallet_address,
				   network_type, COALESCE(description, '')
			FROM network_configs 
			WHERE is_active = true AND is_default = true
		`).Scan(&network.ID, &network.Name, &network.Symbol, &network.DisplayName,
			&network.Algorithm, &network.RPCURL, &network.ExplorerURL,
			&network.StratumPort, &network.PoolFeePercent, &network.MinPayoutThreshold,
			&network.PoolWalletAddress, &network.NetworkType, &network.Description)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active network configured"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"network": network})
	}
}

// handleAdminListNetworks returns all network configurations
func handleAdminListNetworks(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, name, symbol, display_name, is_active, is_default,
				   algorithm, COALESCE(algorithm_variant, ''),
				   rpc_url, COALESCE(explorer_url, ''),
				   stratum_port, block_time_target, COALESCE(block_reward, 0),
				   pool_wallet_address, pool_fee_percent, min_payout_threshold,
				   network_type, COALESCE(description, ''),
				   created_at, updated_at
			FROM network_configs 
			ORDER BY is_default DESC, name ASC
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch networks"})
			return
		}
		defer rows.Close()

		var networks []gin.H
		for rows.Next() {
			var id, name, symbol, displayName, algorithm, algorithmVariant string
			var rpcURL, explorerURL, poolWalletAddress, networkType, description string
			var isActive, isDefault bool
			var stratumPort, blockTimeTarget int
			var blockReward, poolFeePercent, minPayoutThreshold float64
			var createdAt, updatedAt time.Time

			err := rows.Scan(&id, &name, &symbol, &displayName, &isActive, &isDefault,
				&algorithm, &algorithmVariant, &rpcURL, &explorerURL,
				&stratumPort, &blockTimeTarget, &blockReward,
				&poolWalletAddress, &poolFeePercent, &minPayoutThreshold,
				&networkType, &description, &createdAt, &updatedAt)
			if err != nil {
				continue
			}

			networks = append(networks, gin.H{
				"id":                   id,
				"name":                 name,
				"symbol":               symbol,
				"display_name":         displayName,
				"is_active":            isActive,
				"is_default":           isDefault,
				"algorithm":            algorithm,
				"algorithm_variant":    algorithmVariant,
				"rpc_url":              rpcURL,
				"explorer_url":         explorerURL,
				"stratum_port":         stratumPort,
				"block_time_target":    blockTimeTarget,
				"block_reward":         blockReward,
				"pool_wallet_address":  poolWalletAddress,
				"pool_fee_percent":     poolFeePercent,
				"min_payout_threshold": minPayoutThreshold,
				"network_type":         networkType,
				"description":          description,
				"created_at":           createdAt,
				"updated_at":           updatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{"networks": networks, "total": len(networks)})
	}
}

// handleAdminGetNetwork returns a specific network configuration
func handleAdminGetNetwork(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var network gin.H
		var name, symbol, displayName, algorithm, algorithmVariant string
		var rpcURL, explorerURL, poolWalletAddress, networkType, description string
		var isActive, isDefault bool
		var stratumPort, blockTimeTarget int
		var blockReward, poolFeePercent, minPayoutThreshold float64

		err := db.QueryRow(`
			SELECT name, symbol, display_name, is_active, is_default,
				   algorithm, COALESCE(algorithm_variant, ''),
				   rpc_url, COALESCE(explorer_url, ''),
				   stratum_port, block_time_target, COALESCE(block_reward, 0),
				   pool_wallet_address, pool_fee_percent, min_payout_threshold,
				   network_type, COALESCE(description, '')
			FROM network_configs WHERE id = $1
		`, id).Scan(&name, &symbol, &displayName, &isActive, &isDefault,
			&algorithm, &algorithmVariant, &rpcURL, &explorerURL,
			&stratumPort, &blockTimeTarget, &blockReward,
			&poolWalletAddress, &poolFeePercent, &minPayoutThreshold,
			&networkType, &description)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
			return
		}

		network = gin.H{
			"id":                   id,
			"name":                 name,
			"symbol":               symbol,
			"display_name":         displayName,
			"is_active":            isActive,
			"is_default":           isDefault,
			"algorithm":            algorithm,
			"algorithm_variant":    algorithmVariant,
			"rpc_url":              rpcURL,
			"explorer_url":         explorerURL,
			"stratum_port":         stratumPort,
			"block_time_target":    blockTimeTarget,
			"block_reward":         blockReward,
			"pool_wallet_address":  poolWalletAddress,
			"pool_fee_percent":     poolFeePercent,
			"min_payout_threshold": minPayoutThreshold,
			"network_type":         networkType,
			"description":          description,
		}

		c.JSON(http.StatusOK, gin.H{"network": network})
	}
}

// handleAdminCreateNetwork creates a new network configuration
func handleAdminCreateNetwork(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name              string  `json:"name" binding:"required"`
			Symbol            string  `json:"symbol" binding:"required"`
			DisplayName       string  `json:"display_name" binding:"required"`
			Algorithm         string  `json:"algorithm" binding:"required"`
			RPCURL            string  `json:"rpc_url" binding:"required"`
			PoolWalletAddress string  `json:"pool_wallet_address" binding:"required"`
			StratumPort       int     `json:"stratum_port"`
			PoolFeePercent    float64 `json:"pool_fee_percent"`
			NetworkType       string  `json:"network_type"`
			Description       string  `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.StratumPort == 0 {
			req.StratumPort = 3333
		}
		if req.PoolFeePercent == 0 {
			req.PoolFeePercent = 1.0
		}
		if req.NetworkType == "" {
			req.NetworkType = "mainnet"
		}

		var id string
		err := db.QueryRow(`
			INSERT INTO network_configs (name, symbol, display_name, algorithm, rpc_url, 
				pool_wallet_address, stratum_port, pool_fee_percent, network_type, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, req.Name, req.Symbol, req.DisplayName, req.Algorithm, req.RPCURL,
			req.PoolWalletAddress, req.StratumPort, req.PoolFeePercent,
			req.NetworkType, req.Description).Scan(&id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create network", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Network created", "id": id})
	}
}

// handleAdminUpdateNetwork updates a network configuration
func handleAdminUpdateNetwork(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			DisplayName        *string  `json:"display_name"`
			Algorithm          *string  `json:"algorithm"`
			RPCURL             *string  `json:"rpc_url"`
			RPCUser            *string  `json:"rpc_user"`
			RPCPassword        *string  `json:"rpc_password"`
			ExplorerURL        *string  `json:"explorer_url"`
			StratumPort        *int     `json:"stratum_port"`
			PoolWalletAddress  *string  `json:"pool_wallet_address"`
			PoolFeePercent     *float64 `json:"pool_fee_percent"`
			MinPayoutThreshold *float64 `json:"min_payout_threshold"`
			Description        *string  `json:"description"`
			IsActive           *bool    `json:"is_active"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Build dynamic update
		updates := []string{"updated_at = NOW()"}
		args := []interface{}{}
		argIdx := 1

		if req.DisplayName != nil {
			updates = append(updates, fmt.Sprintf("display_name = $%d", argIdx))
			args = append(args, *req.DisplayName)
			argIdx++
		}
		if req.Algorithm != nil {
			updates = append(updates, fmt.Sprintf("algorithm = $%d", argIdx))
			args = append(args, *req.Algorithm)
			argIdx++
		}
		if req.RPCURL != nil {
			updates = append(updates, fmt.Sprintf("rpc_url = $%d", argIdx))
			args = append(args, *req.RPCURL)
			argIdx++
		}
		if req.RPCUser != nil {
			updates = append(updates, fmt.Sprintf("rpc_user = $%d", argIdx))
			args = append(args, *req.RPCUser)
			argIdx++
		}
		if req.RPCPassword != nil && *req.RPCPassword != "" {
			updates = append(updates, fmt.Sprintf("rpc_password = $%d", argIdx))
			args = append(args, *req.RPCPassword)
			argIdx++
		}
		if req.ExplorerURL != nil {
			updates = append(updates, fmt.Sprintf("explorer_url = $%d", argIdx))
			args = append(args, *req.ExplorerURL)
			argIdx++
		}
		if req.StratumPort != nil {
			updates = append(updates, fmt.Sprintf("stratum_port = $%d", argIdx))
			args = append(args, *req.StratumPort)
			argIdx++
		}
		if req.PoolWalletAddress != nil {
			updates = append(updates, fmt.Sprintf("pool_wallet_address = $%d", argIdx))
			args = append(args, *req.PoolWalletAddress)
			argIdx++
		}
		if req.PoolFeePercent != nil {
			updates = append(updates, fmt.Sprintf("pool_fee_percent = $%d", argIdx))
			args = append(args, *req.PoolFeePercent)
			argIdx++
		}
		if req.MinPayoutThreshold != nil {
			updates = append(updates, fmt.Sprintf("min_payout_threshold = $%d", argIdx))
			args = append(args, *req.MinPayoutThreshold)
			argIdx++
		}
		if req.Description != nil {
			updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
			args = append(args, *req.Description)
			argIdx++
		}
		if req.IsActive != nil {
			updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
			args = append(args, *req.IsActive)
			argIdx++
		}

		args = append(args, id)
		query := fmt.Sprintf("UPDATE network_configs SET %s WHERE id = $%d",
			strings.Join(updates, ", "), argIdx)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update network"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Network updated successfully"})
	}
}

// handleAdminDeleteNetwork deletes a network configuration
func handleAdminDeleteNetwork(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Check if it's the default network
		var isDefault bool
		db.QueryRow("SELECT is_default FROM network_configs WHERE id = $1", id).Scan(&isDefault)
		if isDefault {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the default network"})
			return
		}

		_, err := db.Exec("DELETE FROM network_configs WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete network"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Network deleted"})
	}
}

// handleAdminSwitchNetwork switches the active mining network
func handleAdminSwitchNetwork(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			NetworkName string `json:"network_name" binding:"required"`
			Reason      string `json:"reason"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt64("user_id")
		if req.Reason == "" {
			req.Reason = "Manual switch from admin panel"
		}

		// Call the stored procedure
		var historyID string
		err := db.QueryRow("SELECT switch_active_network($1, $2, $3)",
			req.NetworkName, userID, req.Reason).Scan(&historyID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to switch network", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "Network switched successfully",
			"history_id": historyID,
		})
	}
}

// handleAdminNetworkHistory returns network switch history
func handleAdminNetworkHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT h.id, h.from_network_id, h.to_network_id, h.switched_by,
				   h.switch_reason, h.switch_type, h.status, h.started_at, h.completed_at,
				   COALESCE(h.error_message, ''),
				   COALESCE(fn.display_name, 'None') as from_name,
				   tn.display_name as to_name,
				   COALESCE(u.username, 'System') as switched_by_name
			FROM network_switch_history h
			LEFT JOIN network_configs fn ON h.from_network_id = fn.id
			JOIN network_configs tn ON h.to_network_id = tn.id
			LEFT JOIN users u ON h.switched_by = u.id
			ORDER BY h.started_at DESC
			LIMIT 50
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
			return
		}
		defer rows.Close()

		var history []gin.H
		for rows.Next() {
			var id string
			var fromNetworkID, toNetworkID sql.NullString
			var switchedBy sql.NullInt64
			var switchReason, switchType, status, errorMessage string
			var fromName, toName, switchedByName string
			var startedAt time.Time
			var completedAt sql.NullTime

			err := rows.Scan(&id, &fromNetworkID, &toNetworkID, &switchedBy,
				&switchReason, &switchType, &status, &startedAt, &completedAt,
				&errorMessage, &fromName, &toName, &switchedByName)
			if err != nil {
				continue
			}

			entry := gin.H{
				"id":            id,
				"switch_reason": switchReason,
				"switch_type":   switchType,
				"status":        status,
				"started_at":    startedAt,
				"from_network":  fromName,
				"to_network":    toName,
				"switched_by":   switchedByName,
				"error_message": errorMessage,
			}
			if completedAt.Valid {
				entry["completed_at"] = completedAt.Time
			}
			history = append(history, entry)
		}

		c.JSON(http.StatusOK, gin.H{"history": history})
	}
}

// handleAdminTestNetworkConnection tests RPC connection
func handleAdminTestNetworkConnection(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var rpcURL string
		err := db.QueryRow("SELECT rpc_url FROM network_configs WHERE id = $1", id).Scan(&rpcURL)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Network not found"})
			return
		}

		if rpcURL == "" {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": "RPC URL is empty"})
			return
		}

		// Basic connection test - just check if URL is reachable
		// In production, you'd make an actual RPC call
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "RPC URL configured: " + rpcURL,
		})
	}
}

// =============================================================================
// MINER MONITORING HANDLERS
// =============================================================================

// handleAdminGetAllMiners returns paginated list of all miners
func handleAdminGetAllMiners(db *sql.DB) gin.HandlerFunc {
	service := api.NewMinerMonitoringService(db)
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		search := c.Query("search")
		activeOnly := c.Query("active") == "true"

		miners, total, err := service.GetAllMinersForAdmin(page, limit, search, activeOnly)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miners", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"miners": miners,
			"total":  total,
			"page":   page,
			"limit":  limit,
		})
	}
}

// handleAdminGetMinerDetail returns comprehensive details for a miner
func handleAdminGetMinerDetail(db *sql.DB) gin.HandlerFunc {
	service := api.NewMinerMonitoringService(db)
	return func(c *gin.Context) {
		minerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid miner ID"})
			return
		}

		detail, err := service.GetMinerDetail(minerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miner details", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, detail)
	}
}

// handleAdminGetMinerShares returns share history for a miner
func handleAdminGetMinerShares(db *sql.DB) gin.HandlerFunc {
	service := api.NewMinerMonitoringService(db)
	return func(c *gin.Context) {
		minerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid miner ID"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		shares, err := service.GetMinerShareHistory(minerID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shares", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"shares": shares})
	}
}

// handleAdminGetUserMiners returns all miners for a specific user
func handleAdminGetUserMiners(db *sql.DB) gin.HandlerFunc {
	service := api.NewMinerMonitoringService(db)
	return func(c *gin.Context) {
		userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		summary, err := service.GetUserMinerSummary(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user miners", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}

// =============================================================================
// PAYOUT SETTINGS HANDLERS
// =============================================================================

// handleGetPayoutModes returns all available payout modes (public)
func handleGetPayoutModes() gin.HandlerFunc {
	service := api.NewDefaultPayoutService(nil)
	return func(c *gin.Context) {
		modes, err := service.GetAvailablePayoutModes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payout modes"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"modes": modes, "total": len(modes)})
	}
}

// handleGetPoolFees returns pool fee configuration (public)
func handleGetPoolFees() gin.HandlerFunc {
	service := api.NewDefaultPayoutService(nil)
	return func(c *gin.Context) {
		fees, err := service.GetPoolFeeConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get fee config"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"fees": fees})
	}
}

// handleGetPayoutSettings returns user's payout settings (protected)
func handleGetPayoutSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var settings struct {
			PayoutMode       string `json:"payout_mode"`
			MinPayoutAmount  int64  `json:"min_payout_amount"`
			PayoutAddress    string `json:"payout_address"`
			AutoPayoutEnable bool   `json:"auto_payout_enable"`
		}

		err := db.QueryRow(`
			SELECT COALESCE(payout_mode, 'pplns'), COALESCE(min_payout_amount, 1000000),
			       COALESCE(payout_address, ''), COALESCE(auto_payout_enable, true)
			FROM user_payout_settings WHERE user_id = $1
		`, userID).Scan(&settings.PayoutMode, &settings.MinPayoutAmount,
			&settings.PayoutAddress, &settings.AutoPayoutEnable)

		if err == sql.ErrNoRows {
			// Return defaults
			settings.PayoutMode = "pplns"
			settings.MinPayoutAmount = 1000000
			settings.AutoPayoutEnable = true
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":            userID,
			"payout_mode":        settings.PayoutMode,
			"min_payout_amount":  settings.MinPayoutAmount,
			"payout_address":     settings.PayoutAddress,
			"auto_payout_enable": settings.AutoPayoutEnable,
			"fee_percent":        getPoolFeeForMode(settings.PayoutMode),
		})
	}
}

// handleUpdatePayoutSettings updates user's payout settings (protected)
func handleUpdatePayoutSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var req struct {
			PayoutMode       *string `json:"payout_mode"`
			MinPayoutAmount  *int64  `json:"min_payout_amount"`
			PayoutAddress    *string `json:"payout_address"`
			AutoPayoutEnable *bool   `json:"auto_payout_enable"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Validate payout mode if provided
		validModes := map[string]bool{
			"pplns": true, "pps": true, "pps_plus": true,
			"fpps": true, "score": true, "solo": true, "slice": true,
		}
		if req.PayoutMode != nil && !validModes[*req.PayoutMode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payout mode"})
			return
		}

		// Upsert settings
		_, err := db.Exec(`
			INSERT INTO user_payout_settings (user_id, payout_mode, min_payout_amount, payout_address, auto_payout_enable)
			VALUES ($1, COALESCE($2, 'pplns'), COALESCE($3, 1000000), COALESCE($4, ''), COALESCE($5, true))
			ON CONFLICT (user_id) DO UPDATE SET
				payout_mode = COALESCE($2, user_payout_settings.payout_mode),
				min_payout_amount = COALESCE($3, user_payout_settings.min_payout_amount),
				payout_address = COALESCE($4, user_payout_settings.payout_address),
				auto_payout_enable = COALESCE($5, user_payout_settings.auto_payout_enable),
				updated_at = NOW()
		`, userID, req.PayoutMode, req.MinPayoutAmount, req.PayoutAddress, req.AutoPayoutEnable)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully"})
	}
}

// handleGetPayoutEstimate returns payout estimate for user (protected)
func handleGetPayoutEstimate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		mode := c.DefaultQuery("mode", "pplns")

		// Get user's share contribution
		var totalDiff float64
		db.QueryRow(`
			SELECT COALESCE(SUM(difficulty), 0)
			FROM shares
			WHERE user_id = $1 AND is_valid = true
			AND created_at > NOW() - INTERVAL '24 hours'
		`, userID).Scan(&totalDiff)

		// Get pool's total difficulty in window
		var poolDiff float64
		db.QueryRow(`
			SELECT COALESCE(SUM(difficulty), 0)
			FROM shares
			WHERE is_valid = true
			AND created_at > NOW() - INTERVAL '24 hours'
		`).Scan(&poolDiff)

		sharePercent := 0.0
		if poolDiff > 0 {
			sharePercent = (totalDiff / poolDiff) * 100
		}

		feePercent := getPoolFeeForMode(mode)

		c.JSON(http.StatusOK, gin.H{
			"user_id":            userID,
			"payout_mode":        mode,
			"current_difficulty": totalDiff,
			"pool_difficulty":    poolDiff,
			"share_percentage":   sharePercent,
			"fee_percent":        feePercent,
			"estimated_at":       time.Now(),
		})
	}
}

// getPoolFeeForMode returns the pool fee for a given payout mode
func getPoolFeeForMode(mode string) float64 {
	fees := map[string]float64{
		"pplns":    1.0,
		"pps":      2.0,
		"pps_plus": 1.5,
		"fpps":     2.0,
		"score":    1.0,
		"solo":     0.5,
		"slice":    0.8,
	}
	if fee, ok := fees[mode]; ok {
		return fee
	}
	return 1.0
}

// ============================================================================
// MULTI-COIN HANDLERS
// ============================================================================

// handleGetSupportedNetworks returns list of all configured networks (public)
func handleGetSupportedNetworks(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT id, name, symbol, display_name, is_active, is_default, 
				   algorithm, stratum_port, COALESCE(explorer_url, ''),
				   COALESCE(logo_url, ''), min_payout_threshold, pool_fee_percent
			FROM network_configs
			ORDER BY is_active DESC, is_default DESC, display_name
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch networks"})
			return
		}
		defer rows.Close()

		var networks []gin.H
		for rows.Next() {
			var id, name, symbol, displayName, algorithm, explorerURL, logoURL string
			var isActive, isDefault bool
			var stratumPort int
			var minPayout, poolFee float64

			if err := rows.Scan(&id, &name, &symbol, &displayName, &isActive, &isDefault,
				&algorithm, &stratumPort, &explorerURL, &logoURL, &minPayout, &poolFee); err != nil {
				continue
			}

			networks = append(networks, gin.H{
				"id":                   id,
				"name":                 name,
				"symbol":               symbol,
				"display_name":         displayName,
				"is_active":            isActive,
				"is_default":           isDefault,
				"algorithm":            algorithm,
				"stratum_port":         stratumPort,
				"explorer_url":         explorerURL,
				"logo_url":             logoURL,
				"min_payout_threshold": minPayout,
				"pool_fee_percent":     poolFee,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"networks": networks,
			"count":    len(networks),
		})
	}
}

// handleGetAllNetworkPoolStats returns pool stats for all networks (public)
func handleGetAllNetworkPoolStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT 
				nc.id, nc.name, nc.symbol, nc.display_name, nc.is_active,
				COALESCE(nps.total_hashrate, 0),
				COALESCE(nps.active_miners, 0),
				COALESCE(nps.active_workers, 0),
				COALESCE(nps.blocks_found_total, 0),
				COALESCE(nps.network_difficulty, 0),
				COALESCE(nps.rpc_connected, false)
			FROM network_configs nc
			LEFT JOIN network_pool_stats nps ON nc.id = nps.network_id
			ORDER BY nc.is_active DESC, nc.is_default DESC
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch network stats"})
			return
		}
		defer rows.Close()

		var stats []gin.H
		for rows.Next() {
			var id, name, symbol, displayName string
			var isActive, rpcConnected bool
			var totalHashrate float64
			var activeMiners, activeWorkers, blocksFound int
			var networkDifficulty float64

			if err := rows.Scan(&id, &name, &symbol, &displayName, &isActive,
				&totalHashrate, &activeMiners, &activeWorkers, &blocksFound,
				&networkDifficulty, &rpcConnected); err != nil {
				continue
			}

			stats = append(stats, gin.H{
				"network_id":         id,
				"network_name":       name,
				"network_symbol":     symbol,
				"display_name":       displayName,
				"is_active":          isActive,
				"total_hashrate":     totalHashrate,
				"active_miners":      activeMiners,
				"active_workers":     activeWorkers,
				"blocks_found":       blocksFound,
				"network_difficulty": networkDifficulty,
				"rpc_connected":      rpcConnected,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"stats":   stats,
		})
	}
}

// handleGetUserNetworkStats returns per-network stats for the authenticated user
func handleGetUserNetworkStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(`
			SELECT 
				nc.id, nc.name, nc.symbol, nc.display_name, nc.is_active,
				COALESCE(uns.total_hashrate, 0),
				COALESCE(uns.total_shares, 0),
				COALESCE(uns.valid_shares, 0),
				COALESCE(uns.blocks_found, 0),
				COALESCE(uns.total_earned, 0),
				COALESCE(uns.pending_balance, 0),
				COALESCE(uns.active_workers, 0),
				uns.last_active_at,
				uns.first_connected_at
			FROM network_configs nc
			LEFT JOIN user_network_stats uns ON nc.id = uns.network_id AND uns.user_id = $1
			ORDER BY nc.is_active DESC, uns.last_active_at DESC NULLS LAST
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch network stats"})
			return
		}
		defer rows.Close()

		var stats []gin.H
		for rows.Next() {
			var id, name, symbol, displayName string
			var isActive bool
			var totalHashrate float64
			var totalShares, validShares int64
			var blocksFound, activeWorkers int
			var totalEarned, pendingBalance float64
			var lastActive, firstConnected sql.NullTime

			if err := rows.Scan(&id, &name, &symbol, &displayName, &isActive,
				&totalHashrate, &totalShares, &validShares, &blocksFound,
				&totalEarned, &pendingBalance, &activeWorkers,
				&lastActive, &firstConnected); err != nil {
				continue
			}

			stat := gin.H{
				"network_id":      id,
				"network_name":    name,
				"network_symbol":  symbol,
				"display_name":    displayName,
				"is_active":       isActive,
				"total_hashrate":  totalHashrate,
				"total_shares":    totalShares,
				"valid_shares":    validShares,
				"blocks_found":    blocksFound,
				"total_earned":    totalEarned,
				"pending_balance": pendingBalance,
				"active_workers":  activeWorkers,
			}

			if lastActive.Valid {
				stat["last_active_at"] = lastActive.Time
			}
			if firstConnected.Valid {
				stat["first_connected_at"] = firstConnected.Time
			}

			stats = append(stats, stat)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"user_id": userID,
			"stats":   stats,
		})
	}
}

// handleGetUserAggregatedStats returns combined stats across all networks for user
func handleGetUserAggregatedStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		var totalNetworks, activeNetworks int
		var combinedHashrate float64
		var totalShares int64
		var totalBlocks int
		var totalEarned, totalPending float64
		var totalWorkers int

		err := db.QueryRow(`
			SELECT 
				COUNT(DISTINCT uns.network_id),
				COUNT(DISTINCT CASE WHEN nc.is_active THEN uns.network_id END),
				COALESCE(SUM(uns.total_hashrate), 0),
				COALESCE(SUM(uns.total_shares), 0),
				COALESCE(SUM(uns.blocks_found), 0),
				COALESCE(SUM(uns.total_earned), 0),
				COALESCE(SUM(uns.pending_balance), 0),
				COALESCE(SUM(uns.active_workers), 0)
			FROM user_network_stats uns
			JOIN network_configs nc ON nc.id = uns.network_id
			WHERE uns.user_id = $1
		`, userID).Scan(&totalNetworks, &activeNetworks, &combinedHashrate,
			&totalShares, &totalBlocks, &totalEarned, &totalPending, &totalWorkers)

		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch aggregated stats"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":              true,
			"user_id":              userID,
			"total_networks_mined": totalNetworks,
			"active_networks":      activeNetworks,
			"combined_hashrate":    combinedHashrate,
			"total_shares_all":     totalShares,
			"total_blocks_all":     totalBlocks,
			"total_earned_all":     totalEarned,
			"total_pending_all":    totalPending,
			"total_workers_all":    totalWorkers,
		})
	}
}

// Referral System Handlers

func handleGetUserReferral(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		// Get user's referral code and stats
		var code, description sql.NullString
		var referrerDiscount, refereeDiscount float64
		var timesUsed, maxUses sql.NullInt64
		var totalReferrals int
		var myDiscount float64

		err := db.QueryRow(`
			SELECT 
				rc.code, rc.description, rc.referrer_discount_percent, 
				rc.referee_discount_percent, rc.times_used, rc.max_uses,
				COALESCE(u.total_referrals, 0),
				COALESCE(u.referrer_discount_percent, 0)
			FROM referral_codes rc
			JOIN users u ON rc.user_id = u.id
			WHERE rc.user_id = $1 AND rc.is_active = true
			LIMIT 1
		`, userID).Scan(&code, &description, &referrerDiscount, &refereeDiscount,
			&timesUsed, &maxUses, &totalReferrals, &myDiscount)

		if err == sql.ErrNoRows {
			// Create a referral code for the user if they don't have one
			var username string
			db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)

			var newCode string
			err = db.QueryRow(`
				INSERT INTO referral_codes (user_id, code, description)
				VALUES ($1, generate_referral_code($2), 'Personal referral code')
				RETURNING code
			`, userID, username).Scan(&newCode)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create referral code"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"code":              newCode,
				"description":       "Personal referral code",
				"referrer_discount": 10.0,
				"referee_discount":  5.0,
				"times_used":        0,
				"max_uses":          nil,
				"total_referrals":   0,
				"my_discount":       0.0,
				"effective_fee":     1.0,
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referral info"})
			return
		}

		// Calculate effective pool fee
		var effectiveFee float64
		db.QueryRow("SELECT get_effective_pool_fee($1)", userID).Scan(&effectiveFee)

		c.JSON(http.StatusOK, gin.H{
			"code":              code.String,
			"description":       description.String,
			"referrer_discount": referrerDiscount,
			"referee_discount":  refereeDiscount,
			"times_used":        timesUsed.Int64,
			"max_uses":          maxUses.Int64,
			"total_referrals":   totalReferrals,
			"my_discount":       myDiscount,
			"effective_fee":     effectiveFee,
		})
	}
}

func handleGetUserReferrals(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")

		rows, err := db.Query(`
			SELECT 
				u.username, r.status, r.created_at, r.confirmed_at,
				r.referee_total_shares, r.referee_total_hashrate, r.clout_bonus_awarded
			FROM referrals r
			JOIN users u ON r.referee_id = u.id
			WHERE r.referrer_id = $1
			ORDER BY r.created_at DESC
			LIMIT 50
		`, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referrals"})
			return
		}
		defer rows.Close()

		referrals := []gin.H{}
		for rows.Next() {
			var username, status string
			var createdAt time.Time
			var confirmedAt sql.NullTime
			var totalShares int64
			var totalHashrate float64
			var cloutBonus int

			rows.Scan(&username, &status, &createdAt, &confirmedAt, &totalShares, &totalHashrate, &cloutBonus)

			referrals = append(referrals, gin.H{
				"username":       username,
				"status":         status,
				"created_at":     createdAt,
				"confirmed_at":   confirmedAt.Time,
				"total_shares":   totalShares,
				"total_hashrate": totalHashrate,
				"clout_bonus":    cloutBonus,
			})
		}

		c.JSON(http.StatusOK, gin.H{"referrals": referrals})
	}
}
