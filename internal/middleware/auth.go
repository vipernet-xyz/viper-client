package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/auth"
	"github.com/illegalcall/viper-client/internal/db"
	"github.com/illegalcall/viper-client/internal/models"
)

// DatabaseInterface defines the interface for database operations needed by middleware
type DatabaseInterface interface {
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(providerUserID, email, name string) (*models.User, error)
}

// AutoAuthMiddleware creates a Gin middleware for automatic user creation based on token email
func AutoAuthMiddleware(database DatabaseInterface) gin.HandlerFunc {
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

		// Extract email from token
		claims, err := auth.ExtractEmailFromToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		// Check if user exists by email
		user, err := database.GetUserByEmail(claims.Email)
		if err != nil {
			if err == db.ErrUserNotFound {
				// User doesn't exist, create a new one
				user, err = database.CreateUser("", claims.Email, claims.Name)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
					return
				}
			} else {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to query database"})
				return
			}
		}

		// Set the user info in the context
		c.Set("user_id", user.ID)
		c.Set("provider_user_id", user.ProviderUserID)
		c.Set("email", user.Email)
		c.Set("name", user.Name)
		c.Set("user", user)

		// Proceed to the next handler
		c.Next()
	}
}
