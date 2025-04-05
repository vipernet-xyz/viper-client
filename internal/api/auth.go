package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/gin-gonic/gin"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	ProviderUserID string `json:"provider_user_id" binding:"required"`
	Email          string `json:"email"`
	Name           string `json:"name"`
}

// AuthResponse represents the response for auth operations
type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// AuthHandler handles authentication requests
type AuthHandler struct {
	DB          *db.DB
	AuthService *auth.Service
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(database *db.DB, authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		DB:          database,
		AuthService: authService,
	}
}

// LoginOrRegister handles login or registration based on provider user ID
// @Summary Login or register a user
// @Description Authenticates a user with their provider ID (Web3Auth) or creates a new account if the user doesn't exist
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body SwaggerLoginRequest true "Login/Register Data"
// @Success 200 {object} SwaggerAuthResponse "Successfully authenticated"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) LoginOrRegister(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate provider user ID
	req.ProviderUserID = strings.TrimSpace(req.ProviderUserID)
	if req.ProviderUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider_user_id is required"})
		return
	}

	// Try to find the user by provider ID
	user, err := h.DB.GetUserByProviderID(req.ProviderUserID)
	if err != nil {
		if !errors.Is(err, db.ErrUserNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query database"})
			return
		}

		// User not found, create a new one
		user, err = h.DB.CreateUser(req.ProviderUserID, req.Email, req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
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

// RegisterRoutes registers the auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.Engine) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", h.LoginOrRegister)
	}
}
