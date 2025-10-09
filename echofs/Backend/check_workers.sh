#!/bin/bash

echo "🔍 Checking EchoFS Workers Status..."
echo ""

# Check if workers are running on their ports
echo "Checking ports:"
lsof -i :8081 && echo "✅ Worker 1 (port 8081) is running" || echo "❌ Worker 1 (port 8081) not running"
lsof -i :8082 && echo "✅ Worker 2 (port 8082) is running" || echo "❌ Worker 2 (port 8082) not running"  
lsof -i :8083 && echo "✅ Worker 3 (port 8083) is running" || echo "❌ Worker 3 (port 8083) not running"
echo ""

# Check storage directories
echo "Checking storage directories:"
[ -d "./storage/worker1/chunks" ] && echo "✅ Worker 1 storage exists" || echo "❌ Worker 1 storage missing"
[ -d "./storage/worker2/chunks" ] && echo "✅ Worker 2 storage exists" || echo "❌ Worker 2 storage missing"
[ -d "./storage/worker3/chunks" ] && echo "✅ Worker 3 storage exists" || echo "❌ Worker 3 storage missing"
echo ""

echo "If all workers are running, test them with: ./test_workers.sh"