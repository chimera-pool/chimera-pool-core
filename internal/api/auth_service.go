package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// AUTH SERVICE IMPLEMENTATIONS
// ISP-compliant services that implement the auth interfaces
// =============================================================================

// -----------------------------------------------------------------------------
// Password Hasher Implementation
// -----------------------------------------------------------------------------

// BcryptHasher implements PasswordHasher using bcrypt
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a new bcrypt password hasher
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

// Hash hashes a password using bcrypt
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify checks if a password matches a hash
func (h *BcryptHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// -----------------------------------------------------------------------------
// Token Generator Implementation
// -----------------------------------------------------------------------------

// JWTTokenGenerator implements TokenGenerator using JWT
type JWTTokenGenerator struct {
	secret     string
	expiration time.Duration
}

// NewJWTTokenGenerator creates a new JWT token generator
func NewJWTTokenGenerator(secret string, expiration time.Duration) *JWTTokenGenerator {
	return &JWTTokenGenerator{
		secret:     secret,
		expiration: expiration,
	}
}

// Generate creates a new JWT token for a user
func (g *JWTTokenGenerator) Generate(userID int64, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(g.expiration).Unix(),
		"iat":      time.Now().Unix(),
	})
	return token.SignedString([]byte(g.secret))
}

// -----------------------------------------------------------------------------
// Token Validator Implementation
// -----------------------------------------------------------------------------

// JWTTokenValidator implements TokenValidator using JWT
type JWTTokenValidator struct {
	secret string
}

// NewJWTTokenValidator creates a new JWT token validator
func NewJWTTokenValidator(secret string) *JWTTokenValidator {
	return &JWTTokenValidator{secret: secret}
}

// Validate validates a JWT token and returns the claims
func (v *JWTTokenValidator) Validate(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(v.secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, errors.New("invalid user_id in token")
	}

	username, _ := claims["username"].(string)
	exp, _ := claims["exp"].(float64)
	iat, _ := claims["iat"].(float64)

	return &TokenClaims{
		UserID:    int64(userID),
		Username:  username,
		ExpiresAt: int64(exp),
		IssuedAt:  int64(iat),
	}, nil
}

// -----------------------------------------------------------------------------
// User Registrar Implementation
// -----------------------------------------------------------------------------

// DBUserRegistrar implements UserRegistrar using database
type DBUserRegistrar struct {
	db     *sql.DB
	hasher PasswordHasher
}

// NewDBUserRegistrar creates a new database user registrar
func NewDBUserRegistrar(db *sql.DB, hasher PasswordHasher) *DBUserRegistrar {
	return &DBUserRegistrar{db: db, hasher: hasher}
}

// Register creates a new user account
func (r *DBUserRegistrar) Register(req *RegisterRequest) (*RegisteredUser, error) {
	// Validate password strength
	if len(req.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Hash password
	hashedPassword, err := r.hasher.Hash(req.Password)
	if err != nil {
		return nil, errors.New("failed to process password")
	}

	// Insert user
	var userID int64
	var createdAt time.Time
	err = r.db.QueryRow(
		"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at",
		req.Username, req.Email, hashedPassword,
	).Scan(&userID, &createdAt)

	if err != nil {
		return nil, errors.New("username or email already exists")
	}

	return &RegisteredUser{
		ID:       userID,
		Username: req.Username,
		Email:    req.Email,
		JoinedAt: createdAt,
	}, nil
}

// -----------------------------------------------------------------------------
// User Authenticator Implementation
// -----------------------------------------------------------------------------

// DBUserAuthenticator implements UserAuthenticator using database
type DBUserAuthenticator struct {
	db     *sql.DB
	hasher PasswordHasher
}

// NewDBUserAuthenticator creates a new database user authenticator
func NewDBUserAuthenticator(db *sql.DB, hasher PasswordHasher) *DBUserAuthenticator {
	return &DBUserAuthenticator{db: db, hasher: hasher}
}

// Authenticate validates user credentials and returns user info
func (a *DBUserAuthenticator) Authenticate(email, password string) (*AuthenticatedUser, error) {
	var userID int64
	var username, passwordHash, role string
	err := a.db.QueryRow(
		"SELECT id, username, password_hash, COALESCE(role, 'user') FROM users WHERE email = $1 AND is_active = true",
		email,
	).Scan(&userID, &username, &passwordHash, &role)

	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !a.hasher.Verify(password, passwordHash) {
		return nil, errors.New("invalid credentials")
	}

	return &AuthenticatedUser{
		ID:       userID,
		Username: username,
		Email:    email,
		Role:     role,
	}, nil
}

// -----------------------------------------------------------------------------
// Password Resetter Implementation
// -----------------------------------------------------------------------------

// DBPasswordResetter implements PasswordResetter using database
type DBPasswordResetter struct {
	db          *sql.DB
	hasher      PasswordHasher
	tokenExpiry time.Duration
}

// NewDBPasswordResetter creates a new database password resetter
func NewDBPasswordResetter(db *sql.DB, hasher PasswordHasher, tokenExpiry time.Duration) *DBPasswordResetter {
	return &DBPasswordResetter{
		db:          db,
		hasher:      hasher,
		tokenExpiry: tokenExpiry,
	}
}

// RequestReset creates a password reset token and stores it
func (r *DBPasswordResetter) RequestReset(email string) error {
	// Find user by email
	var userID int64
	err := r.db.QueryRow("SELECT id FROM users WHERE email = $1 AND is_active = true", email).Scan(&userID)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Generate secure token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return errors.New("failed to generate token")
	}
	token := hex.EncodeToString(tokenBytes)

	// Store token with expiry
	expiresAt := time.Now().Add(r.tokenExpiry)
	_, err = r.db.Exec(
		"INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO UPDATE SET token = $2, expires_at = $3, used = false",
		userID, token, expiresAt,
	)

	// In production, send email with token here
	// For now, just store it

	return err
}

// ValidateToken validates a password reset token and returns the user ID
func (r *DBPasswordResetter) ValidateToken(token string) (int64, error) {
	var userID int64
	var expiresAt time.Time
	var used bool

	err := r.db.QueryRow(
		"SELECT user_id, expires_at, used FROM password_reset_tokens WHERE token = $1",
		token,
	).Scan(&userID, &expiresAt, &used)

	if err != nil {
		return 0, errors.New("invalid token")
	}

	if used {
		return 0, errors.New("token already used")
	}

	if time.Now().After(expiresAt) {
		return 0, errors.New("token expired")
	}

	return userID, nil
}

// ResetPassword resets the user's password using a valid token
func (r *DBPasswordResetter) ResetPassword(token, newPassword string) error {
	userID, err := r.ValidateToken(token)
	if err != nil {
		return err
	}

	// Validate password strength
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// Hash new password
	hashedPassword, err := r.hasher.Hash(newPassword)
	if err != nil {
		return errors.New("failed to process password")
	}

	// Update password
	_, err = r.db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", hashedPassword, userID)
	if err != nil {
		return errors.New("failed to update password")
	}

	// Mark token as used
	_, err = r.db.Exec("UPDATE password_reset_tokens SET used = true WHERE token = $1", token)

	return err
}

// =============================================================================
// AUTH SERVICE FACTORY
// Creates all auth services with proper dependencies
// =============================================================================

// AuthServices holds all auth-related service implementations
type AuthServices struct {
	Hasher           PasswordHasher
	TokenGenerator   TokenGenerator
	TokenValidator   TokenValidator
	Registrar        UserRegistrar
	Authenticator    UserAuthenticator
	PasswordResetter PasswordResetter
}

// NewAuthServices creates all auth services with the given configuration
func NewAuthServices(db *sql.DB, jwtSecret string) *AuthServices {
	hasher := NewBcryptHasher()

	return &AuthServices{
		Hasher:           hasher,
		TokenGenerator:   NewJWTTokenGenerator(jwtSecret, 24*time.Hour),
		TokenValidator:   NewJWTTokenValidator(jwtSecret),
		Registrar:        NewDBUserRegistrar(db, hasher),
		Authenticator:    NewDBUserAuthenticator(db, hasher),
		PasswordResetter: NewDBPasswordResetter(db, hasher, 1*time.Hour),
	}
}

// CreateAuthHandlers creates AuthHandlers with all services wired up
func (s *AuthServices) CreateAuthHandlers() *AuthHandlers {
	return NewAuthHandlers(
		s.Registrar,
		s.Authenticator,
		s.TokenGenerator,
		s.PasswordResetter,
	)
}
