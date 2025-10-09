# EchoFS Backend

A distributed file system backend built with Go, featuring file chunking, compression, and distributed storage across worker nodes with S3 backend and gRPC communication.

## Features

- **HTTP REST API** for file operations
- **File Chunking** with configurable chunk sizes
- **File Compression** using gzip
- **Distributed Storage** with AWS S3 backend
- **gRPC Communication** between master and workers
- **Session Management** for upload tracking
- **Health Monitoring** for worker nodes
- **AWS Integration** with S3 and DynamoDB

## Architecture

### Master Node
- Handles client requests via HTTP API
- Manages file metadata and chunk placement
- Coordinates with worker nodes via gRPC
- Tracks upload sessions
- Stores metadata in DynamoDB

### Worker Nodes
- Store file chunks in AWS S3
- Handle chunk operations via gRPC
- Provide health status to master

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

### Required Environment Variables:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_ADDR` - Redis server address
- `JWT_SECRET` - JWT signing secret

### AWS Configuration:
- `AWS_REGION` - AWS region (default: us-east-1)
- `S3_BUCKET_NAME` - S3 bucket for chunk storage
- `DYNAMODB_FILES_TABLE` - DynamoDB table for file metadata
- `DYNAMODB_CHUNKS_TABLE` - DynamoDB table for chunk metadata
- `DYNAMODB_SESSIONS_TABLE` - DynamoDB table for upload sessions

### Optional Environment Variables:
- `MASTER_HOST` - Master server host (default: 0.0.0.0)
- `MASTER_PORT` - Master server port (default: 8080)
- `LOG_LEVEL` - Logging level (default: info)
- `REPLICATION_FACTOR` - Number of replicas per chunk (default: 3)
- `CHUNK_SIZE` - Chunk size in bytes (default: 1MB)

## Setup

### 1. Configure AWS Credentials
```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your actual AWS credentials
nano .env
```

Or use the config script:
```bash
# Edit aws_test_config.sh with your real AWS credentials
nano aws_test_config.sh
```

### 2. Install Dependencies
```bash
go mod download
```

## Running the System

### Option A: Start All Services at Once
```bash
# Load AWS configuration
source ./aws_test_config.sh

# Start worker in background
WORKER_ID=worker1 WORKER_PORT=8091 go run cmd/worker1/main.go cmd/worker1/server.go &

# Wait for worker to start
sleep 3

# Start master server
DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable" REDIS_ADDR="localhost:6379" JWT_SECRET="test-secret" go run cmd/master/server/main.go cmd/master/server/server.go &
```

### Option B: Start Services in Separate Terminals

#### Terminal 1 - Start Worker
```bash
cd echofs/Backend
source ./aws_test_config.sh
WORKER_ID=worker1 WORKER_PORT=8091 go run cmd/worker1/main.go cmd/worker1/server.go
```

#### Terminal 2 - Start Master
```bash
cd echofs/Backend
source ./aws_test_config.sh
DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable" REDIS_ADDR="localhost:6379" JWT_SECRET="test-secret" go run cmd/master/server/main.go cmd/master/server/server.go
```

#### Terminal 3 - Start Frontend (Optional)
```bash
cd echofs/frontend
npm run dev
```

## Access Points

- **Frontend Web Interface**: http://localhost:3000
- **Backend API**: http://localhost:8080/api/v1/
- **Worker HTTP**: http://localhost:8091
- **Worker gRPC**: localhost:9091

## Testing the System

### Health Checks
```bash
# Check master health
curl http://localhost:8080/api/v1/health

# Check worker health
curl http://localhost:8091/health

# Check all workers health via master
curl http://localhost:8080/api/v1/workers/health
```

### File Operations
```bash
# Upload a file
curl -X POST http://localhost:8080/api/v1/files/upload \
  -F "file=@example.txt" \
  -F "user_id=test-user"

# List all files
curl http://localhost:8080/api/v1/files

# Download a file (replace FILE_ID with actual file ID)
curl http://localhost:8080/api/v1/files/FILE_ID/download -o downloaded_file.txt
```

## Project Structure

```
Backend/
├── cmd/                     # Application entry points
│   ├── client/              # Client applications
│   ├── master/              # Master server
│   │   ├── core/            # Master node core logic
│   │   └── server/          # HTTP server implementation
│   ├── worker1/             # Worker node 1
│   ├── worker2/             # Worker node 2
│   └── worker3/             # Worker node 3
├── pkg/                     # Public packages
│   ├── auth/                # Authentication utilities
│   ├── aws/                 # AWS service integration
│   ├── cache/               # Caching layer
│   ├── config/              # Configuration management
│   ├── database/            # Database operations
│   └── fileops/             # File operations
│       ├── Chunker/         # File chunking logic
│       └── Compressor/      # File compression
├── internal/                # Private packages
│   ├── grpc/                # gRPC client/server implementations
│   ├── metadata/            # Metadata management
│   ├── metrics/             # Metrics collection (planned)
│   ├── redis/               # Redis client
│   ├── scheduler/           # Task scheduler (planned)
│   └── storage/             # Storage implementations (S3, local)
├── proto/                   # Protocol buffer definitions
│   └── v1/                  # Version 1 of gRPC contracts
├── scripts/                 # Utility scripts
│   ├── generate_proto.sh    # Generate gRPC code
│   ├── start_aws.sh         # Start with AWS services
│   └── start_workers.sh     # Start worker nodes
├── test/                    # Test scripts and integration tests
│   ├── integration/         # Integration test suites
│   ├── api.sh              # API endpoint tests
│   ├── aws_integration.sh   # AWS service tests
│   ├── grpc_integration.sh  # gRPC communication tests
│   └── workers.sh          # Worker node tests
├── deployment/              # Deployment configurations
│   ├── docker/              # Docker configurations
│   └── swagger/             # API documentation
├── infra/                   # Infrastructure as code
│   └── terraform/           # Terraform configurations
├── .env.example             # Environment variables template
├── aws_test_config.sh       # AWS configuration script
└── go.mod                   # Go module definition
```

## Service Details

### Master Server
- **HTTP Port**: 8080
- **Functionality**: REST API, file coordination, worker management
- **Storage**: File metadata and original files stored locally
- **Communication**: gRPC clients to workers

### Worker Server
- **HTTP Port**: 8091 (worker1), 8092 (worker2), etc.
- **gRPC Port**: 9091 (worker1), 9092 (worker2), etc.
- **Functionality**: Chunk storage in S3, gRPC server
- **Storage**: File chunks stored in AWS S3 bucket

### Frontend (Optional)
- **Port**: 3000
- **Functionality**: Web interface for file uploads and downloads
- **Features**: Drag & drop uploads, file listing, download management

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Kill processes using the ports
   lsof -ti:8080,8091,9091 | xargs kill -9
   ```

2. **AWS Credentials Not Found**
   - Ensure aws_test_config.sh has valid credentials
   - Check that AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are set

3. **S3 Bucket Access Denied**
   - Verify S3 bucket exists and is accessible
   - Check AWS credentials have S3 permissions

4. **Worker Connection Failed**
   - Ensure worker is started before master
   - Check that gRPC ports (9091, 9092, etc.) are available

## Development Status

### Completed Features
- File chunking and compression
- S3 storage integration
- gRPC communication between master and workers
- REST API for file operations
- Web frontend interface
- AWS DynamoDB integration
- Environment-based configuration

### Planned Features
- Metrics collection and monitoring
- Task scheduler for background jobs
- Enhanced error handling and retry logic
- Load balancing across multiple workers
- File replication and redundancy