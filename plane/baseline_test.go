package plane

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"sort"
	"strings"
	"testing"
)

// This is the baseline gate for the plane's first embedding: the text encoder
// over the labeled observation corpus exported by adhd's spec tooling.
//
// The corpus carries four kinds — request, path, skill, knock — each with a
// numeric label.target. Every kind is measured separately and has its own prior
// values, so a later embedding has to beat each one rather than hide a
// regression behind a better-performing kind.
//
//	UPDATE_BASELINE=1 go test ./plane -run TestPlaneBaseline
//
// Regenerating the corpus (in the adhd repo):
//
//	go run ./cmd/speccorpus -out ../isotope/plane/testdata/spec-corpus.jsonl

const (
	baselineCorpusPath = "testdata/spec-corpus.jsonl"
	baselinePath       = "testdata/baseline.json"
	baselineEncoding   = "observation-text-v1"
	baselineHoldout    = 0.2
	baselineTarget     = "target"
)

type seedObservation struct {
	ID    string             `json:"id"`
	Kind  string             `json:"kind"`
	Text  string             `json:"text"`
	Label map[string]float64 `json:"label"`
}

type baselineMetrics struct {
	Accuracy        float64 `json:"accuracy"`
	Macro           float64 `json:"macro"`
	MAE             float64 `json:"mae"`
	Spearman        float64 `json:"spearman"`
	NearestAccuracy float64 `json:"nearest_accuracy"`
	NearestMacro    float64 `json:"nearest_macro"`
}

type baselineAnchor struct {
	Level      int     `json:"level"`
	N          int     `json:"n"`
	Zeta       []Cplx  `json:"zeta"`
	Dispersion float64 `json:"dispersion"`
}

type baselineKind struct {
	Target  string           `json:"target"`
	Metrics baselineMetrics  `json:"metrics"`
	Anchors []baselineAnchor `json:"anchors"`
}

type baselineDoc struct {
	Version      int                     `json:"version"`
	CorpusSHA256 string                  `json:"corpus_sha256"`
	ModelSHA256  string                  `json:"model_sha256"`
	Encoding     string                  `json:"encoding"`
	Holdout      float64                 `json:"holdout"`
	Kinds        map[string]baselineKind `json:"kinds"`
}

func TestPlaneBaseline(t *testing.T) {
	raw, err := os.ReadFile(baselineCorpusPath)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	observations := parseSeedCorpus(t, raw)
	if len(observations) < 100 {
		t.Fatalf("corpus has %d observations; refusing to gate on it", len(observations))
	}

	model, err := EmbeddedGenericModel()
	if err != nil {
		t.Fatalf("embedded generic model: %v", err)
	}
	got := measureSeed(model, observations)
	if len(got) == 0 {
		t.Fatal("no kind in the corpus carried a label.target")
	}
	doc := baselineDoc{
		Version:      1,
		CorpusSHA256: sha256Hex(raw),
		ModelSHA256:  modelIdentity(model),
		Encoding:     baselineEncoding,
		Holdout:      baselineHoldout,
		Kinds:        got,
	}

	prior, priorErr := loadBaseline()
	update := os.Getenv("UPDATE_BASELINE") != ""

	switch {
	case priorErr != nil:
		if !update {
			t.Fatalf("no baseline yet (%v); run with UPDATE_BASELINE=1 to record the prior values", priorErr)
		}
		writeBaseline(t, doc)
		t.Logf("recorded first baseline: %s", formatKinds(doc.Kinds))
		return
	case update:
		if doc.CorpusSHA256 != prior.CorpusSHA256 || doc.ModelSHA256 != prior.ModelSHA256 {
			writeBaseline(t, doc)
			t.Logf("baseline reset for new corpus/model: %s", formatKinds(doc.Kinds))
			return
		}
		var all []string
		for kind, priorKind := range prior.Kinds {
			gotKind, ok := doc.Kinds[kind]
			if !ok {
				t.Fatalf("refusing to update: kind %q disappeared", kind)
			}
			all = append(all, compareMetrics(priorKind.Metrics, gotKind.Metrics)...)
			all = append(all, compareAnchors(priorKind.Anchors, gotKind.Anchors)...)
		}
		if len(all) > 0 {
			t.Fatalf("refusing to update baseline, prior values are better: %v", all)
		}
		if sameKinds(prior.Kinds, doc.Kinds) {
			t.Log("no metric improved; baseline left unchanged")
			return
		}
		writeBaseline(t, doc)
		t.Logf("baseline improved: %s", formatKinds(doc.Kinds))
		return
	}

	if doc.CorpusSHA256 != prior.CorpusSHA256 {
		t.Fatalf("corpus changed: prior %s, last run %s\nregenerate the baseline deliberately (UPDATE_BASELINE=1) after reviewing the corpus diff",
			prior.CorpusSHA256[:12], doc.CorpusSHA256[:12])
	}
	if doc.ModelSHA256 != prior.ModelSHA256 {
		t.Fatalf("model changed: prior %s, last run %s\nthis is a new embedding; it must be compared and then recorded deliberately",
			prior.ModelSHA256[:12], doc.ModelSHA256[:12])
	}
	if len(doc.Kinds) != len(prior.Kinds) {
		t.Fatalf("kind set changed: prior %d, last run %d; a new kind needs a recorded baseline", len(prior.Kinds), len(doc.Kinds))
	}

	for kind, priorKind := range prior.Kinds {
		gotKind, ok := doc.Kinds[kind]
		if !ok {
			t.Fatalf("kind %q disappeared from the corpus", kind)
		}
		if regressions := compareMetrics(priorKind.Metrics, gotKind.Metrics); len(regressions) > 0 {
			t.Errorf("kind %q is worse than the prior values (%s):\n  %v",
				kind, formatMetrics(priorKind.Metrics), regressions)
		}
		if regressions := compareAnchors(priorKind.Anchors, gotKind.Anchors); len(regressions) > 0 {
			t.Errorf("kind %q anchors moved from the prior values:\n  %v", kind, regressions)
		}
	}
	t.Logf("baseline holds: %s", formatKinds(doc.Kinds))
}

func parseSeedCorpus(t *testing.T, raw []byte) []seedObservation {
	t.Helper()
	var out []seedObservation
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for sc.Scan() {
		line++
		if len(bytes.TrimSpace(sc.Bytes())) == 0 {
			continue
		}
		var obs seedObservation
		if err := json.Unmarshal(sc.Bytes(), &obs); err != nil {
			t.Fatalf("corpus line %d: %v", line, err)
		}
		if obs.Kind == "" || obs.Text == "" {
			t.Fatalf("corpus line %d: incomplete observation", line)
		}
		out = append(out, obs)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan corpus: %v", err)
	}
	return out
}

// measureSeed measures each kind independently. A kind without label.target is
// skipped with a log line rather than silently polluted into another target.
func measureSeed(model *Model, observations []seedObservation) map[string]baselineKind {
	groups := map[string][]seedObservation{}
	for _, obs := range observations {
		groups[obs.Kind] = append(groups[obs.Kind], obs)
	}
	out := make(map[string]baselineKind, len(groups))
	for kind, list := range groups {
		if _, ok := list[0].Label[baselineTarget]; !ok {
			continue
		}
		out[kind] = measureKind(model, list)
	}
	return out
}

// measureKind computes the prior-run-style metrics for one kind: a linear
// readout and nearest-centroid classification of label.target, with a
// deterministic 20% holdout keyed by the observation text.
func measureKind(model *Model, observations []seedObservation) baselineKind {
	nplanes := len(model.Space.Planes)
	dim := 2*nplanes + 1

	type point struct {
		zeta   []complex128
		target float64
		class  int
		hold   bool
	}
	points := make([]point, len(observations))
	for i, obs := range observations {
		target := obs.Label[baselineTarget]
		points[i] = point{
			zeta:   model.Node(obs.Text).Zeta,
			target: target,
			class:  int(math.Round(target)),
			hold:   fnv32(obs.Text)%5 == 0,
		}
	}

	sums := map[int][]complex128{}
	counts := map[int]int{}
	var xtx [][]float64
	var xty []float64
	xtx = make([][]float64, dim)
	for i := range xtx {
		xtx[i] = make([]float64, dim)
	}
	xty = make([]float64, dim)

	for _, p := range points {
		if p.hold {
			continue
		}
		sums[p.class] = addComplex(sums[p.class], p.zeta, nplanes)
		counts[p.class]++

		x := flatten(p.zeta, dim)
		for a := 0; a < dim; a++ {
			for b := 0; b < dim; b++ {
				xtx[a][b] += x[a] * x[b]
			}
			xty[a] += x[a] * p.target
		}
	}

	centroid := map[int][]complex128{}
	anchors := make([]baselineAnchor, 0, len(counts))
	for class, sum := range sums {
		c := make([]complex128, nplanes)
		for j := range sum {
			c[j] = sum[j] / complex(float64(counts[class]), 0)
		}
		centroid[class] = c

		var spread float64
		var seen int
		for _, p := range points {
			if p.hold || p.class != class {
				continue
			}
			spread += distance(p.zeta, c)
			seen++
		}
		if seen > 0 {
			spread /= float64(seen)
		}
		anchors = append(anchors, baselineAnchor{
			Level: class, N: counts[class], Zeta: toCplxSlice(c), Dispersion: spread,
		})
	}
	sort.Slice(anchors, func(i, j int) bool { return anchors[i].Level < anchors[j].Level })

	weights, ok := solveLinear(xtx, xty)
	if !ok {
		panic("plane baseline: singular normal equations")
	}

	var m baselineMetrics
	var n, linearCorrect, nearestCorrect int
	var absErr float64
	var predictions, actuals []float64
	perClassTotal := map[int]int{}
	perClassCorrect := map[int]int{}

	for _, p := range points {
		if !p.hold {
			continue
		}
		n++
		perClassTotal[p.class]++

		x := flatten(p.zeta, dim)
		estimate := 0.0
		for a := 0; a < dim; a++ {
			estimate += weights[a] * x[a]
		}
		predictions = append(predictions, estimate)
		actuals = append(actuals, p.target)
		absErr += math.Abs(estimate - p.target)
		if int(math.Round(estimate)) == p.class {
			linearCorrect++
			perClassCorrect[p.class]++
		}

		predicted, _ := nearestCentroid(p.zeta, centroid)
		if predicted == p.class {
			nearestCorrect++
		}
	}

	if n == 0 {
		panic("plane baseline: empty holdout")
	}
	m.Accuracy = float64(linearCorrect) / float64(n)
	m.MAE = absErr / float64(n)
	m.Spearman = spearman(predictions, actuals)
	m.NearestAccuracy = float64(nearestCorrect) / float64(n)
	for class, total := range perClassTotal {
		if total == 0 || counts[class] == 0 {
			continue
		}
		m.Macro += float64(perClassCorrect[class]) / float64(total)
	}
	m.Macro /= float64(len(perClassTotal))

	return baselineKind{Target: baselineTarget, Metrics: m, Anchors: anchors}
}

func flatten(z []complex128, dim int) []float64 {
	out := make([]float64, dim)
	for j := range z {
		out[2*j] = real(z[j])
		out[2*j+1] = imag(z[j])
	}
	out[dim-1] = 1
	return out
}

func toCplxSlice(z []complex128) []Cplx {
	out := make([]Cplx, len(z))
	for i := range z {
		out[i] = toCplx(z[i])
	}
	return out
}

func addComplex(dst, src []complex128, n int) []complex128 {
	if dst == nil {
		dst = make([]complex128, n)
	}
	for j := 0; j < n; j++ {
		dst[j] += src[j]
	}
	return dst
}

func distance(a, b []complex128) float64 {
	var d float64
	for j := range a {
		d += cmplx.Abs(a[j] - b[j])
	}
	return d
}

func nearestCentroid(z []complex128, centroids map[int][]complex128) (int, float64) {
	keys := make([]int, 0, len(centroids))
	for class := range centroids {
		keys = append(keys, class)
	}
	sort.Ints(keys)

	best, bestD := -1, math.Inf(1)
	for _, class := range keys {
		if d := distance(z, centroids[class]); d < bestD {
			best, bestD = class, d
		}
	}
	return best, bestD
}

func solveLinear(a [][]float64, b []float64) ([]float64, bool) {
	n := len(b)
	m := make([][]float64, n)
	for i := range m {
		m[i] = append(append([]float64(nil), a[i]...), b[i])
	}
	for col := 0; col < n; col++ {
		pivot := col
		for r := col + 1; r < n; r++ {
			if math.Abs(m[r][col]) > math.Abs(m[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(m[pivot][col]) < 1e-12 {
			return nil, false
		}
		m[col], m[pivot] = m[pivot], m[col]
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := m[r][col] / m[col][col]
			for c := col; c <= n; c++ {
				m[r][c] -= f * m[col][c]
			}
		}
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = m[i][n] / m[i][i]
	}
	return out, true
}

func spearman(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 3 {
		return 0
	}
	return pearson(ranks(x), ranks(y))
}

func ranks(v []float64) []float64 {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(i, j int) bool { return v[idx[i]] < v[idx[j]] })
	out := make([]float64, len(v))
	for i := 0; i < len(idx); {
		j := i
		for j+1 < len(idx) && v[idx[j+1]] == v[idx[i]] {
			j++
		}
		rank := (float64(i)+float64(j))/2 + 1
		for k := i; k <= j; k++ {
			out[idx[k]] = rank
		}
		i = j + 1
	}
	return out
}

func pearson(x, y []float64) float64 {
	var sx, sy, sxx, syy, sxy float64
	n := float64(len(x))
	for i := range x {
		sx += x[i]
		sy += y[i]
		sxx += x[i] * x[i]
		syy += y[i] * y[i]
		sxy += x[i] * y[i]
	}
	num := sxy - sx*sy/n
	den := math.Sqrt((sxx - sx*sx/n) * (syy - sy*sy/n))
	if den == 0 {
		return 0
	}
	return num / den
}

func fnv32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// modelIdentity pins the model the baseline was measured with: charges, anchor
// and encoder parameters. A different weights file is a different embedding and
// must be compared, not silently swapped in.
func modelIdentity(m *Model) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "charges=%v;anchor=%v;", m.Space.Charges(), m.Space.Anchor)
	fmt.Fprintf(&b, "freq=%v;phase=%v;gain=%v;decay=%v",
		m.Params.Freq, m.Params.Phase, m.Params.Gain, m.Params.Decay)
	return sha256Hex(b.Bytes())
}

func compareMetrics(prior, got baselineMetrics) []string {
	const tolerance = 1e-6
	var regressions []string
	checkAtLeast := func(name string, priorV, gotV float64) {
		if gotV+tolerance < priorV {
			regressions = append(regressions, fmt.Sprintf("%s %.4f < prior %.4f", name, gotV, priorV))
		}
	}
	checkAtMost := func(name string, priorV, gotV float64) {
		if gotV > priorV+tolerance {
			regressions = append(regressions, fmt.Sprintf("%s %.4f > prior %.4f", name, gotV, priorV))
		}
	}
	checkAtLeast("accuracy", prior.Accuracy, got.Accuracy)
	checkAtLeast("macro", prior.Macro, got.Macro)
	checkAtLeast("spearman", prior.Spearman, got.Spearman)
	checkAtLeast("nearest_accuracy", prior.NearestAccuracy, got.NearestAccuracy)
	checkAtLeast("nearest_macro", prior.NearestMacro, got.NearestMacro)
	checkAtMost("mae", prior.MAE, got.MAE)
	return regressions
}

func compareAnchors(prior, got []baselineAnchor) []string {
	const tolerance = 1e-9
	if len(prior) != len(got) {
		return []string{fmt.Sprintf("anchor count %d -> %d", len(prior), len(got))}
	}
	var out []string
	for i := range prior {
		if prior[i].Level != got[i].Level {
			out = append(out, fmt.Sprintf("anchor order changed at %d: %d -> %d", i, prior[i].Level, got[i].Level))
			continue
		}
		if len(prior[i].Zeta) != len(got[i].Zeta) {
			out = append(out, fmt.Sprintf("class %d has %d planes, want %d", prior[i].Level, len(got[i].Zeta), len(prior[i].Zeta)))
			continue
		}
		for j := range prior[i].Zeta {
			dr := math.Abs(prior[i].Zeta[j].Real - got[i].Zeta[j].Real)
			di := math.Abs(prior[i].Zeta[j].Imag - got[i].Zeta[j].Imag)
			if dr > tolerance || di > tolerance {
				out = append(out, fmt.Sprintf("class %d plane %d moved by (%.3g, %.3g)", prior[i].Level, j, dr, di))
				break
			}
		}
	}
	return out
}

func sameKinds(a, b map[string]baselineKind) bool {
	if len(a) != len(b) {
		return false
	}
	for kind, ak := range a {
		bk, ok := b[kind]
		if !ok || len(compareMetrics(ak.Metrics, bk.Metrics)) > 0 || len(compareMetrics(bk.Metrics, ak.Metrics)) > 0 {
			return false
		}
	}
	return true
}

func loadBaseline() (baselineDoc, error) {
	var out baselineDoc
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(data, &out)
	return out, err
}

func writeBaseline(t *testing.T, doc baselineDoc) {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal baseline: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(baselinePath, data, 0o644); err != nil { //nolint:gosec // checked-in test fixture
		t.Fatalf("write baseline: %v", err)
	}
}

func formatMetrics(m baselineMetrics) string {
	return fmt.Sprintf("acc %.1f%% macro %.1f%% mae %.3f spearman %.3f nearest %.1f%%/%.1f%%",
		m.Accuracy*100, m.Macro*100, m.MAE, m.Spearman, m.NearestAccuracy*100, m.NearestMacro*100)
}

func formatKinds(kinds map[string]baselineKind) string {
	names := make([]string, 0, len(kinds))
	for kind := range kinds {
		names = append(names, kind)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, kind := range names {
		parts = append(parts, fmt.Sprintf("%s[%s]", kind, formatMetrics(kinds[kind].Metrics)))
	}
	return strings.Join(parts, " ")
}
