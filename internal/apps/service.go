package apps

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"

	"github.com/illegalcall/viper-client/internal/models"
	"github.com/lib/pq"
)

// Service provides functionality for managing decentralized applications
type Service struct {
	db *sql.DB
}

// NewService creates a new apps service with the provided database connection
func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

// CreateAppRequest contains the data needed to create a new app
type CreateAppRequest struct {
	UserID         int      `json:"user_id"`
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	AllowedChains  []int    `json:"allowed_chains,omitempty"`
}

// CreateAppResponse contains the data returned after creating a new app
type CreateAppResponse struct {
	App    models.App `json:"app"`
	APIKey string     `json:"api_key"`
}

// GenerateAPIKey generates a random API key
func (s *Service) GenerateAPIKey() (string, error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode to base64 for a user-friendly key
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateApp creates a new decentralized application
func (s *Service) CreateApp(req CreateAppRequest) (*CreateAppResponse, error) {

	// Generate API key and hash
	apiKey, err := s.GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	// Default rate limit if not specified
	rateLimit := 10000

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO apps (api_key, user_id, name, description, allowed_origins, allowed_chains, rate_limit)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	var app models.App
	app.APIKey = apiKey
	app.UserID = req.UserID
	app.Name = req.Name
	app.Description = req.Description
	app.AllowedOrigins = req.AllowedOrigins
	app.AllowedChains = models.IntArray(req.AllowedChains)
	app.RateLimit = rateLimit

	err = tx.QueryRow(
		query,
		app.APIKey,
		app.UserID,
		app.Name,
		app.Description,
		pq.Array(app.AllowedOrigins),
		app.AllowedChains,
		app.RateLimit,
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &CreateAppResponse{
		App:    app,
		APIKey: apiKey,
	}, nil
}

// GetApp retrieves an app by its ID
func (s *Service) GetApp(id int) (*models.App, error) {
	query := `
		SELECT id, api_key, user_id, name, description, allowed_origins, allowed_chains, 
		       rate_limit, created_at, updated_at
		FROM apps
		WHERE id = $1
	`

	var app models.App
	var description sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&app.ID,
		&app.APIKey,
		&app.UserID,
		&app.Name,
		&description,
		pq.Array(&app.AllowedOrigins),
		&app.AllowedChains,
		&app.RateLimit,
		&app.CreatedAt,
		&app.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("app not found")
		}
		return nil, err
	}

	if description.Valid {
		app.Description = description.String
	}

	return &app, nil
}

// GetAppsByUserID retrieves all apps belonging to a user
func (s *Service) GetAppsByUserID(userID int) ([]models.App, error) {
	query := `
		SELECT id, api_key, user_id, name, description, allowed_origins, allowed_chains, 
		       rate_limit, created_at, updated_at
		FROM apps
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []models.App
	for rows.Next() {
		var app models.App
		var description sql.NullString

		err := rows.Scan(
			&app.ID,
			&app.APIKey,
			&app.UserID,
			&app.Name,
			&description,
			pq.Array(&app.AllowedOrigins),
			&app.AllowedChains,
			&app.RateLimit,
			&app.CreatedAt,
			&app.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			app.Description = description.String
		}

		apps = append(apps, app)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return apps, nil
}

// UpdateAppRequest contains the data needed to update an app
type UpdateAppRequest struct {
	Name           string   `json:"name,omitempty"`
	Description    string   `json:"description,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	AllowedChains  []int    `json:"allowed_chains,omitempty"`
	RateLimit      int      `json:"rate_limit,omitempty"`
}

// UpdateApp updates an existing app
func (s *Service) UpdateApp(id int, userID int, req UpdateAppRequest) (*models.App, error) {
	// First check if the app exists and belongs to the user
	app, err := s.GetApp(id)
	if err != nil {
		return nil, err
	}

	if app.UserID != userID {
		return nil, errors.New("access denied: app does not belong to the user")
	}

	// Set update values, keeping existing values if not provided
	name := app.Name
	if req.Name != "" {
		name = req.Name
	}

	description := app.Description
	if req.Description != "" {
		description = req.Description
	}

	allowedOrigins := app.AllowedOrigins
	if req.AllowedOrigins != nil {
		allowedOrigins = req.AllowedOrigins
	}

	allowedChains := app.AllowedChains
	if req.AllowedChains != nil {
		allowedChains = models.IntArray(req.AllowedChains)
	}

	rateLimit := app.RateLimit
	if req.RateLimit > 0 {
		rateLimit = req.RateLimit
	}

	// Update the app
	query := `
		UPDATE apps
		SET name = $1, description = $2, allowed_origins = $3, allowed_chains = $4, 
		    rate_limit = $5, updated_at = NOW()
		WHERE id = $6 AND user_id = $7
		RETURNING id, api_key, user_id, name, description, allowed_origins, allowed_chains, 
		         rate_limit, created_at, updated_at
	`

	var updatedApp models.App
	var dbDescription sql.NullString

	err = s.db.QueryRow(
		query,
		name,
		description,
		pq.Array(allowedOrigins),
		allowedChains,
		rateLimit,
		id,
		userID,
	).Scan(
		&updatedApp.ID,
		&updatedApp.APIKey,
		&updatedApp.UserID,
		&updatedApp.Name,
		&dbDescription,
		pq.Array(&updatedApp.AllowedOrigins),
		&updatedApp.AllowedChains,
		&updatedApp.RateLimit,
		&updatedApp.CreatedAt,
		&updatedApp.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("app not found or user does not have permission")
		}
		return nil, err
	}

	if dbDescription.Valid {
		updatedApp.Description = dbDescription.String
	}

	return &updatedApp, nil
}

// DeleteApp deletes an app by its ID
func (s *Service) DeleteApp(id int, userID int) error {
	// First check if the app exists and belongs to the user
	app, err := s.GetApp(id)
	if err != nil {
		return err
	}

	if app.UserID != userID {
		return errors.New("access denied: app does not belong to the user")
	}

	// Delete the app
	query := `
		DELETE FROM apps
		WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("app not found or user does not have permission")
	}

	return nil
}

// ValidateAPIKey validates an API key against the stored hash
func (s *Service) ValidateAPIKey(apiKey string) (bool, error) {
	query := `
		SELECT 1
		FROM apps
		WHERE api_key = $1
	`

	var exists bool
	err := s.db.QueryRow(query, apiKey).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}

// GetAppByAPIKey retrieves an app by its API key
func (s *Service) GetAppByAPIKey(apiKey string) (*models.App, error) {
	query := `
		SELECT id, api_key, user_id, name, description, allowed_origins, allowed_chains, 
		       rate_limit, created_at, updated_at
		FROM apps
		WHERE api_key = $1
	`

	var app models.App
	var description sql.NullString

	err := s.db.QueryRow(query, apiKey).Scan(
		&app.ID,
		&app.APIKey,
		&app.UserID,
		&app.Name,
		&description,
		pq.Array(&app.AllowedOrigins),
		&app.AllowedChains,
		&app.RateLimit,
		&app.CreatedAt,
		&app.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("app not found")
		}
		return nil, err
	}

	if description.Valid {
		app.Description = description.String
	}

	return &app, nil
}
