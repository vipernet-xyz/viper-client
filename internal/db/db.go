package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB represents a database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// IsUniqueConstraintViolation checks if an error is a PostgreSQL unique constraint violation
// for a specific constraint name.
func IsUniqueConstraintViolation(err error, constraintName string) bool {
	if pgErr, ok := err.(*pq.Error); ok {
		return pgErr.Code == "23505" && pgErr.Constraint == constraintName // 23505 is unique_violation
	}
	return false
}
