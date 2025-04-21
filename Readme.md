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
/migrations                     # Database migration files
/docs                           # Documentation files
```

## Features

### RPC Forwarding

Provides a unified API to forward RPC requests to various blockchain networks through verified RPC endpoints.

### Viper Network Integration

Offers direct access to Viper Network functionality including height queries, transaction submission, and account information.

### Relay Functionality

Implements the full relay protocol for secure communication with the Viper Network, including:
- Session dispatching
- Application Authentication Token (AAT) generation
- Cryptographically signed relay requests
- Direct and full-flow relay execution

For detailed information about the relay functionality, see [Relay Documentation](docs/relay.md).

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

Alternatively, you can use the Makefile:

```
make docker-run
```

Any changes you make to the Go code will automatically trigger a rebuild and restart of the application thanks to Air.

#### Running Locally

The application can be run locally without Docker by setting up a PostgreSQL database and configuring the connection string in the environment variables:

```
export DATABASE_URL=postgres://postgres:password@localhost:5432/viperdb?sslmode=disable
go run cmd/server/main.go
```

Or use the Makefile:

```
make run
```

## API Usage

### RPC Endpoints

Send RPC requests to blockchain networks:

```bash
curl -X POST http://localhost:8080/rpc/1 \
  -H "Content-Type: application/json" \
  -H "X-App-ID: your_app_id" \
  -H "X-API-Key: your_api_key" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### Relay Endpoints

Execute a complete relay process:

```bash
curl -X POST http://localhost:8080/relay/execute \
  -H "Content-Type: application/json" \
  -H "X-App-ID: your_app_id" \
  -H "X-API-Key: your_api_key" \
  -d '{
     "pub_key": "a0b7789c0aa164cbee08638cf7a22c2c68eabb98247d559b4b650ef7675a92d7",
     "blockchain": "0001",
     "geo_zone": "0001",
     "num_servicers": 1,
     "data": "{\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1,\"jsonrpc\":\"2.0\"}",
     "method": "POST",
     "headers": {
       "Content-Type": "application/json"
     }
  }'
```

For more relay examples, see the [Relay Documentation](docs/relay.md).

## Database Migrations

The application uses [golang-migrate](https://github.com/golang-migrate/migrate) for managing database schema. Migrations are located in the `/migrations` directory and are automatically run when the application starts.

## Rate Limiting

The application implements token bucket rate limiting to protect against abuse and ensure fair usage. Two types of rate limiting are available:

### IP-Based Rate Limiting

By default, all API requests are rate-limited by client IP address to provide basic protection against abuse. The default limits are:

- 30 requests per second
- 60 request burst capacity

### App-Based Rate Limiting

For RPC endpoints, additional rate limiting is applied based on the application ID provided in the `X-App-ID` header. Each app can have its own rate limit configured during app creation or update.

### Configuration

Rate limits can be configured using environment variables:

```
# Global rate limiting (applied to all requests)
export DEFAULT_RATE_LIMIT=30       # Requests per second
export DEFAULT_BURST_CAPACITY=60   # Maximum burst capacity

# Start the server with custom rate limits
go run cmd/server/main.go
```

Rate limits can also be adjusted on a per-app basis through the app management API.

## Testing

Run unit tests with:

```
make test-unit
```

Run integration tests (requires a running PostgreSQL instance):

```
make test-integration
```

Run all tests (unit and integration):

```
make test
```

Run migration tests specifically:

```
make test-migrations
```

Run integration tests with Docker (starts a PostgreSQL container automatically):

```
make docker-test-integration
```

## License

[MIT](LICENSE) 



Client public key: ba5a588d707a557c9914d2070f19d7f250cb9bdc2b39ce79201106713ed9ad01
Client address: deabf47d228c0df4ff6c766d93acf56bd44dc214
Client private key (seed only): b3cc669e939f6c8d51d34129d4445777eb3caf9c311c1947e0e927178848205d
Account created successfully:
Address: 665b339250e56cb4dae5a9f25143253eefd4d027

viper wallet transfer b826a8a10bc7363702ad7f7ae358b157993aa2bd 665b339250e56cb4dae5a9f25143253eefd4d027 120000000000 viper-test ""
viper requestors stake 665b339250e56cb4dae5a9f25143253eefd4d027 120000000000 0001,0002 0001 1 viper-test