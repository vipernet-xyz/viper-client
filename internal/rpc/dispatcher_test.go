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

// Helper to create *int for ResponseTimeMs
func intPtr(i int) *int {
	return &i
}

func TestDispatcher_Forward_NodeSelection_RoundRobinTop3(t *testing.T) {
	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
	}))
	defer server.Close()

	mockManager := new(MockEndpointManager)
	endpoints := []models.RpcEndpoint{
		{ID: 1, EndpointURL: server.URL + "/ep1", ResponseTimeMs: intPtr(10)}, // Top 1
		{ID: 2, EndpointURL: server.URL + "/ep2", ResponseTimeMs: intPtr(20)}, // Top 2
		{ID: 3, EndpointURL: server.URL + "/ep3", ResponseTimeMs: intPtr(30)}, // Top 3
		{ID: 4, EndpointURL: server.URL + "/ep4", ResponseTimeMs: intPtr(40)},
		{ID: 5, EndpointURL: server.URL + "/ep5", ResponseTimeMs: intPtr(5)}, // Actually Top 1, to test sorting
	}

	// Expected order after sorting: ep5 (5ms), ep1 (10ms), ep2 (20ms), ep3 (30ms), ep4 (40ms)
	// Top 3 for round robin: ep5, ep1, ep2

	mockManager.On("GetActiveEndpoints", 1).Return(endpoints, nil)
	// Mock UpdateEndpointHealth for the endpoints that will be selected
	mockManager.On("UpdateEndpointHealth", 5, "healthy").Return(nil)
	mockManager.On("UpdateEndpointHealth", 1, "healthy").Return(nil)
	mockManager.On("UpdateEndpointHealth", 2, "healthy").Return(nil)

	dispatcher := NewDispatcher(mockManager)
	request := []byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":1}`)

	expectedSelectionURLs := []string{
		server.URL + "/ep5", // 5ms
		server.URL + "/ep1", // 10ms
		server.URL + "/ep2", // 20ms
		server.URL + "/ep5", // 5ms
		server.URL + "/ep1", // 10ms
		server.URL + "/ep2", // 20ms
	}

	for i := 0; i < len(expectedSelectionURLs); i++ {
		// If GetActiveEndpoints is called multiple times, the mock needs to be configured for it.
		// For this test, it's called once, and the dispatcher caches the round-robin index.
		// If the test were structured to call GetActiveEndpoints each time, the mock setup would be different.

		// We need to capture the actual selected endpoint to verify.
		// The Forward function uses a new http.Client for each call if not careful.
		// Let's make httpClient injectable or capture the URL from the request.
		// For now, let's assume the test server can tell us which URL was hit, or we modify Forward to return it for testing.

		// Since Forward doesn't directly return selected URL, we rely on the mock server or modify Dispatcher for testability.
		// The current Dispatcher creates its own httpClient.
		// To check which endpoint was used, we'd typically have the mock server record the request path.
		// Let's adjust the server to record the path.

		var lastRequestPath string
		dynamicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lastRequestPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
		}))
		defer dynamicServer.Close()

		// Update endpoint URLs to use the dynamic server
		endpointsForCall := []models.RpcEndpoint{
			{ID: 1, EndpointURL: dynamicServer.URL + "/ep1", ResponseTimeMs: intPtr(10)},
			{ID: 2, EndpointURL: dynamicServer.URL + "/ep2", ResponseTimeMs: intPtr(20)},
			{ID: 3, EndpointURL: dynamicServer.URL + "/ep3", ResponseTimeMs: intPtr(30)},
			{ID: 4, EndpointURL: dynamicServer.URL + "/ep4", ResponseTimeMs: intPtr(40)},
			{ID: 5, EndpointURL: dynamicServer.URL + "/ep5", ResponseTimeMs: intPtr(5)},
		}

		// Reset mock for each call iteration if GetActiveEndpoints is called per Forward
		// For this test, GetActiveEndpoints is called ONCE. The dispatcher then uses its internal state.
		// The issue is that all endpoints in the dispatcher will point to the *same* dynamicServer URL if we just update server.URL
		// We need to ensure each RpcEndpoint has its unique URL that the dynamicServer can distinguish.
		// The current setup with dynamicServer re-created in loop is flawed.

		// Let's re-initialize dispatcher for each call to simplify state, OR ensure GetActiveEndpoints is called each time.
		// The simplest is to have GetActiveEndpoints return slightly different objects if needed, or rely on the dispatcher's internal logic.

		// Let's stick to one dispatcher and one server. The key is how to identify the endpoint.
		// The selectedEndpoint.EndpointURL is used. We can check this.
		// We need a way to get the selectedEndpoint from the Forward call for robust testing.
		// Modifying Forward to return selectedEndpoint for testing is an option.
		// Alternative: have the mockManager's UpdateEndpointHealth tell us which ID was updated.

		// Let's refine the mock server to record the path and check that.
		// The server instance must be the same for all endpoints.
		// The different paths (/ep1, /ep2) will distinguish them.

		// Re-setup mock for GetActiveEndpoints for *this specific iteration's expectation* if it were called multiple times.
		// But it's called once.

		// The issue is that `endpoints` has URLs like `server.URL + "/epX"`.
		// If `server` is the generic one, we can't distinguish.
		// Let's use the dynamicServer approach but define it once.

		if i == 0 { // Setup mock for the first call (and it's only called once for GetActiveEndpoints)
			// URLs need to be updated to the dynamicServer defined outside the loop.
			// This is getting complicated. Let's simplify.
			// The core logic is sorting and round-robin. The actual HTTP call is secondary for this part of the test.
			// We can test the selection by checking which endpoint's health was updated.

			// Redefine endpoints with corrected URLs for the single dynamicServer
			// This was the flaw in previous thinking - server was per loop.
		}
		// For this test, GetActiveEndpoints is called once. The dispatcher then uses its internal state.
		// The mock for UpdateEndpointHealth will help verify.

		_, err := dispatcher.Forward(context.Background(), 1, request)
		assert.NoError(t, err)

		// To assert which endpoint was chosen, we need to inspect something.
		// The easiest is to ensure UpdateEndpointHealth is called with the correct ID.
		// The mock calls are asserted at the end.
		// We need to ensure the *sequence* of health updates.
		// Testify mock doesn't easily assert sequence of calls to *different* methods or same method with different args like this.

		// Let's make the mock server record the path.
	}
	// This test structure needs a way to verify the endpoint URL used in each iteration.
	// The httptest.Server given to endpoints needs to be the one that records.

	// Corrected approach:
	var recordedPaths []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPaths = append(recordedPaths, r.URL.Path) // Record the path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
	}))
	defer s.Close()

	mockManagerRetry := new(MockEndpointManager) // New mock for this test
	endpointsRetry := []models.RpcEndpoint{
		{ID: 1, EndpointURL: s.URL + "/ep1", ResponseTimeMs: intPtr(10)},
		{ID: 5, EndpointURL: s.URL + "/ep5", ResponseTimeMs: intPtr(5)},  // Top 1
		{ID: 2, EndpointURL: s.URL + "/ep2", ResponseTimeMs: intPtr(20)},
		{ID: 3, EndpointURL: s.URL + "/ep3", ResponseTimeMs: intPtr(30)},
		{ID: 4, EndpointURL: s.URL + "/ep4", ResponseTimeMs: intPtr(40)},
	}
	// Expected order by response time: ep5 (5ms), ep1 (10ms), ep2 (20ms), ep3 (30ms), ep4 (40ms)
	// Top 3 for round robin: ep5, ep1, ep2

	mockManagerRetry.On("GetActiveEndpoints", 1).Return(endpointsRetry, nil).Once() // Called only once
	mockManagerRetry.On("UpdateEndpointHealth", mock.AnythingOfType("int"), "healthy").Return(nil) // Allow any ID for now

	dispatcherRetry := NewDispatcher(mockManagerRetry)
	requestBody := []byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":1}`)

	expectedSelectedPaths := []string{
		"/ep5", // 5ms
		"/ep1", // 10ms
		"/ep2", // 20ms
		"/ep5", // 5ms
		"/ep1", // 10ms
		"/ep2", // 20ms
	}

	for i := 0; i < len(expectedSelectedPaths); i++ {
		_, err := dispatcherRetry.Forward(context.Background(), 1, requestBody)
		assert.NoError(t, err)
	}

	assert.Equal(t, expectedSelectedPaths, recordedPaths)
	mockManagerRetry.AssertExpectations(t)
	// We also need to assert that UpdateEndpointHealth was called for the correct IDs in sequence.
	// The recordedPaths confirms the URL, which is tied to the ID.
	// We can refine the UpdateEndpointHealth mock to be more specific if needed,
	// but path verification is strong.
	// Assert that UpdateEndpointHealth was called 6 times.
	mockManagerRetry.AssertNumberOfCalls(t, "UpdateEndpointHealth", 6)

	// To be very precise about which IDs were health-updated:
	// Expected health update IDs in order: 5, 1, 2, 5, 1, 2
	// Testify's mock doesn't directly give ordered calls for different args easily.
	// However, since `recordedPaths` is correct, and each path corresponds to a unique RpcEndpoint ID,
	// this implicitly verifies the correct endpoint (and thus ID) was chosen.
}

func TestDispatcher_Forward_NodeSelection_RoundRobinLessThanTopN(t *testing.T) {
	var recordedPaths []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPaths = append(recordedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
	}))
	defer s.Close()

	mockManager := new(MockEndpointManager)
	endpoints := []models.RpcEndpoint{
		{ID: 1, EndpointURL: s.URL + "/ep1", ResponseTimeMs: intPtr(10)}, // Top 1
		{ID: 2, EndpointURL: s.URL + "/ep2", ResponseTimeMs: intPtr(5)},  // Actually Top 1, sorted
	}
	// Expected order: ep2 (5ms), ep1 (10ms)

	mockManager.On("GetActiveEndpoints", 1).Return(endpoints, nil).Once()
	mockManager.On("UpdateEndpointHealth", mock.AnythingOfType("int"), "healthy").Return(nil)

	dispatcher := NewDispatcher(mockManager)
	requestBody := []byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":1}`)

	expectedSelectedPaths := []string{
		"/ep2", // 5ms
		"/ep1", // 10ms
		"/ep2", // 5ms
		"/ep1", // 10ms
	}

	for i := 0; i < len(expectedSelectedPaths); i++ {
		_, err := dispatcher.Forward(context.Background(), 1, requestBody)
		assert.NoError(t, err)
	}

	assert.Equal(t, expectedSelectedPaths, recordedPaths)
	mockManager.AssertExpectations(t)
	mockManager.AssertNumberOfCalls(t, "UpdateEndpointHealth", 4)
}

func TestDispatcher_Forward_NodeSelection_AllNilResponseTime(t *testing.T) {
	mockManager := new(MockEndpointManager)
	endpoints := []models.RpcEndpoint{
		{ID: 1, EndpointURL: "http://nil1.example.com", ResponseTimeMs: nil},
		{ID: 2, EndpointURL: "http://nil2.example.com", ResponseTimeMs: nil},
	}

	mockManager.On("GetActiveEndpoints", 1).Return(endpoints, nil).Once()
	// UpdateEndpointHealth should NOT be called

	dispatcher := NewDispatcher(mockManager)
	requestBody := []byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":1}`)

	_, err := dispatcher.Forward(context.Background(), 1, requestBody)
	assert.Error(t, err)
	assert.EqualError(t, err, ErrNoEndpoints.Error()) // Use the exported error variable

	mockManager.AssertExpectations(t)
	mockManager.AssertNotCalled(t, "UpdateEndpointHealth")
}

func TestDispatcher_Forward_NodeSelection_MixedResponseTimes(t *testing.T) {
	var recordedPaths []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPaths = append(recordedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
	}))
	defer s.Close()

	mockManager := new(MockEndpointManager)
	endpoints := []models.RpcEndpoint{
		{ID: 1, EndpointURL: s.URL + "/ep1", ResponseTimeMs: intPtr(10)},
		{ID: 2, EndpointURL: s.URL + "/ep2", ResponseTimeMs: nil},
		{ID: 3, EndpointURL: s.URL + "/ep3", ResponseTimeMs: intPtr(5)}, // Top 1 (sorted)
		{ID: 4, EndpointURL: s.URL + "/ep4", ResponseTimeMs: nil},
		{ID: 5, EndpointURL: s.URL + "/ep5", ResponseTimeMs: intPtr(20)},
	}
	// Expected order of valid: ep3 (5ms), ep1 (10ms), ep5 (20ms)
	// These are also the top 3.

	mockManager.On("GetActiveEndpoints", 1).Return(endpoints, nil).Once()
	mockManager.On("UpdateEndpointHealth", mock.AnythingOfType("int"), "healthy").Return(nil)

	dispatcher := NewDispatcher(mockManager)
	requestBody := []byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":1}`)

	expectedSelectedPaths := []string{
		"/ep3", // 5ms
		"/ep1", // 10ms
		"/ep5", // 20ms
		"/ep3",
		"/ep1",
		"/ep5",
	}

	for i := 0; i < len(expectedSelectedPaths); i++ {
		_, err := dispatcher.Forward(context.Background(), 1, requestBody)
		assert.NoError(t, err)
	}

	assert.Equal(t, expectedSelectedPaths, recordedPaths)
	mockManager.AssertExpectations(t)
	mockManager.AssertNumberOfCalls(t, "UpdateEndpointHealth", 6)
}

func TestDispatcher_Forward_NodeSelection_SingleValidEndpoint(t *testing.T) {
	var recordedPaths []string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPaths = append(recordedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
	}))
	defer s.Close()

	mockManager := new(MockEndpointManager)
	endpoints := []models.RpcEndpoint{
		{ID: 1, EndpointURL: s.URL + "/ep1", ResponseTimeMs: nil},
		{ID: 2, EndpointURL: s.URL + "/ep2", ResponseTimeMs: intPtr(50)}, // The only valid one
		{ID: 3, EndpointURL: s.URL + "/ep3", ResponseTimeMs: nil},
	}

	mockManager.On("GetActiveEndpoints", 1).Return(endpoints, nil).Once()
	mockManager.On("UpdateEndpointHealth", 2, "healthy").Return(nil) // Specifically ep2

	dispatcher := NewDispatcher(mockManager)
	requestBody := []byte(`{"jsonrpc":"2.0","method":"test","params":[],"id":1}`)

	expectedSelectedPaths := []string{
		"/ep2",
		"/ep2",
		"/ep2",
	}

	for i := 0; i < len(expectedSelectedPaths); i++ {
		_, err := dispatcher.Forward(context.Background(), 1, requestBody)
		assert.NoError(t, err)
	}

	assert.Equal(t, expectedSelectedPaths, recordedPaths)
	mockManager.AssertExpectations(t)
	mockManager.AssertNumberOfCalls(t, "UpdateEndpointHealth", 3)
}

// TestDispatcher_Forward_NoEndpoints (existing test) already covers the case of empty endpoint list from manager.
// It asserts: assert.Equal(t, "no active endpoints available for the requested chain", err.Error())
// which is ErrNoEndpoints.Error().

// TestDispatcher_Forward_EndpointError (existing test) already covers GetActiveEndpoints returning an error.
// It asserts: assert.Equal(t, "database error", err.Error()) which is the mocked error.

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
