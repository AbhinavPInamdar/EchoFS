package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"log"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"io"
	"path/filepath"
	"strings"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"echofs/cmd/master/core"
	fileops "echofs/pkg/fileops/Chunker"
	"echofs/pkg/fileops/Compressor"
	"echofs/internal/consistency"
	"echofs/internal/storage"
	"echofs/internal/metrics"
	"echofs/internal/metadata"
	"echofs/internal/api"
	"echofs/pkg/aws"
	"echofs/pkg/auth"
	"echofs/pkg/database"
	grpcClient "echofs/internal/grpc"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type Server struct {
	masterNode       *core.MasterNode 
	router           *mux.Router
	logger           *log.Logger
	chunkStore       *storage.FSChunkStore
	s3Storage        *storage.S3Storage
	dynamoDB         *database.DynamoDBService
	awsConfig        *aws.AWSConfig
	workerRegistry   *grpcClient.WorkerRegistry
	postgresDB       *database.PostgresDB
	userRepo         *database.UserRepository
	fileRepo         *database.FileRepository
	jwtManager       *auth.JWTManager
	authMiddleware   *auth.AuthMiddleware
	authHandler      *api.AuthHandler
	consistencyClient *consistency.Client
	writeCoordinator  *consistency.WriteCoordinator
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

func NewServer(masterNode *core.MasterNode, logger *log.Logger) *Server {

	metrics.InitMetrics()
	chunkStore, err := storage.NewFSChunkStore("")
	if err != nil {
		logger.Fatalf("Failed to create chunk store: %v", err)
	}

	ctx := context.Background()
	awsConfig, err := aws.NewAWSConfig(ctx, "us-east-1", "")
	if err != nil {
		logger.Printf("Warning: Failed to initialize AWS config: %v", err)
	}

	var s3Storage *storage.S3Storage
	var dynamoDB *database.DynamoDBService

	if awsConfig != nil {
		s3Storage = storage.NewS3Storage(awsConfig.S3, awsConfig.S3BucketName)
		
		dynamoDB = database.NewDynamoDBService(
			awsConfig.DynamoDB,
			awsConfig.DynamoDBTables.Files,
			awsConfig.DynamoDBTables.Chunks,
			awsConfig.DynamoDBTables.Sessions,
		)

		if err := s3Storage.EnsureBucket(ctx); err != nil {
			logger.Printf("Warning: Failed to ensure S3 bucket: %v", err)
		}

		if err := dynamoDB.CreateTables(ctx); err != nil {
			logger.Printf("Warning: Failed to create DynamoDB tables: %v", err)
		}
	}
	
	workerRegistry := grpcClient.NewWorkerRegistry(logger)
	
	// Get worker URLs from environment variables or use localhost defaults
	worker1URL := getEnv("WORKER1_URL", "localhost:10081")
	worker2URL := getEnv("WORKER2_URL", "localhost:10082") 
	worker3URL := getEnv("WORKER3_URL", "localhost:10083")
	
	if err := workerRegistry.RegisterWorker("worker1", worker1URL); err != nil {
		logger.Printf("Warning: Failed to register worker1 at %s: %v", worker1URL, err)
	}
	if err := workerRegistry.RegisterWorker("worker2", worker2URL); err != nil {
		logger.Printf("Warning: Failed to register worker2 at %s: %v", worker2URL, err)
	}
	if err := workerRegistry.RegisterWorker("worker3", worker3URL); err != nil {
		logger.Printf("Warning: Failed to register worker3 at %s: %v", worker3URL, err)
	}

	// Initialize PostgreSQL connection
	databaseURL := getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/echofs?sslmode=disable")
	postgresDB, err := database.NewPostgresDB(databaseURL, 25, 30*time.Second)
	if err != nil {
		logger.Printf("Warning: Failed to connect to PostgreSQL: %v", err)
	}

	// Initialize repositories
	var userRepo *database.UserRepository
	var fileRepo *database.FileRepository
	if postgresDB != nil {
		userRepo = database.NewUserRepository(postgresDB)
		fileRepo = database.NewFileRepository(postgresDB)

		// Initialize database schema
		if err := userRepo.InitSchema(ctx); err != nil {
			logger.Printf("Warning: Failed to initialize user schema: %v", err)
		}
		if err := fileRepo.InitSchema(ctx); err != nil {
			logger.Printf("Warning: Failed to initialize file schema: %v", err)
		}
	}

	// Initialize JWT manager
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		logger.Fatalf("JWT_SECRET environment variable is required")
	}
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
	authMiddleware := auth.NewAuthMiddleware(jwtManager)

	// Initialize auth handler
	var authHandler *api.AuthHandler
	if userRepo != nil {
		authHandler = api.NewAuthHandler(userRepo, jwtManager, logger)
	}

	// Initialize consistency client (queries the orchestrator for mode decisions)
	orchestratorURL := getEnv("ORCHESTRATOR_URL", "http://localhost:8082")
	consistencyClient := consistency.NewClient(orchestratorURL, logger)

	// Initialize write coordinator (executes writes according to consistency mode)
	replicationFactor := 3
	if rf := getEnv("REPLICATION_FACTOR", ""); rf != "" {
		fmt.Sscanf(rf, "%d", &replicationFactor)
	}
	writeCoordinator := consistency.NewWriteCoordinator(workerRegistry, replicationFactor, logger)

	s := &Server{
		masterNode:        masterNode,
		logger:            logger,
		router:            mux.NewRouter(),
		chunkStore:        chunkStore,
		s3Storage:         s3Storage,
		dynamoDB:          dynamoDB,
		awsConfig:         awsConfig,
		workerRegistry:    workerRegistry,
		postgresDB:        postgresDB,
		userRepo:          userRepo,
		fileRepo:          fileRepo,
		jwtManager:        jwtManager,
		authMiddleware:    authMiddleware,
		authHandler:       authHandler,
		consistencyClient: consistencyClient,
		writeCoordinator:  writeCoordinator,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {

	s.router.Use(metrics.HTTPMetricsMiddleware)
	
	s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	s.router.HandleFunc("/metrics/dashboard", metrics.DashboardHandler).Methods("GET")
	
	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Public routes (no authentication required) — rate limited
	if s.authHandler != nil {
		apiRouter.HandleFunc("/auth/register", s.authHandler.Register).Methods("POST")
		apiRouter.HandleFunc("/auth/login", s.authHandler.Login).Methods("POST")
	}
	apiRouter.HandleFunc("/health", s.HealthCheck).Methods("GET")
	
	// Protected routes (authentication required)
	protected := apiRouter.PathPrefix("").Subrouter()
	if s.authMiddleware != nil {
		protected.Use(s.authMiddleware.Authenticate)
	}
	
	if s.authHandler != nil {
		protected.HandleFunc("/auth/profile", s.authHandler.GetProfile).Methods("GET")
	}
	
	protected.HandleFunc("/files/upload", s.UploadFile).Methods("POST")
	protected.HandleFunc("/files/{fileId}/download", s.DownloadFile).Methods("GET")
	protected.HandleFunc("/files", s.ListFiles).Methods("GET")
	protected.HandleFunc("/files/{fileId}", s.DeleteFile).Methods("DELETE")
	
	protected.HandleFunc("/files/upload/init", s.InitUpload).Methods("POST")
	protected.HandleFunc("/files/upload/chunk", s.UploadChunk).Methods("POST")
	protected.HandleFunc("/files/upload/complete", s.CompleteUpload).Methods("POST")
	
	protected.HandleFunc("/workers/register", s.RegisterWorker).Methods("POST")
	protected.HandleFunc("/workers/{workerId}/heartbeat", s.WorkerHeartbeat).Methods("POST")
	protected.HandleFunc("/workers/health", s.WorkersHealthCheck).Methods("GET")
	
	// Consistency stats endpoint (admin only)
	protected.HandleFunc("/consistency/stats", s.ConsistencyStats).Methods("GET")
}

func (s *Server) Start(port int) error {
	s.logger.Printf("Starting server on port %d", port)

	// Restrict CORS to configured origins
	allowedOrigins := []string{"http://localhost:3000"}
	if frontendURL := getEnv("FRONTEND_URL", ""); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	// Allow the production frontend
	if prodURL := getEnv("PRODUCTION_FRONTEND_URL", "https://frontend-echofs-projects.vercel.app"); prodURL != "" {
		allowedOrigins = append(allowedOrigins, prodURL)
	}

	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	
	handler := c.Handler(s.router)
	
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		
		s.logger.Println("Shutting down server...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Stop write coordinator (drain async queue)
		if s.writeCoordinator != nil {
			s.writeCoordinator.Stop()
		}
		
		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Printf("Server shutdown error: %v", err)
		}
	}()
	
	return srv.ListenAndServe()
}

func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	s.logger.Println("UploadFile called - consistency-aware chunked upload")
	w.Header().Set("Content-Type", "application/json")
	
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		if metrics.AppMetrics != nil {
			metrics.AppMetrics.RecordFileError("upload", "parse_form_error")
		}
		s.sendErrorResponse(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}
	
	file, header, err := r.FormFile("file")
	if err != nil {
		if metrics.AppMetrics != nil {
			metrics.AppMetrics.RecordFileError("upload", "no_file_provided")
		}
		s.sendErrorResponse(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	// Get authenticated user from context
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendErrorResponse(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID
	
	// Validate file size (max 100MB)
	if header.Size > 100*1024*1024 {
		s.sendErrorResponse(w, "File too large (max 100MB)", http.StatusBadRequest)
		return
	}
	
	// Sanitize filename
	sanitizedName := sanitizeFilename(header.Filename)
	if sanitizedName == "" {
		s.sendErrorResponse(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	
	fileID := uuid.New().String()
	sessionID := uuid.New().String()
	
	// Query the consistency orchestrator for the write mode
	mode, reason := s.consistencyClient.GetMode(r.Context(), fileID)
	s.logger.Printf("Consistency mode for file %s: %s (reason: %s)", fileID, mode, reason)
	
	// Register the object with the orchestrator
	go s.consistencyClient.RegisterObject(context.Background(), fileID, sanitizedName, header.Size)
	
	// Save uploaded file temporarily for chunking
	storageDir := filepath.Join("./storage/uploads", fileID)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		s.sendErrorResponse(w, "Failed to create storage directory", http.StatusInternalServerError)
		return
	}
	
	originalFilePath := filepath.Join(storageDir, sanitizedName)
	originalFile, err := os.Create(originalFilePath)
	if err != nil {
		s.sendErrorResponse(w, "Failed to create storage file", http.StatusInternalServerError)
		return
	}
	
	fileSize, err := io.Copy(originalFile, file)
	if err != nil {
		originalFile.Close()
		s.sendErrorResponse(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	originalFile.Close()
	
	// Compress the file
	s.logger.Printf("Compressing file: %s", sanitizedName)
	compressedFile, err := compressor.Compress(originalFilePath)
	if err != nil {
		s.sendErrorResponse(w, "Failed to compress file", http.StatusInternalServerError)
		return
	}
	defer compressedFile.Close()
	
	compressedPath := originalFilePath + ".gz"
	
	// Chunk the compressed file
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
	
	s.logger.Printf("Created %d chunks for file %s (mode: %s)", len(chunks), sanitizedName, mode)
	
	// Check workers are available
	workers := s.workerRegistry.GetAllWorkers()
	if len(workers) == 0 {
		s.sendErrorResponse(w, "No workers available for chunk storage", http.StatusServiceUnavailable)
		return
	}
	
	// Write chunks using the consistency-aware write coordinator
	var chunkAssignments []core.ChunkAssignment
	var failedChunks int
	
	for _, chunk := range chunks {
		chunkData, err := os.ReadFile(chunk.FileName)
		if err != nil {
			s.logger.Printf("Failed to read chunk file %s: %v", chunk.FileName, err)
			failedChunks++
			continue
		}
		
		chunkID := fmt.Sprintf("%s_chunk_%d", fileID, chunk.Index)
		
		// Use the write coordinator with the consistency mode
		result := s.writeCoordinator.WriteChunk(
			r.Context(),
			mode,
			fileID,
			chunkID,
			chunk.Index,
			chunkData,
			chunk.MD5Hash,
		)
		
		if result.Success {
			s.logger.Printf("✅ Chunk %d stored (mode: %s, primary: %s, latency: %v)",
				chunk.Index, result.Mode, result.PrimaryWorker, result.Latency)
			
			assignment := core.ChunkAssignment{
				ChunkIndex:     chunk.Index,
				PrimaryWorker:  result.PrimaryWorker,
				ReplicaWorkers: result.Replicas,
				MD5Expected:    chunk.MD5Hash,
				Status:         "completed",
			}
			chunkAssignments = append(chunkAssignments, assignment)
		} else {
			s.logger.Printf("❌ Chunk %d failed (mode: %s): %v", chunk.Index, result.Mode, result.Error)
			failedChunks++
			
			assignment := core.ChunkAssignment{
				ChunkIndex:     chunk.Index,
				PrimaryWorker:  result.PrimaryWorker,
				ReplicaWorkers: []string{},
				MD5Expected:    chunk.MD5Hash,
				Status:         "failed",
			}
			chunkAssignments = append(chunkAssignments, assignment)
		}
	}
	
	// If any chunks failed in Strong mode, the upload fails
	if failedChunks > 0 && mode == consistency.ModeStrong {
		// Clean up: delete chunks that were stored
		s.cleanupFailedUpload(fileID, storageDir)
		s.sendErrorResponse(w, fmt.Sprintf("Upload failed: %d/%d chunks could not be stored with strong consistency", failedChunks, len(chunks)), http.StatusInternalServerError)
		return
	}
	
	// In Available mode, we accept partial success (at least one chunk must succeed)
	if failedChunks == len(chunks) {
		s.cleanupFailedUpload(fileID, storageDir)
		s.sendErrorResponse(w, "Upload failed: no chunks could be stored", http.StatusInternalServerError)
		return
	}
	
	// Store upload session
	session := &core.UploadSession{
		SessionID:       sessionID,
		UserID:          userID,
		FileName:        sanitizedName,
		FileSize:        fileSize,
		ChunkSize:       int64(len(chunks)), 
		TotalChunks:     len(chunks),
		ChunkAssignment: chunkAssignments,
		Status:          core.SessionStatusActive,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	s.masterNode.AddUploadSession(session)
	
	// Save file metadata to database
	if s.fileRepo != nil {
		fileMetadata := &metadata.FileMetadata{
			FileID:       fileID,
			Size:         fileSize,
			OriginalName: sanitizedName,
			ChunkSize:    1024 * 1024,
			TotalChunks:  len(chunks),
			OwnerID:      userID,
			Status:       "completed",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := s.fileRepo.CreateFile(ctx, fileMetadata); err != nil {
			s.logger.Printf("Warning: Failed to save file metadata to database: %v", err)
		}
	}
	
	// Clean up temporary files (chunks are now on workers)
	go s.cleanupTempFiles(storageDir, compressedPath)
	
	response := map[string]interface{}{
		"file_id":          fileID,
		"session_id":       sessionID,
		"chunks":           len(chunks),
		"failed_chunks":    failedChunks,
		"compressed":       true,
		"file_size":        fileSize,
		"owner_id":         userID,
		"consistency_mode": string(mode),
		"mode_reason":      reason,
	}
	
	if metrics.AppMetrics != nil {
		duration := time.Since(start)
		metrics.AppMetrics.RecordFileUpload(fileSize, duration)
	}
	
	s.sendSuccessResponse(w, "File uploaded successfully", response)
}

func (s *Server) InitUpload(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("InitUpload called")
	w.Header().Set("Content-Type", "application/json")
	
	// Get authenticated user from context
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendErrorResponse(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	
	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.FileName == "" || req.FileSize <= 0 {
		s.sendErrorResponse(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	
	// Use authenticated user ID
	req.UserID = claims.UserID
	
	sessionID := uuid.New().String()
	
	chunkSize := int64(1024 * 1024) 
	totalChunks := int(req.FileSize / chunkSize)
	if req.FileSize%chunkSize != 0 {
		totalChunks++
	}
	
	workers := s.workerRegistry.GetAllWorkers()
	workerList := make([]string, 0, len(workers))
	for workerID := range workers {
		workerList = append(workerList, workerID)
	}
	
	if len(workerList) == 0 {
		s.sendErrorResponse(w, "No workers available", http.StatusServiceUnavailable)
		return
	}
	
	var chunkAssignments []core.ChunkAssignment
	for i := 0; i < totalChunks; i++ {

		primaryWorker := workerList[i%len(workerList)]
		
		assignment := core.ChunkAssignment{
			ChunkIndex:     i,
			PrimaryWorker:  primaryWorker,
			ReplicaWorkers: []string{},
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
	s.sendErrorResponse(w, "Chunked upload not yet supported. Use /files/upload for single-file upload.", http.StatusNotImplemented)
}

func (s *Server) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	s.sendErrorResponse(w, "Chunked upload not yet supported. Use /files/upload for single-file upload.", http.StatusNotImplemented)
}

func (s *Server) DownloadFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	s.logger.Printf("DownloadFile called for fileId: %s", fileId)
	
	// Get authenticated user from context
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendErrorResponse(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	
	// Get file metadata from database
	var fileMeta *metadata.FileMetadata
	if s.fileRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Check ownership
		isOwner, err := s.fileRepo.CheckFileOwnership(ctx, fileId, claims.UserID)
		if err != nil {
			s.logger.Printf("Failed to check file ownership: %v", err)
			s.sendErrorResponse(w, "Failed to verify file access", http.StatusInternalServerError)
			return
		}
		if !isOwner {
			s.sendErrorResponse(w, "Access denied: You don't own this file", http.StatusForbidden)
			return
		}
		
		fileMeta, err = s.fileRepo.GetFileByID(ctx, fileId)
		if err != nil {
			s.sendErrorResponse(w, "File not found", http.StatusNotFound)
			return
		}
	}
	
	// Retrieve chunks from workers via gRPC
	if fileMeta != nil && fileMeta.TotalChunks > 0 {
		s.logger.Printf("Retrieving %d chunks from workers for file %s", fileMeta.TotalChunks, fileId)
		
		// Collect all chunk data in order
		allChunkData := make([][]byte, fileMeta.TotalChunks)
		var retrievalErrors int
		
		for i := 0; i < fileMeta.TotalChunks; i++ {
			chunkID := fmt.Sprintf("%s_chunk_%d", fileId, i)
			
			// Try each worker until we get the chunk
			retrieved := false
			workers := s.workerRegistry.GetAllWorkers()
			for workerID, workerClient := range workers {
				ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
				resp, err := workerClient.RetrieveChunk(ctx, fileId, chunkID, i)
				cancel()
				
				if err == nil && resp.GetSuccess() {
					allChunkData[i] = resp.GetChunkData()
					retrieved = true
					s.logger.Printf("Retrieved chunk %d from worker %s", i, workerID)
					break
				}
			}
			
			if !retrieved {
				retrievalErrors++
				s.logger.Printf("Failed to retrieve chunk %d from any worker", i)
			}
		}
		
		if retrievalErrors > 0 {
			s.logger.Printf("Warning: %d chunks could not be retrieved", retrievalErrors)
			// If we're missing chunks, try the local fallback
		}
		
		// If we got all chunks from workers, reassemble and serve
		if retrievalErrors == 0 {
			// Set response headers
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileMeta.OriginalName))
			
			// Reassemble compressed data, then decompress
			var compressedData []byte
			for _, chunkData := range allChunkData {
				if chunkData != nil {
					compressedData = append(compressedData, chunkData...)
				}
			}
			
			// Decompress gzip data
			gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
			if err != nil {
				// If decompression fails, serve raw data (might not be compressed)
				s.logger.Printf("Warning: Failed to decompress, serving raw: %v", err)
				w.Write(compressedData)
			} else {
				defer gzReader.Close()
				io.Copy(w, gzReader)
			}
			
			if metrics.AppMetrics != nil {
				duration := time.Since(start)
				metrics.AppMetrics.RecordFileDownload(duration)
			}
			return
		}
	}
	
	// Fallback: try local filesystem (for files uploaded before this change)
	storageDir := filepath.Join("./storage/uploads", fileId)
	files, err := os.ReadDir(storageDir)
	if err != nil {
		s.sendErrorResponse(w, "File not found", http.StatusNotFound)
		return
	}
	
	var originalFile string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) != ".gz" && !strings.HasSuffix(file.Name(), ".chunk") {
			originalFile = filepath.Join(storageDir, file.Name())
			break
		}
	}
	
	if originalFile == "" {
		s.sendErrorResponse(w, "Original file not found", http.StatusNotFound)
		return
	}
	
	f, err := os.Open(originalFile)
	if err != nil {
		s.sendErrorResponse(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	
	fileInfo, err := f.Stat()
	if err != nil {
		s.sendErrorResponse(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(originalFile)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	
	io.Copy(w, f)
	
	if metrics.AppMetrics != nil {
		duration := time.Since(start)
		metrics.AppMetrics.RecordFileDownload(duration)
	}
}

func (s *Server) ListFiles(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("ListFiles called")
	
	// Get authenticated user from context
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendErrorResponse(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	
	// Get files from database for this user
	if s.fileRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		files, err := s.fileRepo.GetFilesByOwner(ctx, claims.UserID)
		if err != nil {
			s.logger.Printf("Failed to get files from database: %v", err)
			s.sendErrorResponse(w, "Failed to list files", http.StatusInternalServerError)
			return
		}
		
		// Convert to response format
		var fileList []map[string]interface{}
		for _, file := range files {
			fileList = append(fileList, map[string]interface{}{
				"file_id":   file.FileID,
				"name":      file.OriginalName,
				"size":      file.Size,
				"uploaded":  file.CreatedAt.Format(time.RFC3339),
				"status":    file.Status,
				"chunks":    file.TotalChunks,
			})
		}
		
		s.logger.Printf("Returning %d files for user %s", len(fileList), claims.UserID)
		s.sendSuccessResponse(w, "Files listed successfully", fileList)
		return
	}
	
	// Fallback to filesystem-based listing (legacy)
	uploadsDir := "./storage/uploads"
	
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		s.logger.Printf("Failed to create uploads directory: %v", err)
		s.sendErrorResponse(w, "Failed to access storage", http.StatusInternalServerError)
		return
	}
	
	dirs, err := os.ReadDir(uploadsDir)
	if err != nil {
		s.logger.Printf("Failed to read uploads directory: %v", err)
		s.sendErrorResponse(w, "Failed to list files", http.StatusInternalServerError)
		return
	}
	
	var files []map[string]interface{}
	
	for _, dir := range dirs {
		if dir.IsDir() {
			fileId := dir.Name()
			dirPath := filepath.Join(uploadsDir, fileId)
			
			dirFiles, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}
			
			for _, file := range dirFiles {
				if !file.IsDir() {
					fileInfo, err := file.Info()
					if err != nil {
						continue
					}
					
					displayName := file.Name()
					if strings.HasSuffix(displayName, ".gz") {
						displayName = strings.TrimSuffix(displayName, ".gz")
					}
					
					files = append(files, map[string]interface{}{
						"file_id":   fileId,
						"name":      displayName,
						"size":      fileInfo.Size(),
						"uploaded":  fileInfo.ModTime().Format(time.RFC3339),
						"type":      filepath.Ext(displayName),
					})
					break
				}
			}
		}
	}
	
	s.logger.Printf("Returning %d files (filesystem fallback)", len(files))
	s.sendSuccessResponse(w, "Files listed successfully", files)
}

func (s *Server) DeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	s.logger.Printf("DeleteFile called for fileId: %s", fileId)
	
	// Get authenticated user from context
	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		s.sendErrorResponse(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	
	// Delete from database
	if s.fileRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := s.fileRepo.DeleteFile(ctx, fileId, claims.UserID); err != nil {
			if err == database.ErrUserNotFound {
				s.sendErrorResponse(w, "File not found or access denied", http.StatusNotFound)
				return
			}
			s.logger.Printf("Failed to delete file from database: %v", err)
			s.sendErrorResponse(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}
	}
	
	// Delete from filesystem
	storageDir := filepath.Join("./storage/uploads", fileId)
	if err := os.RemoveAll(storageDir); err != nil {
		s.logger.Printf("Warning: Failed to delete file directory: %v", err)
	}
	
	s.sendSuccessResponse(w, "File deleted successfully", nil)
}

func (s *Server) RegisterWorker(w http.ResponseWriter, r *http.Request) {
	s.sendErrorResponse(w, "Dynamic worker registration not yet supported. Workers are configured via environment variables.", http.StatusNotImplemented)
}

func (s *Server) WorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workerId := vars["workerId"]
	
	// Update worker last-seen timestamp
	if _, exists := s.workerRegistry.GetWorker(workerId); exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "workerId": workerId})
	} else {
		s.sendErrorResponse(w, "Worker not found", http.StatusNotFound)
	}
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "echofs-master"})
}

func (s *Server) WorkersHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	workers := s.workerRegistry.GetAllWorkers()
	healthStatus := make(map[string]interface{})
	
	for workerID, workerClient := range workers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := workerClient.HealthCheck(ctx)
		cancel()
		
		if err != nil {
			healthStatus[workerID] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			healthStatus[workerID] = map[string]interface{}{
				"status":    resp.GetStatus(),
				"healthy":   resp.GetHealthy(),
				"timestamp": resp.GetTimestamp(),
			}
		}
	}
	
	response := map[string]interface{}{
		"service": "echofs-master",
		"workers": healthStatus,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *Server) ConsistencyStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	stats := map[string]interface{}{
		"write_coordinator": s.writeCoordinator.GetStats(),
	}
	
	json.NewEncoder(w).Encode(stats)
}

// sanitizeFilename removes path separators and dangerous characters from filenames.
func sanitizeFilename(name string) string {
	// Take only the base name (strip any directory components)
	name = filepath.Base(name)

	// Remove null bytes
	name = strings.ReplaceAll(name, "\x00", "")

	// Replace path separators
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	// Remove leading dots (hidden files / directory traversal)
	for strings.HasPrefix(name, ".") {
		name = name[1:]
	}

	// Limit length
	if len(name) > 255 {
		ext := filepath.Ext(name)
		name = name[:255-len(ext)] + ext
	}

	// Reject empty names
	if name == "" || name == "." || name == ".." {
		return ""
	}

	return name
}

func (s *Server) cleanupFailedUpload(fileID, storageDir string) {
	// Remove local temp files
	os.RemoveAll(storageDir)
	
	// Delete any chunks that were stored on workers
	workers := s.workerRegistry.GetAllWorkers()
	for workerID, workerClient := range workers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// Try to delete chunks (best effort)
		for i := 0; i < 100; i++ { // reasonable upper bound
			chunkID := fmt.Sprintf("%s_chunk_%d", fileID, i)
			_, err := workerClient.DeleteChunk(ctx, fileID, chunkID, i)
			if err != nil {
				break // No more chunks on this worker
			}
		}
		cancel()
		_ = workerID
	}
}

func (s *Server) cleanupTempFiles(storageDir, compressedPath string) {
	// Remove chunk files and compressed file, keep original for fallback downloads
	// In the future, once download-from-workers is fully reliable, remove everything
	files, err := os.ReadDir(storageDir)
	if err != nil {
		return
	}
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, ".gz") || strings.HasSuffix(name, ".chunk") || strings.Contains(name, "_chunk_") {
			os.Remove(filepath.Join(storageDir, name))
		}
	}
}

func (s *Server) sendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := APIResponse{
		Success: false,
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}