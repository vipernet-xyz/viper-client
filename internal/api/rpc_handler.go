package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/dhruvsharma/viper-client/internal/apps"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/rpc"
	"github.com/gin-gonic/gin"
)

// RPCHandler handles blockchain RPC requests
type RPCHandler struct {
	dispatcher   *rpc.Dispatcher
	appsService  *apps.Service
	chainService ChainService
}

// ChainService defines the interface for chain operations
type ChainService interface {
	GetChainByID(id int) (*db.ChainInfo, error)
	GetChainByIdentifier(identifier string) (*db.ChainInfo, error)
}

// NewRPCHandler creates a new RPC handler
func NewRPCHandler(dispatcher *rpc.Dispatcher, appsService *apps.Service, chainService ChainService) *RPCHandler {
	return &RPCHandler{
		dispatcher:   dispatcher,
		appsService:  appsService,
		chainService: chainService,
	}
}

// RegisterRoutes registers the RPC handler routes
func (h *RPCHandler) RegisterRoutes(router *gin.Engine) {
	// Public RPC endpoints - require API key in headers or request params
	router.POST("/rpc/:chain_id", h.handleRPCRequestByChainID)
	router.POST("/api/rpc/:identifier/:network", h.handleRPCRequest)
}

// handleRPCRequestByChainID handles a JSON-RPC request for a specific blockchain using chain ID
func (h *RPCHandler) handleRPCRequestByChainID(c *gin.Context) {
	// Get chain ID from URL
	chainIDStr := c.Param("chain_id")
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	// Get app identifier and API key from headers
	appIdentifier := c.GetHeader("X-App-ID")
	apiKey := c.GetHeader("X-API-Key")

	if appIdentifier == "" || apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Missing application credentials",
		})
		return
	}

	// Validate API key
	valid, err := h.appsService.ValidateAPIKey(appIdentifier, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate API key",
		})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid API key",
		})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// Forward the request to the RPC dispatcher
	response, err := h.dispatcher.Forward(c.Request.Context(), chainID, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process RPC request: " + err.Error(),
		})
		return
	}

	// Set content type and return the raw response
	c.Header("Content-Type", "application/json")
	c.Writer.Write(response)
}

// handleRPCRequest handles a JSON-RPC request using app identifier and network name
func (h *RPCHandler) handleRPCRequest(c *gin.Context) {
	// Get app identifier and network from URL
	appIdentifier := c.Param("identifier")
	networkIdentifier := c.Param("network")

	// Get API key from headers or query params
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		apiKey = c.Query("apiKey")
	}

	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Missing API key",
		})
		return
	}

	// Validate API key
	valid, err := h.appsService.ValidateAPIKey(appIdentifier, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate API key: " + err.Error(),
		})
		return
	}

	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid API key",
		})
		return
	}

	// Resolve network identifier to chain ID
	chainInfo, err := h.chainService.GetChainByIdentifier(networkIdentifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid network identifier: " + err.Error(),
		})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// Forward the request to the RPC dispatcher
	response, err := h.dispatcher.Forward(c.Request.Context(), chainInfo.ID, body)
	if err != nil {
		if errors.Is(err, rpc.ErrNoEndpoints) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "No active endpoints available for the requested chain",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process RPC request: " + err.Error(),
		})
		return
	}

	// Set content type and return the raw response
	c.Header("Content-Type", "application/json")
	c.Writer.Write(response)
}
