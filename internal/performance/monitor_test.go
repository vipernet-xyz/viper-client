package performance

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chainbound/apollo/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceMonitor_pingServicer(t *testing.T) {
	dummyEndpoint := models.RpcEndpoint{ID: 1, EndpointURL: ""} // URL will be set by mock server

	t.Run("Successful ping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
		}))
		defer server.Close()

		dummyEndpoint.EndpointURL = server.URL
		pm := NewPerformanceMonitor(nil, server.Client())

		responseTimeMs, httpStatusCode, err := pm.pingServicer(dummyEndpoint, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`, 5*time.Second)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Greater(t, responseTimeMs, int64(0))
	})

	t.Run("Server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, `Internal Server Error`)
		}))
		defer server.Close()

		dummyEndpoint.EndpointURL = server.URL
		pm := NewPerformanceMonitor(nil, server.Client())

		_, httpStatusCode, err := pm.pingServicer(dummyEndpoint, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`, 5*time.Second)

		assert.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, httpStatusCode)
		assert.Contains(t, err.Error(), "HTTP status 500")
	})

	t.Run("Request timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Sleep longer than timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		dummyEndpoint.EndpointURL = server.URL
		// Use a new client for this test to ensure timeout is not shared
		pm := NewPerformanceMonitor(nil, &http.Client{})

		_, _, err := pm.pingServicer(dummyEndpoint, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`, 50*time.Millisecond) // 50ms timeout

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request timed out")
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Non-JSON response with 200 OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "This is not JSON, but it's okay.")
		}))
		defer server.Close()

		dummyEndpoint.EndpointURL = server.URL
		pm := NewPerformanceMonitor(nil, server.Client())

		responseTimeMs, httpStatusCode, err := pm.pingServicer(dummyEndpoint, `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`, 5*time.Second)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpStatusCode)
		assert.Greater(t, responseTimeMs, int64(0))
	})
}

func TestPerformanceMonitor_fetchServicersToPing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pm := NewPerformanceMonitor(db, nil)

	t.Run("Active servicers found for specific chainID", func(t *testing.T) {
		chainID := 1
		expectedEndpoints := []models.RpcEndpoint{
			{ID: 1, ChainID: 1, EndpointURL: "http://node1.com", Provider: "ProviderA", IsActive: true, IsHealthy: true, Geozone: "IND", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: 2, ChainID: 1, EndpointURL: "http://node2.com", Provider: "ProviderB", IsActive: true, IsHealthy: false, Geozone: "IND", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		rows := sqlmock.NewRows([]string{"id", "chain_id", "endpoint_url", "provider", "is_active", "is_healthy", "health_check_timestamp", "health_status", "geozone", "created_at", "updated_at"}).
			AddRow(expectedEndpoints[0].ID, expectedEndpoints[0].ChainID, expectedEndpoints[0].EndpointURL, expectedEndpoints[0].Provider, expectedEndpoints[0].IsActive, expectedEndpoints[0].IsHealthy, sql.NullTime{Time: time.Now(), Valid: true}, sql.NullString{String: "OK", Valid: true}, expectedEndpoints[0].Geozone, expectedEndpoints[0].CreatedAt, expectedEndpoints[0].UpdatedAt).
			AddRow(expectedEndpoints[1].ID, expectedEndpoints[1].ChainID, expectedEndpoints[1].EndpointURL, expectedEndpoints[1].Provider, expectedEndpoints[1].IsActive, expectedEndpoints[1].IsHealthy, sql.NullTime{}, sql.NullString{}, expectedEndpoints[1].Geozone, expectedEndpoints[1].CreatedAt, expectedEndpoints[1].UpdatedAt)

		// Note: The query in fetchServicersToPing dynamically adds the chain_id placeholder.
		// We use regexp.QuoteMeta to escape the query string for the mock.
		expectedSQL := `
		SELECT
			id,
			chain_id,
			endpoint_url,
			provider,
			is_active,
			is_healthy,
			health_check_timestamp,
			health_status,
			geozone,
			created_at,
			updated_at
		FROM rpc_endpoints
		WHERE is_active = true AND geozone = 'IND'
	 AND chain_id = $1`
		mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).WithArgs(chainID).WillReturnRows(rows)

		endpoints, err := pm.fetchServicersToPing(chainID)

		assert.NoError(t, err)
		assert.Len(t, endpoints, 2)
		// Detailed comparison if needed, ensuring timestamps are handled (they might differ slightly)
		assert.Equal(t, expectedEndpoints[0].EndpointURL, endpoints[0].EndpointURL)
		assert.Equal(t, expectedEndpoints[1].EndpointURL, endpoints[1].EndpointURL)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("No active servicers found for chainID", func(t *testing.T) {
		chainID := 2
		expectedSQL := `
		SELECT
			id,
			chain_id,
			endpoint_url,
			provider,
			is_active,
			is_healthy,
			health_check_timestamp,
			health_status,
			geozone,
			created_at,
			updated_at
		FROM rpc_endpoints
		WHERE is_active = true AND geozone = 'IND'
	 AND chain_id = $1`
		mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).WithArgs(chainID).WillReturnRows(sqlmock.NewRows([]string{}))

		endpoints, err := pm.fetchServicersToPing(chainID)

		assert.NoError(t, err)
		assert.Empty(t, endpoints)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Fetching for all chain IDs", func(t *testing.T) {
		expectedEndpoints := []models.RpcEndpoint{
			{ID: 3, ChainID: 5, EndpointURL: "http://node3.com", Provider: "ProviderC", IsActive: true, IsHealthy: true, Geozone: "IND"},
			{ID: 4, ChainID: 10, EndpointURL: "http://node4.com", Provider: "ProviderD", IsActive: true, IsHealthy: true, Geozone: "IND"},
		}

		rows := sqlmock.NewRows([]string{"id", "chain_id", "endpoint_url", "provider", "is_active", "is_healthy", "health_check_timestamp", "health_status", "geozone", "created_at", "updated_at"}).
			AddRow(expectedEndpoints[0].ID, expectedEndpoints[0].ChainID, expectedEndpoints[0].EndpointURL, expectedEndpoints[0].Provider, true, true, nil, nil, "IND", time.Now(), time.Now()).
			AddRow(expectedEndpoints[1].ID, expectedEndpoints[1].ChainID, expectedEndpoints[1].EndpointURL, expectedEndpoints[1].Provider, true, true, nil, nil, "IND", time.Now(), time.Now())

		expectedSQL := `
		SELECT
			id,
			chain_id,
			endpoint_url,
			provider,
			is_active,
			is_healthy,
			health_check_timestamp,
			health_status,
			geozone,
			created_at,
			updated_at
		FROM rpc_endpoints
		WHERE is_active = true AND geozone = 'IND'
	` // No chain_id filter
		mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).WillReturnRows(rows)

		endpoints, err := pm.fetchServicersToPing(0) // chainID <= 0 means all chains

		assert.NoError(t, err)
		assert.Len(t, endpoints, 2)
		assert.Equal(t, expectedEndpoints[0].EndpointURL, endpoints[0].EndpointURL)
		assert.Equal(t, expectedEndpoints[1].EndpointURL, endpoints[1].EndpointURL)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		chainID := 3
		dbErr := errors.New("DB error")
		expectedSQL := `
		SELECT
			id,
			chain_id,
			endpoint_url,
			provider,
			is_active,
			is_healthy,
			health_check_timestamp,
			health_status,
			geozone,
			created_at,
			updated_at
		FROM rpc_endpoints
		WHERE is_active = true AND geozone = 'IND'
	 AND chain_id = $1`
		mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).WithArgs(chainID).WillReturnError(dbErr)

		_, err := pm.fetchServicersToPing(chainID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), dbErr.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPerformanceMonitor_recordPerformanceMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	pm := NewPerformanceMonitor(db, nil)

	t.Run("Successful insert", func(t *testing.T) {
		endpointID := 1
		pingTime := time.Now()
		responseTime := int64(120)
		httpStatus := 200
		errorMsg := ""

		expectedSQL := `
		INSERT INTO endpoint_performance_metrics
		(rpc_endpoint_id, ping_timestamp, response_time_ms, http_status_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
		mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
			WithArgs(endpointID, pingTime, responseTime, httpStatus, errorMsg).
			WillReturnResult(sqlmock.NewResult(1, 1)) // 1 new id, 1 row affected

		err := pm.recordPerformanceMetric(endpointID, pingTime, responseTime, httpStatus, errorMsg)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Successful insert with error message", func(t *testing.T) {
		endpointID := 2
		pingTime := time.Now()
		responseTime := int64(5000)
		httpStatus := 0 // e.g. timeout
		errorMsg := "context deadline exceeded"

		expectedSQL := `
		INSERT INTO endpoint_performance_metrics
		(rpc_endpoint_id, ping_timestamp, response_time_ms, http_status_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
		mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
			WithArgs(endpointID, pingTime, responseTime, httpStatus, errorMsg).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := pm.recordPerformanceMetric(endpointID, pingTime, responseTime, httpStatus, errorMsg)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error on insert", func(t *testing.T) {
		endpointID := 3
		pingTime := time.Now()
		responseTime := int64(150)
		httpStatus := 503
		errorMsg := "Service Unavailable"
		dbErr := errors.New("DB insert error")

		expectedSQL := `
		INSERT INTO endpoint_performance_metrics
		(rpc_endpoint_id, ping_timestamp, response_time_ms, http_status_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
		mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
			WithArgs(endpointID, pingTime, responseTime, httpStatus, errorMsg).
			WillReturnError(dbErr)

		err := pm.recordPerformanceMetric(endpointID, pingTime, responseTime, httpStatus, errorMsg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), dbErr.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error message truncation", func(t *testing.T) {
		endpointID := 4
		pingTime := time.Now()
		responseTime := int64(200)
		httpStatus := 400
		// Create a very long error message
		longErrorMsg := string(make([]byte, 70000)) 
		truncatedErrorMsg := longErrorMsg[:65530] + "..."


		expectedSQL := `
		INSERT INTO endpoint_performance_metrics
		(rpc_endpoint_id, ping_timestamp, response_time_ms, http_status_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
		mock.ExpectExec(regexp.QuoteMeta(expectedSQL)).
			WithArgs(endpointID, pingTime, responseTime, httpStatus, truncatedErrorMsg).
			WillReturnResult(sqlmock.NewResult(1,1))

		err := pm.recordPerformanceMetric(endpointID, pingTime, responseTime, httpStatus, longErrorMsg)
		
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())

	})
}

// Note: Ensure your models.RpcEndpoint has all the fields being scanned in fetchServicersToPing
// including Provider, HealthCheckTimestamp, HealthStatus if they are part of the SELECT query.
// The provided monitor.go uses sql.NullString and sql.NullTime for these, so the mock rows should reflect that.
