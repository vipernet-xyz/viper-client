package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrations(t *testing.T) {
	// Skip if no DATABASE_URL environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	// Connect to the database
	db, err := New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get absolute path to migrations directory
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Go up one directory to project root
	projectRoot := filepath.Dir(filepath.Dir(pwd))
	migrationsPath := "file://" + filepath.Join(projectRoot, "migrations")
	t.Logf("Migrations path: %s", migrationsPath)

	// Run migrations
	err = db.MigrateDB(migrationsPath)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables were created
	tables := []string{"users", "chain_static", "apps", "rpc_endpoints"}
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public'
			AND table_name = $1
		)`
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			t.Fatalf("Failed to check if table %s exists: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s does not exist after migrations", table)
		}
	}

	// Test that the schema matches our expectations by checking some columns
	type columnCheck struct {
		table      string
		column     string
		dataType   string
		isNullable string
	}

	checks := []columnCheck{
		{"users", "provider_user_id", "character varying", "NO"},
		{"chain_static", "chain_id", "integer", "NO"},
		{"apps", "app_identifier", "character varying", "NO"},
		{"rpc_endpoints", "endpoint_url", "text", "NO"},
	}

	for _, check := range checks {
		var dataType, isNullable string
		query := `
			SELECT data_type, is_nullable
			FROM information_schema.columns
			WHERE table_name = $1
			AND column_name = $2
		`
		err := db.QueryRow(query, check.table, check.column).Scan(&dataType, &isNullable)
		if err != nil {
			if err == sql.ErrNoRows {
				t.Errorf("Column %s does not exist in table %s", check.column, check.table)
				continue
			}
			t.Fatalf("Failed to check column %s in table %s: %v", check.column, check.table, err)
		}

		if dataType != check.dataType {
			t.Errorf("Column %s in table %s has data type %s, expected %s", check.column, check.table, dataType, check.dataType)
		}

		if isNullable != check.isNullable {
			t.Errorf("Column %s in table %s has is_nullable=%s, expected %s", check.column, check.table, isNullable, check.isNullable)
		}
	}

	// Clean up by dropping tables (for idempotent tests)
	// Comment this out if you want to manually inspect the database
	err = db.DropDB(migrationsPath)
	if err != nil {
		t.Fatalf("Failed to drop database: %v", err)
	}
}
