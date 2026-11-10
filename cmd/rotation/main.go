// Command rotation projects a set of charges through the plane at both standard
// anchors and reports the two readings and the rotation between them.
//
// It exists so brine_queue.py can record a frame-aware reading for every spec it
// classifies without reimplementing the plane: the model stays the authority,
// and the queue only supplies the projection.
//
// Input (stdin), one batch:
//
//	{"requests": [{"id": "...", "charges": [1,-1,...], "probe": "..."}]}
//
// Output (stdout):
//
//	{"results": [{"id": "...", "frames": {"42": {...}, "42i": {...}},
//	              "rotation": {...}}], "anchor_a": "42", "anchor_b": "42i"}
//
// The rotation is per plane; `uniform` reports whether every plane moved by the
// same displacement, which is the invariant the specs pin. If a probe does not
// resolve, the result carries an `error` instead of frames.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"os"

	"github.com/james-gibson/isotope/plane"
)

const (
	anchorA = "42"
	anchorB = "42i"
)

type request struct {
	ID      string `json:"id"`
	Charges []int  `json:"charges"`
	Probe   string `json:"probe"`
}

type batch struct {
	Requests []request `json:"requests"`
}

type frameReading struct {
	KeyReal     float64 `json:"key_real"`
	KeyImag     float64 `json:"key_imag"`
	Magnitude   float64 `json:"magnitude"`
	Planes      int     `json:"planes"`
	Quarantined bool    `json:"quarantined"`
}

type rotation struct {
	DisplacementReal float64 `json:"displacement_real"`
	DisplacementImag float64 `json:"displacement_imag"`
	Magnitude        float64 `json:"magnitude"`
	AngleDegrees     float64 `json:"angle_deg"`
	Uniform          bool    `json:"uniform"`
	MatchesInvariant bool    `json:"matches_invariant"`
}

type result struct {
	ID       string                   `json:"id"`
	Frames   map[string]*frameReading `json:"frames,omitempty"`
	Rotation *rotation                `json:"rotation,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

type response struct {
	AnchorA string   `json:"anchor_a"`
	AnchorB string   `json:"anchor_b"`
	Results []result `json:"results"`
}

func readFrame(charges []int, probe, anchorName string) (*frameReading, []complex128, error) {
	anchor, err := plane.ParseAnchor(anchorName)
	if err != nil {
		return nil, nil, err
	}
	model, err := plane.GenerateModel(charges, anchor, "")
	if err != nil {
		return nil, nil, err
	}
	node := model.Node(probe)
	key, err := model.Key(node)
	if err != nil {
		return nil, node.Zeta, err
	}
	return &frameReading{
		KeyReal:   real(key),
		KeyImag:   imag(key),
		Magnitude: cmplx.Abs(key),
		Planes:    len(node.Zeta),
	}, node.Zeta, nil
}

func compute(req request) result {
	res := result{ID: req.ID}
	if len(req.Charges) == 0 {
		res.Error = "no charges: a spec with no parts has no projection"
		return res
	}
	ra, za, errA := readFrame(req.Charges, req.Probe, anchorA)
	rb, zb, errB := readFrame(req.Charges, req.Probe, anchorB)
	if errA != nil || errB != nil {
		res.Error = fmt.Sprintf("frame read failed: %v / %v", errA, errB)
		return res
	}
	if len(za) != len(zb) || len(za) == 0 {
		res.Error = "plane counts disagree between frames"
		return res
	}

	delta := zb[0] - za[0]
	uniform := true
	for j := range za {
		d := zb[j] - za[j]
		if cmplx.Abs(d-delta) > 1e-9 {
			uniform = false
			break
		}
	}
	res.Rotation = &rotation{
		DisplacementReal: real(delta),
		DisplacementImag: imag(delta),
		Magnitude:        cmplx.Abs(delta),
		AngleDegrees:     cmplx.Phase(delta) * 180 / math.Pi,
		Uniform:          uniform,
		MatchesInvariant: math.Abs(real(delta)-42) < 1e-9 && math.Abs(imag(delta)+42) < 1e-9,
	}
	res.Frames = map[string]*frameReading{anchorA: ra, anchorB: rb}
	res.Frames[anchorA].Quarantined = false
	res.Frames[anchorB].Quarantined = false
	return res
}

func main() {
	var in batch
	dec := json.NewDecoder(bufio.NewReader(os.Stdin))
	if err := dec.Decode(&in); err != nil {
		fmt.Fprintf(os.Stderr, "rotation: bad input: %v\n", err)
		os.Exit(2)
	}
	out := response{AnchorA: anchorA, AnchorB: anchorB, Results: make([]result, 0, len(in.Requests))}
	for _, req := range in.Requests {
		out.Results = append(out.Results, compute(req))
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "rotation: cannot write output: %v\n", err)
		os.Exit(2)
	}
}
