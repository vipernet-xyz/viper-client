package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/apps"
	"github.com/illegalcall/viper-client/internal/rpc"
)

// ViperNetworkHandler handles direct requests to the Viper Network
type ViperNetworkHandler struct {
	viperHandler *rpc.ViperNetworkHandler
	appsService  *apps.Service
}

// NewViperNetworkHandler creates a new handler for Viper Network
func NewViperNetworkHandler(viperHandler *rpc.ViperNetworkHandler, appsService *apps.Service) *ViperNetworkHandler {
	return &ViperNetworkHandler{
		viperHandler: viperHandler,
		appsService:  appsService,
	}
}

// RegisterRoutes registers the Viper Network handler routes
func (h *ViperNetworkHandler) RegisterRoutes(router *gin.Engine) {
	// Direct Viper Network endpoints
	viperGroup := router.Group("/viper")

	// Requires API key in headers
	viperGroup.POST("/height", h.authenticate, h.handleHeight)
	viperGroup.POST("/relay", h.authenticate, h.handleRelay)
	viperGroup.POST("/servicers", h.authenticate, h.handleServicers)
	viperGroup.POST("/block", h.authenticate, h.handleBlock)
	viperGroup.POST("/tx", h.authenticate, h.handleTx)
	viperGroup.POST("/account", h.authenticate, h.handleAccount)
	viperGroup.POST("/supportedchains", h.authenticate, h.handleSupportedChains)
	viperGroup.POST("/dispatch", h.authenticate, h.handleDispatch)
	viperGroup.POST("/challenge", h.authenticate, h.handleChallenge)

	// WebSocket endpoint
	router.GET("/viper/websocket", h.authenticate, h.handleWebSocket)
}

// Authentication middleware for Viper Network requests
func (h *ViperNetworkHandler) authenticate(c *gin.Context) {
	// Get app identifier and API key from headers
	apiKey := c.GetHeader("X-API-Key")

	if apiKey == "" {
		// Try from query parameters
		apiKey = c.Query("apiKey")

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing application credentials",
			})
			c.Abort()
			return
		}
	}

	// Validate API key
	valid, err := h.appsService.ValidateAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate API key: " + err.Error(),
		})
		c.Abort()
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid API key",
		})
		c.Abort()
		return
	}

	c.Next()
}

// Handler functions for different Viper Network endpoints

func (h *ViperNetworkHandler) handleHeight(c *gin.Context) {
	h.proxyViperRequest(c, "height")
}

func (h *ViperNetworkHandler) handleRelay(c *gin.Context) {
	h.proxyViperRequest(c, "relay")
}

func (h *ViperNetworkHandler) handleServicers(c *gin.Context) {
	h.proxyViperRequest(c, "servicers")
}

func (h *ViperNetworkHandler) handleBlock(c *gin.Context) {
	h.proxyViperRequest(c, "block")
}

func (h *ViperNetworkHandler) handleTx(c *gin.Context) {
	h.proxyViperRequest(c, "tx")
}

func (h *ViperNetworkHandler) handleAccount(c *gin.Context) {
	h.proxyViperRequest(c, "account")
}

func (h *ViperNetworkHandler) handleSupportedChains(c *gin.Context) {
	h.proxyViperRequest(c, "supportedchains")
}

func (h *ViperNetworkHandler) handleDispatch(c *gin.Context) {
	h.proxyViperRequest(c, "dispatch")
}

func (h *ViperNetworkHandler) handleChallenge(c *gin.Context) {
	h.proxyViperRequest(c, "challenge")
}

func (h *ViperNetworkHandler) handleWebSocket(c *gin.Context) {
	h.proxyViperRequest(c, "websocket")
}

// proxyViperRequest handles forwarding the request to the Viper Network
func (h *ViperNetworkHandler) proxyViperRequest(c *gin.Context, requestType string) {
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body: " + err.Error(),
		})
		return
	}

	// If body is empty for endpoints that require data, provide an empty JSON object
	if len(body) == 0 {
		body = []byte("{}")
	}

	// Forward the request to the Viper Network
	response, err := h.viperHandler.HandleViperRequest(c.Request.Context(), requestType, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process Viper Network request: " + err.Error(),
		})
		return
	}

	// Set content type and return the raw response
	c.Header("Content-Type", "application/json")
	c.Writer.Write(response)
}
