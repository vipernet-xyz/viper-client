package models

import "time"

// User represents a user in the system
type User struct {
	ID             int       `json:"id"`
	ProviderUserID string    `json:"provider_user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
