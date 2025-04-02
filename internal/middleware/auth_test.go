package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/gin-gonic/gin"
)

func setupAuth() (*auth.Service, string) {
	// Setup auth service
	config := auth.Config{
		SecretKey:     "test-secret-key",
		TokenDuration: time.Hour,
	}
	authService := auth.NewAuthService(config)

	// Generate a valid token
	token, _ := authService.GenerateToken("1", "provider-123", "test@example.com", "Test User")

	return authService, token
}

func setupRouter(authService *auth.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Protected route
	protected := router.Group("/api")
	protected.Use(AuthMiddleware(authService))
	protected.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"userID": c.GetString("user_id"),
		})
	})

	// Web3Auth protected route
	web3Protected := router.Group("/api/web3")
	web3Protected.Use(Web3AuthMiddleware(authService))
	web3Protected.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"userID": c.GetString("user_id"),
		})
	})

	return router
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	authService, token := setupAuth()
	router := setupRouter(authService)

	// Create a request to a protected route with a valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	authService, _ := setupAuth()
	router := setupRouter(authService)

	// Create a request to a protected route without a token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	authService, _ := setupAuth()
	router := setupRouter(authService)

	// Create a request to a protected route with an invalid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	authService, token := setupAuth()
	router := setupRouter(authService)

	// Create a request with invalid Authorization format
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Token "+token) // Wrong format
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestWeb3AuthMiddleware_ValidToken(t *testing.T) {
	authService, token := setupAuth()
	router := setupRouter(authService)

	// Create a request to a Web3Auth protected route with a valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/web3/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
