#!/bin/bash

# Set required environment variables for testing
export DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export JWT_SECRET="test-secret-key-for-development-only"
export LOG_LEVEL="info"
export MASTER_PORT="8080"

echo "Starting EchoFS Master Server with test configuration..."
echo "DATABASE_URL: $DATABASE_URL"
echo "REDIS_ADDR: $REDIS_ADDR"
echo "PORT: $MASTER_PORT"
echo ""

go run cmd/master/server/main.go cmd/master/server/server.go