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
	"github.com/rs/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"echofs/cmd/master/core"
	fileops "echofs/pkg/fileops/Chunker"
	"echofs/pkg/fileops/Compressor"
	"echofs/internal/storage"
	"echofs/internal/metrics"
	"echofs/internal/auth"
	"echofs/pkg/aws"
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
	authService    *auth.AuthService
	authHandlers   *auth.AuthHandlers
	authMiddleware *auth.AuthMiddleware
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

	// Initialize authentication system
	var authService *auth.AuthService
	var authHandlers *auth.AuthHandlers
	var authMiddleware *auth.AuthMiddleware

	if awsConfig != nil {
		// Create JWT manager
		jwtSecret := getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production")
		jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

		// Create user repository
		userRepo := auth.NewUserRepository(awsConfig.DynamoDB, "echofs-users")
		if err := userRepo.CreateTable(ctx); err != nil {
			logger.Printf("Warning: Failed to create users table: %v", err)
		}

		// Create auth service and handlers
		authService = auth.NewAuthService(userRepo, jwtManager, logger)
		authHandlers = auth.NewAuthHandlers(authService)
		authMiddleware = auth.NewAuthMiddleware(jwtManager)
		
		// Set up OAuth handlers
		oauthHandlers := auth.NewOAuthHandlers(authService, jwtManager)
		authHandlers.SetOAuthHandlers(oauthHandlers)
		
		// Set up simple OAuth handler (fallback)
		simpleOAuthHandler := auth.NewSimpleOAuthHandler(jwtManager)
		authHandlers.SetSimpleOAuthHandler(simpleOAuthHandler)

		logger.Printf("✅ Authentication system initialized")
	} else {
		logger.Printf("⚠️  Authentication disabled (AWS not configured)")
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
		authService:    authService,
		authHandlers:   authHandlers,
		authMiddleware: authMiddleware,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(metrics.HTTPMetricsMiddleware)
	
	// Public routes (no auth required)
	s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	s.router.HandleFunc("/metrics/dashboard", metrics.DashboardHandler).Methods("GET")
	s.router.HandleFunc("/health", s.HealthCheck).Methods("GET")
	
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Authentication routes (public)
	if s.authHandlers != nil {
		auth := api.PathPrefix("/auth").Subrouter()
		auth.HandleFunc("/register", s.authHandlers.Register).Methods("POST")
		auth.HandleFunc("/login", s.authHandlers.Login).Methods("POST")
		auth.HandleFunc("/refresh", s.authHandlers.RefreshToken).Methods("POST")
		auth.HandleFunc("/logout", s.authHandlers.Logout).Methods("POST")
		
		// OAuth callback endpoint
		if s.authHandlers.OAuthHandlers != nil {
			auth.HandleFunc("/oauth/callback", s.authHandlers.OAuthHandlers.HandleOAuthCallback).Methods("POST")
		}
		
		// Simple OAuth endpoint (fallback)
		if s.authHandlers.SimpleOAuthHandler != nil {
			auth.HandleFunc("/oauth/simple", s.authHandlers.SimpleOAuthHandler.HandleSimpleOAuth).Methods("POST")
		}
		
		// Protected auth routes
		if s.authMiddleware != nil {
			auth.HandleFunc("/profile", s.authMiddleware.RequireAuth(s.authHandlers.Profile)).Methods("GET")
		}
	}
	
	// File operations - protected routes
	if s.authMiddleware != nil {
		api.HandleFunc("/files/upload", s.authMiddleware.RequireAuth(s.UploadFile)).Methods("POST")
		api.HandleFunc("/files/{fileId}/download", s.authMiddleware.RequireAuth(s.DownloadFile)).Methods("GET")
		api.HandleFunc("/files", s.authMiddleware.RequireAuth(s.ListFiles)).Methods("GET")
		api.HandleFunc("/files/{fileId}", s.authMiddleware.RequireAuth(s.DeleteFile)).Methods("DELETE")
		
		api.HandleFunc("/files/upload/init", s.authMiddleware.RequireAuth(s.InitUpload)).Methods("POST")
		api.HandleFunc("/files/upload/chunk", s.authMiddleware.RequireAuth(s.UploadChunk)).Methods("POST")
		api.HandleFunc("/files/upload/complete", s.authMiddleware.RequireAuth(s.CompleteUpload)).Methods("POST")
	} else {
		// Fallback for when auth is disabled
		api.HandleFunc("/files/upload", s.UploadFile).Methods("POST")
		api.HandleFunc("/files/{fileId}/download", s.DownloadFile).Methods("GET")
		api.HandleFunc("/files", s.ListFiles).Methods("GET")
		api.HandleFunc("/files/{fileId}", s.DeleteFile).Methods("DELETE")
		
		api.HandleFunc("/files/upload/init", s.InitUpload).Methods("POST")
		api.HandleFunc("/files/upload/chunk", s.UploadChunk).Methods("POST")
		api.HandleFunc("/files/upload/complete", s.CompleteUpload).Methods("POST")
	}
	
	// Worker management routes (internal, no auth needed)
	api.HandleFunc("/workers/register", s.RegisterWorker).Methods("POST")
	api.HandleFunc("/workers/{workerId}/heartbeat", s.WorkerHeartbeat).Methods("POST")
	api.HandleFunc("/workers/health", s.WorkersHealthCheck).Methods("GET")
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
	var userID string
	if s.authMiddleware != nil {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			s.sendErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		userID = user.UserID
		s.logger.Printf("File upload by authenticated user: %s (%s)", user.Username, userID)
	} else {
		// Fallback for when auth is disabled
		userID = r.FormValue("user_id")
		if userID == "" {
			userID = "anonymous"
		}
	}
	
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
	
	if metrics.AppMetrics != nil {
		duration := time.Since(start)
		metrics.AppMetrics.RecordFileUpload(fileSize, duration)
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
	
	// Get authenticated user from context
	var userID string
	if s.authMiddleware != nil {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			s.sendErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		userID = user.UserID
		s.logger.Printf("Upload init by authenticated user: %s (%s)", user.Username, userID)
	} else {
		// Fallback for when auth is disabled
		userID = req.UserID
		if userID == "" {
			userID = "anonymous"
		}
	}
	
	if req.FileName == "" || req.FileSize <= 0 {
		s.sendErrorResponse(w, "Missing required fields (file_name, file_size)", http.StatusBadRequest)
		return
	}
	
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
		UserID:          userID,
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
	var userID string
	if s.authMiddleware != nil {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			s.sendErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		userID = user.UserID
		
		// Check if user owns this file
		if s.dynamoDB != nil {
			fileMetadata, err := s.dynamoDB.GetFileMetadata(r.Context(), fileId)
			if err != nil {
				s.sendErrorResponse(w, "File not found", http.StatusNotFound)
				return
			}
			
			if fileMetadata.OwnerID != userID && fileMetadata.UploadedBy != userID {
				s.logger.Printf("User %s attempted to download file %s owned by %s", userID, fileId, fileMetadata.OwnerID)
				s.sendErrorResponse(w, "Access denied", http.StatusForbidden)
				return
			}
		}
		
		s.logger.Printf("File download by authenticated user: %s (%s)", user.Username, userID)
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
	var userID string
	if s.authMiddleware != nil {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			s.sendErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		userID = user.UserID
		s.logger.Printf("Listing files for authenticated user: %s (%s)", user.Username, userID)
	} else {
		// Fallback for when auth is disabled - show all files
		userID = ""
	}
	
	// Use DynamoDB to list files if available, otherwise fallback to filesystem
	if s.dynamoDB != nil {
		files, err := s.dynamoDB.ListFilesByOwner(r.Context(), userID)
		if err != nil {
			s.logger.Printf("Failed to list files from DynamoDB: %v", err)
			s.sendErrorResponse(w, "Failed to list files", http.StatusInternalServerError)
			return
		}
		
		s.sendSuccessResponse(w, "Files listed successfully", files)
		return
	}
	
	// Fallback to filesystem listing (for backward compatibility)
	uploadsDir := "./storage/uploads"
	
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		s.sendErrorResponse(w, "Failed to access storage", http.StatusInternalServerError)
		return
	}
	
	dirs, err := os.ReadDir(uploadsDir)
	if err != nil {
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
				if !file.IsDir() && filepath.Ext(file.Name()) != ".gz" {
					fileInfo, err := file.Info()
					if err != nil {
						continue
					}
					
					files = append(files, map[string]interface{}{
						"file_id":   fileId,
						"name":      file.Name(),
						"size":      fileInfo.Size(),
						"uploaded":  fileInfo.ModTime(),
						"type":      filepath.Ext(file.Name()),
					})
					break
				}
			}
		}
	}
	
	s.sendSuccessResponse(w, "Files listed successfully", files)
}

func (s *Server) DeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileId := vars["fileId"]
	s.logger.Printf("DeleteFile called for fileId: %s", fileId)
	
	// Get authenticated user from context
	var userID string
	if s.authMiddleware != nil {
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			s.sendErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		userID = user.UserID
		
		// Check if user owns this file
		if s.dynamoDB != nil {
			fileMetadata, err := s.dynamoDB.GetFileMetadata(r.Context(), fileId)
			if err != nil {
				s.sendErrorResponse(w, "File not found", http.StatusNotFound)
				return
			}
			
			if fileMetadata.OwnerID != userID && fileMetadata.UploadedBy != userID {
				s.logger.Printf("User %s attempted to delete file %s owned by %s", userID, fileId, fileMetadata.OwnerID)
				s.sendErrorResponse(w, "Access denied", http.StatusForbidden)
				return
			}
			
			// Delete from DynamoDB
			if err := s.dynamoDB.DeleteFileMetadata(r.Context(), fileId); err != nil {
				s.logger.Printf("Failed to delete file metadata: %v", err)
				s.sendErrorResponse(w, "Failed to delete file", http.StatusInternalServerError)
				return
			}
		}
		
		s.logger.Printf("File deletion by authenticated user: %s (%s)", user.Username, userID)
	}
	
	// Delete from filesystem
	storageDir := filepath.Join("./storage/uploads", fileId)
	if err := os.RemoveAll(storageDir); err != nil {
		s.logger.Printf("Failed to delete file directory: %v", err)
		s.sendErrorResponse(w, "Failed to delete file from storage", http.StatusInternalServerError)
		return
	}
	
	s.sendSuccessResponse(w, "File deleted successfully", map[string]string{
		"file_id": fileId,
		"status":  "deleted",
	})
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