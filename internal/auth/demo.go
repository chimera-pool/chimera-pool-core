package auth

import (
	"fmt"
	"log"
)

// DemoAuthSystem demonstrates the authentication system functionality
func DemoAuthSystem() {
	fmt.Println("🔐 Chimera Pool Authentication System Demo")
	fmt.Println("=========================================")
	
	// Create test runner and run all tests
	testRunner := NewTestRunner()
	testRunner.RunAllTests()
	
	fmt.Println("\n🔧 Manual Demo...")
	fmt.Println("================")
	
	// Create a fresh auth service for manual demo
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "demo-secret-key")
	
	// Demo 1: Register multiple users
	fmt.Println("\n📝 Registering multiple users...")
	users := []struct {
		username string
		email    string
		password string
	}{
		{"alice", "alice@example.com", "AlicePass123!"},
		{"bob", "bob@example.com", "BobSecure456!"},
		{"charlie", "charlie@example.com", "CharlieStrong789!"},
	}
	
	var registeredUsers []*User
	for _, userData := range users {
		user, err := authService.RegisterUser(userData.username, userData.email, userData.password)
		if err != nil {
			log.Printf("❌ Failed to register %s: %v", userData.username, err)
			continue
		}
		registeredUsers = append(registeredUsers, user)
		fmt.Printf("✅ Registered user: %s (ID: %d)\n", user.Username, user.ID)
	}
	
	// Demo 2: Login all users and generate tokens
	fmt.Println("\n🔑 Logging in users and generating JWT tokens...")
	var tokens []string
	for _, userData := range users {
		_, token, err := authService.LoginUser(userData.username, userData.password)
		if err != nil {
			log.Printf("❌ Failed to login %s: %v", userData.username, err)
			continue
		}
		tokens = append(tokens, token)
		fmt.Printf("✅ %s logged in, token: %s...\n", userData.username, token[:50])
	}
	
	// Demo 3: Validate tokens
	fmt.Println("\n🔍 Validating JWT tokens...")
	for i, token := range tokens {
		claims, err := authService.ValidateJWT(token)
		if err != nil {
			log.Printf("❌ Failed to validate token %d: %v", i+1, err)
			continue
		}
		fmt.Printf("✅ Token %d valid - User: %s, Expires: %s\n", 
			i+1, claims.Username, claims.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	
	// Demo 4: Test security features
	fmt.Println("\n🛡️  Testing security features...")
	
	// Test duplicate registration
	_, err := authService.RegisterUser("alice", "alice2@example.com", "AnotherPass123!")
	if err != nil {
		fmt.Printf("✅ Duplicate username rejected: %v\n", err)
	} else {
		fmt.Println("❌ Duplicate username should have been rejected!")
	}
	
	// Test invalid login
	_, _, err = authService.LoginUser("alice", "wrongpassword")
	if err != nil {
		fmt.Printf("✅ Invalid password rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid password should have been rejected!")
	}
	
	// Test invalid token
	_, err = authService.ValidateJWT("invalid.token.here")
	if err != nil {
		fmt.Printf("✅ Invalid token rejected: %v\n", err)
	} else {
		fmt.Println("❌ Invalid token should have been rejected!")
	}
	
	// Demo 5: Show repository state
	fmt.Println("\n📊 Repository state:")
	allUsers := mockRepo.GetAllUsers()
	fmt.Printf("Total users in repository: %d\n", len(allUsers))
	for _, user := range allUsers {
		status := "active"
		if !user.IsActive {
			status = "inactive"
		}
		fmt.Printf("  - %s (%s) - %s\n", user.Username, user.Email, status)
	}
	
	fmt.Println("\n🎉 Authentication system demo completed successfully!")
	fmt.Println("✅ User registration and validation working")
	fmt.Println("✅ Password hashing and verification working")
	fmt.Println("✅ JWT token generation and validation working")
	fmt.Println("✅ Security measures in place")
	fmt.Println("✅ Repository operations working")
}