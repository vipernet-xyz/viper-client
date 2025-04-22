package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/apps"
)

// CreateAppRequest represents the request to create a new app
// @Description Request data for creating a new application
type CreateAppRequest struct {
	// Application name
	// @example "My Blockchain App"
	Name string `json:"name" binding:"required"`

	// Application description
	// @example "A decentralized exchange app"
	Description string `json:"description"`

	// List of allowed origins for CORS
	// @example ["https://myapp.com", "https://dev.myapp.com"]
	AllowedOrigins []string `json:"allowed_origins"`

	// List of allowed blockchain chain IDs
	// @example [1, 137, 56]
	AllowedChains []int `json:"allowed_chains" binding:"required"`
}

// CreateAppResponse represents the response for app creation
// @Description Response data after creating a new application
type CreateAppResponse struct {
	// The created application object
	App interface{} `json:"app"`

	// Generated API key (only shown once)
	// @example "vpr_12345abcdef67890"
	APIKey string `json:"api_key"`

	// Success message
	// @example "App created successfully. Keep your API key safe, as it won't be shown again."
	Message string `json:"message"`
}

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
func (h *AppsHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Routes require authentication
	router.POST("/apps/", h.createApp)
	router.GET("/apps/:id", h.getApp)
	router.GET("/apps/", h.getUserApps)
	router.PUT("/apps/:id", h.updateApp)
	router.DELETE("/apps/:id", h.deleteApp)
}

// createApp handles the creation of a new app
// @Summary Create a new application
// @Description Creates a new application for the authenticated user
// @Tags Apps
// @Accept json
// @Produce json
// @Param request body SwaggerCreateAppRequest true "Application Details"
// @Success 201 {object} SwaggerCreateAppResponse "Application created successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/apps/ [post]
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
// @Summary Get an application by ID
// @Description Retrieves a specific application by its ID
// @Tags Apps
// @Accept json
// @Produce json
// @Param id path int true "Application ID"
// @Success 200 {object} AppResponse "Application details"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Not found"
// @Security BearerAuth
// @Router /api/apps/{id} [get]
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
// @Summary Get all applications for a user
// @Description Retrieves all applications owned by the authenticated user
// @Tags Apps
// @Accept json
// @Produce json
// @Success 200 {object} AppsResponse "List of applications"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/apps/ [get]
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
// @Summary Update an application
// @Description Updates an existing application by its ID
// @Tags Apps
// @Accept json
// @Produce json
// @Param id path int true "Application ID"
// @Param request body UpdateAppRequest true "Update Data"
// @Success 200 {object} UpdateAppResponse "Updated application"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/apps/{id} [put]
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
// @Summary Delete an application
// @Description Deletes an application by its ID
// @Tags Apps
// @Accept json
// @Produce json
// @Param id path int true "Application ID"
// @Success 200 {object} DeleteAppResponse "Success message"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 404 {object} ErrorResponse "Not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/apps/{id} [delete]
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
