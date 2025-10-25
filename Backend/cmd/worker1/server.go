package main

import (
    "fmt"
    "net/http"
    "os"
    "github.com/gorilla/mux"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func getWorkerID() string {
    workerID := os.Getenv("WORKER_ID")
    if workerID == "" {
        return "worker1"
    }
    return workerID
}

func StoreChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkId := vars["chunkId"]
	workerID := getWorkerID()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status": "chunk stored", "chunk_id": "%s", "worker": "%s"}`, chunkId, workerID)))
}

func RetrieveChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkId := vars["chunkId"]
	workerID := getWorkerID()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status": "chunk retrieved", "chunk_id": "%s", "worker": "%s"}`, chunkId, workerID)))
}

func DeleteChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chunkId := vars["chunkId"]
	workerID := getWorkerID()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status": "chunk deleted", "chunk_id": "%s", "worker": "%s"}`, chunkId, workerID)))
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	workerID := getWorkerID()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"status": "healthy", "worker": "%s", "service": "echofs-worker"}`, workerID)))
}

func StatusCheck(w http.ResponseWriter, r *http.Request) {
	workerID := getWorkerID()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"worker": "%s", "available_space": 1000000000, "current_load": 0, "chunks_stored": 0, "protocols": ["http", "grpc"], "grpc_enabled": true}`, workerID)))
}

func setupRoutes() *mux.Router {
	router := mux.NewRouter()
    
    router.HandleFunc("/chunks/{chunkId}", StoreChunk).Methods("POST")
    router.HandleFunc("/chunks/{chunkId}", RetrieveChunk).Methods("GET")
    router.HandleFunc("/chunks/{chunkId}", DeleteChunk).Methods("DELETE")
    router.HandleFunc("/health", HealthCheck).Methods("GET")
    router.HandleFunc("/status", StatusCheck).Methods("GET")
    router.Handle("/metrics", promhttp.Handler()).Methods("GET")
    
    return router

}