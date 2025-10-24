package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"echofs/internal/controller"
	"echofs/internal/metrics"
)

func main() {
	var (
		port         = flag.String("port", "8082", "Controller HTTP port")
		metricsAddr  = flag.String("metrics", "localhost:9090", "Prometheus metrics address")
		pollInterval = flag.Duration("poll", 10*time.Second, "Metrics polling interval")
	)
	flag.Parse()

	log.Println("Starting EchoFS Consistency Controller...")

	metricsClient := metrics.NewPrometheusClient(*metricsAddr)

	ctrl := controller.New(controller.Config{
		MetricsClient:      metricsClient,
		PollInterval:       *pollInterval,
		SampleWindow:       30 * time.Second,
		ConfirmationCount:  3,
		EmergencyThreshold: 0.8,
		CooldownPeriod:     30 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ctrl.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mode", ctrl.HandleGetMode)
	mux.HandleFunc("/v1/hint", ctrl.HandleSetHint)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":" + *port,
		Handler: mux,
	}

	go func() {
		log.Printf("Controller listening on port %s", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down controller...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	cancel()
	log.Println("Controller stopped")
}