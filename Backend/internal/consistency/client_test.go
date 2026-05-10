package consistency

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testLogger struct{}

func (l *testLogger) Printf(format string, args ...interface{}) {}

func TestGetMode_ReturnsStrongWhenOrchestratorUnavailable(t *testing.T) {
	client := NewClient("http://localhost:99999", &testLogger{})

	mode, reason := client.GetMode(context.Background(), "test-object")

	if mode != ModeStrong {
		t.Errorf("expected ModeStrong when orchestrator unreachable, got %s", mode)
	}
	if reason != "orchestrator_unreachable" {
		t.Errorf("expected reason 'orchestrator_unreachable', got %s", reason)
	}
}

func TestGetMode_ParsesOrchestratorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mode" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		objectID := r.URL.Query().Get("object_id")
		if objectID != "file-123" {
			t.Errorf("expected object_id=file-123, got %s", objectID)
		}

		json.NewEncoder(w).Encode(ModeResponse{
			Mode:   "A",
			TTL:    30,
			Reason: "high_partition_risk",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, &testLogger{})
	mode, reason := client.GetMode(context.Background(), "file-123")

	if mode != ModeAvailable {
		t.Errorf("expected ModeAvailable, got %s", mode)
	}
	if reason != "high_partition_risk" {
		t.Errorf("expected reason 'high_partition_risk', got %s", reason)
	}
}

func TestGetMode_CachesResult(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(ModeResponse{
			Mode:   "C",
			TTL:    60,
			Reason: "low_latency",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, &testLogger{})

	// First call hits the server
	mode1, _ := client.GetMode(context.Background(), "file-456")
	// Second call should use cache
	mode2, _ := client.GetMode(context.Background(), "file-456")

	if mode1 != ModeStrong || mode2 != ModeStrong {
		t.Errorf("expected ModeStrong for both calls, got %s and %s", mode1, mode2)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call (cached), got %d", callCount)
	}
}

func TestGetMode_InvalidateCacheForcesFreshQuery(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(ModeResponse{
			Mode:   "C",
			TTL:    60,
			Reason: "test",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, &testLogger{})

	client.GetMode(context.Background(), "file-789")
	client.InvalidateCache("file-789")
	client.GetMode(context.Background(), "file-789")

	if callCount != 2 {
		t.Errorf("expected 2 server calls after invalidation, got %d", callCount)
	}
}

func TestGetMode_HandlesUnknownMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ModeResponse{
			Mode:   "INVALID",
			TTL:    10,
			Reason: "test",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, &testLogger{})
	mode, _ := client.GetMode(context.Background(), "test")

	if mode != ModeStrong {
		t.Errorf("expected ModeStrong for unknown mode, got %s", mode)
	}
}

func TestGetMode_CacheExpires(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(ModeResponse{
			Mode:   "A",
			TTL:    1, // 1 second TTL
			Reason: "test",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, &testLogger{})

	client.GetMode(context.Background(), "expire-test")
	time.Sleep(1100 * time.Millisecond) // Wait for TTL to expire
	client.GetMode(context.Background(), "expire-test")

	if callCount != 2 {
		t.Errorf("expected 2 calls after TTL expiry, got %d", callCount)
	}
}
