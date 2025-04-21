package relay

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/dhruvsharma/viper-client/internal/utils"
)

// Constants
const (
	// Default viper network endpoints
	DefaultViperNetworkEndpoint = "http://127.0.0.1:8082"

	// Viper endpoints
	ViperHeightEndpoint   = "/v1/query/height"
	ViperDispatchEndpoint = "/v1/client/dispatch"
	ViperRelayEndpoint    = "/v1/client/relay"
)

// Client provides a high-level client for interacting with the relay API
type Client struct {
	baseURL    string
	appID      string
	apiKey     string
	httpClient *http.Client
	signer     *utils.Signer
}

// NewClient creates a new relay client
func NewClient(baseURL, appID, apiKey string) (*Client, error) {
	// Create a random signer for crypto operations
	signer, err := utils.NewRandomSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto signer: %w", err)
	}

	return &Client{
		baseURL: baseURL,
		appID:   appID,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		signer: signer,
	}, nil
}

// NewClientWithSigner creates a new relay client with a specific signer
func NewClientWithSigner(baseURL, appID, apiKey string, privateKey string) (*Client, error) {
	// Create a signer from the provided private key
	signer, err := utils.NewSignerFromPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto signer from private key: %w", err)
	}

	return &Client{
		baseURL: baseURL,
		appID:   appID,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		signer: signer,
	}, nil
}

// Options contains options for relay requests
type Options struct {
	PubKey       string            // Requestor public key
	Blockchain   string            // Target blockchain ID (hex)
	GeoZone      string            // Geo zone (hex)
	NumServicers int64             // Number of servicers to include
	Data         string            // Payload data (usually JSON-RPC)
	Method       string            // HTTP method (POST, GET, etc.)
	Path         string            // Custom path for the relay
	Headers      map[string]string // HTTP headers
}

// GetHeight gets the current block height from viper network
func (c *Client) GetHeight(ctx context.Context) (int64, error) {
	// For direct connection to viper network
	url := DefaultViperNetworkEndpoint + ViperHeightEndpoint

	// If using viper-client REST API
	if c.baseURL != "" {
		url = c.baseURL + "/relay/height"
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.baseURL != "" {
		req.Header.Set("X-App-ID", c.appID)
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("error from server: %s (status %d)", string(respBody), resp.StatusCode)
	}

	// Parse the height response
	var heightResp struct {
		Height int64 `json:"height"`
	}
	if err := json.Unmarshal(respBody, &heightResp); err != nil {
		return 0, err
	}

	return heightResp.Height, nil
}

// Dispatch sends a dispatch request to get a session
func (c *Client) Dispatch(ctx context.Context, opts Options) (*models.DispatchResponse, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"requestor_public_key": opts.PubKey,
		"chain":                opts.Blockchain,
		"geo_zone":             opts.GeoZone,
		"num_servicers":        opts.NumServicers,
	}

	// Marshal to JSON
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Choose the right URL
	url := DefaultViperNetworkEndpoint + ViperDispatchEndpoint
	if c.baseURL != "" {
		url = c.baseURL + "/relay/dispatch"
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.baseURL != "" {
		req.Header.Set("X-App-ID", c.appID)
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from server: %s (status %d)", string(respBody), resp.StatusCode)
	}

	// Parse the dispatch response
	var dispatchResp models.DispatchResponse
	if err := json.Unmarshal(respBody, &dispatchResp); err != nil {
		return nil, err
	}

	return &dispatchResp, nil
}

// generateAAT generates an Application Authentication Token
func (c *Client) generateAAT(requestorPubKey string) (*models.ViperAAT, error) {
	// Create the AAT object
	aat := &models.ViperAAT{
		Version:            "1.0",
		RequestorPublicKey: requestorPubKey,
		ClientPublicKey:    c.signer.GetPublicKey(),
	}

	// Create a message to sign: version + requestorPubKey + clientPubKey
	message := fmt.Sprintf("%s%s%s", aat.Version, aat.RequestorPublicKey, aat.ClientPublicKey)

	// Sign the message
	signature, err := c.signer.Sign([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("error signing AAT: %w", err)
	}

	aat.Signature = signature
	return aat, nil
}

// generateEntropy generates a random number for relay entropy
func (c *Client) generateEntropy() (int64, error) {
	entropyInt, err := rand.Int(rand.Reader, big.NewInt(9000000000000000000))
	if err != nil {
		return 0, fmt.Errorf("error generating entropy: %w", err)
	}
	return entropyInt.Int64(), nil
}

// hashRequest creates a proper hash of the relay payload and meta
func (c *Client) hashRequest(payload *models.RelayPayload, meta *models.RelayMeta) (string, error) {
	// Create combined structure to hash
	combined := struct {
		Payload *models.RelayPayload `json:"payload"`
		Meta    *models.RelayMeta    `json:"meta"`
	}{
		Payload: payload,
		Meta:    meta,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(combined)
	if err != nil {
		return "", fmt.Errorf("error marshaling payload+meta: %w", err)
	}

	// Use SHA3 hash function
	return utils.SHA3Hash(string(jsonData)), nil
}

// buildRelayProof builds a properly signed relay proof
func (c *Client) buildRelayProof(
	requestHash string,
	entropy int64,
	sessionHeight int64,
	servicerPubKey string,
	blockchain string,
	aat *models.ViperAAT,
	geoZone string,
	numServicers int64,
) (*models.RelayProof, error) {
	// Create relay proof
	proof := &models.RelayProof{
		RequestHash:        requestHash,
		Entropy:            entropy,
		SessionBlockHeight: sessionHeight,
		ServicerPubKey:     servicerPubKey,
		Blockchain:         blockchain,
		Token:              *aat,
		GeoZone:            geoZone,
		NumServicers:       numServicers,
		RelayType:          1, // Regular relay
		Weight:             1,
		Address:            c.signer.GetAddress(),
	}

	// Create message to sign
	// Format: requestHash+entropy+sessionHeight+servicerPubKey+blockchain+geoZone+token.signature+numServicers+relayType+weight+address
	proofMsg := fmt.Sprintf("%s%d%d%s%s%s%s%d%d%d%s",
		proof.RequestHash,
		proof.Entropy,
		proof.SessionBlockHeight,
		proof.ServicerPubKey,
		proof.Blockchain,
		proof.GeoZone,
		proof.Token.Signature,
		proof.NumServicers,
		proof.RelayType,
		proof.Weight,
		proof.Address,
	)

	// Sign the proof
	signature, err := c.signer.Sign([]byte(proofMsg))
	if err != nil {
		return nil, fmt.Errorf("error signing relay proof: %w", err)
	}
	proof.Signature = signature

	return proof, nil
}

// BuildRelay builds a complete relay request
func (c *Client) BuildRelay(ctx context.Context, session *models.Session, opts Options) (*models.Relay, error) {
	if session == nil || len(session.Servicers) == 0 {
		return nil, fmt.Errorf("invalid session or no servicers available")
	}

	// Get the first servicer
	servicer := session.Servicers[0]

	// Get session block height
	sessionHeight := session.Header.SessionBlockHeight

	// Create the payload
	payload := &models.RelayPayload{
		Data:    opts.Data,
		Method:  opts.Method,
		Path:    opts.Path,
		Headers: opts.Headers,
	}

	// Create metadata
	meta := &models.RelayMeta{
		BlockHeight:  sessionHeight,
		Subscription: false, // Regular relay
		AI:           false, // No AI needed
	}

	// Generate request hash
	requestHash, err := c.hashRequest(payload, meta)
	if err != nil {
		return nil, fmt.Errorf("error creating request hash: %w", err)
	}

	// Generate entropy
	entropy, err := c.generateEntropy()
	if err != nil {
		return nil, fmt.Errorf("error generating entropy: %w", err)
	}

	// Generate AAT
	aat, err := c.generateAAT(opts.PubKey)
	if err != nil {
		return nil, fmt.Errorf("error generating AAT: %w", err)
	}

	// Build the proof
	proof, err := c.buildRelayProof(
		requestHash,
		entropy,
		sessionHeight,
		servicer.PublicKey,
		opts.Blockchain,
		aat,
		opts.GeoZone,
		opts.NumServicers,
	)
	if err != nil {
		return nil, fmt.Errorf("error building proof: %w", err)
	}

	// Create the complete relay
	relay := &models.Relay{
		Payload: *payload,
		Meta:    *meta,
		Proof:   *proof,
	}

	return relay, nil
}

// SendRelay sends a relay request to a servicer
func (c *Client) SendRelay(ctx context.Context, relay *models.Relay, serviceURL string) (*models.RelayResponse, error) {
	// Marshal relay to JSON
	relayJSON, err := json.Marshal(relay)
	if err != nil {
		return nil, fmt.Errorf("error marshaling relay: %w", err)
	}

	// If serviceURL is not provided, use default
	if serviceURL == "" {
		serviceURL = DefaultViperNetworkEndpoint
	}

	// Create request URL
	url := serviceURL + ViperRelayEndpoint

	// Create and send the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(relayJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from server: %s (status %d)", string(respBody), resp.StatusCode)
	}

	// Parse response
	var relayResp models.RelayResponse
	if err := json.Unmarshal(respBody, &relayResp); err != nil {
		return nil, err
	}

	return &relayResp, nil
}

// ExecuteRelay performs a complete relay operation
func (c *Client) ExecuteRelay(ctx context.Context, opts Options) (*models.RelayResponse, error) {
	// Step 1: Dispatch a session
	dispatchResp, err := c.Dispatch(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("dispatch error: %w", err)
	}

	if len(dispatchResp.Session.Servicers) == 0 {
		return nil, fmt.Errorf("no servicers available in the dispatched session")
	}

	// Step 2: Build the relay using the session
	relay, err := c.BuildRelay(ctx, &dispatchResp.Session, opts)
	if err != nil {
		return nil, fmt.Errorf("error building relay: %w", err)
	}

	// Step 3: Send the relay
	servicerURL := dispatchResp.Session.Servicers[0].NodeURL
	relayResp, err := c.SendRelay(ctx, relay, servicerURL)
	if err != nil {
		return nil, fmt.Errorf("error sending relay: %w", err)
	}

	return relayResp, nil
}

// DirectRelay sends a relay directly to a specific servicer
func (c *Client) DirectRelay(ctx context.Context, opts Options, servicerURL, servicerPubKey string) (*models.RelayResponse, error) {
	// Get current height
	height, err := c.GetHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting height: %w", err)
	}

	// Create a minimal session
	session := &models.Session{
		Header: models.Header{
			RequestorPubKey:    opts.PubKey,
			Chain:              opts.Blockchain,
			GeoZone:            opts.GeoZone,
			NumServicers:       opts.NumServicers,
			SessionBlockHeight: height,
		},
		Servicers: []models.Servicer{
			{
				PublicKey: servicerPubKey,
				NodeURL:   servicerURL,
				Address:   c.signer.GetAddress(), // Use our address
			},
		},
	}

	// Build relay
	relay, err := c.BuildRelay(ctx, session, opts)
	if err != nil {
		return nil, fmt.Errorf("error building relay: %w", err)
	}

	// Send relay
	return c.SendRelay(ctx, relay, servicerURL)
}

// BlockchainRPC sends a simplified RPC request to a blockchain
func (c *Client) BlockchainRPC(ctx context.Context, blockchain, method string, params []interface{}) (interface{}, error) {
	// Create JSON-RPC 2.0 request
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}

	// Convert to JSON
	rpcJSON, err := json.Marshal(rpcRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshaling RPC request: %w", err)
	}

	// Create relay options
	opts := Options{
		PubKey:       "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7", // Default public key
		Blockchain:   blockchain,
		GeoZone:      "0001", // Default geo zone
		NumServicers: 1,      // Default number of servicers
		Data:         string(rpcJSON),
		Method:       "POST",
		Headers:      map[string]string{"Content-Type": "application/json"},
	}

	// Execute relay
	relayResp, err := c.ExecuteRelay(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var rpcResponse map[string]interface{}
	if err := json.Unmarshal([]byte(relayResp.Response), &rpcResponse); err != nil {
		return nil, fmt.Errorf("error parsing RPC response: %w", err)
	}

	// Check for RPC error
	if errVal, ok := rpcResponse["error"]; ok && errVal != nil {
		return nil, fmt.Errorf("RPC error: %v", errVal)
	}

	return rpcResponse["result"], nil
}

// GetPublicKey returns the client's public key
func (c *Client) GetPublicKey() (string, error) {
	return c.signer.GetPublicKey(), nil
}

// GetAddress returns the client's address
func (c *Client) GetAddress() (string, error) {
	return c.signer.GetAddress(), nil
}

// GetPrivateKey returns the client's private key (only the seed portion)
func (c *Client) GetPrivateKey() (string, error) {
	// For ED25519, we only need the first 32 bytes (64 hex chars) as the seed
	fullKey := c.signer.GetPrivateKey()
	if len(fullKey) >= 64 {
		return fullKey[:64], nil
	}
	return fullKey, nil
}
