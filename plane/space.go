package plane

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Plane is one plane j carrying an integer U(1) charge. Under the U(1) action
// a plane's coordinate transforms as z_j -> exp(i*q_j*theta) z_j.
type Plane struct {
	Name   string `json:"name"`
	Charge int    `json:"charge"`
}

// Node is a centered coordinate vector: exactly one complex value per plane.
// Zeta has the same length as the owning Space's plane list.
type Node struct {
	Zeta []complex128
}

// Space is an ordered set of planes together with the single anchor fixed for
// the run. Anchor is 42+0i (proven consensus, real axis) or 0+42i (working
// hypothesis, imaginary axis). Exactly one anchor is used for a whole run.
type Space struct {
	Planes []Plane
	Anchor complex128
}

// ErrIncomparable is returned when two planes live in different connected
// components of the admissibility graph. Such planes are genuinely
// incomparable: the library never substitutes a default phase of zero for them.
var ErrIncomparable = fmt.Errorf("plane: planes are in different components")

// ErrDisconnected is returned when the admissibility graph does not span all
// planes, so no spanning tree exists.
var ErrDisconnected = fmt.Errorf("plane: charge graph is disconnected")

// iabs returns the absolute value of an int without a float round-trip.
func iabs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ParseAnchor parses an --anchor flag value. Accepted spellings are "42",
// "42i", "42+0i", and "0+42i"; anything else is an error.
func ParseAnchor(s string) (complex128, error) {
	switch strings.TrimSpace(s) {
	case "42", "42+0i", "(42+0i)":
		return complex(42, 0), nil
	case "42i", "0+42i", "(0+42i)":
		return complex(0, 42), nil
	default:
		return 0, fmt.Errorf("plane: invalid anchor %q: want 42 or 42i", s)
	}
}

// ParseCharges parses a comma-separated charge list such as "1,-1,1".
func ParseCharges(s string) ([]int, error) {
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("plane: empty charge list")
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("plane: empty charge in list %q", s)
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("plane: invalid charge %q: %w", p, err)
		}
		out = append(out, n)
	}
	return out, nil
}

// NewSpace builds a Space from a charge list and an anchor, naming each plane
// deterministically as "p<index>:q<charge>".
func NewSpace(charges []int, anchor complex128) (Space, error) {
	if len(charges) == 0 {
		return Space{}, fmt.Errorf("plane: need at least one plane")
	}
	planes := make([]Plane, len(charges))
	for j, q := range charges {
		planes[j] = Plane{Name: fmt.Sprintf("p%d:q%d", j, q), Charge: q}
	}
	return Space{Planes: planes, Anchor: anchor}, nil
}

// Charges returns the charge of each plane, in plane order.
func (s Space) Charges() []int {
	out := make([]int, len(s.Planes))
	for j := range s.Planes {
		out[j] = s.Planes[j].Charge
	}
	return out
}

// Rotate applies the U(1) action R_theta to a node: for every plane j,
// zeta_j -> exp(i*q_j*theta) zeta_j. It returns a fresh node; the input is not
// modified.
func (s Space) Rotate(n Node, theta float64) Node {
	out := Node{Zeta: make([]complex128, len(n.Zeta))}
	for j := range n.Zeta {
		q := 0.0
		if j < len(s.Planes) {
			q = float64(s.Planes[j].Charge)
		}
		ph := cmplxExp(0, q*theta)
		out.Zeta[j] = n.Zeta[j] * ph
	}
	return out
}

// PlaneIndex resolves a plane selector: either a 0-based integer index or an
// exact plane name. It reports an error for out-of-range or unknown selectors.
func (s Space) PlaneIndex(sel string) (int, error) {
	sel = strings.TrimSpace(sel)
	if n, err := strconv.Atoi(sel); err == nil {
		if n < 0 || n >= len(s.Planes) {
			return 0, fmt.Errorf("plane: plane index %d out of range [0,%d)", n, len(s.Planes))
		}
		return n, nil
	}
	for j := range s.Planes {
		if s.Planes[j].Name == sel {
			return j, nil
		}
	}
	return 0, fmt.Errorf("plane: unknown plane %q", sel)
}

// cmplxExp is exp(i*theta) with real part cos(theta) and imaginary part
// sin(theta). Kept local so the encoder and rotation share one definition.
func cmplxExp(_, theta float64) complex128 {
	return complex(math.Cos(theta), math.Sin(theta))
}
