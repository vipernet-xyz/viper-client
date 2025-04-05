package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateDB runs database migrations
func (db *DB) MigrateDB(migrationsPath string) error {
	if migrationsPath == "" {
		// Try to get the current working directory
		wd, err := os.Getwd()
		if err == nil {
			// First check if we're in the project root
			if _, err := os.Stat(filepath.Join(wd, "migrations")); err == nil {
				migrationsPath = "file://" + filepath.Join(wd, "migrations")
			} else {
				// We might be in a subdirectory, try to find migrations relative to working dir
				rootPath := wd
				for i := 0; i < 3; i++ { // Try going up max 3 levels
					rootPath = filepath.Dir(rootPath)
					if _, err := os.Stat(filepath.Join(rootPath, "migrations")); err == nil {
						migrationsPath = "file://" + filepath.Join(rootPath, "migrations")
						break
					}
				}

				// If we still haven't found it, fall back to default
				if migrationsPath == "" {
					migrationsPath = "file://migrations"
				}
			}
		} else {
			migrationsPath = "file://migrations"
		}
	}

	log.Printf("Using migrations path: %s", migrationsPath)

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create the postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("could not create the migration instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Migrations completed successfully")
	return nil
}

// DropDB drops all database tables (for testing)
func (db *DB) DropDB(migrationsPath string) error {
	if migrationsPath == "" {
		// Use the same logic as MigrateDB
		wd, err := os.Getwd()
		if err == nil {
			if _, err := os.Stat(filepath.Join(wd, "migrations")); err == nil {
				migrationsPath = "file://" + filepath.Join(wd, "migrations")
			} else {
				rootPath := wd
				for i := 0; i < 3; i++ {
					rootPath = filepath.Dir(rootPath)
					if _, err := os.Stat(filepath.Join(rootPath, "migrations")); err == nil {
						migrationsPath = "file://" + filepath.Join(rootPath, "migrations")
						break
					}
				}

				if migrationsPath == "" {
					migrationsPath = "file://migrations"
				}
			}
		} else {
			migrationsPath = "file://migrations"
		}
	}

	log.Printf("Using migrations path for drop: %s", migrationsPath)

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create the postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("could not create the migration instance: %w", err)
	}

	if err := m.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	log.Println("Database dropped successfully")
	return nil
}
