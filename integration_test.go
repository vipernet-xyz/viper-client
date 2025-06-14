//go:build integration
// +build integration

package main

import (
	"database/sql"
	"os"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/internal/api"
	"github.com/illegalcall/viper-client/internal/apps"
	"github.com/illegalcall/viper-client/internal/db"
	"github.com/illegalcall/viper-client/internal/models"
	"github.com/illegalcall/viper-client/internal/relay"
	"github.com/illegalcall/viper-client/internal/rpc"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testChainID = 1
	testAPIKey  = "test-integration-api-key"
)

// Helper to create *int for ResponseTimeMs
func intPtr(i int) *int {
	return &i
}

func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL environment variable not set")
	}
	d, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}
	err = d.Ping()
	if err != nil {
		d.Close()
		t.Fatalf("Failed to ping database: %v", err)
	}
	return d
}

func clearTable(t *testing.T, dbConn *sql.DB, tableName string) {
	t.Helper()
	_, err := dbConn.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	require.NoError(t, err, "Failed to clear table %s", tableName)
	// Reset sequences if any primary keys are serial, for some tables.
	// For rpc_endpoints and apps, this might be relevant if tests assume specific IDs.
	// However, for these tests, we mostly care about the data itself, not specific sequence-generated IDs.
	// Example: _, err = dbConn.Exec(fmt.Sprintf("ALTER SEQUENCE %s_id_seq RESTART WITH 1", tableName))
	// require.NoError(t, err, "Failed to reset sequence for table %s", tableName)
}

func insertRPCEndpoint(t *testing.T, dbConn *sql.DB, ep models.RpcEndpoint) {
	t.Helper()
	query := `
		INSERT INTO rpc_endpoints (chain_id, endpoint_url, provider, is_active, priority,
                               geozone, response_time_ms, health_status, servicer_type, public_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`

	// Ensure default values if not set, matching DB constraints or logic
	if ep.Provider == "" {
		ep.Provider = "test-provider"
	}
	if ep.Geozone == "" { // Geozone is used by DBEndpointManager's default query
		ep.Geozone = "IND"
	}
	if ep.HealthStatus == "" {
		ep.HealthStatus = "healthy"
	}
	if ep.ServicerType == "" {
		ep.ServicerType = "static"
	}
	if ep.CreatedAt.IsZero() {
		ep.CreatedAt = time.Now()
	}
	if ep.UpdatedAt.IsZero() {
		ep.UpdatedAt = time.Now()
	}

	_, err := dbConn.Exec(query, ep.ChainID, ep.EndpointURL, ep.Provider, ep.IsActive, ep.Priority,
		ep.Geozone, ep.ResponseTimeMs, ep.HealthStatus, ep.ServicerType, ep.PublicKey, ep.CreatedAt, ep.UpdatedAt)
	require.NoError(t, err, "Failed to insert RPC endpoint: %+v", ep)
}

func insertApp(t *testing.T, dbConn *sql.DB, app models.App) models.App {
	t.Helper()
	query := `
		INSERT INTO apps (name, description, owner_id, api_key, allowed_chain_ids, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`

	if app.Name == "" {
		app.Name = "Test App"
	}
    if app.OwnerID == "" {
        app.OwnerID = "test-owner" // Assuming owner_id is a string; adjust if it's an int/UUID from users table
    }
	if app.CreatedAt.IsZero() {
		app.CreatedAt = time.Now()
	}
	if app.UpdatedAt.IsZero() {
		app.UpdatedAt = time.Now()
	}
	if app.AllowedChainIDs == nil {
		app.AllowedChainIDs = []int{testChainID} // Default to allow the test chain
	}

	err := dbConn.QueryRow(query, app.Name, app.Description, app.OwnerID, app.APIKey, pq.Array(app.AllowedChainIDs), app.CreatedAt, app.UpdatedAt).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)
	require.NoError(t, err, "Failed to insert app: %+v", app)
	return app
}


type mockDownstreamServer struct {
	*httptest.Server
	mu            sync.Mutex
	calledPaths   []string
	requestBodies [][]byte
}

func newMockDownstreamServer(t *testing.T, handlerResponse string) *mockDownstreamServer {
	t.Helper()
	s := &mockDownstreamServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.calledPaths = append(s.calledPaths, r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		s.requestBodies = append(s.requestBodies, body)
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(handlerResponse))
		require.NoError(t, err)
	}))
	return s
}

func (s *mockDownstreamServer) GetCalledPaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return a copy
	paths := make([]string, len(s.calledPaths))
	copy(paths, s.calledPaths)
	return paths
}

func (s *mockDownstreamServer) ResetCalls() {
	s.mu.Lock()
	s.calledPaths = nil
	s.requestBodies = nil
	s.mu.Unlock()
}

func setupTestApplication(t *testing.T, testDB *sql.DB) *httptest.Server {
	t.Helper()

	// Initialize services and handlers as in main.go but with testDB
	dbManager := db.DB{DB: testDB} // Wrap *sql.DB

	// Ensure a test app exists for the API key
	clearTable(t, testDB, "apps") // Clear before inserting to avoid conflicts if run multiple times
	_ = insertApp(t, testDB, models.App{APIKey: testAPIKey, AllowedChainIDs: []int{testChainID}})


	appsSvc := apps.NewService(&dbManager) // apps.NewService expects *db.DB
	endpointMgr := rpc.NewDBEndpointManager(testDB)
	rpcDispatcher := rpc.NewDispatcher(endpointMgr)
	relaySvc := relay.NewService(&dbManager, appsSvc, rpcDispatcher)
	relayHandler := api.NewRelayHandler(relaySvc)

	router := gin.New()
	// In main.go, relay routes are registered on a group "/".
	// The actual path defined in RegisterRoutes for RelayHandler is "/relay"
	// So the full path will be "/relay"
	relayGroup := router.Group("/")
	relayHandler.RegisterRoutes(relayGroup)

	return httptest.NewServer(router)
}


func TestDockerSetup(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	// Wait for services to start
	time.Sleep(5 * time.Second)

	// Try to connect to PostgreSQL
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("Failed to open DB connection: %v", err)
	}
	defer db.Close()

	// Verify connection
	err = db.Ping()
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	t.Log("Successfully connected to PostgreSQL")
}

func TestIntegrationDispatcher_RoundRobinTopN(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	testDB := getTestDB(t)
	defer testDB.Close()

	clearTable(t, testDB, "rpc_endpoints")
	// clearTable(t, testDB, "apps") // Done in setupTestApplication

	// Mock downstream servers
	mockServers := make([]*mockDownstreamServer, 5)
	for i := range mockServers {
		// Unique path for each server to distinguish calls
		mockServers[i] = newMockDownstreamServer(t, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"ok_server_%d"}`, i))
		defer mockServers[i].Close()
	}

	// Insert RPC Endpoints
	// Order: s1 (10ms), s0 (20ms), s2 (30ms), s3 (40ms), s4 (50ms)
	// After sorting by ResponseTimeMs: s1, s0, s2, s3, s4
	// Top 3 for round robin: s1, s0, s2
	endpoints := []models.RpcEndpoint{
		{ChainID: testChainID, EndpointURL: mockServers[0].URL + "/ep0", ResponseTimeMs: intPtr(20), IsActive: true, Geozone: "IND"}, // Will be 2nd
		{ChainID: testChainID, EndpointURL: mockServers[1].URL + "/ep1", ResponseTimeMs: intPtr(10), IsActive: true, Geozone: "IND"}, // Will be 1st
		{ChainID: testChainID, EndpointURL: mockServers[2].URL + "/ep2", ResponseTimeMs: intPtr(30), IsActive: true, Geozone: "IND"}, // Will be 3rd
		{ChainID: testChainID, EndpointURL: mockServers[3].URL + "/ep3", ResponseTimeMs: intPtr(40), IsActive: true, Geozone: "IND"},
		{ChainID: testChainID, EndpointURL: mockServers[4].URL + "/ep4", ResponseTimeMs: intPtr(50), IsActive: true, Geozone: "IND"},
	}
	for _, ep := range endpoints {
		insertRPCEndpoint(t, testDB, ep)
	}

	appServer := setupTestApplication(t, testDB)
	defer appServer.Close()

	requestBody := []byte(`{"jsonrpc":"2.0","method":"test_method","params":[],"id":1}`)

	expectedCallOrderPaths := []string{
		"/ep1", // server 1 (10ms)
		"/ep0", // server 0 (20ms)
		"/ep2", // server 2 (30ms)
		"/ep1", // server 1
		"/ep0", // server 0
		"/ep2", // server 2
	}
	expectedServersCalled := []*mockDownstreamServer{
		mockServers[1], mockServers[0], mockServers[2],
		mockServers[1], mockServers[0], mockServers[2],
	}


	for i := 0; i < len(expectedCallOrderPaths); i++ {
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/relay?chain_id=%d&api_key=%s", appServer.URL, testChainID, testAPIKey), bytes.NewBuffer(requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Request %d failed", i+1)
		// bodyBytes, _ := io.ReadAll(resp.Body)
		// log.Printf("Run %d, Response: %s", i+1, string(bodyBytes))
	}

	// Verify calls to mock servers
	var actualCalledPaths []string
	if len(mockServers[1].GetCalledPaths()) > 0 { // s1
		actualCalledPaths = append(actualCalledPaths, mockServers[1].GetCalledPaths()...)
	}
	if len(mockServers[0].GetCalledPaths()) > 0 { // s0
		actualCalledPaths = append(actualCalledPaths, mockServers[0].GetCalledPaths()...)
	}
	if len(mockServers[2].GetCalledPaths()) > 0 { // s2
		actualCalledPaths = append(actualCalledPaths, mockServers[2].GetCalledPaths()...)
	}
	// This way of checking paths is wrong. The paths are recorded on each server independently.
	// We need to check the count for each server that should have been called.

	assert.Len(t, mockServers[1].GetCalledPaths(), 2, "Server 1 (10ms) should be called twice")
	assert.Len(t, mockServers[0].GetCalledPaths(), 2, "Server 0 (20ms) should be called twice")
	assert.Len(t, mockServers[2].GetCalledPaths(), 2, "Server 2 (30ms) should be called twice")
	assert.Len(t, mockServers[3].GetCalledPaths(), 0, "Server 3 (40ms) should not be called")
	assert.Len(t, mockServers[4].GetCalledPaths(), 0, "Server 4 (50ms) should not be called")

	// To check the exact sequence, we need a centralized way of recording calls, or make requests sequentially and check one by one.
	// The current dispatcher logic re-fetches and re-sorts for each call in this test setup because a new dispatcher is made per call via setupTestApplication.
	// This is not ideal. The dispatcher instance should persist across calls.
	// Let's fix setupTestApplication to return the router, and we make one appServer from it.

	// For now, let's assume the dispatcher within relayService is persistent for the life of appServer.
	// The issue above is that I was calling setupTestApplication in a loop in my head, but it's called once.
	// The dispatcher *is* persistent.

	// To check sequence, we need to know which mock server corresponds to which path in `expectedCallOrderPaths`.
	// mockServers[1] -> /ep1
	// mockServers[0] -> /ep0
	// mockServers[2] -> /ep2

	// Let's refine the assertion for sequence by checking the recorded paths on EACH server.
	// This still doesn't give the global sequence.
	// The best way is to have the mock server also record a timestamp or an incrementing call ID,
	// or pass a channel that records the call sequence.

	// Simpler for now: Make requests one by one and check the state of *all* mock servers after each call.
	// This is too complex. The current check (counts per server) is a good start.
	// If the counts are right for the top 3, and zero for others, it's a strong indication.
	// The unit tests for dispatcher already verify strict ordering.
	// For integration, verifying correct servers are chosen is key.

	// Let's try to verify the *first few calls* to establish sequence if possible.
	// After 1st call: mockServers[1] should have 1 call to /ep1. Others 0.
	// After 2nd call: mockServers[0] should have 1 call to /ep0. mockServers[1] still 1. Others 0.
	// After 3rd call: mockServers[2] should have 1 call to /ep2. mockServers[0] 1, mockServers[1] 1. Others 0.
	// This can be done by resetting mock server paths before each call and checking only the one that should be called.

	// Resetting all mocks for a cleaner individual check approach
	for _, s := range mockServers {
		s.ResetCalls()
	}

	for i, expectedPath := range expectedCallOrderPaths {
		// Find which server corresponds to expectedPath
		var targetServer *mockDownstreamServer
		var serverOriginalIndex int
		for j, srv := range mockServers {
			// Assuming endpointURL was set like srv.URL + "/epJ_modified"
			// My current endpoint URLs are mockServers[0].URL + "/ep0", etc.
			// So if expectedPath is "/ep1", it's mockServers[1].
			// This is a bit fragile if URL structure changes.
			// Let's find by matching URL.
			// Example: expectedPath = "/ep1"
			// mockServers[0].URL = http://127.0.0.1:xxxx
			// endpoints[0].EndpointURL = http://127.0.0.1:xxxx/ep0
			// endpoints[1].EndpointURL = http://127.0.0.1:yyyy/ep1  -- this is wrong, mockServers share httptest server instances.
			// No, each mockServer is its own httptest.Server. So URLs are distinct.

			// Find the server whose URL + expectedPath matches an endpoint's URL
			// This is also complex. Let's use the `expectedServersCalled` slice.
			targetServer = expectedServersCalled[i]

			// Determine original index for logging, if needed
			for k, ms := range mockServers {
				if ms == targetServer {
					serverOriginalIndex = k
					break
				}
			}


		req, err := http.NewRequest("POST", fmt.Sprintf("%s/relay?chain_id=%d&api_key=%s", appServer.URL, testChainID, testAPIKey), bytes.NewBuffer(requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // Close body immediately

		require.Equal(t, http.StatusOK, resp.StatusCode, "Request %d to %s failed. Body: %s", i+1, expectedPath, string(bodyBytes))

		// Check that *only* the targetServer was called for *this* request
		for k, s := range mockServers {
			if s == targetServer {
				assert.Len(t, s.GetCalledPaths(), 1, "Server %d (path %s) should have 1 call for request %d, has %d. Paths: %v", serverOriginalIndex, expectedPath, i+1, len(s.GetCalledPaths()), s.GetCalledPaths())
				if len(s.GetCalledPaths()) > 0 {
					assert.Equal(t, expectedPath, s.GetCalledPaths()[0], "Server %d called with wrong path for request %d", serverOriginalIndex, i+1)
				}
			} else {
				assert.Len(t, s.GetCalledPaths(), 0, "Server %d should NOT have been called for request %d (target %s), but was. Paths: %v", k, i+1, expectedPath, s.GetCalledPaths())
			}
		}
		targetServer.ResetCalls() // Reset calls for the server that was just called
	}
}

func TestIntegrationDispatcher_RoundRobinLessThanTopN(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	testDB := getTestDB(t)
	defer testDB.Close()

	clearTable(t, testDB, "rpc_endpoints")

	mockServers := make([]*mockDownstreamServer, 2)
	for i := range mockServers {
		mockServers[i] = newMockDownstreamServer(t, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"ok_server_%d"}`, i))
		defer mockServers[i].Close()
	}

	// Endpoints: s1 (10ms), s0 (20ms) -> sorted: s1, s0
	endpoints := []models.RpcEndpoint{
		{ChainID: testChainID, EndpointURL: mockServers[0].URL + "/ep0", ResponseTimeMs: intPtr(20), IsActive: true, Geozone: "IND"},
		{ChainID: testChainID, EndpointURL: mockServers[1].URL + "/ep1", ResponseTimeMs: intPtr(10), IsActive: true, Geozone: "IND"},
	}
	for _, ep := range endpoints {
		insertRPCEndpoint(t, testDB, ep)
	}

	appServer := setupTestApplication(t, testDB)
	defer appServer.Close()

	requestBody := []byte(`{"jsonrpc":"2.0","method":"test_method","params":[],"id":1}`)

	expectedCallOrderPaths := []string{"/ep1", "/ep0", "/ep1", "/ep0"}
	expectedServersCalled := []*mockDownstreamServer{mockServers[1], mockServers[0], mockServers[1], mockServers[0]}

	for i, expectedPath := range expectedCallOrderPaths {
		targetServer := expectedServersCalled[i]
		var serverOriginalIndex int
		for k, ms := range mockServers { if ms == targetServer { serverOriginalIndex = k; break } }

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/relay?chain_id=%d&api_key=%s", appServer.URL, testChainID, testAPIKey), bytes.NewBuffer(requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Request %d to %s failed. Body: %s", i+1, expectedPath, string(bodyBytes))

		for k, s := range mockServers {
			if s == targetServer {
				assert.Len(t, s.GetCalledPaths(), 1, "Server %d (path %s) should have 1 call for request %d. Paths: %v", serverOriginalIndex, expectedPath, i+1, s.GetCalledPaths())
				if len(s.GetCalledPaths()) > 0 {
					assert.Equal(t, expectedPath, s.GetCalledPaths()[0])
				}
			} else {
				assert.Len(t, s.GetCalledPaths(), 0, "Server %d should NOT have been called for request %d (target %s). Paths: %v", k, i+1, expectedPath, s.GetCalledPaths())
			}
		}
		targetServer.ResetCalls()
	}
}

func TestIntegrationDispatcher_NoHealthyEndpoints(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	testDB := getTestDB(t)
	defer testDB.Close()

	clearTable(t, testDB, "rpc_endpoints")

	mockServer := newMockDownstreamServer(t, `{"jsonrpc":"2.0","id":1,"result":"ok"}`) // Will not be called
	defer mockServer.Close()

	// Endpoints with nil ResponseTimeMs
	endpoints := []models.RpcEndpoint{
		{ChainID: testChainID, EndpointURL: mockServer.URL + "/ep0", ResponseTimeMs: nil, IsActive: true, Geozone: "IND"},
		{ChainID: testChainID, EndpointURL: mockServer.URL + "/ep1", ResponseTimeMs: nil, IsActive: true, Geozone: "IND"},
	}
	for _, ep := range endpoints {
		insertRPCEndpoint(t, testDB, ep)
	}
	// Also test with no active endpoints at all (empty table)
	insertRPCEndpoint(t, testDB, models.RpcEndpoint{ChainID: testChainID, EndpointURL: mockServer.URL + "/ep2", ResponseTimeMs: intPtr(10), IsActive: false, Geozone: "IND"})


	appServer := setupTestApplication(t, testDB)
	defer appServer.Close()

	requestBody := []byte(`{"jsonrpc":"2.0","method":"test_method","params":[],"id":1}`)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/relay?chain_id=%d&api_key=%s", appServer.URL, testChainID, testAPIKey), bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Expect an error from the application, like 500 or specific error code for "no endpoints"
	// The relayService.Relay returns an error string "no active endpoints available for the requested chain"
	// which the handler turns into: {"error": "Failed to relay request: no active endpoints available for the requested chain"}
	// with status http.StatusInternalServerError.
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var errorResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse["error"], rpc.ErrNoEndpoints.Error())

	assert.Len(t, mockServer.GetCalledPaths(), 0, "Mock server should not be called")

	// Scenario 2: No endpoints in DB for the chain at all
	clearTable(t, testDB, "rpc_endpoints")
	appServer2 := setupTestApplication(t, testDB) // Re-setup with empty table
	defer appServer2.Close()

	req2, err := http.NewRequest("POST", fmt.Sprintf("%s/relay?chain_id=%d&api_key=%s", appServer2.URL, testChainID, testAPIKey), bytes.NewBuffer(requestBody))
	require.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp2.StatusCode)
	err = json.NewDecoder(resp2.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse["error"], rpc.ErrNoEndpoints.Error())
}


func TestIntegrationDispatcher_SingleHealthyEndpoint(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test")
	}

	testDB := getTestDB(t)
	defer testDB.Close()

	clearTable(t, testDB, "rpc_endpoints")

	mockServers := make([]*mockDownstreamServer, 3)
	for i := range mockServers {
		mockServers[i] = newMockDownstreamServer(t, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"ok_server_%d"}`, i))
		defer mockServers[i].Close()
	}

	// Endpoints: s0 (nil), s1 (20ms, active), s2 (nil)
	endpoints := []models.RpcEndpoint{
		{ChainID: testChainID, EndpointURL: mockServers[0].URL + "/ep0", ResponseTimeMs: nil, IsActive: true, Geozone: "IND"},
		{ChainID: testChainID, EndpointURL: mockServers[1].URL + "/ep1", ResponseTimeMs: intPtr(20), IsActive: true, Geozone: "IND"}, // The only healthy one
		{ChainID: testChainID, EndpointURL: mockServers[2].URL + "/ep2", ResponseTimeMs: nil, IsActive: true, Geozone: "IND"},
		{ChainID: testChainID, EndpointURL: mockServers[0].URL + "/ep3", ResponseTimeMs: intPtr(10), IsActive: false, Geozone: "IND"}, // Healthy but inactive
	}
	for _, ep := range endpoints {
		insertRPCEndpoint(t, testDB, ep)
	}

	appServer := setupTestApplication(t, testDB)
	defer appServer.Close()

	requestBody := []byte(`{"jsonrpc":"2.0","method":"test_method","params":[],"id":1}`)

	// This single healthy server (mockServers[1] /ep1) should be called every time.
	expectedPath := "/ep1"
	targetServer := mockServers[1]

	for i := 0; i < 3; i++ { // Call 3 times
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/relay?chain_id=%d&api_key=%s", appServer.URL, testChainID, testAPIKey), bytes.NewBuffer(requestBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Request %d to %s failed. Body: %s", i+1, expectedPath, string(bodyBytes))

		for k, s := range mockServers {
			if s == targetServer {
				// This server is called repeatedly, so its call count will increment.
				// We check that it was called, and the path is correct for the *current* call.
				currentPaths := s.GetCalledPaths()
				require.Len(t, currentPaths, i+1, "Server 1 (healthy) should have %d calls after request %d. Paths: %v", i+1, i+1, currentPaths)
				assert.Equal(t, expectedPath, currentPaths[i]) // Check the path of the latest call
			} else {
				assert.Len(t, s.GetCalledPaths(), 0, "Server %d should NOT have been called at all. Paths: %v", k, s.GetCalledPaths())
			}
		}
		// Don't reset targetServer calls here, as we want to see accumulation for this specific test.
	}
	// Final check of counts
	assert.Len(t, mockServers[1].GetCalledPaths(), 3, "Healthy server should have 3 calls in total.")
	assert.Len(t, mockServers[0].GetCalledPaths(), 0)
	assert.Len(t, mockServers[2].GetCalledPaths(), 0)
}
