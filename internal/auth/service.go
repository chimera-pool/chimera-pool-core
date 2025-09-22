package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides authentication functionality
type AuthService struct {
	userRepo  UserRepository
	jwtSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// RegisterUser registers a new user with validation
func (s *AuthService) RegisterUser(username, email, password string) (*User, error) {
	// Validate input
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("username is required")
	}
	
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("email is required")
	}
	
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("password is required")
	}
	
	// Validate password strength
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters long")
	}
	
	// Create user model
	user := &User{
		Username: strings.TrimSpace(username),
		Email:    strings.TrimSpace(email),
		IsActive: true,
	}
	
	// Validate user model
	if err := user.Validate(); err != nil {
		return nil, err
	}
	
	// Check if username already exists
	if s.userRepo != nil {
		existingUser, _ := s.userRepo.GetUserByUsername(user.Username)
		if existingUser != nil {
			return nil, errors.New("username already exists")
		}
		
		// Check if email already exists
		existingUser, _ = s.userRepo.GetUserByEmail(user.Email)
		if existingUser != nil {
			return nil, errors.New("email already exists")
		}
	}
	
	// Hash password
	passwordHash, err := s.HashPassword(password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}
	
	user.PasswordHash = passwordHash
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	
	// Save user to database
	if s.userRepo != nil {
		if err := s.userRepo.CreateUser(user); err != nil {
			return nil, errors.New("failed to create user")
		}
	}
	
	return user, nil
}

// LoginUser authenticates a user and returns user info and JWT token
func (s *AuthService) LoginUser(username, password string) (*User, string, error) {
	// Validate input
	if strings.TrimSpace(username) == "" {
		return nil, "", errors.New("username is required")
	}
	
	if strings.TrimSpace(password) == "" {
		return nil, "", errors.New("password is required")
	}
	
	// Get user from database
	if s.userRepo == nil {
		return nil, "", errors.New("user repository not configured")
	}
	
	user, err := s.userRepo.GetUserByUsername(strings.TrimSpace(username))
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	
	if user == nil {
		return nil, "", errors.New("invalid credentials")
	}
	
	// Check if user is active
	if !user.IsActive {
		return nil, "", errors.New("account is disabled")
	}
	
	// Verify password
	if !s.VerifyPassword(password, user.PasswordHash) {
		return nil, "", errors.New("invalid credentials")
	}
	
	// Generate JWT token
	token, err := s.GenerateJWT(user)
	if err != nil {
		return nil, "", errors.New("failed to generate token")
	}
	
	return user, token, nil
}

// GenerateJWT generates a JWT token for the user
func (s *AuthService) GenerateJWT(user *User) (string, error) {
	if user == nil {
		return "", errors.New("user is required")
	}
	
	// Create claims
	claims := &JWTClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Token expires in 24 hours
	}
	
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"email":    claims.Email,
		"iat":      claims.IssuedAt.Unix(),
		"exp":      claims.ExpiresAt.Unix(),
	})
	
	// Sign token with secret
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the claims
func (s *AuthService) ValidateJWT(tokenString string) (*JWTClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token is required")
	}
	
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return s.jwtSecret, nil
	})
	
	if err != nil {
		return nil, errors.New("invalid token")
	}
	
	// Validate token
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	
	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	
	// Parse claims
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("invalid user_id in token")
	}
	
	username, ok := claims["username"].(string)
	if !ok {
		return nil, errors.New("invalid username in token")
	}
	
	email, ok := claims["email"].(string)
	if !ok {
		return nil, errors.New("invalid email in token")
	}
	
	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, errors.New("invalid iat in token")
	}
	
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid exp in token")
	}
	
	// Check if token is expired
	expiresAt := time.Unix(int64(exp), 0)
	if time.Now().After(expiresAt) {
		return nil, errors.New("token expired")
	}
	
	return &JWTClaims{
		UserID:    int64(userID),
		Username:  username,
		Email:     email,
		IssuedAt:  time.Unix(int64(iat), 0),
		ExpiresAt: expiresAt,
	}, nil
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("password is required")
	}
	
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (s *AuthService) VerifyPassword(password, hash string) bool {
	if strings.TrimSpace(password) == "" || strings.TrimSpace(hash) == "" {
		return false
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}