package performance

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"

	"github.com/chainbound/apollo/internal/models"
)

// PerformanceMonitor is responsible for monitoring the performance of RPC endpoints.
type PerformanceMonitor struct {
	db         *sql.DB
	httpClient *http.Client
}

// NewPerformanceMonitor creates a new PerformanceMonitor.
func NewPerformanceMonitor(db *sql.DB, httpClient *http.Client) *PerformanceMonitor {
	return &PerformanceMonitor{
		db:         db,
		httpClient: httpClient,
	}
}

// fetchServicersToPing fetches active RPC endpoints for a given chainID and geozone.
// If chainID is 0, it fetches for all chains.
func (pm *PerformanceMonitor) fetchServicersToPing(chainID int) ([]models.RpcEndpoint, error) {
	var query strings.Builder
	args := make([]interface{}, 0)

	query.WriteString(`
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
	`)

	if chainID > 0 {
		query.WriteString(" AND chain_id = $")
		args = append(args, chainID)
		query.WriteString(fmt.Sprintf("%d", len(args)))
	}

	rows, err := pm.db.Query(query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("querying rpc_endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []models.RpcEndpoint

	for rows.Next() {
		var ep models.RpcEndpoint
		// Handling nullable fields for provider, health_check_timestamp, and health_status
		var provider sql.NullString
		var healthCheckTimestamp sql.NullTime
		var healthStatus sql.NullString

		if err := rows.Scan(
			&ep.ID,
			&ep.ChainID,
			&ep.EndpointURL,
			&provider, // scan into sql.NullString
			&ep.IsActive,
			&ep.IsHealthy,
			&healthCheckTimestamp, // scan into sql.NullTime
			&healthStatus,         // scan into sql.NullString
			&ep.Geozone,
			&ep.CreatedAt,
			&ep.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning rpc_endpoint row: %w", err)
		}

		// Assign values from nullable types to the struct fields
		ep.Provider = provider.String
		if healthCheckTimestamp.Valid {
			ep.HealthCheckTimestamp = healthCheckTimestamp.Time
		} else {
			// Handle nil time appropriately, perhaps set to zero value or a specific indicator
			ep.HealthCheckTimestamp = time.Time{}
		}
		ep.HealthStatus = healthStatus.String

		endpoints = append(endpoints, ep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rpc_endpoint rows: %w", err)
	}

	return endpoints, nil
}

// pingServicer sends a JSON-RPC request to an endpoint and measures the response.
func (pm *PerformanceMonitor) pingServicer(endpoint models.RpcEndpoint, jsonRpcPayload string, timeout time.Duration) (responseTimeMs int64, httpStatusCode int, err error) {
	client := &http.Client{
		Timeout: timeout, // Apply timeout for this specific request
	}

	req, err := http.NewRequest("POST", endpoint.EndpointURL, bytes.NewBufferString(jsonRpcPayload))
	if err != nil {
		return 0, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := client.Do(req)
	responseTimeMs = time.Since(startTime).Milliseconds()

	if err != nil {
		// Check if the error is due to context deadline exceeded (timeout)
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return responseTimeMs, 0, fmt.Errorf("request timed out after %v: %w", timeout, err)
		}
		return responseTimeMs, 0, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	httpStatusCode = resp.StatusCode

	// Read and discard the response body to allow connection reuse
	_, ioErr := io.Copy(io.Discard, resp.Body)
	if ioErr != nil {
		logrus.WithFields(logrus.Fields{
			"endpoint_url": endpoint.EndpointURL,
			"error":        ioErr,
		}).Warn("Error discarding response body")
		// Do not return this error as the main error, as the request itself might have succeeded
	}


	if httpStatusCode < 200 || httpStatusCode >= 300 {
		// Attempt to read a snippet of the body for error context if status is not 2xx
		// This is a simplified error handling; in a real app, you might parse JSON error responses
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024)) // Limit reading to 1KB
        var errorMsg string
		if readErr == nil && len(bodyBytes) > 0 {
			errorMsg = string(bodyBytes)
		} else {
			errorMsg = "Failed to read error response body"
		}
		return responseTimeMs, httpStatusCode, fmt.Errorf("HTTP status %d: %s", httpStatusCode, errorMsg)
	}
	

	return responseTimeMs, httpStatusCode, nil
}

// recordPerformanceMetric inserts a performance metric into the database.
func (pm *PerformanceMonitor) recordPerformanceMetric(endpointID int, pingTimestamp time.Time, responseTimeMs int64, httpStatusCode int, errorMsg string) error {
	// Truncate errorMsg if it's too long for the TEXT column (e.g., 65535 bytes for TEXT in PostgreSQL)
	// This is a safeguard; actual TEXT types can often hold more, but it's good practice.
	const maxErrorMsgLength = 65530 // A bit less than typical max to be safe
	if len(errorMsg) > maxErrorMsgLength {
		errorMsg = errorMsg[:maxErrorMsgLength] + "..." // Indicate truncation
	}

	query := `
		INSERT INTO endpoint_performance_metrics
		(rpc_endpoint_id, ping_timestamp, response_time_ms, http_status_code, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err := pm.db.Exec(query, endpointID, pingTimestamp, responseTimeMs, httpStatusCode, errorMsg)
	if err != nil {
		return fmt.Errorf("inserting performance metric: %w", err)
	}
	return nil
}

// Helper function to convert interface{} to testcontainers.ContainerProvider
// This is a placeholder and might need adjustment based on actual usage context
func convertToContainerProvider(p interface{}) (testcontainers.ContainerProvider, error) {
	if provider, ok := p.(testcontainers.ContainerProvider); ok {
		return provider, nil
	}
	return nil, fmt.Errorf("cannot convert %T to testcontainers.ContainerProvider", p)
}

// Example of how to use the json.RawMessage for params if needed in the future
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"` // Use json.RawMessage for params
	ID      int             `json:"id"`
}
