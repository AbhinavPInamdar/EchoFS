package main

import (
	"context"
	"encoding/json"
	"net/http"
	"log"
	"fmt"
	"time"
	"io"
	"os"
	"path/filepath"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"echofs/cmd/master/core"
	fileops "echofs/pkg/fileops/Chunker"
	"echofs/pkg/fileops/Compressor"
	"echofs/internal/storage"
)

type Server struct {
	masterNode *core.MasterNode 
	router     *mux.Router
	logger     *log.Logger
	chunkStore *storage.FSChunkStore
}


type InitUploadRequest struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	UserID   string `json:"user_id"`
}

type InitUploadResponse struct {
	SessionID        string                  `json:"session_id"`
	ChunkSize        int64                   `json:"chunk_size"`
	TotalChunks      int                     `json:"total_chunks"`
	ChunkAssignments []core.ChunkAssignment  `json:"chunk_assignments"`
}

type UploadChunkRequest struct {
	SessionID  string `json:"session_id"`
	ChunkIndex int    `json:"chunk_index"`
	MD5Hash    string `json:"md5_hash"`
}

type CompleteUploadRequest struct {
	SessionID string `json:"session_id"`
	MD5Hash   string `json:"file_md5_hash"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MockWorkerRegistry struct{}
type MockChunkPlacer struct{}
type MockSessionManager struct{}

func (m *MockWorkerRegistry) GetHealthyWorkers(ctx context.Context) ([]*core.WorkerNode, error) {
	return []*core.WorkerNode{
		{ID: "worker-1", Address: "localhost", Port: 8081, Status: core.WorkerStatusOnline},
		{ID: "worker-2", Address: "localhost", Port: 8082, Status: core.WorkerStatusOnline},
		{ID: "worker-3", Address: "localhost", Port: 8083, Status: core.WorkerStatusOnline},
	}, nil
}

func (m *MockChunkPlacer) PlaceChunk(ctx context.Context, fileID string, chunkIndex int) ([]string, error) {
	workers := []string{"worker-1", "worker-2", "worker-3"}
	replicationFactor := 2
	
	var assigned []string
	for i := 0; i < replicationFactor; i++ {
		workerIndex := (chunkIndex + i) % len(workers)
		assigned = append(assigned, workers[workerIndex])
	}
	return assigned, nil
}



func NewServer(masterNode *core.MasterNode, logger *log.Logger) *Server {
	chunkStore, err := storage.NewFSChunkStore("")
	if err != nil {
		logger.Fatalf("Failed to create chunk store: %v", err)
	}
	
	s := &Server{
		masterNode: masterNode,
		logger:     logger,
		router:     mux.NewRouter(),
		chunkStore: chunkStore,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	api.HandleFunc("/files/upload", s.UploadFile).Methods("POST")
	api.HandleFunc("/files/{fileId}/download", s.DownloadFile).Methods("GET")
	
	api.HandleFunc("/files/upload/init", s.InitUpload).Methods("POST")
	api.HandleFunc("/files/upload/chunk", s.UploadChunk).Methods("POST")
	api.HandleFunc("/files/upload/complete", s.CompleteUpload).Methods("POST")
	
	api.HandleFunc("/workers/register", s.RegisterWorker).Methods("POST")
	api.HandleFunc("/workers/{workerId}/heartbeat", s.WorkerHeartbeat).Methods("POST")
	
	api.HandleFunc("/health", s.HealthCheck).Methods("GET")
}

func (s *Server) Start(port int) error {
	s.logger.Printf("Starting server on port %d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s.router)
}

func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("UploadFile called - integrated chunking and compression")
	w.Header().Set("Content-Type", "application/json")
	
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		s.sendErrorResponse(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}
	
	file, header, err := r.FormFile("file")
	if err != nil {
		s.sendErrorResponse(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	userID := r.FormValue("user_id")
	if userID == "" {
		s.sendErrorResponse(w, "user_id is required", http.StatusBadRequest)
		return
	}
	
	sessionID := uuid.New().String()
	fileID := uuid.New().String()
	
	tempDir := filepath.Join(os.TempDir(), "echofs", sessionID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		s.sendErrorResponse(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)
	
	tempFilePath := filepath.Join(tempDir, header.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		s.sendErrorResponse(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	
	fileSize, err := io.Copy(tempFile, file)
	if err != nil {
		tempFile.Close()
		s.sendErrorResponse(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	tempFile.Close()
	
	s.logger.Printf("Compressing file: %s", header.Filename)
	compressedFile, err := compressor.Compress(tempFilePath)
	if err != nil {
		s.sendErrorResponse(w, "Failed to compress file", http.StatusInternalServerError)
		return
	}
	defer compressedFile.Close()
	
	compressedPath := tempFilePath + ".gz"
	
	s.logger.Printf("Chunking compressed file")
	chunker := fileops.NewDefaultFileChunker(1024 * 1024) 
	
	var chunks []fileops.ChunkMeta
	if fileSize > 100*1024*1024 { 
		chunks, err = chunker.ChunkLargeFile(compressedPath)
	} else {
		chunks, err = chunker.ChunkFile(compressedPath)
	}
	
	if err != nil {
		s.sendErrorResponse(w, "Failed to chunk file", http.StatusInternalServerError)
		return
	}
	
	s.logger.Printf("Created %d chunks for file %s", len(chunks), header.Filename)
	
	mockPlacer := &MockChunkPlacer{}
	var chunkAssignments []core.ChunkAssignment
	
	for i, chunk := range chunks {
		workers, err := mockPlacer.PlaceChunk(context.Background(), fileID, i)
		if err != nil {
			s.sendErrorResponse(w, "Failed to assign chunks to workers", http.StatusInternalServerError)
			return
		}
		
		assignment := core.ChunkAssignment{
			ChunkIndex:     chunk.Index,
			PrimaryWorker:  workers[0],
			ReplicaWorkers: workers[1:],
			MD5Expected:    chunk.MD5Hash,
			Status:         "pending",
		}
		chunkAssignments = append(chunkAssignments, assignment)
	}
	
	session := &core.UploadSession{
		SessionID:       sessionID,
		UserID:          userID,
		FileName:        header.Filename,
		FileSize:        fileSize,
		ChunkSize:       int64(len(chunks)), 
		TotalChunks:     len(chunks),
		ChunkAssignment: chunkAssignments,
		Status:          core.SessionStatusActive,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	
	s.masterNode.AddUploadSession(session)
	
	s.logger.Printf("TODO: Upload %d chunks to workers", len(chunks))
	
	fileMetadata := &core.FileMetadata{
		FileID:       fileID,
		Size:         fileSize,
		OriginalName: header.Filename,
		ChunkSize:    int64(1024 * 1024), 
		TotalChunks:  len(chunks),
		UploadedBy:   userID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       "completed",
	}
	
	response := map[string]interface{}{
		"file_id":     fileID,
		"session_id":  sessionID,
		"chunks":      len(chunks),
		"compressed":  true,
		"file_size":   fileSize,
		"metadata":    fileMetadata,
	}
	
	s.sendSuccessResponse(w, "File uploaded, compressed, and chunked successfully", response)
}

func (s *Server) InitUpload(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("InitUpload called")
	w.Header().Set("Content-Type", "application/json")
	
	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.FileName == "" || req.FileSize <= 0 || req.UserID == "" {
		s.sendErrorResponse(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	sessionID := uuid.New().String()
	
	chunkSize := int64(1024 * 1024) 
	totalChunks := int(req.FileSize / chunkSize)
	if req.FileSize%chunkSize != 0 {
		totalChunks++
	}
	
	mockPlacer := &MockChunkPlacer{}
	
	var chunkAssignments []core.ChunkAssignment
	for i := 0; i < totalChunks; i++ {
		workers, err := mockPlacer.PlaceChunk(context.Background(), sessionID, i)
		if err != nil {
			s.sendErrorResponse(w, "Failed to assign chunks", http.StatusInternalServerError)
			return
		}
		
		assignment := core.ChunkAssignment{
			ChunkIndex:     i,
			PrimaryWorker:  workers[0],
			ReplicaWorkers: workers[1:],
			Status:         "pending",
		}
		chunkAssignments = append(chunkAssignments, assignment)
	}
	
	session := &core.UploadSession{
		SessionID:       sessionID,
		UserID:          req.UserID,
		FileName:        req.FileName,
		FileSize:        req.FileSize,
		ChunkSize:       chunkSize,
		TotalChunks:     totalChunks,
		ChunkAssignment: chunkAssignments,
		Status:          core.SessionStatusActive,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	
	s.masterNode.AddUploadSession(session)
	
	response := InitUploadResponse{
		SessionID:        sessionID,
		ChunkSize:        chunkSize,
		TotalChunks:      totalChunks,
		ChunkAssignments: chunkAssignments,
	}
	
	s.sendSuccessResponse(w, "Upload session created", response)
}

func (s *Server) UploadChunk(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("UploadChunk called")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "upload chunk - not implemented yet"})
}

func (s *Server) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("CompleteUpload called")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "complete upload - not implemented yet"})
}

func (s *Server) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	s.logger.Printf("DownloadFile called for fileId: %s", fileId)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "download - not implemented yet", "fileId": fileId})
}

func (s *Server) RegisterWorker(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("RegisterWorker called")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "register worker - not implemented yet"})
}

func (s *Server) WorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workerId := vars["workerId"]
	s.logger.Printf("WorkerHeartbeat called for workerId: %s", workerId)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "heartbeat - not implemented yet", "workerId": workerId})
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "echofs-master"})
}

func (s *Server) sendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Message: message,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}