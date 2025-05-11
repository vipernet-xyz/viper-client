package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/illegalcall/viper-client/docs"
	"github.com/illegalcall/viper-client/internal/api"
	"github.com/illegalcall/viper-client/internal/apps"
	"github.com/illegalcall/viper-client/internal/chains"
	"github.com/illegalcall/viper-client/internal/db"
	"github.com/illegalcall/viper-client/internal/middleware"
	"github.com/illegalcall/viper-client/internal/relay"
	"github.com/illegalcall/viper-client/internal/rpc"
	"github.com/illegalcall/viper-client/internal/stats"
	"github.com/illegalcall/viper-client/internal/utils"
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

	// Initialize services
	appsService := apps.NewService(database.DB)
	statsService := stats.NewService(database.DB)
	chainsService := chains.NewService(database.DB)

	// Initialize RPC components
	endpointManager := rpc.NewDBEndpointManager(database.DB)
	rpcDispatcher := rpc.NewDispatcher(endpointManager)

	relayService := relay.NewService(database.DB, appsService, rpcDispatcher)

	// Initialize Viper Network handler
	viperNetworkHandler := rpc.NewViperNetworkHandler(endpointManager)

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
	router := gin.New()
	router.Use(middleware.CORSMiddleware)

	// Apply global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.IPRateLimiter(defaultRateLimit, defaultBurstCapacity))

	// Setup Swagger
	docs.SetupSwaggerHandlers(router)

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

	// Initialize and register Viper Network handler
	viperNetworkAPIHandler := api.NewViperNetworkHandler(viperNetworkHandler, appsService)
	viperNetworkAPIHandler.RegisterRoutes(router)

	// Initialize and register Relay handler
	// relayHandler := api.NewRelayHandler(endpointManager)
	// relayHandler.RegisterRoutes(router)

	// API routes - protected by Auto Authentication middleware
	apiGroup := router.Group("/api")
	apiGroup.Use(middleware.AutoAuthMiddleware(database))

	// Initialize and register apps handler
	appsHandler := api.NewAppsHandler(appsService)
	appsHandler.RegisterRoutes(apiGroup)

	// Initialize and register stats handler
	statsHandler := api.NewStatsHandler(statsService)
	statsHandler.RegisterRoutes(apiGroup)

	// Initialize and register chains handler
	chainsHandler := api.NewChainsHandler(chainsService)
	chainsHandler.RegisterRoutes(apiGroup)

	// Initialize and register relay handler
	relayHandler := api.NewRelayHandler(relayService)
	relayHandler.RegisterRoutes(apiGroup)

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
