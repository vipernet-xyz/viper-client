//go:build integration
// +build integration

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/apps"
	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestEnv(t *testing.T) (*gin.Engine, *db.DB, *auth.Service, int) {
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

	// Create a test user
	uniqueProviderID := "apps-integration-test-" + time.Now().Format("20060102150405")
	user, err := database.CreateUser(uniqueProviderID, "apps-test@example.com", "Apps Test User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Generate a token for the test user - we don't use this token but it's kept for reference
	// in case we need to add Authorization header tests later
	_, err = authService.GenerateToken(
		strconv.Itoa(user.ID),
		user.ProviderUserID,
		user.Email,
		user.Name,
	)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Add middleware to set user info for testing
	router.Use(func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Set("provider_user_id", user.ProviderUserID)
		c.Set("email", user.Email)
		c.Set("name", user.Name)
		c.Next()
	})

	return router, database, authService, user.ID
}

func cleanupTestUser(t *testing.T, database *db.DB, userID int) {
	// Clean up test user and associated apps
	_, err := database.DB.Exec("DELETE FROM apps WHERE user_id = $1", userID)
	if err != nil {
		t.Fatalf("Failed to clean up test apps: %v", err)
	}

	_, err = database.DB.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Fatalf("Failed to clean up test user: %v", err)
	}
}

func TestAppsIntegration(t *testing.T) {
	router, database, _, userID := setupTestEnv(t)
	defer database.Close()
	defer cleanupTestUser(t, database, userID)

	// Setup apps service and handler
	appsService := apps.NewService(database.DB)
	appsHandler := NewAppsHandler(appsService)

	// Register app routes
	appRoutes := router.Group("/api/apps")
	appRoutes.POST("/", appsHandler.createApp)
	appRoutes.GET("/:id", appsHandler.getApp)
	appRoutes.GET("/", appsHandler.getUserApps)
	appRoutes.PUT("/:id", appsHandler.updateApp)
	appRoutes.DELETE("/:id", appsHandler.deleteApp)

	// Test creating an app
	createReq := struct {
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedChains  []int    `json:"allowed_chains"`
	}{
		Name:           "Test App",
		Description:    "This is a test app",
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedChains:  []int{1, 2, 3},
	}
	createBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/apps/", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify create response
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		App struct {
			ID             int      `json:"id"`
			AppIdentifier  string   `json:"app_identifier"`
			UserID         int      `json:"user_id"`
			Name           string   `json:"name"`
			Description    string   `json:"description"`
			AllowedOrigins []string `json:"allowed_origins"`
			AllowedChains  []int    `json:"allowed_chains"`
		} `json:"app"`
		APIKey  string `json:"api_key"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)
	assert.Equal(t, "Test App", createResp.App.Name)
	assert.Equal(t, "This is a test app", createResp.App.Description)
	assert.Equal(t, userID, createResp.App.UserID)
	assert.NotEmpty(t, createResp.APIKey)

	appID := createResp.App.ID

	// Test getting the created app
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/apps/%d", appID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify get response
	assert.Equal(t, http.StatusOK, w.Code)

	var getResp struct {
		App struct {
			ID             int      `json:"id"`
			AppIdentifier  string   `json:"app_identifier"`
			UserID         int      `json:"user_id"`
			Name           string   `json:"name"`
			Description    string   `json:"description"`
			AllowedOrigins []string `json:"allowed_origins"`
			AllowedChains  []int    `json:"allowed_chains"`
		} `json:"app"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.NoError(t, err)
	assert.Equal(t, appID, getResp.App.ID)
	assert.Equal(t, "Test App", getResp.App.Name)

	// Test getting all user apps
	req, _ = http.NewRequest("GET", "/api/apps/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify list response
	assert.Equal(t, http.StatusOK, w.Code)

	var listResp struct {
		Apps []struct {
			ID            int    `json:"id"`
			AppIdentifier string `json:"app_identifier"`
			UserID        int    `json:"user_id"`
			Name          string `json:"name"`
		} `json:"apps"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listResp)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Apps), 1)

	// Test updating the app
	updateReq := apps.UpdateAppRequest{
		Name:           "Updated App Name",
		Description:    "Updated description",
		AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
		RateLimit:      15000,
	}
	updateBody, _ := json.Marshal(updateReq)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/apps/%d", appID), bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify update response
	assert.Equal(t, http.StatusOK, w.Code)

	var updateResp struct {
		App struct {
			ID             int      `json:"id"`
			Name           string   `json:"name"`
			Description    string   `json:"description"`
			AllowedOrigins []string `json:"allowed_origins"`
			RateLimit      int      `json:"rate_limit"`
		} `json:"app"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &updateResp)
	assert.NoError(t, err)
	assert.Equal(t, "Updated App Name", updateResp.App.Name)
	assert.Equal(t, "Updated description", updateResp.App.Description)
	assert.Equal(t, 15000, updateResp.App.RateLimit)
	assert.Len(t, updateResp.App.AllowedOrigins, 2)

	// Test deleting the app
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/apps/%d", appID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify delete response
	assert.Equal(t, http.StatusOK, w.Code)

	var deleteResp struct {
		Message string `json:"message"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &deleteResp)
	assert.NoError(t, err)
	assert.Equal(t, "App deleted successfully", deleteResp.Message)

	// Verify app is deleted by trying to get it
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/apps/%d", appID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return not found
	assert.Equal(t, http.StatusNotFound, w.Code)
}
