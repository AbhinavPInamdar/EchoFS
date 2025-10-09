#!/bin/bash
curl -s http://localhost:8081/health | jq '.' || echo "Worker 1 not responding"
curl -s http://localhost:8082/health | jq '.' || echo "Worker 2 not responding"
curl -s http://localhost:8083/health | jq '.' || echo "Worker 3 not responding"

curl -s -X POST http://localhost:8081/chunks/test-chunk-123 | jq '.'
curl -s http://localhost:8081/chunks/test-chunk-123 | jq '.'
curl -s http://localhost:8082/status | jq '.'