package api

import (
	"net/http"
	"strconv"

	"github.com/dhruvsharma/viper-client/internal/apps"
	"github.com/gin-gonic/gin"
)

// AppsHandler handles app-related API requests
type AppsHandler struct {
	appsService *apps.Service
}

// NewAppsHandler creates a new apps handler
func NewAppsHandler(appsService *apps.Service) *AppsHandler {
	return &AppsHandler{
		appsService: appsService,
	}
}

// RegisterRoutes registers the app-related routes
func (h *AppsHandler) RegisterRoutes(router *gin.Engine) {
	appRoutes := router.Group("/api/apps")

	// Routes require authentication
	appRoutes.POST("/", h.createApp)
	appRoutes.GET("/:id", h.getApp)
	appRoutes.GET("/", h.getUserApps)
	appRoutes.PUT("/:id", h.updateApp)
	appRoutes.DELETE("/:id", h.deleteApp)
}

// createApp handles the creation of a new app
func (h *AppsHandler) createApp(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		Name           string   `json:"name" binding:"required"`
		Description    string   `json:"description"`
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedChains  []int    `json:"allowed_chains" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data: " + err.Error(),
		})
		return
	}

	createReq := apps.CreateAppRequest{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		AllowedOrigins: req.AllowedOrigins,
		AllowedChains:  req.AllowedChains,
	}

	result, err := h.appsService.CreateApp(createReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create app: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"app":     result.App,
		"api_key": result.APIKey,
		"message": "App created successfully. Keep your API key safe, as it won't be shown again.",
	})
}

// getApp retrieves an app by ID
func (h *AppsHandler) getApp(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get app ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app ID",
		})
		return
	}

	app, err := h.appsService.GetApp(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "App not found: " + err.Error(),
		})
		return
	}

	// Check if the app belongs to the authenticated user
	if app.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this app",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"app": app,
	})
}

// getUserApps retrieves all apps belonging to the authenticated user
func (h *AppsHandler) getUserApps(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	apps, err := h.appsService.GetAppsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve apps: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"apps": apps,
	})
}

// updateApp updates an app by ID
func (h *AppsHandler) updateApp(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get app ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app ID",
		})
		return
	}

	var req apps.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data: " + err.Error(),
		})
		return
	}

	// Update the app
	updatedApp, err := h.appsService.UpdateApp(id, userID, req)
	if err != nil {
		if err.Error() == "app not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "App not found",
			})
			return
		} else if err.Error() == "access denied: app does not belong to the user" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied to this app",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update app: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"app":     updatedApp,
		"message": "App updated successfully",
	})
}

// deleteApp deletes an app by ID
func (h *AppsHandler) deleteApp(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get app ID from URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid app ID",
		})
		return
	}

	// Delete the app
	err = h.appsService.DeleteApp(id, userID)
	if err != nil {
		if err.Error() == "app not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "App not found",
			})
			return
		} else if err.Error() == "access denied: app does not belong to the user" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied to this app",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete app: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "App deleted successfully",
	})
}
