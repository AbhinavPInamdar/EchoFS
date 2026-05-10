// Package health provides standard health check utilities for EchoFS services.
package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a service.
type Status struct {
	Service   string            `json:"service"`
	Status    string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]Check  `json:"checks,omitempty"`
}

// Check represents an individual health check result.
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Checker manages health checks for a service.
type Checker struct {
	serviceName string
	checks      map[string]func() Check
	mu          sync.RWMutex
}

// NewChecker creates a new health checker for the given service.
func NewChecker(serviceName string) *Checker {
	return &Checker{
		serviceName: serviceName,
		checks:      make(map[string]func() Check),
	}
}

// AddCheck registers a named health check function.
func (c *Checker) AddCheck(name string, check func() Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// Run executes all registered health checks and returns the overall status.
func (c *Checker) Run() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := Status{
		Service:   c.serviceName,
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]Check),
	}

	for name, check := range c.checks {
		result := check()
		status.Checks[name] = result
		if result.Status == "unhealthy" {
			status.Status = "unhealthy"
		} else if result.Status == "degraded" && status.Status == "healthy" {
			status.Status = "degraded"
		}
	}

	return status
}

// LivenessHandler returns an HTTP handler for liveness probes.
// Always returns 200 if the process is running.
func LivenessHandler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "alive",
			"service": serviceName,
		})
	}
}

// ReadinessHandler returns an HTTP handler for readiness probes.
// Returns 200 only if all health checks pass.
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := c.Run()

		w.Header().Set("Content-Type", "application/json")
		if status.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(status)
	}
}
