package api

import (
	"net/http"

	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/middleware"
	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	DB *db.DB
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(database *db.DB) *AuthHandler {
	return &AuthHandler{
		DB: database,
	}
}

// GetCurrentUser returns the current authenticated user
// @Summary Get current user
// @Description Returns the currently authenticated user information
// @Tags Authentication
// @Produce json
// @Success 200 {object} models.User "User data"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
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

// RegisterRoutes registers the auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.Engine) {
	authGroup := router.Group("/auth")
	
	// Protected routes with auto authentication
	protected := authGroup.Group("")
	protected.Use(middleware.AutoAuthMiddleware(h.DB))
	{
		protected.GET("/me", h.GetCurrentUser)
	}
}
