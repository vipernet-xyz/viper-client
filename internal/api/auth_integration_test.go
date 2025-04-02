//go:build integration
// +build integration

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/gin-gonic/gin"
)

func TestAuthIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	// Skip if no DATABASE_URL environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	// Connect to the database
	database, err := db.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Get absolute path to migrations directory
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Go up one directory to project root
	projectRoot := filepath.Dir(filepath.Dir(pwd))
	migrationsPath := "file://" + filepath.Join(projectRoot, "migrations")
	t.Logf("Migrations path: %s", migrationsPath)

	// Run migrations
	if err := database.MigrateDB(migrationsPath); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup auth service
	authService := auth.NewAuthService(auth.Config{
		SecretKey:     "integration-test-secret",
		TokenDuration: time.Hour,
	})

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup auth handler
	authHandler := NewAuthHandler(database, authService)
	authHandler.RegisterRoutes(router)

	// Test login/register with a new user
	uniqueProviderID := "integration-test-" + time.Now().Format("20060102150405")
	loginReq := LoginRequest{
		ProviderUserID: uniqueProviderID,
		Email:          "integration@example.com",
		Name:           "Integration Test User",
	}

	// First request - should create a new user
	loginBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var firstResponse AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if firstResponse.Token == "" {
		t.Error("Expected token to be non-empty")
	}
	if firstResponse.User == nil {
		t.Error("Expected user to be non-nil")
	}
	if firstResponse.User.ProviderUserID != uniqueProviderID {
		t.Errorf("Expected provider user ID to be '%s', got '%s'", uniqueProviderID, firstResponse.User.ProviderUserID)
	}

	// Second request - should retrieve the existing user
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var secondResponse AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &secondResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if secondResponse.User.ID != firstResponse.User.ID {
		t.Errorf("Expected same user ID %d, got %d", firstResponse.User.ID, secondResponse.User.ID)
	}

	// Update user request
	updateReq := LoginRequest{
		ProviderUserID: uniqueProviderID,
		Email:          "updated@example.com",
		Name:           "Updated User",
	}
	updateBody, _ := json.Marshal(updateReq)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var updateResponse AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &updateResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updateResponse.User.Email != "updated@example.com" {
		t.Errorf("Expected email to be 'updated@example.com', got '%s'", updateResponse.User.Email)
	}
	if updateResponse.User.Name != "Updated User" {
		t.Errorf("Expected name to be 'Updated User', got '%s'", updateResponse.User.Name)
	}

	// Clean up test user
	_, err = database.DB.Exec("DELETE FROM users WHERE provider_user_id = $1", uniqueProviderID)
	if err != nil {
		t.Fatalf("Failed to clean up test user: %v", err)
	}
}
