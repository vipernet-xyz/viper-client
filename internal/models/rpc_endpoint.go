package models

import (
	"time"
)

// RpcEndpoint represents a blockchain RPC endpoint
type RpcEndpoint struct {
	ID                   int        `json:"id"`
	ChainID              int        `json:"chain_id"` // Keep existing json tag for chain_id
	EndpointURL          string     `json:"endpoint_url"`
	Provider             string     `json:"provider,omitempty"`
	IsActive             bool       `json:"is_active"`
	Priority             int        `json:"priority"`
	HealthCheckTimestamp *time.Time `json:"health_check_timestamp,omitempty"`
	HealthStatus         string     `json:"health_status,omitempty"` // Will store JSON
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// New fields for servicer monitoring
	PublicKey       *string    `json:"public_key,omitempty"` // Pointer to allow NULL values
	ResponseTimeMs  *int       `json:"response_time_ms,omitempty"` // Pointer to allow NULL values
	LastPingTimestamp *time.Time `json:"last_ping_timestamp,omitempty"`
	ServicerType    string     `json:"servicer_type,omitempty"` // Default 'static'
}
