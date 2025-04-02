package models

import (
	"time"
)

// RpcEndpoint represents a blockchain RPC endpoint
type RpcEndpoint struct {
	ID                   int        `json:"id"`
	ChainID              int        `json:"chain_id"`
	EndpointURL          string     `json:"endpoint_url"`
	Provider             string     `json:"provider,omitempty"`
	IsActive             bool       `json:"is_active"`
	Priority             int        `json:"priority"`
	HealthCheckTimestamp *time.Time `json:"health_check_timestamp,omitempty"`
	HealthStatus         string     `json:"health_status,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
