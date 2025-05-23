package api

import (
	"log"
	"net/http"
	// "strconv" // strconv is not used in the provided code for this file

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/analytics"
)

// AnalyticsHandler handles analytics-related API requests.
type AnalyticsHandler struct {
	analyzerService *analytics.PerformanceAnalyzerService
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(service *analytics.PerformanceAnalyzerService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyzerService: service,
	}
}

// RegisterRoutes registers the analytics API routes.
func (h *AnalyticsHandler) RegisterRoutes(router *gin.RouterGroup) {
	analyticsGroup := router.Group("/analytics")
	// Potentially add auth middleware here if these endpoints need protection
	analyticsGroup.GET("/ranked-servicers", h.getRankedServicers)
}

// GetRankedServicersRequest defines query parameters for the ranked servicers endpoint.
// Using struct tags for binding query parameters.
type GetRankedServicersRequest struct {
	Limit        int     `form:"limit"`
	ChainID      *int    `form:"chain_id"`
	ServicerType *string `form:"servicer_type"` // 'static' or 'discovered'
	// Geozone      *string `form:"geozone"` // Placeholder
}

// ErrorResponse is a generic error response struct.
// Define it here if not globally available.
// type ErrorResponse struct {
//    Error string `json:"error"`
// }

// GetRankedServicers godoc
// @Summary Get ranked list of servicers
// @Description Retrieves a list of active servicers, ranked by performance (lower response_time_ms is better), with optional filters.
// @Tags Analytics
// @Accept json
// @Produce json
// @Param limit query int false "Limit number of results (default 20, max 100)"
// @Param chain_id query int false "Filter by Chain ID (e.g., 1 for Viper, 2 for Ethereum)"
// @Param servicer_type query string false "Filter by Servicer Type ('static' or 'discovered')"
// @Success 200 {array} analytics.RankedServicer "List of ranked servicers"
// @Failure 400 {object} gin.H "Bad request (e.g., invalid query parameters)"
// @Failure 500 {object} gin.H "Internal server error"
// @Router /api/analytics/ranked-servicers [get]
func (h *AnalyticsHandler) getRankedServicers(c *gin.Context) {
	var req GetRankedServicersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 20 // Default limit
	}
	if req.Limit > 100 { // Max limit
		req.Limit = 100
	}

	opts := analytics.GetRankedServicersOptions{
		Limit:        req.Limit,
		ChainID:      req.ChainID,
		ServicerType: req.ServicerType,
		// Geozone:   req.Geozone, // Add when/if geozone implemented
	}

	servicers, err := h.analyzerService.GetRankedServicers(c.Request.Context(), opts)
	if err != nil {
		log.Printf("Error getting ranked servicers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve ranked servicers"})
		return
	}

	if servicers == nil {
		servicers = []analytics.RankedServicer{} // Return empty array instead of null
	}

	c.JSON(http.StatusOK, servicers)
}
