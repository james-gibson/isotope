package isotope

import (
	"reflect"
	"testing"
)

const workspaceSource = `Feature: Workspace VFS — minimal contract

  Rule: §1 the on-disk conventions
    Scenario: the layout is the documented one
      Then the tree contains "/data/task.json"

  Rule: §2 the MCP mapping
    Scenario: resources are files
      Given the resource file "/resources/workspace/config.json"
      Then "resources/list" includes "resource://workspace/config"
      And the set from "find /resources" equals the set from "resources/list"

    Scenario: tools are files
      Given an executable endpoint "/tools/echo@"
      Then "tools/list" includes "echo"

    Scenario: prompts are templates
      Given the template "/prompts/summarize.tmpl"
      When "prompts/get" is called for "summarize"
      Then the rendered template is returned
`

func TestDefineReadsWorkspaceDeclarations(t *testing.T) {
	spec, err := Define(workspaceSource, WorkspaceStepDefinitions()...)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	if spec.Feature != "Workspace VFS — minimal contract" {
		t.Errorf("feature = %q", spec.Feature)
	}
	if got, want := spec.ByKind("resource"), []string{"resource://workspace/config"}; !reflect.DeepEqual(got, want) {
		t.Errorf("resources = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("tool"), []string{"echo"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tools = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("prompt"), []string{"summarize"}; !reflect.DeepEqual(got, want) {
		t.Errorf("prompts = %v, want %v", got, want)
	}
	// the spec's own file layout: what the steps name on disk
	if got, want := spec.ByKind("resource_file"), []string{"/resources/workspace/config.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("resource files = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("tool_file"), []string{"/tools/echo@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tool files = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("prompt_file"), []string{"/prompts/summarize.tmpl"}; !reflect.DeepEqual(got, want) {
		t.Errorf("prompt files = %v, want %v", got, want)
	}
	if got, want := spec.Kinds(), []string{"prompt", "prompt_file", "resource", "resource_file", "tool", "tool_file"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}
}

// A step may declare more than one thing: the file it names and the name the
// surface presents are both consequences of one line of the spec.
func TestOneStepMayDeclareTwoThings(t *testing.T) {
	src := "Feature: f\n  Scenario: s\n" +
		"    Given the resource file \"/resources/a/b.json\"\n"
	spec, err := Define(src, WorkspaceStepDefinitions()...)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := spec.ByKind("resource_file"), []string{"/resources/a/b.json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("resource files = %v, want %v", got, want)
	}
	if got, want := spec.ByKind("resource"), []string{"resource://a/b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("resources = %v, want %v", got, want)
	}
}

func TestDefineIgnoresStepsThatDeclareNothing(t *testing.T) {
	src := "Feature: f\n  Scenario: s\n" +
		"    Given an unremarkable step\n" +
		"    And the resource file \"/resources/a.json\"\n"
	spec, err := Define(src, WorkspaceStepDefinitions()...)
	if err != nil {
		t.Fatal(err)
	}
	// "an unremarkable step" declares nothing; the resource step declares the
	// file *and* the URI.
	if len(spec.Declarations) != 2 {
		t.Fatalf("declarations = %+v, want exactly 2", spec.Declarations)
	}
	if got := spec.ByKind("resource"); len(got) != 1 {
		t.Errorf("resources = %v, want one", got)
	}
	if got := spec.ByKind("resource_file"); len(got) != 1 {
		t.Errorf("resource files = %v, want one", got)
	}
}

func TestStepPatternCapturesAndDeduplicates(t *testing.T) {
	defs := []StepDefinition{{
		Pattern: `the widget "{name}" is "{colour}"`,
		Kind:    "widget",
		Value:   func(v map[string]string) string { return v["name"] + ":" + v["colour"] },
	}}
	src := "Feature: f\n  Scenario: s\n" +
		"    Given the widget \"a\" is \"red\"\n" +
		"    And the widget \"b\" is \"blue\"\n" +
		"    And the widget \"a\" is \"red\"\n"
	spec, err := Define(src, defs...)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := spec.ByKind("widget"), []string{"a:red", "b:blue"}; !reflect.DeepEqual(got, want) {
		t.Errorf("widgets = %v, want %v", got, want)
	}
}

func TestWorkspacePathMappings(t *testing.T) {
	cases := []struct {
		in, want string
		fn       func(string) string
	}{
		{"/resources/workspace/config.json", "resource://workspace/config", ResourceURI},
		{"/resources/a/b/c.json", "resource://a/b/c", ResourceURI},
		{"/tools/echo@", "echo", ToolName},
		{"/tools/late@", "late", ToolName},
		{"/prompts/summarize.tmpl", "summarize", PromptName},
	}
	for _, tc := range cases {
		if got := tc.fn(tc.in); got != tc.want {
			t.Errorf("%q -> %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Two readings of one source must agree: the canonical form and the meaning come
// from the same parse.
func TestCanonicalizeAndDefineAgreeOnTheSource(t *testing.T) {
	one, err := CanonicalizeGherkinToIsotope(workspaceSource, []byte("k"), "fam", 1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := CanonicalizeGherkinToIsotope(workspaceSource, []byte("k"), "fam", 1)
	if err != nil {
		t.Fatal(err)
	}
	if one.Signature != two.Signature {
		t.Errorf("the same source signed twice differs: %s vs %s", one.Signature, two.Signature)
	}
	spec, err := Define(workspaceSource, WorkspaceStepDefinitions()...)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Declarations) == 0 {
		t.Error("Define found no declarations in a source that declares three")
	}
}
