package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if basePath == "" {
		basePath = "./storage/files"
	}
	
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &LocalStorage{basePath: basePath}, nil
}

func (ls *LocalStorage) StoreFile(fileID string, reader io.Reader) error {
	filePath := filepath.Join(ls.basePath, fileID)
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

func (ls *LocalStorage) RetrieveFile(fileID string) (io.ReadCloser, error) {
	filePath := filepath.Join(ls.basePath, fileID)
	
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, nil
}

func (ls *LocalStorage) DeleteFile(fileID string) error {
	filePath := filepath.Join(ls.basePath, fileID)
	return os.Remove(filePath)
}

func (ls *LocalStorage) FileExists(fileID string) bool {
	filePath := filepath.Join(ls.basePath, fileID)
	_, err := os.Stat(filePath)
	return err == nil
}