#!/bin/bash
curl -s http://localhost:8080/api/v1/health | jq '.'

curl -s -X POST http://localhost:8080/api/v1/files/upload/init \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "test.txt",
    "file_size": 1024,
    "user_id": "test-user"
  }' | jq '.'