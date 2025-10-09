#!/bin/bash
echo "Terminal 1: go run cmd/worker1/main.go cmd/worker1/server.go"
echo "Terminal 2: WORKER_ID=worker2 WORKER_PORT=8082 go run cmd/worker2/main.go cmd/worker2/server.go"
echo "Terminal 3: WORKER_ID=worker3 WORKER_PORT=8083 go run cmd/worker3/main.go cmd/worker3/server.go"