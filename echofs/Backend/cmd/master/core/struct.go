package core

import (
	"context"
	"sync"
	"time"
)

type WorkerStatus int

const (
	WorkerStatusUnknown WorkerStatus = iota
	WorkerStatusOnline
	WorkerStatusOffline
	WorkerStatusDraining
	WorkerStatusFailed
)

type WorkerNode struct {
	ID               string        `json:"id"`
	Address          string        `json:"address"`
	Port             int           `json:"port"`
	Status           WorkerStatus  `json:"status"`
	LastHeartbeat    time.Time     `json:"last_heartbeat"`
	TotalStorage     int64         `json:"total_storage"`
	UsedStorage      int64         `json:"used_storage"`
	AvailableStorage int64         `json:"available_storage"`
	CPUUsage         float64       `json:"cpu_usage"`
	MemoryUsage      float64       `json:"memory_usage"`
	Version          string        `json:"version"`
	Capabilities     []string      `json:"capabilities"`

	mutex          sync.Mutex      `json:"-"`
	ConnectionPool interface{}     `json:"-"` 
}

type SessionStatus int

const (
	SessionStatusPending SessionStatus = iota
	SessionStatusActive
	SessionStatusCompleted
	SessionStatusFailed
	SessionStatusExpired
)

type ChunkAssignment struct {
	ChunkIndex     int      `json:"chunk_index"`
	PrimaryWorker  string   `json:"primary_worker"`
	ReplicaWorkers []string `json:"replica_workers"`
	MD5Expected    string   `json:"md5_expected"`
	Status         string   `json:"status"` 
}

type UploadSession struct {
	SessionID       string             `json:"session_id"`
	UserID          string             `json:"user_id"`
	FileName        string             `json:"file_name"`
	FileSize        int64              `json:"file_size"`
	ChunkSize       int64              `json:"chunk_size"`
	TotalChunks     int                `json:"total_chunks"`
	ChunkAssignment []ChunkAssignment  `json:"chunk_assignment"`
	UploadChunks    []int              `json:"upload_chunks"` 
	Status          SessionStatus      `json:"status"`
	CreatedAt       time.Time          `json:"created_at"`
	ExpiresAt       time.Time          `json:"expires_at"`

	mutex     sync.RWMutex          `json:"-"`
	ChunkChan chan ChunkComplete    `json:"-"`
	ErrorChan chan error            `json:"-"`
}

type ChunkComplete struct {
	ChunkIndex int    `json:"chunk_index"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

type FileMetadata struct {
	FileID        string    `json:"file_id"`
	Size          int64     `json:"size"`
	OriginalName  string    `json:"original_name"`
	ChunkSize     int64     `json:"chunk_size"`
	TotalChunks   int       `json:"total_chunks"`
	MD5Hash       string    `json:"md5_hash"`
	UploadedBy    string    `json:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Status        string    `json:"status"` 
}

type ChunkMetadata struct {
	ChunkID     string    `json:"chunk_id"`
	FileID      string    `json:"file_id"`
	ChunkIndex  int       `json:"chunk_index"`
	Size        int64     `json:"size"`
	MD5Hash     string    `json:"md5_hash"`
	WorkerNodes []string  `json:"worker_nodes"` 
	CreatedAt   time.Time `json:"created_at"`
}
type WorkerRegistry interface {
	RegisterWorker(ctx context.Context, worker *WorkerNode) error
	DeregisterWorker(ctx context.Context, workerID string) error
	GetWorker(ctx context.Context, workerID string) (*WorkerNode, error)
	GetHealthyWorkers(ctx context.Context) ([]*WorkerNode, error)
	UpdateWorkerStatus(ctx context.Context, workerID string, status WorkerStatus) error
	GetWorkerLoad(ctx context.Context, workerID string) (float64, error)
}

type MetadataStore interface {
	SaveFileMetadata(ctx context.Context, metadata *FileMetadata) error
	GetFileMetadata(ctx context.Context, fileID string) (*FileMetadata, error)
	DeleteFileMetadata(ctx context.Context, fileID string) error
	SaveChunkMetadata(ctx context.Context, metadata *ChunkMetadata) error
	GetChunkMetadata(ctx context.Context, fileID string) ([]*ChunkMetadata, error)
	DeleteChunkMetadata(ctx context.Context, chunkID string) error
}

type SessionManager interface {
	CreateUploadSession(ctx context.Context, session *UploadSession) error
	GetUploadSession(ctx context.Context, sessionID string) (*UploadSession, error)
	UpdateUploadSession(ctx context.Context, session *UploadSession) error
	DeleteUploadSession(ctx context.Context, sessionID string) error
	CleanupExpiredSessions(ctx context.Context) error
}

type ChunkPlacer interface {
	PlaceChunk(ctx context.Context, fileID string, chunkIndex int) ([]string, error)
	RebalanceChunk(ctx context.Context, chunkID string) ([]string, error)
	GetOptimalWorkers(ctx context.Context, count int) ([]*WorkerNode, error)
}

type HealthChecker interface {
	CheckWorkerHealth(ctx context.Context, workerID string) error
	StartHealthChecking(ctx context.Context) error
	StopHealthChecking(ctx context.Context) error
}




