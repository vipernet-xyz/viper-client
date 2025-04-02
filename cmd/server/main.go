package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/utils"
)

func main() {
	// Load configuration
	config := utils.LoadConfig()

	// Connect to the database
	database, err := db.New(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Test database connection
		err := database.Ping()
		if err != nil {
			log.Printf("Database ping failed: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Service is unhealthy: database connection failed")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Service is healthy and connected to database")
	})

	log.Printf("Server starting on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
