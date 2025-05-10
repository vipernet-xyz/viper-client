package relay

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/illegalcall/viper-client/internal/apps"
	"github.com/illegalcall/viper-client/internal/models"
	"github.com/illegalcall/viper-client/internal/rpc"
)

// Service provides functionality for relaying RPC requests
// @Description Handles relaying of RPC requests to appropriate blockchain nodes
type Service struct {
	db            *sql.DB
	appsService   *apps.Service
	rpcDispatcher *rpc.Dispatcher
	httpClient    *http.Client
}

// NewService creates a new relay service
func NewService(db *sql.DB, appsService *apps.Service, rpcDispatcher *rpc.Dispatcher) *Service {
	return &Service{
		db:            db,
		appsService:   appsService,
		rpcDispatcher: rpcDispatcher,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RelayRequest represents the request to relay an RPC call
// @Description Request structure for relaying an RPC call
type RelayRequest struct {
	APIKey  string          `json:"api_key" binding:"required" example:"your-api-key-here"`
	ChainID int             `json:"chain_id" binding:"required" example:"1"`
	Request json.RawMessage `json:"request" binding:"required" example:"{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}"`
}

// RelayResponse represents the response from the relay service
// @Description Response structure from the relay service
type RelayResponse struct {
	Response json.RawMessage `json:"response" example:"{\"jsonrpc\":\"2.0\",\"result\":\"0x1234\",\"id\":1}"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid API key"`
}

// Relay forwards an RPC request to the appropriate endpoint
// @Summary Relay RPC request
// @Description Forwards an RPC request to the appropriate blockchain node
// @Tags Relay
// @Accept json
// @Produce json
// @Param request body RelayRequest true "RPC request to relay"
// @Success 200 {object} RelayResponse "RPC response from the blockchain node"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Invalid API key"
// @Failure 403 {object} ErrorResponse "Chain not allowed for this app"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/relay [post]
func (s *Service) Relay(ctx context.Context, req RelayRequest) (*RelayResponse, error) {
	// 1. Verify API key
	app, err := s.verifyAPIKey(req.APIKey)
	if err != nil {
		return nil, err
	}

	// 2. Check if chain is allowed for this app
	if !s.isChainAllowed(app, req.ChainID) {
		return nil, errors.New("chain not allowed for this app")
	}

	// 3. Forward the request to the RPC dispatcher
	response, err := s.rpcDispatcher.Forward(ctx, req.ChainID, req.Request)
	if err != nil {
		return nil, err
	}

	// 4. Log the request in stats
	if err := s.logRequest(req.APIKey, req.ChainID); err != nil {
		// Log error but don't fail the request
		// TODO: Add proper error logging
	}

	return &RelayResponse{
		Response: response,
	}, nil
}

// verifyAPIKey verifies the API key and returns the associated app
func (s *Service) verifyAPIKey(apiKey string) (*models.App, error) {
	// Get app by API key
	app, err := s.appsService.GetAppByAPIKey(apiKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid API key")
		}
		return nil, err
	}

	return app, nil
}

// isChainAllowed checks if the chain is allowed for the app
func (s *Service) isChainAllowed(app *models.App, chainID int) bool {
	for _, allowedChain := range app.AllowedChains {
		if allowedChain == chainID {
			return true
		}
	}
	return false
}

// logRequest logs the request in the stats table
func (s *Service) logRequest(apiKey string, chainID int) error {
	query := `
		INSERT INTO logs (endpoint, api_key, chain_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`

	_, err := s.db.Exec(query, "relay", apiKey, chainID)
	return err
}
