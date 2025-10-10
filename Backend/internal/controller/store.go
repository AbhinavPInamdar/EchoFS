package controller

import (
	"sync"

	"echofs/internal/metadata"
)

// Store manages object metadata for the consistency controller
type Store struct {
	mu      sync.RWMutex
	objects map[string]*metadata.ObjectMeta
}

func NewStore() *Store {
	return &Store{
		objects: make(map[string]*metadata.ObjectMeta),
	}
}

func (s *Store) GetObject(objectID string) *metadata.ObjectMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	obj, exists := s.objects[objectID]
	if !exists {
		return nil
	}
	
	// Return a copy to prevent external modification
	objCopy := *obj
	return &objCopy
}

func (s *Store) UpdateObject(obj *metadata.ObjectMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Store a copy to prevent external modification
	objCopy := *obj
	s.objects[obj.FileID] = &objCopy
}

func (s *Store) DeleteObject(objectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.objects, objectID)
}

func (s *Store) GetAllObjects() []*metadata.ObjectMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	objects := make([]*metadata.ObjectMeta, 0, len(s.objects))
	for _, obj := range s.objects {
		// Return copies to prevent external modification
		objCopy := *obj
		objects = append(objects, &objCopy)
	}
	
	return objects
}

func (s *Store) ListObjectIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	ids := make([]string, 0, len(s.objects))
	for id := range s.objects {
		ids = append(ids, id)
	}
	
	return ids
}

func (s *Store) ObjectCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return len(s.objects)
}

// RegisterObject adds a new object to the store if it doesn't exist
func (s *Store) RegisterObject(obj *metadata.ObjectMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.objects[obj.FileID]; !exists {
		objCopy := *obj
		s.objects[obj.FileID] = &objCopy
	}
}

// GetObjectsByMode returns all objects currently in the specified mode
func (s *Store) GetObjectsByMode(mode string) []*metadata.ObjectMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var objects []*metadata.ObjectMeta
	for _, obj := range s.objects {
		if obj.CurrentMode == mode {
			objCopy := *obj
			objects = append(objects, &objCopy)
		}
	}
	
	return objects
}