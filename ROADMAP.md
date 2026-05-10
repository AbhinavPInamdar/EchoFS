# EchoFS Roadmap — From PoC to Production Distributed File System

> **Goal**: Transform EchoFS from a proof-of-concept into a real, operational distributed file system with adaptive consistency — running 24/7 on free infrastructure, handling real users and real data.

---

## Phase 0: Foundation & Cleanup (Week 1-2)

> Fix what's broken. Make the current system actually do what it claims.

### 0.1 — Wire the Consistency Engine into the Data Path
- [x] Ingest pipeline queries the Consistency Orchestrator before every write
- [x] Strong mode (C): quorum write to N/2+1 storage nodes, wait for acks
- [x] Available mode (A): write to 1 node, return immediately, async replicate
- [ ] Remove hardcoded metrics from `gatherObjectMetrics()` — read from Prometheus
- [ ] Verify: upload a file in C mode, observe quorum behavior in logs

### 0.2 — Fix the Download Path
- [x] Downloads retrieve chunks from storage nodes via gRPC (not local filesystem)
- [x] Reassemble chunks in-memory or via streaming
- [ ] Decompress before returning to client
- [x] Delete local filesystem fallback code

### 0.3 — Implement State Persistence
- [x] Replace `persistState()` no-op with file-based JSON store (atomic writes)
- [x] Controller state survives restarts
- [x] Object mode map, critical keys, and global override persisted
- [x] Write recovery logic on startup

### 0.4 — Security Hardening
- [x] Remove default JWT secret — refuse to start without explicit secret
- [x] Restrict CORS to actual frontend domain
- [x] Fix admin role assignment (remove username-based check, add PromoteToAdmin method)
- [x] Add rate limiting middleware (token bucket, per-IP)
- [ ] Sanitize filenames on upload

### 0.5 — Code Cleanup
- [ ] Delete duplicate frontend pages (`upload/page.tsx`, `files/page.tsx` vs monolithic `page.tsx`)
- [ ] Remove the Gemini API key / file summarizer feature (unrelated to core system)
- [ ] Remove all `"not implemented yet"` stub endpoints
- [x] Add graceful shutdown to all services

---

## Phase 1: Repository Restructure (Week 2-3)

> Reorganize into the monorepo structure that supports independent services.

### 1.1 — New Directory Layout
- [x] Created top-level `services/`, `proto/`, `pkg/`, `infra/`, `scripts/`, `docs/` directories
- [x] Created `docs/adr/` for Architecture Decision Records
- [x] Documented monorepo decision (ADR-001) and adaptive consistency model (ADR-002)
```
echofs/
├── services/
│   ├── gateway/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── ingest/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── storage-node/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── orchestrator/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   ├── Dockerfile
│   │   └── go.mod
│   ├── metadata/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   ├── Dockerfile
│   │   └── go.mod
│   └── frontend/
│       ├── app/
│       ├── Dockerfile
│       └── package.json
├── proto/
│   ├── echofs/v1/
│   │   ├── storage.proto
│   │   ├── metadata.proto
│   │   ├── orchestrator.proto
│   │   └── ingest.proto
│   ├── buf.yaml
│   └── buf.gen.yaml
├── pkg/
│   ├── auth/
│   ├── config/
│   ├── observability/
│   └── health/
├── infra/
│   ├── terraform/
│   │   ├── oracle/
│   │   └── modules/
│   └── k8s/
│       ├── base/
│       ├── overlays/
│       │   ├── local/
│       │   ├── oracle/
│       │   └── production/
│       └── kustomization.yaml
├── scripts/
│   ├── setup-local.sh
│   ├── chaos/
│   └── load-test/
├── docs/
│   ├── architecture.md
│   ├── runbook.md
│   └── adr/              # Architecture Decision Records
├── go.work
├── go.work.sum
├── Makefile
├── ROADMAP.md
└── .github/
    └── workflows/
        ├── ci.yaml
        ├── build-images.yaml
        └── deploy.yaml
```

### 1.2 — Go Workspace Setup
- [x] Create `go.work` with all service modules
- [x] Move shared code to `pkg/` with its own `go.mod` (auth, health, config, observability)
- [x] Proto gen module with its own `go.mod`
- [x] Verify: `go build ./...` works from root for all modules

### 1.3 — Proto Reorganization
- [x] Split monolithic `echofs.proto` into domain-specific protos (storage, orchestrator, metadata)
- [x] Set up `buf` for proto linting and generation (buf.yaml + buf.gen.yaml)
- [x] Generate Go code via protoc (storage, orchestrator, metadata — 6 files)
- [x] Proto drift check in CI (regenerate and fail if diff)
- [x] Proto generation script (`scripts/proto-gen.sh`)

### 1.4 — Multi-stage Dockerfiles
- [x] Each service: build stage → alpine runtime (master, worker, orchestrator)
- [x] ARM64 compatible builds (CGO_ENABLED=0)
- [x] iproute2 in worker image (for chaos testing with `tc netem`)

---

## Phase 2: Service Decomposition (Week 3-5)

> Extract real microservices from the monolithic master.

### 2.1 — API Gateway Service ✓
**Responsibility**: Authentication, rate limiting, request routing, TLS termination.

- [x] Extract JWT validation from master into standalone service (`cmd/gateway/`)
- [x] Implement token bucket rate limiter (per-user and per-IP)
- [x] Route `/api/v1/files/upload`, `/files/{id}/download` → Ingest service
- [x] Route `/api/v1/auth/*` → Metadata service
- [x] Route `/api/v1/files` (list/delete) → Metadata service
- [x] Health check endpoint
- [x] Forward user context via X-User-ID/X-User-Email/X-User-Role headers
- [ ] Structured logging with request IDs (slog) — deferred to Phase 4
- [ ] OpenTelemetry trace propagation — deferred to Phase 4

### 2.2 — Metadata Service ✓
**Responsibility**: File metadata CRUD, user management, PostgreSQL ownership.

- [x] Own the PostgreSQL connection (only service that talks to PG)
- [x] HTTP API: CreateFile, ListFiles, DeleteFile, GetProfile
- [x] Internal endpoint `/internal/files` for ingest service to record uploads
- [x] Ownership and access control checks (via X-User-ID header)
- [x] Auth endpoints (register/login) — JWT token generation
- [x] Schema initialization on startup
- [ ] Schema migrations via golang-migrate — deferred
- [ ] Connection pooling with pgxpool — deferred (using database/sql pool)

### 2.3 — Ingest Pipeline Service ✓
**Responsibility**: Receive uploads, chunk, compress, coordinate writes.

- [x] Accept multipart uploads from gateway
- [x] Chunk files (1MB chunks)
- [x] Compress with gzip
- [x] Query Orchestrator for consistency mode
- [x] Execute write strategy (Strong/Available/Hybrid via WriteCoordinator)
- [x] Report chunk placement to Metadata service (async HTTP call)
- [x] Download: retrieve chunks from workers, decompress, serve
- [x] Emit metrics: upload latency, chunk count
- [ ] Stream large files (don't buffer entire file in memory) — deferred

### 2.4 — Storage Node Service (already existed)
**Responsibility**: Store and retrieve chunk data.

- [x] gRPC API: StoreChunk, RetrieveChunk, DeleteChunk, HealthCheck, GetStatus
- [x] S3 backend for chunk storage
- [x] HTTP health endpoint
- [ ] Local disk storage with S3 tiering — deferred to Phase 6
- [ ] Write-ahead log (WAL) — deferred
- [ ] Anti-entropy — deferred to Phase 6

### 2.5 — Consistency Orchestrator Service (already existed)
**Responsibility**: Observe metrics, decide consistency mode.

- [x] Query Prometheus for real metrics (wired in Phase 0)
- [x] Policy engine with weighted scoring + hysteresis
- [x] Persist state with file-based store
- [x] HTTP API: GetMode, SetHint, RegisterObject, SetGlobalOverride, CriticalKeys
- [ ] gRPC streaming for mode change push — deferred to Phase 6
- [ ] Emergency mode: immediate switch when partition risk > 80%
- [ ] Expose decision audit log (why did mode change?)

### 2.6 — Frontend Rebuild
- [ ] Proper Next.js app router (no single-file monolith)
- [ ] Pages: `/`, `/login`, `/register`, `/files`, `/upload`, `/dashboard`
- [ ] Server components where possible (reduce client JS)
- [ ] Auth context with token refresh logic
- [ ] Real-time dashboard showing consistency mode, node health, metrics
- [ ] Responsive design (works on mobile)
- [ ] Environment-based API URL (no hardcoded URLs)

---

## Phase 3: Infrastructure as Code (Week 5-6)

> Provision real infrastructure. Zero manual setup.

### 3.1 — Oracle Cloud Terraform
- [ ] VCN (Virtual Cloud Network) with public + private subnets
- [ ] 1x ARM instance (4 cores, 24GB RAM) or 2-3 smaller instances
- [ ] Block volumes for persistent storage (200GB free)
- [ ] Security lists (only expose 80, 443, 6443 for K8s API)
- [ ] Cloud-init script to bootstrap k3s
- [ ] Output: kubeconfig, public IP, SSH key

### 3.2 — k3s Cluster Bootstrap
- [ ] Single-node or multi-node k3s (depending on instance split)
- [ ] Traefik disabled (we'll use Caddy for ingress)
- [ ] Local-path provisioner for PVCs
- [ ] Metrics server for HPA
- [ ] Automatic kubeconfig export for CI/CD

### 3.3 — Kustomize Manifests

**Base layer** (shared across all environments):
- [ ] Namespace: `echofs`
- [ ] Gateway: Deployment + Service + Ingress
- [ ] Ingest: Deployment + Service
- [ ] Storage Nodes: StatefulSet + Headless Service + PVCs
- [ ] Orchestrator: Deployment + Service
- [ ] Metadata: Deployment + Service
- [ ] Frontend: Deployment + Service + Ingress
- [ ] PostgreSQL: StatefulSet + PVC + Service
- [ ] MinIO: StatefulSet + PVC + Service
- [ ] ConfigMaps for non-secret config
- [ ] Secrets (sealed-secrets or SOPS-encrypted)
- [ ] NetworkPolicies (storage nodes only accept traffic from ingest)
- [ ] ResourceQuotas and LimitRanges

**Local overlay** (k3d on laptop):
- [ ] Reduced resource requests (256Mi RAM per service)
- [ ] NodePort services instead of LoadBalancer
- [ ] Single storage node replica
- [ ] No TLS (localhost)

**Oracle overlay** (production):
- [ ] Full resource requests/limits
- [ ] Caddy ingress with automatic Let's Encrypt
- [ ] 3 storage node replicas
- [ ] PVC sizes matched to free tier limits
- [ ] Pod anti-affinity (spread across fault domains)

### 3.4 — Ingress & TLS
- [ ] Caddy as ingress controller (automatic HTTPS, zero config)
- [ ] `echofs.yourdomain.dev` → frontend
- [ ] `api.echofs.yourdomain.dev` → gateway
- [ ] `grafana.echofs.yourdomain.dev` → monitoring
- [ ] DuckDNS or Cloudflare free tier for DNS

---

## Phase 4: Observability (Week 6-7)

> You can't operate what you can't see.

### 4.1 — Metrics (Prometheus)
- [ ] Deploy Prometheus (single replica, 5GB PVC for retention)
- [ ] ServiceMonitors for each service
- [ ] Key metrics per service:
  - **Gateway**: request rate, latency P50/P95/P99, auth failures
  - **Ingest**: upload latency, chunk throughput, compression ratio
  - **Storage**: disk usage, chunk count, read/write latency, replication lag
  - **Orchestrator**: mode transitions, policy scores, emergency triggers
  - **Metadata**: query latency, connection pool usage
- [ ] Recording rules for expensive queries
- [ ] Retention: 15 days (fits in free tier storage)

### 4.2 — Dashboards (Grafana)
- [ ] Deploy Grafana (1 replica, persistent dashboard storage)
- [ ] Dashboard: System Overview (all services health at a glance)
- [ ] Dashboard: Adaptive Consistency (mode over time, transition reasons, latency comparison)
- [ ] Dashboard: Storage (capacity, replication status, hot/cold distribution)
- [ ] Dashboard: Request Flow (end-to-end latency breakdown)
- [ ] Public read-only access (no login required to view dashboards)

### 4.3 — Logging
- [ ] Structured JSON logging in all services (Go `slog`)
- [ ] Request ID propagation across service boundaries
- [ ] Log to stdout (k3s collects via containerd)
- [ ] Loki for log aggregation (lightweight, fits in free tier)
- [ ] Grafana log exploration linked to dashboards

### 4.4 — Alerting
- [ ] Alertmanager with webhook to Discord/Telegram (free)
- [ ] Critical alerts:
  - Storage node down for >2 minutes
  - Replication lag >10 seconds
  - Disk usage >80%
  - Emergency mode triggered
  - Upload error rate >5%
- [ ] Warning alerts:
  - Mode flapping (>5 transitions in 10 minutes)
  - High P99 latency
  - Connection pool exhaustion

### 4.5 — Tracing (OpenTelemetry)
- [ ] OTel SDK in all Go services
- [ ] Trace context propagation via gRPC metadata
- [ ] Jaeger (all-in-one) for trace visualization
- [ ] Key traces: full upload path, full download path, mode transition

---

## Phase 5: CI/CD Pipeline (Week 7-8)

> Push to main → deployed in 5 minutes. No manual steps.

### 5.1 — Continuous Integration
```yaml
# .github/workflows/ci.yaml
# Triggers on every PR and push to main

jobs:
  detect-changes:
    # Determine which services changed

  lint:
    # golangci-lint, eslint, buf lint

  test:
    # Unit tests per service
    # Integration tests (docker-compose with real deps)

  build:
    # Multi-arch Docker builds (amd64 + arm64)
    # Push to ghcr.io/yourusername/echofs-*

  proto:
    # buf breaking change detection
    # Regenerate and verify no drift
```

### 5.2 — Continuous Deployment
```yaml
# .github/workflows/deploy.yaml
# Triggers on push to main (after CI passes)

jobs:
  deploy:
    # kubectl apply -k infra/k8s/overlays/oracle/
    # Wait for rollout
    # Run smoke tests
    # Rollback on failure
```

- [ ] GitHub Actions workflow for CI (lint, test, build)
- [ ] GitHub Actions workflow for CD (deploy to Oracle)
- [ ] Kubeconfig stored as GitHub secret
- [ ] Build only changed services (path-based triggers)
- [ ] ARM64 cross-compilation in CI
- [ ] Smoke test after deploy (upload a file, download it, verify checksum)
- [ ] Auto-rollback if smoke test fails

### 5.3 — Image Strategy
- [ ] Tag images with git SHA (immutable, traceable)
- [ ] `latest` tag for convenience in local dev
- [ ] Image scanning with Trivy (fail build on critical CVEs)
- [ ] Multi-arch manifests (run same image on laptop amd64 and Oracle arm64)

---

## Phase 6: Make the Consistency Engine Real (Week 8-10)

> This is the research contribution. Make it undeniable.

### 6.1 — Real Metric Collection
- [ ] Storage nodes emit `echofs_replication_lag_seconds` (time since last sync with peers)
- [ ] Storage nodes emit `echofs_chunk_write_duration_seconds`
- [ ] Ingest pipeline emits `echofs_write_rate_total` (writes per second per object prefix)
- [ ] Orchestrator probes inter-node latency every 10s → `echofs_node_rtt_seconds`
- [ ] Orchestrator detects partitions via failed health checks → `echofs_partition_detected`

### 6.2 — Policy Engine Improvements
- [ ] Configurable weights via ConfigMap (hot-reload without restart)
- [ ] A/B testing: run two policies simultaneously, compare decisions
- [ ] Decision audit log: every evaluation stored with inputs, score, and outcome
- [ ] Expose policy evaluation via API (input metrics → get mode without applying)
- [ ] Backtest: replay historical metrics through new policy, compare results

### 6.3 — Mode Transition Protocol
- [ ] Orchestrator pushes mode changes via gRPC server-streaming to ingest
- [ ] Ingest caches current mode per-object (avoids querying orchestrator per-request)
- [ ] Cache invalidation on mode change event
- [ ] Transition is atomic: all in-flight writes complete under old mode before switch
- [ ] Fencing: during transition, reject writes that arrive with stale mode

### 6.4 — Anti-Entropy & Convergence
- [ ] Storage nodes run background Merkle tree comparison with peers
- [ ] Detect missing/divergent chunks
- [ ] Repair by pulling from peer that has the correct version
- [ ] Measure convergence time (time from partition heal to full sync)
- [ ] Emit `echofs_convergence_duration_seconds`

### 6.5 — Conflict Resolution
- [ ] Vector clocks on every chunk write
- [ ] Detect concurrent writes (neither version dominates)
- [ ] Resolution strategies: last-writer-wins (default), merge (for append-only), user-choice
- [ ] Emit `echofs_conflict_detected_total` and `echofs_conflict_resolved_total`

---

## Phase 7: Chaos Engineering (Week 10-11)

> Prove the system works by breaking it systematically.

### 7.1 — Chaos Toolkit
- [ ] Install Chaos Mesh in cluster (or use `tc netem` for lightweight approach)
- [ ] Chaos experiments as code (YAML definitions, version controlled)

### 7.2 — Experiment Suite
- [ ] **Network partition**: isolate one storage node → verify mode switches to A
- [ ] **Latency injection**: add 500ms to one node → verify orchestrator detects degradation
- [ ] **Node kill**: terminate a storage node pod → verify reads still work from replicas
- [ ] **Disk pressure**: fill PVC to 90% → verify alerts fire, cold tiering kicks in
- [ ] **Split brain**: partition cluster into two halves → verify no data corruption after heal
- [ ] **Slow consumer**: throttle one node's network → verify it gets fewer chunk assignments

### 7.3 — Automated Chaos in CI
- [ ] Weekly scheduled chaos run (GitHub Actions cron)
- [ ] Run experiment → collect metrics → verify invariants → report
- [ ] Invariants:
  - No data loss (upload N files, partition, heal, verify all N downloadable)
  - Convergence time < 10 seconds
  - Mode switch latency < 3 seconds
  - Zero corrupt chunks (checksum verification)

---

## Phase 8: Performance & Load Testing (Week 11-12)

> Know your limits. Publish real numbers.

### 8.1 — Load Test Suite
- [ ] Tool: k6 or vegeta (both free, scriptable)
- [ ] Scenarios:
  - Sustained upload: 10 concurrent users, 10MB files, 5 minutes
  - Burst read: 100 concurrent downloads
  - Mixed workload: 70% reads, 30% writes
  - Large file: single 1GB upload
  - Many small files: 1000x 1KB files

### 8.2 — Benchmarks to Publish
- [ ] Upload throughput (MB/s) by file size
- [ ] Download latency P50/P95/P99
- [ ] Mode transition time (partition detected → mode changed)
- [ ] Convergence time (partition healed → all replicas synced)
- [ ] Latency comparison: C mode vs A mode (the 85% improvement claim — prove it)
- [ ] Max concurrent users before degradation

### 8.3 — Capacity Planning
- [ ] Determine max storage on free tier (200GB block storage)
- [ ] Determine max throughput (network: 10TB/month = ~3.8 MB/s sustained)
- [ ] Document limits clearly for users
- [ ] Auto-reject uploads when approaching capacity

---

## Phase 9: Documentation & Developer Experience (Week 12-13)

> If it's not documented, it doesn't exist.

### 9.1 — Architecture Documentation
- [ ] Architecture Decision Records (ADRs) for key choices
- [ ] System architecture diagram (C4 model: context, container, component)
- [ ] Data flow diagrams (upload, download, mode transition)
- [ ] Failure mode analysis (what happens when X dies?)

### 9.2 — Operational Runbook
- [ ] How to deploy from scratch (one command)
- [ ] How to add a storage node
- [ ] How to recover from data loss
- [ ] How to investigate a mode transition
- [ ] How to read the Grafana dashboards
- [ ] Common failure scenarios and resolution steps

### 9.3 — API Documentation
- [ ] OpenAPI spec for HTTP endpoints
- [ ] Proto documentation (buf generates docs)
- [ ] Authentication flow documentation
- [ ] SDK/client library (optional, stretch goal)

### 9.4 — Developer Setup
- [ ] `make setup` — installs all tools (k3d, kubectl, buf, etc.)
- [ ] `make dev` — starts local cluster with hot-reload
- [ ] `make test` — runs all tests
- [ ] `make chaos` — runs chaos experiments locally
- [ ] `make bench` — runs load tests
- [ ] Contributing guide

---

## Phase 10: Stretch Goals (Week 13+)

> Nice-to-haves that make it exceptional.

### 10.1 — Multi-tenancy
- [ ] Namespace isolation per user
- [ ] Storage quotas per user
- [ ] Per-tenant consistency preferences

### 10.2 — Erasure Coding
- [ ] Replace simple replication with Reed-Solomon erasure coding
- [ ] Store N+K chunks (e.g., 4 data + 2 parity)
- [ ] Survive 2 node failures with 50% less storage overhead than 3x replication

### 10.3 — Geo-Distribution
- [ ] Deploy storage nodes across Oracle Cloud regions (free tier allows multiple regions)
- [ ] Consistency mode per-region (strong within region, eventual across regions)
- [ ] Read-your-writes guarantee within a session

### 10.4 — CLI Tool
- [ ] `echofs upload <file>` — upload from terminal
- [ ] `echofs download <file-id>` — download
- [ ] `echofs status` — show cluster health and consistency mode
- [ ] `echofs mode set <object> <C|A>` — manual override

### 10.5 — ML-Based Policy (the "next leap")
- [ ] Collect historical metrics + mode decisions as training data
- [ ] Train a simple model (decision tree or small neural net) to predict optimal mode
- [ ] Compare ML policy vs rule-based policy
- [ ] Publish results: does ML outperform hand-tuned weights?

---

## Timeline Summary

| Phase | Duration | Outcome |
|-------|----------|---------|
| 0: Foundation | Week 1-2 | System actually works end-to-end |
| 1: Restructure | Week 2-3 | Clean monorepo, proper service boundaries |
| 2: Decomposition | Week 3-5 | Real microservices, independent deployment |
| 3: Infrastructure | Week 5-6 | Running on Oracle Cloud, zero cost |
| 4: Observability | Week 6-7 | Full visibility into system behavior |
| 5: CI/CD | Week 7-8 | Push-to-deploy, automated testing |
| 6: Consistency Engine | Week 8-10 | The research contribution, made real |
| 7: Chaos Engineering | Week 10-11 | Proven resilience under failure |
| 8: Performance | Week 11-12 | Published benchmarks, known limits |
| 9: Documentation | Week 12-13 | Operational, documented, reproducible |
| 10: Stretch | Week 13+ | Differentiation, innovation |

---

## Success Criteria

When this roadmap is complete, you can:

1. **Show a live URL** — `https://echofs.yourdomain.dev` running 24/7 on free infrastructure
2. **Demonstrate adaptive consistency** — inject a partition, watch the mode switch live on Grafana
3. **Prove the numbers** — published benchmarks showing latency improvement during degraded conditions
4. **Survive chaos** — kill nodes, partition networks, system self-heals with zero data loss
5. **Deploy in one command** — `make deploy` from zero to running system
6. **Explain every decision** — ADRs, architecture docs, runbooks

This isn't a demo. It's a system you operate.
