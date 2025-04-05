// Package docs provides Swagger documentation generated for the API
package docs

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}{
	Version:     "1.0",
	Host:        "localhost:8080",
	BasePath:    "/",
	Schemes:     []string{"http", "https"},
	Title:       "Viper Client API",
	Description: "A decentralized RPC provider backend with JWT authentication and request routing.",
}

// SwaggerDoc represents the swagger documentation generated from annotations
type SwaggerDoc struct{}

// SwaggerDict holds the swagger JSON documentation data
var SwaggerDict = SwaggerDoc{}
