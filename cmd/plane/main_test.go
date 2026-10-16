package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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
