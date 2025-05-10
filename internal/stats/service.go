package stats

import (
	"database/sql"
	"fmt"
	"time"
)

// Service provides functionality for retrieving usage statistics
// @Description Manages and retrieves usage statistics for RPC requests
type Service struct {
	db *sql.DB
}

// NewService creates a new stats service with the provided database connection
func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

// LogStats represents a single statistics entry
// @Description Statistics entry for a specific time period
type LogStats struct {
	Period     string `json:"period" example:"2024-03-20T10:00:00Z"`
	Count      int    `json:"count" example:"150"`
	ChainID    int    `json:"chain_id" example:"1"`
	Endpoint   string `json:"endpoint,omitempty" example:"eth_blockNumber"`
	APIKey     string `json:"api_key,omitempty" example:"your-api-key-here"`
	EndpointID string `json:"endpoint_id,omitempty" example:"endpoint-123"`
}

// StatsResponse is the response for statistics queries
// @Description Response containing statistics data
type StatsResponse struct {
	Stats []LogStats `json:"stats"`
}

// StatsFilter is used to filter statistics queries
// @Description Filter parameters for statistics queries
type StatsFilter struct {
	ChainID    int       `json:"chain_id,omitempty" example:"1"`
	APIKey     string    `json:"api_key,omitempty" example:"your-api-key-here"`
	StartDate  time.Time `json:"start_date,omitempty" example:"2024-03-20T00:00:00Z"`
	EndDate    time.Time `json:"end_date,omitempty" example:"2024-03-21T00:00:00Z"`
	Interval   string    `json:"interval,omitempty" example:"1hour"` // "1hour", "4hour", "6hour", "12hour", "24hour"
	Endpoint   string    `json:"endpoint,omitempty" example:"eth_blockNumber"`
	EndpointID string    `json:"endpoint_id,omitempty" example:"endpoint-123"`
}

// GetStats retrieves aggregated statistics based on the provided filter
// @Summary Get statistics
// @Description Retrieves aggregated statistics based on filter parameters
// @Tags Stats
// @Accept json
// @Produce json
// @Param filter body StatsFilter true "Statistics filter parameters"
// @Success 200 {object} StatsResponse "Statistics data"
// @Failure 400 {object} ErrorResponse "Invalid filter parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/stats [post]
func (s *Service) GetStats(filter StatsFilter) ([]LogStats, error) {
	// Base query
	baseQuery := `
		SELECT 
			COUNT(*) as request_count,
			chain_id,
			%s
			api_key,
			endpoint
		FROM logs
		WHERE 1=1
	`

	// Initialize params array and next param index
	params := []interface{}{}
	paramIndex := 1

	// Add time interval formatter based on the interval parameter
	var timeFormatter string
	switch filter.Interval {
	case "1hour":
		timeFormatter = "date_trunc('hour', created_at) as period,"
	case "4hour":
		timeFormatter = "date_trunc('hour', created_at) - INTERVAL '1 minute' * EXTRACT(MINUTE FROM created_at) - INTERVAL '1 second' * EXTRACT(SECOND FROM created_at) - INTERVAL '1 hour' * (EXTRACT(HOUR FROM created_at) %% 4) as period,"
	case "6hour":
		timeFormatter = "date_trunc('hour', created_at) - INTERVAL '1 minute' * EXTRACT(MINUTE FROM created_at) - INTERVAL '1 second' * EXTRACT(SECOND FROM created_at) - INTERVAL '1 hour' * (EXTRACT(HOUR FROM created_at) %% 6) as period,"
	case "12hour":
		timeFormatter = "date_trunc('hour', created_at) - INTERVAL '1 minute' * EXTRACT(MINUTE FROM created_at) - INTERVAL '1 second' * EXTRACT(SECOND FROM created_at) - INTERVAL '1 hour' * (EXTRACT(HOUR FROM created_at) %% 12) as period,"
	case "24hour":
		timeFormatter = "date_trunc('day', created_at) as period,"
	default:
		timeFormatter = "date_trunc('hour', created_at) as period,"
	}

	// Format query with time interval
	query := fmt.Sprintf(baseQuery, timeFormatter)

	// Add conditions based on filter
	if filter.ChainID > 0 {
		query += fmt.Sprintf(" AND chain_id = $%d", paramIndex)
		params = append(params, filter.ChainID)
		paramIndex++
	}

	if filter.APIKey != "" {
		query += fmt.Sprintf(" AND api_key = $%d", paramIndex)
		params = append(params, filter.APIKey)
		paramIndex++
	}

	if filter.Endpoint != "" {
		query += fmt.Sprintf(" AND endpoint = $%d", paramIndex)
		params = append(params, filter.Endpoint)
		paramIndex++
	}

	if !filter.StartDate.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", paramIndex)
		params = append(params, filter.StartDate)
		paramIndex++
	}

	if !filter.EndDate.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", paramIndex)
		params = append(params, filter.EndDate)
		paramIndex++
	}

	// Group by
	query += `
		GROUP BY 
			period,
			chain_id,
			api_key,
			endpoint
		ORDER BY 
			period ASC
	`

	// Execute query
	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Parse results
	var stats []LogStats
	for rows.Next() {
		var stat LogStats
		var period time.Time

		if err := rows.Scan(&stat.Count, &stat.ChainID, &period, &stat.APIKey, &stat.Endpoint); err != nil {
			return nil, err
		}

		stat.Period = period.Format(time.RFC3339)
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetAPIKeyStats retrieves statistics for a specific API key
// @Summary Get API key statistics
// @Description Retrieves statistics for a specific API key on a specific chain
// @Tags Stats
// @Accept json
// @Produce json
// @Param apiKey path string true "API Key"
// @Param chainID path int true "Chain ID"
// @Param interval query string false "Time interval (1hour, 4hour, 6hour, 12hour, 24hour)" default(1hour)
// @Success 200 {object} StatsResponse "API key statistics"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/stats/api-key/{apiKey}/{chainID} [get]
func (s *Service) GetAPIKeyStats(apiKey string, chainID int, interval string) ([]LogStats, error) {
	filter := StatsFilter{
		APIKey:   apiKey,
		ChainID:  chainID,
		Interval: interval,
	}
	return s.GetStats(filter)
}

// GetChainStats retrieves statistics for a specific chain
// @Summary Get chain statistics
// @Description Retrieves statistics for a specific chain
// @Tags Stats
// @Accept json
// @Produce json
// @Param chainID path int true "Chain ID"
// @Param interval query string false "Time interval (1hour, 4hour, 6hour, 12hour, 24hour)" default(1hour)
// @Success 200 {object} StatsResponse "Chain statistics"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/stats/chain/{chainID} [get]
func (s *Service) GetChainStats(chainID int, interval string) ([]LogStats, error) {
	filter := StatsFilter{
		ChainID:  chainID,
		Interval: interval,
	}
	return s.GetStats(filter)
}

// GetEndpointStats retrieves statistics for a specific endpoint
// @Summary Get endpoint statistics
// @Description Retrieves statistics for a specific endpoint on a specific chain
// @Tags Stats
// @Accept json
// @Produce json
// @Param endpoint path string true "Endpoint"
// @Param chainID path int true "Chain ID"
// @Param interval query string false "Time interval (1hour, 4hour, 6hour, 12hour, 24hour)" default(1hour)
// @Success 200 {object} StatsResponse "Endpoint statistics"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/stats/endpoint/{endpoint}/{chainID} [get]
func (s *Service) GetEndpointStats(endpoint string, chainID int, interval string) ([]LogStats, error) {
	filter := StatsFilter{
		Endpoint: endpoint,
		ChainID:  chainID,
		Interval: interval,
	}
	return s.GetStats(filter)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid parameters"`
}
