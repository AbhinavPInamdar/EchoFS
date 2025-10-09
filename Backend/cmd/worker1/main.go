package main

import (
    "context"
    "fmt"
    "os"
    "log"
    "strconv"
    "path/filepath"
	"net/http"
	grpc "echofs/internal/grpc"
	"echofs/internal/storage"
	"echofs/pkg/aws"
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

	portStr :=	os.Getenv("WORKER_PORT")
	if portStr == "" {
		portStr = "8081"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("invalid port")
		port = 8081
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


	router := setupRoutes()
	go func() {
		fmt.Printf("Worker HTTP server listening on port %d\n", worker.Port)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", worker.Port), router))
	}()

	grpcPort := worker.Port + 1000
	fmt.Printf("Worker gRPC server listening on port %d\n", grpcPort)
	

	ctx := context.Background()
	awsConfig, err := aws.NewAWSConfig(ctx, "us-east-1", "", "")
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


	logger := log.New(os.Stdout, fmt.Sprintf("[gRPC-%s] ", worker.WorkerID), log.LstdFlags)
	grpcServer := grpc.NewWorkerGRPCServer(worker.WorkerID, s3Storage, logger)
	
	go func() {
		logger.Printf("Starting gRPC server on port %d", grpcPort)
		if err := grpcServer.StartGRPCServer(grpcPort); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()
	

	select {}
}