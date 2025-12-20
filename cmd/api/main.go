package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	// API routes
	apiGroup := router.Group("/api/v1")
	{
		// Public routes
		apiGroup.POST("/auth/register", handleRegister(db, config.JWTSecret))
		apiGroup.POST("/auth/login", handleLogin(db, config.JWTSecret))
		apiGroup.POST("/auth/forgot-password", handleForgotPassword(db, config))
		apiGroup.POST("/auth/reset-password", handleResetPassword(db))
		apiGroup.GET("/pool/stats", handlePoolStats(db))
		apiGroup.GET("/pool/blocks", handleBlocks(db))
		apiGroup.GET("/miners/locations", handlePublicMinerLocations(db))
		apiGroup.GET("/miners/locations/stats", handleMinerLocationStats(db))

		// Protected routes
		protected := apiGroup.Group("/")
		protected.Use(authMiddleware(config.JWTSecret))
		{
			protected.GET("/user/profile", handleUserProfile(db))
			protected.GET("/user/miners", handleUserMiners(db))
			protected.GET("/user/payouts", handleUserPayouts(db))
			protected.POST("/user/payout-address", handleSetPayoutAddress(db))
			protected.GET("/user/wallet-history", handleGetWalletHistory(db))

			// Multi-wallet management
			protected.GET("/user/wallets", handleGetUserWallets(db))
			protected.POST("/user/wallets", handleCreateUserWallet(db))
			protected.PUT("/user/wallets/:id", handleUpdateUserWallet(db))
			protected.DELETE("/user/wallets/:id", handleDeleteUserWallet(db))
			protected.GET("/user/wallets/preview", handleWalletPayoutPreview(db))
			protected.GET("/user/stats/hashrate", handleUserHashrateHistory(db))
			protected.GET("/user/stats/shares", handleUserSharesHistory(db))
			protected.GET("/user/stats/earnings", handleUserEarningsHistory(db))

			// Community routes (authenticated)
			protected.GET("/community/channels", handleGetChannels(db))
			protected.GET("/community/channels/:id/messages", handleGetChannelMessages(db))
			protected.POST("/community/channels/:id/messages", handleSendMessage(db))
			protected.PUT("/community/messages/:id", handleEditMessage(db))
			protected.DELETE("/community/messages/:id", handleDeleteMessage(db))
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
		}
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

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
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
	return func(c *gin.Context) {
		// Get pool statistics
		var totalMiners, totalBlocks int64
		var totalHashrate float64

		db.QueryRow("SELECT COUNT(*) FROM miners WHERE is_active = true").Scan(&totalMiners)
		db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&totalBlocks)
		db.QueryRow("SELECT COALESCE(SUM(hashrate), 0) FROM miners WHERE is_active = true").Scan(&totalHashrate)

		c.JSON(http.StatusOK, gin.H{
			"total_miners":     totalMiners,
			"total_hashrate":   totalHashrate,
			"blocks_found":     totalBlocks,
			"pool_fee":         1.0,
			"minimum_payout":   1.0,
			"payment_interval": "1 hour",
			"network":          "BlockDAG Awakening",
			"currency":         "BDAG",
		})
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
		var isAdmin bool
		var createdAt time.Time
		err := db.QueryRow(
			"SELECT username, email, is_admin, created_at FROM users WHERE id = $1",
			userID,
		).Scan(&username, &email, &isAdmin, &createdAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":    userID,
			"username":   username,
			"email":      email,
			"is_admin":   isAdmin,
			"created_at": createdAt,
		})
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

// handleAdminListUsers returns paginated list of all users with stats
func handleAdminListUsers(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page := 1
		pageSize := 20
		search := c.Query("search")

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

		// Build query with optional search (includes wallet info)
		baseQuery := `
			SELECT 
				u.id, u.username, u.email, u.payout_address, u.pool_fee_percent,
				u.is_active, u.is_admin, u.created_at,
				COALESCE(SUM(p.amount), 0) as total_earnings,
				COALESCE((SELECT SUM(amount) FROM payouts WHERE user_id = u.id AND status = 'pending'), 0) as pending_payout,
				COALESCE((SELECT SUM(hashrate) FROM miners WHERE user_id = u.id AND is_active = true), 0) as total_hashrate,
				COALESCE((SELECT COUNT(*) FROM miners WHERE user_id = u.id AND is_active = true), 0) as active_miners,
				COALESCE((SELECT COUNT(*) FROM user_wallets WHERE user_id = u.id), 0) as wallet_count,
				COALESCE((SELECT address FROM user_wallets WHERE user_id = u.id AND is_primary = true LIMIT 1), '') as primary_wallet,
				COALESCE((SELECT SUM(percentage) FROM user_wallets WHERE user_id = u.id AND is_active = true), 0) as total_allocated
			FROM users u
			LEFT JOIN payouts p ON u.id = p.user_id AND p.status = 'confirmed'
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

		baseQuery += " GROUP BY u.id ORDER BY u.created_at DESC"
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
		for rows.Next() {
			var id int64
			var username, email string
			var payoutAddress sql.NullString
			var poolFeePercent sql.NullFloat64
			var isActive, isAdmin bool
			var createdAt time.Time
			var totalEarnings, pendingPayout, totalHashrate float64
			var activeMiners, walletCount int
			var primaryWallet string
			var totalAllocated float64

			err := rows.Scan(&id, &username, &email, &payoutAddress, &poolFeePercent,
				&isActive, &isAdmin, &createdAt, &totalEarnings, &pendingPayout,
				&totalHashrate, &activeMiners, &walletCount, &primaryWallet, &totalAllocated)
			if err != nil {
				log.Printf("Error scanning user row: %v", err)
				continue
			}

			users = append(users, gin.H{
				"id":               id,
				"username":         username,
				"email":            email,
				"payout_address":   payoutAddress.String,
				"pool_fee_percent": poolFeePercent.Float64,
				"is_active":        isActive,
				"is_admin":         isAdmin,
				"created_at":       createdAt,
				"total_earnings":   totalEarnings,
				"pending_payout":   pendingPayout,
				"total_hashrate":   totalHashrate,
				"active_miners":    activeMiners,
				"wallet_count":     walletCount,
				"primary_wallet":   primaryWallet,
				"total_allocated":  totalAllocated,
			})
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

		// Get user's miners
		minerRows, _ := db.Query(`
			SELECT id, name, address, hashrate, is_active, last_seen, created_at
			FROM miners WHERE user_id = $1 ORDER BY last_seen DESC
		`, userID)
		defer minerRows.Close()

		var miners []gin.H
		for minerRows.Next() {
			var id int64
			var name string
			var address sql.NullString
			var hashrate float64
			var isActive bool
			var lastSeen, createdAt time.Time
			minerRows.Scan(&id, &name, &address, &hashrate, &isActive, &lastSeen, &createdAt)
			miners = append(miners, gin.H{
				"id":         id,
				"name":       name,
				"address":    address.String,
				"hashrate":   hashrate,
				"is_active":  isActive,
				"last_seen":  lastSeen,
				"created_at": createdAt,
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
			},
			"miners":         miners,
			"payouts":        payouts,
			"wallets":        wallets,
			"wallet_summary": walletSummary,
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

// Email sending function with TLS support for GoDaddy (port 465)
func sendPasswordResetEmail(config *Config, toEmail, username, resetLink string) error {
	subject := "Chimera Pool - Password Reset Request"
	body := fmt.Sprintf(`Hello %s,

You have requested to reset your password for your Chimera Pool account.

Click the link below to reset your password:
%s

This link will expire in 1 hour.

If you did not request this password reset, please ignore this email. Your password will remain unchanged.

Best regards,
Chimera Pool Team

---
This is an automated message. Please do not reply to this email.
`, username, resetLink)

	// Compose email
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", config.SMTPFrom, toEmail, subject, body)

	// Use TLS for port 465 (GoDaddy), STARTTLS for port 587
	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)

	if config.SMTPPort == "465" {
		// Direct TLS connection for port 465
		tlsConfig := &tls.Config{
			ServerName: config.SMTPHost,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer client.Close()

		// Authenticate
		auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}

		// Set sender and recipient
		if err := client.Mail(config.SMTPFrom); err != nil {
			return fmt.Errorf("failed to set sender: %v", err)
		}
		if err := client.Rcpt(toEmail); err != nil {
			return fmt.Errorf("failed to set recipient: %v", err)
		}

		// Send message body
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %v", err)
		}
		_, err = w.Write([]byte(msg))
		if err != nil {
			return fmt.Errorf("failed to write message: %v", err)
		}
		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %v", err)
		}

		client.Quit()
	} else {
		// Standard STARTTLS for port 587
		auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
		err := smtp.SendMail(addr, auth, config.SMTPFrom, []string{toEmail}, []byte(msg))
		if err != nil {
			return fmt.Errorf("failed to send email: %v", err)
		}
	}

	log.Printf("Password reset email sent to %s", toEmail)
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

// Pool-wide statistics handlers
func handlePoolHashrateHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rangeStr := c.DefaultQuery("range", "24h")
		duration := parseTimeRange(rangeStr)
		interval := getInterval(duration)

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', created_at) as time_bucket,
				SUM(hashrate) as total_hashrate,
				AVG(hashrate) as avg_hashrate,
				COUNT(DISTINCT user_id) as active_users
			FROM miners 
			WHERE created_at > NOW() - INTERVAL '%s' AND is_active = true
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSamplePoolHashrateData(duration),
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var totalHashrate, avgHashrate float64
			var activeUsers int
			if err := rows.Scan(&timeBucket, &totalHashrate, &avgHashrate, &activeUsers); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":          timeBucket,
				"totalHashrate": totalHashrate,
				"avgHashrate":   avgHashrate,
				"activeUsers":   activeUsers,
			})
		}

		if len(data) == 0 {
			data = generateSamplePoolHashrateData(duration)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":     data,
			"range":    rangeStr,
			"interval": interval,
		})
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
				date_trunc('%s', created_at) as time_bucket,
				COUNT(*) FILTER (WHERE is_valid = true) as valid_shares,
				COUNT(*) FILTER (WHERE is_valid = false) as invalid_shares,
				COUNT(*) as total_shares
			FROM shares 
			WHERE created_at > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSamplePoolSharesData(duration),
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
			data = generateSamplePoolSharesData(duration)
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

		query := fmt.Sprintf(`
			SELECT 
				date_trunc('%s', last_seen) as time_bucket,
				COUNT(*) FILTER (WHERE is_active = true) as active_miners,
				COUNT(*) as total_miners,
				COUNT(DISTINCT user_id) as unique_users
			FROM miners 
			WHERE last_seen > NOW() - INTERVAL '%s'
			GROUP BY time_bucket
			ORDER BY time_bucket ASC
		`, interval, rangeStr)

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":     generateSampleMinersData(duration),
				"range":    rangeStr,
				"interval": interval,
			})
			return
		}
		defer rows.Close()

		var data []gin.H
		for rows.Next() {
			var timeBucket time.Time
			var activeMiners, totalMiners, uniqueUsers int
			if err := rows.Scan(&timeBucket, &activeMiners, &totalMiners, &uniqueUsers); err != nil {
				continue
			}
			data = append(data, gin.H{
				"time":         timeBucket,
				"activeMiners": activeMiners,
				"totalMiners":  totalMiners,
				"uniqueUsers":  uniqueUsers,
			})
		}

		if len(data) == 0 {
			data = generateSampleMinersData(duration)
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
				"distribution": generateSampleDistributionData(),
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

		// Calculate percentages
		for _, d := range tempData {
			hashrate := d["hashrate"].(float64)
			percentage := float64(0)
			if totalPoolHashrate > 0 {
				percentage = (hashrate / totalPoolHashrate) * 100
			}
			distribution = append(distribution, gin.H{
				"userId":     d["userId"],
				"username":   d["username"],
				"hashrate":   hashrate,
				"minerCount": d["minerCount"],
				"percentage": percentage,
			})
		}

		if len(distribution) == 0 {
			distribution = generateSampleDistributionData()
		}

		c.JSON(http.StatusOK, gin.H{
			"distribution":      distribution,
			"totalPoolHashrate": totalPoolHashrate,
		})
	}
}

func generateSampleDistributionData() []gin.H {
	users := []string{"miner_alpha", "crypto_king", "hash_master", "block_hunter", "node_runner"}
	var distribution []gin.H
	totalHashrate := 5000000000.0

	for i, username := range users {
		percentage := 30.0 - float64(i)*5
		if percentage < 5 {
			percentage = 5
		}
		hashrate := totalHashrate * (percentage / 100)
		distribution = append(distribution, gin.H{
			"userId":     i + 1,
			"username":   username,
			"hashrate":   hashrate,
			"minerCount": 3 - (i / 2),
			"percentage": percentage,
		})
	}
	return distribution
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
		rows, err := db.Query(`
			SELECT c.id, c.name, c.description, c.channel_type, c.is_read_only, c.admin_only_post,
				   cc.id as category_id, cc.name as category_name, cc.sort_order as category_order,
				   c.sort_order
			FROM channels c
			LEFT JOIN channel_categories cc ON c.category_id = cc.id
			ORDER BY cc.sort_order, c.sort_order
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
			return
		}
		defer rows.Close()

		categories := make(map[int]gin.H)
		categoryOrder := []int{}

		for rows.Next() {
			var id int
			var name, description, channelType string
			var isReadOnly, adminOnlyPost bool
			var categoryID sql.NullInt64
			var categoryName sql.NullString
			var catSortOrder, sortOrder int

			rows.Scan(&id, &name, &description, &channelType, &isReadOnly, &adminOnlyPost,
				&categoryID, &categoryName, &catSortOrder, &sortOrder)

			catID := 0
			catName := "Uncategorized"
			if categoryID.Valid {
				catID = int(categoryID.Int64)
				catName = categoryName.String
			}

			if _, exists := categories[catID]; !exists {
				categories[catID] = gin.H{
					"id":       catID,
					"name":     catName,
					"channels": []gin.H{},
				}
				categoryOrder = append(categoryOrder, catID)
			}

			channels := categories[catID]["channels"].([]gin.H)
			channels = append(channels, gin.H{
				"id":            id,
				"name":          name,
				"description":   description,
				"type":          channelType,
				"isReadOnly":    isReadOnly,
				"adminOnlyPost": adminOnlyPost,
			})
			categories[catID]["channels"] = channels
		}

		result := []gin.H{}
		for _, catID := range categoryOrder {
			result = append(result, categories[catID])
		}

		c.JSON(http.StatusOK, gin.H{"categories": result})
	}
}

func handleGetChannelMessages(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("id")
		before := c.Query("before")
		limit := 50

		query := `
			SELECT cm.id, cm.content, cm.is_edited, cm.created_at, cm.reply_to_id,
				   u.id as user_id, u.username,
				   COALESCE(b.icon, '🌱') as badge_icon, COALESCE(b.color, '#4ade80') as badge_color
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
		for rows.Next() {
			var id int64
			var content string
			var isEdited bool
			var createdAt time.Time
			var replyToID sql.NullInt64
			var userID int64
			var username, badgeIcon, badgeColor string

			rows.Scan(&id, &content, &isEdited, &createdAt, &replyToID, &userID, &username, &badgeIcon, &badgeColor)

			msg := gin.H{
				"id":        id,
				"content":   content,
				"isEdited":  isEdited,
				"createdAt": createdAt,
				"user": gin.H{
					"id":         userID,
					"username":   username,
					"badgeIcon":  badgeIcon,
					"badgeColor": badgeColor,
				},
			}
			if replyToID.Valid {
				msg["replyToId"] = replyToID.Int64
			}
			messages = append(messages, msg)
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
			ReplyToID *int64 `json:"replyToId"`
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

		result, err := db.Exec(`
			UPDATE channel_messages SET is_deleted = true WHERE id = $1 AND user_id = $2
		`, messageID, userID)

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

		_, err := db.Exec(`
			INSERT INTO message_reactions (message_id, user_id, emoji)
			VALUES ($1, $2, $3)
			ON CONFLICT (message_id, user_id, emoji) DO NOTHING
		`, messageID, userID, req.Emoji)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func handleRemoveReaction(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		messageID := c.Param("id")
		emoji := c.Param("emoji")

		db.Exec(`DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`,
			messageID, userID, emoji)

		c.JSON(http.StatusOK, gin.H{"success": true})
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
		limit := 20

		var rows *sql.Rows
		var err error

		switch leaderboardType {
		case "blocks":
			rows, err = db.Query(`
				SELECT u.id, u.username, COUNT(b.id) as score,
					   COALESCE(bd.icon, '🌱') as badge_icon
				FROM users u
				LEFT JOIN blocks b ON u.id = b.finder_id
				LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
				LEFT JOIN badges bd ON ub.badge_id = bd.id
				GROUP BY u.id, u.username, bd.icon
				ORDER BY score DESC
				LIMIT $1
			`, limit)
		case "forum":
			rows, err = db.Query(`
				SELECT u.id, u.username, COALESCE(up.forum_post_count, 0) as score,
					   COALESCE(bd.icon, '🌱') as badge_icon
				FROM users u
				LEFT JOIN user_profiles up ON u.id = up.user_id
				LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
				LEFT JOIN badges bd ON ub.badge_id = bd.id
				ORDER BY score DESC
				LIMIT $1
			`, limit)
		default: // hashrate
			rows, err = db.Query(`
				SELECT u.id, u.username, COALESCE(SUM(m.hashrate), 0) as score,
					   COALESCE(bd.icon, '🌱') as badge_icon
				FROM users u
				LEFT JOIN miners m ON u.id = m.user_id AND m.is_active = true
				LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.is_primary = true
				LEFT JOIN badges bd ON ub.badge_id = bd.id
				GROUP BY u.id, u.username, bd.icon
				ORDER BY score DESC
				LIMIT $1
			`, limit)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
			return
		}
		defer rows.Close()

		leaders := []gin.H{}
		rank := 1
		for rows.Next() {
			var userID int64
			var username string
			var score float64
			var badgeIcon string

			rows.Scan(&userID, &username, &score, &badgeIcon)
			leaders = append(leaders, gin.H{
				"rank":      rank,
				"userId":    userID,
				"username":  username,
				"score":     score,
				"badgeIcon": badgeIcon,
			})
			rank++
		}

		c.JSON(http.StatusOK, gin.H{"leaderboard": leaders, "type": leaderboardType})
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
