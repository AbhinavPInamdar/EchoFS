package replication

import (
	"context"
	"time"

	"echofs/internal/metadata"
)

type Replicator interface {
	Write(ctx context.Context, obj *metadata.ObjectMeta, chunk []byte) (*WriteResult, error)
	Read(ctx context.Context, obj *metadata.ObjectMeta, chunkID string) ([]byte, error)
	GetStrategy() string
}

type WriteResult struct {
	Acked     bool      `json:"acked"`
	Version   int64     `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Replicas  int       `json:"replicas"`
	Latency   time.Duration `json:"latency"`
}

type ReplicationConfig struct {

	QuorumSize      int           `json:"quorum_size"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	ReplicationFactor int         `json:"replication_factor"`

	AsyncQueueSize  int           `json:"async_queue_size"`
	AsyncBatchSize  int           `json:"async_batch_size"`
	AsyncFlushInterval time.Duration `json:"async_flush_interval"`

	WorkerNodes     []string      `json:"worker_nodes"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

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

func (rm *ReplicationManager) SelectReplicator(obj *metadata.ObjectMeta) Replicator {
	switch obj.CurrentMode {
	case "C":
		return rm.syncStrategy
	case "A":
		return rm.asyncStrategy
	case "Hybrid":

		if obj.ModeHint == "Strong" {
			return rm.syncStrategy
		}
		return rm.asyncStrategy
	default:

		return rm.syncStrategy
	}
}

func (rm *ReplicationManager) GetSyncStrategy() *SyncStrategy {
	return rm.syncStrategy
}

func (rm *ReplicationManager) GetAsyncStrategy() *AsyncStrategy {
	return rm.asyncStrategy
}

func (rm *ReplicationManager) GetWorkerPool() *WorkerPool {
	return rm.workerPool
}

func (rm *ReplicationManager) UpdateConfig(config ReplicationConfig) {
	rm.config = config
	rm.syncStrategy.UpdateConfig(config)
	rm.asyncStrategy.UpdateConfig(config)
	rm.workerPool.UpdateNodes(config.WorkerNodes)
}

func (rm *ReplicationManager) GetStats() ReplicationStats {
	return ReplicationStats{
		SyncStats:  rm.syncStrategy.GetStats(),
		AsyncStats: rm.asyncStrategy.GetStats(),
		WorkerStats: rm.workerPool.GetStats(),
	}
}

type ReplicationStats struct {
	SyncStats   SyncStats   `json:"sync_stats"`
	AsyncStats  AsyncStats  `json:"async_stats"`
	WorkerStats WorkerStats `json:"worker_stats"`
}

type SyncStats struct {
	TotalWrites     int64         `json:"total_writes"`
	SuccessfulWrites int64        `json:"successful_writes"`
	FailedWrites    int64         `json:"failed_writes"`
	AverageLatency  time.Duration `json:"average_latency"`
	QuorumFailures  int64         `json:"quorum_failures"`
}

type AsyncStats struct {
	TotalWrites      int64         `json:"total_writes"`
	QueuedWrites     int64         `json:"queued_writes"`
	ProcessedWrites  int64         `json:"processed_writes"`
	FailedWrites     int64         `json:"failed_writes"`
	QueueSize        int           `json:"current_queue_size"`
	AverageLatency   time.Duration `json:"average_latency"`
}

type WorkerStats struct {
	TotalNodes    int                    `json:"total_nodes"`
	HealthyNodes  int                    `json:"healthy_nodes"`
	NodeLatencies map[string]time.Duration `json:"node_latencies"`
	NodeErrors    map[string]int64       `json:"node_errors"`
}