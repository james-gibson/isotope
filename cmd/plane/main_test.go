package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/james-gibson/isotope/plane"
)

func runCapture(t *testing.T, args []string, stdin string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(args, strings.NewReader(stdin), &out); err != nil {
		t.Fatalf("run(%v): %v", args, err)
	}
	return out.String()
}

func TestCLIByteDeterminism(t *testing.T) {
	cmds := [][]string{
		{"node", "--anchor", "42", "42"},
		{"edge", "--anchor", "42", "0", "2"},
		{"triple", "--anchor", "42", "0", "1", "2"},
	}
	for _, c := range cmds {
		first := runCapture(t, c, "")
		second := runCapture(t, c, "")
		if first != second {
			t.Fatalf("%v not byte-deterministic", c)
		}
		if !strings.Contains(first, "{") {
			t.Fatalf("%v did not produce JSON: %q", c, first)
		}
	}
	sortArgs := []string{"sort", "--anchor", "42"}
	first := runCapture(t, sortArgs, "0\n1\n2\n")
	second := runCapture(t, sortArgs, "0\n1\n2\n")
	if first != second {
		t.Fatal("sort not byte-deterministic")
	}
}

func TestCLISortRequiresAnchor(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"sort"}, strings.NewReader("0\n"), &out)
	if err == nil {
		t.Fatal("expected sort without --anchor to fail")
	}
}

func TestCLIGenAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weights.json")
	if err := run([]string{"gen", "--charges", "1,1,-1", "--anchor", "42i", "--out", path}, nil, &bytes.Buffer{}); err != nil {
		t.Fatalf("gen: %v", err)
	}
	first := runCapture(t, []string{"triple", "--weights", path, "--anchor", "42i", "0", "1", "2"}, "")
	second := runCapture(t, []string{"triple", "--weights", path, "--anchor", "42i", "0", "1", "2"}, "")
	if first != second {
		t.Fatal("reloaded weights not deterministic")
	}
}

func TestCLIDisconnected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weights.json")
	err := run([]string{"gen", "--charges", "0,1,1,2", "--out", path}, nil, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected gen to reject a disconnected charge graph")
	}
}

func TestCLISearchQueriesPlaneVector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/kb/points/search" {
			t.Fatalf("path = %q, want named-vector search path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"id":"doc-1","score":0.9}]}`))
	}))
	defer server.Close()

	out := runCapture(t, []string{
		"search", "--endpoint", server.URL, "--anchor", "42i", "--query", "hello",
	}, "")
	if !strings.Contains(out, `"vector": "plane"`) || !strings.Contains(out, `"doc-1"`) {
		t.Fatalf("search output = %s", out)
	}
}

// TestParseCharges pins the values a charge list produces. A parser that
// silently collapses every charge to zero still builds a model, so this is the
// only place that mistake is visible.
func TestParseCharges(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []int
	}{
		{"bare", "1,-1,1", []int{1, -1, 1}},
		{"negative first", "-1,1", []int{-1, 1}},
		{"whitespace", " 1 , -1 , 2 ", []int{1, -1, 2}},
		{"repeat", "3x-1", []int{-1, -1, -1}},
		{"repeat mixed", "2x1,-1", []int{1, 1, -1}},
		{"repeat positive", "2x2", []int{2, 2}},
		{"single", "0", []int{0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := plane.ParseCharges(tc.in)
			if err != nil {
				t.Fatalf("ParseCharges(%q): %v", tc.in, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("ParseCharges(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseChargesRejects covers the inputs that must not silently parse.
func TestParseChargesRejects(t *testing.T) {
	for _, in := range []string{"", "  ", "1,,2", "1,abc", "-1x", "abcx2", "2x", "1x2x3", "-1x-1"} {
		if _, err := plane.ParseCharges(in); err == nil {
			t.Fatalf("ParseCharges(%q) unexpectedly succeeded", in)
		}
	}
}
