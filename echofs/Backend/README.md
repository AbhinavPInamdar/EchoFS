# EchoFS Backend

A distributed file system backend built with Go, featuring file chunking, compression, and distributed storage across worker nodes.

## Features

- **HTTP REST API** for file operations
- **File Chunking** with configurable chunk sizes
- **File Compression** using gzip
- **Distributed Storage** across multiple worker nodes
- **Replication** for fault tolerance
- **Session Management** for upload tracking
- **Health Monitoring** for worker nodes

## Architecture

### Master Node
- Handles client requests via HTTP API
- Manages file metadata and chunk placement
- Coordinates with worker nodes
- Tracks upload sessions

### Worker Nodes
- Store file chunks
- Handle chunk upload/download requests
- Send heartbeats to master

## API Endpoints

### File Operations
- `POST /api/v1/files/upload` - Upload a file (with chunking and compression)
- `GET /api/v1/files/{fileId}/download` - Download a file
- `POST /api/v1/files/upload/init` - Initialize chunked upload
- `POST /api/v1/files/upload/chunk` - Upload individual chunk
- `POST /api/v1/files/upload/complete` - Complete chunked upload

### Worker Management
- `POST /api/v1/workers/register` - Register a new worker
- `POST /api/v1/workers/{workerId}/heartbeat` - Worker heartbeat

### System
- `GET /api/v1/health` - Health check

## Configuration

Required environment variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_ADDR` - Redis server address
- `JWT_SECRET` - JWT signing secret

Optional environment variables:
- `MASTER_HOST` - Master server host (default: 0.0.0.0)
- `MASTER_PORT` - Master server port (default: 8080)
- `LOG_LEVEL` - Logging level (default: info)
- `REPLICATION_FACTOR` - Number of replicas per chunk (default: 3)
- `CHUNK_SIZE` - Chunk size in bytes (default: 1MB)

## Running the Server

### Development Mode
```bash
# Set environment variables
export DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export JWT_SECRET="your-secret-key"

# Run the master server
go run cmd/master/server/main.go cmd/master/server/server.go
```

### Using the provided script
```bash
./run_master.sh
```

### Testing the API
```bash
./test_api.sh
```

## Project Structure

```
Backend/
├── cmd/
│   ├── master/
│   │   ├── server/          # HTTP server implementation
│   │   └── core/            # Master node core logic
│   └── worker/              # Worker node implementation
├── pkg/
│   ├── config/              # Configuration management
│   ├── fileops/
│   │   ├── Chunker/         # File chunking logic
│   │   └── Compressor/      # File compression
│   ├── aws/                 # AWS service integration
│   ├── cache/               # Caching layer
│   └── database/            # Database operations
├── internal/
│   ├── storage/             # Storage implementations
│   ├── grpc/                # gRPC services
│   └── redis/               # Redis client
└── proto/                   # Protocol buffer definitions
```

## Next Steps

1. **Database Integration** - Connect to PostgreSQL for metadata storage
2. **Redis Integration** - Implement caching and session storage
3. **Worker Implementation** - Complete worker node functionality
4. **gRPC Communication** - Implement gRPC for master-worker communication
5. **Authentication** - Add JWT-based authentication
6. **Monitoring** - Add metrics and monitoring
7. **Testing** - Add comprehensive test suite
8. **Docker** - Containerize the application