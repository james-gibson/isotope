package plane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestQdrantSearchUsesNamedVector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/kb/points/search" {
			t.Fatalf("request = %s %s, want POST /collections/kb/points/search", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("api-key"); got != "secret" {
			t.Fatalf("api-key = %q, want secret", got)
		}
		var body struct {
			Vector struct {
				Name   string    `json:"name"`
				Values []float64 `json:"vector"`
			} `json:"vector"`
			Limit       int  `json:"limit"`
			WithPayload bool `json:"with_payload"`
			WithVectors bool `json:"with_vectors"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Vector.Name != "plane" || !reflect.DeepEqual(body.Vector.Values, []float64{1, 2, 3, 4}) {
			t.Fatalf("vector = %#v, want named plane [1 2 3 4]", body.Vector)
		}
		if body.Limit != 7 || !body.WithPayload || body.WithVectors {
			t.Fatalf("options = limit %d payload %t vectors %t", body.Limit, body.WithPayload, body.WithVectors)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"id":"doc-1","score":0.75,"payload":{"universe_id":"example:one:2026"}}]}`))
	}))
	defer server.Close()

	client := NewQdrantClient(server.URL)
	client.APIKey = "secret"
	client.HTTPClient = server.Client()
	got, err := client.Search(context.Background(), "kb", "plane", []float64{1, 2, 3, 4}, 7)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || string(got[0].ID) != `"doc-1"` || got[0].Score != 0.75 {
		t.Fatalf("results = %#v", got)
	}
}

func TestQdrantSearchReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"status":"failed"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := NewQdrantClient(server.URL).Search(context.Background(), "kb", "plane", []float64{1}, 1)
	if err == nil {
		t.Fatal("Search succeeded for an HTTP error")
	}
}
