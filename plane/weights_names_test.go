package plane

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestWeightsRoundTripNames pins that plane names — which carry universe ids for
// a named corpus — survive MarshalWeights -> LoadWeights. Without this, a
// reloaded model regenerates structural names (p<j>:q<charge>) and can no longer
// resolve an id selector, so `plane sort <universe-id>` fails.
func TestWeightsRoundTripNames(t *testing.T) {
	m := mustModel(t, []int{1, -1, 1})
	want := []string{"example:alpha:1", "example:beta:2", "example:gamma:3"}
	for j := range want {
		m.Space.Planes[j].Name = want[j]
	}

	data, err := m.MarshalWeights()
	if err != nil {
		t.Fatalf("MarshalWeights: %v", err)
	}
	for _, id := range want {
		if !strings.Contains(string(data), id) {
			t.Fatalf("marshalled weights omit name %q:\n%s", id, data)
		}
	}

	got, err := LoadWeights(data)
	if err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	for j, id := range want {
		if got.Space.Planes[j].Name != id {
			t.Fatalf("plane %d name = %q, want %q", j, got.Space.Planes[j].Name, id)
		}
		idx, err := got.Space.PlaneIndex(id)
		if err != nil {
			t.Fatalf("PlaneIndex(%q): %v", id, err)
		}
		if idx != j {
			t.Fatalf("PlaneIndex(%q) = %d, want %d", id, idx, j)
		}
	}
}

// TestLoadWeightsWithoutNamesFallsBack pins backward compatibility: a document
// predating the names field still loads, with structural names.
func TestLoadWeightsWithoutNamesFallsBack(t *testing.T) {
	m := mustModel(t, []int{1, -1})
	data, err := m.MarshalWeights()
	if err != nil {
		t.Fatalf("MarshalWeights: %v", err)
	}
	stripped := stripKey(t, data, "names")

	got, err := LoadWeights(stripped)
	if err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	if got.Space.Planes[0].Name != "p0:q1" || got.Space.Planes[1].Name != "p1:q-1" {
		t.Fatalf("fallback names = %q, %q; want p0:q1, p1:q-1",
			got.Space.Planes[0].Name, got.Space.Planes[1].Name)
	}
}

// TestLoadWeightsRejectsNameCountMismatch guards against a silently truncated or
// padded id table.
func TestLoadWeightsRejectsNameCountMismatch(t *testing.T) {
	m := mustModel(t, []int{1, -1})
	data, err := m.MarshalWeights()
	if err != nil {
		t.Fatalf("MarshalWeights: %v", err)
	}

	var wf map[string]any
	if err := json.Unmarshal(data, &wf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	wf["names"] = []string{"only-one"}
	bad, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := LoadWeights(bad); err == nil {
		t.Fatal("LoadWeights accepted a name/plane count mismatch")
	}
}

// stripKey removes a top-level key from a marshalled weights document.
func stripKey(t *testing.T, data []byte, key string) []byte {
	t.Helper()
	var wf map[string]any
	if err := json.Unmarshal(data, &wf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	delete(wf, key)
	out, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}
