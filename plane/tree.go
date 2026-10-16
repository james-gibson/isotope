package plane

import (
	"cmp"
	"fmt"
	"math"
	"math/cmplx"
	"sort"
	"strings"

	"slices"
)

// Edge is an admissible plane pair (J < K) with its inverse-tensor-norm weight.
type Edge struct {
	J      int
	K      int
	Weight float64
}

// Tree is a spanning tree over the planes. Gamma holds the unwrapped (lifted)
// cumulative phase from each plane to the root along parent edges; it is filled
// lazily by CacheTransports and must not be recomputed inside a comparator.
type Tree struct {
	N         int
	Parent    []int     // -1 at the root
	Depth     []int     // root depth 0
	Gamma     []float64 // unlifted placeholder until CacheTransports
	Root      int
	Edges     []Edge // tree edges actually used
	NonTree   []Edge // admissible edges that closed a fundamental cycle
	Component []int  // union-find representative per plane
	space     Space
}

// BuildTree builds a spanning tree with Kruskal over the admissible edges,
// excluding any edge in forbidden (normalised to J < K). Edges are sorted
// stably by (weight, J, K) so the tree — and therefore the holonomy — is
// deterministic. It returns ErrDisconnected when the graph does not span every
// plane.
func BuildTree(space Space, tensor *Tensor, forbidden []Edge) (*Tree, error) {
	n := len(space.Planes)
	edges := admissibleEdges(space, tensor, forbidden)
	slices.SortStableFunc(edges, func(a, b Edge) int {
		if c := cmp.Compare(a.Weight, b.Weight); c != 0 {
			return c
		}
		if c := cmp.Compare(a.J, b.J); c != 0 {
			return c
		}
		return cmp.Compare(a.K, b.K)
	})

	uf := newUnionFind(n)
	chosen := make([]Edge, 0, n-1)
	nonTree := make([]Edge, 0, len(edges))
	for _, e := range edges {
		if uf.union(e.J, e.K) {
			chosen = append(chosen, e)
		} else {
			nonTree = append(nonTree, e)
		}
	}

	comp := make([]int, n)
	for i := 0; i < n; i++ {
		comp[i] = uf.find(i)
	}
	for i := 0; i < n; i++ {
		if comp[i] != comp[0] {
			return nil, fmt.Errorf("%w: %s", ErrDisconnected, unreachableList(space, comp))
		}
	}

	root := chooseRoot(space)
	adj := make([][]int, n)
	for _, e := range chosen {
		adj[e.J] = append(adj[e.J], e.K)
		adj[e.K] = append(adj[e.K], e.J)
	}
	for i := range adj {
		sort.Ints(adj[i])
	}

	parent := make([]int, n)
	depth := make([]int, n)
	for i := range parent {
		parent[i] = -1
	}
	seen := make([]bool, n)
	seen[root] = true
	queue := []int{root}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		for _, w := range adj[v] {
			if seen[w] {
				continue
			}
			seen[w] = true
			parent[w] = v
			depth[w] = depth[v] + 1
			queue = append(queue, w)
		}
	}

	return &Tree{
		N:         n,
		Parent:    parent,
		Depth:     depth,
		Gamma:     make([]float64, n),
		Root:      root,
		Edges:     chosen,
		NonTree:   nonTree,
		Component: comp,
		space:     space,
	}, nil
}

// CacheTransports fills Gamma with the unwrapped cumulative phase from each
// plane to the root, computed from the given coordinates. Call once per sort.
func (t *Tree) CacheTransports(z []complex128) {
	for i := range t.Gamma {
		t.Gamma[i] = 0
	}
	order := t.bfsOrder()
	for _, v := range order {
		p := t.Parent[v]
		if p < 0 {
			continue
		}
		t.Gamma[v] = t.Gamma[p] + wrapPi(edgeAngle(t.space, z, v, p))
	}
}

// Transport returns plane j's coordinate transported into the root's frame. It
// returns ErrIncomparable when j is in a different component from the root.
func (t *Tree) Transport(z []complex128, j int) (complex128, error) {
	return t.TransportTo(z, j, t.Root)
}

// TransportTo transports plane j's coordinate into the target plane's frame
// along the tree path: hat_zeta_j = zeta_j * exp(i*Gamma_{j->target}), where
// Gamma is the unwrapped cumulative phase cached by CacheTransports. It returns
// ErrIncomparable for planes in different components.
func (t *Tree) TransportTo(z []complex128, j, target int) (complex128, error) {
	if j < 0 || j >= t.N || target < 0 || target >= t.N {
		return 0, fmt.Errorf("plane: plane index out of range")
	}
	if t.Component[j] != t.Component[target] {
		return 0, fmt.Errorf("%w: planes %d and %d", ErrIncomparable, j, target)
	}
	if len(t.Gamma) != t.N {
		return 0, fmt.Errorf("plane: transports not cached; call CacheTransports first")
	}
	return z[j] * cmplxExp(0, t.Gamma[j]-t.Gamma[target]), nil
}

// LCA returns the lowest common ancestor of planes a and b in the tree.
func (t *Tree) LCA(a, b int) int {
	x, y := a, b
	for t.Depth[x] > t.Depth[y] {
		x = t.Parent[x]
	}
	for t.Depth[y] > t.Depth[x] {
		y = t.Parent[y]
	}
	for x != y {
		x = t.Parent[x]
		y = t.Parent[y]
	}
	return x
}

// ComponentOf returns the connected-component representative of a plane.
func (t *Tree) ComponentOf(j int) int {
	if j < 0 || j >= t.N {
		return -1
	}
	return t.Component[j]
}

// edgeAngle is the principal edge phase between planes j and k:
//
//	arg(z_j / z_k) when q_j == q_k,
//	arg(z_j * z_k) when q_j == -q_k.
func edgeAngle(space Space, z []complex128, j, k int) float64 {
	qj := space.Planes[j].Charge
	qk := space.Planes[k].Charge
	var a float64
	if qj == qk {
		a = cmplx.Phase(z[j]) - cmplx.Phase(z[k])
	} else {
		a = cmplx.Phase(z[j]) + cmplx.Phase(z[k])
	}
	return wrapPi(a)
}

// wrapPi wraps an angle into (-pi, pi]. It maps exactly zero to zero.
func wrapPi(x float64) float64 {
	r := math.Mod(x, 2*math.Pi)
	if r > math.Pi {
		r -= 2 * math.Pi
	}
	if r <= -math.Pi {
		r += 2 * math.Pi
	}
	return r
}

func admissibleEdges(space Space, tensor *Tensor, forbidden []Edge) []Edge {
	n := len(space.Planes)
	edges := make([]Edge, 0, n*(n-1)/2)
	for j := 0; j < n; j++ {
		for k := j + 1; k < n; k++ {
			qj := space.Planes[j].Charge
			qk := space.Planes[k].Charge
			if qj != qk && qj != -qk {
				continue
			}
			if isForbidden(forbidden, j, k) {
				continue
			}
			norm := 0.0
			if tensor != nil {
				norm = tensor.Norm(qj)
			}
			edges = append(edges, Edge{J: j, K: k, Weight: 1.0 / (norm + 1e-9)})
		}
	}
	return edges
}

func isForbidden(forbidden []Edge, j, k int) bool {
	lo, hi := j, k
	if lo > hi {
		lo, hi = hi, lo
	}
	for _, e := range forbidden {
		a, b := e.J, e.K
		if a > b {
			a, b = b, a
		}
		if a == lo && b == hi {
			return true
		}
	}
	return false
}

func chooseRoot(space Space) int {
	for j := range space.Planes {
		if space.Planes[j].Charge == 0 {
			return j
		}
	}
	return 0
}

func unreachableList(space Space, comp []int) string {
	var b strings.Builder
	first := comp[0]
	for i := range space.Planes {
		if comp[i] != first {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			b.WriteString(space.Planes[i].Name)
		}
	}
	return b.String()
}

func (t *Tree) bfsOrder() []int {
	order := make([]int, 0, t.N)
	seen := make([]bool, t.N)
	queue := []int{t.Root}
	seen[t.Root] = true
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		order = append(order, v)
		var children []int
		for c := 0; c < t.N; c++ {
			if t.Parent[c] == v {
				children = append(children, c)
			}
		}
		sort.Ints(children)
		for _, c := range children {
			if !seen[c] {
				seen[c] = true
				queue = append(queue, c)
			}
		}
	}
	return order
}

type unionFind struct {
	parent []int
	rank   []int
}

func newUnionFind(n int) *unionFind {
	uf := &unionFind{parent: make([]int, n), rank: make([]int, n)}
	for i := range uf.parent {
		uf.parent[i] = i
	}
	return uf
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) bool {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return false
	}
	if u.rank[ra] < u.rank[rb] {
		ra, rb = rb, ra
	}
	u.parent[rb] = ra
	if u.rank[ra] == u.rank[rb] {
		u.rank[ra]++
	}
	return true
}
