#!/bin/bash

# Set environment variables for local testing
export JWT_SECRET="your-secret-key-here"
export AWS_REGION="us-east-1"
export PORT="8080"

echo "Starting EchoFS Master Server locally..."
go run cmd/master/main.go