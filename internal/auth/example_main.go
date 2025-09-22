package auth

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ExampleMain demonstrates how to use the authentication system
func ExampleMain() {
	fmt.Println("🚀 Starting Chimera Pool Authentication Example...")
	
	// Run the demo first
	DemoAuthSystem()
	
	fmt.Println("\n🌐 Starting HTTP server example...")
	
	// Create auth service with mock repository
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "example-jwt-secret-key")
	
	// Create Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// Setup auth routes
	SetupAuthRoutes(router, authService)
	
	// Add health check
	router.GET("/health", HealthCheck)
	
	// Add a simple welcome endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Chimera Pool Authentication API",
			"version": "1.0.0",
			"endpoints": gin.H{
				"register":      "POST /api/auth/register",
				"login":         "POST /api/auth/login",
				"validate":      "POST /api/auth/validate",
				"profile":       "GET /api/user/profile (requires auth)",
				"health":        "GET /health",
			},
			"example_usage": gin.H{
				"register": gin.H{
					"method": "POST",
					"url":    "/api/auth/register",
					"body": gin.H{
						"username": "testuser",
						"email":    "test@example.com",
						"password": "SecurePass123!",
					},
				},
				"login": gin.H{
					"method": "POST",
					"url":    "/api/auth/login",
					"body": gin.H{
						"username": "testuser",
						"password": "SecurePass123!",
					},
				},
			},
		})
	})
	
	// Pre-populate with a demo user
	demoUser, err := authService.RegisterUser("demo", "demo@example.com", "DemoPass123!")
	if err != nil {
		log.Printf("Failed to create demo user: %v", err)
	} else {
		fmt.Printf("✅ Demo user created: %s (ID: %d)\n", demoUser.Username, demoUser.ID)
		fmt.Println("   You can login with username: demo, password: DemoPass123!")
	}
	
	fmt.Println("\n📡 Server ready!")
	fmt.Println("   • Health check: http://localhost:8080/health")
	fmt.Println("   • API docs: http://localhost:8080/")
	fmt.Println("   • Register: POST http://localhost:8080/api/auth/register")
	fmt.Println("   • Login: POST http://localhost:8080/api/auth/login")
	fmt.Println("\n🔧 Example curl commands:")
	fmt.Println("   # Register new user:")
	fmt.Println(`   curl -X POST http://localhost:8080/api/auth/register \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -d '{"username":"newuser","email":"new@example.com","password":"NewPass123!"}'`)
	fmt.Println("\n   # Login:")
	fmt.Println(`   curl -X POST http://localhost:8080/api/auth/login \`)
	fmt.Println(`     -H "Content-Type: application/json" \`)
	fmt.Println(`     -d '{"username":"demo","password":"DemoPass123!"}'`)
	
	// Start server
	fmt.Println("\n🎯 Starting server on :8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}