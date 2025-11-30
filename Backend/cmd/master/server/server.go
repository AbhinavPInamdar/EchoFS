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
	"strings"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"echofs/cmd/master/core"
	fileops "echofs/pkg/fileops/Chunker"
	"echofs/pkg/fileops/Compressor"
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
	masterNode     *core.MasterNode 
	router         *mux.Router
	logger         *log.Logger
	chunkStore     *storage.FSChunkStore
	s3Storage      *storage.S3Storage
	dynamoDB       *database.DynamoDBService
	awsConfig      *aws.AWSConfig
	workerRegistry *grpcClient.WorkerRegistry
	postgresDB     *database.PostgresDB
	userRepo       *database.UserRepository
	fileRepo       *database.FileRepository
	jwtManager     *auth.JWTManager
	authMiddleware *auth.AuthMiddleware
	authHandler    *api.AuthHandler
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
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
	authMiddleware := auth.NewAuthMiddleware(jwtManager)

	// Initialize auth handler
	var authHandler *api.AuthHandler
	if userRepo != nil {
		authHandler = api.NewAuthHandler(userRepo, jwtManager, logger)
	}

	s := &Server{
		masterNode:     masterNode,
		logger:         logger,
		router:         mux.NewRouter(),
		chunkStore:     chunkStore,
		s3Storage:      s3Storage,
		dynamoDB:       dynamoDB,
		awsConfig:      awsConfig,
		workerRegistry: workerRegistry,
		postgresDB:     postgresDB,
		userRepo:       userRepo,
		fileRepo:       fileRepo,
		jwtManager:     jwtManager,
		authMiddleware: authMiddleware,
		authHandler:    authHandler,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {

	s.router.Use(metrics.HTTPMetricsMiddleware)
	
	s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	s.router.HandleFunc("/metrics/dashboard", metrics.DashboardHandler).Methods("GET")
	
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Public routes (no authentication required)
	if s.authHandler != nil {
		api.HandleFunc("/auth/register", s.authHandler.Register).Methods("POST")
		api.HandleFunc("/auth/login", s.authHandler.Login).Methods("POST")
	}
	api.HandleFunc("/health", s.HealthCheck).Methods("GET")
	
	// Protected routes (authentication required)
	protected := api.PathPrefix("").Subrouter()
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
}

func (s *Server) Start(port int) error {
	s.logger.Printf("Starting server on port %d", port)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		AllowCredentials: false,
	})
	
	handler := c.Handler(s.router)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}

func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	s.logger.Println("UploadFile called - integrated chunking and compression")
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
	
	sessionID := uuid.New().String()
	fileID := uuid.New().String()
	
	storageDir := filepath.Join("./storage/uploads", fileID)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		s.sendErrorResponse(w, "Failed to create storage directory", http.StatusInternalServerError)
		return
	}
	
	originalFilePath := filepath.Join(storageDir, header.Filename)
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
	
	s.logger.Printf("Compressing file: %s", header.Filename)
	compressedFile, err := compressor.Compress(originalFilePath)
	if err != nil {
		s.sendErrorResponse(w, "Failed to compress file", http.StatusInternalServerError)
		return
	}
	defer compressedFile.Close()
	
	compressedPath := originalFilePath + ".gz"
	
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
	
	var chunkAssignments []core.ChunkAssignment
	workers := s.workerRegistry.GetAllWorkers()
	workerList := make([]string, 0, len(workers))
	for workerID := range workers {
		workerList = append(workerList, workerID)
	}
	
	if len(workerList) == 0 {
		s.sendErrorResponse(w, "No workers available for chunk storage", http.StatusServiceUnavailable)
		return
	}
	
	for i, chunk := range chunks {

		primaryWorker := workerList[i%len(workerList)]
		
		if workerClient, exists := s.workerRegistry.GetWorker(primaryWorker); exists {
			chunkData, err := os.ReadFile(chunk.FileName)
			if err != nil {
				s.logger.Printf("Failed to read chunk file %s: %v", chunk.FileName, err)
				continue
			}
			
			chunkID := fmt.Sprintf("%s_chunk_%d", fileID, chunk.Index)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			resp, err := workerClient.StoreChunk(ctx, fileID, chunkID, chunk.Index, chunkData, chunk.MD5Hash)
			cancel()
			
			if err != nil {
				s.logger.Printf("Failed to store chunk %s on worker %s via gRPC: %v", chunkID, primaryWorker, err)

				assignment := core.ChunkAssignment{
					ChunkIndex:     chunk.Index,
					PrimaryWorker:  primaryWorker,
					ReplicaWorkers: []string{},
					MD5Expected:    chunk.MD5Hash,
					Status:         "failed",
				}
				chunkAssignments = append(chunkAssignments, assignment)
				continue
			}
			
			s.logger.Printf("✅ Stored chunk %s on worker %s via gRPC: %s", chunkID, primaryWorker, resp.GetMessage())
			
			assignment := core.ChunkAssignment{
				ChunkIndex:     chunk.Index,
				PrimaryWorker:  primaryWorker,
				ReplicaWorkers: []string{},
				MD5Expected:    chunk.MD5Hash,
				Status:         "completed",
			}
			chunkAssignments = append(chunkAssignments, assignment)
		} else {
			s.logger.Printf("❌ Worker %s not found in registry", primaryWorker)
			assignment := core.ChunkAssignment{
				ChunkIndex:     chunk.Index,
				PrimaryWorker:  primaryWorker,
				ReplicaWorkers: []string{},
				MD5Expected:    chunk.MD5Hash,
				Status:         "failed",
			}
			chunkAssignments = append(chunkAssignments, assignment)
		}
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
	
	s.logger.Printf("Successfully uploaded %d chunks to workers via gRPC", len(chunks))
	
	// Save file metadata to database
	if s.fileRepo != nil {
		fileMetadata := &metadata.FileMetadata{
			FileID:       fileID,
			Size:         fileSize,
			OriginalName: header.Filename,
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
	
	response := map[string]interface{}{
		"file_id":     fileID,
		"session_id":  sessionID,
		"chunks":      len(chunks),
		"compressed":  true,
		"file_size":   fileSize,
		"owner_id":    userID,
	}
	
	if metrics.AppMetrics != nil {
		duration := time.Since(start)
		metrics.AppMetrics.RecordFileUpload(fileSize, duration)
	}
	
	s.sendSuccessResponse(w, "File uploaded, compressed, and chunked successfully", response)
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
	
	// Check file ownership
	if s.fileRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
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
	}
	
	storageDir := filepath.Join("./storage/uploads", fileId)
	
	files, err := os.ReadDir(storageDir)
	if err != nil {
		s.sendErrorResponse(w, "File not found", http.StatusNotFound)
		return
	}
	
	var originalFile string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) != ".gz" {
			originalFile = filepath.Join(storageDir, file.Name())
			break
		}
	}
	
	if originalFile == "" {
		s.sendErrorResponse(w, "Original file not found", http.StatusNotFound)
		return
	}
	
	file, err := os.Open(originalFile)
	if err != nil {
		s.sendErrorResponse(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	
	fileInfo, err := file.Stat()
	if err != nil {
		s.sendErrorResponse(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(originalFile)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	
	io.Copy(w, file)
	
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