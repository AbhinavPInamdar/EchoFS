// Ingest service — file upload, chunking, compression, and write coordination.
// Receives files from the gateway, chunks them, queries the consistency orchestrator
// for the write mode, and coordinates writes to storage nodes.
package main

import (
	"compress/gzip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"echofs/internal/consistency"
	grpcClient "echofs/internal/grpc"
	"echofs/internal/metrics"
	fileops "echofs/pkg/fileops/Chunker"
	"echofs/pkg/fileops/Compressor"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

type IngestService struct {
	router            *mux.Router
	logger            *log.Logger
	workerRegistry    *grpcClient.WorkerRegistry
	consistencyClient *consistency.Client
	writeCoordinator  *consistency.WriteCoordinator
	metadataURL       string
}

func main() {
	logger := log.New(os.Stdout, "[INGEST] ", log.LstdFlags|log.Lshortfile)

	metrics.InitMetrics()

	// Worker registry
	workerRegistry := grpcClient.NewWorkerRegistry(logger)
	worker1URL := getEnv("WORKER1_URL", "localhost:10081")
	worker2URL := getEnv("WORKER2_URL", "localhost:10082")
	worker3URL := getEnv("WORKER3_URL", "localhost:10083")

	if err := workerRegistry.RegisterWorker("worker1", worker1URL); err != nil {
		logger.Printf("Warning: Failed to register worker1: %v", err)
	}
	if err := workerRegistry.RegisterWorker("worker2", worker2URL); err != nil {
		logger.Printf("Warning: Failed to register worker2: %v", err)
	}
	if err := workerRegistry.RegisterWorker("worker3", worker3URL); err != nil {
		logger.Printf("Warning: Failed to register worker3: %v", err)
	}

	// Consistency
	orchestratorURL := getEnv("ORCHESTRATOR_URL", "http://localhost:8082")
	consistencyClient := consistency.NewClient(orchestratorURL, logger)

	replicationFactor := 3
	fmt.Sscanf(getEnv("REPLICATION_FACTOR", "3"), "%d", &replicationFactor)
	writeCoordinator := consistency.NewWriteCoordinator(workerRegistry, replicationFactor, logger)

	svc := &IngestService{
		router:            mux.NewRouter(),
		logger:            logger,
		workerRegistry:    workerRegistry,
		consistencyClient: consistencyClient,
		writeCoordinator:  writeCoordinator,
		metadataURL:       getEnv("METADATA_URL", "http://localhost:8083"),
	}

	svc.setupRoutes()

	port := getEnv("PORT", "8081")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      svc.router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		logger.Printf("Ingest service listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Println("Shutting down ingest service...")
	writeCoordinator.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	logger.Println("Ingest service stopped")
}

func (svc *IngestService) setupRoutes() {
	svc.router.HandleFunc("/api/v1/files/upload", svc.uploadFile).Methods("POST")
	svc.router.HandleFunc("/api/v1/files/{fileId}/download", svc.downloadFile).Methods("GET")
	svc.router.HandleFunc("/health", svc.healthCheck).Methods("GET")
}

func (svc *IngestService) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "echofs-ingest"})
}

func (svc *IngestService) uploadFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Content-Type", "application/json")

	// User ID comes from gateway via header
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		sendError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		sendError(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size > 100*1024*1024 {
		sendError(w, "File too large (max 100MB)", http.StatusBadRequest)
		return
	}

	// Sanitize filename
	sanitizedName := sanitizeFilename(header.Filename)
	if sanitizedName == "" {
		sendError(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	fileID := uuid.New().String()

	// Query consistency mode
	mode, reason := svc.consistencyClient.GetMode(r.Context(), fileID)
	svc.logger.Printf("Upload %s: mode=%s reason=%s", fileID, mode, reason)

	// Register with orchestrator (async)
	go svc.consistencyClient.RegisterObject(context.Background(), fileID, sanitizedName, header.Size)

	// Save to temp, compress, chunk
	storageDir := filepath.Join("./storage/ingest", fileID)
	os.MkdirAll(storageDir, 0755)

	tmpPath := filepath.Join(storageDir, sanitizedName)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		sendError(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	fileSize, err := io.Copy(tmpFile, file)
	tmpFile.Close()
	if err != nil {
		sendError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Compress
	compressedFile, err := compressor.Compress(tmpPath)
	if err != nil {
		sendError(w, "Compression failed", http.StatusInternalServerError)
		return
	}
	compressedFile.Close()
	compressedPath := tmpPath + ".gz"

	// Chunk
	chunker := fileops.NewDefaultFileChunker(1024 * 1024)
	chunks, err := chunker.ChunkFile(compressedPath)
	if err != nil {
		sendError(w, "Chunking failed", http.StatusInternalServerError)
		return
	}

	// Write chunks via write coordinator
	var failedChunks int
	type chunkResult struct {
		Index   int    `json:"index"`
		Worker  string `json:"worker"`
		Status  string `json:"status"`
	}
	var results []chunkResult

	for _, chunk := range chunks {
		chunkData, err := os.ReadFile(chunk.FileName)
		if err != nil {
			failedChunks++
			continue
		}

		chunkID := fmt.Sprintf("%s_chunk_%d", fileID, chunk.Index)
		result := svc.writeCoordinator.WriteChunk(r.Context(), mode, fileID, chunkID, chunk.Index, chunkData, chunk.MD5Hash)

		if result.Success {
			results = append(results, chunkResult{Index: chunk.Index, Worker: result.PrimaryWorker, Status: "ok"})
		} else {
			failedChunks++
			results = append(results, chunkResult{Index: chunk.Index, Status: "failed"})
		}
	}

	// In Strong mode, any failure = upload failure
	if failedChunks > 0 && mode == consistency.ModeStrong {
		os.RemoveAll(storageDir)
		sendError(w, fmt.Sprintf("Upload failed: %d/%d chunks failed (strong consistency)", failedChunks, len(chunks)), http.StatusInternalServerError)
		return
	}
	if failedChunks == len(chunks) {
		os.RemoveAll(storageDir)
		sendError(w, "Upload failed: all chunks failed", http.StatusInternalServerError)
		return
	}

	// Notify metadata service
	go svc.notifyMetadata(fileID, userID, sanitizedName, fileSize, len(chunks))

	// Cleanup temp files
	go os.RemoveAll(storageDir)

	duration := time.Since(start)
	if metrics.AppMetrics != nil {
		metrics.AppMetrics.RecordFileUpload(fileSize, duration)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"message":          "File uploaded successfully",
		"data": map[string]interface{}{
			"file_id":          fileID,
			"chunks":           len(chunks),
			"failed_chunks":    failedChunks,
			"compressed":       true,
			"file_size":        fileSize,
			"consistency_mode": string(mode),
			"mode_reason":      reason,
			"latency_ms":       duration.Milliseconds(),
		},
	})
}

func (svc *IngestService) downloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["fileId"]

	// TODO: query metadata service for chunk count and original filename
	// For now, try to retrieve chunks by iterating
	var allChunks [][]byte
	for i := 0; i < 1000; i++ {
		chunkID := fmt.Sprintf("%s_chunk_%d", fileID, i)
		retrieved := false

		for _, client := range svc.workerRegistry.GetAllWorkers() {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			resp, err := client.RetrieveChunk(ctx, fileID, chunkID, i)
			cancel()

			if err == nil && resp.GetSuccess() && len(resp.GetChunkData()) > 0 {
				allChunks = append(allChunks, resp.GetChunkData())
				retrieved = true
				break
			}
		}

		if !retrieved {
			break // No more chunks
		}
	}

	if len(allChunks) == 0 {
		sendError(w, "File not found or no chunks available", http.StatusNotFound)
		return
	}

	// Reassemble and decompress
	var compressed []byte
	for _, chunk := range allChunks {
		compressed = append(compressed, chunk...)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileID))

	gzReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		// Not compressed, serve raw
		w.Write(compressed)
		return
	}
	defer gzReader.Close()
	io.Copy(w, gzReader)
}

func (svc *IngestService) notifyMetadata(fileID, ownerID, name string, size int64, chunks int) {
	body := fmt.Sprintf(`{"file_id":"%s","owner_id":"%s","original_name":"%s","size":%d,"total_chunks":%d,"status":"completed"}`,
		fileID, ownerID, name, size, chunks)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "POST", svc.metadataURL+"/internal/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		svc.logger.Printf("Warning: Failed to notify metadata service: %v", err)
		return
	}
	resp.Body.Close()
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	for strings.HasPrefix(name, ".") {
		name = name[1:]
	}
	if len(name) > 255 {
		ext := filepath.Ext(name)
		name = name[:255-len(ext)] + ext
	}
	if name == "" || name == "." || name == ".." {
		return ""
	}
	return name
}

func sendError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": msg})
}

// Suppress unused import warnings
var _ = bytes.NewReader
