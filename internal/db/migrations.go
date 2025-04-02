package db

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateDB runs database migrations
func (db *DB) MigrateDB(migrationsPath string) error {
	if migrationsPath == "" {
		migrationsPath = "file://migrations"
	}

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
		migrationsPath = "file://migrations"
	}

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
