# Viper Client - Decentralized RPC Provider Backend

This project provides a backend service for managing decentralized RPC (Remote Procedure Call) endpoints across multiple blockchain networks. It allows applications to register and use a single gateway to access various blockchain networks through verified endpoints.

## Project Structure

```
/cmd
   └── server/                  # Entry point of the application
/internal
   ├── api/                     # HTTP handlers and router setup (using Gin, Echo, etc.)
   ├── auth/                    # Authentication logic (JWT validation, token parsing)
   ├── apps/                    # Business logic for managing decentralized apps
   ├── rpc/                     # RPC dispatcher & forwarding logic to blockchain nodes
   ├── db/                      # Database connection, ORM models, and queries
   ├── models/                  # Data models for users, apps, chain configuration, rpc_endpoints
   ├── middleware/              # Middleware for logging, rate limiting, error handling, etc.
   └── utils/                   # Utility functions, configuration loaders, etc.
```

## Getting Started

### Prerequisites

- Go 1.20 or higher
- Docker and Docker Compose (for local development with PostgreSQL)

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/dhruvsharma/viper-client.git
   cd viper-client
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

### Development

#### Running with Docker (Recommended)

The easiest way to run the application is with Docker Compose, which sets up both the Go application with hot reloading and a PostgreSQL database:

```
docker compose up
```

Any changes you make to the Go code will automatically trigger a rebuild and restart of the application thanks to Air.

#### Running Locally

The application can be run locally without Docker by setting up a PostgreSQL database and configuring the connection string in the environment variables:

```
export DATABASE_URL=postgres://postgres:password@localhost:5432/viperdb?sslmode=disable
go run cmd/server/main.go
```

## Testing

Run unit tests with:

```
go test ./...
```

Run integration tests (with Docker):

```
# Start the environment if not already running
docker compose up -d

# Run integration tests
docker exec -e RUN_INTEGRATION_TESTS=true viper-client-app-1 go test -tags=integration -v ./...

# Run integration tests for a specific package
docker exec -e RUN_INTEGRATION_TESTS=true viper-client-app-1 go test -tags=integration -v ./internal/db
```

## License

[MIT](LICENSE)
