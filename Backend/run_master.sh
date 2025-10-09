#!/bin/bash
export DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export JWT_SECRET="test-secret-key-for-development-only"
export LOG_LEVEL="info"
export MASTER_PORT="8080"

go run cmd/master/server/main.go cmd/master/server/server.go