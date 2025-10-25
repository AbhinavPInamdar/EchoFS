package main

import (
    "context"
    "fmt"
    "os"
    "log"
    "strconv"
    "path/filepath"
	"net"
	"net/http"
	grpcServer "echofs/internal/grpc"
	"echofs/internal/storage"
	"echofs/pkg/aws"
	"github.com/soheilhy/cmux"
)

type Worker struct {
	WorkerID string
	WorkerStatus string
	Port int
	Metrics WorkerMetrics
	StoragePath string

}

type WorkerMetrics struct {
	AvailableSpace int64
	CurrentLoad int64
}

func setConfig() Worker{
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "worker1"
	}

	portStr :=	os.Getenv("PORT")
	if portStr == "" {
		portStr = "9081"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("invalid port")
		port = 9081
	}
	return Worker{
		WorkerID: workerID,
		WorkerStatus: "starting",
		Port: port,
		Metrics: WorkerMetrics{},
		StoragePath: "",
	}
}

func SetStoragePath(workerID string) string {
	storagePath := filepath.Join("./storage", workerID,"chunks")
	err := os.MkdirAll(storagePath,0755)

	if err != nil {
		log.Fatalf("Faledd to create Storage Directory")
	}

	return storagePath
}

func main() {
	worker := setConfig()
	worker.StoragePath = SetStoragePath(worker.WorkerID)

	fmt.Printf("Starting %s on port %d\n", worker.WorkerID, worker.Port)
    fmt.Printf("Storage path: %s\n", worker.StoragePath)

	// Set up AWS and storage
	ctx := context.Background()
	awsConfig, err := aws.NewAWSConfig(ctx, "us-east-1", "")
	var s3Storage *storage.S3Storage
	if err != nil {
		fmt.Printf("Warning: Failed to initialize AWS config: %v. Using simulation mode.\n", err)
	} else {
		s3Storage = storage.NewS3Storage(awsConfig.S3, awsConfig.S3BucketName)

		if err := s3Storage.EnsureBucket(ctx); err != nil {
			fmt.Printf("Warning: Failed to ensure S3 bucket: %v. Using simulation mode.\n", err)
			s3Storage = nil
		} else {
			fmt.Printf("✅ S3 storage initialized with bucket: %s\n", awsConfig.S3BucketName)
		}
	}

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", worker.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create connection multiplexer
	m := cmux.New(lis)
	
	// Match gRPC connections
	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	
	// Match HTTP connections
	httpL := m.Match(cmux.HTTP1Fast())

	// Set up gRPC server
	logger := log.New(os.Stdout, fmt.Sprintf("[gRPC-%s] ", worker.WorkerID), log.LstdFlags)
	grpcSrv := grpcServer.NewWorkerGRPCServer(worker.WorkerID, s3Storage, logger)
	
	// Set up HTTP server
	router := setupRoutes()
	httpServer := &http.Server{Handler: router}

	// Start servers
	go func() {
		fmt.Printf("Starting gRPC server on port %d\n", worker.Port)
		if err := grpcSrv.ServeGRPC(grpcL); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	go func() {
		fmt.Printf("Starting HTTP server on port %d\n", worker.Port)
		if err := httpServer.Serve(httpL); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	fmt.Printf("Worker server (HTTP + gRPC) listening on port %d\n", worker.Port)
	
	// Start the multiplexer
	if err := m.Serve(); err != nil {
		log.Fatalf("Multiplexer failed: %v", err)
	}
}