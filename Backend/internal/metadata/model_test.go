package metadata

import (
	"testing"
)

func TestNewObjectMeta(t *testing.T) {
	obj := NewObjectMeta("file-1", "test.txt", "user-1", 1024)

	if obj.FileID != "file-1" {
		t.Errorf("expected FileID 'file-1', got '%s'", obj.FileID)
	}
	if obj.Name != "test.txt" {
		t.Errorf("expected Name 'test.txt', got '%s'", obj.Name)
	}
	if obj.Size != 1024 {
		t.Errorf("expected Size 1024, got %d", obj.Size)
	}
	if obj.CurrentMode != "C" {
		t.Errorf("expected default mode 'C', got '%s'", obj.CurrentMode)
	}
	if obj.ModeHint != "Auto" {
		t.Errorf("expected default hint 'Auto', got '%s'", obj.ModeHint)
	}
	if obj.LastVersion != 0 {
		t.Errorf("expected version 0, got %d", obj.LastVersion)
	}
}

func TestObjectMeta_AddChunk(t *testing.T) {
	obj := NewObjectMeta("file-1", "test.txt", "user-1", 2048)

	obj.AddChunk("chunk-0", 0, 1024, "abc123", []string{"worker1", "worker2"})
	obj.AddChunk("chunk-1", 1, 1024, "def456", []string{"worker2", "worker3"})

	if len(obj.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(obj.Chunks))
	}
	if obj.LastVersion != 2 {
		t.Errorf("expected version 2 after 2 chunks, got %d", obj.LastVersion)
	}
	if obj.Chunks[0].ChunkID != "chunk-0" {
		t.Errorf("expected chunk-0, got %s", obj.Chunks[0].ChunkID)
	}
	if obj.Chunks[1].Version != 2 {
		t.Errorf("expected chunk 1 version 2, got %d", obj.Chunks[1].Version)
	}
}

func TestObjectMeta_UpdateVectorClock(t *testing.T) {
	obj := NewObjectMeta("file-1", "test.txt", "user-1", 1024)

	obj.UpdateVectorClock("node-A")
	obj.UpdateVectorClock("node-A")
	obj.UpdateVectorClock("node-B")

	if obj.VectorClock["node-A"] != 2 {
		t.Errorf("expected node-A clock=2, got %d", obj.VectorClock["node-A"])
	}
	if obj.VectorClock["node-B"] != 1 {
		t.Errorf("expected node-B clock=1, got %d", obj.VectorClock["node-B"])
	}
}

func TestObjectMeta_IsNewerThan(t *testing.T) {
	obj1 := NewObjectMeta("f1", "a", "u", 100)
	obj2 := NewObjectMeta("f1", "a", "u", 100)

	obj1.UpdateVectorClock("node-A")
	obj1.UpdateVectorClock("node-A")

	obj2.UpdateVectorClock("node-A")

	if !obj1.IsNewerThan(obj2) {
		t.Error("obj1 should be newer than obj2 (higher clock)")
	}
	if obj2.IsNewerThan(obj1) {
		t.Error("obj2 should NOT be newer than obj1")
	}
}

func TestObjectMeta_HasConflictWith(t *testing.T) {
	obj1 := NewObjectMeta("f1", "a", "u", 100)
	obj2 := NewObjectMeta("f1", "a", "u", 100)

	// Concurrent writes on different nodes — divergent vector clocks
	obj1.UpdateVectorClock("node-A")
	obj2.UpdateVectorClock("node-B")

	// Both have advanced their version independently
	obj1.LastVersion = 1
	obj2.LastVersion = 2 // Different versions

	// Neither is strictly newer: obj1 has node-A=1, obj2 has node-B=1
	// obj1.IsNewerThan(obj2) = false (obj2 has node-B which obj1 doesn't)
	// obj2.IsNewerThan(obj1) = false (obj1 has node-A which obj2 doesn't)
	// And versions differ, so HasConflictWith should be true
	if !obj1.HasConflictWith(obj2) {
		t.Error("expected conflict between concurrent writes on different nodes")
	}
}

func TestObjectMeta_NoConflictWhenOneIsNewer(t *testing.T) {
	obj1 := NewObjectMeta("f1", "a", "u", 100)
	obj2 := NewObjectMeta("f1", "a", "u", 100)

	obj1.UpdateVectorClock("node-A")
	obj1.UpdateVectorClock("node-A")

	obj2.UpdateVectorClock("node-A")

	obj1.LastVersion = 2
	obj2.LastVersion = 1

	if obj1.HasConflictWith(obj2) {
		t.Error("no conflict expected when one is clearly newer")
	}
}
