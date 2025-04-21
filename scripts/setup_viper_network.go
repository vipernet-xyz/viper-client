package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/viperdb?sslmode=disable"
	}

	// Connect to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	fmt.Println("Connected to database successfully")

	// Prepare context
	ctx := context.Background()

	// Check table structure
	var tableExists bool
	tableErr := db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'chain_static')").Scan(&tableExists)

	if tableErr != nil {
		log.Fatalf("Error checking if chain_static table exists: %v", tableErr)
	}

	if !tableExists {
		fmt.Println("Creating chain_static table...")
		_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS chain_static (
			id INTEGER PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			logo TEXT,
			description TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`)
		if err != nil {
			log.Fatalf("Error creating chain_static table: %v", err)
		}
		fmt.Println("Successfully created chain_static table")
	} else {
		fmt.Println("chain_static table already exists")
	}

	// Check if Viper Network chain exists in chain_static
	var chainExists bool
	chainErr := db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM chain_static WHERE id = 0)").Scan(&chainExists)

	if chainErr != nil {
		log.Fatalf("Error checking for viper network chain: %v", chainErr)
	}

	if !chainExists {
		fmt.Println("Adding Viper Network chain to chain_static table...")
		_, err = db.ExecContext(ctx,
			`INSERT INTO chain_static
			(id, name, description, created_at, updated_at)
			VALUES (0, 'Viper Network', 'Special chain ID for the Viper Network', NOW(), NOW())`)
		if err != nil {
			log.Fatalf("Error inserting into chain_static: %v", err)
		}
		fmt.Println("Successfully added Viper Network chain to chain_static")
	} else {
		fmt.Println("Viper Network chain already exists in chain_static")
	}

	// Check if rpc_endpoints table exists
	tableErr = db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'rpc_endpoints')").Scan(&tableExists)

	if tableErr != nil {
		log.Fatalf("Error checking if rpc_endpoints table exists: %v", tableErr)
	}

	if !tableExists {
		fmt.Println("Creating rpc_endpoints table...")
		_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS rpc_endpoints (
			id SERIAL PRIMARY KEY,
			chain_id INTEGER NOT NULL REFERENCES chain_static(id),
			endpoint_url VARCHAR(255) NOT NULL,
			priority INTEGER NOT NULL DEFAULT 1,
			health_status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`)
		if err != nil {
			log.Fatalf("Error creating rpc_endpoints table: %v", err)
		}
		fmt.Println("Successfully created rpc_endpoints table")
	} else {
		fmt.Println("rpc_endpoints table already exists")
	}

	// Check if viper network endpoint already exists
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM rpc_endpoints WHERE chain_id = 0 AND endpoint_url LIKE '%127.0.0.1:8082%'").Scan(&count)
	if err != nil {
		log.Fatalf("Error checking for existing endpoint: %v", err)
	}

	if count > 0 {
		fmt.Println("Viper network endpoint already exists, skipping insertion")
	} else {
		// Insert default viper network endpoint
		_, err = db.ExecContext(ctx,
			`INSERT INTO rpc_endpoints 
			(chain_id, endpoint_url, priority, health_status, created_at, updated_at) 
			VALUES (0, 'http://127.0.0.1:8082', 1, 'active', NOW(), NOW())`)
		if err != nil {
			log.Fatalf("Error inserting viper network endpoint: %v", err)
		}
		fmt.Println("Successfully added viper network endpoint")
	}

	// Get all viper network endpoints
	rows, err := db.QueryContext(ctx,
		"SELECT id, endpoint_url, priority, health_status FROM rpc_endpoints WHERE chain_id = 0")
	if err != nil {
		log.Fatalf("Error querying viper network endpoints: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nViper Network Endpoints:")
	fmt.Println("-------------------------")
	for rows.Next() {
		var id int
		var url string
		var priority int
		var status string
		if err := rows.Scan(&id, &url, &priority, &status); err != nil {
			log.Fatalf("Error scanning row: %v", err)
		}
		fmt.Printf("ID: %d, URL: %s, Priority: %d, Status: %s\n", id, url, priority, status)
	}

	fmt.Println("\nViper Network is now configured for use with viper-client")
	fmt.Println("To change the viper-network endpoint, update the rpc_endpoints table in the database")
}

// ensureDatabaseSetup checks if the necessary tables exist and creates them if they don't
func ensureDatabaseSetup(ctx context.Context, db *sql.DB) error {
	// Check if chains table exists
	var tableExists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'chains')").Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("error checking if chains table exists: %w", err)
	}

	if !tableExists {
		fmt.Println("Creating database schema...")

		// Create chains table
		_, err = db.ExecContext(ctx, `
		CREATE TABLE chains (
			id INTEGER PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`)
		if err != nil {
			return fmt.Errorf("error creating chains table: %w", err)
		}

		// Create RPC endpoints table
		_, err = db.ExecContext(ctx, `
		CREATE TABLE rpc_endpoints (
			id SERIAL PRIMARY KEY,
			chain_id INTEGER NOT NULL REFERENCES chains(id),
			endpoint_url VARCHAR(255) NOT NULL,
			priority INTEGER NOT NULL DEFAULT 1,
			health_status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`)
		if err != nil {
			return fmt.Errorf("error creating rpc_endpoints table: %w", err)
		}

		fmt.Println("Database schema created successfully")
	}

	return nil
}
