package isotope

// What the name is language-free about, and what it is not (#21).
//
// The README and USAGE once claimed that one test written in several languages
// produces one isotope. Measured, only the *keyword* normalizes: the step prose
// is hashed as written, so a translation is a new name. That is correct for a
// route — the name identifies wording — and it is why comparison moved to the
// declared-intent layer in intent.go. These tests pin the name layer so a change
// to the canon surfaces as a failing test instead of leaving the documentation
// quietly wrong.

import "testing"

// One feature name, one scenario name, only the step prose localized.
var localizedLogin = map[string]string{
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

// TestLocalizedKeywordsShareAKeywordLayer: the keyword normalizes across
// languages. This is the part the documentation got right.
func TestLocalizedKeywordsShareAKeywordLayer(t *testing.T) {
	for lang, gherkin := range localizedLogin {
		ast, err := ParseGherkin(gherkin)
		if err != nil {
			t.Fatalf("%s: ParseGherkin: %v", lang, err)
		}
		steps := ast.Features[0].Scenarios[0].Steps
		if len(steps) != 2 {
			t.Fatalf("%s: expected 2 steps, got %d", lang, len(steps))
		}
		if steps[0].Keyword != KeywordGiven {
			t.Errorf("%s: expected first step GIVEN, got %v", lang, steps[0].Keyword)
		}
		if steps[1].Keyword != KeywordThen {
			t.Errorf("%s: expected second step THEN, got %v", lang, steps[1].Keyword)
		}
	}
}

// TestLocalizedProseIsANewIsotope: the step prose is hashed as written. The
// hashes below are the ones measured in the issue, and they must not be equal.
func TestLocalizedProseIsANewIsotope(t *testing.T) {
	key := []byte("test-fire-marshal-key")
	wantGiven := map[string]string{
		"english": "120937abc278cbb8",
		"spanish": "b3eecf9b587d8ef0",
		"french":  "81a73d899e2766e7",
	}

	seen := map[string]string{}
	for lang, gherkin := range localizedLogin {
		iso, err := CanonicalizeGherkinToIsotope(gherkin, key, "iso-login", 1)
		if err != nil {
			t.Fatalf("%s: CanonicalizeGherkinToIsotope: %v", lang, err)
		}

		ast, _ := ParseGherkin(gherkin)
		got := ast.Canonicalize()[2].Value // FEATURE, SCENARIO, then the first GIVEN
		if got != wantGiven[lang] {
			t.Errorf("%s: GIVEN hash = %s, expected %s", lang, got, wantGiven[lang])
		}

		if other, dup := seen[iso.Signature]; dup {
			t.Errorf("%s and %s produced the same isotope", other, lang)
		}
		seen[iso.Signature] = lang
	}
}
