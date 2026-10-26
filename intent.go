package isotope

// The intent layer (#21).
//
// The name isotope routes: it identifies a promise's *wording*, and it is
// therefore language-bound — the step prose is hashed as written, so a
// translation is a new isotope. That is the right behavior for a route, and
// the wrong layer to compare tests on.
//
// The intent layer is the comparison layer. A caller declares a golden,
// unlocalized intent and the wording that realizes it in each language; the
// canon then resolves a spec's prose to those declared intents. Two localized
// specs whose intents match produce the same *intent* isotope even though their
// names differ.
//
// The canon does not guess. A wording that is not declared resolves to nothing
// and is reported as unresolved; no similarity score, no stemming, no
// translation are invented here. The declaration is the caller's, and it is
// only as good as the intents it lists.
//
// Because each realization is declared per intent, the relation between two
// locales is constant by construction: the same rotation holds for every test
// it covers. A locale that realizes only some of the intents has a partial
// rotation, reported by Coverage.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Intent is one golden, unlocalized promise and the wordings that realize it.
//
// Key is language-free and stable — it is what the intent isotope hashes, so
// changing it changes every intent isotope that contains it. Anchor is the
// golden wording the intent is written against; Locales maps a language tag to
// the wording that carries the same promise in that language.
type Intent struct {
	Key     string            `json:"key"`
	Anchor  string            `json:"anchor"`
	Locales map[string]string `json:"locales,omitempty"`
}

// IntentIndex resolves step and name prose to declared intents. The zero value
// is not usable; build one with NewIntentIndex.
type IntentIndex struct {
	intents []Intent
	byText  map[string]int // normalizeStepText(wording) -> index into intents
}

// ErrAmbiguousWording is returned when one wording is declared for two intents:
// the canon cannot tell which promise the author meant, so it refuses.
var ErrAmbiguousWording = errors.New("isotope: wording declared for more than one intent")

// NewIntentIndex validates and indexes a set of declared intents.
//
// A key must be non-empty, must not contain ':' or a newline (the token stream
// is colon-delimited, newline-separated), and must be unique. A wording — the
// anchor or any locale's realization — may not normalize to the same text as
// another intent's wording.
func NewIntentIndex(intents ...Intent) (*IntentIndex, error) {
	ix := &IntentIndex{
		intents: make([]Intent, 0, len(intents)),
		byText:  make(map[string]int, len(intents)),
	}
	keys := make(map[string]bool, len(intents))

	for _, in := range intents {
		if strings.TrimSpace(in.Key) == "" {
			return nil, errors.New("isotope: intent with empty key")
		}
		if strings.ContainsAny(in.Key, ":\n\r") {
			return nil, fmt.Errorf("isotope: intent key %q contains ':' or a newline", in.Key)
		}
		if keys[in.Key] {
			return nil, fmt.Errorf("isotope: duplicate intent key %q", in.Key)
		}
		keys[in.Key] = true

		if strings.TrimSpace(in.Anchor) == "" {
			return nil, fmt.Errorf("isotope: intent %q has empty anchor", in.Key)
		}

		locales := make(map[string]string, len(in.Locales))
		wording := map[string]string{"anchor": in.Anchor}
		for locale, text := range in.Locales {
			if strings.TrimSpace(locale) == "" {
				return nil, fmt.Errorf("isotope: intent %q has a realization under an empty locale", in.Key)
			}
			if locale == "anchor" {
				return nil, fmt.Errorf("isotope: intent %q declares a locale named \"anchor\", which is reserved for the anchor wording", in.Key)
			}
			if strings.TrimSpace(text) == "" {
				return nil, fmt.Errorf("isotope: intent %q has an empty %s realization", in.Key, locale)
			}
			locales[locale] = text
			wording[locale] = text
		}
		in.Locales = locales

		// Index after the intent is appended, so index == position.
		ix.intents = append(ix.intents, in)
		idx := len(ix.intents) - 1

		for label, text := range wording {
			norm := normalizeStepText(text)
			if norm == "" {
				return nil, fmt.Errorf("isotope: intent %q %s wording normalizes to nothing", in.Key, label)
			}
			if prev, seen := ix.byText[norm]; seen && prev != idx {
				return nil, fmt.Errorf("%w: %q is intent %q and intent %q",
					ErrAmbiguousWording, text, ix.intents[prev].Key, in.Key)
			}
			ix.byText[norm] = idx
		}
	}

	return ix, nil
}

// Len returns the number of declared intents.
func (ix *IntentIndex) Len() int {
	if ix == nil {
		return 0
	}
	return len(ix.intents)
}

// Intents returns a copy of the declared intents, ordered by declaration.
func (ix *IntentIndex) Intents() []Intent {
	if ix == nil {
		return nil
	}
	out := make([]Intent, len(ix.intents))
	copy(out, ix.intents)
	return out
}

// Resolve reports the declared intent a wording realizes, if any. Matching uses
// the same normalization the canon hashes, so resolution agrees with the name
// isotope: two wordings that hash alike resolve alike.
func (ix *IntentIndex) Resolve(text string) (Intent, bool) {
	if ix == nil {
		return Intent{}, false
	}
	idx, ok := ix.byText[normalizeStepText(text)]
	if !ok {
		return Intent{}, false
	}
	return ix.intents[idx], true
}

// Coverage reports how many declared intents have a realization in locale. A
// locale that covers every intent rotates the whole index; a locale that covers
// some of it rotates only the intents it names, and every other step in that
// language will be reported unresolved.
func (ix *IntentIndex) Coverage(locale string) (covered, total int) {
	if ix == nil {
		return 0, 0
	}
	for _, in := range ix.intents {
		if _, ok := in.Locales[locale]; ok {
			covered++
		}
	}
	return covered, len(ix.intents)
}

// Locales returns the language tags declared anywhere in the index, sorted.
func (ix *IntentIndex) Locales() []string {
	if ix == nil {
		return nil
	}
	seen := map[string]bool{}
	for _, in := range ix.intents {
		for locale := range in.Locales {
			seen[locale] = true
		}
	}
	out := make([]string, 0, len(seen))
	for locale := range seen {
		out = append(out, locale)
	}
	sort.Strings(out)
	return out
}

// IntentToken is one token of the intent stream. It has the same shape as a
// canonical token, so an intent stream is signed the same way.
//
// Value is the intent hash when Intent is set, and the normalized wording when
// the wording is not declared.
type IntentToken struct {
	Type   string // "FEATURE", "SCENARIO", or a canonical keyword
	Key    string // "name" or "hash"
	Value  string
	Intent string // declared intent key, empty when unresolved
	Text   string // the wording as written
}

// IntentCanonical is a spec canonicalized onto declared intents.
//
// Unresolved lists the wordings the index did not cover, in the order they
// occur. They are kept verbatim in the stream — the canon does not guess — so
// two specs with the same unresolved wording still compare equal.
type IntentCanonical struct {
	Tokens     []IntentToken
	Unresolved []string
}

// String returns the newline-delimited intent token stream.
func (c IntentCanonical) String() string {
	lines := make([]string, len(c.Tokens))
	for i, t := range c.Tokens {
		lines[i] = fmt.Sprintf("%s:%s:%s", t.Type, t.Key, t.Value)
	}
	return strings.Join(lines, "\n")
}

// Isotope signs the intent stream, keyless-nameable in the same way as the name
// isotope: the two streams differ only in what a step contributes.
func (c IntentCanonical) Isotope(signingKey []byte, family string, version int) Isotope {
	return GenerateIsotope(intentTokensToCanonical(c.Tokens), signingKey, family, version)
}

// CanonicalizeIntents canonicalizes the spec onto declared intents. Feature
// names, scenario names, and step prose all resolve through the same index.
//
// With an empty index nothing resolves and the stream is byte-identical to the
// name stream: the intent layer is the name layer plus resolutions, never a
// rewrite of it.
func (ix *IntentIndex) CanonicalizeIntents(ast *GherkinAST) IntentCanonical {
	var out IntentCanonical
	resolve := func(typ, key, text string) IntentToken {
		tok := IntentToken{Type: typ, Key: key, Text: text}
		if in, ok := ix.Resolve(text); ok {
			tok.Intent = in.Key
			tok.Value = hashText("intent:" + in.Key)
			return tok
		}
		out.Unresolved = append(out.Unresolved, text)
		if key == "hash" {
			tok.Value = hashText(normalizeStepText(text))
		} else {
			tok.Value = normalizeStepText(text)
		}
		return tok
	}

	for _, feature := range ast.Features {
		out.Tokens = append(out.Tokens, resolve("FEATURE", "name", feature.Name))
		for _, scenario := range feature.Scenarios {
			out.Tokens = append(out.Tokens, resolve("SCENARIO", "name", scenario.Name))
			for _, step := range scenario.Steps {
				out.Tokens = append(out.Tokens, resolve(string(step.Keyword), "hash", step.Text))
			}
		}
	}
	return out
}

// CanonicalizeGherkinToIntentIsotope is the one-shot form: parse, resolve, sign.
// The returned canonical value reports any wording the index did not cover.
func CanonicalizeGherkinToIntentIsotope(gherkinText string, ix *IntentIndex, signingKey []byte, family string, version int) (Isotope, IntentCanonical, error) {
	ast, err := ParseGherkin(gherkinText)
	if err != nil {
		return Isotope{}, IntentCanonical{}, err
	}
	c := ix.CanonicalizeIntents(ast)
	return c.Isotope(signingKey, family, version), c, nil
}

// IntentComparison is the answer to "is this the same promise, in any
// language?" — and, separately, to whether each side was fully declared.
type IntentComparison struct {
	SameIntent  bool
	UnresolvedA []string
	UnresolvedB []string
}

// CompareIntents compares two intent streams. It is keyless: the signer is not
// part of the comparison, and neither is the family.
func CompareIntents(a, b IntentCanonical) IntentComparison {
	return IntentComparison{
		SameIntent:  a.String() == b.String(),
		UnresolvedA: a.Unresolved,
		UnresolvedB: b.Unresolved,
	}
}

// intentTokensToCanonical renders an intent stream as canonical tokens so the
// existing signer can be used unchanged. The token type vocabulary is shared;
// only the value's provenance differs.
func intentTokensToCanonical(tokens []IntentToken) []CanonicalToken {
	out := make([]CanonicalToken, len(tokens))
	for i, t := range tokens {
		out[i] = CanonicalToken{Type: t.Type, Key: t.Key, Value: t.Value}
	}
	return out
}
