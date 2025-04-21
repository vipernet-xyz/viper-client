package api

import (
	"net/http"

	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/dhruvsharma/viper-client/internal/rpc"
	"github.com/dhruvsharma/viper-client/internal/utils"
	"github.com/gin-gonic/gin"
)

// RelayHandler handles the relay API endpoints
type RelayHandler struct {
	relayService *rpc.RelayService
	signer       *utils.Signer
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(endpointManager rpc.EndpointManager) *RelayHandler {
	// Create a random signer for relay operations
	signer, err := utils.NewRandomSigner()
	if err != nil {
		panic(err)
	}

	// Initialize the relay service
	relayService := rpc.NewRelayService(endpointManager, signer)

	return &RelayHandler{
		relayService: relayService,
		signer:       signer,
	}
}

// RegisterRoutes registers the relay handler routes
func (h *RelayHandler) RegisterRoutes(router *gin.Engine) {
	relayGroup := router.Group("/relay")

	// Authentication middleware
	relayGroup.Use(func(c *gin.Context) {
		// Get app identifier and API key from headers
		appIdentifier := c.GetHeader("X-App-ID")
		apiKey := c.GetHeader("X-API-Key")

		if appIdentifier == "" || apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing application credentials",
			})
			c.Abort()
			return
		}

		// For now, just pass through - the actual auth would need to be implemented
		// with app service validation

		c.Next()
	})

	// Register the relay endpoints
	relayGroup.POST("/dispatch", h.handleDispatch)
	relayGroup.POST("/direct", h.handleDirectRelay)
	relayGroup.POST("/execute", h.handleExecuteRelay)
}

// RelayRequest represents a request to relay data to a blockchain
type RelayRequest struct {
	PubKey         string            `json:"pub_key" binding:"required"`
	Blockchain     string            `json:"blockchain" binding:"required"`
	GeoZone        string            `json:"geo_zone" binding:"required"`
	NumServicers   int64             `json:"num_servicers" binding:"required"`
	Data           string            `json:"data" binding:"required"`
	Method         string            `json:"method" binding:"required"`
	Path           string            `json:"path"`
	Headers        map[string]string `json:"headers"`
	ServicerURL    string            `json:"servicer_url"`
	ServicerPubKey string            `json:"servicer_pub_key"`
}

// handleDispatch handles a dispatch request
// @Summary Dispatch a session with the Viper Network
// @Description Get a new session for interacting with the Viper Network
// @Tags Relay
// @Accept json
// @Produce json
// @Param X-App-ID header string true "Application Identifier"
// @Param X-API-Key header string true "API Key"
// @Param request body RelayRequest true "Dispatch Request"
// @Success 200 {object} models.DispatchResponse "Session information"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /relay/dispatch [post]
func (h *RelayHandler) handleDispatch(c *gin.Context) {
	var req RelayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Dispatch a session
	dispatchResponse, err := h.relayService.Dispatch(
		c.Request.Context(),
		req.PubKey,
		req.Blockchain,
		req.GeoZone,
		req.NumServicers,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to dispatch session: " + err.Error(),
		})
		return
	}

	// Return the dispatch response
	c.JSON(http.StatusOK, dispatchResponse)
}

// handleDirectRelay handles a direct relay request
// @Summary Send a direct relay to a specific servicer
// @Description Send a relay request directly to a specific servicer
// @Tags Relay
// @Accept json
// @Produce json
// @Param X-App-ID header string true "Application Identifier"
// @Param X-API-Key header string true "API Key"
// @Param request body RelayRequest true "Direct Relay Request"
// @Success 200 {object} models.RelayResponse "Relay response"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /relay/direct [post]
func (h *RelayHandler) handleDirectRelay(c *gin.Context) {
	var req RelayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.ServicerURL == "" || req.ServicerPubKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Servicer URL and public key are required for direct relay",
		})
		return
	}

	// Generate AAT
	aat, err := h.relayService.GenerateAAT(req.PubKey, h.signer.GetPublicKey())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate AAT: " + err.Error(),
		})
		return
	}

	// Create servicer object
	servicer := &models.Servicer{
		PublicKey: req.ServicerPubKey,
		NodeURL:   req.ServicerURL,
	}

	// Build relay request
	relay, err := h.relayService.BuildRelayRequest(
		c.Request.Context(),
		servicer,
		req.Blockchain,
		req.GeoZone,
		req.NumServicers,
		req.Data,
		req.Method,
		req.Path,
		req.Headers,
		aat,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to build relay request: " + err.Error(),
		})
		return
	}

	// Send the relay
	response, err := h.relayService.SendRelay(c.Request.Context(), relay, servicer.NodeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send relay: " + err.Error(),
		})
		return
	}

	// Return the relay response
	c.JSON(http.StatusOK, response)
}

// handleExecuteRelay handles a complete relay execution
// @Summary Execute a complete relay process
// @Description Execute the complete relay process (dispatch + relay)
// @Tags Relay
// @Accept json
// @Produce json
// @Param X-App-ID header string true "Application Identifier"
// @Param X-API-Key header string true "API Key"
// @Param request body RelayRequest true "Relay Execution Request"
// @Success 200 {object} models.RelayResponse "Relay response"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /relay/execute [post]
func (h *RelayHandler) handleExecuteRelay(c *gin.Context) {
	var req RelayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Execute the relay
	response, err := h.relayService.ExecuteRelay(
		c.Request.Context(),
		req.PubKey,
		req.Blockchain,
		req.GeoZone,
		req.NumServicers,
		req.Data,
		req.Method,
		req.Path,
		req.Headers,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute relay: " + err.Error(),
		})
		return
	}

	// Return the relay response
	c.JSON(http.StatusOK, response)
}
