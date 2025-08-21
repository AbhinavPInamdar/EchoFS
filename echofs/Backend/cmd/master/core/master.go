package core
// internal/master/core/master.go
package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"echofs/pkg/config"
)

// MasterNode represents the main master node instance
type MasterNode struct {
	// Configuration
	config *config.MasterConfig
	logger *log.Logger
	
	// Core components
	workerRegistry WorkerRegistry
	metadataStore  MetadataStore
	sessionManager SessionManager
	chunkPlacer    ChunkPlacer
	healthChecker  HealthChecker
	
	// State management
	uploadSessions   map[string]*UploadSession
	downloadSessions map[string]*DownloadSession
	sessionsMutex    sync.RWMutex
	
	// Background workers
	cleanupTicker    *time.Ticker
	healthTicker     *time.Ticker
	
	// Lifecycle management
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	running   bool
	runMutex  sync.RWMutex
}

// DownloadSession represents a file download session
type DownloadSession struct {
	SessionID   string    `json:"session_id"`
	FileID      string    `json:"file_id"`
	UserID      string    `json:"user_id"`
	Status      SessionStatus `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	
	// Internal state
	chunks      []*ChunkMetadata
	mutex       sync.RWMutex
}

// NewMasterNode creates a new master node instance
func NewMasterNode(config *config.MasterConfig, logger *log.Logger) *MasterNode {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MasterNode{
		config:           config,
		logger:           logger,
		uploadSessions:   make(map[string]*UploadSession),
		downloadSessions: make(map[string]*DownloadSession),
		ctx:              ctx,
		cancel:           cancel,
		running:          false,
	}
}

// SetDependencies injects the required dependencies
func (m *MasterNode) SetDependencies(
	workerRegistry WorkerRegistry,
	metadataStore MetadataStore,
	sessionManager SessionManager,
	chunkPlacer ChunkPlacer,
	healthChecker HealthChecker,
) {
	m.workerRegistry = workerRegistry
	m.metadataStore = metadataStore
	m.sessionManager = sessionManager
	m.chunkPlacer = chunkPlacer
	m.healthChecker = healthChecker
}

// Start initializes and starts the master node
func (m *MasterNode) Start(ctx context.Context) error {
	m.runMutex.Lock()
	defer m.runMutex.Unlock()
	
	if m.running {
		return fmt.Errorf("master node is already running")
	}
	
	m.logger.Printf("Starting master node on %s:%d", m.config.Host, m.config.Port)
	
	// Start background services
	if err := m.startBackgroundServices(); err != nil {
		return fmt.Errorf("failed to start background services: %w", err)
	}
	
	// Start health checker
	if err := m.healthChecker.StartHealthChecking(m.ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}
	
	m.running = true
	m.logger.Println("Master node started successfully")
	
	return nil
}

// Stop gracefully shuts down the master node
func (m *MasterNode) Stop(ctx context.Context) error {
	m.runMutex.Lock()
	defer m.runMutex.Unlock()
	
	if !m.running {
		return nil
	}
	
	m.logger.Println("Shutting down master node...")
	
	// Cancel context to signal shutdown
	m.cancel()
	
	// Stop background services
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	if m.healthTicker != nil {
		m.healthTicker.Stop()
	}
	
	// Stop health checker
	if err := m.healthChecker.StopHealthChecking(ctx); err != nil {
		m.logger.Printf("Error stopping health checker: %v", err)
	}
	
	// Wait for background goroutines to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		m.logger.Println("All background services stopped")
	case <-time.After(10 * time.Second):
		m.logger.Println("Timeout waiting for background services to stop")
	}
	
	m.running = false
	m.logger.Println("Master node stopped")
	
	return nil
}

// IsRunning returns whether the master node is currently running
func (m *MasterNode) IsRunning() bool {
	m.runMutex.RLock()
	defer m.runMutex.RUnlock()
	return m.running
}

// startBackgroundServices starts all background services
func (m *MasterNode) startBackgroundServices() error {
	// Start cleanup service
	m.cleanupTicker = time.NewTicker(m.config.CleanupInterval)
	m.wg.Add(1)
	go m.cleanupService()
	
	// Start session monitoring
	m.wg.Add(1)
	go m.sessionMonitoringService()
	
	// Start metrics collection (if enabled)
	if m.config.MetricsEnabled {
		m.wg.Add(1)
		go m.metricsService()
	}
	
	return nil
}

// cleanupService handles periodic cleanup tasks
func (m *MasterNode) cleanupService() {
	defer m.wg.Done()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.cleanupTicker.C:
			m.performCleanup()
		}
	}
}

// sessionMonitoringService monitors session health and expiry
func (m *MasterNode) sessionMonitoringService() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.monitorSessions()
		}
	}
}

// metricsService collects and reports metrics
func (m *MasterNode) metricsService() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

// performCleanup performs periodic cleanup tasks
func (m *MasterNode) performCleanup() {
	ctx := context.Background()
	
	// Clean up expired sessions
	if err := m.sessionManager.CleanupExpiredSessions(ctx); err != nil {
		m.logger.Printf("Error cleaning up expired sessions: %v", err)
	}
	
	// Clean up in-memory sessions
	m.cleanupExpiredInMemorySessions()
	
	m.logger.Println("Cleanup completed")
}

// cleanupExpiredInMemorySessions removes expired sessions from memory
func (m *MasterNode) cleanupExpiredInMemorySessions() {
	now := time.Now()
	
	m.sessionsMutex.Lock()
	defer m.sessionsMutex.Unlock()
	
	// Clean upload sessions
	for sessionID, session := range m.uploadSessions {
		if now.After(session.ExpiresAt) {
			delete(m.uploadSessions, sessionID)
		}
	}
	
	// Clean download sessions
	for sessionID, session := range m.downloadSessions {
		if now.After(session.ExpiresAt) {
			delete(m.downloadSessions, sessionID)
		}
	}
}

// monitorSessions monitors session health and handles failures
func (m *MasterNode) monitorSessions() {
	m.sessionsMutex.RLock()
	uploadSessions := make([]*UploadSession, 0, len(m.uploadSessions))
	for _, session := range m.uploadSessions {
		uploadSessions = append(uploadSessions, session)
	}
	m.sessionsMutex.RUnlock()
	
	// Check upload sessions for timeouts or failures
	for _, session := range uploadSessions {
		session.mutex.RLock()
		status := session.Status
		createdAt := session.CreatedAt
		session.mutex.RUnlock()
		
		if status == SessionStatusActive && time.Since(createdAt) > m.config.SessionTimeout {
			m.logger.Printf("Session %s timed out, marking as failed", session.SessionID)
			m.updateSessionStatus(session.SessionID, SessionStatusFailed)
		}
	}
}

// collectMetrics collects and reports system metrics
func (m *MasterNode) collectMetrics() {
	// Get worker count
	workers, err := m.workerRegistry.GetHealthyWorkers(context.Background())
	if err != nil {
		m.logger.Printf("Error getting healthy workers for metrics: %v", err)
		return
	}
	
	// Get session counts
	m.sessionsMutex.RLock()
	uploadSessionCount := len(m.uploadSessions)
	downloadSessionCount := len(m.downloadSessions)
	m.sessionsMutex.RUnlock()
	
	m.logger.Printf("Metrics - Workers: %d, Upload Sessions: %d, Download Sessions: %d",
		len(workers), uploadSessionCount, downloadSessionCount)
}

// updateSessionStatus updates the status of a session
func (m *MasterNode) updateSessionStatus(sessionID string, status SessionStatus) {
	m.sessionsMutex.Lock()
	defer m.sessionsMutex.Unlock()
	
	if session, exists := m.uploadSessions[sessionID]; exists {
		session.mutex.Lock()
		session.Status = status
		session.mutex.Unlock()
	}
}

// GetUploadSession retrieves an upload session
func (m *MasterNode) GetUploadSession(sessionID string) (*UploadSession, bool) {
	m.sessionsMutex.RLock()
	defer m.sessionsMutex.RUnlock()
	
	session, exists := m.uploadSessions[sessionID]
	return session, exists
}

// AddUploadSession adds a new upload session
func (m *MasterNode) AddUploadSession(session *UploadSession) {
	m.sessionsMutex.Lock()
	defer m.sessionsMutex.Unlock()
	
	m.uploadSessions[session.SessionID] = session
}

// RemoveUploadSession removes an upload session
func (m *MasterNode) RemoveUploadSession(sessionID string) {
	m.sessionsMutex.Lock()
	defer m.sessionsMutex.Unlock()
	
	delete(m.uploadSessions, sessionID)
}