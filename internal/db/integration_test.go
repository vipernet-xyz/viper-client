//go:build integration
// +build integration

package db

import (
	"os"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/utils"
)

func TestDatabaseIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	// Load configuration
	config := utils.LoadConfig()

	// Allow some time for the database to initialize
	time.Sleep(2 * time.Second)

	// Test with config's DatabaseURL
	db, err := New(config.DatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database with config URL: %v", err)
	}
	defer db.Close()

	// Execute a simple query
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if result != 1 {
		t.Fatalf("Expected query result to be 1, got %d", result)
	}

	t.Log("Successfully executed query against PostgreSQL")
}
