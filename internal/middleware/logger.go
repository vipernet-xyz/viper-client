package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a middleware to log request data using zap
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		// Get client info
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userID := c.GetString("user_id")
		appID := c.GetHeader("X-App-ID")

		// Check if there was an error
		var errorMessage string
		if len(c.Errors) > 0 {
			errorMessage = c.Errors.String()
		}

		// Create fields for structured logging
		fields := []zapcore.Field{
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.String("method", method),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
		}

		// Add user context if available
		if userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		// Add app context if available
		if appID != "" {
			fields = append(fields, zap.String("app_id", appID))
		}

		// Add error if present
		if errorMessage != "" {
			fields = append(fields, zap.String("error", errorMessage))
		}

		// Log with appropriate level based on status code
		if statusCode >= 500 {
			logger.Error("Server error", fields...)
		} else if statusCode >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request processed", fields...)
		}
	}
}

// NewZapLogger creates a development or production zap logger
func NewZapLogger(isDevelopment bool) (*zap.Logger, error) {
	if isDevelopment {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}
