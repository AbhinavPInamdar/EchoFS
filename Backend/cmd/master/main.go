package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"echofs/cmd/master/core"
	"echofs/pkg/config"
)

func main() {
	logger := log.New(os.Stdout, "[MASTER] ", log.LstdFlags|log.Lshortfile)
	
	logger.Println("Loading configuration...")
	cfg, err := config.LoadMasterConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}
	
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}
	
	logger.Printf("Configuration loaded successfully")
	logger.Printf("Server will run on %s:%d", cfg.Host, cfg.Port)
	logger.Printf("Replication factor: %d", cfg.ReplicationFactor)
	logger.Printf("Chunk size: %d bytes", cfg.ChunkSize)
	
	masterNode := core.NewMasterNode(cfg, logger)
	
	// Initialize dependencies
	logger.Println("Initializing dependencies...")
	workerRegistry := core.NewInMemoryWorkerRegistry(logger)
	metadataStore := core.NewInMemoryMetadataStore(logger)
	sessionManager := core.NewInMemorySessionManager(logger)
	chunkPlacer := core.NewSimpleChunkPlacer(workerRegistry, logger)
	healthChecker := core.NewSimpleHealthChecker(workerRegistry, logger)
	
	// Set dependencies
	masterNode.SetDependencies(workerRegistry, metadataStore, sessionManager, chunkPlacer, healthChecker)
	logger.Println("Dependencies initialized successfully")
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	errChan := make(chan error, 1)
	go func() {
		if err := masterNode.Start(ctx); err != nil {
			errChan <- err
		}
	}()
	
	select {
	case err := <-errChan:
		logger.Fatalf("Failed to start master node: %v", err)
	case sig := <-sigChan:
		logger.Printf("Received signal %v, initiating graceful shutdown...", sig)
		cancel()
		
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		if err := masterNode.Stop(shutdownCtx); err != nil {
			logger.Printf("Error during shutdown: %v", err)
		}
		
		logger.Println("Master node shut down complete")
	}
}