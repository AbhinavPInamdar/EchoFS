package server

import (
	"encoding/json"
	"net/http"
	"log"
	"fmt"
	"github.com/gorilla/mux"
	"echofs/cmd/master/core"
)

type Server struct {
	masterNode *core.MasterNode 
	router     *mux.Router
	logger     *log.Logger
}

func NewServer(masterNode *core.MasterNode, logger *log.Logger) *Server {
	s := &Server{
		masterNode: masterNode,
		logger:     logger,
		router:     mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// File operations
	api.HandleFunc("/files/upload/init", s.InitUpload).Methods("POST")
	api.HandleFunc("/files/upload/chunk", s.UploadChunk).Methods("POST")
	api.HandleFunc("/files/upload/complete", s.CompleteUpload).Methods("POST")
	api.HandleFunc("/files/{fileId}/download", s.DownloadFile).Methods("GET")
	
	// Worker operations
	api.HandleFunc("/workers/register", s.RegisterWorker).Methods("POST")
	api.HandleFunc("/workers/{workerId}/heartbeat", s.WorkerHeartbeat).Methods("POST")
	
	// Health check
	api.HandleFunc("/health", s.HealthCheck).Methods("GET")
}

func (s *Server) Start(port int) error {
	s.logger.Printf("Starting server on port %d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s.router)
}

// Handler functions
func (s *Server) InitUpload(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("InitUpload called")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "upload init - not implemented yet"})
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