package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client provides a client interface to the consistency controller
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new controller client
func NewClient(controllerURL string) *Client {
	return &Client{
		baseURL: controllerURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetMode queries the controller for the current consistency mode of an object
func (c *Client) GetMode(ctx context.Context, objectID string) (*ModeResponse, error) {
	url := fmt.Sprintf("%s/v1/mode?object_id=%s", c.baseURL, objectID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("controller returned status %d", resp.StatusCode)
	}

	var modeResp ModeResponse
	if err := json.NewDecoder(resp.Body).Decode(&modeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &modeResp, nil
}

// SetHint sets a consistency hint for an object
func (c *Client) SetHint(ctx context.Context, objectID, hint string) error {
	url := fmt.Sprintf("%s/v1/hint", c.baseURL)
	
	reqBody := HintRequest{
		ObjectID: objectID,
		Hint:     hint,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("controller returned status %d", resp.StatusCode)
	}

	return nil
}

// Health checks if the controller is healthy
func (c *Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("controller unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}