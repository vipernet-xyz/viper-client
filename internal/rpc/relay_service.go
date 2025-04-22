package rpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/illegalcall/viper-client/internal/models"
	"github.com/illegalcall/viper-client/internal/utils"
)

const (
	// RelayType constants
	RelayTypeRegular      = 1
	RelayTypeSubscription = 2
)

// RelayService provides functionality for interacting with the Viper Network's relay system
type RelayService struct {
	httpClient      *http.Client
	endpointManager EndpointManager
	cryptoSigner    CryptoSigner
}

// CryptoSigner defines the interface for signing operations
type CryptoSigner interface {
	Sign(message []byte) (signature string, err error)
	GetAddress() string
	GetPublicKey() string
}

// NewRelayService creates a new relay service
func NewRelayService(manager EndpointManager, signer CryptoSigner) *RelayService {
	return &RelayService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Longer timeout for relay operations
		},
		endpointManager: manager,
		cryptoSigner:    signer,
	}
}

// getHeight fetches the current block height from the viper network
func (s *RelayService) getHeight(ctx context.Context) (int64, error) {
	// Get active endpoints for viper network
	endpoints, err := s.endpointManager.GetActiveEndpoints(ViperNetworkChainID)
	if err != nil {
		return 0, err
	}

	if len(endpoints) == 0 {
		return 0, ErrNoEndpoints
	}

	// Select the highest priority endpoint
	selectedEndpoint := endpoints[0]

	// Construct the full URL
	fullURL := selectedEndpoint.EndpointURL + ViperHeightEndpoint

	// Create and send the request
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return 0, err
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return 0, fmt.Errorf("error from viper network: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read and parse the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var heightResponse struct {
		Height int64 `json:"height"`
	}
	if err := json.Unmarshal(responseBody, &heightResponse); err != nil {
		return 0, err
	}

	// Update endpoint health to healthy
	s.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "healthy")

	return heightResponse.Height, nil
}

// Dispatch sends a dispatch request to the Viper Network
func (s *RelayService) Dispatch(ctx context.Context, pubKey, chain, geoZone string, numServicers int64) (*models.DispatchResponse, error) {
	// Create dispatch request body
	dispatch := struct {
		RequestorPubKey    string `json:"requestor_public_key"`
		Chain              string `json:"chain"`
		GeoZone            string `json:"geo_zone"`
		NumServicers       int64  `json:"num_servicers"`
		SessionBlockHeight int64  `json:"session_block_height"`
	}{
		RequestorPubKey:    pubKey,
		Chain:              chain,
		GeoZone:            geoZone,
		NumServicers:       numServicers,
		SessionBlockHeight: 0, // Will be filled by the network
	}

	// Marshal to JSON
	dispatchJSON, err := json.Marshal(dispatch)
	if err != nil {
		return nil, fmt.Errorf("error marshaling dispatch request: %w", err)
	}

	// Get active endpoints for viper network
	endpoints, err := s.endpointManager.GetActiveEndpoints(ViperNetworkChainID)
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	// Select the highest priority endpoint
	selectedEndpoint := endpoints[0]

	// Construct the full URL
	fullURL := selectedEndpoint.EndpointURL + ViperDispatchEndpoint

	// Create and send the request
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(dispatchJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return nil, err
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "error")
		return nil, fmt.Errorf("error from viper network: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read and parse the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dispatchResponse models.DispatchResponse
	if err := json.Unmarshal(responseBody, &dispatchResponse); err != nil {
		return nil, err
	}

	// Update endpoint health to healthy
	s.endpointManager.UpdateEndpointHealth(selectedEndpoint.ID, "healthy")

	return &dispatchResponse, nil
}

// GenerateAAT generates an Application Authentication Token
func (s *RelayService) GenerateAAT(requestorPubKey, clientPubKey string) (*models.ViperAAT, error) {
	// Create the AAT object
	aat := &models.ViperAAT{
		Version:            "1.0",
		RequestorPublicKey: requestorPubKey,
		ClientPublicKey:    clientPubKey,
	}

	// Create a message to sign
	message := fmt.Sprintf("%s%s%s", aat.Version, aat.RequestorPublicKey, aat.ClientPublicKey)

	// Sign the message
	signature, err := s.cryptoSigner.Sign([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("error signing AAT: %w", err)
	}

	aat.Signature = signature
	return aat, nil
}

// generateEntropy generates a random number for relay entropy
func (s *RelayService) generateEntropy() (int64, error) {
	entropyInt, err := rand.Int(rand.Reader, big.NewInt(9000000000000000000))
	if err != nil {
		return 0, fmt.Errorf("error generating entropy: %w", err)
	}
	return entropyInt.Int64(), nil
}

// hashRequest creates a hash of the relay payload and meta
func (s *RelayService) hashRequest(payload *models.RelayPayload, meta *models.RelayMeta) (string, error) {
	// In a real implementation, this would create a proper hash
	// For simplicity, we're just creating a string representation
	combined, err := json.Marshal(struct {
		Payload *models.RelayPayload `json:"payload"`
		Meta    *models.RelayMeta    `json:"meta"`
	}{
		Payload: payload,
		Meta:    meta,
	})
	if err != nil {
		return "", err
	}

	return utils.SHA3Hash(string(combined)), nil
}

// BuildRelayRequest constructs a complete relay request with proof
func (s *RelayService) BuildRelayRequest(ctx context.Context, servicer *models.Servicer,
	blockchain, geoZone string, numServicers int64,
	data, method, path string, headers map[string]string,
	aat *models.ViperAAT) (*models.Relay, error) {

	// Get the current height
	height, err := s.getHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting height: %w", err)
	}

	// Create relay payload
	payload := &models.RelayPayload{
		Data:    data,
		Method:  method,
		Path:    path,
		Headers: headers,
	}

	// Create relay metadata
	meta := &models.RelayMeta{
		BlockHeight:  height,
		Subscription: false,
		AI:           false,
	}

	// Generate request hash
	requestHash, err := s.hashRequest(payload, meta)
	if err != nil {
		return nil, fmt.Errorf("error hashing request: %w", err)
	}

	// Generate entropy
	entropy, err := s.generateEntropy()
	if err != nil {
		return nil, fmt.Errorf("error generating entropy: %w", err)
	}

	// Create relay proof
	proof := &models.RelayProof{
		RequestHash:        requestHash,
		Entropy:            entropy,
		SessionBlockHeight: height,
		ServicerPubKey:     servicer.PublicKey,
		Blockchain:         blockchain,
		Token:              *aat,
		GeoZone:            geoZone,
		NumServicers:       numServicers,
		RelayType:          RelayTypeRegular,
		Weight:             1,
		Address:            s.cryptoSigner.GetAddress(),
	}

	// Create message to sign
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
	signature, err := s.cryptoSigner.Sign([]byte(proofMsg))
	if err != nil {
		return nil, fmt.Errorf("error signing relay proof: %w", err)
	}
	proof.Signature = signature

	// Create the complete relay request
	relay := &models.Relay{
		Payload: *payload,
		Meta:    *meta,
		Proof:   *proof,
	}

	return relay, nil
}

// SendRelay sends a relay request to a Viper Network servicer
func (s *RelayService) SendRelay(ctx context.Context, relay *models.Relay, serviceURL string) (*models.RelayResponse, error) {
	if serviceURL == "" {
		return nil, errors.New("servicer URL is required")
	}

	// Marshal relay to JSON
	relayJSON, err := json.Marshal(relay)
	if err != nil {
		return nil, fmt.Errorf("error marshaling relay: %w", err)
	}

	// Create the request URL
	requestURL := serviceURL + ViperRelayEndpoint

	// Create and send the request
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(relayJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error from servicer: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read and parse the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var relayResponse models.RelayResponse
	if err := json.Unmarshal(responseBody, &relayResponse); err != nil {
		return nil, err
	}

	return &relayResponse, nil
}

// ExecuteRelay performs the complete relay process
func (s *RelayService) ExecuteRelay(ctx context.Context,
	pubKey, blockchain, geoZone string, numServicers int64,
	data, method, path string, headers map[string]string) (*models.RelayResponse, error) {

	// Step 1: Dispatch a session
	dispatchResponse, err := s.Dispatch(ctx, pubKey, blockchain, geoZone, numServicers)
	if err != nil {
		return nil, fmt.Errorf("dispatch error: %w", err)
	}

	if len(dispatchResponse.Session.Servicers) == 0 {
		return nil, errors.New("no servicers available in the dispatched session")
	}

	// Step 2: Generate AAT
	aat, err := s.GenerateAAT(pubKey, s.cryptoSigner.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("AAT generation error: %w", err)
	}

	// Step 3: Build relay request
	servicer := &models.Servicer{
		Address:   dispatchResponse.Session.Servicers[0].Address,
		PublicKey: dispatchResponse.Session.Servicers[0].PublicKey,
		NodeURL:   dispatchResponse.Session.Servicers[0].NodeURL,
	}

	relay, err := s.BuildRelayRequest(ctx, servicer, blockchain, geoZone, numServicers,
		data, method, path, headers, aat)
	if err != nil {
		return nil, fmt.Errorf("relay request build error: %w", err)
	}

	// Step 4: Send relay to servicer
	relayResponse, err := s.SendRelay(ctx, relay, servicer.NodeURL)
	if err != nil {
		return nil, fmt.Errorf("relay send error: %w", err)
	}

	return relayResponse, nil
}
