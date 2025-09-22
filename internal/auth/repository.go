package auth

import (
	"database/sql"
	"errors"
	"time"
)

// PostgreSQLUserRepository implements UserRepository for PostgreSQL
type PostgreSQLUserRepository struct {
	db *sql.DB
}

// NewPostgreSQLUserRepository creates a new PostgreSQL user repository
func NewPostgreSQLUserRepository(db *sql.DB) *PostgreSQLUserRepository {
	return &PostgreSQLUserRepository{
		db: db,
	}
}

// CreateUser creates a new user in the database
func (r *PostgreSQLUserRepository) CreateUser(user *User) error {
	if user == nil {
		return errors.New("user is required")
	}
	
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	
	err := r.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
		user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	
	if err != nil {
		return err
	}
	
	return nil
}

// GetUserByUsername retrieves a user by username
func (r *PostgreSQLUserRepository) GetUserByUsername(username string) (*User, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}
	
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE username = $1 AND is_active = true
	`
	
	user := &User{}
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}
	
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgreSQLUserRepository) GetUserByEmail(email string) (*User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}
	
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE email = $1 AND is_active = true
	`
	
	user := &User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}
	
	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *PostgreSQLUserRepository) GetUserByID(id int64) (*User, error) {
	if id <= 0 {
		return nil, errors.New("valid user ID is required")
	}
	
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE id = $1
	`
	
	user := &User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}
	
	return user, nil
}

// UpdateUser updates an existing user
func (r *PostgreSQLUserRepository) UpdateUser(user *User) error {
	if user == nil {
		return errors.New("user is required")
	}
	
	if user.ID <= 0 {
		return errors.New("valid user ID is required")
	}
	
	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, updated_at = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at
	`
	
	user.UpdatedAt = time.Now()
	
	err := r.db.QueryRow(
		query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.UpdatedAt,
		user.IsActive,
	).Scan(&user.UpdatedAt)
	
	if err != nil {
		return err
	}
	
	return nil
}

// DeleteUser soft deletes a user by setting is_active to false
func (r *PostgreSQLUserRepository) DeleteUser(id int64) error {
	if id <= 0 {
		return errors.New("valid user ID is required")
	}
	
	query := `
		UPDATE users
		SET is_active = false, updated_at = $2
		WHERE id = $1
	`
	
	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return errors.New("user not found")
	}
	
	return nil
}