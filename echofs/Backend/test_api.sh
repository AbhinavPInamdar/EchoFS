#!/bin/bash

echo "Testing EchoFS Master API endpoints..."
echo ""

# Test health check
echo "1. Testing health check endpoint:"
curl -s http://localhost:8080/api/v1/health | jq '.' || echo "Health check failed"
echo ""

# Test init upload
echo "2. Testing init upload endpoint:"
curl -s -X POST http://localhost:8080/api/v1/files/upload/init \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "test.txt",
    "file_size": 1024,
    "user_id": "test-user"
  }' | jq '.' || echo "Init upload failed"
echo ""

echo "API tests completed!"