package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dhruvsharma/viper-client/internal/api"
	"github.com/dhruvsharma/viper-client/internal/apps"
	"github.com/dhruvsharma/viper-client/internal/db"
	"github.com/dhruvsharma/viper-client/internal/middleware"
	"github.com/dhruvsharma/viper-client/internal/rpc"
	"github.com/dhruvsharma/viper-client/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	config := utils.LoadConfig()

	// Initialize logger
	isDevelopment := os.Getenv("ENV") != "production"
	logger, err := middleware.NewZapLogger(isDevelopment)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Connect to the database
	database, err := db.New(config.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database",
			zap.Error(err),
			zap.String("database_url", config.DatabaseURL))
	}
	defer database.Close()

	// Run migrations
	if err := database.MigrateDB(""); err != nil {
		logger.Fatal("Failed to run migrations",
			zap.Error(err))
	}
	logger.Info("Database migrations completed successfully")

	// Initialize apps service
	appsService := apps.NewService(database.DB)

	// Initialize RPC components
	endpointManager := rpc.NewDBEndpointManager(database.DB)
	rpcDispatcher := rpc.NewDispatcher(endpointManager)

	// Configure default rate limits (requests per second and burst capacity)
	defaultRateLimit := 30
	defaultBurstCapacity := 60

	// Check for custom rate limit settings
	if rateStr := os.Getenv("DEFAULT_RATE_LIMIT"); rateStr != "" {
		if rate, err := strconv.Atoi(rateStr); err == nil && rate > 0 {
			defaultRateLimit = rate
		}
	}
	if burstStr := os.Getenv("DEFAULT_BURST_CAPACITY"); burstStr != "" {
		if burst, err := strconv.Atoi(burstStr); err == nil && burst > 0 {
			defaultBurstCapacity = burst
		}
	}

	// Initialize Gin router with logger middleware
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://app.vipernet.xyz", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Cosmos-Address", "X-Cosmos-Signature"},
		AllowCredentials: true,
	}))

	// Apply global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.IPRateLimiter(defaultRateLimit, defaultBurstCapacity))

	// Setup Swagger
	api.SetupSwagger(router)

	// Public routes
	// @Summary Health check endpoint
	// @Description Check if the service is running and connected to the database
	// @Tags Health
	// @Accept json
	// @Produce json
	// @Success 200 {object} api.HealthResponse "Service is healthy"
	// @Failure 503 {object} api.ErrorResponse "Service is unhealthy"
	// @Router /health [get]
	router.GET("/health", func(c *gin.Context) {
		// Test database connection
		err := database.Ping()
		if err != nil {
			logger.Error("Database ping failed", zap.Error(err))
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
	authHandler := api.NewAuthHandler(database)
	authHandler.RegisterRoutes(router)

	// Initialize and register RPC handler
	rpcHandler := api.NewRPCHandler(rpcDispatcher, appsService, database)
	rpcHandler.RegisterRoutes(router)

	// API routes - protected by Auto Authentication middleware
	apiGroup := router.Group("/api")
	apiGroup.Use(middleware.AutoAuthMiddleware(database))

	// Initialize and register apps handler
	appsHandler := api.NewAppsHandler(appsService)
	appsHandler.RegisterRoutes(apiGroup)

	// Sample protected endpoint
	// @Summary Get user profile
	// @Description Retrieves the authenticated user's profile information
	// @Tags Authentication
	// @Accept json
	// @Produce json
	// @Success 200 {object} api.UserProfile "User profile information"
	// @Failure 401 {object} api.ErrorResponse "Unauthorized"
	// @Security BearerAuth
	// @Router /api/profile [get]
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
	logger.Info("Server starting", zap.String("port", config.Port))
	if err := router.Run(":" + config.Port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
