# ADR-001: Monorepo with Independent Services

## Status
Accepted

## Context
EchoFS consists of multiple services (master/gateway, storage nodes, consistency orchestrator, metadata service, frontend). We needed to decide between:
1. Monorepo — all services in one Git repository
2. Polyrepo — each service in its own repository

## Decision
We chose a **monorepo** structure with Go workspaces for the backend services.

## Rationale
- **Single developer**: No team coordination overhead that polyrepo solves
- **Atomic changes**: A proto change + service update + K8s manifest update is one PR
- **Shared tooling**: One CI pipeline, one Makefile, one set of linting rules
- **Go workspaces**: Each service has its own `go.mod` for independent builds, but shares code via the workspace

## Consequences
- CI must detect which services changed and build only those
- Docker builds use the full Backend context (acceptable for our image sizes)
- All services share the same Go version and proto toolchain
- Future team scaling would require evaluating CODEOWNERS or splitting repos

## Structure
```
echofs/
├── Backend/          # All Go services (current, migrating to services/)
├── Frontend/         # Next.js frontend
├── proto/            # Shared protobuf definitions
├── infra/            # Terraform + Kubernetes manifests
├── scripts/          # Dev tooling
├── docs/             # Documentation
├── go.work           # Go workspace root
└── Makefile          # Top-level orchestration
```
