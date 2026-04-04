package isotope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client calls smoke-alarm's isotope REST endpoints.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client targeting the given smoke-alarm base URL (e.g. "http://127.0.0.1:8088").
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Register POSTs to /isotope/register and returns the assigned Record.
func (c *Client) Register(ctx context.Context, reg Registration) (*Record, error) {
	body, err := json.Marshal(reg)
	if err != nil {
		return nil, fmt.Errorf("isotope register: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/isotope/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("isotope register: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("isotope register: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("isotope register: status %d", resp.StatusCode)
	}
	var record Record
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("isotope register: decode response: %w", err)
	}
	return &record, nil
}

// List GETs /isotope/list and returns all registered isotope records.
func (c *Client) List(ctx context.Context) ([]Record, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/isotope/list", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("isotope list: build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("isotope list: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("isotope list: status %d", resp.StatusCode)
	}
	var records []Record
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("isotope list: decode response: %w", err)
	}
	return records, nil
}
