package utils

import (
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecretKey   string
	JWTExpiryHours int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/viperdb?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-should-be-long-and-secure" // Default for development only
	}

	// JWT expiry in hours
	jwtExpiry := 24 // Default to 24 hours

	return &Config{
		Port:           port,
		DatabaseURL:    dbURL,
		JWTSecretKey:   jwtSecret,
		JWTExpiryHours: jwtExpiry,
	}
}

// GetJWTDuration returns the JWT token duration
func (c *Config) GetJWTDuration() time.Duration {
	return time.Duration(c.JWTExpiryHours) * time.Hour
}
