package middleware

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// SwaggerUI returns a middleware that serves Swagger UI files
func SwaggerUI(swaggerDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		relativePath := c.Param("path")

		// Default to index.html
		if relativePath == "" {
			relativePath = "index.html"
		}

		// Resolve the absolute path
		absolutePath := filepath.Join(swaggerDir, relativePath)

		// If index.html, handle custom loading of index.html
		if strings.HasSuffix(relativePath, "index.html") || relativePath == "" {
			// Serve the index.html file with custom headers to prevent caching
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.File(filepath.Join(swaggerDir, "index.html"))
			return
		}

		// For all other files, serve them as static files
		c.File(absolutePath)
	}
}
