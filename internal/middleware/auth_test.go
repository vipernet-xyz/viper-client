package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock DB
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
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

// Ensure MockDB implements DatabaseInterface
var _ DatabaseInterface = (*MockDB)(nil)

// Generate a mock token
func generateMockToken(email, name string) string {
	// Create payload
	payload := map[string]interface{}{
		"email": email,
		"name":  name,
	}
	
	// Encode the payload
	jsonPayload, _ := json.Marshal(payload)
	base64Payload := base64.StdEncoding.EncodeToString(jsonPayload)
	// Replace standard base64 chars with URL-safe chars
	base64Payload = strings.ReplaceAll(base64Payload, "+", "-")
	base64Payload = strings.ReplaceAll(base64Payload, "/", "_")
	// Remove padding
	base64Payload = strings.TrimRight(base64Payload, "=")
	
	// Create a mock token
	return "header." + base64Payload + ".signature"
}

func setupTestRouter() (*gin.Engine, *MockDB) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create mock DB
	mockDB := new(MockDB)
	
	return router, mockDB
}

func TestAutoAuthMiddleware_ExistingUser(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Setup test route
	router.GET("/test", AutoAuthMiddleware(mockDB), func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})
	
	// Setup mock DB response
	mockUser := &models.User{
		ID:             1,
		ProviderUserID: "provider-123",
		Email:          "test@example.com",
		Name:           "Test User",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	mockDB.On("GetUserByEmail", "test@example.com").Return(mockUser, nil)
	
	// Create test request
	mockToken := generateMockToken("test@example.com", "Test User")
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Add("Authorization", "Bearer "+mockToken)
	resp := httptest.NewRecorder()
	
	// Perform the request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Body.String())
	mockDB.AssertExpectations(t)
}

func TestAutoAuthMiddleware_NewUser(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Setup test route
	router.GET("/test", AutoAuthMiddleware(mockDB), func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})
	
	// Setup mock DB responses
	mockDB.On("GetUserByEmail", "new@example.com").Return(nil, db.ErrUserNotFound)
	
	mockUser := &models.User{
		ID:             1,
		ProviderUserID: "",
		Email:          "new@example.com",
		Name:           "New User",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	mockDB.On("CreateUser", "", "new@example.com", "New User").Return(mockUser, nil)
	
	// Create test request
	mockToken := generateMockToken("new@example.com", "New User")
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Add("Authorization", "Bearer "+mockToken)
	resp := httptest.NewRecorder()
	
	// Perform the request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "success", resp.Body.String())
	mockDB.AssertExpectations(t)
}

func TestAutoAuthMiddleware_NoToken(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Setup test route
	router.GET("/test", AutoAuthMiddleware(mockDB), func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})
	
	// Create test request without token
	req, _ := http.NewRequest("GET", "/test", nil)
	resp := httptest.NewRecorder()
	
	// Perform the request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Authorization header is required")
}

func TestAutoAuthMiddleware_InvalidToken(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Setup test route
	router.GET("/test", AutoAuthMiddleware(mockDB), func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})
	
	// Create test request with invalid token
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Add("Authorization", "Bearer invalid.token")
	resp := httptest.NewRecorder()
	
	// Perform the request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid token")
}

func TestAutoAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Setup test route
	router.GET("/test", AutoAuthMiddleware(mockDB), func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})
	
	// Create test request with invalid authorization format
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Add("Authorization", "InvalidFormat token123")
	resp := httptest.NewRecorder()
	
	// Perform the request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid authorization format")
}

func TestAutoAuthMiddleware_DatabaseError(t *testing.T) {
	router, mockDB := setupTestRouter()
	
	// Setup test route
	router.GET("/test", AutoAuthMiddleware(mockDB), func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})
	
	// Setup mock DB responses with error
	mockDB.On("GetUserByEmail", "test@example.com").Return(nil, fmt.Errorf("database error"))
	
	// Create test request
	mockToken := generateMockToken("test@example.com", "Test User")
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Add("Authorization", "Bearer "+mockToken)
	resp := httptest.NewRecorder()
	
	// Perform the request
	router.ServeHTTP(resp, req)
	
	// Assertions
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "Failed to query database")
	mockDB.AssertExpectations(t)
}
