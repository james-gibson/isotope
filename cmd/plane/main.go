// Command plane is the one-shot isotope plane deriver. It emits node vectors,
// edge tuples, and a quarantine decision from a charge assignment, with no
// hyperparameter search.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/james-gibson/isotope/plane"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "plane: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: plane <gen|node|edge|triple|sort> [flags]")
	}
	switch args[0] {
	case "gen":
		return runGen(args[1:])
	case "node":
		return runNode(args[1:], stdout)
	case "edge":
		return runEdge(args[1:], stdout)
	case "triple":
		return runTriple(args[1:], stdout)
	case "sort":
		return runSort(args[1:], stdin, stdout)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func loadModel(weightsPath string) (*plane.Model, error) {
	if weightsPath == "" {
		return plane.EmbeddedModel()
	}
	data, err := os.ReadFile(weightsPath) //nolint:gosec // the CLI flag is the point
	if err != nil {
		return nil, err
	}
	return plane.LoadWeights(data)
}

func addCommonFlags(fs *flag.FlagSet) (anchor, weights *string) {
	anchor = fs.String("anchor", "42", "anchor: 42 (proven consensus) or 42i (working hypothesis)")
	weights = fs.String("weights", "", "path to weights.json (default: embedded)")
	return anchor, weights
}

func runGen(args []string) error {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	charges := fs.String("charges", "", "comma-separated charge list, e.g. 1,-1,-1")
	corpus := fs.String("corpus", "", "path to a corpus file (optional)")
	out := fs.String("out", "weights.json", "output weights.json path")
	anchorStr, _ := addCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *charges == "" {
		return fmt.Errorf("gen: --charges is required")
	}
	chargeList, err := plane.ParseCharges(*charges)
	if err != nil {
		return err
	}
	anchor, err := plane.ParseAnchor(*anchorStr)
	if err != nil {
		return err
	}
	text := ""
	if *corpus != "" {
		corpusBytes, readErr := os.ReadFile(*corpus) //nolint:gosec // the CLI flag is the point
		if readErr != nil {
			return readErr
		}
		text = string(corpusBytes)
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("gen: corpus %q is empty", *corpus)
		}
	}
	model, err := plane.GenerateModel(chargeList, anchor, text)
	if err != nil {
		return err
	}
	data, err := model.MarshalWeights()
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(data, '\n'), 0o600) //nolint:gosec // non-secret build artifact
}

func runNode(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("node", flag.ContinueOnError)
	anchorStr, weights := addCommonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("node: want exactly one string")
	}
	model, err := loadModelForCompare(*anchorStr, *weights)
	if err != nil {
		return err
	}
	node := model.Node(fs.Arg(0))
	key, err := model.Key(node)
	if err != nil {
		return err
	}
	return writeJSON(stdout, nodeOut{Zeta: cplxList(node.Zeta), Key: plane.Cplx{Real: real(key), Imag: imag(key)}})
}

func withAnchor(model *plane.Model, anchor complex128) (*plane.Model, error) {
	if model.Space.Anchor == anchor {
		return model, nil
	}
	return plane.GenerateModel(model.Space.Charges(), anchor, "")
}

func runEdge(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("edge", flag.ContinueOnError)
	anchorStr, weights := addCommonFlags(fs)
	probeStr := fs.String("node", "42", "probe string encoded to select coordinates")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("edge: want two plane selectors")
	}
	model, err := loadModelForCompare(*anchorStr, *weights)
	if err != nil {
		return err
	}
	a, err := model.Space.PlaneIndex(fs.Arg(0))
	if err != nil {
		return err
	}
	b, err := model.Space.PlaneIndex(fs.Arg(1))
	if err != nil {
		return err
	}
	res, err := model.Edge(a, b, model.Node(*probeStr))
	if err != nil {
		return err
	}
	return writeJSON(stdout, res)
}

func runTriple(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("triple", flag.ContinueOnError)
	anchorStr, weights := addCommonFlags(fs)
	probeStr := fs.String("node", "42", "probe string encoded to select coordinates")
	tol := fs.Float64("tol", 1e-9, "frustration tolerance on |holonomy|")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 3 {
		return fmt.Errorf("triple: want three plane selectors")
	}
	model, err := loadModelForCompare(*anchorStr, *weights)
	if err != nil {
		return err
	}
	var idx [3]int
	for i := 0; i < 3; i++ {
		idx[i], err = model.Space.PlaneIndex(fs.Arg(i))
		if err != nil {
			return err
		}
	}
	key, err := model.TripleKeyPlanes(idx[0], idx[1], idx[2], model.Node(*probeStr), *tol)
	if err != nil {
		return err
	}
	return writeJSON(stdout, key)
}

func runSort(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("sort", flag.ContinueOnError)
	anchorStr, weights := addCommonFlags(fs)
	probeStr := fs.String("node", "42", "probe string encoded to select coordinates")
	tol := fs.Float64("tol", 1e-9, "frustration tolerance on |holonomy|")
	if err := fs.Parse(args); err != nil {
		return err
	}
	anchorSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "anchor" {
			anchorSet = true
		}
	})
	if !anchorSet {
		return fmt.Errorf("sort: --anchor is required")
	}
	model, err := loadModelForCompare(*anchorStr, *weights)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdin)
	var selectors []int
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		idx, idxErr := model.Space.PlaneIndex(line)
		if idxErr != nil {
			return idxErr
		}
		selectors = append(selectors, idx)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return scanErr
	}
	res, err := model.SortPlanes(selectors, model.Node(*probeStr), *tol, plane.DefaultQuarantineKey())
	if err != nil {
		return err
	}
	return writeJSON(stdout, res)
}

func loadModelForCompare(anchorStr, weights string) (*plane.Model, error) {
	anchor, err := plane.ParseAnchor(anchorStr)
	if err != nil {
		return nil, err
	}
	model, err := loadModel(weights)
	if err != nil {
		return nil, err
	}
	return withAnchor(model, anchor)
}

type nodeOut struct {
	Zeta []plane.Cplx `json:"zeta"`
	Key  plane.Cplx   `json:"key"`
}

func cplxList(zs []complex128) []plane.Cplx {
	out := make([]plane.Cplx, len(zs))
	for i, z := range zs {
		out[i] = plane.Cplx{Real: real(z), Imag: imag(z)}
	}
	return out
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
