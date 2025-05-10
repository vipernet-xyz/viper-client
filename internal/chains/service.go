package chains

import (
	"database/sql"
	"errors"

	"github.com/illegalcall/viper-client/internal/models"
)

// Service provides functionality for managing blockchain networks
type Service struct {
	db *sql.DB
}

// NewService creates a new chains service with the provided database connection
func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

// @Summary Get chain by ID
// @Description Retrieves a chain by its database ID
// @Tags chains
// @Accept json
// @Produce json
// @Param id path int true "Chain ID"
// @Success 200 {object} models.Chain
// @Failure 404 {object} ErrorResponse "Chain not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chains/{id} [get]
func (s *Service) GetChain(id int) (*models.Chain, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm, chain_details, created_at, updated_at
		FROM chain_static
		WHERE id = $1
	`

	var chain models.Chain
	err := s.db.QueryRow(query, id).Scan(
		&chain.ID,
		&chain.ChainID,
		&chain.Name,
		&chain.Symbol,
		&chain.NetworkType,
		&chain.IsEVM,
		&chain.ChainDetails,
		&chain.CreatedAt,
		&chain.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("chain not found")
		}
		return nil, err
	}

	return &chain, nil
}

// @Summary Get chain by chain ID
// @Description Retrieves a chain by its chain ID (e.g., 1 for Viper Network, 2 for Ethereum)
// @Tags chains
// @Accept json
// @Produce json
// @Param chain_id path int true "Chain ID (e.g., 1 for Viper Network)"
// @Success 200 {object} models.Chain
// @Failure 404 {object} ErrorResponse "Chain not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chains/by-chain-id/{chain_id} [get]
func (s *Service) GetChainByChainID(chainID int) (*models.Chain, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm, chain_details, created_at, updated_at
		FROM chain_static
		WHERE chain_id = $1
	`

	var chain models.Chain
	err := s.db.QueryRow(query, chainID).Scan(
		&chain.ID,
		&chain.ChainID,
		&chain.Name,
		&chain.Symbol,
		&chain.NetworkType,
		&chain.IsEVM,
		&chain.ChainDetails,
		&chain.CreatedAt,
		&chain.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("chain not found")
		}
		return nil, err
	}

	return &chain, nil
}

// @Summary Get all chains
// @Description Retrieves all supported chains
// @Tags chains
// @Accept json
// @Produce json
// @Success 200 {array} models.Chain
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chains [get]
func (s *Service) GetAllChains() ([]models.Chain, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm, chain_details, created_at, updated_at
		FROM chain_static
		ORDER BY chain_id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []models.Chain
	for rows.Next() {
		var chain models.Chain
		err := rows.Scan(
			&chain.ID,
			&chain.ChainID,
			&chain.Name,
			&chain.Symbol,
			&chain.NetworkType,
			&chain.IsEVM,
			&chain.ChainDetails,
			&chain.CreatedAt,
			&chain.UpdatedAt,
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

// @Summary Get chains by network type
// @Description Retrieves chains by their network type (mainnet/testnet)
// @Tags chains
// @Accept json
// @Produce json
// @Param network_type path string true "Network type (mainnet/testnet)"
// @Success 200 {array} models.Chain
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chains/network-type/{network_type} [get]
func (s *Service) GetChainsByNetworkType(networkType string) ([]models.Chain, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm, chain_details, created_at, updated_at
		FROM chain_static
		WHERE network_type = $1
		ORDER BY chain_id ASC
	`

	rows, err := s.db.Query(query, networkType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []models.Chain
	for rows.Next() {
		var chain models.Chain
		err := rows.Scan(
			&chain.ID,
			&chain.ChainID,
			&chain.Name,
			&chain.Symbol,
			&chain.NetworkType,
			&chain.IsEVM,
			&chain.ChainDetails,
			&chain.CreatedAt,
			&chain.UpdatedAt,
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

// @Summary Get EVM chains
// @Description Retrieves all EVM-compatible chains
// @Tags chains
// @Accept json
// @Produce json
// @Success 200 {array} models.Chain
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chains/evm [get]
func (s *Service) GetEVMChains() ([]models.Chain, error) {
	query := `
		SELECT id, chain_id, name, symbol, network_type, is_evm, chain_details, created_at, updated_at
		FROM chain_static
		WHERE is_evm = true
		ORDER BY chain_id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []models.Chain
	for rows.Next() {
		var chain models.Chain
		err := rows.Scan(
			&chain.ID,
			&chain.ChainID,
			&chain.Name,
			&chain.Symbol,
			&chain.NetworkType,
			&chain.IsEVM,
			&chain.ChainDetails,
			&chain.CreatedAt,
			&chain.UpdatedAt,
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

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
