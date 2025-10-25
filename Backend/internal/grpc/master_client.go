package grpc

import (
	"context"
	"fmt"
	"log"
	"time"
	"strings"
	"crypto/tls"

	"echofs/internal/metrics"
	pb "echofs/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type WorkerClient struct {
	conn     *grpc.ClientConn
	client   pb.WorkerServiceClient
	workerID string
	address  string
	logger   *log.Logger
}

func NewWorkerClient(workerID, address string, logger *log.Logger) (*WorkerClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Determine if we should use TLS based on the address
	var creds credentials.TransportCredentials
	if strings.Contains(address, "onrender.com") || strings.Contains(address, ":443") {
		// Use TLS for production endpoints
		creds = credentials.NewTLS(&tls.Config{
			ServerName: strings.Split(address, ":")[0], // Extract hostname
		})
	} else {
		// Use insecure for localhost/development
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.DialContext(ctx, address, 
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(metrics.UnaryClientInterceptor()),
		grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to worker %s at %s: %v", workerID, address, err)
	}

	client := pb.NewWorkerServiceClient(conn)

	return &WorkerClient{
		conn:     conn,
		client:   client,
		workerID: workerID,
		address:  address,
		logger:   logger,
	}, nil
}

func (wc *WorkerClient) StoreChunk(ctx context.Context, fileID, chunkID string, chunkIndex int, data []byte, md5Hash string) (*pb.StoreChunkResponse, error) {
	req := &pb.StoreChunkRequest{
		FileId:     fileID,
		ChunkId:    chunkID,
		ChunkIndex: int32(chunkIndex),
		ChunkData:  data,
		Md5Hash:    md5Hash,
	}

	wc.logger.Printf("Sending chunk %s (index %d) to worker %s via gRPC", chunkID, chunkIndex, wc.workerID)

	resp, err := wc.client.StoreChunk(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to store chunk on worker %s: %v", wc.workerID, err)
	}

	return resp, nil
}

func (wc *WorkerClient) RetrieveChunk(ctx context.Context, fileID, chunkID string, chunkIndex int) (*pb.RetrieveChunkResponse, error) {
	req := &pb.RetrieveChunkRequest{
		FileId:     fileID,
		ChunkId:    chunkID,
		ChunkIndex: int32(chunkIndex),
	}

	wc.logger.Printf("Retrieving chunk %s (index %d) from worker %s via gRPC", chunkID, chunkIndex, wc.workerID)

	resp, err := wc.client.RetrieveChunk(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunk from worker %s: %v", wc.workerID, err)
	}

	return resp, nil
}

func (wc *WorkerClient) DeleteChunk(ctx context.Context, fileID, chunkID string, chunkIndex int) (*pb.DeleteChunkResponse, error) {
	req := &pb.DeleteChunkRequest{
		FileId:     fileID,
		ChunkId:    chunkID,
		ChunkIndex: int32(chunkIndex),
	}

	wc.logger.Printf("Deleting chunk %s (index %d) from worker %s via gRPC", chunkID, chunkIndex, wc.workerID)

	resp, err := wc.client.DeleteChunk(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete chunk from worker %s: %v", wc.workerID, err)
	}

	return resp, nil
}

func (wc *WorkerClient) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
	req := &pb.HealthCheckRequest{
		WorkerId: wc.workerID,
	}

	resp, err := wc.client.HealthCheck(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("health check failed for worker %s: %v", wc.workerID, err)
	}

	return resp, nil
}

func (wc *WorkerClient) GetStatus(ctx context.Context) (*pb.WorkerStatusResponse, error) {
	req := &pb.WorkerStatusRequest{
		WorkerId: wc.workerID,
	}

	resp, err := wc.client.GetStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get status from worker %s: %v", wc.workerID, err)
	}

	return resp, nil
}

func (wc *WorkerClient) Close() error {
	return wc.conn.Close()
}

type WorkerRegistry struct {
	workers map[string]*WorkerClient
	logger  *log.Logger
}

func NewWorkerRegistry(logger *log.Logger) *WorkerRegistry {
	return &WorkerRegistry{
		workers: make(map[string]*WorkerClient),
		logger:  logger,
	}
}

func (wr *WorkerRegistry) RegisterWorker(workerID, address string) error {
	client, err := NewWorkerClient(workerID, address, wr.logger)
	if err != nil {
		return err
	}

	wr.workers[workerID] = client
	wr.logger.Printf("Registered worker %s at %s via gRPC", workerID, address)
	return nil
}

func (wr *WorkerRegistry) GetWorker(workerID string) (*WorkerClient, bool) {
	client, exists := wr.workers[workerID]
	return client, exists
}

func (wr *WorkerRegistry) GetAllWorkers() map[string]*WorkerClient {
	return wr.workers
}

func (wr *WorkerRegistry) RemoveWorker(workerID string) {
	if client, exists := wr.workers[workerID]; exists {
		client.Close()
		delete(wr.workers, workerID)
		wr.logger.Printf("Removed worker %s from registry", workerID)
	}
}