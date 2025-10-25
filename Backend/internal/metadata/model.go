package metadata

import (
	"time"
)

type FileMetadata struct {
	FileID       string    `json:"file_id" dynamodb:"file_id"`
	Size         int64     `json:"size" dynamodb:"size"`
	OriginalName string    `json:"original_name" dynamodb:"original_name"`
	ChunkSize    int       `json:"chunk_size" dynamodb:"chunk_size"`
	TotalChunks  int       `json:"total_chunks" dynamodb:"total_chunks"`
	MD5Hash      string    `json:"md5_hash" dynamodb:"md5_hash"`
	OwnerID      string    `json:"owner_id" dynamodb:"owner_id"`
	UploadedBy   string    `json:"uploaded_by" dynamodb:"uploaded_by"` // Keep for backward compatibility
	CreatedAt    time.Time `json:"created_at" dynamodb:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" dynamodb:"updated_at"`
	Status       string    `json:"status" dynamodb:"status"`
}

type ChunkRef struct {
	ChunkID   string   `json:"chunk_id"`
	Index     int      `json:"index"`
	Size      int64    `json:"size"`
	Checksum  string   `json:"checksum"`
	Workers   []string `json:"workers"`
	Version   int64    `json:"version"`
}

type ObjectMeta struct {
	FileID         string               `json:"file_id"`
	Name           string               `json:"name"`
	Size           int64                `json:"size"`
	Chunks         []ChunkRef           `json:"chunks"`
	ModeHint       string               `json:"mode_hint"`
	CurrentMode    string               `json:"current_mode"`
	LastVersion    int64                `json:"last_version"`
	VectorClock    map[string]int64     `json:"vector_clock"`
	LastSyncTs     time.Time            `json:"last_sync_ts"`
	LastModeChange time.Time            `json:"last_mode_change"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	OwnerID        string               `json:"owner_id"`
	UploadedBy     string               `json:"uploaded_by"` // Keep for backward compatibility
	Status         string               `json:"status"`
}

func NewObjectMeta(fileID, name, ownerID string, size int64) *ObjectMeta {
	now := time.Now()
	return &ObjectMeta{
		FileID:         fileID,
		Name:           name,
		OwnerID:        ownerID,
		Size:           size,
		Chunks:         make([]ChunkRef, 0),
		ModeHint:       "Auto",
		CurrentMode:    "C",
		LastVersion:    0,
		VectorClock:    make(map[string]int64),
		LastSyncTs:     now,
		LastModeChange: now,
		CreatedAt:      now,
		UpdatedAt:      now,
		UploadedBy:     ownerID, // Use ownerID for backward compatibility
		Status:         "active",
	}
}

func (om *ObjectMeta) AddChunk(chunkID string, index int, size int64, checksum string, workers []string) {
	chunk := ChunkRef{
		ChunkID:  chunkID,
		Index:    index,
		Size:     size,
		Checksum: checksum,
		Workers:  workers,
		Version:  om.LastVersion + 1,
	}
	
	om.Chunks = append(om.Chunks, chunk)
	om.LastVersion++
	om.UpdatedAt = time.Now()
}

func (om *ObjectMeta) UpdateVectorClock(nodeID string) {
	if om.VectorClock == nil {
		om.VectorClock = make(map[string]int64)
	}
	om.VectorClock[nodeID]++
	om.UpdatedAt = time.Now()
}

func (om *ObjectMeta) IsNewerThan(other *ObjectMeta) bool {
	if om.VectorClock == nil || other.VectorClock == nil {
		return om.LastVersion > other.LastVersion
	}
	
	hasGreater := false
	for nodeID, clock := range om.VectorClock {
		otherClock, exists := other.VectorClock[nodeID]
		if !exists || clock > otherClock {
			hasGreater = true
		} else if clock < otherClock {
			return false
		}
	}
	
	for nodeID, otherClock := range other.VectorClock {
		if _, exists := om.VectorClock[nodeID]; !exists && otherClock > 0 {
			return false
		}
	}
	
	return hasGreater
}

func (om *ObjectMeta) HasConflictWith(other *ObjectMeta) bool {
	return !om.IsNewerThan(other) && !other.IsNewerThan(om) && om.LastVersion != other.LastVersion
}