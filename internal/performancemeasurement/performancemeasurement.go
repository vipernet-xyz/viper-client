package performancemeasurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/illegalcall/viper-client/internal/models"
	"github.com/illegalcall/viper-client/internal/utils"
)

// PerformanceMeasurerService handles measuring the performance of RPC endpoints.
type PerformanceMeasurerService struct {
	config *utils.MonitoringConfig
	db     *sql.DB
}

// NewPerformanceMeasurerService creates a new instance of PerformanceMeasurerService.
func NewPerformanceMeasurerService(cfg *utils.MonitoringConfig, dbConn *sql.DB) *PerformanceMeasurerService {
	return &PerformanceMeasurerService{
		config: cfg,
		db:     dbConn,
	}
}

// MeasurePerformance queries all RPC endpoints and updates their health status.
func (s *PerformanceMeasurerService) MeasurePerformance(ctx context.Context) error {
	log.Println("[PerformanceMeasurement] Starting performance measurement cycle...")

	endpoints, err := s.getAllRpcEndpoints(ctx)
	if err != nil {
		log.Printf("[PerformanceMeasurement] Error fetching RPC endpoints: %v", err)
		return err
	}

	if len(endpoints) == 0 {
		log.Println("[PerformanceMeasurement] No RPC endpoints found to measure.")
		return nil
	}

	log.Printf("[PerformanceMeasurement] Measuring performance for %d RPC endpoints...", len(endpoints))

	var wg sync.WaitGroup
	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, s.config.MaxConcurrentPings)

	for _, endpoint := range endpoints {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a slot

		go func(ep models.RpcEndpoint) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the slot

			s.pingAndUpdateEndpoint(ctx, ep)
		}(endpoint)
	}

	wg.Wait()
	log.Println("[PerformanceMeasurement] Performance measurement cycle completed.")
	return nil
}

func (s *PerformanceMeasurerService) getAllRpcEndpoints(ctx context.Context) ([]models.RpcEndpoint, error) {
	query := `SELECT id, chain_id, endpoint_url, public_key, response_time_ms, last_ping_timestamp, servicer_type, health_status, is_active FROM rpc_endpoints`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []models.RpcEndpoint
	for rows.Next() {
		var ep models.RpcEndpoint
		// Make sure to scan into pointers for nullable fields if they are defined as pointers in the struct
		err := rows.Scan(
			&ep.ID,
			&ep.ChainID,
			&ep.EndpointURL,
			&ep.PublicKey,         // Assumes PublicKey in RpcEndpoint struct is *string
			&ep.ResponseTimeMs,    // Assumes ResponseTimeMs in RpcEndpoint struct is *int
			&ep.LastPingTimestamp, // Assumes LastPingTimestamp in RpcEndpoint struct is *time.Time
			&ep.ServicerType,
			&ep.HealthStatus,      // Assumes HealthStatus is string (for JSON)
			&ep.IsActive,
		)
		if err != nil {
			log.Printf("[PerformanceMeasurement] Error scanning RPC endpoint row: %v", err)
			continue
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, rows.Err()
}

// HealthStatusPayload defines the structure for health_status JSON
type HealthStatusPayload struct {
	Status              string              `json:"status"` // e.g., "healthy", "unhealthy", "degraded"
	ResponseTimeMs      *int                `json:"response_time_ms,omitempty"`
	Error               *string             `json:"error,omitempty"`
	ErrorRate           float64             `json:"error_rate"`           // Placeholder, implement calculation if needed
	LastSuccess         *time.Time          `json:"last_success,omitempty"`
	LastFailure         *time.Time          `json:"last_failure,omitempty"`
	ConsecutiveFailures int                 `json:"consecutive_failures"`
	PerformanceHistory  []PerformanceSample `json:"performance_history,omitempty"` // Optional: for trend analysis
}

// PerformanceSample stores a single performance data point for history
type PerformanceSample struct {
	Timestamp      time.Time `json:"timestamp"`
	ResponseTimeMs int       `json:"response_time_ms"`
	Error          *string   `json:"error,omitempty"`
}

func (s *PerformanceMeasurerService) pingAndUpdateEndpoint(ctx context.Context, endpoint models.RpcEndpoint) {
	var responseTimeMs int
	var currentStatus string
	var specificError *string
	var isActive bool

	pingTimeout, err := time.ParseDuration(s.config.PingTimeout)
	if err != nil {
		log.Printf("[PerformanceMeasurement] Error parsing ping timeout '%s', using default 5s: %v", s.config.PingTimeout, err)
		pingTimeout = 5 * time.Second
	}

	checkURL := endpoint.EndpointURL

	var finalErr error
	for i := 0; i < s.config.PingRetries; i++ {
		startTime := time.Now()
		httpClient := http.Client{Timeout: pingTimeout}
		req, _ := http.NewRequestWithContext(ctx, "GET", checkURL, nil) // Use GET for health check
		resp, err := httpClient.Do(req)
		duration := time.Since(startTime)

		if err != nil {
			errMsg := err.Error()
			specificError = &errMsg
			currentStatus = "unhealthy"
			isActive = false
			finalErr = err
			log.Printf("[PerformanceMeasurement] Attempt %d: Ping failed for %s (ID: %d): %v", i+1, endpoint.EndpointURL, endpoint.ID, err)
			if i < s.config.PingRetries-1 {
				retryInterval, _ := time.ParseDuration(s.config.PingInterval)
				time.Sleep(retryInterval) // Wait before retrying
			}
			continue // Try next attempt
		}
		// It's important to close the response body, even if not used, to prevent resource leaks.
		// Deferring it here ensures it's closed after the loop iteration or if an early exit occurs.
		// However, since we might break early, ensure it's managed correctly.
		// A common pattern is to read and discard body if not needed, then close.
		// For now, just close.

		responseTimeMs = int(duration.Milliseconds())
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			currentStatus = "healthy"
			isActive = true
			specificError = nil
			finalErr = nil // Clear error on success
			// log.Printf("[PerformanceMeasurement] Ping successful for %s (ID: %d): %d ms, Status: %d", endpoint.EndpointURL, endpoint.ID, responseTimeMs, resp.StatusCode)
			resp.Body.Close() // Close body on success
			break             // Successful ping, exit retry loop
		} else {
			errMsg := fmt.Sprintf("HTTP status %d", resp.StatusCode)
			specificError = &errMsg
			currentStatus = "unhealthy" // Or "degraded" based on status code ranges
			isActive = false            // Consider if non-2xx means inactive or just unhealthy
			finalErr = fmt.Errorf(errMsg)
			log.Printf("[PerformanceMeasurement] Attempt %d: Ping unhealthy for %s (ID: %d): Status %d, Response %d ms", i+1, endpoint.EndpointURL, endpoint.ID, resp.StatusCode, responseTimeMs)
			resp.Body.Close() // Also close body on non-2xx responses
			if i < s.config.PingRetries-1 {
				retryInterval, _ := time.ParseDuration(s.config.PingInterval)
				time.Sleep(retryInterval) // Wait before retrying
			}
		}
	} // End of retry loop

	if finalErr != nil {
		// All retries failed
		log.Printf("[PerformanceMeasurement] All %d ping attempts failed for %s (ID: %d). Marking as unhealthy/inactive.", s.config.PingRetries, endpoint.EndpointURL, endpoint.ID)
	}

	now := time.Now()
	var newHealthStatusPayload HealthStatusPayload

	// Try to unmarshal existing health status
	var existingPayload HealthStatusPayload
	if endpoint.HealthStatus != "" { // HealthStatus is string in model, not *string
		if err := json.Unmarshal([]byte(endpoint.HealthStatus), &existingPayload); err != nil {
			log.Printf("[PerformanceMeasurement] Warning: Could not unmarshal existing health_status for endpoint %d: %v. Initializing new history.", endpoint.ID, err)
			existingPayload.PerformanceHistory = []PerformanceSample{} // Initialize if unmarshal fails
		}
	} else {
		existingPayload.PerformanceHistory = []PerformanceSample{} // Initialize if empty
	}

	newHealthStatusPayload = existingPayload // Copy existing data first

	newHealthStatusPayload.Status = currentStatus
	// Only set ResponseTimeMs if the ping was somewhat successful (got a response)
	if finalErr == nil || (finalErr != nil && responseTimeMs > 0) {
		newHealthStatusPayload.ResponseTimeMs = &responseTimeMs
	} else {
		newHealthStatusPayload.ResponseTimeMs = nil // Ensure it's null if ping failed outright
	}
	newHealthStatusPayload.Error = specificError

	if currentStatus == "healthy" {
		newHealthStatusPayload.LastSuccess = &now
		newHealthStatusPayload.ConsecutiveFailures = 0
	} else {
		newHealthStatusPayload.LastFailure = &now
		// If existingPayload.ConsecutiveFailures was not loaded, it defaults to 0.
		// So, incrementing here works for both first failure and subsequent ones.
		newHealthStatusPayload.ConsecutiveFailures = existingPayload.ConsecutiveFailures + 1
	}

	// Add to performance history (keep it bounded, e.g., last 10-20 samples)
	currentSample := PerformanceSample{Timestamp: now, ResponseTimeMs: responseTimeMs, Error: specificError}
	if finalErr != nil && responseTimeMs == 0 { // If there was a complete failure (e.g. timeout on first try), record 0 or a specific error code for time
		// currentSample.ResponseTimeMs = -1 // Or some indicator of complete failure
	}

	newHealthStatusPayload.PerformanceHistory = append(newHealthStatusPayload.PerformanceHistory, currentSample)
	maxHistoryLength := 20 // Configurable?
	if len(newHealthStatusPayload.PerformanceHistory) > maxHistoryLength {
		newHealthStatusPayload.PerformanceHistory = newHealthStatusPayload.PerformanceHistory[len(newHealthStatusPayload.PerformanceHistory)-maxHistoryLength:]
	}

	healthStatusJSON, err := json.Marshal(newHealthStatusPayload)
	if err != nil {
		log.Printf("[PerformanceMeasurement] Error marshalling health status JSON for endpoint %d: %v", endpoint.ID, err)
		healthStatusJSON = []byte(`{"status":"error_marshalling_status"}`)
	}

	updateQuery := `
		UPDATE rpc_endpoints
		SET response_time_ms = $1, last_ping_timestamp = $2, health_check_timestamp = $3, health_status = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`
	// Prepare parameters for DB update
	var dbResponseTimeMs *int
	if newHealthStatusPayload.ResponseTimeMs != nil {
		dbResponseTimeMs = newHealthStatusPayload.ResponseTimeMs
	}
	
	var dbLastPingTimestamp *time.Time
	if currentStatus == "healthy" { // Only update last_ping_timestamp on successful ping
		dbLastPingTimestamp = &now
	} else if endpoint.LastPingTimestamp != nil { // Preserve existing if current ping failed
		dbLastPingTimestamp = endpoint.LastPingTimestamp
	}


	_, err = s.db.ExecContext(ctx, updateQuery,
		dbResponseTimeMs,        // Nullable integer
		dbLastPingTimestamp,     // Nullable timestamp
		&now,                    // health_check_timestamp (always update)
		string(healthStatusJSON),
		isActive,
		now,                     // updated_at
		endpoint.ID,
	)
	if err != nil {
		log.Printf("[PerformanceMeasurement] Error updating RPC endpoint %d in DB: %v", endpoint.ID, err)
	}
}
