package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a token bucket rate limiting algorithm
type RateLimiter struct {
	mu           sync.Mutex
	buckets      map[string]*bucket
	rate         int           // Tokens per second
	capacity     int           // Maximum bucket capacity
	cleanupEvery time.Duration // How often to clean up old buckets
	lastCleanup  time.Time
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter with specified rate and capacity
func NewRateLimiter(rate, capacity int) *RateLimiter {
	return &RateLimiter{
		buckets:      make(map[string]*bucket),
		rate:         rate,
		capacity:     capacity,
		cleanupEvery: 10 * time.Minute,
		lastCleanup:  time.Now(),
	}
}

// RateLimitMiddleware returns a Gin middleware that rate limits requests
// The key function extracts the rate limiting key from the request context (e.g., app ID or IP)
func RateLimitMiddleware(limiter *RateLimiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Cannot identify client for rate limiting",
			})
			return
		}

		// Allow the request if there are enough tokens
		if !limiter.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

// allow checks if a request should be allowed and takes a token if it is
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up old buckets periodically
	if now.Sub(rl.lastCleanup) > rl.cleanupEvery {
		for k, b := range rl.buckets {
			if now.Sub(b.lastRefill) > rl.cleanupEvery {
				delete(rl.buckets, k)
			}
		}
		rl.lastCleanup = now
	}

	// Get or create the token bucket for this key
	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     float64(rl.capacity),
			lastRefill: now,
		}
		rl.buckets[key] = b
	} else {
		// Refill tokens based on time elapsed
		elapsed := now.Sub(b.lastRefill).Seconds()
		newTokens := float64(rl.rate) * elapsed
		b.tokens = min(float64(rl.capacity), b.tokens+newTokens)
		b.lastRefill = now
	}

	// Attempt to take a token
	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// min returns the smaller of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// IPRateLimiter creates a rate limiter middleware based on client IP
func IPRateLimiter(rate, capacity int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, capacity)
	return RateLimitMiddleware(limiter, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// AppRateLimiter creates a rate limiter middleware based on app identifier
func AppRateLimiter(rate, capacity int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, capacity)
	return RateLimitMiddleware(limiter, func(c *gin.Context) string {
		return c.GetHeader("X-App-ID")
	})
}
