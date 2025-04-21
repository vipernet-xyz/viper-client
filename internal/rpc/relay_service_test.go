package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhruvsharma/viper-client/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// RelayEndpointManager mocks the EndpointManager interface for relay tests
type RelayEndpointManager struct {
	mock.Mock
}

func (m *RelayEndpointManager) GetActiveEndpoints(chainID int) ([]models.RpcEndpoint, error) {
	args := m.Called(chainID)
	return args.Get(0).([]models.RpcEndpoint), args.Error(1)
}

func (m *RelayEndpointManager) UpdateEndpointHealth(endpointID int, status string) error {
	args := m.Called(endpointID, status)
	return args.Error(0)
}

// MockSigner mocks the CryptoSigner interface
type MockSigner struct {
	mock.Mock
}

func (m *MockSigner) Sign(message []byte) (string, error) {
	args := m.Called(message)
	return args.String(0), args.Error(1)
}

func (m *MockSigner) GetAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSigner) GetPublicKey() string {
	args := m.Called()
	return args.String(0)
}

func TestGenerateAAT(t *testing.T) {
	// Create mocks
	mockEndpointManager := new(RelayEndpointManager)
	mockSigner := new(MockSigner)

	// Set up test data
	requestorPubKey := "requestor_pub_key"
	clientPubKey := "client_pub_key"
	signature := "test_signature"

	// Configure mock behavior
	mockSigner.On("Sign", mock.Anything).Return(signature, nil)

	// Create service
	service := NewRelayService(mockEndpointManager, mockSigner)

	// Test the GenerateAAT function
	aat, err := service.GenerateAAT(requestorPubKey, clientPubKey)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, aat)
	assert.Equal(t, requestorPubKey, aat.RequestorPublicKey)
	assert.Equal(t, clientPubKey, aat.ClientPublicKey)
	assert.Equal(t, signature, aat.Signature)
	assert.Equal(t, "1.0", aat.Version)

	// Verify mock calls
	mockSigner.AssertExpectations(t)
}

func TestDispatch(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"session": {
				"key": "session_key",
				"header": {
					"requestor_public_key": "test_pubkey",
					"chain": "0001",
					"geo_zone": "0001",
					"num_servicers": 1,
					"session_block_height": 100
				},
				"servicers": [
					{
						"address": "servicer_address",
						"public_key": "servicer_pubkey",
						"node_url": "http://servicer.example.com"
					}
				]
			},
			"block_height": 100
		}`))
	}))
	defer ts.Close()

	// Create mocks
	mockEndpointManager := new(RelayEndpointManager)
	mockSigner := new(MockSigner)

	// Configure mock behavior
	mockEndpointManager.On("GetActiveEndpoints", ViperNetworkChainID).Return([]models.RpcEndpoint{
		{
			ID:           1,
			EndpointURL:  ts.URL,
			Priority:     1,
			HealthStatus: "active",
		},
	}, nil)
	mockEndpointManager.On("UpdateEndpointHealth", 1, "healthy").Return(nil)

	// Create service
	service := NewRelayService(mockEndpointManager, mockSigner)

	// Test the Dispatch function
	dispatchResponse, err := service.Dispatch(
		context.Background(),
		"test_pubkey",
		"0001",
		"0001",
		1,
	)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, dispatchResponse)
	assert.Equal(t, "session_key", dispatchResponse.Session.Key)
	assert.Equal(t, "0001", dispatchResponse.Session.Header.Chain)
	assert.Equal(t, int64(1), dispatchResponse.Session.Header.NumServicers)
	assert.Len(t, dispatchResponse.Session.Servicers, 1)
	assert.Equal(t, "servicer_pubkey", dispatchResponse.Session.Servicers[0].PublicKey)

	// Verify mock calls
	mockEndpointManager.AssertExpectations(t)
}

func TestHashRequest(t *testing.T) {
	// Create mocks
	mockEndpointManager := new(RelayEndpointManager)
	mockSigner := new(MockSigner)

	// Create service
	service := NewRelayService(mockEndpointManager, mockSigner)

	// Create test data
	payload := &models.RelayPayload{
		Data:    "test_data",
		Method:  "POST",
		Path:    "/test",
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	meta := &models.RelayMeta{
		BlockHeight:  100,
		Subscription: false,
		AI:           false,
	}

	// Test the hashRequest function
	hash, err := service.hashRequest(payload, meta)

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test consistency
	hash2, err := service.hashRequest(payload, meta)
	assert.NoError(t, err)
	assert.Equal(t, hash, hash2, "Hash should be deterministic for the same input")
}

func TestGenerateEntropy(t *testing.T) {
	// Create mocks
	mockEndpointManager := new(RelayEndpointManager)
	mockSigner := new(MockSigner)

	// Create service
	service := NewRelayService(mockEndpointManager, mockSigner)

	// Test the generateEntropy function
	entropy1, err := service.generateEntropy()
	assert.NoError(t, err)
	assert.NotZero(t, entropy1)

	// Generate a second entropy value to make sure they're different
	entropy2, err := service.generateEntropy()
	assert.NoError(t, err)
	assert.NotZero(t, entropy2)

	// They should be different (this is probabilistic but highly likely)
	assert.NotEqual(t, entropy1, entropy2, "Entropy values should be random and different")
}
