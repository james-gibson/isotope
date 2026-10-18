package isotope

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

// Step definitions: read a spec for *meaning*, not only for identity.
//
// CanonicalizeGherkinToIsotope answers "is this the same test?"; a step
// definition answers "what does this test ask for?". They are two readings of
// one Gherkin source, and they belong side by side — a caller that canonicalizes
// a spec to sign it and separately hand-rolls a parser to interpret it has two
// readings that can drift apart.
//
// A Declaration is one thing a step declares; a StepDefinition binds a step
// pattern to the declaration it makes; Define applies a set of definitions to a
// source. Nothing here executes a step — a step *declares*, and the caller
// decides what a declaration is worth.

// Declaration is one thing a Gherkin step declares.
type Declaration struct {
	Kind  string // what the step declares: "resource", "tool", "prompt", …
	Value string // the declared value, in the caller's vocabulary
}

// StepDefinition binds a step pattern to the declaration it makes. Patterns use
// {name} placeholders, e.g. `the resource file "{path}"`.
type StepDefinition struct {
	Pattern string
	Kind    string
	Value   func(vars map[string]string) string
}

// Spec is what a Gherkin source declares.
type Spec struct {
	Feature      string
	Declarations []Declaration
}

// ByKind returns the values declared for a kind, sorted and de-duplicated.
func (s *Spec) ByKind(kind string) []string {
	if s == nil {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, d := range s.Declarations {
		if d.Kind != kind || d.Value == "" || seen[d.Value] {
			continue
		}
		seen[d.Value] = true
		out = append(out, d.Value)
	}
	sort.Strings(out)
	return out
}

// Kinds returns the kinds declared, sorted.
func (s *Spec) Kinds() []string {
	if s == nil {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	for _, d := range s.Declarations {
		if d.Kind == "" || seen[d.Kind] {
			continue
		}
		seen[d.Kind] = true
		out = append(out, d.Kind)
	}
	sort.Strings(out)
	return out
}

var placeholderRe = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

type compiledDefinition struct {
	def   StepDefinition
	re    *regexp.Regexp
	names []string
}

func compileDefinition(d StepDefinition) compiledDefinition {
	names := []string{}
	for _, m := range placeholderRe.FindAllStringSubmatch(d.Pattern, -1) {
		names = append(names, m[1])
	}
	parts := placeholderRe.Split(d.Pattern, -1)
	var b strings.Builder
	b.WriteString("^")
	for i, p := range parts {
		if i > 0 {
			b.WriteString(`([^"]*)`)
		}
		b.WriteString(regexp.QuoteMeta(p))
	}
	b.WriteString("$")
	return compiledDefinition{def: d, re: regexp.MustCompile(b.String()), names: names}
}

func (c compiledDefinition) match(text string) (map[string]string, bool) {
	m := c.re.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return nil, false
	}
	vars := make(map[string]string, len(c.names))
	for i, n := range c.names {
		vars[n] = m[i+1]
	}
	return vars, true
}

// Define parses gherkin and applies defs to every step, in source order. A step
// that matches no definition declares nothing; a definition that yields "" is
// skipped. Every *matching* definition is applied — one step may declare more
// than one thing (the file it names and the name the surface presents).
func Define(gherkin string, defs ...StepDefinition) (*Spec, error) {
	ast, err := ParseGherkin(gherkin)
	if err != nil {
		return nil, err
	}
	compiled := make([]compiledDefinition, 0, len(defs))
	for _, d := range defs {
		compiled = append(compiled, compileDefinition(d))
	}

	spec := &Spec{}
	for _, f := range ast.Features {
		if spec.Feature == "" {
			spec.Feature = f.Name
		}
		for _, sc := range f.Scenarios {
			for _, st := range sc.Steps {
				for _, cd := range compiled {
					vars, ok := cd.match(st.Text)
					if !ok {
						continue
					}
					if v := cd.def.Value(vars); v != "" {
						spec.Declarations = append(spec.Declarations, Declaration{Kind: cd.def.Kind, Value: v})
					}
				}
			}
		}
	}
	return spec, nil
}

// ---- the workspace-VFS vocabulary ----
//
// The path conventions live here, once, so a spec read for meaning and an
// observation read from a filesystem cannot drift apart.

const (
	// WorkspaceResources is where a session's MCP resources are declared.
	WorkspaceResources = "/resources"
	// WorkspaceTools is where a session's MCP tools are declared.
	WorkspaceTools = "/tools"
	// WorkspacePrompts is where a session's MCP prompts are declared.
	WorkspacePrompts = "/prompts"
)

// ResourceURI maps a declared resource file to the URI it serves:
// "/resources/workspace/config.json" -> "resource://workspace/config".
func ResourceURI(p string) string {
	rel := strings.TrimPrefix(p, WorkspaceResources+"/")
	rel = strings.TrimSuffix(rel, path.Ext(rel))
	return "resource://" + rel
}

// ToolName maps a declared tool endpoint to its name: "/tools/echo@" -> "echo".
func ToolName(p string) string { return path.Base(strings.TrimSuffix(p, "@")) }

// PromptName maps a declared template to its name:
// "/prompts/summarize.tmpl" -> "summarize".
func PromptName(p string) string { return path.Base(strings.TrimSuffix(p, ".tmpl")) }

// WorkspaceStepDefinitions returns the definitions for the workspace-VFS gate
// vocabulary (lab-safety sheets/workspaces/minimal.feature §2).
//
// Each surface declares two things, and both matter: the *file* the step names
// (what must exist on disk — the on-disk conventions the spec fixes) and the
// *name* the surface presents (what resources/list, tools/list and prompts/list
// answer with). Reading only the name loses what the spec said; reading only the
// file loses what the surface says. One step declares both.
func WorkspaceStepDefinitions() []StepDefinition {
	return []StepDefinition{
		{
			Pattern: `the resource file "{path}"`,
			Kind:    "resource_file",
			Value:   func(v map[string]string) string { return v["path"] },
		},
		{
			Pattern: `the resource file "{path}"`,
			Kind:    "resource",
			Value:   func(v map[string]string) string { return ResourceURI(v["path"]) },
		},
		{
			Pattern: `an executable endpoint "{path}"`,
			Kind:    "tool_file",
			Value:   func(v map[string]string) string { return v["path"] },
		},
		{
			Pattern: `an executable endpoint "{path}"`,
			Kind:    "tool",
			Value:   func(v map[string]string) string { return ToolName(v["path"]) },
		},
		{
			Pattern: `the template "{path}"`,
			Kind:    "prompt_file",
			Value:   func(v map[string]string) string { return v["path"] },
		},
		{
			Pattern: `the template "{path}"`,
			Kind:    "prompt",
			Value:   func(v map[string]string) string { return PromptName(v["path"]) },
		},
	}
}
