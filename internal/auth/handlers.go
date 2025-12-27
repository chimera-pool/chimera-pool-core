package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthHandlers provides HTTP handlers for authentication
type AuthHandlers struct {
	authService *AuthService
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(authService *AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

// AuthErrorResponse is a local alias for consistent error responses
// Note: For cross-package usage, prefer api.ErrorResponse
type AuthErrorResponse = ErrorResponse

// ErrorResponse represents an error response (local to auth package)
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Register handles user registration
func (h *AuthHandlers) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	user, err := h.authService.RegisterUser(req.Username, req.Email, req.Password)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   "registration_failed",
			Message: err.Error(),
			Code:    statusCode,
		})
		return
	}

	// Generate JWT token for the new user
	token, err := h.authService.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_generation_failed",
			Message: "Failed to generate authentication token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		User:  user,
		Token: token,
	})
}

// Login handles user login
func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	user, token, err := h.authService.LoginUser(req.Username, req.Password)
	if err != nil {
		statusCode := http.StatusUnauthorized
		if strings.Contains(err.Error(), "required") {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   "authentication_failed",
			Message: err.Error(),
			Code:    statusCode,
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:  user,
		Token: token,
	})
}

// Profile returns the current user's profile
func (h *AuthHandlers) Profile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// ValidateToken validates a JWT token
func (h *AuthHandlers) ValidateToken(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_token",
			Message: "Authorization header is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Remove "Bearer " prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	claims, err := h.authService.ValidateJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_token",
			Message: err.Error(),
			Code:    http.StatusUnauthorized,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"claims":     claims,
		"expires_at": claims.ExpiresAt,
		"user_id":    claims.UserID,
		"username":   claims.Username,
	})
}

// AuthMiddleware provides JWT authentication middleware
func (h *AuthHandlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		}

		claims, err := h.authService.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: err.Error(),
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_email", claims.Email)
		c.Set("user", &User{
			ID:       claims.UserID,
			Username: claims.Username,
			Email:    claims.Email,
			IsActive: true,
		})

		c.Next()
	}
}

// SetupAuthRoutes sets up authentication routes
func SetupAuthRoutes(router *gin.Engine, authService *AuthService) {
	handlers := NewAuthHandlers(authService)

	// Public routes
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.POST("/validate", handlers.ValidateToken)
	}

	// Protected routes
	protected := router.Group("/api/user")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.GET("/profile", handlers.Profile)
	}
}

// HealthCheck provides a health check endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "chimera-pool-auth",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}
