#!/bin/bash

echo "Testing EchoFS File Management API..."
echo ""

echo "1. List all uploaded files:"
curl -s http://localhost:8080/api/v1/files | jq '.' || echo "Failed to list files"
echo ""

echo "2. Test file download (replace with actual file ID):"
echo "curl -O http://localhost:8080/api/v1/files/81bf3c02-a87c-4813-bf6a-89d8c2d5cc61/download"
echo ""

echo "API tests completed!"