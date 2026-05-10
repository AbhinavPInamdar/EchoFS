# EchoFS Services

This directory documents the service architecture. The actual service code lives in `Backend/cmd/` since all services share the `echofs` Go module.

## Service Map

| Service | Binary | Port | Responsibility |
|---------|--------|------|----------------|
| **Gateway** | `Backend/cmd/gateway/` | 8080 | Auth, rate limiting, routing |
| **Ingest** | `Backend/cmd/ingest/` | 8081 | Upload, chunking, compression, write coordination |
| **Orchestrator** | `Backend/cmd/consistency-controller/` | 8082 | Consistency mode decisions |
| **Metadata** | `Backend/cmd/metadata/` | 8083 | File/user CRUD, PostgreSQL |
| **Storage Node** | `Backend/cmd/worker1/` | 10081+ | Chunk storage (gRPC) |

## Deployment Options

### Option A: Monolith (current production)
Run `Backend/cmd/master/server/` — single binary that does everything.

### Option B: Decomposed (target architecture)
Run each service independently:
```bash
make run-gateway      # Port 8080
make run-ingest       # Port 8081
make run-controller   # Port 8082
make run-metadata     # Port 8083
make run-worker1      # Port 10081
make run-worker2      # Port 10082
make run-worker3      # Port 10083
```

### Request Flow (Decomposed)

```
Client → Gateway (auth + route) → Ingest (upload) → Storage Nodes (gRPC)
                                                   → Orchestrator (mode query)
                                                   → Metadata (save record)

Client → Gateway (auth + route) → Metadata (list files)
Client → Gateway (auth + route) → Ingest (download) → Storage Nodes (retrieve)
```

## Communication

- **Client → Gateway**: HTTPS/REST
- **Gateway → Ingest/Metadata**: HTTP (internal, no auth — gateway already validated)
- **Ingest → Storage Nodes**: gRPC
- **Ingest → Orchestrator**: HTTP
- **Ingest → Metadata**: HTTP (internal)
