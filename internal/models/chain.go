package models

import (
	"encoding/json"
	"time"
)

// ChainStatic represents a blockchain network configuration
type ChainStatic struct {
	ID           int             `json:"id"`
	ChainID      int             `json:"chain_id"`
	Name         string          `json:"name"`
	Symbol       string          `json:"symbol"`
	NetworkType  string          `json:"network_type"` // mainnet, testnet
	IsEVM        bool            `json:"is_evm"`
	ChainDetails json.RawMessage `json:"chain_details,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}
