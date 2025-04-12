package db

import (
	"database/sql"
	"errors"

	"github.com/dhruvsharma/viper-client/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(id int) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		SELECT id, provider_user_id, email, name, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.ProviderUserID,
		&user.Email,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByProviderID retrieves a user by provider ID
func (db *DB) GetUserByProviderID(providerID string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		SELECT id, provider_user_id, email, name, created_at, updated_at
		FROM users
		WHERE provider_user_id = $1
	`, providerID).Scan(
		&user.ID,
		&user.ProviderUserID,
		&user.Email,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user
func (db *DB) CreateUser(providerUserID, email, name string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		INSERT INTO users (provider_user_id, email, name)
		VALUES ($1, $2, $3)
		RETURNING id, provider_user_id, email, name, created_at, updated_at
	`, providerUserID, email, name).Scan(
		&user.ID,
		&user.ProviderUserID,
		&user.Email,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates a user
func (db *DB) UpdateUser(id int, email, name string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		UPDATE users
		SET email = $2, name = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, provider_user_id, email, name, created_at, updated_at
	`, id, email, name).Scan(
		&user.ID,
		&user.ProviderUserID,
		&user.Email,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(`
		SELECT id, provider_user_id, email, name, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.ProviderUserID,
		&user.Email,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
