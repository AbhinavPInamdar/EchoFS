# EchoFS

A distributed file system with web interface, built with Go backend and Next.js frontend. Features file chunking, compression, and distributed storage across worker nodes with AWS S3 backend and gRPC communication.

## Overview

EchoFS is a scalable distributed file storage system that provides:

- **Web Interface** for easy file management
- **Distributed Storage** with AWS S3 backend
- **File Chunking** and compression for efficient storage
- **gRPC Communication** between services
- **REST API** for programmatic access
- **Real-time Health Monitoring**

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Master Node   │    │  Worker Nodes   │
│   (Next.js)     │◄──►│   (Go + HTTP)   │◄──►│  (Go + gRPC)    │
│   Port: 3000    │    │   Port: 8080    │    │  Ports: 9091+   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   DynamoDB      │    │   AWS S3        │
                       │   (Metadata)    │    │   (Chunks)      │
                       └─────────────────┘    └─────────────────┘
```

## Quick Start

### Prerequisites
- Go 1.19+
- Node.js 18+
- AWS Account with S3 and DynamoDB access

### 1. Clone and Setup
```bash
git clone https://github.com/AbhinavPInamdar/EchoFS.git
cd EchoFS
```

### 2. Configure AWS Credentials
```bash
cd Backend
cp .env.example .env
# Edit .env with your AWS credentials
```

### 3. Start Backend Services
```bash
# Terminal 1 - Start Worker
cd Backend
source ./aws_test_config.sh
WORKER_ID=worker1 WORKER_PORT=8091 go run cmd/worker1/main.go cmd/worker1/server.go

# Terminal 2 - Start Master
cd Backend
source ./aws_test_config.sh
DATABASE_URL="postgres://user:password@localhost:5432/echofs?sslmode=disable" REDIS_ADDR="localhost:6379" JWT_SECRET="test-secret" go run cmd/master/server/main.go cmd/master/server/server.go
```

### 4. Start Frontend
```bash
# Terminal 3 - Start Frontend
cd frontend
npm install
npm run dev
```

### 5. Access the Application
- **Web Interface**: http://localhost:3000
- **API Documentation**: http://localhost:8080/api/v1/health

## Project Structure

```
EchoFS/
├── Backend/                 # Go backend services
│   ├── cmd/                 # Application entry points
│   │   ├── master/          # Master server (HTTP API)
│   │   └── worker*/         # Worker nodes (gRPC)
│   ├── internal/            # Private packages
│   │   ├── grpc/            # gRPC implementations
│   │   ├── storage/         # S3 storage layer
│   │   └── metrics/         # Monitoring (planned)
│   ├── pkg/                 # Public packages
│   │   ├── aws/             # AWS integrations
│   │   ├── fileops/         # File operations
│   │   └── database/        # Database operations
│   └── proto/               # gRPC protocol definitions
├── frontend/                # Next.js web interface
│   ├── app/                 # Next.js 13+ app directory
│   ├── components/          # React components
│   └── public/              # Static assets
└── README.md               # This file
```

## Features

### Backend (Go)
- **Master Node**: HTTP REST API, file coordination, worker management
- **Worker Nodes**: gRPC servers, S3 chunk storage, health monitoring
- **File Operations**: Chunking, compression, distributed storage
- **AWS Integration**: S3 for storage, DynamoDB for metadata
- **Health Monitoring**: Real-time worker status tracking

### Frontend (Next.js)
- **File Upload**: Drag & drop interface with progress tracking
- **File Management**: Browse, download, and manage stored files
- **System Monitoring**: View worker health and system status
- **Responsive Design**: Works on desktop and mobile devices

## API Endpoints

### File Operations
- `POST /api/v1/files/upload` - Upload files with automatic chunking
- `GET /api/v1/files` - List all stored files
- `GET /api/v1/files/{id}/download` - Download files
- `DELETE /api/v1/files/{id}` - Delete files

### System Monitoring
- `GET /api/v1/health` - Master server health
- `GET /api/v1/workers/health` - All workers health status

## Configuration

### Environment Variables
```bash
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_REGION=us-east-1
S3_BUCKET_NAME=echofs-persistent-storage

DATABASE_URL=postgres://user:pass@localhost:5432/echofs
REDIS_ADDR=localhost:6379
JWT_SECRET=your-jwt-secret

MASTER_PORT=8080
LOG_LEVEL=info
```

## Development

### Running Tests
```bash
cd Backend
go test ./...

./test/api.sh
./test/grpc_integration.sh
```

### Building for Production
```bash
cd Backend
go build -o echofs-master cmd/master/server/main.go
go build -o echofs-worker cmd/worker1/main.go

# Build frontend
cd frontend
npm run build
```

## Deployment

### Docker
```bash
docker-compose up -d
```

### Manual Deployment
See individual README files in `Backend/` and `frontend/` for detailed deployment instructions.


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
