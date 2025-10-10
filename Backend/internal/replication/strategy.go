package replication

import (
	"context"
	"time"

	"echofs/internal/metadata"
)

// Replicator defines the interface for different replication strategies
type Replicator interface {
	Write(ctx context.Context, obj *metadata.ObjectMeta, chunk []byte) (*WriteResult, error)
	Read(ctx context.Context, obj *metadata.ObjectMeta, chunkID string) ([]byte, error)
	GetStrategy() string
}

// WriteResult contains the result of a write operation
type WriteResult struct {
	Acked     bool      `json:"acked"`
	Version   int64     `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Replicas  int       `json:"replicas"`  // Number of replicas written
	Latency   time.Duration `json:"latency"` // Write latency
}

// ReplicationConfig holds configuration for replication strategies
type ReplicationConfig struct {
	// Sync strategy config
	QuorumSize      int           `json:"quorum_size"`       // Required replicas for quorum
	WriteTimeout    time.Duration `json:"write_timeout"`     // Timeout for sync writes
	ReplicationFactor int         `json:"replication_factor"` // Total number of replicas

	// Async strategy config
	AsyncQueueSize  int           `json:"async_queue_size"`  // Background replication queue size
	AsyncBatchSize  int           `json:"async_batch_size"`  // Batch size for async replication
	AsyncFlushInterval time.Duration `json:"async_flush_interval"` // How often to flush async queue

	// Worker configuration
	WorkerNodes     []string      `json:"worker_nodes"`      // Available worker nodes
	HealthCheckInterval time.Duration `json:"health_check_interval"` // Worker health check frequency
}

// ReplicationManager manages different replication strategies
type ReplicationManager struct {
	config       ReplicationConfig
	syncStrategy *SyncStrategy
	asyncStrategy *AsyncStrategy
	workerPool   *WorkerPool
}

func NewReplicationManager(config ReplicationConfig) *ReplicationManager {
	workerPool := NewWorkerPool(config.WorkerNodes)
	
	return &ReplicationManager{
		config:        config,
		syncStrategy:  NewSyncStrategy(config, workerPool),
		asyncStrategy: NewAsyncStrategy(config, workerPool),
		workerPool:    workerPool,
	}
}

// SelectReplicator returns the appropriate replicator based on object mode
func (rm *ReplicationManager) SelectReplicator(obj *metadata.ObjectMeta) Replicator {
	switch obj.CurrentMode {
	case "C": // Strong consistency
		return rm.syncStrategy
	case "A": // Available/Eventual consistency
		return rm.asyncStrategy
	case "Hybrid":
		// For hybrid mode, use sync for critical operations, async for others
		// This could be enhanced with more sophisticated logic
		if obj.ModeHint == "Strong" {
			return rm.syncStrategy
		}
		return rm.asyncStrategy
	default:
		// Default to strong consistency for safety
		return rm.syncStrategy
	}
}

// GetSyncStrategy returns the synchronous replication strategy
func (rm *ReplicationManager) GetSyncStrategy() *SyncStrategy {
	return rm.syncStrategy
}

// GetAsyncStrategy returns the asynchronous replication strategy
func (rm *ReplicationManager) GetAsyncStrategy() *AsyncStrategy {
	return rm.asyncStrategy
}

// GetWorkerPool returns the worker pool
func (rm *ReplicationManager) GetWorkerPool() *WorkerPool {
	return rm.workerPool
}

// UpdateConfig updates the replication configuration
func (rm *ReplicationManager) UpdateConfig(config ReplicationConfig) {
	rm.config = config
	rm.syncStrategy.UpdateConfig(config)
	rm.asyncStrategy.UpdateConfig(config)
	rm.workerPool.UpdateNodes(config.WorkerNodes)
}

// GetStats returns replication statistics
func (rm *ReplicationManager) GetStats() ReplicationStats {
	return ReplicationStats{
		SyncStats:  rm.syncStrategy.GetStats(),
		AsyncStats: rm.asyncStrategy.GetStats(),
		WorkerStats: rm.workerPool.GetStats(),
	}
}

// ReplicationStats contains statistics for all replication strategies
type ReplicationStats struct {
	SyncStats   SyncStats   `json:"sync_stats"`
	AsyncStats  AsyncStats  `json:"async_stats"`
	WorkerStats WorkerStats `json:"worker_stats"`
}

// SyncStats contains statistics for synchronous replication
type SyncStats struct {
	TotalWrites     int64         `json:"total_writes"`
	SuccessfulWrites int64        `json:"successful_writes"`
	FailedWrites    int64         `json:"failed_writes"`
	AverageLatency  time.Duration `json:"average_latency"`
	QuorumFailures  int64         `json:"quorum_failures"`
}

// AsyncStats contains statistics for asynchronous replication
type AsyncStats struct {
	TotalWrites      int64         `json:"total_writes"`
	QueuedWrites     int64         `json:"queued_writes"`
	ProcessedWrites  int64         `json:"processed_writes"`
	FailedWrites     int64         `json:"failed_writes"`
	QueueSize        int           `json:"current_queue_size"`
	AverageLatency   time.Duration `json:"average_latency"`
}

// WorkerStats contains statistics for worker nodes
type WorkerStats struct {
	TotalNodes    int                    `json:"total_nodes"`
	HealthyNodes  int                    `json:"healthy_nodes"`
	NodeLatencies map[string]time.Duration `json:"node_latencies"`
	NodeErrors    map[string]int64       `json:"node_errors"`
}