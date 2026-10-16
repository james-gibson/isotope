package plane

import (
	"fmt"
	"math"
)

// CycleHolonomy returns the holonomy of the fundamental cycle closed by the
// non-tree edge (j,k):
//
//	Phi = gamma_(j,k) - (Gamma_{j->LCA} - Gamma_{k->LCA}), wrapped to (-pi,pi].
//
// The bool reports whether (j,k) is a non-tree edge. When it is not there is no
// fundamental cycle, and the holonomy is reported as zero. Gamma must have been
// filled by CacheTransports.
func (t *Tree) CycleHolonomy(z []complex128, j, k int) (float64, bool, error) {
	if len(t.Gamma) != t.N {
		return 0, false, fmt.Errorf("plane: transports not cached; call CacheTransports first")
	}
	e, ok := t.nonTreeEdge(j, k)
	if !ok {
		return 0, false, nil
	}
	gamma := edgeAngle(t.space, z, e.J, e.K)
	phi := wrapPi(gamma - (t.Gamma[e.J] - t.Gamma[e.K]))
	return phi, true, nil
}

// FrustratedCycles returns, in deterministic non-tree-edge order, the edges
// whose fundamental cycle is frustrated (|Phi| > tol), together with their
// holonomies.
func (t *Tree) FrustratedCycles(z []complex128, tol float64) ([]Edge, []float64, error) {
	var edges []Edge
	var phis []float64
	for _, e := range t.NonTree {
		phi, ok, err := t.CycleHolonomy(z, e.J, e.K)
		if err != nil {
			return nil, nil, err
		}
		if ok && math.Abs(phi) > tol {
			edges = append(edges, e)
			phis = append(phis, phi)
		}
	}
	return edges, phis, nil
}

// FundamentalCycleCount is the number of independent cycles in the plane graph:
// E_total - (N-1), where E_total is every admissible edge.
func (t *Tree) FundamentalCycleCount() int {
	return len(t.NonTree)
}

func (t *Tree) nonTreeEdge(j, k int) (Edge, bool) {
	lo, hi := j, k
	if lo > hi {
		lo, hi = hi, lo
	}
	for _, e := range t.NonTree {
		if e.J == lo && e.K == hi {
			return e, true
		}
	}
	return Edge{}, false
}
