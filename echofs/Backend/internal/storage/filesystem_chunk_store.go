package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultStorageRoot = "./storage/chunks"
)

type FSChunkStore struct {
	StorageRoot string
}

func NewFSChunkStore(root string) (*FSChunkStore, error) {
	if root == "" {
		root = DefaultStorageRoot
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create storage root directory %s: %w", root, err)
	}
	return &FSChunkStore{StorageRoot: root}, nil
}

func (f *FSChunkStore) GetChunkPath(chunkID string) string {
	return filepath.Join(f.StorageRoot, chunkID)
}

func (f *FSChunkStore) StoreChunk(ctx context.Context, chunkID string, data []byte) error {
	chunkPath := f.GetChunkPath(chunkID)
	if err := os.WriteFile(chunkPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write chunk %s to disk: %w", chunkID, err)
	}
	return nil
}

func (f *FSChunkStore) RetrieveChunk(ctx context.Context, chunkID string) ([]byte, error) {
	chunkPath := f.GetChunkPath(chunkID)
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chunk %s not found", chunkID)
		}
		return nil, fmt.Errorf("failed to read chunk %s from disk: %w", chunkID, err)
	}
	return data, nil
}

func (f *FSChunkStore) DeleteChunk(ctx context.Context, chunkID string) error {
	chunkPath := f.GetChunkPath(chunkID)
	if err := os.Remove(chunkPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete chunk %s: %w", chunkID, err)
	}
	return nil
}