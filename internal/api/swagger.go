package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupSwagger sets up the Swagger documentation endpoints
func SetupSwagger(router *gin.Engine) {
	// Serve Swagger JSON
	router.StaticFile("/swagger.json", "./docs/swagger.json")

	// Serve Swagger YAML
	router.StaticFile("/swagger.yaml", "./docs/swagger.yaml")

	// Serve custom Swagger UI
	router.StaticFile("/swagger", "./docs/swagger-ui.html")

	// Redirect /swagger/ to /swagger
	router.GET("/swagger/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger")
	})

	// Serve any other static files from docs directory
	router.Static("/docs", "./docs")
}
