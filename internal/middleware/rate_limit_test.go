package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create a new rate limiter with 2 requests per second and 3 burst capacity
	limiter := NewRateLimiter(2, 3)

	// Key for testing
	key := "test-key"

	// First three requests should be allowed (up to burst capacity)
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.allow(key), "Request %d should be allowed", i+1)
	}

	// Fourth request should be denied (exceeded burst capacity)
	assert.False(t, limiter.allow(key), "Fourth request should be denied")

	// Wait for tokens to replenish (at least 1 token)
	time.Sleep(500 * time.Millisecond) // Wait for 1 token (500ms * 2 tokens/s = 1 token)

	// Next request should be allowed
	assert.True(t, limiter.allow(key), "Request after wait should be allowed")

	// But the next one should be denied again
	assert.False(t, limiter.allow(key), "Second request after wait should be denied")
}

func TestIPRateLimiter(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply rate limiter middleware with very limiting settings
	// 1 request per second, 1 burst capacity
	router.Use(IPRateLimiter(1, 1))

	// Add a simple route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// First request should succeed
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345" // Set a client IP
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code, "First request should succeed")

	// Second request from same IP should be rate limited
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345" // Same client IP
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code, "Second request should be rate limited")

	// Request from different IP should succeed
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.2:12345" // Different client IP
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code, "Request from different IP should succeed")
}

func TestAppRateLimiter(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply rate limiter middleware with very limiting settings
	// 1 request per second, 1 burst capacity
	router.Use(AppRateLimiter(1, 1))

	// Add a simple route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// First request should succeed
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-App-ID", "test-app-1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code, "First request should succeed")

	// Second request from same app should be rate limited
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-App-ID", "test-app-1")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code, "Second request should be rate limited")

	// Request from different app should succeed
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.Header.Set("X-App-ID", "test-app-2")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code, "Request from different app should succeed")
}

func TestRateLimitMiddleware_MissingKey(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply custom rate limiter middleware
	limiter := NewRateLimiter(1, 1)
	router.Use(RateLimitMiddleware(limiter, func(c *gin.Context) string {
		return "" // Return empty key to simulate missing key
	}))

	// Add a simple route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// Request should be rejected due to missing key
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Request should be rejected due to missing key")
}
