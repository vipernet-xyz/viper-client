package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Chain represents a blockchain network in the system
type Chain struct {
	ID           int       `json:"id"`
	ChainID      int       `json:"chain_id"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	NetworkType  string    `json:"network_type"`
	IsEVM        bool      `json:"is_evm"`
	ChainDetails JSONB     `json:"chain_details"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JSONB is a custom type for handling JSONB data from PostgreSQL
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}
