// Package docs provides swagger documentation for the API
package docs

// @title           Viper Client API
// @version         1.0
// @description     A decentralized RPC provider backend with JWT authentication and request routing.
// @termsOfService  http://example.com/terms/

// @contact.name   API Support
// @contact.url    http://example.com/support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.

// @securityDefinitions.apikey APIKey
// @in header
// @name X-API-Key
// @description API key for RPC endpoints.

// @tag.name Authentication
// @tag.description User authentication endpoints

// @tag.name Apps
// @tag.description Application management endpoints

// @tag.name RPC
// @tag.description RPC forwarding endpoints

// @tag.name Health
// @tag.description Health check endpoint
