package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/illegalcall/viper-client/internal/models"
)

var (
	// ErrNoEndpoints is returned when no active endpoints are available for a chain
	ErrNoEndpoints = errors.New("no active endpoints available for the requested chain")
)

// StatsLogger interface for logging RPC request statistics
type StatsLogger interface {
	LogRequest(chainID int, endpointID int, method string, success bool) error
}

// RPCRequest represents a JSON-RPC request
// @Description JSON-RPC request structure
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// RPCResponse represents a JSON-RPC response
// @Description JSON-RPC response structure
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// RPCError represents a JSON-RPC error
// @Description JSON-RPC error structure
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Dispatcher handles forwarding RPC requests to blockchain nodes
type Dispatcher struct {
	endpointManager     EndpointManager
	httpClient          *http.Client
	viperNetworkHandler *ViperNetworkHandler
}

// EndpointManager defines the interface for retrieving and managing RPC endpoints
type EndpointManager interface {
	GetActiveEndpoints(chainID int) ([]models.RpcEndpoint, error)
	UpdateEndpointHealth(id int, status string) error
}

// NewDispatcher creates a new RPC dispatcher with the given endpoint manager
func NewDispatcher(manager EndpointManager) *Dispatcher {
	viperHandler := NewViperNetworkHandler(manager)

	return &Dispatcher{
		endpointManager: manager,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		viperNetworkHandler: viperHandler,
	}
}

// Forward forwards an RPC request to an available endpoint for the given chain
func (d *Dispatcher) Forward(ctx context.Context, chainID int, requestBody []byte) ([]byte, error) {
	// Check if this is a request for the Viper Network
	if chainID == ViperNetworkChainID {
		return d.ForwardToViperNetwork(ctx, requestBody)
	}

	// Parse the incoming request to validate and potentially use for caching
	var rpcRequest RPCRequest
	if err := json.Unmarshal(requestBody, &rpcRequest); err != nil {
		return nil, errors.New("invalid JSON-RPC request format")
	}

	// Get available endpoints for the chain
	endpoints, err := d.endpointManager.GetActiveEndpoints(chainID)
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Sort endpoints by priority (assuming they are already sorted from the database)
	// For now, just use the first endpoint
	selectedEndpoint := endpoints[0]

	// Forward the request to the selected endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", selectedEndpoint.EndpointURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		// Update endpoint health status
		d.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Update endpoint health status to healthy
	d.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "healthy")

	return responseBody, nil
}

// ForwardToViperNetwork handles forwarding requests to the Viper Network,
// translating between JSON-RPC and Viper Network formats
func (d *Dispatcher) ForwardToViperNetwork(ctx context.Context, requestBody []byte) ([]byte, error) {
	// Convert from JSON-RPC format to Viper Network format
	requestType, viperRequest, err := ConvertJSONRPCToViperFormat(requestBody)
	if err != nil {
		return nil, err
	}

	// Send the request to Viper Network
	viperResponse, err := d.viperNetworkHandler.HandleViperRequest(ctx, requestType, viperRequest)
	if err != nil {
		return nil, err
	}

	// Convert the response back to JSON-RPC format
	jsonRPCResponse, err := ConvertViperResponseToJSONRPC(viperResponse, requestBody)
	if err != nil {
		return nil, err
	}

	return jsonRPCResponse, nil
}

// RPCDispatcher handles RPC request dispatching
// @Description Handles dispatching of RPC requests to appropriate endpoints
type RPCDispatcher struct {
	endpointManager EndpointManager
	statsLogger     StatsLogger
	httpClient      *http.Client
}

// NewRPCDispatcher creates a new RPC dispatcher
func NewRPCDispatcher(em EndpointManager, sl StatsLogger) *RPCDispatcher {
	return &RPCDispatcher{
		endpointManager: em,
		statsLogger:     sl,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DispatchRequest forwards an RPC request to the appropriate endpoint
// @Summary Dispatch RPC request
// @Description Forwards an RPC request to the appropriate endpoint based on chain ID and geolocation
// @Tags RPC
// @Accept json
// @Produce json
// @Param chainID path int true "Chain ID"
// @Param request body RPCRequest true "RPC request"
// @Success 200 {object} RPCResponse "RPC response"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "No active endpoints found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /rpc/{chainID} [post]
func (d *RPCDispatcher) DispatchRequest(chainID int, request *RPCRequest) (*RPCResponse, error) {
	// Get active endpoints for the chain
	endpoints, err := d.endpointManager.GetActiveEndpoints(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no active endpoints found for chain ID %d", chainID)
	}

	// Try each endpoint in order of priority
	var lastErr error
	for _, endpoint := range endpoints {
		response, err := d.tryEndpoint(endpoint, request)
		if err == nil {
			// Log successful request
			if err := d.statsLogger.LogRequest(chainID, endpoint.ID, request.Method, true); err != nil {
				// Log error but don't fail the request
				fmt.Printf("Failed to log request stats: %v\n", err)
			}
			return response, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all endpoints failed: %w", lastErr)
}

// tryEndpoint attempts to send the request to a specific endpoint
func (d *RPCDispatcher) tryEndpoint(endpoint models.RpcEndpoint, request *RPCRequest) (*RPCResponse, error) {
	// Marshal request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint.EndpointURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var rpcResp RPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	return &rpcResp, nil
}
