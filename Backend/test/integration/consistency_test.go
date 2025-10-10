package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"echofs/internal/controller"
	"echofs/internal/metadata"
	"echofs/internal/replication"
)

// TestConsistencyModes tests the adaptive consistency system
func TestConsistencyModes(t *testing.T) {
	// This is an integration test that would require running services
	// For demonstration purposes, we'll show the test structure

	tests := []struct {
		name        string
		consistency string
		hint        string
		expectMode  string
	}{
		{
			name:        "Strong consistency request",
			consistency: "strong",
			hint:        "",
			expectMode:  "C",
		},
		{
			name:        "Available consistency request",
			consistency: "available",
			hint:        "",
			expectMode:  "A",
		},
		{
			name:        "Auto with strong hint",
			consistency: "auto",
			hint:        "Strong",
			expectMode:  "C",
		},
		{
			name:        "Auto with available hint",
			consistency: "auto",
			hint:        "Available",
			expectMode:  "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test upload with consistency parameters
			fileData := []byte("test file content for consistency testing")
			
			response, err := uploadFileWithConsistency(
				"http://localhost:8080/api/v1/files/upload/consistency",
				"test.txt",
				fileData,
				"test-user",
				tt.consistency,
				tt.hint,
			)
			
			if err != nil {
				t.Fatalf("Upload failed: %v", err)
			}

			// Verify consistency mode was applied correctly
			if response.Consistency.ModeUsed != tt.expectMode {
				t.Errorf("Expected mode %s, got %s", tt.expectMode, response.Consistency.ModeUsed)
			}

			// Verify metrics were recorded
			if response.Consistency.Replicas == 0 {
				t.Error("Expected at least 1 replica")
			}

			if response.Consistency.Latency == "" {
				t.Error("Expected latency to be recorded")
			}

			t.Logf("Upload successful: mode=%s, replicas=%d, latency=%s",
				response.Consistency.ModeUsed,
				response.Consistency.Replicas,
				response.Consistency.Latency)
		})
	}
}

// TestControllerDecisionMaking tests the controller's decision-making logic
func TestControllerDecisionMaking(t *testing.T) {
	// Test controller policy decisions
	policy := controller.NewPolicy()
	
	testCases := []struct {
		name     string
		metrics  controller.ObjectMetrics
		expected string
	}{
		{
			name: "High partition risk should favor Available mode",
			metrics: controller.ObjectMetrics{
				PartitionRisk:    0.8,
				ReplicationLag:   10 * time.Millisecond,
				WriteRate:        5.0,
				TransitionReason: "",
			},
			expected: "A",
		},
		{
			name: "Low latency should favor Strong mode",
			metrics: controller.ObjectMetrics{
				PartitionRisk:    0.1,
				ReplicationLag:   5 * time.Millisecond,
				WriteRate:        2.0,
				TransitionReason: "",
			},
			expected: "C",
		},
		{
			name: "Medium conditions should favor Hybrid mode",
			metrics: controller.ObjectMetrics{
				PartitionRisk:    0.4,
				ReplicationLag:   50 * time.Millisecond,
				WriteRate:        25.0,
				TransitionReason: "",
			},
			expected: "Hybrid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock object metadata
			objMeta := &metadata.ObjectMeta{
				FileID:      "test-file",
				ModeHint:    "Auto",
				CurrentMode: "C",
			}

			currentState := &controller.ObjectModeState{
				ObjectID:    "test-file",
				CurrentMode: "C",
				LastChange:  time.Now().Add(-5 * time.Minute), // Old enough to avoid penalty
			}

			decision := policy.DecideMode(*objMeta, tc.metrics, currentState)
			
			if decision != tc.expected {
				t.Errorf("Expected %s, got %s for case: %s", tc.expected, decision, tc.name)
			}
		})
	}
}

// TestReplicationStrategies tests sync vs async replication
func TestReplicationStrategies(t *testing.T) {
	config := replication.ReplicationConfig{
		QuorumSize:        2,
		WriteTimeout:      5 * time.Second,
		ReplicationFactor: 3,
		AsyncQueueSize:    100,
		WorkerNodes:       []string{"worker1:8091", "worker2:8092", "worker3:8093"},
	}

	replicationMgr := replication.NewReplicationManager(config)

	t.Run("Sync strategy performance", func(t *testing.T) {
		syncStrategy := replicationMgr.GetSyncStrategy()
		
		// Test write performance
		ctx := context.Background()
		objMeta := &metadata.ObjectMeta{
			FileID:      "sync-test-file",
			CurrentMode: "C",
		}
		
		testData := []byte("test data for sync replication")
		
		start := time.Now()
		result, err := syncStrategy.Write(ctx, objMeta, testData)
		latency := time.Since(start)
		
		if err != nil {
			t.Fatalf("Sync write failed: %v", err)
		}

		if !result.Acked {
			t.Error("Expected write to be acknowledged")
		}

		if latency > 100*time.Millisecond {
			t.Errorf("Sync write took too long: %v", latency)
		}

		t.Logf("Sync write: latency=%v, replicas=%d", latency, result.Replicas)
	})

	t.Run("Async strategy performance", func(t *testing.T) {
		asyncStrategy := replicationMgr.GetAsyncStrategy()
		
		// Test write performance
		ctx := context.Background()
		objMeta := &metadata.ObjectMeta{
			FileID:      "async-test-file",
			CurrentMode: "A",
		}
		
		testData := []byte("test data for async replication")
		
		start := time.Now()
		result, err := asyncStrategy.Write(ctx, objMeta, testData)
		latency := time.Since(start)
		
		if err != nil {
			t.Fatalf("Async write failed: %v", err)
		}

		if !result.Acked {
			t.Error("Expected write to be acknowledged immediately")
		}

		// Async should be much faster (only primary write)
		if latency > 50*time.Millisecond {
			t.Errorf("Async write took too long: %v", latency)
		}

		t.Logf("Async write: latency=%v, replicas=%d", latency, result.Replicas)
		
		// Check that background replication was queued
		queueSize := asyncStrategy.GetQueueSize()
		if queueSize == 0 {
			t.Error("Expected background replication to be queued")
		}
	})
}

// Helper functions

type UploadResponse struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	Consistency *ConsistencyInfo `json:"consistency"`
}

type ConsistencyInfo struct {
	ModeUsed  string `json:"mode_used"`
	Version   int64  `json:"version"`
	Replicas  int    `json:"replicas"`
	Latency   string `json:"latency"`
}

func uploadFileWithConsistency(url, filename string, data []byte, userID, consistency, hint string) (*UploadResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	fileWriter.Write(data)

	// Add form fields
	writer.WriteField("user_id", userID)
	writer.WriteField("consistency", consistency)
	if hint != "" {
		writer.WriteField("hint", hint)
	}

	writer.Close()

	// Make request
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}