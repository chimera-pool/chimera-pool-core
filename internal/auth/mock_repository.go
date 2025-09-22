package auth

import (
	"errors"
	"sync"
	"time"
)

// MockUserRepository is a mock implementation of UserRepository for testing
type MockUserRepository struct {
	users  map[int64]*User
	nextID int64
	mutex  sync.RWMutex
}

// NewMockUserRepository creates a new mock user repository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[int64]*User),
		nextID: 1,
	}
}

// CreateUser creates a new user in memory
func (r *MockUserRepository) CreateUser(user *User) error {
	if user == nil {
		return errors.New("user is required")
	}
	
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Check for duplicate username
	for _, existingUser := range r.users {
		if existingUser.Username == user.Username && existingUser.IsActive {
			return errors.New("username already exists")
		}
	}
	
	// Check for duplicate email
	for _, existingUser := range r.users {
		if existingUser.Email == user.Email && existingUser.IsActive {
			return errors.New("email already exists")
		}
	}
	
	// Assign ID and timestamps
	user.ID = r.nextID
	r.nextID++
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	
	// Store user
	r.users[user.ID] = &User{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		IsActive:     user.IsActive,
	}
	
	return nil
}

// GetUserByUsername retrieves a user by username
func (r *MockUserRepository) GetUserByUsername(username string) (*User, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}
	
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, user := range r.users {
		if user.Username == username && user.IsActive {
			return &User{
				ID:           user.ID,
				Username:     user.Username,
				Email:        user.Email,
				PasswordHash: user.PasswordHash,
				CreatedAt:    user.CreatedAt,
				UpdatedAt:    user.UpdatedAt,
				IsActive:     user.IsActive,
			}, nil
		}
	}
	
	return nil, nil // User not found
}

// GetUserByEmail retrieves a user by email
func (r *MockUserRepository) GetUserByEmail(email string) (*User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}
	
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, user := range r.users {
		if user.Email == email && user.IsActive {
			return &User{
				ID:           user.ID,
				Username:     user.Username,
				Email:        user.Email,
				PasswordHash: user.PasswordHash,
				CreatedAt:    user.CreatedAt,
				UpdatedAt:    user.UpdatedAt,
				IsActive:     user.IsActive,
			}, nil
		}
	}
	
	return nil, nil // User not found
}

// GetUserByID retrieves a user by ID
func (r *MockUserRepository) GetUserByID(id int64) (*User, error) {
	if id <= 0 {
		return nil, errors.New("valid user ID is required")
	}
	
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	user, exists := r.users[id]
	if !exists {
		return nil, nil // User not found
	}
	
	return &User{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		IsActive:     user.IsActive,
	}, nil
}

// UpdateUser updates an existing user
func (r *MockUserRepository) UpdateUser(user *User) error {
	if user == nil {
		return errors.New("user is required")
	}
	
	if user.ID <= 0 {
		return errors.New("valid user ID is required")
	}
	
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	existingUser, exists := r.users[user.ID]
	if !exists {
		return errors.New("user not found")
	}
	
	// Check for duplicate username (excluding current user)
	for id, otherUser := range r.users {
		if id != user.ID && otherUser.Username == user.Username && otherUser.IsActive {
			return errors.New("username already exists")
		}
	}
	
	// Check for duplicate email (excluding current user)
	for id, otherUser := range r.users {
		if id != user.ID && otherUser.Email == user.Email && otherUser.IsActive {
			return errors.New("email already exists")
		}
	}
	
	// Update user
	existingUser.Username = user.Username
	existingUser.Email = user.Email
	existingUser.PasswordHash = user.PasswordHash
	existingUser.UpdatedAt = time.Now()
	existingUser.IsActive = user.IsActive
	
	// Update the passed user with new timestamp
	user.UpdatedAt = existingUser.UpdatedAt
	
	return nil
}

// DeleteUser soft deletes a user by setting is_active to false
func (r *MockUserRepository) DeleteUser(id int64) error {
	if id <= 0 {
		return errors.New("valid user ID is required")
	}
	
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	user, exists := r.users[id]
	if !exists {
		return errors.New("user not found")
	}
	
	user.IsActive = false
	user.UpdatedAt = time.Now()
	
	return nil
}

// Reset clears all users (useful for testing)
func (r *MockUserRepository) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.users = make(map[int64]*User)
	r.nextID = 1
}

// GetAllUsers returns all users (useful for testing)
func (r *MockUserRepository) GetAllUsers() []*User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, &User{
			ID:           user.ID,
			Username:     user.Username,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			IsActive:     user.IsActive,
		})
	}
	
	return users
}