package models

import (
	"time"
)

// App represents a decentralized application registered in the system
type App struct {
	ID             int       `json:"id"`
	AppIdentifier  string    `json:"app_identifier"`
	UserID         int       `json:"user_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	AllowedOrigins []string  `json:"allowed_origins,omitempty"`
	AllowedChains  []int     `json:"allowed_chains,omitempty"`
	APIKeyHash     string    `json:"api_key_hash"`
	RateLimit      int       `json:"rate_limit"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
