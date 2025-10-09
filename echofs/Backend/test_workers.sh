#!/bin/bash

echo "Testing EchoFS Workers..."
echo ""

echo "Testing Worker 1 (port 8081):"
curl -s http://localhost:8081/health | jq '.' || echo "Worker 1 not responding"
echo ""

echo "Testing Worker 2 (port 8082):"
curl -s http://localhost:8082/health | jq '.' || echo "Worker 2 not responding"
echo ""

echo "Testing Worker 3 (port 8083):"
curl -s http://localhost:8083/health | jq '.' || echo "Worker 3 not responding"
echo ""

echo "Testing chunk operations on Worker 1:"
echo "Store chunk:"
curl -s -X POST http://localhost:8081/chunks/test-chunk-123 | jq '.' || echo "Store failed"
echo ""

echo "Retrieve chunk:"
curl -s http://localhost:8081/chunks/test-chunk-123 | jq '.' || echo "Retrieve failed"
echo ""

echo "Worker status:"
curl -s http://localhost:8082/status | jq '.' || echo "Status failed"
echo ""

echo "All tests completed!"