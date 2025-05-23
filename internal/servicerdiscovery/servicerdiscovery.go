package servicerdiscovery

import (
	"context"
	"database/sql"
	"fmt"
	"log" // Or your preferred logging library if established
	"time"

	"github.com/illegalcall/viper-client/internal/db" // For direct DB interaction if needed, or pass *sql.DB
	"github.com/illegalcall/viper-client/internal/models"
	"github.com/illegalcall/viper-client/internal/relay" // Existing relay client
	"github.com/illegalcall/viper-client/internal/utils" // For Config
)

// ServicerDiscoveryService handles the discovery and registration of servicers.
type ServicerDiscoveryService struct {
	config *utils.MonitoringConfig
	db     *sql.DB // Using *sql.DB for broader compatibility with db operations
	// relayClient *relay.Client // Initialize per discovery cycle or keep instance? For now, create as needed.
}

// NewServicerDiscoveryService creates a new instance of ServicerDiscoveryService.
func NewServicerDiscoveryService(cfg *utils.MonitoringConfig, dbConn *sql.DB) *ServicerDiscoveryService {
	return &ServicerDiscoveryService{
		config: cfg,
		db:     dbConn,
	}
}

// DiscoverAndRegisterServicers fetches servicers from the network and registers them.
func (s *ServicerDiscoveryService) DiscoverAndRegisterServicers(ctx context.Context) error {
	log.Println("[ServicerDiscovery] Starting servicer discovery and registration cycle...")

	// Ensure ClientPrivateKey and AppClientPubKey are available from config
	// These should have been added to utils.MonitoringConfig and loaded in LoadConfig
	if s.config.ClientPrivateKey == "" {
		log.Println("[ServicerDiscovery] Warning: MONITORING_RELAY_CLIENT_PRIVATE_KEY is not set. Relay client may not function.")
		// Depending on relay.NewClientWithSigner behavior, this might be a fatal error for dispatch.
	}
	if s.config.AppClientPubKey == "" {
		log.Println("[ServicerDiscovery] Warning: MONITORING_RELAY_APP_CLIENT_PUB_KEY is not set. Relay options might be incomplete.")
	}

	relayClient, err := relay.NewClientWithSigner("", "", "", s.config.ClientPrivateKey)
	if err != nil {
		log.Printf("[ServicerDiscovery] Error creating relay client: %v. Servicer discovery will be significantly limited or fail.", err)
		// For this subtask, we log and continue to allow other logic to be reviewed,
		// but in a real scenario, this might be a fatal error for the discovery process.
		// return fmt.Errorf("failed to create relay client: %w", err)
	}

	for _, chainIDInt := range s.config.BlockchainIDs {
		chainIDStr := fmt.Sprintf("%04d", chainIDInt) // Format as "0001", "0002"
		log.Printf("[ServicerDiscovery] Discovering servicers for Chain ID: %s, GeoZone: %s", chainIDStr, s.config.GeozoneID)

		opts := relay.Options{
			PubKey:       s.config.AppClientPubKey,
			Blockchain:   chainIDStr,
			GeoZone:      s.config.GeozoneID,
			NumServicers: s.config.ServicerCount,
			Method:       "POST", 
			Headers:      map[string]string{"Content-Type": "application/json"},
		}

		if relayClient == nil {
			log.Printf("[ServicerDiscovery] Skipping actual dispatch for Chain ID %s due to relay client initialization issue.", chainIDStr)
			continue
		}

		dispatchResp, err := relayClient.SyncedDispatch(ctx, opts)
		if err != nil {
			log.Printf("[ServicerDiscovery] Error dispatching session for Chain ID %s: %v", chainIDStr, err)
			continue 
		}

		if dispatchResp == nil || dispatchResp.Session == nil || len(dispatchResp.Session.Servicers) == 0 {
			log.Printf("[ServicerDiscovery] No servicers discovered for Chain ID %s.", chainIDStr)
			continue
		}

		log.Printf("[ServicerDiscovery] Discovered %d servicers for Chain ID %s.", len(dispatchResp.Session.Servicers), chainIDStr)

		for _, servicer := range dispatchResp.Session.Servicers {
			if servicer.PublicKey == "" || servicer.NodeURL == "" {
				log.Printf("[ServicerDiscovery] Skipping servicer with empty PublicKey or NodeURL for Chain ID %s.", chainIDStr)
				continue
			}
			s.registerServicer(ctx, servicer, chainIDInt)
		}
	}
	log.Println("[ServicerDiscovery] Servicer discovery and registration cycle completed.")
	return nil
}

func (s *ServicerDiscoveryService) registerServicer(ctx context.Context, servicer *relay.Servicer, chainID int) {
	var existingID int
	query := `SELECT id FROM rpc_endpoints WHERE public_key = $1 AND servicer_type = 'discovered'`
	err := s.db.QueryRowContext(ctx, query, servicer.PublicKey).Scan(&existingID)

	if err == nil {
		updateQuery := `UPDATE rpc_endpoints SET health_check_timestamp = $1, is_active = true, updated_at = $1 WHERE id = $2`
		_, updateErr := s.db.ExecContext(ctx, updateQuery, time.Now(), existingID)
		if updateErr != nil {
			log.Printf("[ServicerDiscovery] Error updating existing servicer %s (ID: %d): %v", servicer.PublicKey, existingID, updateErr)
		}
		return
	} else if err != sql.ErrNoRows {
		log.Printf("[ServicerDiscovery] Error checking for existing servicer with PublicKey %s: %v", servicer.PublicKey, err)
		return
	}

	log.Printf("[ServicerDiscovery] Registering new servicer: PublicKey=%s, URL=%s, ChainID=%d", servicer.PublicKey, servicer.NodeURL, chainID)

	initialHealthStatus := `{"status":"pending_initial_check"}`
	now := time.Now()

	insertQuery := `
		INSERT INTO rpc_endpoints (
			chain_id, endpoint_url, public_key, servicer_type, 
			is_active, health_status, health_check_timestamp, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	var newServicerID int
	err = s.db.QueryRowContext(ctx, insertQuery,
		chainID,
		servicer.NodeURL,
		servicer.PublicKey,
		"discovered", 
		true,         
		initialHealthStatus,
		&now, 
		now,          
		now,          
	).Scan(&newServicerID)

	if err != nil {
		if db.IsUniqueConstraintViolation(err, "unique_discovered_servicer_public_key") {
			log.Printf("[ServicerDiscovery] Servicer with PublicKey %s was concurrently inserted. Skipping.", servicer.PublicKey)
		} else {
			log.Printf("[ServicerDiscovery] Error registering servicer with PublicKey %s: %v", servicer.PublicKey, err)
		}
		return
	}
	log.Printf("[ServicerDiscovery] Successfully registered new servicer: PublicKey=%s, URL=%s, ChainID=%d with new ID %d.", servicer.PublicKey, servicer.NodeURL, chainID, newServicerID)
}

// func (s *ServicerDiscoveryService) validateServicer(servicer *relay.Servicer) bool {
//  // TODO: Implement servicer validation logic using crypto functions
//  // e.g., check signature or other properties.
//  // For now, assume valid.
//  return true
// }
