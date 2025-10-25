package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Simple in-memory implementations for local testing

type InMemoryWorkerRegistry struct {
	workers map[string]*WorkerNode
	mutex   sync.RWMutex
	logger  *log.Logger
}

func NewInMemoryWorkerRegistry(logger *log.Logger) *InMemoryWorkerRegistry {
	return &InMemoryWorkerRegistry{
		workers: make(map[string]*WorkerNode),
		logger:  logger,
	}
}

func (wr *InMemoryWorkerRegistry) RegisterWorker(ctx context.Context, worker *WorkerNode) error {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()
	
	wr.workers[worker.ID] = worker
	wr.logger.Printf("Worker %s registered at %s:%d", worker.ID, worker.Address, worker.Port)
	return nil
}

func (wr *InMemoryWorkerRegistry) DeregisterWorker(ctx context.Context, workerID string) error {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()
	
	delete(wr.workers, workerID)
	wr.logger.Printf("Worker %s deregistered", workerID)
	return nil
}

func (wr *InMemoryWorkerRegistry) GetWorker(ctx context.Context, workerID string) (*WorkerNode, error) {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()
	
	worker, exists := wr.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker %s not found", workerID)
	}
	return worker, nil
}

func (wr *InMemoryWorkerRegistry) GetHealthyWorkers(ctx context.Context) ([]*WorkerNode, error) {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()
	
	var healthy []*WorkerNode
	for _, worker := range wr.workers {
		if worker.Status == WorkerStatusOnline {
			healthy = append(healthy, worker)
		}
	}
	return healthy, nil
}

func (wr *InMemoryWorkerRegistry) UpdateWorkerStatus(ctx context.Context, workerID string, status WorkerStatus) error {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()
	
	if worker, exists := wr.workers[workerID]; exists {
		worker.Status = status
		worker.LastHeartbeat = time.Now()
	}
	return nil
}

func (wr *InMemoryWorkerRegistry) GetWorkerLoad(ctx context.Context, workerID string) (float64, error) {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()
	
	if worker, exists := wr.workers[workerID]; exists {
		return worker.CPUUsage, nil
	}
	return 0, fmt.Errorf("worker %s not found", workerID)
}

// InMemoryMetadataStore
type InMemoryMetadataStore struct {
	files  map[string]*FileMetadata
	chunks map[string]*ChunkMetadata
	mutex  sync.RWMutex
	logger *log.Logger
}

func NewInMemoryMetadataStore(logger *log.Logger) *InMemoryMetadataStore {
	return &InMemoryMetadataStore{
		files:  make(map[string]*FileMetadata),
		chunks: make(map[string]*ChunkMetadata),
		logger: logger,
	}
}

func (ms *InMemoryMetadataStore) SaveFileMetadata(ctx context.Context, metadata *FileMetadata) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.files[metadata.FileID] = metadata
	return nil
}

func (ms *InMemoryMetadataStore) GetFileMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	metadata, exists := ms.files[fileID]
	if !exists {
		return nil, fmt.Errorf("file %s not found", fileID)
	}
	return metadata, nil
}

func (ms *InMemoryMetadataStore) DeleteFileMetadata(ctx context.Context, fileID string) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	delete(ms.files, fileID)
	return nil
}

func (ms *InMemoryMetadataStore) SaveChunkMetadata(ctx context.Context, metadata *ChunkMetadata) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	ms.chunks[metadata.ChunkID] = metadata
	return nil
}

func (ms *InMemoryMetadataStore) GetChunkMetadata(ctx context.Context, fileID string) ([]*ChunkMetadata, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	var chunks []*ChunkMetadata
	for _, chunk := range ms.chunks {
		if chunk.FileID == fileID {
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func (ms *InMemoryMetadataStore) DeleteChunkMetadata(ctx context.Context, chunkID string) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	delete(ms.chunks, chunkID)
	return nil
}

// InMemorySessionManager
type InMemorySessionManager struct {
	sessions map[string]*UploadSession
	mutex    sync.RWMutex
	logger   *log.Logger
}

func NewInMemorySessionManager(logger *log.Logger) *InMemorySessionManager {
	return &InMemorySessionManager{
		sessions: make(map[string]*UploadSession),
		logger:   logger,
	}
}

func (sm *InMemorySessionManager) CreateUploadSession(ctx context.Context, session *UploadSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	sm.sessions[session.SessionID] = session
	return nil
}

func (sm *InMemorySessionManager) GetUploadSession(ctx context.Context, sessionID string) (*UploadSession, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return session, nil
}

func (sm *InMemorySessionManager) UpdateUploadSession(ctx context.Context, session *UploadSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	sm.sessions[session.SessionID] = session
	return nil
}

func (sm *InMemorySessionManager) DeleteUploadSession(ctx context.Context, sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	delete(sm.sessions, sessionID)
	return nil
}

func (sm *InMemorySessionManager) CleanupExpiredSessions(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	now := time.Now()
	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, sessionID)
		}
	}
	return nil
}

// SimpleChunkPlacer
type SimpleChunkPlacer struct {
	workerRegistry WorkerRegistry
	logger         *log.Logger
}

func NewSimpleChunkPlacer(workerRegistry WorkerRegistry, logger *log.Logger) *SimpleChunkPlacer {
	return &SimpleChunkPlacer{
		workerRegistry: workerRegistry,
		logger:         logger,
	}
}

func (cp *SimpleChunkPlacer) PlaceChunk(ctx context.Context, fileID string, chunkIndex int) ([]string, error) {
	workers, err := cp.workerRegistry.GetHealthyWorkers(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(workers) == 0 {
		return nil, fmt.Errorf("no healthy workers available")
	}
	
	// Simple placement: use first available worker
	return []string{workers[0].ID}, nil
}

func (cp *SimpleChunkPlacer) RebalanceChunk(ctx context.Context, chunkID string) ([]string, error) {
	// Simple implementation: no rebalancing
	return nil, nil
}

func (cp *SimpleChunkPlacer) GetOptimalWorkers(ctx context.Context, count int) ([]*WorkerNode, error) {
	workers, err := cp.workerRegistry.GetHealthyWorkers(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(workers) < count {
		return workers, nil
	}
	
	return workers[:count], nil
}

// SimpleHealthChecker
type SimpleHealthChecker struct {
	workerRegistry WorkerRegistry
	logger         *log.Logger
	ticker         *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewSimpleHealthChecker(workerRegistry WorkerRegistry, logger *log.Logger) *SimpleHealthChecker {
	return &SimpleHealthChecker{
		workerRegistry: workerRegistry,
		logger:         logger,
	}
}

func (hc *SimpleHealthChecker) CheckWorkerHealth(ctx context.Context, workerID string) error {
	// Simple implementation: assume all workers are healthy
	return nil
}

func (hc *SimpleHealthChecker) StartHealthChecking(ctx context.Context) error {
	hc.ctx, hc.cancel = context.WithCancel(ctx)
	hc.ticker = time.NewTicker(30 * time.Second)
	
	go func() {
		for {
			select {
			case <-hc.ctx.Done():
				return
			case <-hc.ticker.C:
				hc.performHealthCheck()
			}
		}
	}()
	
	return nil
}

func (hc *SimpleHealthChecker) StopHealthChecking(ctx context.Context) error {
	if hc.cancel != nil {
		hc.cancel()
	}
	if hc.ticker != nil {
		hc.ticker.Stop()
	}
	return nil
}

func (hc *SimpleHealthChecker) performHealthCheck() {
	// Simple implementation: log that health check is running
	hc.logger.Println("Performing health check on workers")
}