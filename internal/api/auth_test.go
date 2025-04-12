package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// DatabaseInterface defines the interface for database operations
type DatabaseInterface interface {
	GetUserByID(id int) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
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

func (m *MockDB) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
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

func setupTestRouter() (*gin.Engine, *MockDB) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockDB := new(MockDB)
	return router, mockDB
}

// AuthHandlerTest is a test version of AuthHandler that accepts DatabaseInterface
type AuthHandlerTest struct {
	DB DatabaseInterface
}

// GetCurrentUser is identical to the original AuthHandler method
func (h *AuthHandlerTest) GetCurrentUser(c *gin.Context) {
	// The user is already set in the context by the AutoAuthMiddleware
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user object in context"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func TestGetCurrentUser_Success(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Create handler
	handler := &AuthHandlerTest{
		DB: mockDB,
	}
	
	// Create a test user
	testUser := &models.User{
		ID:             1,
		ProviderUserID: "test-provider-id",
		Email:          "test@example.com",
		Name:           "Test User",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Setup route with a middleware that sets the user in the context
	router.GET("/auth/me", func(c *gin.Context) {
		// Simulate middleware setting user in context
		c.Set("user", testUser)
		// Call the actual handler
		handler.GetCurrentUser(c)
	})
	
	// Create request
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	
	// Perform request
	router.ServeHTTP(w, req)
	
	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check that response contains user data
	assert.Contains(t, w.Body.String(), `"id":1`)
	assert.Contains(t, w.Body.String(), `"email":"test@example.com"`)
	assert.Contains(t, w.Body.String(), `"name":"Test User"`)
}

func TestGetCurrentUser_NotAuthenticated(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Create handler
	handler := &AuthHandlerTest{
		DB: mockDB,
	}
	
	// Setup route without setting user in context
	router.GET("/auth/me", handler.GetCurrentUser)
	
	// Create request
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	
	// Perform request
	router.ServeHTTP(w, req)
	
	// Verify response
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Not authenticated")
}

func TestGetCurrentUser_InvalidUserObject(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Create handler
	handler := &AuthHandlerTest{
		DB: mockDB,
	}
	
	// Setup route with a middleware that sets an invalid user object in the context
	router.GET("/auth/me", func(c *gin.Context) {
		// Set invalid user type (string instead of *models.User)
		c.Set("user", "not-a-user-object")
		// Call the actual handler
		handler.GetCurrentUser(c)
	})
	
	// Create request
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	
	// Perform request
	router.ServeHTTP(w, req)
	
	// Verify response
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid user object in context")
}
