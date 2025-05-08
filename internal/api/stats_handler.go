package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/stats"
)

// StatsHandler handles statistics-related API requests
type StatsHandler struct {
	statsService *stats.Service
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(statsService *stats.Service) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// RegisterRoutes registers the stats-related routes
func (h *StatsHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Routes require authentication
	router.GET("/stats/chain/:chainId", h.getChainStats)
	router.GET("/stats/api-key/:apiKey/:chainId", h.getAPIKeyStats)
	router.GET("/stats/endpoint/:endpoint/:chainId", h.getEndpointStats)
}

// getChainStats retrieves statistics for a specific chain
// @Summary Get chain statistics
// @Description Retrieves usage statistics for a specific blockchain chain
// @Tags Stats
// @Accept json
// @Produce json
// @Param chainId path int true "Chain ID"
// @Param interval query string false "Time interval (1hour, 4hour, 6hour, 12hour, 24hour)" default(1hour)
// @Param startDate query string false "Start date (RFC3339 format)"
// @Param endDate query string false "End date (RFC3339 format)"
// @Success 200 {object} stats.StatsResponse "Chain statistics"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/stats/chain/{chainId} [get]
func (h *StatsHandler) getChainStats(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get chain ID from URL
	chainIDStr := c.Param("chainId")
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	// Get interval from query parameters
	interval := c.DefaultQuery("interval", "1hour")
	if !isValidInterval(interval) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid interval. Must be one of: 1hour, 4hour, 6hour, 12hour, 24hour",
		})
		return
	}

	// Get optional date filters
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Create filter
	filter := stats.StatsFilter{
		ChainID:  chainID,
		Interval: interval,
	}

	// Parse start date if provided
	if startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start date format. Use RFC3339 format (e.g., 2025-01-02T15:04:05Z)",
			})
			return
		}
		filter.StartDate = startDate
	}

	// Parse end date if provided
	if endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end date format. Use RFC3339 format (e.g., 2025-01-02T15:04:05Z)",
			})
			return
		}
		filter.EndDate = endDate
	}

	// Get statistics
	statistics, err := h.statsService.GetStats(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chain statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": statistics,
	})
}

// getAPIKeyStats retrieves statistics for a specific API key
// @Summary Get API key statistics
// @Description Retrieves usage statistics for a specific API key on a specific chain
// @Tags Stats
// @Accept json
// @Produce json
// @Param apiKey path string true "API Key"
// @Param chainId path int true "Chain ID"
// @Param interval query string false "Time interval (1hour, 4hour, 6hour, 12hour, 24hour)" default(1hour)
// @Param startDate query string false "Start date (RFC3339 format)"
// @Param endDate query string false "End date (RFC3339 format)"
// @Success 200 {object} stats.StatsResponse "API key statistics"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/stats/api-key/{apiKey}/{chainId} [get]
func (h *StatsHandler) getAPIKeyStats(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get API key and chain ID from URL
	apiKey := c.Param("apiKey")
	chainIDStr := c.Param("chainId")
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	// Get interval from query parameters
	interval := c.DefaultQuery("interval", "1hour")
	if !isValidInterval(interval) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid interval. Must be one of: 1hour, 4hour, 6hour, 12hour, 24hour",
		})
		return
	}

	// Get optional date filters
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Create filter
	filter := stats.StatsFilter{
		ChainID:  chainID,
		APIKey:   apiKey,
		Interval: interval,
	}

	// Parse start date if provided
	if startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start date format. Use RFC3339 format (e.g., 2025-01-02T15:04:05Z)",
			})
			return
		}
		filter.StartDate = startDate
	}

	// Parse end date if provided
	if endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end date format. Use RFC3339 format (e.g., 2025-01-02T15:04:05Z)",
			})
			return
		}
		filter.EndDate = endDate
	}

	// Get statistics
	statistics, err := h.statsService.GetStats(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve API key statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": statistics,
	})
}

// getEndpointStats retrieves statistics for a specific endpoint
// @Summary Get endpoint statistics
// @Description Retrieves usage statistics for a specific endpoint on a specific chain
// @Tags Stats
// @Accept json
// @Produce json
// @Param endpoint path string true "Endpoint"
// @Param chainId path int true "Chain ID"
// @Param interval query string false "Time interval (1hour, 4hour, 6hour, 12hour, 24hour)" default(1hour)
// @Param startDate query string false "Start date (RFC3339 format)"
// @Param endDate query string false "End date (RFC3339 format)"
// @Success 200 {object} stats.StatsResponse "Endpoint statistics"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/stats/endpoint/{endpoint}/{chainId} [get]
func (h *StatsHandler) getEndpointStats(c *gin.Context) {
	// Get user ID from the authenticated context
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get endpoint and chain ID from URL
	endpoint := c.Param("endpoint")
	chainIDStr := c.Param("chainId")
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	// Get interval from query parameters
	interval := c.DefaultQuery("interval", "1hour")
	if !isValidInterval(interval) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid interval. Must be one of: 1hour, 4hour, 6hour, 12hour, 24hour",
		})
		return
	}

	// Get optional date filters
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Create filter
	filter := stats.StatsFilter{
		ChainID:  chainID,
		Endpoint: endpoint,
		Interval: interval,
	}

	// Parse start date if provided
	if startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start date format. Use RFC3339 format (e.g., 2025-01-02T15:04:05Z)",
			})
			return
		}
		filter.StartDate = startDate
	}

	// Parse end date if provided
	if endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end date format. Use RFC3339 format (e.g., 2025-01-02T15:04:05Z)",
			})
			return
		}
		filter.EndDate = endDate
	}

	// Get statistics
	statistics, err := h.statsService.GetStats(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve endpoint statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": statistics,
	})
}

// Helper function to validate interval values
func isValidInterval(interval string) bool {
	validIntervals := map[string]bool{
		"1hour":  true,
		"4hour":  true,
		"6hour":  true,
		"12hour": true,
		"24hour": true,
	}
	return validIntervals[interval]
}
