package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger(t *testing.T) {
	// Create a logger with an observer for testing
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// Setup Gin router with our logger middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger(logger))

	// Add a success route
	router.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// Add a client error route
	router.GET("/client-error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client error"})
	})

	// Add a server error route
	router.GET("/server-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	// Test successful request
	req, _ := http.NewRequest("GET", "/success", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify log output for success
	assert.Equal(t, 1, logs.Len(), "Should have 1 log entry")
	entry := logs.All()[0]
	assert.Equal(t, "Request processed", entry.Message)
	assert.Equal(t, zapcore.InfoLevel, entry.Level)
	assert.Equal(t, "/success", entry.Context[0].String)

	// Find status code field and assert its value
	var statusCodeFound bool
	for _, field := range entry.Context {
		if field.Key == "status" {
			assert.Equal(t, int64(http.StatusOK), field.Integer)
			statusCodeFound = true
			break
		}
	}
	assert.True(t, statusCodeFound, "Status code field not found in log entry")

	// Reset logs
	logs.TakeAll()

	// Test client error request
	req, _ = http.NewRequest("GET", "/client-error", nil)
	req.Header.Set("X-App-ID", "test-app-id")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify log output for client error
	assert.Equal(t, 1, logs.Len(), "Should have 1 log entry")
	entry = logs.All()[0]
	assert.Equal(t, "Client error", entry.Message)
	assert.Equal(t, zapcore.WarnLevel, entry.Level)

	// Find status code field and assert its value
	statusCodeFound = false
	for _, field := range entry.Context {
		if field.Key == "status" {
			assert.Equal(t, int64(http.StatusBadRequest), field.Integer)
			statusCodeFound = true
			break
		}
	}
	assert.True(t, statusCodeFound, "Status code field not found in log entry")

	// Reset logs
	logs.TakeAll()

	// Test server error request
	req, _ = http.NewRequest("GET", "/server-error", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify log output for server error
	assert.Equal(t, 1, logs.Len(), "Should have 1 log entry")
	entry = logs.All()[0]
	assert.Equal(t, "Server error", entry.Message)
	assert.Equal(t, zapcore.ErrorLevel, entry.Level)

	// Find status code field and assert its value
	statusCodeFound = false
	for _, field := range entry.Context {
		if field.Key == "status" {
			assert.Equal(t, int64(http.StatusInternalServerError), field.Integer)
			statusCodeFound = true
			break
		}
	}
	assert.True(t, statusCodeFound, "Status code field not found in log entry")
}

func TestNewZapLogger(t *testing.T) {
	// Test development logger
	devLogger, err := NewZapLogger(true)
	assert.NoError(t, err)
	assert.NotNil(t, devLogger)

	// Test production logger
	prodLogger, err := NewZapLogger(false)
	assert.NoError(t, err)
	assert.NotNil(t, prodLogger)
}
