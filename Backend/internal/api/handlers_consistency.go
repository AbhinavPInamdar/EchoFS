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

type ConsistencyHandler struct {
	controllerClient *controller.Client
	replicationMgr   *replication.ReplicationManager
}

type UploadRequest struct {
	UserID      string `json:"user_id"`
	Consistency string `json:"consistency"`
	Hint        string `json:"hint"`
}

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
	ModeUsed    string    `json:"mode_used"`
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

func (ch *ConsistencyHandler) HandleUploadWithConsistency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	userID := r.FormValue("user_id")
	consistency := r.FormValue("consistency")
	hint := r.FormValue("hint")

	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	if consistency == "" {
		consistency = "auto"
	}

	validModes := map[string]bool{
		"auto":      true,
		"strong":    true,
		"available": true,
	}
	if !validModes[consistency] {
		http.Error(w, "Invalid consistency mode. Must be auto, strong, or available", http.StatusBadRequest)
		return
	}

	fileData := make([]byte, header.Size)
	_, err = file.Read(fileData)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	fileID := generateFileID()
	objMeta := metadata.NewObjectMeta(fileID, header.Filename, userID, header.Size)
	
	if hint != "" {
		objMeta.ModeHint = hint
	}

	var actualMode string
	switch consistency {
	case "strong":
		actualMode = "C"
		objMeta.CurrentMode = "C"
	case "available":
		actualMode = "A"
		objMeta.CurrentMode = "A"
	case "auto":

		mode, err := ch.controllerClient.GetMode(r.Context(), fileID)
		if err != nil {

			actualMode = "C"
			objMeta.CurrentMode = "C"
		} else {
			actualMode = mode.Mode
			objMeta.CurrentMode = mode.Mode
		}
	}

	replicator := ch.replicationMgr.SelectReplicator(objMeta)

	_ = time.Now() // startTime for potential metrics
	writeResult, err := replicator.Write(r.Context(), objMeta, fileData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Write failed: %v", err), http.StatusInternalServerError)
		return
	}

	objMeta.LastVersion = writeResult.Version
	objMeta.UpdateVectorClock("master")
	objMeta.LastSyncTs = writeResult.Timestamp

	response := UploadResponse{
		Success: true,
		Message: "File uploaded successfully with " + replicator.GetStrategy() + " replication",
		Data: &UploadResponseData{
			FileID:     fileID,
			Chunks:     1,
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

	mode, err := ch.controllerClient.GetMode(r.Context(), objectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get mode: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mode)
}

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

func (ch *ConsistencyHandler) HandleReplicationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := ch.replicationMgr.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func generateFileID() string {

	return fmt.Sprintf("file_%d", time.Now().UnixNano())
}

func generateSessionID() string {

	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}