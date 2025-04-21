.PHONY: all build test test-unit test-integration run clean setup-viper-network run-example run-with-viper-network run-relay-example

all: build

build:
	go build -o bin/server ./cmd/server

test: test-unit test-integration

test-unit:
	go test -v ./...

test-integration:
	go test -tags=integration -v ./...

test-migrations:
	DATABASE_URL="postgres://postgres:password@localhost:5432/viperdb?sslmode=disable" go test -v ./internal/db -run TestMigrations

run:
	go run ./cmd/server

docker-run:
	docker compose up

docker-build:
	docker compose build

docker-test-integration:
	docker compose up -d db
	DATABASE_URL="postgres://postgres:password@localhost:5432/viperdb?sslmode=disable" RUN_INTEGRATION_TESTS=true go test -tags=integration -v ./...
	docker compose down

clean:
	rm -rf bin/
	rm -f server

# Setup viper-network connection
setup-viper-network:
	@echo "Setting up viper-network connection..."
	go run scripts/setup_viper_network.go

# Run the relay client example
run-example:
	@echo "Running relay client example..."
	VIPER_CLIENT_URL=http://localhost:8080 go run examples/direct_query.go

# Start viper-client with viper-network support
run-with-viper-network: setup-viper-network
	@echo "Starting viper-client with viper-network support..."
	go run cmd/server/main.go

# Run the relay example that connects directly to viper-network
run-relay-example:
	@echo "Running relay example connecting directly to viper-network..."
	go run cmd/relay-example/main.go 