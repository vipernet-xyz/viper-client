package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/illegalcall/viper-client/internal/models"
)

// PerformanceAnalyzerService provides methods for analyzing servicer performance.
type PerformanceAnalyzerService struct {
	db *sql.DB
}

// NewPerformanceAnalyzerService creates a new instance of PerformanceAnalyzerService.
func NewPerformanceAnalyzerService(dbConn *sql.DB) *PerformanceAnalyzerService {
	return &PerformanceAnalyzerService{
		db: dbConn,
	}
}

// RankedServicer extends RpcEndpoint with a rank.
type RankedServicer struct {
	models.RpcEndpoint
	Rank int `json:"rank"`
}

// GetRankedServicersOptions holds filter options for GetRankedServicers.
type GetRankedServicersOptions struct {
	Limit        int
	ChainID      *int    // Pointer to allow optional filtering
	Geozone      *string // Placeholder, as geozone isn't directly on rpc_endpoints
	ServicerType *string // 'static' or 'discovered'
}

// GetRankedServicers retrieves servicers, ranked by performance and filtered by options.
// Currently ranks by response_time_ms (lower is better).
func (s *PerformanceAnalyzerService) GetRankedServicers(ctx context.Context, opts GetRankedServicersOptions) ([]RankedServicer, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT id, chain_id, endpoint_url, public_key, response_time_ms, 
		       last_ping_timestamp, servicer_type, health_status, is_active, created_at, updated_at
		FROM rpc_endpoints
		WHERE is_active = TRUE AND response_time_ms IS NOT NULL`) // Only include active and pinged servicers

	var args []interface{}
	argCounter := 1

	if opts.ChainID != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND chain_id = $%d", argCounter))
		args = append(args, *opts.ChainID)
		argCounter++
	}
	if opts.ServicerType != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND servicer_type = $%d", argCounter))
		args = append(args, *opts.ServicerType)
		argCounter++
	}
	// Note: Geozone filtering would require joining with another table or having geozone info in rpc_endpoints.
	// This is a placeholder as per the issue description for rpc_endpoints schema.

	queryBuilder.WriteString(" ORDER BY response_time_ms ASC") // Lower response time is better

	if opts.Limit > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argCounter))
		args = append(args, opts.Limit)
		argCounter++
	}

	query := queryBuilder.String()
	log.Printf("[Analytics] Executing analytics query: %s with args: %v", query, args)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying ranked servicers: %w", err)
	}
	defer rows.Close()

	var rankedServicers []RankedServicer
	rank := 1
	for rows.Next() {
		var ep models.RpcEndpoint
		// Ensure all fields from the RpcEndpoint struct are scanned, especially new ones.
		// Nullable fields in the struct (like PublicKey, ResponseTimeMs, LastPingTimestamp)
		// must be pointers to be scanned correctly.
		err := rows.Scan(
			&ep.ID,
			&ep.ChainID,
			&ep.EndpointURL,
			&ep.PublicKey,         // Assumes RpcEndpoint.PublicKey is *string
			&ep.ResponseTimeMs,    // Assumes RpcEndpoint.ResponseTimeMs is *int
			&ep.LastPingTimestamp, // Assumes RpcEndpoint.LastPingTimestamp is *time.Time
			&ep.ServicerType,      // Assumes RpcEndpoint.ServicerType is string
			&ep.HealthStatus,      // Assumes RpcEndpoint.HealthStatus is string
			&ep.IsActive,          // Assumes RpcEndpoint.IsActive is bool
			&ep.CreatedAt,         // Assumes RpcEndpoint.CreatedAt is time.Time
			&ep.UpdatedAt,         // Assumes RpcEndpoint.UpdatedAt is time.Time
		)
		if err != nil {
			log.Printf("[Analytics] Error scanning ranked servicer row: %v", err)
			continue // Skip rows with scan errors
		}
		rankedServicers = append(rankedServicers, RankedServicer{RpcEndpoint: ep, Rank: rank})
		rank++
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("processing rows for ranked servicers: %w", err)
	}

	return rankedServicers, nil
}
