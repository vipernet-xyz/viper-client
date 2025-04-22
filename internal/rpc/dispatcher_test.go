package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/illegalcall/viper-client/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEndpointManager is a mock implementation of the EndpointManager interface
type MockEndpointManager struct {
	mock.Mock
}

func (m *MockEndpointManager) GetActiveEndpoints(chainID int) ([]models.RpcEndpoint, error) {
	args := m.Called(chainID)
	return args.Get(0).([]models.RpcEndpoint), args.Error(1)
}

func (m *MockEndpointManager) UpdateEndpointHealth(id int, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func TestDispatcher_Forward_NoEndpoints(t *testing.T) {
	// Setup mock
	mockManager := new(MockEndpointManager)
	mockManager.On("GetActiveEndpoints", 1).Return([]models.RpcEndpoint{}, nil)

	// Create dispatcher with mock
	dispatcher := NewDispatcher(mockManager)

	// Test with valid JSON-RPC request
	request := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	_, err := dispatcher.Forward(context.Background(), 1, request)

	// Expect error for no endpoints
	assert.Error(t, err)
	assert.Equal(t, "no active endpoints available for the requested chain", err.Error())

	// Verify expectations
	mockManager.AssertExpectations(t)
}

func TestDispatcher_Forward_InvalidJSON(t *testing.T) {
	// Setup mock
	mockManager := new(MockEndpointManager)

	// Create dispatcher with mock
	dispatcher := NewDispatcher(mockManager)

	// Test with invalid JSON request
	request := []byte(`{invalid json}`)
	_, err := dispatcher.Forward(context.Background(), 1, request)

	// Expect error for invalid JSON
	assert.Error(t, err)
	assert.Equal(t, "invalid JSON-RPC request format", err.Error())

	// Verify no calls were made to the endpoint manager
	mockManager.AssertNotCalled(t, "GetActiveEndpoints")
}

func TestDispatcher_Forward_EndpointError(t *testing.T) {
	// Setup mock
	mockManager := new(MockEndpointManager)
	mockManager.On("GetActiveEndpoints", 1).Return([]models.RpcEndpoint{}, errors.New("database error"))

	// Create dispatcher with mock
	dispatcher := NewDispatcher(mockManager)

	// Test with valid JSON-RPC request
	request := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	_, err := dispatcher.Forward(context.Background(), 1, request)

	// Expect error from endpoint manager
	assert.Error(t, err)
	assert.Equal(t, "database error", err.Error())

	// Verify expectations
	mockManager.AssertExpectations(t)
}

func TestDispatcher_Forward_Success(t *testing.T) {
	// Start a test HTTP server that mimics an RPC endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1234"}`))
	}))
	defer server.Close()

	// Setup mock with a test endpoint using the test server URL
	mockManager := new(MockEndpointManager)
	endpoints := []models.RpcEndpoint{
		{
			ID:          1,
			ChainID:     1,
			EndpointURL: server.URL,
			IsActive:    true,
			Priority:    10,
		},
	}
	mockManager.On("GetActiveEndpoints", 1).Return(endpoints, nil)
	mockManager.On("UpdateEndpointHealth", 1, "healthy").Return(nil)

	// Create dispatcher with mock
	dispatcher := NewDispatcher(mockManager)

	// Test with valid JSON-RPC request
	request := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	response, err := dispatcher.Forward(context.Background(), 1, request)

	// Expect success
	assert.NoError(t, err)
	assert.Contains(t, string(response), `"result":"0x1234"`)

	// Verify expectations
	mockManager.AssertExpectations(t)
}
