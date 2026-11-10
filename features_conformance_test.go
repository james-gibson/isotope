package isotope

// Conformance for the claims written in features/42i-*.feature.
//
// The specs are prose: they cannot fail a build. Every figure in them was first
// obtained by running this package, and the ones that are not already pinned by
// fortytwointegration_test.go / consensus_test.go are pinned here, so drift in
// the binary breaks a test instead of quietly invalidating a spec.
//
// Deliberately absent: the plane sort's weight-invariance claim is measured by
// the CLI rather than asserted here (see TestSpecClaimSortOrderIgnoresTensor for
// the part that is asserted).

import (
	"encoding/json"
	"math"
	"math/cmplx"
	"testing"

	"github.com/james-gibson/isotope/plane"
)

func TestSpecClaimAnchorDisplacement(t *testing.T) {
	charges := []int{1, -1, -1}
	a42, err := plane.ParseAnchor("42")
	if err != nil {
		t.Fatal(err)
	}
	a42i, err := plane.ParseAnchor("42i")
	if err != nil {
		t.Fatal(err)
	}
	m42, err := plane.GenerateModel(charges, a42, "")
	if err != nil {
		t.Fatal(err)
	}
	m42i, err := plane.GenerateModel(charges, a42i, "")
	if err != nil {
		t.Fatal(err)
	}

	// Hold the probe fixed: only the anchor varies. Comparing "42" against "42i"
	// varies two things and produces a misleading displacement.
	n42 := m42.Node("42")
	n42i := m42i.Node("42")

	for j := range n42.Zeta {
		d := n42i.Zeta[j] - n42.Zeta[j]
		if math.Abs(real(d)-42) > 1e-9 || math.Abs(imag(d)+42) > 1e-9 {
			t.Errorf("plane %d: displacement %v, want 42-42i", j, d)
		}
		if got := cmplx.Abs(d); math.Abs(got-42*math.Sqrt2) > 1e-9 {
			t.Errorf("plane %d: |displacement| = %.6f, want 42*sqrt(2) = %.6f",
				j, got, 42*math.Sqrt2)
		}
		if got := cmplx.Phase(d) * 180 / math.Pi; math.Abs(got+45) > 1e-9 {
			t.Errorf("plane %d: angle = %.6f deg, want -45", j, got)
		}
	}

	k42, err := m42.Key(n42)
	if err != nil {
		t.Fatal(err)
	}
	k42i, err := m42i.Key(n42i)
	if err != nil {
		t.Fatal(err)
	}
	if k42 == k42i {
		t.Error("the derived key is equal in both frames: the key would be translation-invariant")
	}
}

func TestSpecClaimParseAnchorAdmitsTwo(t *testing.T) {
	for _, s := range []string{"42", "42+0i", "(42+0i)", "42i", "0+42i", "(0+42i)"} {
		if _, err := plane.ParseAnchor(s); err != nil {
			t.Errorf("ParseAnchor(%q) rejected: %v", s, err)
		}
	}
	// The other two quarter-turn anchors are distinct concepts, not aliases.
	for _, s := range []string{"-42i", "-42", "84", "1", "0"} {
		if _, err := plane.ParseAnchor(s); err == nil {
			t.Errorf("ParseAnchor(%q) accepted; the mirror and other magnitudes are not admitted", s)
		}
	}
}

func TestSpecClaimAtRisk(t *testing.T) {
	seed := NewAgentState("probe")
	if seed.Position.AtRisk {
		t.Error("an agent at seed-42 is at risk")
	}

	seeded := NewAgentStateAtRung("probe", 2)
	if seeded.Position.AtRisk {
		t.Error("a freshly seeded rung-2 agent is at risk")
	}
	if seeded.TotalDistance != 21 {
		t.Errorf("seeded rung 2 distance = %d, want 21", seeded.TotalDistance)
	}

	seeded.RecordTestFailure("timing-correlation") // +24 -> 45, inside rung 2's 60
	if seeded.TotalDistance != 45 {
		t.Fatalf("distance = %d, want 45", seeded.TotalDistance)
	}
	if seeded.Position.Rung != 2 {
		t.Fatalf("rung = %d, want 2", seeded.Position.Rung)
	}
	if !seeded.Position.AtRisk {
		t.Error("an agent 15 inside its ceiling is not flagged at risk")
	}
}

// TestSpecClaimSortOrderIgnoresTensor pins the split the specs depend on: the
// tensor weights carry pairing, not ordering. Scaling every block leaves the
// order alone.
func TestSpecClaimSortOrderIgnoresTensor(t *testing.T) {
	charges := []int{1, -1, -1}
	names := []string{"alpha", "beta", "gamma"}
	anchor, _ := plane.ParseAnchor("42")

	build := func() *plane.Model {
		m, err := plane.GenerateModel(charges, anchor, "")
		if err != nil {
			t.Fatal(err)
		}
		for j, n := range names {
			m.Space.Planes[j].Name = n
		}
		return m
	}

	base := build()
	data, err := base.MarshalWeights()
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if uerr := json.Unmarshal(data, &doc); uerr != nil {
		t.Fatal(uerr)
	}
	blocks, _ := doc["tensor"].([]any)
	for _, raw := range blocks {
		blk, _ := raw.(map[string]any)
		ws, _ := blk["w"].([]any)
		for _, wraw := range ws {
			w, _ := wraw.(map[string]any)
			w["real"] = w["real"].(float64) * 100000
			w["imag"] = w["imag"].(float64) * 100000
		}
	}
	scaledJSON, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	scaled, err := plane.LoadWeights(scaledJSON)
	if err != nil {
		t.Fatal(err)
	}
	for j, n := range names {
		scaled.Space.Planes[j].Name = n
	}

	var selectors []int
	for _, n := range names {
		idx, err := base.Space.PlaneIndex(n)
		if err != nil {
			t.Fatal(err)
		}
		selectors = append(selectors, idx)
	}

	sortOf := func(m *plane.Model) []int {
		res, err := m.SortPlanes(selectors, m.Node("42"), 1e-9, plane.DefaultQuarantineKey())
		if err != nil {
			t.Fatal(err)
		}
		return res.Order
	}

	a, b := sortOf(base), sortOf(scaled)
	if len(a) != len(b) {
		t.Fatalf("order lengths differ: %v vs %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("scaling the tensor changed the order: %v vs %v", a, b)
		}
	}
	if len(a) == 0 {
		t.Fatal("no order produced; the fixture is not exercising the sort")
	}
}

// TestSpecClaimAgreementDoesNotVerify_Issue20 records the split the security
// spec describes: agreement groups by a signature field it never checks, and
// only the separate verification call rejects a forgery. If agreement ever
// starts verifying, this test should fail and be updated deliberately.
func TestSpecClaimAgreementDoesNotVerify_Issue20(t *testing.T) {
	forged := make([]ConsensusReport, 3)
	for i := range forged {
		forged[i] = ConsensusReport{
			AlarmID:   AlarmID("alarm-1"),
			Isotope:   Isotope{Family: "f", Version: 1, Signature: "forged-signature-000000000000000000000000000000000000000000000000000"},
			Signature: "forged",
		}
	}
	keys := map[AlarmID][]byte{AlarmID("alarm-1"): []byte("seed-42")}

	status := VerifyAgreement(forged, 2)
	if !status.ConsensusFormed {
		t.Fatal("agreement did not form over a forged signature; behavior changed (see isotope#20)")
	}
	if err := VerifyConsensusSignatures(forged, keys); err == nil {
		t.Fatal("verification accepted a forged signature")
	}
}

// TestSpecClaimShortSignatureRejected is the contract from the security spec:
// verification must fail rather than panic. It panics today (isotope#19), so the
// test is skipped until that is fixed; remove the Skip with the fix.
func TestSpecClaimShortSignatureRejected(t *testing.T) {
	t.Skip("isotope#19: VerifyConsensusSignatures panics on signatures shorter than 16 chars")

	reports := []ConsensusReport{{AlarmID: "alarm-1", Isotope: Isotope{Signature: "abc"}}}
	keys := map[AlarmID][]byte{"alarm-1": []byte("seed-42")}
	if err := VerifyConsensusSignatures(reports, keys); err == nil {
		t.Error("a malformed signature was accepted")
	}
}

// TestSpecClaimRotationIsUniversal pins the claim that the frame rotation is a
// property of the anchor pair alone: every agent, charge set, plane count and
// probe sees the same 42-42i displacement.
func TestSpecClaimRotationIsUniversal(t *testing.T) {
	cases := []struct {
		charges string
		probes  []string
	}{
		{"1,-1,-1", []string{"42", "42i", "alpha", "some arbitrary probe"}},
		{"1,1", []string{"42", "beta"}},
		{"1,-1", []string{"42", "gamma"}},
		{"1,-1,1,-1", []string{"42", "delta"}},
		{"1,-1,-1,1,1", []string{"42", "epsilon"}},
		{"1", []string{"42", "zeta"}},
		{"0", []string{"42"}},
		{"42", []string{"42"}},
		{"3x-1", []string{"42", "eta"}},
	}

	a42, _ := plane.ParseAnchor("42")
	a42i, _ := plane.ParseAnchor("42i")
	planes := 0

	for _, c := range cases {
		charges, err := plane.ParseCharges(c.charges)
		if err != nil {
			t.Fatalf("charges %q: %v", c.charges, err)
		}
		for _, probe := range c.probes {
			m42, err := plane.GenerateModel(charges, a42, "")
			if err != nil {
				t.Fatalf("%s/%s: %v", c.charges, probe, err)
			}
			m42i, err := plane.GenerateModel(charges, a42i, "")
			if err != nil {
				t.Fatalf("%s/%s: %v", c.charges, probe, err)
			}
			n42, n42i := m42.Node(probe), m42i.Node(probe)
			if len(n42.Zeta) != len(n42i.Zeta) {
				t.Fatalf("%s/%s: plane counts differ", c.charges, probe)
			}
			for j := range n42.Zeta {
				planes++
				d := n42i.Zeta[j] - n42.Zeta[j]
				if math.Abs(real(d)-42) > 1e-9 || math.Abs(imag(d)+42) > 1e-9 {
					t.Errorf("charges %q probe %q plane %d: displacement %v, want 42-42i",
						c.charges, probe, j, d)
				}
			}
		}
	}
	if planes < 40 {
		t.Fatalf("only %d planes exercised; the matrix is not covering the claim", planes)
	}
}
