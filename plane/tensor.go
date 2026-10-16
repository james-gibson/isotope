package plane

import (
	"math"
	"sort"
)

// Block is one charge-matched block of the covariant tensor. Rows lists plane
// indices whose charge is Qa, Cols lists plane indices whose charge is Qb, and
// W holds the block entries row-major (len(Rows)*len(Cols) values). The block's
// outgoing charge is Out = Qa+Qb. Entries outside charge-matched blocks do not
// exist: the tensor is never materialized as a dense N x N matrix.
type Block struct {
	Qa   int   `json:"qa"`
	Qb   int   `json:"qb"`
	Out  int   `json:"out"`
	Rows []int `json:"rows"`
	Cols []int `json:"cols"`
	W    []complex128
}

// Tensor is the covariant, block-diagonal edge tensor T = sum over charge
// pairs of T_{Qa,Qb}. The invariant (charge-conserving) channel is the sum of
// blocks with Out == 0; the covariant relaxation admits every Qa+Qb == Out.
type Tensor struct {
	Blocks  []Block
	charges []int
}

// BuildTensor derives the block layout and weights from a space's charge list.
// Blocks are produced in a fixed order: distinct charges ascending, then the
// column charge ascending. No maps are used.
func BuildTensor(space Space) *Tensor {
	n := len(space.Planes)
	charges := space.Charges()
	vals := distinctSorted(charges)
	t := &Tensor{charges: charges}
	for _, qa := range vals {
		for _, qb := range vals {
			rows := planesWithCharge(charges, qa)
			cols := planesWithCharge(charges, qb)
			if len(rows) == 0 || len(cols) == 0 {
				continue
			}
			blk := Block{
				Qa:   qa,
				Qb:   qb,
				Out:  qa + qb,
				Rows: rows,
				Cols: cols,
				W:    make([]complex128, len(rows)*len(cols)),
			}
			for ri, i := range rows {
				for ci, k := range cols {
					blk.W[ri*len(cols)+ci] = blockWeight(qa, qb, i, k, n)
				}
			}
			t.Blocks = append(t.Blocks, blk)
		}
	}
	return t
}

// blockWeight is the deterministic weight of a single tensor entry. It depends
// only on the two charges and the two plane indices, never on a clock.
func blockWeight(qa, qb, i, k, n int) complex128 {
	theta := math.Pi*float64(qa-qb)/float64(n+1) + 0.017*float64(i-k)
	amp := 1.0 / float64(1+iabs(qa)+iabs(qb))
	return complex(amp*math.Cos(theta), amp*math.Sin(theta))
}

// At returns the tensor entry coupling plane i to plane k, or zero when the
// pair has no charge-matched block. It never allocates a dense matrix.
func (t *Tensor) At(i, k int) complex128 {
	if i < 0 || i >= len(t.charges) || k < 0 || k >= len(t.charges) {
		return 0
	}
	qa := t.charges[i]
	qb := t.charges[k]
	for bi := range t.Blocks {
		b := &t.Blocks[bi]
		if b.Qa != qa || b.Qb != qb {
			continue
		}
		ri := indexOf(b.Rows, i)
		ci := indexOf(b.Cols, k)
		if ri < 0 || ci < 0 {
			return 0
		}
		return b.W[ri*len(b.Cols)+ci]
	}
	return 0
}

// Score returns the real and imaginary parts of the U(1)-invariant pairing
//
//	s(a,b) = sum over charge-conjugate blocks (Out == 0)
//	         of zeta_a[rows] . T[rows,cols] . zeta_b[cols].
//
// The pairing is bilinear rather than sesquilinear so that it is invariant
// under the charge-weighted action R_theta; the relative-rotation channel is
// preserved in the imaginary part and reported separately by callers.
func (t *Tensor) Score(a, b Node) (realPart, imagPart float64) {
	var s complex128
	for bi := range t.Blocks {
		blk := &t.Blocks[bi]
		if blk.Out != 0 {
			continue
		}
		nc := len(blk.Cols)
		for ri, i := range blk.Rows {
			if i >= len(a.Zeta) {
				continue
			}
			for ci, k := range blk.Cols {
				if k >= len(b.Zeta) {
					continue
				}
				s += a.Zeta[i] * blk.W[ri*nc+ci] * b.Zeta[k]
			}
		}
	}
	return real(s), imag(s)
}

// Norm returns the Frobenius norm of the block coupling charge c to its
// conjugate -c. It is the source of the inverse-tensor-norm edge weights.
func (t *Tensor) Norm(c int) float64 {
	var s float64
	for bi := range t.Blocks {
		b := &t.Blocks[bi]
		if b.Qa != c || b.Qb != -c {
			continue
		}
		for _, w := range b.W {
			s += real(w)*real(w) + imag(w)*imag(w)
		}
	}
	return math.Sqrt(s)
}

func distinctSorted(xs []int) []int {
	cp := append([]int(nil), xs...)
	sort.Ints(cp)
	out := cp[:0]
	for i, v := range cp {
		if i == 0 || v != cp[i-1] {
			out = append(out, v)
		}
	}
	return append([]int(nil), out...)
}

func planesWithCharge(charges []int, q int) []int {
	var out []int
	for i, c := range charges {
		if c == q {
			out = append(out, i)
		}
	}
	return out
}

func indexOf(xs []int, v int) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}
