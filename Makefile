# EchoFS - Adaptive Consistency Distributed File System
# Top-level Makefile for development and operations

.PHONY: help build test lint run stop clean proto setup dev

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# =============================================================================
# Development
# =============================================================================

setup: ## Install development dependencies
	@echo "Installing development tools..."
	@which buf > /dev/null 2>&1 || (echo "Install buf: https://buf.build/docs/installation" && exit 1)
	@which k3d > /dev/null 2>&1 || (echo "Install k3d: https://k3d.io" && exit 1)
	@which kubectl > /dev/null 2>&1 || (echo "Install kubectl" && exit 1)
	@echo "All tools installed ✓"

build: ## Build all services
	@echo "Building all services..."
	cd Backend && go build ./...
	@echo "Build complete ✓"

test: ## Run all tests
	@echo "Running tests..."
	cd Backend && go test ./... -v -count=1
	@echo "Tests complete ✓"

lint: ## Run linters
	@echo "Running linters..."
	cd Backend && go vet ./...
	@echo "Lint complete ✓"

# =============================================================================
# Local Development (run services directly)
# =============================================================================

run-master: ## Run the legacy monolithic master server
	cd Backend && JWT_SECRET=dev-secret-do-not-use-in-prod ORCHESTRATOR_URL=http://localhost:8082 go run ./cmd/master/server/

run-gateway: ## Run the gateway service (decomposed)
	cd Backend && JWT_SECRET=dev-secret-do-not-use-in-prod INGEST_URL=http://localhost:8081 METADATA_URL=http://localhost:8083 go run ./cmd/gateway/

run-ingest: ## Run the ingest service (decomposed)
	cd Backend && ORCHESTRATOR_URL=http://localhost:8082 METADATA_URL=http://localhost:8083 go run ./cmd/ingest/

run-metadata: ## Run the metadata service (decomposed)
	cd Backend && JWT_SECRET=dev-secret-do-not-use-in-prod DATABASE_URL=postgres://user:password@localhost:5432/echofs?sslmode=disable PORT=8083 go run ./cmd/metadata/

run-worker1: ## Run worker 1
	cd Backend && WORKER_ID=worker1 PORT=10081 go run ./cmd/worker1/

run-worker2: ## Run worker 2
	cd Backend && WORKER_ID=worker2 PORT=10082 go run ./cmd/worker2/

run-worker3: ## Run worker 3
	cd Backend && WORKER_ID=worker3 PORT=10083 go run ./cmd/worker3/

run-controller: ## Run the consistency controller
	cd Backend && PORT=8082 METRICS_ADDR=localhost:9090 go run ./cmd/consistency-controller/

run-frontend: ## Run the frontend dev server
	cd frontend && npm run dev

run-all-decomposed: ## Run all decomposed services (requires multiple terminals)
	@echo "Start each in a separate terminal:"
	@echo "  make run-gateway"
	@echo "  make run-ingest"
	@echo "  make run-metadata"
	@echo "  make run-controller"
	@echo "  make run-worker1"
	@echo "  make run-worker2"
	@echo "  make run-worker3"
	@echo "  make run-frontend"

# =============================================================================
# Kubernetes (local k3d cluster)
# =============================================================================

cluster-up: ## Create local k3d cluster
	k3d cluster create echofs \
		--servers 1 \
		--agents 3 \
		--port "8080:80@loadbalancer" \
		--port "8443:443@loadbalancer" \
		--k3s-arg "--disable=traefik@server:0"
	@echo "Cluster created ✓"
	@echo "Run 'make deploy-local' to deploy services"

cluster-down: ## Delete local k3d cluster
	k3d cluster delete echofs

deploy-local: ## Deploy all services to local cluster
	kubectl apply -k infra/k8s/overlays/local/

deploy-oracle: ## Deploy all services to Oracle Cloud cluster
	kubectl apply -k infra/k8s/overlays/oracle/

# =============================================================================
# Proto
# =============================================================================

proto: ## Generate protobuf code
	./scripts/proto-gen.sh
	@echo "Proto generation complete ✓"

proto-lint: ## Lint proto files (requires buf)
	@which buf > /dev/null 2>&1 && (cd proto && buf lint) || echo "buf not installed, skipping lint"

proto-breaking: ## Check for breaking proto changes (requires buf)
	@which buf > /dev/null 2>&1 && (cd proto && buf breaking --against '.git#subdir=proto') || echo "buf not installed, skipping breaking check"

# =============================================================================
# Chaos Engineering
# =============================================================================

chaos-partition: ## Simulate network partition (isolate worker-2)
	@echo "Injecting network partition on worker-2..."
	kubectl exec -n echofs deploy/storage-node-2 -- tc qdisc add dev eth0 root netem loss 100%
	@echo "Partition active. Run 'make chaos-heal' to restore."

chaos-latency: ## Inject 500ms latency on worker-1
	@echo "Injecting 500ms latency on worker-1..."
	kubectl exec -n echofs deploy/storage-node-1 -- tc qdisc add dev eth0 root netem delay 500ms
	@echo "Latency injected. Run 'make chaos-heal' to restore."

chaos-heal: ## Remove all chaos injections
	@echo "Healing all chaos injections..."
	kubectl exec -n echofs deploy/storage-node-1 -- tc qdisc del dev eth0 root 2>/dev/null || true
	kubectl exec -n echofs deploy/storage-node-2 -- tc qdisc del dev eth0 root 2>/dev/null || true
	kubectl exec -n echofs deploy/storage-node-3 -- tc qdisc del dev eth0 root 2>/dev/null || true
	@echo "All partitions healed ✓"

# =============================================================================
# Cleanup
# =============================================================================

clean: ## Clean build artifacts
	cd Backend && go clean ./...
	rm -rf Backend/storage/uploads/*
	@echo "Clean complete ✓"
