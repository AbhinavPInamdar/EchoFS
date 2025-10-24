package replication

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"echofs/internal/metadata"
)

type AsyncStrategy struct {
	config       ReplicationConfig
	workerPool   *WorkerPool
	stats        AsyncStats
	mu           sync.RWMutex
	
	replicationQueue chan *ReplicationTask
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

type ReplicationTask struct {
	ObjectID    string
	ChunkID     string
	Data        []byte
	Version     int64
	TargetNodes []*Worker
	Timestamp   time.Time
	Retries     int
}

func NewAsyncStrategy(config ReplicationConfig, workerPool *WorkerPool) *AsyncStrategy {
	strategy := &AsyncStrategy{
		config:           config,
		workerPool:       workerPool,
		stats:            AsyncStats{},
		replicationQueue: make(chan *ReplicationTask, config.AsyncQueueSize),
		stopCh:          make(chan struct{}),
	}
	
	strategy.startReplicationWorkers()
	
	return strategy
}

func (a *AsyncStrategy) Write(ctx context.Context, obj *metadata.ObjectMeta, chunk []byte) (*WriteResult, error) {
	startTime := time.Now()
	atomic.AddInt64(&a.stats.TotalWrites, 1)

	workers, err := a.workerPool.SelectWorkers(1)
	if err != nil {
		atomic.AddInt64(&a.stats.FailedWrites, 1)
		return nil, fmt.Errorf("no workers available: %w", err)
	}
	
	primaryWorker := workers[0]
	newVersion := obj.LastVersion + 1
	
	err = primaryWorker.WriteChunk(ctx, obj.FileID, chunk, newVersion)
	if err != nil {
		atomic.AddInt64(&a.stats.FailedWrites, 1)
		return nil, fmt.Errorf("primary write failed: %w", err)
	}
	
	replicaWorkers, err := a.workerPool.SelectWorkers(a.config.ReplicationFactor - 1)
	if err == nil && len(replicaWorkers) > 0 {
		task := &ReplicationTask{
			ObjectID:    obj.FileID,
			ChunkID:     fmt.Sprintf("%s_chunk", obj.FileID),
			Data:        chunk,
			Version:     newVersion,
			TargetNodes: replicaWorkers,
			Timestamp:   time.Now(),
			Retries:     0,
		}
		
		select {
		case a.replicationQueue <- task:
			atomic.AddInt64(&a.stats.QueuedWrites, 1)
		default:

		}
	}
	
	latency := time.Since(startTime)
	a.updateLatencyStats(latency)
	
	return &WriteResult{
		Acked:     true,
		Version:   newVersion,
		Timestamp: time.Now(),
		Replicas:  1,
		Latency:   latency,
	}, nil
}

func (a *AsyncStrategy) Read(ctx context.Context, obj *metadata.ObjectMeta, chunkID string) ([]byte, error) {

	workers, err := a.workerPool.SelectWorkers(a.config.ReplicationFactor)
	if err != nil {
		return nil, fmt.Errorf("no workers available: %w", err)
	}

	for _, worker := range workers {
		data, err := worker.ReadChunk(ctx, chunkID)
		if err == nil {
			return data, nil
		}

	}

	return nil, fmt.Errorf("failed to read chunk from any replica")
}

func (a *AsyncStrategy) startReplicationWorkers() {

	numWorkers := 3
	
	for i := 0; i < numWorkers; i++ {
		a.wg.Add(1)
		go a.replicationWorker()
	}
	
	a.wg.Add(1)
	go a.flushWorker()
}

func (a *AsyncStrategy) replicationWorker() {
	defer a.wg.Done()
	
	for {
		select {
		case task := <-a.replicationQueue:
			a.processReplicationTask(task)
		case <-a.stopCh:
			return
		}
	}
}

func (a *AsyncStrategy) flushWorker() {
	defer a.wg.Done()
	
	ticker := time.NewTicker(a.config.AsyncFlushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			a.flushPendingReplications()
		case <-a.stopCh:
			return
		}
	}
}

func (a *AsyncStrategy) processReplicationTask(task *ReplicationTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	successCount := 0
	
	for _, worker := range task.TargetNodes {
		err := worker.WriteChunk(ctx, task.ObjectID, task.Data, task.Version)
		if err == nil {
			successCount++
		}
	}
	
	if successCount > 0 {
		atomic.AddInt64(&a.stats.ProcessedWrites, 1)
	} else {

		if task.Retries < 3 {
			task.Retries++
			select {
			case a.replicationQueue <- task:

			default:

				atomic.AddInt64(&a.stats.FailedWrites, 1)
			}
		} else {
			atomic.AddInt64(&a.stats.FailedWrites, 1)
		}
	}
}

func (a *AsyncStrategy) flushPendingReplications() {

}

func (a *AsyncStrategy) updateLatencyStats(latency time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	totalWrites := atomic.LoadInt64(&a.stats.TotalWrites)
	if totalWrites == 1 {
		a.stats.AverageLatency = latency
	} else {

		alpha := 0.1
		a.stats.AverageLatency = time.Duration(
			float64(a.stats.AverageLatency)*(1-alpha) + float64(latency)*alpha,
		)
	}
}

func (a *AsyncStrategy) GetStrategy() string {
	return "async"
}

func (a *AsyncStrategy) GetStats() AsyncStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	return AsyncStats{
		TotalWrites:     atomic.LoadInt64(&a.stats.TotalWrites),
		QueuedWrites:    atomic.LoadInt64(&a.stats.QueuedWrites),
		ProcessedWrites: atomic.LoadInt64(&a.stats.ProcessedWrites),
		FailedWrites:    atomic.LoadInt64(&a.stats.FailedWrites),
		QueueSize:       len(a.replicationQueue),
		AverageLatency:  a.stats.AverageLatency,
	}
}

func (a *AsyncStrategy) UpdateConfig(config ReplicationConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config = config
}

func (a *AsyncStrategy) Stop() {
	close(a.stopCh)
	a.wg.Wait()
	close(a.replicationQueue)
}

func (a *AsyncStrategy) GetQueueSize() int {
	return len(a.replicationQueue)
}

func (a *AsyncStrategy) DrainQueue() {
	for {
		select {
		case task := <-a.replicationQueue:
			a.processReplicationTask(task)
		default:
			return
		}
	}
}