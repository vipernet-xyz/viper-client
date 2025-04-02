package main

import (
	"log"
	"net/http"

	"github.com/dhruvsharma/viper-client/internal/api"
	"github.com/dhruvsharma/viper-client/internal/apps"
	"github.com/dhruvsharma/viper-client/internal/auth"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/middleware"
	"github.com/dhruvsharma/viper-client/internal/rpc"
	"github.com/dhruvsharma/viper-client/internal/utils"
	"github.com/gin-gonic/gin"
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

	// Run migrations
	if err := database.MigrateDB(""); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	// Initialize auth service
	authService := auth.NewAuthService(auth.Config{
		SecretKey:     config.JWTSecretKey,
		TokenDuration: config.GetJWTDuration(),
	})

	// Initialize apps service
	appsService := apps.NewService(database.DB)

	// Initialize RPC components
	endpointManager := rpc.NewDBEndpointManager(database.DB)
	rpcDispatcher := rpc.NewDispatcher(endpointManager)

	// Initialize Gin router
	router := gin.Default()

	// Public routes
	router.GET("/health", func(c *gin.Context) {
		// Test database connection
		err := database.Ping()
		if err != nil {
			log.Printf("Database ping failed: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Service is unhealthy: database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Service is healthy and connected to database",
		})
	})

	// Initialize and register handlers
	authHandler := api.NewAuthHandler(database, authService)
	authHandler.RegisterRoutes(router)

	// Initialize and register RPC handler
	rpcHandler := api.NewRPCHandler(rpcDispatcher, appsService, database)
	rpcHandler.RegisterRoutes(router)

	// API routes - protected by JWT authentication
	apiGroup := router.Group("/api")
	apiGroup.Use(middleware.AuthMiddleware(authService))

	// Initialize and register apps handler
	appsHandler := api.NewAppsHandler(appsService)
	appsHandler.RegisterRoutes(router)

	// Sample protected endpoint
	apiGroup.GET("/profile", func(c *gin.Context) {
		userID := c.GetString("user_id")
		email := c.GetString("email")
		name := c.GetString("name")

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"user": gin.H{
				"id":    userID,
				"email": email,
				"name":  name,
			},
		})
	})

	// Start server
	log.Printf("Server starting on port %s", config.Port)
	if err := router.Run(":" + config.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
