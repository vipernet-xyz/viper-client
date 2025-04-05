package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a Gin middleware for JWT authentication
func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		// Check if the Authorization header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format, expected 'Bearer {token}'"})
			return
		}

		// Extract the token from the Authorization header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate the token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Convert the user ID from string to integer
		userID, err := strconv.Atoi(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}

		// Set the user info in the context
		c.Set("user_id", userID)
		c.Set("provider_user_id", claims.ProviderUserID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("user_claims", claims)

		// Proceed to the next handler
		c.Next()
	}
}

// Web3AuthMiddleware creates a Gin middleware for Web3Auth token validation
func Web3AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		// Check if the Authorization header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format, expected 'Bearer {token}'"})
			return
		}

		// Extract the token from the Authorization header
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Simulate token verification from Web3Auth
		claims, err := authService.SimulateWeb3AuthToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Web3Auth token"})
			return
		}

		// Convert the user ID from string to integer
		userID, err := strconv.Atoi(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}

		// Set the user info in the context
		c.Set("user_id", userID)
		c.Set("provider_user_id", claims.ProviderUserID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("user_claims", claims)

		// Proceed to the next handler
		c.Next()
	}
}
