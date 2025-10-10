package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"echofs/internal/controller"
	"echofs/internal/metadata"
	"echofs/internal/replication"
)

// ConsistencyHandler handles consistency-related API endpoints
type ConsistencyHandler struct {
	controllerClient *controller.Client
	replicationMgr   *replication.ReplicationManager
}

// UploadRequest represents an upload request with consistency parameters
type UploadRequest struct {
	UserID      string `json:"user_id"`
	Consistency string `json:"consistency"` // "auto", "strong", "available"
	Hint        string `json:"hint"`        // Optional hint for controller
}

// UploadResponse represents the response from an upload operation
type UploadResponse struct {
	Success     bool                  `json:"success"`
	Message     string                `json:"message"`
	Data        *UploadResponseData   `json:"data,omitempty"`
	Consistency *ConsistencyInfo      `json:"consistency,omitempty"`
}

type UploadResponseData struct {
	FileID      string    `json:"file_id"`
	Chunks      int       `json:"chunks"`
	Compressed  bool      `json:"compressed"`
	FileSize    int64     `json:"file_size"`
	Metadata    *metadata.ObjectMeta `json:"metadata"`
	SessionID   string    `json:"session_id"`
}

type ConsistencyInfo struct {
	ModeUsed    string    `json:"mode_used"`    // "C", "A", "Hybrid"
	Version     int64     `json:"version"`
	Replicas    int       `json:"replicas"`
	Latency     string    `json:"latency"`
	Timestamp   time.Time `json:"timestamp"`
}

func NewConsistencyHandler(controllerClient *controller.Client, replicationMgr *replication.ReplicationManager) *ConsistencyHandler {
	return &ConsistencyHandler{
		controllerClient: controllerClient,
		replicationMgr:   replicationMgr,
	}
}

// HandleUploadWithConsistency handles file uploads with consistency parameters
func (ch *ConsistencyHandler) HandleUploadWithConsistency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get consistency parameters
	userID := r.FormValue("user_id")
	consistency := r.FormValue("consistency")
	hint := r.FormValue("hint")

	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Default consistency mode
	if consistency == "" {
		consistency = "auto"
	}

	// Validate consistency parameter
	validModes := map[string]bool{
		"auto":      true,
		"strong":    true,
		"available": true,
	}
	if !validModes[consistency] {
		http.Error(w, "Invalid consistency mode. Must be auto, strong, or available", http.StatusBadRequest)
		return
	}

	// Read file data
	fileData := make([]byte, header.Size)
	_, err = file.Read(fileData)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Create object metadata
	fileID := generateFileID()
	objMeta := metadata.NewObjectMeta(fileID, header.Filename, userID, header.Size)
	
	// Set hint if provided
	if hint != "" {
		objMeta.ModeHint = hint
	}

	// Determine consistency mode
	var actualMode string
	switch consistency {
	case "strong":
		actualMode = "C"
		objMeta.CurrentMode = "C"
	case "available":
		actualMode = "A"
		objMeta.CurrentMode = "A"
	case "auto":
		// Query controller for recommended mode
		mode, err := ch.controllerClient.GetMode(r.Context(), fileID)
		if err != nil {
			// Fallback to strong consistency if controller unavailable
			actualMode = "C"
			objMeta.CurrentMode = "C"
		} else {
			actualMode = mode.Mode
			objMeta.CurrentMode = mode.Mode
		}
	}

	// Select appropriate replicator
	replicator := ch.replicationMgr.SelectReplicator(objMeta)

	// Perform write operation
	startTime := time.Now()
	writeResult, err := replicator.Write(r.Context(), objMeta, fileData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Write failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Update object metadata with write result
	objMeta.LastVersion = writeResult.Version
	objMeta.UpdateVectorClock("master") // Update vector clock for master node
	objMeta.LastSyncTs = writeResult.Timestamp

	// Prepare response
	response := UploadResponse{
		Success: true,
		Message: "File uploaded successfully with " + replicator.GetStrategy() + " replication",
		Data: &UploadResponseData{
			FileID:     fileID,
			Chunks:     1, // Simplified - single chunk
			Compressed: false,
			FileSize:   header.Size,
			Metadata:   objMeta,
			SessionID:  generateSessionID(),
		},
		Consistency: &ConsistencyInfo{
			ModeUsed:  actualMode,
			Version:   writeResult.Version,
			Replicas:  writeResult.Replicas,
			Latency:   writeResult.Latency.String(),
			Timestamp: writeResult.Timestamp,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleGetConsistencyMode returns the current consistency mode for an object
func (ch *ConsistencyHandler) HandleGetConsistencyMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	objectID := r.URL.Query().Get("object_id")
	if objectID == "" {
		http.Error(w, "object_id parameter required", http.StatusBadRequest)
		return
	}

	// Query controller for current mode
	mode, err := ch.controllerClient.GetMode(r.Context(), objectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get mode: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mode)
}

// HandleSetConsistencyHint sets a consistency hint for an object
func (ch *ConsistencyHandler) HandleSetConsistencyHint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ObjectID string `json:"object_id"`
		Hint     string `json:"hint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ObjectID == "" || req.Hint == "" {
		http.Error(w, "object_id and hint are required", http.StatusBadRequest)
		return
	}

	// Set hint via controller
	err := ch.controllerClient.SetHint(r.Context(), req.ObjectID, req.Hint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to set hint: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"status":    "success",
		"object_id": req.ObjectID,
		"hint":      req.Hint,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleReplicationStats returns replication statistics
func (ch *ConsistencyHandler) HandleReplicationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := ch.replicationMgr.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Utility functions

func generateFileID() string {
	// In a real implementation, use UUID or similar
	return fmt.Sprintf("file_%d", time.Now().UnixNano())
}

func generateSessionID() string {
	// In a real implementation, use UUID or similar
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}