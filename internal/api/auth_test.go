package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

// DatabaseInterface defines the interface for database operations
type DatabaseInterface interface {
	GetUserByID(id int) (*models.User, error)
	GetUserByProviderID(providerID string) (*models.User, error)
	CreateUser(providerUserID, email, name string) (*models.User, error)
	UpdateUser(id int, email, name string) (*models.User, error)
	Close() error
	Ping() error
	MigrateDB(migrationsPath string) error
}

// MockDB is a mock implementation of the database interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetUserByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockDB) GetUserByProviderID(providerID string) (*models.User, error) {
	args := m.Called(providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockDB) CreateUser(providerUserID, email, name string) (*models.User, error) {
	args := m.Called(providerUserID, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockDB) UpdateUser(id int, email, name string) (*models.User, error) {
	args := m.Called(id, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockDB) MigrateDB(migrationsPath string) error {
	return nil
}

func (m *MockDB) Close() error {
	return nil
}

func (m *MockDB) Ping() error {
	return nil
}

// AuthHandler with the database interface
type TestAuthHandler struct {
	DB          DatabaseInterface
	AuthService *auth.Service
}

func (h *TestAuthHandler) LoginOrRegister(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Try to find the user by provider ID
	user, err := h.DB.GetUserByProviderID(req.ProviderUserID)
	if err != nil {
		if err == db.ErrUserNotFound {
			// User not found, create a new one
			user, err = h.DB.CreateUser(req.ProviderUserID, req.Email, req.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query database"})
			return
		}
	} else if req.Email != "" || req.Name != "" {
		// User found, update if email or name provided
		if req.Email != "" && req.Email != user.Email || req.Name != "" && req.Name != user.Name {
			userEmail := req.Email
			if userEmail == "" {
				userEmail = user.Email
			}
			userName := req.Name
			if userName == "" {
				userName = user.Name
			}

			user, err = h.DB.UpdateUser(user.ID, userEmail, userName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
				return
			}
		}
	}

	// Generate a JWT token
	token, err := h.AuthService.GenerateToken(
		strconv.Itoa(user.ID),
		user.ProviderUserID,
		user.Email,
		user.Name,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Return the token and user
	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

func setupTestRouter() (*gin.Engine, *MockDB, *auth.Service) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockDB := new(MockDB)
	authService := auth.NewAuthService(auth.Config{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
	})

	return router, mockDB, authService
}

func TestLoginOrRegister_NewUser(t *testing.T) {
	router, mockDB, authService := setupTestRouter()

	// Create handler with mockDB directly
	handler := &TestAuthHandler{
		DB:          mockDB,
		AuthService: authService,
	}

	// Mock DB behavior - user not found, then created
	mockDB.On("GetUserByProviderID", "new-provider-id").Return(nil, db.ErrUserNotFound)
	mockDB.On("CreateUser", "new-provider-id", "new@example.com", "New User").Return(&models.User{
		ID:             1,
		ProviderUserID: "new-provider-id",
		Email:          "new@example.com",
		Name:           "New User",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil)

	// Setup route
	router.POST("/auth/login", handler.LoginOrRegister)

	// Prepare request
	reqBody := LoginRequest{
		ProviderUserID: "new-provider-id",
		Email:          "new@example.com",
		Name:           "New User",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token to be non-empty")
	}
	if response.User == nil {
		t.Error("Expected user to be non-nil")
	}
	if response.User.ID != 1 {
		t.Errorf("Expected user ID to be 1, got %d", response.User.ID)
	}
	if response.User.ProviderUserID != "new-provider-id" {
		t.Errorf("Expected provider user ID to be 'new-provider-id', got '%s'", response.User.ProviderUserID)
	}

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestLoginOrRegister_ExistingUser(t *testing.T) {
	router, mockDB, authService := setupTestRouter()

	// Create handler with mockDB directly
	handler := &TestAuthHandler{
		DB:          mockDB,
		AuthService: authService,
	}

	// Mock DB behavior - user found
	existingUser := &models.User{
		ID:             2,
		ProviderUserID: "existing-provider-id",
		Email:          "existing@example.com",
		Name:           "Existing User",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	mockDB.On("GetUserByProviderID", "existing-provider-id").Return(existingUser, nil)

	// Setup route
	router.POST("/auth/login", handler.LoginOrRegister)

	// Prepare request
	reqBody := LoginRequest{
		ProviderUserID: "existing-provider-id",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token to be non-empty")
	}
	if response.User == nil {
		t.Error("Expected user to be non-nil")
	}
	if response.User.ID != 2 {
		t.Errorf("Expected user ID to be 2, got %d", response.User.ID)
	}
	if response.User.ProviderUserID != "existing-provider-id" {
		t.Errorf("Expected provider user ID to be 'existing-provider-id', got '%s'", response.User.ProviderUserID)
	}

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestLoginOrRegister_UpdateUser(t *testing.T) {
	router, mockDB, authService := setupTestRouter()

	// Create handler with mockDB directly
	handler := &TestAuthHandler{
		DB:          mockDB,
		AuthService: authService,
	}

	// Mock DB behavior - user found and updated
	existingUser := &models.User{
		ID:             3,
		ProviderUserID: "update-provider-id",
		Email:          "old@example.com",
		Name:           "Old User",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	mockDB.On("GetUserByProviderID", "update-provider-id").Return(existingUser, nil)
	mockDB.On("UpdateUser", 3, "new@example.com", "New User").Return(&models.User{
		ID:             3,
		ProviderUserID: "update-provider-id",
		Email:          "new@example.com",
		Name:           "New User",
		CreatedAt:      existingUser.CreatedAt,
		UpdatedAt:      time.Now(),
	}, nil)

	// Setup route
	router.POST("/auth/login", handler.LoginOrRegister)

	// Prepare request
	reqBody := LoginRequest{
		ProviderUserID: "update-provider-id",
		Email:          "new@example.com",
		Name:           "New User",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.User.Email != "new@example.com" {
		t.Errorf("Expected email to be 'new@example.com', got '%s'", response.User.Email)
	}
	if response.User.Name != "New User" {
		t.Errorf("Expected name to be 'New User', got '%s'", response.User.Name)
	}

	// Verify mock expectations
	mockDB.AssertExpectations(t)
}

func TestLoginOrRegister_InvalidRequest(t *testing.T) {
	router, _, authService := setupTestRouter()
	handler := NewAuthHandler(nil, authService)

	// Setup route
	router.POST("/auth/login", handler.LoginOrRegister)

	// Prepare request with empty provider ID
	reqBody := LoginRequest{
		ProviderUserID: "",
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}
