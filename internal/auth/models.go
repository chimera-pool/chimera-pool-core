package auth

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Role represents a user's role in the system
type Role string

const (
	RoleUser       Role = "user"
	RoleModerator  Role = "moderator"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

// IsValid checks if the role is a valid role type
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleModerator, RoleAdmin, RoleSuperAdmin:
		return true
	}
	return false
}

// Level returns the permission level of a role (higher = more permissions)
func (r Role) Level() int {
	switch r {
	case RoleUser:
		return 1
	case RoleModerator:
		return 2
	case RoleAdmin:
		return 3
	case RoleSuperAdmin:
		return 4
	}
	return 0
}

// CanManageRole checks if this role can manage (promote/demote) the target role
func (r Role) CanManageRole(target Role) bool {
	// Super admin can manage anyone
	if r == RoleSuperAdmin {
		return true
	}
	// Admin can manage moderators and users, but not admins or super_admins
	if r == RoleAdmin {
		return target == RoleModerator || target == RoleUser
	}
	// Moderators and users cannot manage roles
	return false
}

// User represents a user in the system
type User struct {
	ID           int64     `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash in JSON
	Role         Role      `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	IsActive     bool      `json:"is_active" db:"is_active"`
}

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// Validate validates the user model
func (u *User) Validate() error {
	if strings.TrimSpace(u.Username) == "" {
		return errors.New("username is required")
	}
	
	if len(u.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	
	if len(u.Username) > 50 {
		return errors.New("username must be at most 50 characters long")
	}
	
	if strings.TrimSpace(u.Email) == "" {
		return errors.New("email is required")
	}
	
	if !isValidEmail(u.Email) {
		return errors.New("invalid email format")
	}
	
	return nil
}

// isValidEmail validates email format using regex
func isValidEmail(email string) bool {
	// Basic email validation that allows common formats but rejects obvious invalid ones
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	// Additional checks for invalid patterns
	if strings.Contains(email, "..") {
		return false // No consecutive dots
	}
	if strings.HasPrefix(email, ".") || strings.HasSuffix(email, ".") {
		return false // No leading or trailing dots
	}
	if strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@") {
		return false // No leading or trailing @
	}
	
	return emailRegex.MatchString(email)
}

// UserRepository defines the interface for user data operations
type UserRepository interface {
	CreateUser(user *User) error
	GetUserByUsername(username string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id int64) (*User, error)
	UpdateUser(user *User) error
	DeleteUser(id int64) error
}