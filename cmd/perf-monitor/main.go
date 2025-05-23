package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/chainbound/apollo/internal/models"
	"github.com/chainbound/apollo/internal/performance"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL        string `envconfig:"PERFMON_DB_URL" required:"true"`
	CronSchedule       string `envconfig:"PERFMON_CRON_SCHEDULE" default:"@every 15m"`
	PingJSONPayload    string `envconfig:"PERFMON_PING_JSON_PAYLOAD" default:"{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}"`
	PingTimeoutSeconds int    `envconfig:"PERFMON_PING_TIMEOUT_SECONDS" default:"10"`
	TargetChainIDs     string `envconfig:"PERFMON_TARGET_CHAIN_IDS" default:""` // Comma-separated, e.g., "1,56,137"
}

var log = logrus.New()

func main() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	var cfg Config
	err := envconfig.Process("perfmon", &cfg)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Infof("Configuration loaded: %+v", cfg)

	db, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	httpClient := &http.Client{}

	perfMonitor := performance.NewPerformanceMonitor(db, httpClient)

	cronScheduler := cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.DefaultLogger),
	))

	task := &monitoringTask{
		log:         log,
		perfMonitor: perfMonitor,
		config:      &cfg,
		db:          db,
	}

	_, err = cronScheduler.AddFunc(cfg.CronSchedule, task.performMonitoring)
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	cronScheduler.Start()
	log.Info("Cron scheduler started")

	// Keep the application running
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down cron scheduler")
	cronScheduler.Stop()
	log.Info("Application stopped")
}

func connectDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	log.Info("Successfully connected to the database")
	return db, nil
}

type monitoringTask struct {
	log         *logrus.Logger
	perfMonitor *performance.PerformanceMonitor
	config      *Config
	db          *sql.DB
}

func (mt *monitoringTask) performMonitoring() {
	mt.log.Info("Starting performance monitoring task")

	chainIDsToMonitor, err := mt.parseTargetChainIDs()
	if err != nil {
		mt.log.Errorf("Failed to parse target chain IDs: %v", err)
		return
	}

	if len(chainIDsToMonitor) == 0 {
		mt.log.Info("No specific chain IDs configured, fetching all active chain IDs from DB for geozone 'IND'")
		fetchedIDs, err := mt.fetchAllActiveChainIDs()
		if err != nil {
			mt.log.Errorf("Failed to fetch active chain IDs: %v", err)
			return
		}
		if len(fetchedIDs) == 0 {
			mt.log.Info("No active chain IDs found for geozone 'IND'. Skipping monitoring cycle.")
			return
		}
		chainIDsToMonitor = fetchedIDs
		mt.log.Infof("Monitoring the following chain IDs found in DB: %v", chainIDsToMonitor)
	} else {
		mt.log.Infof("Monitoring configured chain IDs: %v", chainIDsToMonitor)
	}

	for _, chainID := range chainIDsToMonitor {
		mt.log.Infof("Processing chain ID: %d", chainID)

		servicers, err := mt.perfMonitor.fetchServicersToPing(chainID)
		if err != nil {
			mt.log.Errorf("Failed to fetch servicers for chain ID %d: %v", chainID, err)
			continue
		}

		if len(servicers) == 0 {
			mt.log.Infof("No active servicers found for chain ID %d and geozone 'IND'", chainID)
			continue
		}

		mt.log.Infof("Found %d servicers for chain ID %d", len(servicers), chainID)

		for _, servicer := range servicers {
			mt.log.Infof("Pinging servicer ID %d (%s) for chain ID %d", servicer.ID, servicer.EndpointURL, chainID)

			var errorMsg string
			pingTimestamp := time.Now()
			responseTimeMs, httpStatusCode, err := mt.perfMonitor.pingServicer(servicer, mt.config.PingJSONPayload, time.Duration(mt.config.PingTimeoutSeconds)*time.Second)

			if err != nil {
				errorMsg = err.Error()
				mt.log.Warnf("Failed to ping servicer ID %d (%s): %v. Response time: %dms, HTTP Status: %d", servicer.ID, servicer.EndpointURL, err, responseTimeMs, httpStatusCode)
			} else {
				mt.log.Infof("Successfully pinged servicer ID %d (%s). Response time: %dms, HTTP Status: %d", servicer.ID, servicer.EndpointURL, responseTimeMs, httpStatusCode)
			}

			// Record metric even if ping failed, to capture the error and attempt details
			errRecord := mt.perfMonitor.recordPerformanceMetric(servicer.ID, pingTimestamp, responseTimeMs, httpStatusCode, errorMsg)
			if errRecord != nil {
				mt.log.Errorf("Failed to record performance metric for servicer ID %d (%s): %v", servicer.ID, servicer.EndpointURL, errRecord)
			}
		}
	}

	mt.log.Info("Performance monitoring task completed")
}

func (mt *monitoringTask) parseTargetChainIDs() ([]int, error) {
	if mt.config.TargetChainIDs == "" {
		return []int{}, nil // Return empty slice if no specific chains are targeted
	}

	parts := strings.Split(mt.config.TargetChainIDs, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}
		id, err := strconv.Atoi(trimmedPart)
		if err != nil {
			return nil, fmt.Errorf("invalid chain ID '%s': %w", part, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (mt *monitoringTask) fetchAllActiveChainIDs() ([]int, error) {
	query := `
		SELECT DISTINCT chain_id
		FROM rpc_endpoints
		WHERE is_active = true AND geozone = 'IND'
		ORDER BY chain_id ASC;
	`
	rows, err := mt.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("querying distinct chain_ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning chain_id: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating chain_id rows: %w", err)
	}
	return ids, nil
}

// Ensure models.RpcEndpoint is available (it should be from internal/models)
var _ models.RpcEndpoint
