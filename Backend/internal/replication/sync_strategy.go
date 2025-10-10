package replication

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"echofs/internal/metadata"
)

// SyncStrategy implements synchronous replication with quorum writes
type SyncStrategy struct {
	config     ReplicationConfig
	workerPool *WorkerPool
	stats      SyncStats
	mu         sync.RWMutex
}

func NewSyncStrategy(config ReplicationConfig, workerPool *WorkerPool) *SyncStrategy {
	return &SyncStrategy{
		config:     config,
		workerPool: workerPool,
		stats:      SyncStats{},
	}
}

func (s *SyncStrategy) Write(ctx context.Context, obj *metadata.ObjectMeta, chunk []byte) (*WriteResult, error) {
	startTime := time.Now()
	atomic.AddInt64(&s.stats.TotalWrites, 1)

	// Create write context with timeout
	writeCtx, cancel := context.WithTimeout(ctx, s.config.WriteTimeout)
	defer cancel()

	// Select worker nodes for replication
	workers, err := s.workerPool.SelectWorkers(s.config.ReplicationFactor)
	if err != nil {
		atomic.AddInt64(&s.stats.FailedWrites, 1)
		return nil, fmt.Errorf("failed to select workers: %w", err)
	}

	// Perform quorum write
	result, err := s.performQuorumWrite(writeCtx, obj, chunk, workers)
	
	// Update statistics
	latency := time.Since(startTime)
	s.updateLatencyStats(latency)
	
	if err != nil {
		atomic.AddInt64(&s.stats.FailedWrites, 1)
		return nil, err
	}

	atomic.AddInt64(&s.stats.SuccessfulWrites, 1)
	result.Latency = latency
	return result, nil
}

func (s *SyncStrategy) Read(ctx context.Context, obj *metadata.ObjectMeta, chunkID string) ([]byte, error) {
	// For reads, we can read from any replica (read-your-writes consistency)
	// In a full implementation, this might involve version checking
	
	workers, err := s.workerPool.SelectWorkers(1) // Just need one healthy worker
	if err != nil {
		return nil, fmt.Errorf("no healthy workers available: %w", err)
	}

	// Try to read from the first available worker
	for _, worker := range workers {
		data, err := worker.ReadChunk(ctx, chunkID)
		if err == nil {
			return data, nil
		}
		// Log error and try next worker
	}

	return nil, fmt.Errorf("failed to read chunk from any replica")
}

func (s *SyncStrategy) performQuorumWrite(ctx context.Context, obj *metadata.ObjectMeta, chunk []byte, workers []*Worker) (*WriteResult, error) {
	type writeResponse struct {
		worker *Worker
		err    error
		latency time.Duration
	}

	// Channel to collect responses
	responses := make(chan writeResponse, len(workers))
	
	// Generate new version for this write
	newVersion := obj.LastVersion + 1
	
	// Start writes to all replicas concurrently
	for _, worker := range workers {
		go func(w *Worker) {
			startTime := time.Now()
			err := w.WriteChunk(ctx, obj.FileID, chunk, newVersion)
			responses <- writeResponse{
				worker:  w,
				err:     err,
				latency: time.Since(startTime),
			}
		}(worker)
	}

	// Collect responses and wait for quorum
	var successCount int
	var totalLatency time.Duration
	var firstError error

	for i := 0; i < len(workers); i++ {
		select {
		case resp := <-responses:
			totalLatency += resp.latency
			if resp.err == nil {
				successCount++
			} else if firstError == nil {
				firstError = resp.err
			}
			
			// Check if we have quorum
			if successCount >= s.config.QuorumSize {
				// We have quorum, can return success
				avgLatency := totalLatency / time.Duration(i+1)
				return &WriteResult{
					Acked:     true,
					Version:   newVersion,
					Timestamp: time.Now(),
					Replicas:  successCount,
					Latency:   avgLatency,
				}, nil
			}
			
			// Check if quorum is impossible
			remaining := len(workers) - i - 1
			if successCount + remaining < s.config.QuorumSize {
				// Quorum impossible, fail fast
				break
			}

		case <-ctx.Done():
			atomic.AddInt64(&s.stats.QuorumFailures, 1)
			return nil, fmt.Errorf("write timeout: %w", ctx.Err())
		}
	}

	// Quorum not achieved
	atomic.AddInt64(&s.stats.QuorumFailures, 1)
	if firstError != nil {
		return nil, fmt.Errorf("quorum write failed (got %d/%d): %w", 
			successCount, s.config.QuorumSize, firstError)
	}
	
	return nil, fmt.Errorf("quorum write failed: only %d/%d replicas succeeded", 
		successCount, s.config.QuorumSize)
}

func (s *SyncStrategy) updateLatencyStats(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Simple moving average (in a real implementation, use a more sophisticated approach)
	totalWrites := atomic.LoadInt64(&s.stats.TotalWrites)
	if totalWrites == 1 {
		s.stats.AverageLatency = latency
	} else {
		// Exponential moving average with alpha = 0.1
		alpha := 0.1
		s.stats.AverageLatency = time.Duration(
			float64(s.stats.AverageLatency)*(1-alpha) + float64(latency)*alpha,
		)
	}
}

func (s *SyncStrategy) GetStrategy() string {
	return "sync"
}

func (s *SyncStrategy) GetStats() SyncStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return SyncStats{
		TotalWrites:      atomic.LoadInt64(&s.stats.TotalWrites),
		SuccessfulWrites: atomic.LoadInt64(&s.stats.SuccessfulWrites),
		FailedWrites:     atomic.LoadInt64(&s.stats.FailedWrites),
		AverageLatency:   s.stats.AverageLatency,
		QuorumFailures:   atomic.LoadInt64(&s.stats.QuorumFailures),
	}
}

func (s *SyncStrategy) UpdateConfig(config ReplicationConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// Reset clears all statistics (useful for testing)
func (s *SyncStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	atomic.StoreInt64(&s.stats.TotalWrites, 0)
	atomic.StoreInt64(&s.stats.SuccessfulWrites, 0)
	atomic.StoreInt64(&s.stats.FailedWrites, 0)
	atomic.StoreInt64(&s.stats.QuorumFailures, 0)
	s.stats.AverageLatency = 0
}