#!/bin/bash

echo "Testing EchoFS Upload API..."
echo ""

# Create a test file
echo "This is a test file for EchoFS upload demo" > test_file.txt

echo "1. Testing health check:"
curl -s http://localhost:8080/api/v1/health | jq '.' || echo "Health check failed - is the server running?"
echo ""

echo "2. Testing file upload:"
curl -X POST http://localhost:8080/api/v1/files/upload \
  -F "file=@test_file.txt" \
  -F "user_id=test-user" \
  -s | jq '.' || echo "Upload failed"
echo ""

# Clean up
rm -f test_file.txt

echo "Test completed!"