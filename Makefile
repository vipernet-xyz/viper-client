.PHONY: all build test test-unit test-integration run clean

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