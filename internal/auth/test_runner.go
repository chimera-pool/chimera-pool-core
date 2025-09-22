package auth

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// TestRunner provides a simple way to run authentication tests manually
type TestRunner struct {
	authService *AuthService
}

// NewTestRunner creates a new test runner
func NewTestRunner() *TestRunner {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-runner-secret-key")
	
	return &TestRunner{
		authService: authService,
	}
}

// RunBasicTests runs basic authentication tests and reports results
func (tr *TestRunner) RunBasicTests() {
	fmt.Println("🧪 Running Basic Authentication Tests...")
	fmt.Println(strings.Repeat("=", 50))
	
	// Test 1: User Registration
	fmt.Println("\n1. Testing User Registration...")
	user, err := tr.authService.RegisterUser("testuser", "test@example.com", "SecurePass123!")
	if err != nil {
		log.Printf("❌ Registration failed: %v", err)
		return
	}
	fmt.Printf("✅ User registered successfully: ID=%d, Username=%s, Email=%s\n", 
		user.ID, user.Username, user.Email)
	
	// Test 2: Password Hashing
	fmt.Println("\n2. Testing Password Security...")
	if user.PasswordHash == "SecurePass123!" {
		log.Printf("❌ Password not hashed!")
		return
	}
	fmt.Printf("✅ Password properly hashed (length: %d)\n", len(user.PasswordHash))
	
	// Test 3: User Login
	fmt.Println("\n3. Testing User Login...")
	_, token, err := tr.authService.LoginUser("testuser", "SecurePass123!")
	if err != nil {
		log.Printf("❌ Login failed: %v", err)
		return
	}
	fmt.Printf("✅ Login successful, JWT token generated (length: %d)\n", len(token))
	
	// Test 4: JWT Token Validation
	fmt.Println("\n4. Testing JWT Token Validation...")
	claims, err := tr.authService.ValidateJWT(token)
	if err != nil {
		log.Printf("❌ Token validation failed: %v", err)
		return
	}
	fmt.Printf("✅ Token validated successfully: UserID=%d, Username=%s, Expires=%s\n",
		claims.UserID, claims.Username, claims.ExpiresAt.Format(time.RFC3339))
	
	// Test 5: Invalid Login
	fmt.Println("\n5. Testing Invalid Login...")
	_, _, err = tr.authService.LoginUser("testuser", "wrongpassword")
	if err == nil {
		log.Printf("❌ Invalid login should have failed!")
		return
	}
	fmt.Printf("✅ Invalid login properly rejected: %v\n", err)
	
	// Test 6: Duplicate Registration
	fmt.Println("\n6. Testing Duplicate Registration Prevention...")
	_, err = tr.authService.RegisterUser("testuser", "different@example.com", "AnotherPass123!")
	if err == nil {
		log.Printf("❌ Duplicate username should have been rejected!")
		return
	}
	fmt.Printf("✅ Duplicate username properly rejected: %v\n", err)
	
	// Test 7: Input Validation
	fmt.Println("\n7. Testing Input Validation...")
	_, err = tr.authService.RegisterUser("", "test@example.com", "ValidPass123!")
	if err == nil {
		log.Printf("❌ Empty username should have been rejected!")
		return
	}
	fmt.Printf("✅ Empty username properly rejected: %v\n", err)
	
	_, err = tr.authService.RegisterUser("validuser", "invalid-email", "ValidPass123!")
	if err == nil {
		log.Printf("❌ Invalid email should have been rejected!")
		return
	}
	fmt.Printf("✅ Invalid email properly rejected: %v\n", err)
	
	_, err = tr.authService.RegisterUser("validuser", "valid@example.com", "123")
	if err == nil {
		log.Printf("❌ Weak password should have been rejected!")
		return
	}
	fmt.Printf("✅ Weak password properly rejected: %v\n", err)
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎉 All basic authentication tests passed!")
	fmt.Println("✅ User registration works correctly")
	fmt.Println("✅ Password hashing is secure")
	fmt.Println("✅ User login works correctly")
	fmt.Println("✅ JWT token generation and validation works")
	fmt.Println("✅ Security validations are in place")
	fmt.Println("✅ Input validation prevents invalid data")
}

// RunPerformanceTests runs basic performance tests
func (tr *TestRunner) RunPerformanceTests() {
	fmt.Println("\n🚀 Running Performance Tests...")
	fmt.Println(strings.Repeat("=", 50))
	
	// Test password hashing performance
	fmt.Println("\n1. Testing Password Hashing Performance...")
	start := time.Now()
	for i := 0; i < 10; i++ {
		_, err := tr.authService.HashPassword("TestPassword123!")
		if err != nil {
			log.Printf("❌ Password hashing failed: %v", err)
			return
		}
	}
	duration := time.Since(start)
	fmt.Printf("✅ 10 password hashes completed in %v (avg: %v per hash)\n", 
		duration, duration/10)
	
	// Test JWT token generation performance
	fmt.Println("\n2. Testing JWT Token Generation Performance...")
	user := &User{
		ID:       1,
		Username: "perftest",
		Email:    "perf@example.com",
		IsActive: true,
	}
	
	start = time.Now()
	for i := 0; i < 100; i++ {
		_, err := tr.authService.GenerateJWT(user)
		if err != nil {
			log.Printf("❌ JWT generation failed: %v", err)
			return
		}
	}
	duration = time.Since(start)
	fmt.Printf("✅ 100 JWT tokens generated in %v (avg: %v per token)\n", 
		duration, duration/100)
	
	// Test JWT token validation performance
	fmt.Println("\n3. Testing JWT Token Validation Performance...")
	token, err := tr.authService.GenerateJWT(user)
	if err != nil {
		log.Printf("❌ JWT generation failed: %v", err)
		return
	}
	
	start = time.Now()
	for i := 0; i < 100; i++ {
		_, err := tr.authService.ValidateJWT(token)
		if err != nil {
			log.Printf("❌ JWT validation failed: %v", err)
			return
		}
	}
	duration = time.Since(start)
	fmt.Printf("✅ 100 JWT validations completed in %v (avg: %v per validation)\n", 
		duration, duration/100)
	
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎉 Performance tests completed!")
}

// RunAllTests runs all available tests
func (tr *TestRunner) RunAllTests() {
	tr.RunBasicTests()
	tr.RunPerformanceTests()
}