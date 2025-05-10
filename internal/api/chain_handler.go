package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/chains"
)

// ChainResponse represents the response for chain operations
// @Description Response data for chain operations
type ChainResponse struct {
	// The chain object
	Chain interface{} `json:"chain"`
}

// ChainsResponse represents the response for multiple chains
// @Description Response data for multiple chains
type ChainsResponse struct {
	// List of chains
	Chains interface{} `json:"chains"`
}

// ChainsHandler handles chain-related API requests
type ChainsHandler struct {
	chainsService *chains.Service
}

// NewChainsHandler creates a new chains handler
func NewChainsHandler(chainsService *chains.Service) *ChainsHandler {
	return &ChainsHandler{
		chainsService: chainsService,
	}
}

// RegisterRoutes registers the chain-related routes
func (h *ChainsHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Public routes - no authentication required
	router.GET("/chains", h.getAllChains)
	router.GET("/chains/:id", h.getChain)
	router.GET("/chains/by-chain-id/:chain_id", h.getChainByChainID)
	router.GET("/chains/network-type/:network_type", h.getChainsByNetworkType)
	router.GET("/chains/evm", h.getEVMChains)
}

// getAllChains retrieves all supported chains
// @Summary Get all chains
// @Description Retrieves all supported blockchain networks
// @Tags Chains
// @Accept json
// @Produce json
// @Success 200 {object} ChainsResponse "List of chains"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/chains [get]
func (h *ChainsHandler) getAllChains(c *gin.Context) {
	chains, err := h.chainsService.GetAllChains()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chains: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chains": chains,
	})
}

// getChain retrieves a chain by ID
// @Summary Get chain by ID
// @Description Retrieves a specific chain by its database ID
// @Tags Chains
// @Accept json
// @Produce json
// @Param id path int true "Chain ID"
// @Success 200 {object} ChainResponse "Chain details"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/chains/{id} [get]
func (h *ChainsHandler) getChain(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	chain, err := h.chainsService.GetChain(id)
	if err != nil {
		if err.Error() == "chain not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Chain not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chain: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chain": chain,
	})
}

// getChainByChainID retrieves a chain by its chain ID
// @Summary Get chain by chain ID
// @Description Retrieves a specific chain by its chain ID (e.g., 1 for Viper Network)
// @Tags Chains
// @Accept json
// @Produce json
// @Param chain_id path int true "Chain ID (e.g., 1 for Viper Network)"
// @Success 200 {object} ChainResponse "Chain details"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 404 {object} ErrorResponse "Not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/chains/by-chain-id/{chain_id} [get]
func (h *ChainsHandler) getChainByChainID(c *gin.Context) {
	chainIDStr := c.Param("chain_id")
	chainID, err := strconv.Atoi(chainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chain ID",
		})
		return
	}

	chain, err := h.chainsService.GetChainByChainID(chainID)
	if err != nil {
		if err.Error() == "chain not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Chain not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chain: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chain": chain,
	})
}

// getChainsByNetworkType retrieves chains by network type
// @Summary Get chains by network type
// @Description Retrieves chains by their network type (mainnet/testnet)
// @Tags Chains
// @Accept json
// @Produce json
// @Param network_type path string true "Network type (mainnet/testnet)"
// @Success 200 {object} ChainsResponse "List of chains"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/chains/network-type/{network_type} [get]
func (h *ChainsHandler) getChainsByNetworkType(c *gin.Context) {
	networkType := c.Param("network_type")
	if networkType != "mainnet" && networkType != "testnet" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid network type. Must be 'mainnet' or 'testnet'",
		})
		return
	}

	chains, err := h.chainsService.GetChainsByNetworkType(networkType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chains: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chains": chains,
	})
}

// getEVMChains retrieves all EVM-compatible chains
// @Summary Get EVM chains
// @Description Retrieves all EVM-compatible chains
// @Tags Chains
// @Accept json
// @Produce json
// @Success 200 {object} ChainsResponse "List of EVM chains"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/chains/evm [get]
func (h *ChainsHandler) getEVMChains(c *gin.Context) {
	chains, err := h.chainsService.GetEVMChains()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve EVM chains: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chains": chains,
	})
}
