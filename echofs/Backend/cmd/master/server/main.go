package main

import (
	"log"
	"os"
	"echofs/cmd/master/core"
	"echofs/cmd/master/server"
	"echofs/pkg/config"
)

func main() {
	// Create logger
	logger := log.New(os.Stdout, "[MASTER] ", log.LstdFlags|log.Lshortfile)
	
	// Load configuration
	cfg, err := config.LoadMasterConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	
	// Create master node (for now with nil dependencies - we'll implement these later)
	masterNode := core.NewMasterNode(cfg, logger)
	
	// Create HTTP server
	httpServer := server.NewServer(masterNode, logger)
	
	// Start server
	logger.Printf("Starting EchoFS Master Node...")
	if err := httpServer.Start(cfg.Port); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}