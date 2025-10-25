#!/bin/bash

# Set environment variables for local testing WITHOUT AWS
export JWT_SECRET="your-secret-key-here"
export PORT="8080"
# Don't set AWS_REGION to disable AWS/DynamoDB

echo "Starting EchoFS HTTP Server locally (simple mode)..."
go run cmd/master/server/main.go cmd/master/server/server.go