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
)

const (
	// Chain ID for Viper Network in the database
	ViperNetworkChainID = 0001

	// Common Viper Network RPC endpoints
	ViperHeightEndpoint    = "/v1/query/height"
	ViperRelayEndpoint     = "/v1/client/relay"
	ViperSupportedChains   = "/v1/query/supportedchains"
	ViperServicersEndpoint = "/v1/query/servicers"
	ViperBlockEndpoint     = "/v1/query/block"
	ViperTxEndpoint        = "/v1/query/tx"
	ViperAccountEndpoint   = "/v1/query/account"
	ViperDispatchEndpoint  = "/v1/client/dispatch"
	ViperChallengeEndpoint = "/v1/client/challenge"
	ViperWebSocketEndpoint = "/v1/client/websocket"
)

// ViperNetworkRequest is the standard request format for viper-network
type ViperNetworkRequest struct {
	Height       int64                  `json:"height,omitempty"`
	Blockchain   string                 `json:"blockchain,omitempty"`
	Data         string                 `json:"data,omitempty"`
	Method       string                 `json:"method,omitempty"`
	Path         string                 `json:"path,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Proof        map[string]interface{} `json:"proof,omitempty"`
	Opts         map[string]interface{} `json:"opts,omitempty"`
	Hash         string                 `json:"hash,omitempty"`
	Address      string                 `json:"address,omitempty"`
	PubKey       string                 `json:"pubkey,omitempty"`
	ChainID      string                 `json:"chain_id,omitempty"`
	Subscription bool                   `json:"subscription,omitempty"`
	AI           bool                   `json:"ai,omitempty"`
}

// ViperNetworkHandler provides functionality to interact with the Viper Network
type ViperNetworkHandler struct {
	endpointManager EndpointManager
	httpClient      *http.Client
}

// NewViperNetworkHandler creates a new handler for Viper Network interactions
func NewViperNetworkHandler(manager EndpointManager) *ViperNetworkHandler {
	return &ViperNetworkHandler{
		endpointManager: manager,
		httpClient: &http.Client{
			Timeout: 15 * time.Second, // Longer timeout for viper-network requests
		},
	}
}

// HandleViperRequest handles a request specifically for the Viper Network
func (v *ViperNetworkHandler) HandleViperRequest(ctx context.Context, requestType string, requestData []byte) ([]byte, error) {
	// Parse the incoming request
	var request ViperNetworkRequest
	if err := json.Unmarshal(requestData, &request); err != nil {
		return nil, fmt.Errorf("invalid viper network request format: %w", err)
	}

	// Get active endpoints for viper network
	endpoints, err := v.endpointManager.GetActiveEndpoints(ViperNetworkChainID)
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Select the highest priority endpoint
	selectedEndpoint := endpoints[0]

	// Determine the target endpoint path based on the request type
	var targetPath string
	switch requestType {
	case "height":
		targetPath = ViperHeightEndpoint
	case "relay":
		targetPath = ViperRelayEndpoint
	case "supportedchains":
		targetPath = ViperSupportedChains
	case "servicers":
		targetPath = ViperServicersEndpoint
	case "block":
		targetPath = ViperBlockEndpoint
	case "tx":
		targetPath = ViperTxEndpoint
	case "account":
		targetPath = ViperAccountEndpoint
	case "dispatch":
		targetPath = ViperDispatchEndpoint
	case "challenge":
		targetPath = ViperChallengeEndpoint
	case "websocket":
		targetPath = ViperWebSocketEndpoint
	default:
		return nil, fmt.Errorf("unsupported viper network request type: %s", requestType)
	}

	// Construct the full URL
	fullURL := selectedEndpoint.EndpointURL + targetPath

	// Create and send the request
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(requestData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		// Update endpoint health
		v.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return nil, err
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		v.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return nil, fmt.Errorf("error from viper network: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read and return the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Update endpoint health to healthy
	v.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "healthy")

	return responseBody, nil
}

// ConvertJSONRPCToViperFormat converts standard JSON-RPC format to viper-network format
func ConvertJSONRPCToViperFormat(jsonRPCRequest []byte) (string, []byte, error) {
	var rpcRequest map[string]interface{}
	if err := json.Unmarshal(jsonRPCRequest, &rpcRequest); err != nil {
		return "", nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
	}

	// Extract the method to determine the viper request type
	method, ok := rpcRequest["method"].(string)
	if !ok {
		return "", nil, errors.New("missing or invalid method in JSON-RPC request")
	}

	var requestType string
	var viperRequest ViperNetworkRequest

	// Convert JSON-RPC to the appropriate Viper Network format
	switch method {
	case "eth_blockNumber":
		requestType = "height"
		viperRequest = ViperNetworkRequest{}

	case "eth_getBlockByNumber", "eth_getBlockByHash":
		requestType = "block"
		params, ok := rpcRequest["params"].([]interface{})
		if !ok || len(params) == 0 {
			return "", nil, errors.New("missing or invalid params for block request")
		}

		var height int64 = 0
		if blockNum, ok := params[0].(string); ok && blockNum != "latest" {
			// Convert hex block number to int if needed
			// For simplicity, we're just setting to 0 here which means latest
			height = 0
		}

		viperRequest = ViperNetworkRequest{
			Height: height,
		}

	case "eth_sendRawTransaction":
		requestType = "relay"
		params, ok := rpcRequest["params"].([]interface{})
		if !ok || len(params) == 0 {
			return "", nil, errors.New("missing or invalid params for transaction request")
		}

		txData, ok := params[0].(string)
		if !ok {
			return "", nil, errors.New("invalid transaction data")
		}

		viperRequest = ViperNetworkRequest{
			Blockchain: "0002", // Ethereum chain ID in viper network
			Data:       txData,
			Method:     "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

	case "eth_call", "eth_estimateGas":
		// For contract calls and gas estimates
		requestType = "relay"

		// Convert params to JSON string
		paramsData, err := json.Marshal(rpcRequest)
		if err != nil {
			return "", nil, fmt.Errorf("error serializing params: %w", err)
		}

		viperRequest = ViperNetworkRequest{
			Blockchain: "0002", // Ethereum chain ID in viper network
			Data:       string(paramsData),
			Method:     "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

	default:
		// For any other methods, we'll use a generic relay format
		requestType = "relay"

		// Convert the entire request to a JSON string
		requestData, err := json.Marshal(rpcRequest)
		if err != nil {
			return "", nil, fmt.Errorf("error serializing request: %w", err)
		}

		viperRequest = ViperNetworkRequest{
			Blockchain: "0002", // Default to Ethereum chain ID in viper network
			Data:       string(requestData),
			Method:     "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}
	}

	// Convert the prepared viper request to JSON
	viperRequestJSON, err := json.Marshal(viperRequest)
	if err != nil {
		return "", nil, fmt.Errorf("error serializing viper request: %w", err)
	}

	return requestType, viperRequestJSON, nil
}

// ConvertViperResponseToJSONRPC converts viper-network response format to standard JSON-RPC
func ConvertViperResponseToJSONRPC(viperResponse []byte, originalRequest []byte) ([]byte, error) {
	// Parse the original request to get the ID for the response
	var originalRPCRequest map[string]interface{}
	if err := json.Unmarshal(originalRequest, &originalRPCRequest); err != nil {
		return nil, fmt.Errorf("error parsing original request: %w", err)
	}

	requestID := originalRPCRequest["id"]

	// Parse the Viper response
	var viperResponseData interface{}
	if err := json.Unmarshal(viperResponse, &viperResponseData); err != nil {
		// If we can't parse as JSON, treat as string
		jsonRPCResponse := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      requestID,
			"result":  string(viperResponse),
		}
		return json.Marshal(jsonRPCResponse)
	}

	// Construct a proper JSON-RPC response
	jsonRPCResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      requestID,
		"result":  viperResponseData,
	}

	// If the viper response has an "error" field, include it
	if viperResponseMap, ok := viperResponseData.(map[string]interface{}); ok {
		if errorData, hasError := viperResponseMap["error"]; hasError {
			jsonRPCResponse["error"] = errorData
			delete(jsonRPCResponse, "result")
		}
	}

	return json.Marshal(jsonRPCResponse)
}
