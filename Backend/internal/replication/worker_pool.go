package replication

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Worker represents a worker node in the cluster
type Worker struct {
	ID       string        `json:"id"`
	Address  string        `json:"address"`
	Healthy  bool          `json:"healthy"`
	LastSeen time.Time     `json:"last_seen"`
	Latency  time.Duration `json:"latency"`
	Errors   int64         `json:"errors"`
	mu       sync.RWMutex
}

// WorkerPool manages a pool of worker nodes
type WorkerPool struct {
	workers map[string]*Worker
	mu      sync.RWMutex
	
	// Health checking
	healthCheckInterval time.Duration
	stopHealthCheck     chan struct{}
	healthCheckWG       sync.WaitGroup
}

func NewWorkerPool(nodeAddresses []string) *WorkerPool {
	pool := &WorkerPool{
		workers:             make(map[string]*Worker),
		healthCheckInterval: 30 * time.Second,
		stopHealthCheck:     make(chan struct{}),
	}
	
	// Initialize workers
	for i, addr := range nodeAddresses {
		workerID := fmt.Sprintf("worker%d", i+1)
		pool.workers[workerID] = &Worker{
			ID:       workerID,
			Address:  addr,
			Healthy:  true, // Assume healthy initially
			LastSeen: time.Now(),
			Latency:  0,
			Errors:   0,
		}
	}
	
	// Start health checking
	pool.startHealthChecking()
	
	return pool
}

// SelectWorkers selects the specified number of healthy workers
func (wp *WorkerPool) SelectWorkers(count int) ([]*Worker, error) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	var healthyWorkers []*Worker
	for _, worker := range wp.workers {
		if worker.Healthy {
			healthyWorkers = append(healthyWorkers, worker)
		}
	}
	
	if len(healthyWorkers) < count {
		return nil, fmt.Errorf("insufficient healthy workers: need %d, have %d", count, len(healthyWorkers))
	}
	
	// Sort by latency (lowest first) and return the best ones
	// For simplicity, just return the first 'count' workers
	// In a real implementation, you'd sort by latency/load
	result := make([]*Worker, count)
	copy(result, healthyWorkers[:count])
	
	return result, nil
}

// GetWorker returns a specific worker by ID
func (wp *WorkerPool) GetWorker(workerID string) (*Worker, error) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	worker, exists := wp.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker %s not found", workerID)
	}
	
	return worker, nil
}

// UpdateNodes updates the worker pool with new node addresses
func (wp *WorkerPool) UpdateNodes(nodeAddresses []string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	// Create new worker map
	newWorkers := make(map[string]*Worker)
	
	for i, addr := range nodeAddresses {
		workerID := fmt.Sprintf("worker%d", i+1)
		
		// Preserve existing worker state if it exists
		if existingWorker, exists := wp.workers[workerID]; exists {
			existingWorker.Address = addr
			newWorkers[workerID] = existingWorker
		} else {
			// Create new worker
			newWorkers[workerID] = &Worker{
				ID:       workerID,
				Address:  addr,
				Healthy:  true,
				LastSeen: time.Now(),
				Latency:  0,
				Errors:   0,
			}
		}
	}
	
	wp.workers = newWorkers
}

// MarkWorkerUnhealthy marks a worker as unhealthy
func (wp *WorkerPool) MarkWorkerUnhealthy(workerID string, err error) {
	wp.mu.RLock()
	worker, exists := wp.workers[workerID]
	wp.mu.RUnlock()
	
	if !exists {
		return
	}
	
	worker.mu.Lock()
	defer worker.mu.Unlock()
	
	worker.Healthy = false
	atomic.AddInt64(&worker.Errors, 1)
}

// MarkWorkerHealthy marks a worker as healthy
func (wp *WorkerPool) MarkWorkerHealthy(workerID string, latency time.Duration) {
	wp.mu.RLock()
	worker, exists := wp.workers[workerID]
	wp.mu.RUnlock()
	
	if !exists {
		return
	}
	
	worker.mu.Lock()
	defer worker.mu.Unlock()
	
	worker.Healthy = true
	worker.LastSeen = time.Now()
	worker.Latency = latency
}

// GetStats returns statistics about the worker pool
func (wp *WorkerPool) GetStats() WorkerStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	stats := WorkerStats{
		TotalNodes:    len(wp.workers),
		HealthyNodes:  0,
		NodeLatencies: make(map[string]time.Duration),
		NodeErrors:    make(map[string]int64),
	}
	
	for _, worker := range wp.workers {
		worker.mu.RLock()
		if worker.Healthy {
			stats.HealthyNodes++
		}
		stats.NodeLatencies[worker.ID] = worker.Latency
		stats.NodeErrors[worker.ID] = atomic.LoadInt64(&worker.Errors)
		worker.mu.RUnlock()
	}
	
	return stats
}

// startHealthChecking starts the background health checking routine
func (wp *WorkerPool) startHealthChecking() {
	wp.healthCheckWG.Add(1)
	go func() {
		defer wp.healthCheckWG.Done()
		
		ticker := time.NewTicker(wp.healthCheckInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				wp.performHealthChecks()
			case <-wp.stopHealthCheck:
				return
			}
		}
	}()
}

// performHealthChecks checks the health of all workers
func (wp *WorkerPool) performHealthChecks() {
	wp.mu.RLock()
	workers := make([]*Worker, 0, len(wp.workers))
	for _, worker := range wp.workers {
		workers = append(workers, worker)
	}
	wp.mu.RUnlock()
	
	// Check each worker concurrently
	var wg sync.WaitGroup
	for _, worker := range workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			wp.checkWorkerHealth(w)
		}(worker)
	}
	wg.Wait()
}

// checkWorkerHealth performs a health check on a single worker
func (wp *WorkerPool) checkWorkerHealth(worker *Worker) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	startTime := time.Now()
	
	// Perform health check (simplified - just check if we can connect)
	// In a real implementation, this would make an actual health check call
	healthy := wp.pingWorker(ctx, worker)
	latency := time.Since(startTime)
	
	if healthy {
		wp.MarkWorkerHealthy(worker.ID, latency)
	} else {
		wp.MarkWorkerUnhealthy(worker.ID, fmt.Errorf("health check failed"))
	}
}

// pingWorker performs a simple ping to check if worker is responsive
func (wp *WorkerPool) pingWorker(ctx context.Context, worker *Worker) bool {
	// Simplified health check - in reality this would make a gRPC health check
	// For now, assume workers are healthy unless they've had too many errors
	worker.mu.RLock()
	errors := atomic.LoadInt64(&worker.Errors)
	worker.mu.RUnlock()
	
	// Mark unhealthy if too many errors
	return errors < 5
}

// Stop stops the worker pool and health checking
func (wp *WorkerPool) Stop() {
	close(wp.stopHealthCheck)
	wp.healthCheckWG.Wait()
}

// Worker methods for chunk operations

// WriteChunk writes a chunk to this worker
func (w *Worker) WriteChunk(ctx context.Context, objectID string, data []byte, version int64) error {
	// Simulate write operation
	// In a real implementation, this would make a gRPC call to the worker
	
	w.mu.RLock()
	healthy := w.Healthy
	w.mu.RUnlock()
	
	if !healthy {
		return fmt.Errorf("worker %s is unhealthy", w.ID)
	}
	
	// Simulate some latency
	select {
	case <-time.After(time.Millisecond * 5):
		return nil // Success
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReadChunk reads a chunk from this worker
func (w *Worker) ReadChunk(ctx context.Context, chunkID string) ([]byte, error) {
	// Simulate read operation
	// In a real implementation, this would make a gRPC call to the worker
	
	w.mu.RLock()
	healthy := w.Healthy
	w.mu.RUnlock()
	
	if !healthy {
		return nil, fmt.Errorf("worker %s is unhealthy", w.ID)
	}
	
	// Simulate some latency and return dummy data
	select {
	case <-time.After(time.Millisecond * 3):
		return []byte("dummy chunk data"), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}