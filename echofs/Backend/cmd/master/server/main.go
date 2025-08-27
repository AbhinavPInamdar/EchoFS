package main

import (
	"log"
	"os"
	"echofs/cmd/master/core"
	"echofs/pkg/config"
)

func main() {
	logger := log.New(os.Stdout, "[MASTER] ", log.LstdFlags|log.Lshortfile)
	
	cfg, err := config.LoadMasterConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	
	masterNode := core.NewMasterNode(cfg, logger)
	
	httpServer := NewServer(masterNode, logger)
	
	logger.Printf("Starting EchoFS Master Node...")
	if err := httpServer.Start(cfg.Port); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}