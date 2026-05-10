// Metadata service — file and user metadata management.
// Owns the PostgreSQL connection. Provides CRUD for users and files.
// Other services communicate with this via HTTP.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"echofs/internal/api"
	"echofs/internal/metadata"
	"echofs/pkg/auth"
	"echofs/pkg/database"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

type MetadataService struct {
	router     *mux.Router
	logger     *log.Logger
	userRepo   *database.UserRepository
	fileRepo   *database.FileRepository
	jwtManager *auth.JWTManager
	authHandler *api.AuthHandler
}

func main() {
	logger := log.New(os.Stdout, "[METADATA] ", log.LstdFlags|log.Lshortfile)

	// Database
	databaseURL := getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/echofs?sslmode=disable")
	postgresDB, err := database.NewPostgresDB(databaseURL, 25, 30*time.Second)
	if err != nil {
		logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	ctx := context.Background()
	userRepo := database.NewUserRepository(postgresDB)
	fileRepo := database.NewFileRepository(postgresDB)

	if err := userRepo.InitSchema(ctx); err != nil {
		logger.Fatalf("Failed to initialize user schema: %v", err)
	}
	if err := fileRepo.InitSchema(ctx); err != nil {
		logger.Fatalf("Failed to initialize file schema: %v", err)
	}

	// JWT (for auth endpoints)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET environment variable is required")
	}
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
	authHandler := api.NewAuthHandler(userRepo, jwtManager, logger)

	svc := &MetadataService{
		router:      mux.NewRouter(),
		logger:      logger,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		jwtManager:  jwtManager,
		authHandler: authHandler,
	}

	svc.setupRoutes()

	port := getEnv("PORT", "8083")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      svc.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		logger.Printf("Metadata service listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Println("Shutting down metadata service...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	logger.Println("Metadata service stopped")
}

func (svc *MetadataService) setupRoutes() {
	// Public auth routes (proxied from gateway)
	svc.router.HandleFunc("/api/v1/auth/register", svc.authHandler.Register).Methods("POST")
	svc.router.HandleFunc("/api/v1/auth/login", svc.authHandler.Login).Methods("POST")
	svc.router.HandleFunc("/api/v1/auth/profile", svc.getProfile).Methods("GET")

	// File metadata routes (user ID comes from X-User-ID header set by gateway)
	svc.router.HandleFunc("/api/v1/files", svc.listFiles).Methods("GET")
	svc.router.HandleFunc("/api/v1/files/{fileId}", svc.deleteFile).Methods("DELETE")

	// Internal routes (called by ingest service, not exposed publicly)
	svc.router.HandleFunc("/internal/files", svc.createFile).Methods("POST")

	// Health
	svc.router.HandleFunc("/health", svc.healthCheck).Methods("GET")
}

func (svc *MetadataService) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "echofs-metadata"})
}

func (svc *MetadataService) getProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"success":false,"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		http.Error(w, `{"success":false,"message":"User not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "user": user})
}

func (svc *MetadataService) listFiles(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"success":false,"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	files, err := svc.fileRepo.GetFilesByOwner(ctx, userID)
	if err != nil {
		svc.logger.Printf("Failed to list files: %v", err)
		http.Error(w, `{"success":false,"message":"Failed to list files"}`, http.StatusInternalServerError)
		return
	}

	var fileList []map[string]interface{}
	for _, f := range files {
		fileList = append(fileList, map[string]interface{}{
			"file_id":  f.FileID,
			"name":     f.OriginalName,
			"size":     f.Size,
			"uploaded": f.CreatedAt.Format(time.RFC3339),
			"status":   f.Status,
			"chunks":   f.TotalChunks,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Files listed successfully",
		"data":    fileList,
	})
}

func (svc *MetadataService) deleteFile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, `{"success":false,"message":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	fileID := vars["fileId"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := svc.fileRepo.DeleteFile(ctx, fileID, userID); err != nil {
		if err == database.ErrUserNotFound {
			http.Error(w, `{"success":false,"message":"File not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"success":false,"message":"Failed to delete"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "File deleted"})
}

// createFile is an internal endpoint called by the ingest service after successful upload.
func (svc *MetadataService) createFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileID       string `json:"file_id"`
		OwnerID      string `json:"owner_id"`
		OriginalName string `json:"original_name"`
		Size         int64  `json:"size"`
		TotalChunks  int    `json:"total_chunks"`
		Status       string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"success":false,"message":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	fileMeta := &metadata.FileMetadata{
		FileID:       req.FileID,
		OwnerID:      req.OwnerID,
		OriginalName: req.OriginalName,
		Size:         req.Size,
		ChunkSize:    1024 * 1024,
		TotalChunks:  req.TotalChunks,
		Status:       req.Status,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := svc.fileRepo.CreateFile(ctx, fileMeta); err != nil {
		svc.logger.Printf("Failed to create file metadata: %v", err)
		http.Error(w, `{"success":false,"message":"Failed to save metadata"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "file_id": req.FileID})
}
