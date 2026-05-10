package controller

import (
	"testing"

	"echofs/internal/metadata"
)

func TestStore_RegisterAndGetObject(t *testing.T) {
	store := NewStore()

	obj := &metadata.ObjectMeta{
		FileID:      "file-1",
		Name:        "test.txt",
		CurrentMode: "C",
	}

	store.RegisterObject(obj)

	retrieved := store.GetObject("file-1")
	if retrieved == nil {
		t.Fatal("expected to retrieve registered object")
	}
	if retrieved.FileID != "file-1" {
		t.Errorf("expected file-1, got %s", retrieved.FileID)
	}
	if retrieved.CurrentMode != "C" {
		t.Errorf("expected mode C, got %s", retrieved.CurrentMode)
	}
}

func TestStore_GetObjectReturnsNilForMissing(t *testing.T) {
	store := NewStore()

	obj := store.GetObject("nonexistent")
	if obj != nil {
		t.Error("expected nil for nonexistent object")
	}
}

func TestStore_UpdateObject(t *testing.T) {
	store := NewStore()

	obj := &metadata.ObjectMeta{
		FileID:      "file-2",
		CurrentMode: "C",
	}
	store.RegisterObject(obj)

	// Update
	obj.CurrentMode = "A"
	store.UpdateObject(obj)

	retrieved := store.GetObject("file-2")
	if retrieved.CurrentMode != "A" {
		t.Errorf("expected updated mode A, got %s", retrieved.CurrentMode)
	}
}

func TestStore_DeleteObject(t *testing.T) {
	store := NewStore()

	obj := &metadata.ObjectMeta{FileID: "file-3"}
	store.RegisterObject(obj)

	store.DeleteObject("file-3")

	if store.GetObject("file-3") != nil {
		t.Error("expected nil after deletion")
	}
}

func TestStore_GetAllObjects(t *testing.T) {
	store := NewStore()

	store.RegisterObject(&metadata.ObjectMeta{FileID: "a"})
	store.RegisterObject(&metadata.ObjectMeta{FileID: "b"})
	store.RegisterObject(&metadata.ObjectMeta{FileID: "c"})

	all := store.GetAllObjects()
	if len(all) != 3 {
		t.Errorf("expected 3 objects, got %d", len(all))
	}
}

func TestStore_ObjectCount(t *testing.T) {
	store := NewStore()

	if store.ObjectCount() != 0 {
		t.Error("expected 0 objects initially")
	}

	store.RegisterObject(&metadata.ObjectMeta{FileID: "x"})
	store.RegisterObject(&metadata.ObjectMeta{FileID: "y"})

	if store.ObjectCount() != 2 {
		t.Errorf("expected 2 objects, got %d", store.ObjectCount())
	}
}

func TestStore_GetObjectsByMode(t *testing.T) {
	store := NewStore()

	store.RegisterObject(&metadata.ObjectMeta{FileID: "c1", CurrentMode: "C"})
	store.RegisterObject(&metadata.ObjectMeta{FileID: "c2", CurrentMode: "C"})
	store.RegisterObject(&metadata.ObjectMeta{FileID: "a1", CurrentMode: "A"})

	cObjects := store.GetObjectsByMode("C")
	if len(cObjects) != 2 {
		t.Errorf("expected 2 objects in mode C, got %d", len(cObjects))
	}

	aObjects := store.GetObjectsByMode("A")
	if len(aObjects) != 1 {
		t.Errorf("expected 1 object in mode A, got %d", len(aObjects))
	}
}

func TestStore_RegisterObjectDoesNotOverwrite(t *testing.T) {
	store := NewStore()

	store.RegisterObject(&metadata.ObjectMeta{FileID: "dup", Name: "first"})
	store.RegisterObject(&metadata.ObjectMeta{FileID: "dup", Name: "second"})

	obj := store.GetObject("dup")
	if obj.Name != "first" {
		t.Errorf("RegisterObject should not overwrite existing, got name=%s", obj.Name)
	}
}

func TestStore_GetObjectReturnsCopy(t *testing.T) {
	store := NewStore()

	store.RegisterObject(&metadata.ObjectMeta{FileID: "copy-test", CurrentMode: "C"})

	// Modify the returned object
	retrieved := store.GetObject("copy-test")
	retrieved.CurrentMode = "A"

	// Original should be unchanged
	original := store.GetObject("copy-test")
	if original.CurrentMode != "C" {
		t.Error("GetObject should return a copy, not a reference")
	}
}
