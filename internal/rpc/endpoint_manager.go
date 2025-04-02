package rpc

import (
	"database/sql"
	"time"

	"github.com/dhruvsharma/viper-client/internal/models"
)

// DBEndpointManager manages RPC endpoints using a database
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
func (em *DBEndpointManager) GetActiveEndpoints(chainID int) ([]models.RpcEndpoint, error) {
	query := `
		SELECT id, chain_id, endpoint_url, provider, is_active, priority, 
		       health_check_timestamp, health_status, created_at, updated_at
		FROM rpc_endpoints
		WHERE chain_id = $1 AND is_active = true
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
