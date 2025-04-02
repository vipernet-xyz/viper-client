package db

import (
	"os"
	"testing"
)

func TestNewDB(t *testing.T) {
	// Skip test if no DATABASE_URL environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	// Test database connection
	db, err := New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// If we got here, the connection was successful
	t.Log("Successfully connected to database")
}
