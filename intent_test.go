package isotope

import (
	"errors"
	"strings"
	"testing"
)

// One golden intent per promise, with the three wordings the issue measured.
func loginIntents(t *testing.T) *IntentIndex {
	t.Helper()
	ix, err := NewIntentIndex(
		Intent{Key: "login.name", Anchor: "login"},
		Intent{
			Key:    "login.user-on-page",
			Anchor: "the user is on the login page",
			Locales: map[string]string{
				"es": "que el usuario está en la página de inicio de sesión",
				"fr": "que l'utilisateur est sur la page de connexion",
			},
		},
		Intent{
			Key:    "login.sees-dashboard",
			Anchor: "they see the dashboard",
			Locales: map[string]string{
				"es": "ven el panel",
				"fr": "ils voient le tableau de bord",
			},
		},
	)
	if err != nil {
		t.Fatalf("NewIntentIndex: %v", err)
	}
	return ix
}

var localizedLoginSpecs = map[string]string{
	"english": `
Feature: login
  Scenario: login
    Given the user is on the login page
    Then they see the dashboard
`,
	"spanish": `
Feature: login
  Scenario: login
    Dado que el usuario está en la página de inicio de sesión
    Entonces ven el panel
`,
	"french": `
Feature: login
  Scenario: login
    Étant donné que l'utilisateur est sur la page de connexion
    Alors ils voient le tableau de bord
`,
}

// TestTranslationIsTheSameIntentAndANewName: the two layers answer different
// questions about the same three specs. This is the fix for #21 — the name
// stays language-bound, the intent compares.
func TestTranslationIsTheSameIntentAndANewName(t *testing.T) {
	ix := loginIntents(t)
	key := []byte("test-fire-marshal-key")

	var intentIsotope string
	names := map[string]bool{}
	for _, lang := range []string{"english", "spanish", "french"} {
		gherkin := localizedLoginSpecs[lang]

		ast, err := ParseGherkin(gherkin)
		if err != nil {
			t.Fatalf("%s: ParseGherkin: %v", lang, err)
		}
		c := ix.CanonicalizeIntents(ast)
		if len(c.Unresolved) != 0 {
			t.Errorf("%s: unresolved wordings: %v", lang, c.Unresolved)
		}

		name, err := CanonicalizeGherkinToIsotope(gherkin, key, "iso-login", 1)
		if err != nil {
			t.Fatalf("%s: CanonicalizeGherkinToIsotope: %v", lang, err)
		}
		if names[name.Signature] {
			t.Errorf("%s: name isotope collides with another language", lang)
		}
		names[name.Signature] = true

		intent := c.Isotope(key, "iso-login-intent", 1)
		if intentIsotope == "" {
			intentIsotope = intent.Signature
		} else if intent.Signature != intentIsotope {
			t.Errorf("%s: intent isotope %s, expected %s (all languages are one intent)",
				lang, intent.Signature[:16], intentIsotope[:16])
		}
	}
}

// TestEmptyIndexIsTheNameStream: without declarations the intent layer must not
// invent anything. It returns the name stream unchanged, with everything
// reported unresolved.
func TestEmptyIndexIsTheNameStream(t *testing.T) {
	ix, err := NewIntentIndex()
	if err != nil {
		t.Fatal(err)
	}

	gherkin := localizedLoginSpecs["english"]
	ast, err := ParseGherkin(gherkin)
	if err != nil {
		t.Fatal(err)
	}

	want := TokensToString(ast.Canonicalize())
	got := ix.CanonicalizeIntents(ast)
	if got.String() != want {
		t.Errorf("intent stream with no intents differs from the name stream:\n got %q\nwant %q",
			got.String(), want)
	}
	if len(got.Unresolved) != 4 { // feature, scenario, two steps
		t.Errorf("expected every wording unresolved, got %v", got.Unresolved)
	}
}

// TestUndeclaredProseIsReportedNotGuessed: a step the index does not cover is
// not matched by similarity. It stays as written and is listed.
func TestUndeclaredProseIsReportedNotGuessed(t *testing.T) {
	ix, err := NewIntentIndex(
		Intent{Key: "login.user-on-page", Anchor: "the user is on the login page"},
	)
	if err != nil {
		t.Fatal(err)
	}

	english := `
Feature: login
  Scenario: login
    Given the user is on the login page
    Then they see the dashboard
`
	other := `
Feature: login
  Scenario: login
    Given the user is on the login page
    Then they see a different panel
`
	a := canonIntents(t, ix, english)
	b := canonIntents(t, ix, other)

	if !listsWording(a.Unresolved, "login") || !listsWording(a.Unresolved, "they see the dashboard") {
		t.Errorf("unresolved wordings not reported: %v", a.Unresolved)
	}
	if CompareIntents(a, b).SameIntent {
		t.Error("undeclared, differing prose compared as the same intent")
	}
}

// TestAmbiguousWordingIsRejected: one wording cannot promise two things.
func TestAmbiguousWordingIsRejected(t *testing.T) {
	_, err := NewIntentIndex(
		Intent{Key: "a", Anchor: "the user is logged in"},
		Intent{Key: "b", Anchor: "The user is logged in!"},
	)
	if !errors.Is(err, ErrAmbiguousWording) {
		t.Fatalf("expected ErrAmbiguousWording, got %v", err)
	}
}

func TestIntentIndexValidation(t *testing.T) {
	tests := []struct {
		name string
		in   Intent
	}{
		{"empty key", Intent{Anchor: "x"}},
		{"colon in key", Intent{Key: "a:b", Anchor: "x"}},
		{"newline in key", Intent{Key: "a\nb", Anchor: "x"}},
		{"empty anchor", Intent{Key: "a"}},
		{"empty locale wording", Intent{Key: "a", Anchor: "x", Locales: map[string]string{"es": "  "}}},
		{"reserved locale", Intent{Key: "a", Anchor: "x", Locales: map[string]string{"anchor": "y"}}},
	}
	for _, tt := range tests {
		if _, err := NewIntentIndex(tt.in); err == nil {
			t.Errorf("%s: expected an error", tt.name)
		}
	}

	if _, err := NewIntentIndex(
		Intent{Key: "a", Anchor: "one"},
		Intent{Key: "a", Anchor: "two"},
	); err == nil {
		t.Error("duplicate key: expected an error")
	}
}

// TestResolveUsesTheCanonNormalization: resolution agrees with the hash, so a
// rewording the canon already treats as identical also resolves.
func TestResolveUsesTheCanonNormalization(t *testing.T) {
	ix := loginIntents(t)
	in, ok := ix.Resolve("The user is on the login page!")
	if !ok {
		t.Fatal("punctuated and capitalized wording did not resolve")
	}
	if in.Key != "login.user-on-page" {
		t.Errorf("resolved to %q", in.Key)
	}
	if _, ok := ix.Resolve("the user opens the login page"); ok {
		t.Error("an undeclared rewording resolved")
	}
}

// TestCoverageReportsAPartialRotation: a locale covering only some intents
// rotates only those.
func TestCoverageReportsAPartialRotation(t *testing.T) {
	ix, err := NewIntentIndex(
		Intent{Key: "a", Anchor: "the user is on the login page", Locales: map[string]string{"es": "el usuario está en la página de inicio de sesión"}},
		Intent{Key: "b", Anchor: "they see the dashboard"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if covered, total := ix.Coverage("es"); covered != 1 || total != 2 {
		t.Errorf("es coverage = %d/%d, expected 1/2", covered, total)
	}
	if covered, total := ix.Coverage("de"); covered != 0 || total != 2 {
		t.Errorf("de coverage = %d/%d, expected 0/2", covered, total)
	}
	if got := ix.Locales(); len(got) != 1 || got[0] != "es" {
		t.Errorf("locales = %v", got)
	}
}

func canonIntents(t *testing.T, ix *IntentIndex, gherkin string) IntentCanonical {
	t.Helper()
	ast, err := ParseGherkin(gherkin)
	if err != nil {
		t.Fatalf("ParseGherkin: %v", err)
	}
	return ix.CanonicalizeIntents(ast)
}

func listsWording(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.EqualFold(s, needle) {
			return true
		}
	}
	return false
}
