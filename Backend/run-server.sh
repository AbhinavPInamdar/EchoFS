#!/bin/bash

# Set environment variables for local testing
export JWT_SECRET="your-secret-key-here"
export AWS_REGION="us-east-1"
export PORT="8080"

echo "Starting EchoFS HTTP Server locally..."
go run cmd/master/server/main.go cmd/master/server/server.go