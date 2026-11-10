//go:build loreweights

package plane

import _ "embed"

// loreWeightsJSON is the corpus-derived model, decrypted from the committed,
// SOPS-encrypted blob (plane/lore-weights.enc.json) at build time. This file is
// only compiled with `-tags loreweights`; the plaintext plane/lore-weights.json
// is written by scripts/ship-plane.sh and is git-ignored, so the corpus labels
// never enter the public tree.
//
//go:embed lore-weights.json
var loreWeightsJSON []byte
