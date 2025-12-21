package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// ISP-COMPLIANT AUTH INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// PasswordHasher handles password hashing (single responsibility)
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

// TokenGenerator generates authentication tokens (single responsibility)
type TokenGenerator interface {
	Generate(userID int64, username string) (string, error)
}

// TokenValidator validates authentication tokens (single responsibility)
type TokenValidator interface {
	Validate(token string) (*TokenClaims, error)
}

// UserAuthenticator handles user authentication (combines validation)
type UserAuthenticator interface {
	Authenticate(email, password string) (*AuthenticatedUser, error)
}

// UserRegistrar handles user registration (single responsibility)
type UserRegistrar interface {
	Register(req *RegisterRequest) (*RegisteredUser, error)
}

// PasswordResetter handles password reset flow (single responsibility)
type PasswordResetter interface {
	RequestReset(email string) error
	ValidateToken(token string) (int64, error)
	ResetPassword(token, newPassword string) error
}

// =============================================================================
// AUTH DATA MODELS
// =============================================================================

// TokenClaims represents decoded JWT claims
type TokenClaims struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// AuthenticatedUser represents a successfully authenticated user
type AuthenticatedUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// RegisterRequest represents user registration data
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// RegisteredUser represents a newly registered user
type RegisteredUser struct {
	ID       int64     `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	JoinedAt time.Time `json:"joined_at"`
}

// LoginRequest represents user login data
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents successful login response
type LoginResponse struct {
	Token    string `json:"token"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

// ForgotPasswordRequest represents password reset request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents password reset with token
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// =============================================================================
// AUTH HANDLERS - ISP-COMPLIANT IMPLEMENTATION
// =============================================================================

// AuthHandlers provides HTTP handlers for authentication endpoints
type AuthHandlers struct {
	registrar        UserRegistrar
	authenticator    UserAuthenticator
	tokenGenerator   TokenGenerator
	passwordResetter PasswordResetter
}

// NewAuthHandlers creates new auth handlers with injected dependencies
func NewAuthHandlers(
	registrar UserRegistrar,
	authenticator UserAuthenticator,
	tokenGenerator TokenGenerator,
	passwordResetter PasswordResetter,
) *AuthHandlers {
	return &AuthHandlers{
		registrar:        registrar,
		authenticator:    authenticator,
		tokenGenerator:   tokenGenerator,
		passwordResetter: passwordResetter,
	}
}

// Register handles user registration
// POST /api/v1/auth/register
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

	user, err := h.registrar.Register(&req)
	if err != nil {
		statusCode := http.StatusConflict
		if err.Error() == "invalid email format" || err.Error() == "password too weak" {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, ErrorResponse{
			Error:   "registration_failed",
			Message: err.Error(),
			Code:    statusCode,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User registered successfully",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

// Login handles user authentication
// POST /api/v1/auth/login
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

	user, err := h.authenticator.Authenticate(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_failed",
			Message: "Invalid credentials",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	token, err := h.tokenGenerator.Generate(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_generation_failed",
			Message: "Failed to generate authentication token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
	})
}

// ForgotPassword initiates password reset flow
// POST /api/v1/auth/forgot-password
func (h *AuthHandlers) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Always return success to prevent email enumeration
	_ = h.passwordResetter.RequestReset(req.Email)

	c.JSON(http.StatusOK, gin.H{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword completes password reset with token
// POST /api/v1/auth/reset-password
func (h *AuthHandlers) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request data: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.passwordResetter.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err.Error() == "token expired" {
			statusCode = http.StatusUnauthorized
		}
		c.JSON(statusCode, ErrorResponse{
			Error:   "reset_failed",
			Message: err.Error(),
			Code:    statusCode,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}

// =============================================================================
// AUTH MIDDLEWARE
// =============================================================================

// AuthMiddleware creates authentication middleware using token validator
func AuthMiddleware(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "Authorization header required",
				Code:    http.StatusUnauthorized,
			})
			return
		}

		// Extract token from "Bearer <token>"
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			token = authHeader
		}

		claims, err := validator.Validate(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired token",
				Code:    http.StatusUnauthorized,
			})
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
