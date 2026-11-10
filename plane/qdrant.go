package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// QdrantClient queries named vectors through Qdrant's REST API. It is kept
// deliberately small so the plane package does not need a Qdrant SDK or any
// generated transport code.
type QdrantClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewQdrantClient creates a client for a Qdrant REST endpoint.
func NewQdrantClient(baseURL string) *QdrantClient {
	return &QdrantClient{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: http.DefaultClient}
}

// QdrantPoint is one point returned by a named-vector search. ID is kept as
// raw JSON because Qdrant accepts both numeric and string point ids.
type QdrantPoint struct {
	ID      json.RawMessage `json:"id"`
	Score   float64         `json:"score"`
	Payload map[string]any  `json:"payload,omitempty"`
}

type qdrantSearchRequest struct {
	Vector      qdrantNamedVector `json:"vector"`
	Limit       int               `json:"limit"`
	WithPayload bool              `json:"with_payload"`
	WithVectors bool              `json:"with_vectors"`
}

type qdrantNamedVector struct {
	Name   string    `json:"name"`
	Vector []float64 `json:"vector"`
}

type qdrantSearchResponse struct {
	Result []QdrantPoint `json:"result"`
}

// Search queries one named vector in a collection using cosine/distance
// semantics configured on that collection. The caller must provide the query
// vector produced by the same plane weights used to index the collection.
func (c *QdrantClient) Search(ctx context.Context, collection, vectorName string, vector []float64, limit int) ([]QdrantPoint, error) {
	if c == nil {
		return nil, fmt.Errorf("plane: nil Qdrant client")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return nil, fmt.Errorf("plane: Qdrant endpoint is required")
	}
	if strings.TrimSpace(collection) == "" {
		return nil, fmt.Errorf("plane: Qdrant collection is required")
	}
	if strings.TrimSpace(vectorName) == "" {
		return nil, fmt.Errorf("plane: Qdrant vector name is required")
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("plane: Qdrant query vector is empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("plane: Qdrant result limit must be positive")
	}

	body, err := json.Marshal(qdrantSearchRequest{
		Vector:      qdrantNamedVector{Name: vectorName, Vector: vector},
		Limit:       limit,
		WithPayload: true,
		WithVectors: false,
	})
	if err != nil {
		return nil, fmt.Errorf("plane: encode Qdrant search: %w", err)
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + "/collections/" + url.PathEscape(collection) + "/points/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("plane: create Qdrant search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("api-key", c.APIKey)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plane: Qdrant search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("plane: Qdrant search returned %s: %s", resp.Status, strings.TrimSpace(string(errorBody)))
	}

	var result qdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("plane: decode Qdrant search response: %w", err)
	}
	return result.Result, nil
}
