.PHONY: build clean test run-all stop-all

# Build all binaries
build:
	@echo "Building EchoFS components..."
	@mkdir -p bin
	go build -o bin/consistency-controller ./cmd/consistency-controller/
	go build -o bin/master ./cmd/master/server/
	go build -o bin/worker1 ./cmd/worker1/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf logs/
	rm -rf storage/

# Run tests
test:
	go test ./...

# Start all services
run-all: build
	@echo "Starting monitoring stack..."
	docker-compose -f monitoring/docker-compose.yml up -d
	@echo "Starting EchoFS services..."
	@mkdir -p logs
	./bin/consistency-controller --port=8082 > logs/controller.log 2>&1 &
	./bin/master > logs/master.log 2>&1 &
	./bin/worker1 > logs/worker1.log 2>&1 &
	@echo "All services started. Check logs/ directory for output."

# Stop all services
stop-all:
	@echo "Stopping EchoFS services..."
	pkill -f "consistency-controller\|master\|worker1" || true
	docker-compose -f monitoring/docker-compose.yml down
	@echo "All services stopped."

# Development helpers
dev: build
	@echo "Starting development environment..."
	@make run-all

logs:
	@echo "=== Controller Logs ==="
	@tail -20 logs/controller.log 2>/dev/null || echo "No controller logs"
	@echo "=== Master Logs ==="
	@tail -20 logs/master.log 2>/dev/null || echo "No master logs"
	@echo "=== Worker Logs ==="
	@tail -20 logs/worker1.log 2>/dev/null || echo "No worker logs"