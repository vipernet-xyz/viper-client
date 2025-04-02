package db

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
)

// ChainInfo represents information about a blockchain network
type ChainInfo struct {
	ID          int    `json:"id"`
	ChainID     int    `json:"chain_id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	NetworkType string `json:"network_type"`
	IsEVM       bool   `json:"is_evm"`
}

var (
	// ErrChainNotFound is returned when a chain is not found
	ErrChainNotFound = errors.New("chain not found")
)

// GetChainByID retrieves chain information by its internal ID
func (db *DB) GetChainByID(id int) (*ChainInfo, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm
		FROM chain_static
		WHERE id = $1
	`

	var chain ChainInfo
	err := db.QueryRow(query, id).Scan(
		&chain.ID,
		&chain.ChainID,
		&chain.Name,
		&chain.Symbol,
		&chain.NetworkType,
		&chain.IsEVM,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrChainNotFound
		}
		return nil, err
	}

	return &chain, nil
}

// GetChainByChainID retrieves chain information by its blockchain chain ID
func (db *DB) GetChainByChainID(chainID int) (*ChainInfo, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm
		FROM chain_static
		WHERE chain_id = $1
	`

	var chain ChainInfo
	err := db.QueryRow(query, chainID).Scan(
		&chain.ID,
		&chain.ChainID,
		&chain.Name,
		&chain.Symbol,
		&chain.NetworkType,
		&chain.IsEVM,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrChainNotFound
		}
		return nil, err
	}

	return &chain, nil
}

// GetChainByIdentifier retrieves chain information by a network identifier
// The identifier can be a name (e.g., "ethereum"), a symbol (e.g., "ETH"),
// or a chain ID (e.g., "1")
func (db *DB) GetChainByIdentifier(identifier string) (*ChainInfo, error) {
	// First, try to parse the identifier as a chain ID
	if chainID, err := parseChainID(identifier); err == nil {
		return db.GetChainByChainID(chainID)
	}

	// Otherwise, try to match by name or symbol
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm
		FROM chain_static
		WHERE LOWER(name) = LOWER($1) OR LOWER(symbol) = LOWER($1)
	`

	var chain ChainInfo
	err := db.QueryRow(query, identifier).Scan(
		&chain.ID,
		&chain.ChainID,
		&chain.Name,
		&chain.Symbol,
		&chain.NetworkType,
		&chain.IsEVM,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrChainNotFound
		}
		return nil, err
	}

	return &chain, nil
}

// ListChains returns all chains
func (db *DB) ListChains() ([]ChainInfo, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm
		FROM chain_static
		ORDER BY name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []ChainInfo
	for rows.Next() {
		var chain ChainInfo
		err := rows.Scan(
			&chain.ID,
			&chain.ChainID,
			&chain.Name,
			&chain.Symbol,
			&chain.NetworkType,
			&chain.IsEVM,
		)
		if err != nil {
			return nil, err
		}

		chains = append(chains, chain)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chains, nil
}

// Helper function to try parsing a string as a chain ID
func parseChainID(s string) (int, error) {
	// Clean up the string (remove "0x" prefix if present)
	s = strings.TrimPrefix(strings.ToLower(s), "0x")

	// Try to parse as an int
	chainID, err := parseDecimalOrHex(s)
	if err != nil {
		return 0, err
	}

	return chainID, nil
}

// parseDecimalOrHex parses a string as either a decimal or hexadecimal integer
func parseDecimalOrHex(s string) (int, error) {
	// If the string has a '0x' prefix, parse as hex
	if strings.HasPrefix(s, "0x") {
		val, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 0, err
		}
		return int(val), nil
	}

	// Otherwise parse as decimal
	val, err := strconv.Atoi(s)
	if err != nil {
		// Try parsing as hex without 0x prefix
		valInt64, err := strconv.ParseInt(s, 16, 64)
		if err != nil {
			return 0, err
		}
		return int(valInt64), nil
	}

	return val, nil
}
