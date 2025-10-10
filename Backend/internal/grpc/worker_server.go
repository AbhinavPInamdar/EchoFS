package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"echofs/internal/storage"
	"echofs/internal/metrics"
	pb "echofs/proto/v1"
	"google.golang.org/grpc"
)


type WorkerGRPCServer struct {
	pb.UnimplementedWorkerServiceServer
	workerID    string
	s3Storage   *storage.S3Storage
	logger      *log.Logger
}


func NewWorkerGRPCServer(workerID string, s3Storage *storage.S3Storage, logger *log.Logger) *WorkerGRPCServer {
	return &WorkerGRPCServer{
		workerID:  workerID,
		s3Storage: s3Storage,
		logger:    logger,
	}
}


func (w *WorkerGRPCServer) StoreChunk(ctx context.Context, req *pb.StoreChunkRequest) (*pb.StoreChunkResponse, error) {
	start := time.Now()
	w.logger.Printf("gRPC StoreChunk called: fileID=%s, chunkID=%s, index=%d", 
		req.GetFileId(), req.GetChunkId(), req.GetChunkIndex())
	
	// Record chunk processing metrics
	if metrics.AppMetrics != nil {
		defer func() {
			duration := time.Since(start)
			chunkSize := int64(len(req.GetChunkData()))
			metrics.AppMetrics.RecordChunkProcessing(chunkSize, duration)
		}()
	}

	if w.s3Storage != nil {
		err := w.s3Storage.StoreChunk(ctx, req.GetFileId(), req.GetChunkId(), int(req.GetChunkIndex()), req.GetChunkData())
		if err != nil {
			return &pb.StoreChunkResponse{
				Success:  false,
				Message:  fmt.Sprintf("Failed to store chunk: %v", err),
				WorkerId: w.workerID,
			}, nil
		}

		s3Key := fmt.Sprintf("files/%s/chunks/%s_%d", req.GetFileId(), req.GetChunkId(), req.GetChunkIndex())
		return &pb.StoreChunkResponse{
			Success:  true,
			Message:  "Chunk stored successfully in S3",
			WorkerId: w.workerID,
			S3Key:    s3Key,
		}, nil
	}


	return &pb.StoreChunkResponse{
		Success:  true,
		Message:  "Chunk stored successfully (simulated)",
		WorkerId: w.workerID,
	}, nil
}


func (w *WorkerGRPCServer) RetrieveChunk(ctx context.Context, req *pb.RetrieveChunkRequest) (*pb.RetrieveChunkResponse, error) {
	w.logger.Printf("gRPC RetrieveChunk called: fileID=%s, chunkID=%s, index=%d", 
		req.GetFileId(), req.GetChunkId(), req.GetChunkIndex())

	if w.s3Storage != nil {
		data, err := w.s3Storage.RetrieveChunk(ctx, req.GetFileId(), req.GetChunkId(), int(req.GetChunkIndex()))
		if err != nil {
			return &pb.RetrieveChunkResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to retrieve chunk: %v", err),
			}, nil
		}

		return &pb.RetrieveChunkResponse{
			Success:   true,
			ChunkData: data,
			Message:   "Chunk retrieved successfully from S3",
		}, nil
	}


	return &pb.RetrieveChunkResponse{
		Success:   true,
		ChunkData: []byte("simulated chunk data"),
		Message:   "Chunk retrieved successfully (simulated)",
	}, nil
}


func (w *WorkerGRPCServer) DeleteChunk(ctx context.Context, req *pb.DeleteChunkRequest) (*pb.DeleteChunkResponse, error) {
	w.logger.Printf("gRPC DeleteChunk called: fileID=%s, chunkID=%s, index=%d", 
		req.GetFileId(), req.GetChunkId(), req.GetChunkIndex())

	if w.s3Storage != nil {
		err := w.s3Storage.DeleteChunk(ctx, req.GetFileId(), req.GetChunkId(), int(req.GetChunkIndex()))
		if err != nil {
			return &pb.DeleteChunkResponse{
				Success:  false,
				Message:  fmt.Sprintf("Failed to delete chunk: %v", err),
				WorkerId: w.workerID,
			}, nil
		}
	}

	return &pb.DeleteChunkResponse{
		Success:  true,
		Message:  "Chunk deleted successfully",
		WorkerId: w.workerID,
	}, nil
}


func (w *WorkerGRPCServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Healthy:   true,
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}, nil
}


func (w *WorkerGRPCServer) GetStatus(ctx context.Context, req *pb.WorkerStatusRequest) (*pb.WorkerStatusResponse, error) {
	return &pb.WorkerStatusResponse{
		WorkerId:       w.workerID,
		Address:        "localhost",
		Port:           8081,
		AvailableSpace: 1000000000,
		CurrentLoad:    0,
		Status:         "online",
		LastHeartbeat:  time.Now().Unix(),
	}, nil
}


func (w *WorkerGRPCServer) StartGRPCServer(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	// Create gRPC server with metrics interceptors
	s := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor()),
		grpc.StreamInterceptor(metrics.StreamServerInterceptor()),
	)
	pb.RegisterWorkerServiceServer(s, w)

	w.logger.Printf("Worker gRPC server listening on port %d", port)
	return s.Serve(lis)
}