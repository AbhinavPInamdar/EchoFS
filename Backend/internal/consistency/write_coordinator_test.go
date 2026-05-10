package consistency

import (
	"context"
	"log"
	"os"
	"testing"

	grpcClient "echofs/internal/grpc"
)

func newTestLogger() *log.Logger {
	return log.New(os.Stderr, "[TEST] ", log.LstdFlags)
}

func TestWriteCoordinator_NoWorkersReturnsError(t *testing.T) {
	logger := newTestLogger()
	registry := grpcClient.NewWorkerRegistry(logger)

	wc := NewWriteCoordinator(registry, 3, logger)
	defer wc.Stop()

	result := wc.WriteChunk(context.Background(), ModeStrong, "file-1", "chunk-1", 0, []byte("data"), "md5")

	if result.Success {
		t.Error("expected failure when no workers available")
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestWriteCoordinator_StatsInitiallyZero(t *testing.T) {
	logger := newTestLogger()
	registry := grpcClient.NewWorkerRegistry(logger)

	wc := NewWriteCoordinator(registry, 3, logger)
	defer wc.Stop()

	stats := wc.GetStats()

	if stats["strong_writes"].(int64) != 0 {
		t.Errorf("expected 0 strong writes, got %d", stats["strong_writes"])
	}
	if stats["available_writes"].(int64) != 0 {
		t.Errorf("expected 0 available writes, got %d", stats["available_writes"])
	}
}

func TestWriteCoordinator_StrongModeIncrementsCounter(t *testing.T) {
	logger := newTestLogger()
	registry := grpcClient.NewWorkerRegistry(logger)

	wc := NewWriteCoordinator(registry, 3, logger)
	defer wc.Stop()

	// This will fail (no workers) but should still increment the counter
	wc.WriteChunk(context.Background(), ModeStrong, "file-1", "chunk-1", 0, []byte("data"), "md5")

	stats := wc.GetStats()
	if stats["strong_writes"].(int64) != 1 {
		t.Errorf("expected 1 strong write attempt, got %d", stats["strong_writes"])
	}
}

func TestWriteCoordinator_AvailableModeIncrementsCounter(t *testing.T) {
	logger := newTestLogger()
	registry := grpcClient.NewWorkerRegistry(logger)

	wc := NewWriteCoordinator(registry, 3, logger)
	defer wc.Stop()

	wc.WriteChunk(context.Background(), ModeAvailable, "file-1", "chunk-1", 0, []byte("data"), "md5")

	stats := wc.GetStats()
	if stats["available_writes"].(int64) != 1 {
		t.Errorf("expected 1 available write attempt, got %d", stats["available_writes"])
	}
}

func TestWriteCoordinator_SelectWorkersDistributes(t *testing.T) {
	logger := newTestLogger()
	registry := grpcClient.NewWorkerRegistry(logger)

	wc := NewWriteCoordinator(registry, 3, logger)
	defer wc.Stop()

	// With no workers, selectWorkers returns nil
	workers := wc.selectWorkers(0)
	if workers != nil {
		t.Errorf("expected nil workers when registry is empty, got %v", workers)
	}
}
