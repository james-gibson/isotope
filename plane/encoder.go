package plane

import (
	"math/cmplx"
)

// EncoderParams are the frozen, one-shot parameters of the plane encoder. They
// are derived deterministically from the charge list and anchor; nothing is
// searched or learned by gradient descent in this package.
type EncoderParams struct {
	Freq  []float64 `json:"freq"`
	Phase []float64 `json:"phase"`
	Gain  []float64 `json:"gain"`
	Decay float64   `json:"decay"`
}

// DefaultParams derives the frozen encoder parameters for a charge list and
// anchor. The derivation is pure and deterministic: same input, same params.
func DefaultParams(charges []int, anchor complex128) EncoderParams {
	n := len(charges)
	p := EncoderParams{
		Freq:  make([]float64, n),
		Phase: make([]float64, n),
		Gain:  make([]float64, n),
		Decay: 0.95,
	}
	base := cmplx.Abs(anchor)
	if base == 0 {
		base = 42
	}
	for j, q := range charges {
		f := 1 + 0.25*float64(q) + 0.1*float64(j)
		if f == 0 {
			f = 0.37
		}
		p.Freq[j] = f
		p.Phase[j] = 0.3*float64(q) + 0.37*float64(j) + 0.011*base
		p.Gain[j] = base + 5*float64(iabs(q)) + float64(j)
	}
	return p
}

// Encode maps a string to a centered node. Accumulation order is fixed: runes
// outer, planes inner, in index order. There is no map iteration and no clock
// read, so the output is bit-identical across runs for the same inputs.
func (s Space) Encode(p EncoderParams, str string) Node {
	n := len(s.Planes)
	z := make([]complex128, n)
	runes := []rune(str)
	cum := 0.0
	pow := 1.0
	for i := 0; i < len(runes); i++ {
		cum += float64(runes[i])/128.0 + float64(i+1)*0.5
		for j := 0; j < n; j++ {
			gain := 0.0
			if j < len(p.Gain) {
				gain = p.Gain[j]
			}
			freq := 0.0
			if j < len(p.Freq) {
				freq = p.Freq[j]
			}
			phase := 0.0
			if j < len(p.Phase) {
				phase = p.Phase[j]
			}
			z[j] += complex(pow*gain, 0) * cmplx.Exp(complex(0, freq*cum+phase))
		}
		pow *= p.Decay
	}
	zeta := make([]complex128, n)
	for j := 0; j < n; j++ {
		zeta[j] = z[j] - s.Anchor
	}
	return Node{Zeta: zeta}
}

// encodeWithGrad returns the analytic derivative of each plane's centered
// coordinate with respect to that plane's frequency parameter. The derivative
// array is diagonal: dz[j][k] is d(zeta_j)/d(freq_k), nonzero only when j == k.
// This powers the finite-difference gradient check.
func (s Space) encodeWithGrad(p EncoderParams, str string) [][]complex128 {
	n := len(s.Planes)
	z := make([]complex128, n)
	dz := make([][]complex128, n)
	for j := range dz {
		dz[j] = make([]complex128, n)
	}
	runes := []rune(str)
	cum := 0.0
	pow := 1.0
	for i := 0; i < len(runes); i++ {
		cum += float64(runes[i])/128.0 + float64(i+1)*0.5
		for j := 0; j < n; j++ {
			gain, freq, phase := 0.0, 0.0, 0.0
			if j < len(p.Gain) {
				gain = p.Gain[j]
			}
			if j < len(p.Freq) {
				freq = p.Freq[j]
			}
			if j < len(p.Phase) {
				phase = p.Phase[j]
			}
			term := complex(pow*gain, 0) * cmplx.Exp(complex(0, freq*cum+phase))
			z[j] += term
			dz[j][j] += term * complex(0, cum)
		}
		pow *= p.Decay
	}
	return dz
}
