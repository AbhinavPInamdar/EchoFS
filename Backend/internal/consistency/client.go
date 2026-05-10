package consistency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Mode represents the consistency mode for an object
type Mode string

const (
	ModeStrong    Mode = "C"  // Strong consistency - quorum writes
	ModeAvailable Mode = "A"  // Available - single write, async replicate
	ModeHybrid    Mode = "Hybrid"
)

// ModeResponse is the response from the consistency controller
type ModeResponse struct {
	Mode      string `json:"mode"`
	TTL       int    `json:"ttl_seconds"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

// Client communicates with the Consistency Orchestrator to determine
// the appropriate write strategy for each object.
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      map[string]*cachedMode
	cacheMu    sync.RWMutex
	logger     interface{ Printf(string, ...interface{}) }
}

type cachedMode struct {
	mode      Mode
	reason    string
	expiresAt time.Time
}

// NewClient creates a new consistency client that queries the orchestrator.
func NewClient(orchestratorURL string, logger interface{ Printf(string, ...interface{}) }) *Client {
	return &Client{
		baseURL: orchestratorURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second, // Fast timeout - don't block data path
		},
		cache:  make(map[string]*cachedMode),
		logger: logger,
	}
}

// GetMode returns the consistency mode for a given object.
// It uses a local cache with TTL to avoid hitting the orchestrator on every request.
// If the orchestrator is unreachable, it defaults to Strong consistency (safe fallback).
func (c *Client) GetMode(ctx context.Context, objectID string) (Mode, string) {
	// Check cache first
	c.cacheMu.RLock()
	if cached, ok := c.cache[objectID]; ok && time.Now().Before(cached.expiresAt) {
		c.cacheMu.RUnlock()
		return cached.mode, cached.reason
	}
	c.cacheMu.RUnlock()

	// Query the orchestrator
	mode, reason, ttl := c.queryOrchestrator(ctx, objectID)

	// Cache the result
	c.cacheMu.Lock()
	c.cache[objectID] = &cachedMode{
		mode:      mode,
		reason:    reason,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
	c.cacheMu.Unlock()

	return mode, reason
}

// InvalidateCache removes a cached mode entry, forcing the next call to query the orchestrator.
func (c *Client) InvalidateCache(objectID string) {
	c.cacheMu.Lock()
	delete(c.cache, objectID)
	c.cacheMu.Unlock()
}

// RegisterObject registers a new object with the consistency orchestrator.
func (c *Client) RegisterObject(ctx context.Context, objectID, name string, size int64) error {
	url := fmt.Sprintf("%s/v1/register", c.baseURL)

	body := fmt.Sprintf(`{"object_id":"%s","name":"%s","size":%d}`, objectID, name, size)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, 
		jsonReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Warning: Failed to register object %s with orchestrator: %v", objectID, err)
		return nil // Non-fatal - object will get default mode
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		c.logger.Printf("Warning: Orchestrator returned %d for object registration %s", resp.StatusCode, objectID)
	}

	return nil
}

func (c *Client) queryOrchestrator(ctx context.Context, objectID string) (Mode, string, int) {
	url := fmt.Sprintf("%s/v1/mode?object_id=%s", c.baseURL, objectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Printf("Warning: Failed to create orchestrator request: %v", err)
		return ModeStrong, "orchestrator_unreachable", 10
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Printf("Warning: Orchestrator unreachable at %s: %v (defaulting to Strong)", c.baseURL, err)
		return ModeStrong, "orchestrator_unreachable", 10
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Printf("Warning: Orchestrator returned %d (defaulting to Strong)", resp.StatusCode)
		return ModeStrong, "orchestrator_error", 10
	}

	var modeResp ModeResponse
	if err := json.NewDecoder(resp.Body).Decode(&modeResp); err != nil {
		c.logger.Printf("Warning: Failed to decode orchestrator response: %v", err)
		return ModeStrong, "decode_error", 10
	}

	mode := Mode(modeResp.Mode)
	switch mode {
	case ModeStrong, ModeAvailable, ModeHybrid:
		return mode, modeResp.Reason, modeResp.TTL
	default:
		c.logger.Printf("Warning: Unknown mode %q from orchestrator, defaulting to Strong", modeResp.Mode)
		return ModeStrong, "unknown_mode", 10
	}
}

func jsonReader(s string) *stringReader {
	return &stringReader{data: []byte(s), pos: 0}
}

type stringReader struct {
	data []byte
	pos  int
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, io.EOF
	}
	return n, nil
}
