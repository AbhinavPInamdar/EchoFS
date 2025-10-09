#!/bin/bash

echo "🚀 Testing Complete gRPC Integration"
echo ""

# Start all workers with gRPC
echo "Starting workers with gRPC servers..."
go run cmd/worker1/main.go cmd/worker1/server.go &
WORKER1_PID=$!

WORKER_ID=worker2 WORKER_PORT=8082 go run cmd/worker2/main.go cmd/worker2/server.go &
WORKER2_PID=$!

WORKER_ID=worker3 WORKER_PORT=8083 go run cmd/worker3/main.go cmd/worker3/server.go &
WORKER3_PID=$!

sleep 5

echo "✅ Workers started:"
echo "- Worker1: HTTP:8081, gRPC:9081"
echo "- Worker2: HTTP:8082, gRPC:9082"
echo "- Worker3: HTTP:8083, gRPC:9083"
echo ""

# Start master with gRPC integration
echo "Starting master with gRPC client..."
source ./aws_test_config.sh
go run cmd/master/server/main.go cmd/master/server/server.go &
MASTER_PID=$!

sleep 5

echo "✅ Master started with gRPC worker registry"
echo ""

# Test gRPC worker health via master
echo "Testing gRPC worker health via master:"
curl -s http://localhost:8080/api/v1/workers/health | jq '.'
echo ""

# Test file upload with gRPC chunk distribution
echo "Testing file upload with gRPC chunk distribution:"
echo "This is a test file for gRPC integration" > test_grpc_file.txt
curl -X POST http://localhost:8080/api/v1/files/upload \
  -F "file=@test_grpc_file.txt" \
  -F "user_id=grpc-test-user" \
  -s | jq '.'
echo ""

# Cleanup
echo "🧹 Cleaning up..."
kill $MASTER_PID $WORKER1_PID $WORKER2_PID $WORKER3_PID 2>/dev/null
rm -f test_grpc_file.txt

echo "✅ gRPC integration test completed!"