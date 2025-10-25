package main

import (
    "fmt"
    "os"
    "log"
    "strconv"
    "path/filepath"
	"net/http"
)

type Worker struct {
	WorkerID string
	WorkerStatus string
	Port int
	Metrics WorkerMetrics
	StoragePath string

}

type WorkerMetrics struct {
	AvailableSpace int64
	CurrentLoad int64
}

func setConfig() Worker{
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "worker1"
	}

	portStr :=	os.Getenv("PORT")
	if portStr == "" {
		portStr = "9082"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("invalid port")
		port = 9082
	}
	return Worker{
		WorkerID: workerID,
		WorkerStatus: "starting",
		Port: port,
		Metrics: WorkerMetrics{},
		StoragePath: "",
	}
}

func SetStoragePath(workerID string) string {
	storagePath := filepath.Join("./storage", workerID,"chunks")
	err := os.MkdirAll(storagePath,0755)

	if err != nil {
		log.Fatalf("Faledd to create Storage Directory")
	}

	return storagePath
}

func main() {
	worker := setConfig()
	worker.StoragePath = SetStoragePath(worker.WorkerID)

	fmt.Printf("Starting %s on port %d\n", worker.WorkerID, worker.Port)
    fmt.Printf("Storage path: %s\n", worker.StoragePath)

	router := setupRoutes()

	fmt.Printf("Worker HTTP server listening on port %d\n", worker.Port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", worker.Port), router))
}