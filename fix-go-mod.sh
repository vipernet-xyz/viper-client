#!/bin/sh
cat > /app/go.mod << 'EOL'
module github.com/illegalcall/viper-client

go 1.20

require (
	github.com/golang-migrate/migrate/v4 v4.18.2
	github.com/lib/pq v1.10.9
)
EOL

cd /app && go mod tidy 