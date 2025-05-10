package stats

import (
	"database/sql"
	"fmt"
	"time"
)

// Service provides functionality for retrieving usage statistics
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
type LogStats struct {
	Period     string `json:"period"`
	Count      int    `json:"count"`
	ChainID    int    `json:"chain_id"`
	Endpoint   string `json:"endpoint,omitempty"`
	APIKey     string `json:"api_key,omitempty"`
	EndpointID string `json:"endpoint_id,omitempty"`
}

// StatsResponse is the response for statistics queries
type StatsResponse struct {
	Stats []LogStats `json:"stats"`
}

// StatsFilter is used to filter statistics queries
type StatsFilter struct {
	ChainID    int
	APIKey     string
	StartDate  time.Time
	EndDate    time.Time
	Interval   string // "1hour", "4hour", "6hour", "12hour", "24hour"
	Endpoint   string
	EndpointID string
}

// GetStats retrieves aggregated statistics based on the provided filter
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
func (s *Service) GetAPIKeyStats(apiKey string, chainID int, interval string) ([]LogStats, error) {
	filter := StatsFilter{
		APIKey:   apiKey,
		ChainID:  chainID,
		Interval: interval,
	}
	return s.GetStats(filter)
}

// GetChainStats retrieves statistics for a specific chain
func (s *Service) GetChainStats(chainID int, interval string) ([]LogStats, error) {
	filter := StatsFilter{
		ChainID:  chainID,
		Interval: interval,
	}
	return s.GetStats(filter)
}

// GetEndpointStats retrieves statistics for a specific endpoint
func (s *Service) GetEndpointStats(endpoint string, chainID int, interval string) ([]LogStats, error) {
	filter := StatsFilter{
		Endpoint: endpoint,
		ChainID:  chainID,
		Interval: interval,
	}
	return s.GetStats(filter)
}
