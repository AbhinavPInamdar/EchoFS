package consistency

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	grpcClient "echofs/internal/grpc"
	"echofs/internal/metrics"
)

// WriteCoordinator executes writes according to the consistency mode.
// In Strong mode: quorum writes (N/2+1 must ack).
// In Available mode: write to one node, return immediately, replicate async.
type WriteCoordinator struct {
	workerRegistry    *grpcClient.WorkerRegistry
	replicationFactor int
	logger            interface{ Printf(string, ...interface{}) }

	// Async replication queue
	asyncQueue chan *asyncReplicationTask
	stopCh     chan struct{}
	wg         sync.WaitGroup

	// Stats
	strongWrites    int64
	availableWrites int64
	quorumFailures  int64
	asyncEnqueued   int64
	asyncCompleted  int64
	asyncFailed     int64
}

type asyncReplicationTask struct {
	fileID     string
	chunkID    string
	chunkIndex int
	data       []byte
	md5Hash    string
	workers    []string // worker IDs to replicate to
	retries    int
	createdAt  time.Time
}

// ChunkWriteResult represents the outcome of writing a single chunk.
type ChunkWriteResult struct {
	ChunkIndex    int
	PrimaryWorker string
	Replicas      []string
	Mode          Mode
	Success       bool
	Error         error
	Latency       time.Duration
}

// NewWriteCoordinator creates a coordinator with the given replication factor.
func NewWriteCoordinator(
	registry *grpcClient.WorkerRegistry,
	replicationFactor int,
	logger interface{ Printf(string, ...interface{}) },
) *WriteCoordinator {
	wc := &WriteCoordinator{
		workerRegistry:    registry,
		replicationFactor: replicationFactor,
		logger:            logger,
		asyncQueue:        make(chan *asyncReplicationTask, 1000),
		stopCh:            make(chan struct{}),
	}

	// Start async replication workers
	for i := 0; i < 3; i++ {
		wc.wg.Add(1)
		go wc.asyncReplicationWorker()
	}

	return wc
}

// WriteChunk writes a chunk according to the specified consistency mode.
func (wc *WriteCoordinator) WriteChunk(
	ctx context.Context,
	mode Mode,
	fileID string,
	chunkID string,
	chunkIndex int,
	data []byte,
	md5Hash string,
) *ChunkWriteResult {
	switch mode {
	case ModeStrong:
		return wc.writeStrong(ctx, fileID, chunkID, chunkIndex, data, md5Hash)
	case ModeAvailable:
		return wc.writeAvailable(ctx, fileID, chunkID, chunkIndex, data, md5Hash)
	case ModeHybrid:
		// Hybrid: write to 2 nodes (less than full quorum but more than 1)
		return wc.writeHybrid(ctx, fileID, chunkID, chunkIndex, data, md5Hash)
	default:
		return wc.writeStrong(ctx, fileID, chunkID, chunkIndex, data, md5Hash)
	}
}

// writeStrong performs a quorum write: sends to all replicas, waits for majority ack.
func (wc *WriteCoordinator) writeStrong(
	ctx context.Context,
	fileID, chunkID string,
	chunkIndex int,
	data []byte,
	md5Hash string,
) *ChunkWriteResult {
	start := time.Now()
	atomic.AddInt64(&wc.strongWrites, 1)

	workers := wc.selectWorkers(chunkIndex)
	if len(workers) == 0 {
		return &ChunkWriteResult{
			ChunkIndex: chunkIndex,
			Mode:       ModeStrong,
			Success:    false,
			Error:      fmt.Errorf("no workers available"),
			Latency:    time.Since(start),
		}
	}

	quorumSize := len(workers)/2 + 1
	if quorumSize > len(workers) {
		quorumSize = len(workers)
	}

	type writeResp struct {
		workerID string
		err      error
	}

	respCh := make(chan writeResp, len(workers))

	// Send to all target workers in parallel
	writeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for _, workerID := range workers {
		go func(wID string) {
			client, exists := wc.workerRegistry.GetWorker(wID)
			if !exists {
				respCh <- writeResp{workerID: wID, err: fmt.Errorf("worker %s not found", wID)}
				return
			}
			_, err := client.StoreChunk(writeCtx, fileID, chunkID, chunkIndex, data, md5Hash)
			respCh <- writeResp{workerID: wID, err: err}
		}(workerID)
	}

	// Wait for quorum
	var successWorkers []string
	var firstErr error
	for i := 0; i < len(workers); i++ {
		select {
		case resp := <-respCh:
			if resp.err == nil {
				successWorkers = append(successWorkers, resp.workerID)
				if len(successWorkers) >= quorumSize {
					latency := time.Since(start)
					wc.recordWriteMetrics("strong", latency, true)
					return &ChunkWriteResult{
						ChunkIndex:    chunkIndex,
						PrimaryWorker: successWorkers[0],
						Replicas:      successWorkers[1:],
						Mode:          ModeStrong,
						Success:       true,
						Latency:       latency,
					}
				}
			} else {
				if firstErr == nil {
					firstErr = resp.err
				}
			}
		case <-writeCtx.Done():
			atomic.AddInt64(&wc.quorumFailures, 1)
			latency := time.Since(start)
			wc.recordWriteMetrics("strong", latency, false)
			return &ChunkWriteResult{
				ChunkIndex: chunkIndex,
				Mode:       ModeStrong,
				Success:    false,
				Error:      fmt.Errorf("quorum timeout: got %d/%d acks", len(successWorkers), quorumSize),
				Latency:    latency,
			}
		}
	}

	// Didn't reach quorum
	atomic.AddInt64(&wc.quorumFailures, 1)
	latency := time.Since(start)
	wc.recordWriteMetrics("strong", latency, false)
	return &ChunkWriteResult{
		ChunkIndex: chunkIndex,
		Mode:       ModeStrong,
		Success:    false,
		Error:      fmt.Errorf("quorum failed: got %d/%d acks: %v", len(successWorkers), quorumSize, firstErr),
		Latency:    latency,
	}
}

// writeAvailable writes to a single node and enqueues async replication.
func (wc *WriteCoordinator) writeAvailable(
	ctx context.Context,
	fileID, chunkID string,
	chunkIndex int,
	data []byte,
	md5Hash string,
) *ChunkWriteResult {
	start := time.Now()
	atomic.AddInt64(&wc.availableWrites, 1)

	workers := wc.selectWorkers(chunkIndex)
	if len(workers) == 0 {
		return &ChunkWriteResult{
			ChunkIndex: chunkIndex,
			Mode:       ModeAvailable,
			Success:    false,
			Error:      fmt.Errorf("no workers available"),
			Latency:    time.Since(start),
		}
	}

	// Write to primary only
	primaryWorker := workers[0]
	client, exists := wc.workerRegistry.GetWorker(primaryWorker)
	if !exists {
		return &ChunkWriteResult{
			ChunkIndex: chunkIndex,
			Mode:       ModeAvailable,
			Success:    false,
			Error:      fmt.Errorf("primary worker %s not found", primaryWorker),
			Latency:    time.Since(start),
		}
	}

	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := client.StoreChunk(writeCtx, fileID, chunkID, chunkIndex, data, md5Hash)
	if err != nil {
		latency := time.Since(start)
		wc.recordWriteMetrics("available", latency, false)
		return &ChunkWriteResult{
			ChunkIndex:    chunkIndex,
			PrimaryWorker: primaryWorker,
			Mode:          ModeAvailable,
			Success:       false,
			Error:         fmt.Errorf("primary write failed: %w", err),
			Latency:       latency,
		}
	}

	// Enqueue async replication to remaining workers
	if len(workers) > 1 {
		task := &asyncReplicationTask{
			fileID:     fileID,
			chunkID:    chunkID,
			chunkIndex: chunkIndex,
			data:       data,
			md5Hash:    md5Hash,
			workers:    workers[1:],
			retries:    0,
			createdAt:  time.Now(),
		}
		select {
		case wc.asyncQueue <- task:
			atomic.AddInt64(&wc.asyncEnqueued, 1)
		default:
			wc.logger.Printf("Warning: async replication queue full, dropping replication for chunk %s", chunkID)
		}
	}

	latency := time.Since(start)
	wc.recordWriteMetrics("available", latency, true)
	return &ChunkWriteResult{
		ChunkIndex:    chunkIndex,
		PrimaryWorker: primaryWorker,
		Replicas:      workers[1:],
		Mode:          ModeAvailable,
		Success:       true,
		Latency:       latency,
	}
}

// writeHybrid writes to 2 nodes synchronously (between quorum and single).
func (wc *WriteCoordinator) writeHybrid(
	ctx context.Context,
	fileID, chunkID string,
	chunkIndex int,
	data []byte,
	md5Hash string,
) *ChunkWriteResult {
	start := time.Now()

	workers := wc.selectWorkers(chunkIndex)
	if len(workers) == 0 {
		return &ChunkWriteResult{
			ChunkIndex: chunkIndex,
			Mode:       ModeHybrid,
			Success:    false,
			Error:      fmt.Errorf("no workers available"),
			Latency:    time.Since(start),
		}
	}

	// Write to min(2, available workers) synchronously
	syncCount := 2
	if syncCount > len(workers) {
		syncCount = len(workers)
	}

	type writeResp struct {
		workerID string
		err      error
	}
	respCh := make(chan writeResp, syncCount)

	writeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, workerID := range workers[:syncCount] {
		go func(wID string) {
			client, exists := wc.workerRegistry.GetWorker(wID)
			if !exists {
				respCh <- writeResp{workerID: wID, err: fmt.Errorf("worker %s not found", wID)}
				return
			}
			_, err := client.StoreChunk(writeCtx, fileID, chunkID, chunkIndex, data, md5Hash)
			respCh <- writeResp{workerID: wID, err: err}
		}(workerID)
	}

	var successWorkers []string
	for i := 0; i < syncCount; i++ {
		select {
		case resp := <-respCh:
			if resp.err == nil {
				successWorkers = append(successWorkers, resp.workerID)
			}
		case <-writeCtx.Done():
			break
		}
	}

	if len(successWorkers) == 0 {
		latency := time.Since(start)
		wc.recordWriteMetrics("hybrid", latency, false)
		return &ChunkWriteResult{
			ChunkIndex: chunkIndex,
			Mode:       ModeHybrid,
			Success:    false,
			Error:      fmt.Errorf("hybrid write failed: no nodes acked"),
			Latency:    latency,
		}
	}

	// Async replicate to remaining workers
	if len(workers) > syncCount {
		task := &asyncReplicationTask{
			fileID:     fileID,
			chunkID:    chunkID,
			chunkIndex: chunkIndex,
			data:       data,
			md5Hash:    md5Hash,
			workers:    workers[syncCount:],
			createdAt:  time.Now(),
		}
		select {
		case wc.asyncQueue <- task:
			atomic.AddInt64(&wc.asyncEnqueued, 1)
		default:
			// Queue full, log and continue
		}
	}

	latency := time.Since(start)
	wc.recordWriteMetrics("hybrid", latency, true)
	return &ChunkWriteResult{
		ChunkIndex:    chunkIndex,
		PrimaryWorker: successWorkers[0],
		Replicas:      successWorkers[1:],
		Mode:          ModeHybrid,
		Success:       true,
		Latency:       latency,
	}
}

// selectWorkers picks which workers should store a given chunk.
// Uses consistent assignment based on chunk index.
func (wc *WriteCoordinator) selectWorkers(chunkIndex int) []string {
	allWorkers := wc.workerRegistry.GetAllWorkers()
	workerIDs := make([]string, 0, len(allWorkers))
	for id := range allWorkers {
		workerIDs = append(workerIDs, id)
	}

	if len(workerIDs) == 0 {
		return nil
	}

	// Rotate starting position based on chunk index for distribution
	count := wc.replicationFactor
	if count > len(workerIDs) {
		count = len(workerIDs)
	}

	result := make([]string, 0, count)
	startIdx := chunkIndex % len(workerIDs)
	for i := 0; i < count; i++ {
		idx := (startIdx + i) % len(workerIDs)
		result = append(result, workerIDs[idx])
	}

	return result
}

// asyncReplicationWorker processes the async replication queue.
func (wc *WriteCoordinator) asyncReplicationWorker() {
	defer wc.wg.Done()

	for {
		select {
		case task := <-wc.asyncQueue:
			wc.processAsyncTask(task)
		case <-wc.stopCh:
			return
		}
	}
}

func (wc *WriteCoordinator) processAsyncTask(task *asyncReplicationTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	successCount := 0
	for _, workerID := range task.workers {
		client, exists := wc.workerRegistry.GetWorker(workerID)
		if !exists {
			continue
		}
		_, err := client.StoreChunk(ctx, task.fileID, task.chunkID, task.chunkIndex, task.data, task.md5Hash)
		if err == nil {
			successCount++
		} else {
			wc.logger.Printf("Async replication failed for chunk %s to worker %s: %v", task.chunkID, workerID, err)
		}
	}

	if successCount > 0 {
		atomic.AddInt64(&wc.asyncCompleted, 1)
	} else if task.retries < 3 {
		// Retry
		task.retries++
		select {
		case wc.asyncQueue <- task:
		default:
			atomic.AddInt64(&wc.asyncFailed, 1)
		}
	} else {
		atomic.AddInt64(&wc.asyncFailed, 1)
		wc.logger.Printf("Async replication permanently failed for chunk %s after %d retries", task.chunkID, task.retries)
	}
}

func (wc *WriteCoordinator) recordWriteMetrics(strategy string, latency time.Duration, success bool) {
	if metrics.AppMetrics != nil {
		status := "success"
		if !success {
			status = "failure"
		}
		metrics.AppMetrics.ReplicationLatency.WithLabelValues(strategy, "write").Observe(latency.Seconds())
		if !success {
			metrics.AppMetrics.QuorumFailures.WithLabelValues(status).Inc()
		}
	}
}

// GetStats returns current write coordinator statistics.
func (wc *WriteCoordinator) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"strong_writes":    atomic.LoadInt64(&wc.strongWrites),
		"available_writes": atomic.LoadInt64(&wc.availableWrites),
		"quorum_failures":  atomic.LoadInt64(&wc.quorumFailures),
		"async_enqueued":   atomic.LoadInt64(&wc.asyncEnqueued),
		"async_completed":  atomic.LoadInt64(&wc.asyncCompleted),
		"async_failed":     atomic.LoadInt64(&wc.asyncFailed),
		"async_queue_size": len(wc.asyncQueue),
	}
}

// Stop gracefully shuts down the write coordinator.
func (wc *WriteCoordinator) Stop() {
	close(wc.stopCh)
	wc.wg.Wait()
}
