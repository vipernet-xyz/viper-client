package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	// We can inject dependencies here if the health check needs to query them,
	// e.g., DB connection, cron job status. For now, it's simple.
	startTime time.Time
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

// RegisterRoutes registers the health check API routes.
func (h *HealthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/healthz", h.getHealth) // Common path for health checks
}

// HealthStatus represents the health status response.
type HealthStatus struct {
	Status    string `json:"status"` // "OK", "UNHEALTHY", etc.
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
	// Can add more details like DB status, last cron run, etc.
}

// getHealth godoc
// @Summary Application Health Check
// @Description Provides the operational status of the application.
// @Tags Health
// @Produce json
// @Success 200 {object} HealthStatus "Application is healthy"
// @Failure 503 {object} HealthStatus "Application is unhealthy"
// @Router /api/healthz [get]
func (h *HealthHandler) getHealth(c *gin.Context) {
	// For now, a basic "OK". This can be expanded to check DB connections,
	// cron job status (e.g., last successful run), etc.
	// If any critical component is down, return http.StatusServiceUnavailable (503).

	uptime := time.Since(h.startTime).Round(time.Second).String()

	c.JSON(http.StatusOK, HealthStatus{
		Status:    "OK",
		Message:   "Application is running.",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    uptime,
	})
}
