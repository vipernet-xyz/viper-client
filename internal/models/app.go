package models

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// IntArray is a custom type that implements sql.Scanner for handling PostgreSQL integer arrays
type IntArray []int

// Scan converts the database array into a Go slice
func (a *IntArray) Scan(value interface{}) error {
	if value == nil {
		*a = IntArray([]int{})
		return nil
	}

	var str string

	switch v := value.(type) {
	case string:
		// For PostgreSQL, the array comes as a string like "{1,2,3}"
		str = v
	case []byte:
		// Database might also return a byte array
		str = string(v)
	default:
		return fmt.Errorf("failed to scan array value: %v", value)
	}

	// Remove the curly braces and split by comma
	str = strings.Trim(str, "{}")
	if str == "" {
		*a = IntArray([]int{})
		return nil
	}

	// Split and convert each element to int
	elems := strings.Split(str, ",")
	result := make([]int, len(elems))
	for i, elem := range elems {
		num, err := strconv.Atoi(elem)
		if err != nil {
			return fmt.Errorf("failed to parse integer: %v", err)
		}
		result[i] = num
	}

	*a = IntArray(result)
	return nil
}

// Value converts the Go slice to a database-friendly format
func (a IntArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}

	// Convert slice of ints to a string like "{1,2,3}"
	values := make([]string, len(a))
	for i, v := range a {
		values[i] = strconv.Itoa(v)
	}

	return fmt.Sprintf("{%s}", strings.Join(values, ",")), nil
}

// App represents a decentralized application registered in the system
type App struct {
	ID             int       `json:"id"`
	AppIdentifier  string    `json:"app_identifier"`
	UserID         int       `json:"user_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	AllowedOrigins []string  `json:"allowed_origins,omitempty"`
	AllowedChains  IntArray  `json:"allowed_chains,omitempty"`
	APIKeyHash     string    `json:"api_key_hash"`
	RateLimit      int       `json:"rate_limit"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
