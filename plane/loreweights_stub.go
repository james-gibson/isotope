//go:build !loreweights

package plane

// loreWeightsJSON is empty in a default build, so EmbeddedModel falls back to
// the generic anchor-42i hypothesis model packed into every binary.
//
// Build with `-tags loreweights` (after decrypting the corpus weights) to
// populate it; see scripts/ship-plane.sh.
var loreWeightsJSON []byte
