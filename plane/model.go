package plane

import (
	"slices"
)

// Cplx is the JSON form of a complex number: real and imaginary parts are
// always emitted as separate fields, never as one opaque value.
type Cplx struct {
	Real float64 `json:"real"`
	Imag float64 `json:"imag"`
}

func toCplx(z complex128) Cplx { return Cplx{Real: real(z), Imag: imag(z)} }

func fromCplx(v Cplx) complex128 { return complex(v.Real, v.Imag) }

// BlockView is the JSON-friendly view of a tensor block: the complex weights
// are expanded to real/imaginary parts.
type BlockView struct {
	Qa   int    `json:"qa"`
	Qb   int    `json:"qb"`
	Out  int    `json:"out"`
	Rows []int  `json:"rows"`
	Cols []int  `json:"cols"`
	W    []Cplx `json:"w"`
}

// EdgeResult is the output of the edge subcommand: the charge-matched blocks
// that couple the two planes, the score (real part) and the phase (arg).
type EdgeResult struct {
	A      int         `json:"a"`
	B      int         `json:"b"`
	Blocks []BlockView `json:"blocks"`
	Score  float64     `json:"score"`
	Phase  float64     `json:"phase"`
}

// SortResult is the output of the sort subcommand: the plane order, the signed
// quarantine events observed, and whether any quarantine happened.
type SortResult struct {
	Order       []int             `json:"order"`
	Names       []string          `json:"names"`
	Events      []QuarantineEvent `json:"events"`
	Quarantined bool              `json:"quarantined"`
}

// Edge returns the tensor blocks coupling planes a and b and the score of their
// probe coordinates: entry * zeta_a * zeta_b, split into real score and phase.
func (m *Model) Edge(a, b int, probe Node) (EdgeResult, error) {
	if m.Tensor == nil || probe.Zeta == nil {
		return EdgeResult{}, ErrIncomparable
	}
	if a < 0 || a >= len(m.Space.Planes) || b < 0 || b >= len(m.Space.Planes) {
		return EdgeResult{}, ErrIncomparable
	}
	res := EdgeResult{A: a, B: b}
	qa := m.Space.Planes[a].Charge
	qb := m.Space.Planes[b].Charge
	for bi := range m.Tensor.Blocks {
		blk := m.Tensor.Blocks[bi]
		if blk.Qa != qa || blk.Qb != qb {
			continue
		}
		view := BlockView{Qa: blk.Qa, Qb: blk.Qb, Out: blk.Out, Rows: blk.Rows, Cols: blk.Cols}
		view.W = make([]Cplx, len(blk.W))
		for i, w := range blk.W {
			view.W[i] = toCplx(w)
		}
		res.Blocks = append(res.Blocks, view)
	}
	entry := m.Tensor.At(a, b)
	s := entry * probe.Zeta[a] * probe.Zeta[b]
	res.Score = real(s)
	res.Phase = phaseOf(s)
	return res, nil
}

// TripleKeyPlanes computes the three-point key for planes a, b and c using the
// probe coordinate vector. Transports are cached first.
func (m *Model) TripleKeyPlanes(a, b, c int, probe Node, tol float64) (TripleKey, error) {
	m.Tree.CacheTransports(probe.Zeta)
	return m.Tree.Triple(probe.Zeta, a, b, c, tol)
}

// SortPlanes orders the selected planes with the three-point comparator, after
// running the quarantine loop on the plane tree. The reference plane for the
// comparator is the tree root.
func (m *Model) SortPlanes(selectors []int, probe Node, tol float64, key []byte) (*SortResult, error) {
	outcome, err := Quarantine(m.Space, m.Tensor, probe.Zeta, tol, key)
	if err != nil {
		return nil, err
	}
	tree := outcome.Tree
	order := append([]int(nil), selectors...)
	var sortErr error
	slices.SortStableFunc(order, func(x, y int) int {
		c, err := tree.Compare(probe.Zeta, x, y, tree.Root)
		if err != nil {
			sortErr = err
			return 0
		}
		return c
	})
	if sortErr != nil {
		return nil, sortErr
	}
	res := &SortResult{Order: order, Quarantined: outcome.Quarantined, Events: outcome.Events}
	res.Names = make([]string, len(order))
	for i, p := range order {
		res.Names[i] = m.Space.Planes[p].Name
	}
	return res, nil
}
