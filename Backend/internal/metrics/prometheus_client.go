package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RealPrometheusClient queries a live Prometheus instance for metrics.
type RealPrometheusClient struct {
	endpoint   string
	httpClient *http.Client
}

// PromQueryResult represents a Prometheus instant query result.
type PromQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"` // [timestamp, value_string]
		} `json:"result"`
	} `json:"data"`
}

// NewRealPrometheusClient creates a client that queries a live Prometheus server.
func NewRealPrometheusClient(endpoint string) *RealPrometheusClient {
	return &RealPrometheusClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *RealPrometheusClient) Query(query string) (interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/query?query=%s", c.endpoint, query)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	var result PromQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	return &result, nil
}

func (c *RealPrometheusClient) QueryRange(query string, start, end time.Time, step time.Duration) (interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
		c.endpoint, query, start.Unix(), end.Unix(), step.String())

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("prometheus range query failed: %w", err)
	}
	defer resp.Body.Close()

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	return result, nil
}

// QueryScalar executes a query and returns the first result as a float64.
// Returns 0 and no error if no results are found.
func (c *RealPrometheusClient) QueryScalar(query string) (float64, error) {
	result, err := c.Query(query)
	if err != nil {
		return 0, err
	}

	promResult, ok := result.(*PromQueryResult)
	if !ok || promResult.Status != "success" {
		return 0, fmt.Errorf("unexpected prometheus response")
	}

	if len(promResult.Data.Result) == 0 {
		return 0, nil // No data yet
	}

	// Value is [timestamp, "value_string"]
	valueStr, ok := promResult.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("unexpected value type in prometheus response")
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse prometheus value: %w", err)
	}

	return value, nil
}
