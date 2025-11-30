package metadata

import (
	"time"
)

type FileMetadata struct {
	FileID       string    `json:"file_id"`
	Size         int64     `json:"size"`
	OriginalName string    `json:"original_name"`
	ChunkSize    int       `json:"chunk_size"`
	TotalChunks  int       `json:"total_chunks"`
	MD5Hash      string    `json:"md5_hash"`
	UploadedBy   string    `json:"uploaded_by"` // User ID
	OwnerID      string    `json:"owner_id"`    // User ID for access control
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Status       string    `json:"status"`
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
	UploadedBy     string               `json:"uploaded_by"`
	Status         string               `json:"status"`
}

func NewObjectMeta(fileID, name, uploadedBy string, size int64) *ObjectMeta {
	now := time.Now()
	return &ObjectMeta{
		FileID:         fileID,
		Name:           name,
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
		UploadedBy:     uploadedBy,
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