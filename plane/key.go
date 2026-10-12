package plane

import (
	"cmp"
	"math"
	"math/cmplx"
)

// Distance returns the gauge-invariant distance d(A,C) between the coordinates
// of planes a and c: the magnitude of a transported into c's frame minus c's
// own coordinate. Both terms transform as exp(i*q_c*theta) under a global
// rotation, so the magnitude is invariant.
func (t *Tree) Distance(z []complex128, a, c int) (float64, error) {
	hat, err := t.TransportTo(z, a, c)
	if err != nil {
		return 0, err
	}
	if c < 0 || c >= len(z) {
		return 0, ErrIncomparable
	}
	return cmplx.Abs(hat - z[c]), nil
}

// TripleKey is the three-point sort key k(A,B;C) together with the holonomy and
// frustration of the triple. Distance is the invariant primary; Phase is the
// cyclic tiebreak only. A triple that sorts cleanly but carries non-zero
// holonomy is a contradiction and is reported, not suppressed.
type TripleKey struct {
	A           int     `json:"a"`
	B           int     `json:"b"`
	C           int     `json:"c"`
	Distance    float64 `json:"distance"`
	Phase       float64 `json:"phase"`
	Holonomy    float64 `json:"holonomy"`
	Frustrated  bool    `json:"frustrated"`
	Quarantined bool    `json:"quarantined"`
}

// Triple computes k(A,B;C) for three planes using the given coordinate vector.
// Distance is d(A,C)+d(B,C); Phase is arg[hat_zeta_A conj(hat_zeta_B)] with both
// coordinates transported into C's frame. Holonomy is the holonomy of the first
// non-tree edge among the pairs (A,B), (B,C), (A,C), in that deterministic
// priority order.
func (t *Tree) Triple(z []complex128, a, b, c int, tol float64) (TripleKey, error) {
	dAC, err := t.Distance(z, a, c)
	if err != nil {
		return TripleKey{}, err
	}
	dBC, err := t.Distance(z, b, c)
	if err != nil {
		return TripleKey{}, err
	}
	hatA, err := t.TransportTo(z, a, c)
	if err != nil {
		return TripleKey{}, err
	}
	hatB, err := t.TransportTo(z, b, c)
	if err != nil {
		return TripleKey{}, err
	}
	phase := wrapPi(cmplx.Phase(hatA) - cmplx.Phase(hatB))

	key := TripleKey{
		A:        a,
		B:        b,
		C:        c,
		Distance: dAC + dBC,
		Phase:    phase,
	}
	for _, pair := range [][2]int{{a, b}, {b, c}, {a, c}} {
		phi, ok, err := t.CycleHolonomy(z, pair[0], pair[1])
		if err != nil {
			return TripleKey{}, err
		}
		if ok {
			key.Holonomy = phi
			break
		}
	}
	key.Frustrated = math.Abs(key.Holonomy) > tol
	key.Quarantined = key.Frustrated
	return key, nil
}

// Compare orders planes a and b using reference plane c: the invariant distance
// to c is primary and the frame phase is the tiebreak. It is NaN-safe via
// cmp.Compare, and lexicographic on two total orders, so it is a total
// preorder suitable for a stable sort.
func (t *Tree) Compare(z []complex128, a, b, c int) (int, error) {
	da, err := t.Distance(z, a, c)
	if err != nil {
		return 0, err
	}
	db, err := t.Distance(z, b, c)
	if err != nil {
		return 0, err
	}
	if r := cmp.Compare(da, db); r != 0 {
		return r, nil
	}
	ha, err := t.TransportTo(z, a, c)
	if err != nil {
		return 0, err
	}
	hb, err := t.TransportTo(z, b, c)
	if err != nil {
		return 0, err
	}
	return cmp.Compare(cmplx.Phase(ha), cmplx.Phase(hb)), nil
}

// Key returns the node's canonical scalar in the root frame: the sum of every
// plane's coordinate transported to the root. It transforms as exp(i*q_root*
// theta) under a global rotation, so ratios and differences of keys are gauge
// invariant.
func (m *Model) Key(n Node) (complex128, error) {
	m.Tree.CacheTransports(n.Zeta)
	var sum complex128
	for j := range n.Zeta {
		v, err := m.Tree.Transport(n.Zeta, j)
		if err != nil {
			return 0, err
		}
		sum += v
	}
	return sum, nil
}
