package plane

import (
	"bytes"
	"math"
	"math/cmplx"
	"testing"
)

const tol12 = 1e-12

func mustModel(t *testing.T, charges []int) *Model {
	t.Helper()
	m, err := GenerateModel(charges, complex(42, 0), "")
	if err != nil {
		t.Fatalf("GenerateModel(%v): %v", charges, err)
	}
	return m
}

func node(zs ...complex128) Node { return Node{Zeta: zs} }

// TestEquivariance asserts s(R_theta a, R_theta b) == s(a, b) to 1e-12 under the
// charge-weighted U(1) action.
func TestEquivariance(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	a := node(complex(2, 1), complex(-3, 0.5), complex(1, -1))
	b := node(complex(0.5, -2), complex(1, 1), complex(-4, 2))
	s0r, s0i := m.Tensor.Score(a, b)
	for _, theta := range []float64{0.1, 1.0, -2.7, 42.0} {
		ra := m.Space.Rotate(a, theta)
		rb := m.Space.Rotate(b, theta)
		sr, si := m.Tensor.Score(ra, rb)
		if math.Abs(sr-s0r) > tol12 || math.Abs(si-s0i) > tol12 {
			t.Fatalf("theta=%v: score changed: (%v,%v) vs (%v,%v)", theta, sr, si, s0r, s0i)
		}
	}
}

// TestDistanceInvariance asserts d(R a, R b) == d(a, b) for same-charge planes.
func TestDistanceInvariance(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	z := []complex128{complex(2, 1), complex(-3, 0.5), complex(1, -1)}
	m.Tree.CacheTransports(z)
	// Planes 0 and 1 share charge +1: the equivariant channel.
	d0, err := m.Tree.Distance(z, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, theta := range []float64{0.3, -1.4, 5.0} {
		rz := m.Space.Rotate(node(z...), theta).Zeta
		rt := m.Tree
		rt.CacheTransports(rz)
		d1, err := rt.Distance(rz, 0, 1)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(d1-d0) > tol12 {
			t.Fatalf("theta=%v: distance changed: %v vs %v", theta, d1, d0)
		}
	}
}

// TestPhaseCovariance asserts the three-point phase term shifts by the expected
// amount (q_A-q_B)*theta under R_theta, and is unchanged when q_A == q_B.
func TestPhaseCovariance(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	z := []complex128{complex(2, 1), complex(-3, 0.5), complex(1, -1)}
	m.Tree.CacheTransports(z)
	k0, err := m.Tree.Triple(z, 0, 1, 2, 1e-9)
	if err != nil {
		t.Fatal(err)
	}
	// Same-charge A=0,B=1: unchanged by a global rotation of all three.
	for _, theta := range []float64{0.2, -1.1, 3.3} {
		rz := m.Space.Rotate(node(z...), theta).Zeta
		rt := m.Tree
		rt.CacheTransports(rz)
		k1, kerr := rt.Triple(rz, 0, 1, 2, 1e-9)
		if kerr != nil {
			t.Fatal(kerr)
		}
		if math.Abs(wrapPi(k1.Phase-k0.Phase)) > 1e-9 {
			t.Fatalf("theta=%v: same-charge phase changed: %v vs %v", theta, k1.Phase, k0.Phase)
		}
	}
	// Mixed charges A=0 (q=1), B=2 (q=-1): shifts by (q_A-q_B)*theta = 2*theta.
	base, err := m.Tree.Triple(z, 0, 2, 1, 1e-9)
	if err != nil {
		t.Fatal(err)
	}
	theta := 0.4
	rz := m.Space.Rotate(node(z...), theta).Zeta
	rt := m.Tree
	rt.CacheTransports(rz)
	rot, err := rt.Triple(rz, 0, 2, 1, 1e-9)
	if err != nil {
		t.Fatal(err)
	}
	want := wrapPi(base.Phase + 2*theta)
	if math.Abs(wrapPi(rot.Phase-want)) > 1e-9 {
		t.Fatalf("mixed-charge phase shift: got %v want %v", rot.Phase, want)
	}
}

// TestHolonomy asserts a consistent triple has exactly zero holonomy and a
// drifted triple has non-zero holonomy.
func TestHolonomy(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	consistent := []complex128{1, 1, 1}
	m.Tree.CacheTransports(consistent)
	phi, ok, err := m.Tree.CycleHolonomy(consistent, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected (1,2) to be a non-tree edge")
	}
	if phi != 0 {
		t.Fatalf("consistent triple holonomy = %v, want exactly 0", phi)
	}

	drifted := []complex128{cmplx.Rect(1, 0.3), 1, 1}
	m.Tree.CacheTransports(drifted)
	phi2, ok, err := m.Tree.CycleHolonomy(drifted, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected (1,2) to be a non-tree edge")
	}
	if math.Abs(phi2) < 1e-6 {
		t.Fatalf("drifted triple holonomy = %v, want non-zero", phi2)
	}
}

// TestCycleCount asserts the number of fundamental cycles is E - (N-1).
func TestCycleCount(t *testing.T) {
	m := mustModel(t, []int{1, 1, 1, -1})
	admissible := len(admissibleEdges(m.Space, m.Tensor, nil))
	want := admissible - (len(m.Space.Planes) - 1)
	if got := m.Tree.FundamentalCycleCount(); got != want {
		t.Fatalf("fundamental cycles = %d, want %d", got, want)
	}
}

// TestTransportConsistency asserts transporting along the tree matches the
// accumulated edge phases to 1e-12.
func TestTransportConsistency(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	z := []complex128{complex(2, 1), complex(-3, 0.5), complex(1, -1)}
	m.Tree.CacheTransports(z)
	got, err := m.Tree.Transport(z, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Manual path product: child 1 -> parent 0.
	want := z[1] * cmplxExp(0, edgeAngle(m.Space, z, 1, m.Tree.Parent[1]))
	if cmplx.Abs(got-want) > tol12 {
		t.Fatalf("transport mismatch: %v vs %v", got, want)
	}
	// Manual accumulated Gamma for plane 2 (child of root).
	wantGamma := edgeAngle(m.Space, z, 2, m.Tree.Parent[2])
	if math.Abs(m.Tree.Gamma[2]-wantGamma) > tol12 {
		t.Fatalf("gamma mismatch: %v vs %v", m.Tree.Gamma[2], wantGamma)
	}
}

// TestDisconnectedPlanes asserts a disconnected charge graph errors, never a
// default phase of zero.
func TestDisconnectedPlanes(t *testing.T) {
	space, err := NewSpace([]int{0, 1, 1, 2}, complex(42, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildTree(space, BuildTensor(space), nil); err == nil {
		t.Fatal("expected ErrDisconnected for charges 0,1,1,2")
	}
}

// TestQuarantine asserts the dropped edge is absent after rebuild, the re-sort
// succeeds, and the event is signed.
func TestQuarantine(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	z := []complex128{cmplx.Rect(1, 0.3), 1, 1}
	key := DefaultQuarantineKey()
	out, err := Quarantine(m.Space, m.Tensor, z, 1e-9, key)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Quarantined || len(out.Events) == 0 {
		t.Fatal("expected at least one quarantine event")
	}
	for _, e := range out.Events {
		if !VerifyEvent(key, e) {
			t.Fatalf("event not signed correctly: %+v", e)
		}
		if e.Action != "quarantine" {
			t.Fatalf("unexpected action %q", e.Action)
		}
		if len(e.Planes) != 3 {
			t.Fatalf("expected all three suspects, got %v", e.Planes)
		}
	}
	// The rebuilt tree must not carry the dropped edge, and must have no
	// remaining frustrated cycles.
	out.Tree.CacheTransports(z)
	cycles, _, err := out.Tree.FrustratedCycles(z, 1e-9)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Fatalf("rebuild still frustrated: %v", cycles)
	}
	if _, err := m.SortPlanes([]int{0, 1, 2}, node(z...), 1e-9, key); err != nil {
		t.Fatalf("re-sort failed: %v", err)
	}
}

// TestBlockStructure asserts all entries outside charge-matched blocks are zero:
// the invariant score only contracts charge-conjugate pairs.
func TestBlockStructure(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	a := node(complex(2, 1), complex(-3, 0.5), complex(1, -1))
	b := node(complex(0.5, -2), complex(1, 1), complex(-4, 2))
	gotR, gotI := m.Tensor.Score(a, b)
	var want complex128
	for i := range a.Zeta {
		for k := range b.Zeta {
			if m.Space.Planes[i].Charge+m.Space.Planes[k].Charge != 0 {
				continue
			}
			want += a.Zeta[i] * m.Tensor.At(i, k) * b.Zeta[k]
		}
	}
	if math.Abs(gotR-real(want)) > tol12 || math.Abs(gotI-imag(want)) > tol12 {
		t.Fatalf("score mismatch: (%v,%v) vs %v", gotR, gotI, want)
	}
	// Every block is charge-matched: rows carry Qa, cols carry Qb, and the
	// block's outgoing charge is Qa+Qb.
	for _, b := range m.Tensor.Blocks {
		if b.Out != b.Qa+b.Qb {
			t.Fatalf("block out %d != %d+%d", b.Out, b.Qa, b.Qb)
		}
		for _, i := range b.Rows {
			if m.Space.Planes[i].Charge != b.Qa {
				t.Fatalf("row %d charge %d != Qa %d", i, m.Space.Planes[i].Charge, b.Qa)
			}
		}
		for _, k := range b.Cols {
			if m.Space.Planes[k].Charge != b.Qb {
				t.Fatalf("col %d charge %d != Qb %d", k, m.Space.Planes[k].Charge, b.Qb)
			}
		}
	}
}

// TestTreeDeterminism asserts the parent array is identical across builds.
func TestTreeDeterminism(t *testing.T) {
	m := mustModel(t, []int{1, 1, 1, -1})
	for r := 0; r < 5; r++ {
		tr, err := BuildTree(m.Space, m.Tensor, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i := range tr.Parent {
			if tr.Parent[i] != m.Tree.Parent[i] {
				t.Fatalf("run %d: parent[%d]=%d want %d", r, i, tr.Parent[i], m.Tree.Parent[i])
			}
		}
	}
}

// TestByteDeterminism asserts two generations produce identical bytes.
func TestByteDeterminism(t *testing.T) {
	m1 := mustModel(t, []int{1, 1, -1})
	m2 := mustModel(t, []int{1, 1, -1})
	b1, err := m1.MarshalWeights()
	if err != nil {
		t.Fatal(err)
	}
	b2, err := m2.MarshalWeights()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("weights bytes differ across runs")
	}
}

// TestTransitivity asserts the comparator is a total preorder.
func TestTransitivity(t *testing.T) {
	m := mustModel(t, []int{1, 1, 1, -1})
	z := []complex128{complex(2, 1), complex(-3, 0.5), complex(1, -1), complex(0.25, 3)}
	m.Tree.CacheTransports(z)
	c := m.Tree.Root
	planes := []int{0, 1, 2, 3}
	for _, a := range planes {
		for _, b := range planes {
			ab, err := m.Tree.Compare(z, a, b, c)
			if err != nil {
				t.Fatal(err)
			}
			ba, err := m.Tree.Compare(z, b, a, c)
			if err != nil {
				t.Fatal(err)
			}
			if ab != -ba {
				t.Fatalf("antisymmetry broken: cmp(%d,%d)=%d cmp(%d,%d)=%d", a, b, ab, b, a, ba)
			}
			for _, d := range planes {
				if ab > 0 {
					continue
				}
				bd, err := m.Tree.Compare(z, b, d, c)
				if err != nil {
					t.Fatal(err)
				}
				ad, err := m.Tree.Compare(z, a, d, c)
				if err != nil {
					t.Fatal(err)
				}
				if bd <= 0 && ad > 0 {
					t.Fatalf("transitivity broken: %d<=%d<=%d but %d>%d", a, b, d, a, d)
				}
			}
		}
	}
}

// TestGradientCheck compares the analytic frequency gradient against a central
// finite difference to 1e-6.
func TestGradientCheck(t *testing.T) {
	m := mustModel(t, []int{1, 1, -1})
	a, b := "alpha", "beta"

	objective := func(p EncoderParams) complex128 {
		na := m.Space.Encode(p, a)
		nb := m.Space.Encode(p, b)
		re, im := m.Tensor.Score(na, nb)
		return complex(re, im)
	}

	dz := m.Space.encodeWithGrad(m.Params, a)
	deta := m.Space.encodeWithGrad(m.Params, b)
	na := m.Space.Encode(m.Params, a)
	nb := m.Space.Encode(m.Params, b)

	const h = 1e-6
	for mm := range m.Params.Freq {
		// analytic dJ/dFreq_m, restricted to the charge-conjugate channel.
		var ds complex128
		for k := range nb.Zeta {
			if m.Space.Planes[mm].Charge+m.Space.Planes[k].Charge != 0 {
				continue
			}
			ds += dz[mm][mm] * m.Tensor.At(mm, k) * nb.Zeta[k]
		}
		for i := range na.Zeta {
			if m.Space.Planes[i].Charge+m.Space.Planes[mm].Charge != 0 {
				continue
			}
			ds += deta[mm][mm] * m.Tensor.At(i, mm) * na.Zeta[i]
		}
		analytic := imag(ds)

		up := m.Params
		up.Freq = append([]float64(nil), m.Params.Freq...)
		down := m.Params
		down.Freq = append([]float64(nil), m.Params.Freq...)
		up.Freq[mm] += h
		down.Freq[mm] -= h
		numeric := (imag(objective(up)) - imag(objective(down))) / (2 * h)
		if math.Abs(analytic-numeric) > 1e-6*math.Max(1, math.Abs(numeric)) {
			t.Fatalf("grad[%d]: analytic %v numeric %v", mm, analytic, numeric)
		}
	}
}
