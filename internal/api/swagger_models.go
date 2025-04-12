package api

// This file contains models used for Swagger documentation

// HealthResponse represents the health check response
// @Description Health check response
type HealthResponse struct {
	// The status of the service
	// @example "success"
	Status string `json:"status"`

	// A message providing more details about the health status
	// @example "Service is healthy and connected to database"
	Message string `json:"message"`
}

// ErrorResponse represents a standard error response
// @Description Standard error response
type ErrorResponse struct {
	// Error message
	// @example "Invalid request parameters"
	Error string `json:"error"`
}

// UserProfile represents a user profile
// @Description User profile information
type UserProfile struct {
	// The status of the response
	// @example "success"
	Status string `json:"status"`

	// User information
	User UserInfo `json:"user"`
}

// UserInfo represents user information
// @Description Basic user information
type UserInfo struct {
	// User ID
	// @example "123"
	ID string `json:"id"`

	// User email
	// @example "user@example.com"
	Email string `json:"email"`

	// User name
	// @example "John Doe"
	Name string `json:"name"`
}

// SwaggerCreateAppRequest represents the request to create a new app
// @Description Request data for creating a new application
type SwaggerCreateAppRequest struct {
	// Application name
	// @example "My Blockchain App"
	Name string `json:"name" binding:"required"`

	// Application description
	// @example "A decentralized exchange app"
	Description string `json:"description"`

	// List of allowed origins for CORS
	// @example ["https://myapp.com", "https://dev.myapp.com"]
	AllowedOrigins []string `json:"allowed_origins"`

	// List of allowed blockchain chain IDs
	// @example [1, 137, 56]
	AllowedChains []int `json:"allowed_chains" binding:"required"`
}

// SwaggerCreateAppResponse represents the response for app creation
// @Description Response data after creating a new application
type SwaggerCreateAppResponse struct {
	// The created application object
	App interface{} `json:"app"`

	// Generated API key (only shown once)
	// @example "vpr_12345abcdef67890"
	APIKey string `json:"api_key"`

	// Success message
	// @example "App created successfully. Keep your API key safe, as it won't be shown again."
	Message string `json:"message"`
}

// UpdateAppRequest represents the request to update an app
// @Description Request data for updating an application
type UpdateAppRequest struct {
	// Application name
	// @example "Updated App Name"
	Name string `json:"name"`

	// Application description
	// @example "An updated description"
	Description string `json:"description"`

	// List of allowed origins for CORS
	// @example ["https://updated-app.com"]
	AllowedOrigins []string `json:"allowed_origins"`

	// List of allowed blockchain chain IDs
	// @example [1, 137, 42161]
	AllowedChains []int `json:"allowed_chains"`
}

// AppResponse represents a standard app response
// @Description Standard application response
type AppResponse struct {
	// The application object
	App interface{} `json:"app"`
}

// AppsResponse represents a response with multiple apps
// @Description Response containing multiple applications
type AppsResponse struct {
	// List of applications
	Apps []interface{} `json:"apps"`
}

// UpdateAppResponse represents the response for app updates
// @Description Response data after updating an application
type UpdateAppResponse struct {
	// The updated application object
	App interface{} `json:"app"`

	// Success message
	// @example "App updated successfully"
	Message string `json:"message"`
}

// DeleteAppResponse represents the response for app deletion
// @Description Response data after deleting an application
type DeleteAppResponse struct {
	// Success message
	// @example "App deleted successfully"
	Message string `json:"message"`
}

// JsonRpcRequest represents a JSON-RPC request
// @Description Standard JSON-RPC 2.0 request
type JsonRpcRequest struct {
	// JSON-RPC version (must be "2.0")
	// @example "2.0"
	JsonRpc string `json:"jsonrpc" binding:"required,eq=2.0"`

	// Method to call
	// @example "eth_blockNumber"
	Method string `json:"method" binding:"required"`

	// Parameters for the method
	// @example [10, {"from": "0x..."}]
	Params interface{} `json:"params"`

	// Request ID
	// @example 1
	ID interface{} `json:"id"`
}

// JsonRpcResponse represents a JSON-RPC response
// @Description Standard JSON-RPC 2.0 response
type JsonRpcResponse struct {
	// JSON-RPC version (always "2.0")
	// @example "2.0"
	JsonRpc string `json:"jsonrpc"`

	// Result of the call (if successful)
	Result interface{} `json:"result,omitempty"`

	// Error information (if unsuccessful)
	Error *JsonRpcError `json:"error,omitempty"`

	// Request ID (same as in the request)
	// @example 1
	ID interface{} `json:"id"`
}

// JsonRpcError represents a JSON-RPC error
// @Description JSON-RPC 2.0 error object
type JsonRpcError struct {
	// Error code
	// @example -32700
	Code int `json:"code"`

	// Error message
	// @example "Parse error"
	Message string `json:"message"`

	// Additional error data
	Data interface{} `json:"data,omitempty"`
}
