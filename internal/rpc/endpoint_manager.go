package rpc

import (
	"database/sql"
	"time"

	"github.com/illegalcall/viper-client/internal/models"
)

// DBEndpointManager manages RPC endpoints using a database
// @Description Manages RPC endpoints for blockchain networks
type DBEndpointManager struct {
	db *sql.DB
}

// NewDBEndpointManager creates a new endpoint manager with the provided database connection
func NewDBEndpointManager(db *sql.DB) *DBEndpointManager {
	return &DBEndpointManager{
		db: db,
	}
}

// GetActiveEndpoints returns all active endpoints for a given chain ID, sorted by priority
// @Summary Get active endpoints
// @Description Retrieves all active RPC endpoints for a specific chain ID, filtered by Indian geolocation
// @Tags RPC
// @Accept json
// @Produce json
// @Param chainID path int true "Chain ID"
// @Success 200 {array} models.RpcEndpoint "List of active endpoints"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /internal/rpc/endpoints/{chainID} [get]
func (em *DBEndpointManager) GetActiveEndpoints(chainID int) ([]models.RpcEndpoint, error) {
	query := `
		SELECT id, chain_id, endpoint_url, provider, is_active, priority, 
		       health_check_timestamp, health_status, created_at, updated_at
		FROM rpc_endpoints
		WHERE chain_id = $1 AND is_active = true AND geozone = 'IND'
		ORDER BY priority DESC
	`

	rows, err := em.db.Query(query, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []models.RpcEndpoint
	for rows.Next() {
		var endpoint models.RpcEndpoint
		var healthCheckTime sql.NullTime
		var healthStatus sql.NullString
		var provider sql.NullString

		err := rows.Scan(
			&endpoint.ID,
			&endpoint.ChainID,
			&endpoint.EndpointURL,
			&provider,
			&endpoint.IsActive,
			&endpoint.Priority,
			&healthCheckTime,
			&healthStatus,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if healthCheckTime.Valid {
			endpoint.HealthCheckTimestamp = &healthCheckTime.Time
		}
		if healthStatus.Valid {
			endpoint.HealthStatus = healthStatus.String
		}
		if provider.Valid {
			endpoint.Provider = provider.String
		}

		endpoints = append(endpoints, endpoint)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return endpoints, nil
}

// UpdateEndpointHealth updates the health status of an endpoint
// @Summary Update endpoint health
// @Description Updates the health status of an RPC endpoint
// @Tags RPC
// @Accept json
// @Produce json
// @Param id path int true "Endpoint ID"
// @Param status body string true "Health status"
// @Success 200 {object} SuccessResponse "Health status updated"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /internal/rpc/endpoints/{id}/health [put]
func (em *DBEndpointManager) UpdateEndpointHealth(id int, status string) error {
	query := `
		UPDATE rpc_endpoints
		SET health_status = $1, health_check_timestamp = $2, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err := em.db.Exec(query, status, now, id)
	return err
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}
