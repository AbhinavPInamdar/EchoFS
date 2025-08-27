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
	// Set up logging
	logger := log.New(os.Stdout, "[MASTER] ", log.LstdFlags|log.Lshortfile)
	
	// Load configuration
	logger.Println("Loading configuration...")
	cfg, err := config.LoadMasterConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("Invalid configuration: %v", err)
	}
	
	logger.Printf("Configuration loaded successfully")
	logger.Printf("Server will run on %s:%d", cfg.Host, cfg.Port)
	logger.Printf("Replication factor: %d", cfg.ReplicationFactor)
	logger.Printf("Chunk size: %d bytes", cfg.ChunkSize)
	
	// Create master node
	masterNode := core.NewMasterNode(cfg, logger)
	
	// TODO: Initialize dependencies (we'll implement these in subsequent steps)
	// For now, we'll use placeholder implementations
	logger.Println("TODO: Initialize worker registry, metadata store, etc.")
	
	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Start master node in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := masterNode.Start(ctx); err != nil {
			errChan <- err
		}
	}()
	
	// Wait for either an error or shutdown signal
	select {
	case err := <-errChan:
		logger.Fatalf("Failed to start master node: %v", err)
	case sig := <-sigChan:
		logger.Printf("Received signal %v, initiating graceful shutdown...", sig)
		cancel()
		
		// Give the master node time to shut down gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		if err := masterNode.Stop(shutdownCtx); err != nil {
			logger.Printf("Error during shutdown: %v", err)
		}
		
		logger.Println("Master node shut down complete")
	}
}