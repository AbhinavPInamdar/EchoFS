# EchoFS Roadmap — From PoC to Production Distributed File System

> **Goal**: Transform EchoFS from a proof-of-concept into a real, operational distributed file system with adaptive consistency — running 24/7 on free infrastructure, handling real users and real data.

---

## Phase 0: Foundation & Cleanup ✅ COMPLETE

> Fix what's broken. Make the current system actually do what it claims.

### 0.1 — Wire the Consistency Engine into the Data Path ✅
- [x] Created `internal/consistency/client.go` — queries orchestrator with local TTL cache
- [x] Created `internal/consistency/write_coordinator.go` — mode-aware write execution
- [x] Strong mode (C): parallel writes to all replicas, wait for quorum (N/2+1) acks
- [x] Available mode (A): write to 1 node, return immediately, async replicate to others
- [x] Hybrid mode: write to 2 nodes synchronously, async replicate the rest
- [x] Safe fallback: defaults to Strong if orchestrator is unreachable
- [x] Upload handler queries orchestrator, logs mode, uses write coordinator
- [x] In Strong mode, if any chunk fails quorum, entire upload is rejected and cleaned up

### 0.2 — Fix the Download Path ✅
- [x] Downloads retrieve chunks from storage nodes via gRPC (not local filesystem)
- [x] Chunks reassembled in order
- [x] Decompression (gzip) before serving to client
- [x] Fallback to local filesystem for legacy files uploaded before this change

### 0.3 — Implement State Persistence ✅
- [x] Created `internal/controller/persistent_store.go` — atomic file-based JSON persistence
- [x] Controller saves state on every mode transition (object modes, critical keys, global override)
- [x] On startup, controller restores from persisted state with version tracking
- [x] Atomic writes via temp file + rename (no corruption on crash)

### 0.4 — Security Hardening ✅
- [x] JWT secret is **required** — server refuses to start without `JWT_SECRET` env var
- [x] CORS restricted to configured frontend domains (no more `AllowedOrigins: ["*"]`)
- [x] Admin role can no longer be claimed by registering with username "admin"
- [x] Added `PromoteToAdmin()` method for explicit admin provisioning
- [x] Created `internal/middleware/rate_limiter.go` — token bucket rate limiter per IP
- [x] Filename sanitization (strips path traversal, null bytes, hidden files, length limit)
- [x] Graceful shutdown with 30s drain period (in-flight requests complete, async queue drains)
- [x] File size validation (max 100MB)

### 0.5 — Code Cleanup ✅
- [x] Deleted duplicate frontend pages (`upload/page.tsx`, `files/page.tsx`)
- [x] Removed FileManagement/Gemini API route from frontend
- [x] Replaced stub endpoints with proper error responses (501 Not Implemented)
- [x] WorkerHeartbeat now validates worker exists in registry
- [x] Graceful shutdown on all services

### 0.6 — Real Prometheus Metrics ✅
- [x] Created `internal/metrics/prometheus_client.go` — real PromQL query client
- [x] `gatherObjectMetrics()` now queries live Prometheus for:
  - Partition risk (ratio of unhealthy workers)
  - Replication lag (max across nodes)
  - Write rate (uploads/sec over 5min window)
  - Node RTT (avg gRPC request duration)
- [x] Falls back to safe defaults when Prometheus is unavailable

---

## Phase 1: Repository Restructure ✅ COMPLETE

> Reorganize into the monorepo structure that supports independent services.

### 1.1 — New Directory Layout ✅
- [x] Created top-level `services/`, `proto/`, `pkg/`, `infra/`, `scripts/`, `docs/` directories
- [x] Created `docs/adr/` for Architecture Decision Records
- [x] ADR-001: Monorepo structure decision
- [x] ADR-002: Adaptive consistency model documentation
- [x] `services/README.md` documenting service map and deployment options

### 1.2 — Go Workspace ✅
- [x] `go.work` ties together `Backend/`, `pkg/`, and `proto/gen/` modules
- [x] Shared `pkg/` module (`github.com/echofs/pkg`) with:
  - `pkg/auth/` — JWT manager + HTTP middleware (canonical shared auth)
  - `pkg/health/` — Liveness/readiness probe handlers with check registry
  - `pkg/config/` — Environment variable helpers (GetEnv, RequireEnv, GetEnvInt, etc.)
  - `pkg/observability/` — Structured logging with `slog`, request ID propagation
- [x] All modules compile independently and together via workspace

### 1.3 — Proto Reorganization ✅
- [x] Split monolithic `echofs.proto` into 3 domain-specific protos:
  - `proto/echofs/v1/storage.proto` — StorageNodeService (chunk CRUD + health)
  - `proto/echofs/v1/orchestrator.proto` — OrchestratorService (mode queries + streaming)
  - `proto/echofs/v1/metadata.proto` — MetadataService (file/chunk metadata CRUD)
- [x] `buf.yaml` + `buf.gen.yaml` for proto tooling
- [x] Generated Go code in `proto/gen/v1/` (6 files: 3 pb.go + 3 grpc.pb.go)
- [x] `scripts/proto-gen.sh` for reproducible generation
- [x] CI checks for proto drift (regenerate → fail if uncommitted changes)

### 1.4 — Multi-stage Dockerfiles ✅
- [x] `Backend/Dockerfile.master` — legacy monolith
- [x] `Backend/Dockerfile.gateway` — gateway service
- [x] `Backend/Dockerfile.ingest` — ingest service
- [x] `Backend/Dockerfile.metadata` — metadata service
- [x] `Backend/Dockerfile.worker` — storage node (includes iproute2 for chaos)
- [x] `Backend/Dockerfile.orchestrator` — consistency controller
- [x] All use multi-stage builds (golang:1.24-alpine → alpine:3.20)
- [x] CGO_ENABLED=0 for static binaries, ARM64 compatible

### 1.5 — Kubernetes Manifests ✅
- [x] `infra/k8s/base/` — full Kustomize base:
  - namespace, gateway, ingest, metadata-svc, storage-node (StatefulSet), orchestrator, postgres
- [x] `infra/k8s/overlays/local/` — reduced resources, single replica, small PVCs, dev secrets
- [x] `infra/k8s/overlays/oracle/` — full resources, 3 replicas, ingress with Caddy
- [x] Storage nodes use StatefulSet with headless service for stable DNS

### 1.6 — Infrastructure as Code ✅
- [x] `infra/terraform/oracle/main.tf` — Oracle Cloud free tier provisioning:
  - VCN with public subnet, internet gateway, security lists
  - ARM instance (4 OCPU, 24GB RAM) with cloud-init k3s bootstrap
  - 50GB block volume for persistent storage
  - Outputs: public IP, SSH command, kubeconfig fetch command
- [x] `terraform.tfvars.example` for configuration

### 1.7 — CI/CD Pipeline ✅
- [x] `.github/workflows/ci.yaml`:
  - Lint (go vet) → Build (all modules) → Test → Proto drift check
  - Multi-arch Docker image builds (amd64 + arm64)
  - Push to ghcr.io on main branch
  - Builds: gateway, ingest, metadata, worker, orchestrator

### 1.8 — Developer Experience ✅
- [x] Top-level `Makefile` with targets:
  - `build`, `test`, `lint` — standard development
  - `run-master`, `run-gateway`, `run-ingest`, `run-metadata` — run services
  - `run-worker1/2/3`, `run-controller` — run infrastructure
  - `cluster-up/down`, `deploy-local/oracle` — K8s management
  - `proto`, `proto-lint` — proto generation
  - `chaos-partition`, `chaos-latency`, `chaos-heal` — chaos engineering
  - `clean` — cleanup

---

## Phase 2: Service Decomposition ✅ COMPLETE

> Extract real microservices from the monolithic master.

### 2.1 — API Gateway Service ✅
**Binary**: `Backend/cmd/gateway/` | **Port**: 8080

- [x] Standalone HTTP service with graceful shutdown
- [x] JWT validation middleware (validates token, extracts claims)
- [x] Token bucket rate limiter on auth endpoints
- [x] Reverse proxy routing:
  - `/api/v1/files/upload`, `/files/{id}/download` → Ingest (port 8081)
  - `/api/v1/auth/*`, `/api/v1/files` (list/delete), `/auth/profile` → Metadata (port 8083)
- [x] Forwards authenticated user context via `X-User-ID`, `X-User-Email`, `X-User-Role` headers
- [x] CORS configuration from environment variables
- [x] Health check endpoint
- [x] Error handler for downstream service failures (502 Bad Gateway)

### 2.2 — Metadata Service ✅
**Binary**: `Backend/cmd/metadata/` | **Port**: 8083

- [x] Owns PostgreSQL connection exclusively (only service that talks to PG)
- [x] Auth endpoints: register, login (JWT token generation)
- [x] File CRUD: list files by owner, delete file with ownership check
- [x] User profile endpoint
- [x] Internal endpoint `/internal/files` (called by ingest after successful upload)
- [x] Schema initialization on startup
- [x] Graceful shutdown

### 2.3 — Ingest Pipeline Service ✅
**Binary**: `Backend/cmd/ingest/` | **Port**: 8081

- [x] Accepts multipart file uploads
- [x] Filename sanitization
- [x] File size validation (max 100MB)
- [x] Chunking (1MB chunks via existing chunker)
- [x] Compression (gzip)
- [x] Queries consistency orchestrator for write mode per-object
- [x] Executes write strategy via WriteCoordinator:
  - Strong: quorum writes, fail upload if quorum not met
  - Available: single write + async replication queue
  - Hybrid: 2 sync + async remainder
- [x] Notifies metadata service after successful upload (async)
- [x] Download: retrieves chunks from workers via gRPC, decompresses, streams to client
- [x] Temp file cleanup after upload
- [x] Metrics emission (upload latency, file size)
- [x] Graceful shutdown (drains async replication queue)

### 2.4 — Storage Node Service (pre-existing) ✅
**Binary**: `Backend/cmd/worker1/` | **Port**: 10081+

- [x] gRPC: StoreChunk, RetrieveChunk, DeleteChunk, HealthCheck, GetStatus
- [x] S3 backend for chunk storage (with simulation fallback)
- [x] HTTP health and status endpoints
- [x] Connection multiplexing (cmux) for HTTP + gRPC on same port
- [x] Prometheus metrics on chunk operations

### 2.5 — Consistency Orchestrator Service (pre-existing) ✅
**Binary**: `Backend/cmd/consistency-controller/` | **Port**: 8082

- [x] Real Prometheus metric queries (partition risk, lag, write rate, RTT)
- [x] Policy engine: weighted scoring (partition 40%, lag 30%, write 20%, hint 10%)
- [x] Hysteresis: 3 consecutive samples before mode transition
- [x] State persistence (survives restarts)
- [x] HTTP API: GetMode, SetHint, RegisterObject, SetGlobalOverride, CriticalKeys, Status
- [x] Cooldown period after transitions (prevents flapping)
- [x] Emergency mode detection (partition risk > 80%)

### 2.6 — Deployment Modes ✅
Both modes work and compile:
- **Monolith**: `Backend/cmd/master/server/` — single binary, all functionality
- **Decomposed**: gateway + ingest + metadata + workers + orchestrator — independent services

---

## Test Suite ✅ (61 tests passing)

| Package | Tests | What's Covered |
|---------|-------|----------------|
| `internal/consistency` | 11 | Client caching, TTL expiry, fallback, write coordinator stats |
| `internal/controller` | 17 | Policy scoring, hysteresis, persistence, object store CRUD |
| `internal/middleware` | 6 | Rate limiting, IP extraction, token refill, separate buckets |
| `internal/metadata` | 6 | Vector clocks, conflict detection, object creation |
| `pkg/auth` | 10 | JWT gen/validate, expiry, wrong secret, password hash, middleware |
| `cmd/master/server` | 11 | Filename sanitization (traversal, null bytes, length, hidden files) |
| **Total** | **61** | |

---

## Phase 3: Infrastructure as Code (Week 5-6) — NEXT

> Provision real infrastructure. Zero manual setup.

### 3.1 — Oracle Cloud Deployment
- [ ] Run `terraform apply` to provision ARM instance
- [ ] Verify k3s bootstrap via cloud-init
- [ ] Export kubeconfig to local machine
- [ ] Deploy services via `make deploy-oracle`
- [ ] Verify health endpoints accessible from internet

### 3.2 — DNS & TLS
- [ ] Set up DuckDNS subdomain (free)
- [ ] Configure Caddy ingress for automatic Let's Encrypt
- [ ] Verify HTTPS works end-to-end

### 3.3 — Database Setup
- [ ] PostgreSQL running in-cluster with PVC
- [ ] Run schema initialization
- [ ] Verify auth flow (register → login → upload)

### 3.4 — Smoke Test
- [ ] Register a user via public endpoint
- [ ] Upload a file
- [ ] List files
- [ ] Download file and verify checksum matches
- [ ] Delete file

---

## Phase 4: Observability (Week 6-7)

> You can't operate what you can't see.

### 4.1 — Metrics (Prometheus)
- [ ] Deploy Prometheus in-cluster (5GB PVC)
- [ ] ServiceMonitors for each service
- [ ] Verify orchestrator reads real metrics

### 4.2 — Dashboards (Grafana)
- [ ] System Overview dashboard
- [ ] Adaptive Consistency dashboard (mode over time, transitions)
- [ ] Storage dashboard (capacity, chunk distribution)

### 4.3 — Structured Logging
- [ ] Migrate all services from `log.Printf` to `slog` (JSON)
- [ ] Request ID propagation across service boundaries
- [ ] Deploy Loki for log aggregation

### 4.4 — Alerting
- [ ] Alertmanager → Discord/Telegram webhook
- [ ] Critical: node down, disk full, emergency mode
- [ ] Warning: mode flapping, high latency

---

## Phase 5: CI/CD Pipeline (Week 7-8)

> Push to main → deployed in 5 minutes.

- [ ] CD workflow: deploy to Oracle after CI passes
- [ ] Smoke test after deploy
- [ ] Auto-rollback on failure
- [ ] Image scanning with Trivy

---

## Phase 6: Make the Consistency Engine Real (Week 8-10)

> The research contribution. Make it undeniable.

- [ ] Storage nodes emit real replication lag metrics
- [ ] Orchestrator probes inter-node latency
- [ ] Configurable policy weights via ConfigMap
- [ ] Decision audit log
- [ ] gRPC streaming for mode change push
- [ ] Anti-entropy (Merkle tree comparison between storage nodes)
- [ ] Measure and publish convergence time

---

## Phase 7: Chaos Engineering (Week 10-11)

> Prove the system works by breaking it.

- [ ] Network partition → verify mode switch
- [ ] Latency injection → verify detection
- [ ] Node kill → verify reads from replicas
- [ ] Automated weekly chaos runs in CI

---

## Phase 8: Performance & Load Testing (Week 11-12)

> Know your limits. Publish real numbers.

- [ ] k6 load test suite
- [ ] Upload/download throughput benchmarks
- [ ] Mode transition time measurement
- [ ] Latency comparison: C mode vs A mode (prove the 85% claim)

---

## Phase 9: Documentation (Week 12-13)

- [ ] Architecture diagrams (C4 model)
- [ ] Operational runbook
- [ ] OpenAPI spec
- [ ] Contributing guide

---

## Phase 10: Stretch Goals (Week 13+)

- [ ] Erasure coding (Reed-Solomon)
- [ ] Geo-distribution across Oracle regions
- [ ] CLI tool (`echofs upload/download/status`)
- [ ] ML-based policy engine
- [ ] Multi-tenancy with storage quotas

---

## Timeline Summary

| Phase | Status | Outcome |
|-------|--------|---------|
| 0: Foundation | ✅ Complete | System works end-to-end with real consistency engine |
| 1: Restructure | ✅ Complete | Monorepo, protos, Dockerfiles, K8s manifests, CI/CD |
| 2: Decomposition | ✅ Complete | 5 independent services, both deployment modes work |
| 3: Infrastructure | 🔜 Next | Running on Oracle Cloud, zero cost |
| 4: Observability | Planned | Full visibility into system behavior |
| 5: CI/CD | Planned | Push-to-deploy, automated testing |
| 6: Consistency Engine | Planned | Research contribution, made real |
| 7: Chaos Engineering | Planned | Proven resilience under failure |
| 8: Performance | Planned | Published benchmarks, known limits |
| 9: Documentation | Planned | Operational, documented, reproducible |
| 10: Stretch | Planned | Differentiation, innovation |

---

## Current State (as of Phase 2 completion)

**Services**: 5 independent binaries (gateway, ingest, metadata, worker, orchestrator)
**Tests**: 61 passing
**Lines changed**: ~9,500 additions across 78 files
**Infrastructure**: K8s manifests, Terraform, CI/CD pipeline — ready to deploy
**Cost**: $0/month (Oracle Cloud free tier)

**What works right now**:
- Upload a file → consistency mode queried → chunks written with quorum/async based on mode
- Download a file → chunks retrieved from workers via gRPC → decompressed → served
- Consistency controller monitors metrics → switches modes with hysteresis → persists state
- Rate limiting, JWT auth, CORS, filename sanitization — all production-ready

**What's next**: Deploy to Oracle Cloud and prove it works under real network conditions.
