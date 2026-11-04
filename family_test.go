package isotope

import "testing"

func TestFamilyType(t *testing.T) {
	if got := FamilyType("# family: deployment\nFeature: The note service\n"); got != "deployment" {
		t.Errorf("FamilyType = %q, want %q", got, "deployment")
	}
	if got := FamilyType("Feature: no family tag\n"); got != "" {
		t.Errorf("FamilyType = %q, want empty", got)
	}
}

func TestFamilyTagDoesNotChangeIsotope(t *testing.T) {
	with := "# family: deployment\nFeature: The note service\n  Scenario: one\n    Given the route \"/authn\"\n"
	without := "Feature: The note service\n  Scenario: one\n    Given the route \"/authn\"\n"
	a, err := CanonicalizeGherkinToIsotope(with, []byte("seed-42"), "nest", 1)
	if err != nil {
		t.Fatalf("with: %v", err)
	}
	b, err := CanonicalizeGherkinToIsotope(without, []byte("seed-42"), "nest", 1)
	if err != nil {
		t.Fatalf("without: %v", err)
	}
	if a.Raw != b.Raw {
		t.Errorf("family tag changed the isotope: %q != %q", a.Raw, b.Raw)
	}
}
