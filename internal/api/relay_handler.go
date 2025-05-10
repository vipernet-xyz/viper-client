package api

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/relay"
)

// RelayHandler handles relay-related API requests
type RelayHandler struct {
	relayService *relay.Service
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(relayService *relay.Service) *RelayHandler {
	return &RelayHandler{
		relayService: relayService,
	}
}

// RegisterRoutes registers the relay-related routes
func (h *RelayHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Public route - requires API key in query params
	router.POST("/relay", h.handleRelay)
}

// handleRelay handles the relay request
// @Summary Relay RPC request
// @Description Forwards an RPC request to the appropriate blockchain endpoint
// @Tags Relay
// @Accept json
// @Produce json
// @Param api_key query string true "API Key"
// @Param chain_id query int true "Chain ID"
// @Param request body object true "RPC Request"
// @Success 200 {object} relay.RelayResponse "RPC Response"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/relay [post]
func (h *RelayHandler) handleRelay(c *gin.Context) {
	// Get API key and chain ID from query parameters
	apiKey := c.Query("api_key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "API key is required",
		})
		return
	}

	chainIDStr := c.Query("chain_id")
	if chainIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Chain ID is required",
		})
		return
	}

	// Parse chain ID
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	// Read request body
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// Create relay request
	req := relay.RelayRequest{
		APIKey:  apiKey,
		ChainID: chainID,
		Request: requestBody,
	}

	// Forward the request
	response, err := h.relayService.Relay(c.Request.Context(), req)
	if err != nil {
		switch err.Error() {
		case "invalid API key":
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
		case "chain not allowed for this app":
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Chain not allowed for this app",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to relay request: " + err.Error(),
			})
		}
		return
	}

	// Return the response
	c.JSON(http.StatusOK, response)
}
