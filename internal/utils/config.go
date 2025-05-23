package utils

import (
	"log" // For logging errors or defaults
	"os"
	"strconv"
	"strings"
)

// Helper function to get environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as int with a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// Helper function to get environment variable as a slice of ints with a default value
func getEnvAsIntSlice(key string, defaultValue []int) []int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	parts := strings.Split(valueStr, ",")
	var result []int
	for _, part := range parts {
		val, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			result = append(result, val)
		} else {
			log.Printf("Warning: Invalid integer value '%s' in env var %s, skipping.", part, key)
		}
	}
	if len(result) == 0 && len(defaultValue) > 0 { // If parsing failed entirely but there's a default
		return defaultValue
	}
	return result
}

// MonitoringConfig holds configuration specific to the servicer monitoring system
type MonitoringConfig struct {
	// Cron job configuration
	CronSchedule       string `env:"MONITORING_CRON_SCHEDULE" default:"*/5 * * * *"`
	SessionCount       int    `env:"MONITORING_SESSION_COUNT" default:"10"`
	PingTimeout        string `env:"MONITORING_PING_TIMEOUT" default:"5s"` // Store as string, parse to time.Duration where needed
	MaxConcurrentPings int    `env:"MONITORING_MAX_CONCURRENT_PINGS" default:"50"`

	// Database cleanup configuration (for future use, define now)
	CleanupInterval  string `env:"MONITORING_CLEANUP_INTERVAL" default:"24h"` // Store as string
	MaxRetentionDays int    `env:"MONITORING_MAX_RETENTION_DAYS" default:"30"`

	// Servicer discovery (references existing chain_static)
	// Use a helper to parse comma-separated string to []int
	BlockchainIDs []int  `env:"MONITORING_BLOCKCHAIN_IDS" default:"1,2"`
	GeozoneID     string `env:"MONITORING_GEOZONE_ID" default:"0001"`
	ServicerCount int    `env:"MONITORING_SERVICER_COUNT" default:"1"`

	// Performance measurement
	PingRetries  int    `env:"MONITORING_PING_RETRIES" default:"3"`
	PingInterval string `env:"MONITORING_PING_INTERVAL" default:"1s"` // Store as string

	// Relay client authentication
	ClientPrivateKey string `env:"MONITORING_RELAY_CLIENT_PRIVATE_KEY"` // KEEP SECRET
	AppClientPubKey  string `env:"MONITORING_RELAY_APP_CLIENT_PUB_KEY"`
}

// Config holds all configuration for the application
type Config struct {
	Port        string
	DatabaseURL string
	Monitoring  MonitoringConfig // Embed the new config
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/viperdb?sslmode=disable")

	monitorCfg := MonitoringConfig{
		CronSchedule:       getEnv("MONITORING_CRON_SCHEDULE", "*/5 * * * *"),
		SessionCount:       getEnvAsInt("MONITORING_SESSION_COUNT", 10),
		PingTimeout:        getEnv("MONITORING_PING_TIMEOUT", "5s"),
		MaxConcurrentPings: getEnvAsInt("MONITORING_MAX_CONCURRENT_PINGS", 50),
		CleanupInterval:    getEnv("MONITORING_CLEANUP_INTERVAL", "24h"),
		MaxRetentionDays:   getEnvAsInt("MONITORING_MAX_RETENTION_DAYS", 30),
		BlockchainIDs:      getEnvAsIntSlice("MONITORING_BLOCKCHAIN_IDS", []int{1, 2}),
		GeozoneID:          getEnv("MONITORING_GEOZONE_ID", "0001"),
		ServicerCount:      getEnvAsInt("MONITORING_SERVICER_COUNT", 1),
		PingRetries:        getEnvAsInt("MONITORING_PING_RETRIES", 3),
		PingInterval:       getEnv("MONITORING_PING_INTERVAL", "1s"),
		ClientPrivateKey:   getEnv("MONITORING_RELAY_CLIENT_PRIVATE_KEY", ""),
		AppClientPubKey:    getEnv("MONITORING_RELAY_APP_CLIENT_PUB_KEY", ""),
	}

	// Basic validation for BlockchainIDs
	if len(monitorCfg.BlockchainIDs) == 0 {
		log.Println("Warning: MONITORING_BLOCKCHAIN_IDS is empty or invalid, using default [1, 2]")
		monitorCfg.BlockchainIDs = []int{1, 2} // Fallback to default if parsing results in empty slice
	}
	// Further validation (e.g., checking IDs against `chain_static` table)
	// would typically be done after DB connection is established,
	// or by passing the DB instance to a validation function.
	// For now, this basic check is sufficient for the LoadConfig scope.

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		Monitoring:  monitorCfg,
	}
}
