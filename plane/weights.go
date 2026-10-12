package plane

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
)

// Model is a fully materialized plane run: the charge space with its fixed
// anchor, the frozen encoder parameters, the covariant tensor, and the
// spanning tree derived from them.
type Model struct {
	Space  Space
	Params EncoderParams
	Tensor *Tensor
	Tree   *Tree
}

//go:embed weights.json
var embeddedWeights []byte

// GenerateModel derives a model in one shot from a charge list, an anchor, and
// a corpus. The corpus is read only to fold one deterministic global phase into
// the encoder parameters; there is no search and no gradient training.
func GenerateModel(charges []int, anchor complex128, corpus string) (*Model, error) {
	space, err := NewSpace(charges, anchor)
	if err != nil {
		return nil, err
	}
	params := DefaultParams(charges, anchor)
	if corpus != "" {
		g := corpusPhase(corpus)
		for j := range params.Phase {
			params.Phase[j] += g
		}
	}
	tensor := BuildTensor(space)
	tree, err := BuildTree(space, tensor, nil)
	if err != nil {
		return nil, err
	}
	return &Model{Space: space, Params: params, Tensor: tensor, Tree: tree}, nil
}

// EmbeddedModel parses the model embedded into the binary at build time.
func EmbeddedModel() (*Model, error) {
	return LoadWeights(embeddedWeights)
}

// Node encodes a string into a centered node using the model's frozen encoder.
func (m *Model) Node(str string) Node {
	return m.Space.Encode(m.Params, str)
}

// corpusPhase folds a corpus into a single deterministic phase in [0,2*pi).
func corpusPhase(corpus string) float64 {
	var h uint64 = 1469598103934665603
	for i := 0; i < len(corpus); i++ {
		h ^= uint64(corpus[i])
		h *= 1099511628211
	}
	return float64(h%1000003) / 1000003.0 * 2 * math.Pi
}

func phaseOf(z complex128) float64 {
	return cmplx.Phase(z)
}

type edgeDTO struct {
	J      int     `json:"j"`
	K      int     `json:"k"`
	Weight float64 `json:"weight"`
}

type blockDTO struct {
	Qa   int    `json:"qa"`
	Qb   int    `json:"qb"`
	Out  int    `json:"out"`
	Rows []int  `json:"rows"`
	Cols []int  `json:"cols"`
	W    []Cplx `json:"w"`
}

type treeDTO struct {
	N       int       `json:"n"`
	Root    int       `json:"root"`
	Parent  []int     `json:"parent"`
	Depth   []int     `json:"depth"`
	Edges   []edgeDTO `json:"edges"`
	NonTree []edgeDTO `json:"non_tree"`
}

type weightFile struct {
	Version int           `json:"version"`
	Charges []int         `json:"charges"`
	Anchor  Cplx          `json:"anchor"`
	Encoder EncoderParams `json:"encoder"`
	Tensor  []blockDTO    `json:"tensor"`
	Tree    treeDTO       `json:"tree"`
}

// MarshalWeights serializes the model to the canonical weights.json form.
func (m *Model) MarshalWeights() ([]byte, error) {
	wf := weightFile{
		Version: 1,
		Charges: m.Space.Charges(),
		Anchor:  toCplx(m.Space.Anchor),
		Encoder: m.Params,
	}
	for bi := range m.Tensor.Blocks {
		b := m.Tensor.Blocks[bi]
		dto := blockDTO{Qa: b.Qa, Qb: b.Qb, Out: b.Out, Rows: b.Rows, Cols: b.Cols}
		dto.W = make([]Cplx, len(b.W))
		for i, w := range b.W {
			dto.W[i] = toCplx(w)
		}
		wf.Tensor = append(wf.Tensor, dto)
	}
	wf.Tree.N = m.Tree.N
	wf.Tree.Root = m.Tree.Root
	wf.Tree.Parent = m.Tree.Parent
	wf.Tree.Depth = m.Tree.Depth
	wf.Tree.Edges = toEdgeDTOs(m.Tree.Edges)
	wf.Tree.NonTree = toEdgeDTOs(m.Tree.NonTree)
	return json.MarshalIndent(wf, "", "  ")
}

// LoadWeights parses a weights.json document into a model.
func LoadWeights(data []byte) (*Model, error) {
	var wf weightFile
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("plane: parse weights: %w", err)
	}
	if len(wf.Charges) == 0 {
		return nil, fmt.Errorf("plane: weights have no charges")
	}
	space, err := NewSpace(wf.Charges, fromCplx(wf.Anchor))
	if err != nil {
		return nil, err
	}
	tensor := &Tensor{charges: wf.Charges}
	for _, b := range wf.Tensor {
		blk := Block{Qa: b.Qa, Qb: b.Qb, Out: b.Out, Rows: b.Rows, Cols: b.Cols}
		blk.W = make([]complex128, len(b.W))
		for i, w := range b.W {
			blk.W[i] = fromCplx(w)
		}
		tensor.Blocks = append(tensor.Blocks, blk)
	}
	tree, err := treeFromDTO(space, wf.Tree)
	if err != nil {
		return nil, err
	}
	return &Model{Space: space, Params: wf.Encoder, Tensor: tensor, Tree: tree}, nil
}

func toEdgeDTOs(edges []Edge) []edgeDTO {
	out := make([]edgeDTO, len(edges))
	for i, e := range edges {
		out[i] = edgeDTO(e)
	}
	return out
}

func treeFromDTO(space Space, dto treeDTO) (*Tree, error) {
	n := len(space.Planes)
	if dto.N != 0 && dto.N != n {
		return nil, fmt.Errorf("plane: tree size %d does not match %d planes", dto.N, n)
	}
	t := &Tree{
		N:       n,
		Root:    dto.Root,
		Parent:  append([]int(nil), dto.Parent...),
		Depth:   append([]int(nil), dto.Depth...),
		Gamma:   make([]float64, n),
		space:   space,
		Edges:   fromEdgeDTOs(dto.Edges),
		NonTree: fromEdgeDTOs(dto.NonTree),
	}
	uf := newUnionFind(n)
	for _, e := range t.Edges {
		uf.union(e.J, e.K)
	}
	t.Component = make([]int, n)
	for i := 0; i < n; i++ {
		t.Component[i] = uf.find(i)
	}
	return t, nil
}

func fromEdgeDTOs(edges []edgeDTO) []Edge {
	out := make([]Edge, len(edges))
	for i, e := range edges {
		out[i] = Edge(e)
	}
	return out
}
