//go:build integration
// +build integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddlewareIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup auth service
	config := auth.Config{
		SecretKey:     "integration-test-secret",
		TokenDuration: time.Hour,
	}
	authService := auth.NewAuthService(config)

	// Generate a valid token
	token, err := authService.GenerateToken("1", "provider-123", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Setup protected route with middleware
	protected := router.Group("/api")
	protected.Use(AuthMiddleware(authService))
	protected.GET("/protected", func(c *gin.Context) {
		userId := c.GetString("user_id")
		providerUserId := c.GetString("provider_user_id")
		email := c.GetString("email")
		name := c.GetString("name")

		// Assert that user info was correctly extracted from token
		if userId != "1" {
			t.Errorf("Expected user_id '1', got '%s'", userId)
		}
		if providerUserId != "provider-123" {
			t.Errorf("Expected provider_user_id 'provider-123', got '%s'", providerUserId)
		}
		if email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", email)
		}
		if name != "Test User" {
			t.Errorf("Expected name 'Test User', got '%s'", name)
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"userID": userId,
		})
	})

	// Test valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Test invalid token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// Test missing token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/protected", nil)
	router.ServeHTTP(w, req)

	// Assert response
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRateLimitingIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	// Setup Gin with rate limiting middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Use a very restrictive rate limiter for testing (1 req/sec, 2 burst)
	router.Use(IPRateLimiter(1, 2))

	// Add test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"time":   time.Now().String(),
		})
	})

	// Make several requests in quick succession
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345" // Set consistent source IP
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code, "First request should succeed")

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "Second request should succeed (within burst capacity)")

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "10.0.0.1:12345"
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code, "Third request should be rate limited")

	// Wait for token bucket to refill
	time.Sleep(1100 * time.Millisecond)

	// Should be able to make another request now
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/test", nil)
	req4.RemoteAddr = "10.0.0.1:12345"
	router.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code, "Request after waiting should succeed")
}
