package replication

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Worker struct {
	ID       string        `json:"id"`
	Address  string        `json:"address"`
	Healthy  bool          `json:"healthy"`
	LastSeen time.Time     `json:"last_seen"`
	Latency  time.Duration `json:"latency"`
	Errors   int64         `json:"errors"`
	mu       sync.RWMutex
}

type WorkerPool struct {
	workers map[string]*Worker
	mu      sync.RWMutex
	
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
	
	for i, addr := range nodeAddresses {
		workerID := fmt.Sprintf("worker%d", i+1)
		pool.workers[workerID] = &Worker{
			ID:       workerID,
			Address:  addr,
			Healthy:  true,
			LastSeen: time.Now(),
			Latency:  0,
			Errors:   0,
		}
	}
	
	pool.startHealthChecking()
	
	return pool
}

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
	
	result := make([]*Worker, count)
	copy(result, healthyWorkers[:count])
	
	return result, nil
}

func (wp *WorkerPool) GetWorker(workerID string) (*Worker, error) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	worker, exists := wp.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker %s not found", workerID)
	}
	
	return worker, nil
}

func (wp *WorkerPool) UpdateNodes(nodeAddresses []string) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	newWorkers := make(map[string]*Worker)
	
	for i, addr := range nodeAddresses {
		workerID := fmt.Sprintf("worker%d", i+1)
		
		if existingWorker, exists := wp.workers[workerID]; exists {
			existingWorker.Address = addr
			newWorkers[workerID] = existingWorker
		} else {

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

func (wp *WorkerPool) performHealthChecks() {
	wp.mu.RLock()
	workers := make([]*Worker, 0, len(wp.workers))
	for _, worker := range wp.workers {
		workers = append(workers, worker)
	}
	wp.mu.RUnlock()
	
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

func (wp *WorkerPool) checkWorkerHealth(worker *Worker) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	startTime := time.Now()
	
	healthy := wp.pingWorker(ctx, worker)
	latency := time.Since(startTime)
	
	if healthy {
		wp.MarkWorkerHealthy(worker.ID, latency)
	} else {
		wp.MarkWorkerUnhealthy(worker.ID, fmt.Errorf("health check failed"))
	}
}

func (wp *WorkerPool) pingWorker(ctx context.Context, worker *Worker) bool {

	worker.mu.RLock()
	errors := atomic.LoadInt64(&worker.Errors)
	worker.mu.RUnlock()
	
	return errors < 5
}

func (wp *WorkerPool) Stop() {
	close(wp.stopHealthCheck)
	wp.healthCheckWG.Wait()
}

func (w *Worker) WriteChunk(ctx context.Context, objectID string, data []byte, version int64) error {

	w.mu.RLock()
	healthy := w.Healthy
	w.mu.RUnlock()
	
	if !healthy {
		return fmt.Errorf("worker %s is unhealthy", w.ID)
	}
	
	select {
	case <-time.After(time.Millisecond * 5):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) ReadChunk(ctx context.Context, chunkID string) ([]byte, error) {

	w.mu.RLock()
	healthy := w.Healthy
	w.mu.RUnlock()
	
	if !healthy {
		return nil, fmt.Errorf("worker %s is unhealthy", w.ID)
	}
	
	select {
	case <-time.After(time.Millisecond * 3):
		return []byte("dummy chunk data"), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}