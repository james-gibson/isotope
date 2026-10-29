package isotope

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// PublicSeedCluster is the conventional *name* of the lab's public base seed.
//
// It is only a name. No key is embedded for it, on purpose: a cluster launched
// offline must have exactly the same trust as one launched online, and a root
// baked into the binary would make "trusted by seed-42" a property of the build
// rather than of what the operator pinned. The key travels with the seed — in
// seed.json, or in the alarm config a cluster generates — and verification never
// reaches the network for it (docs/trust-roots.md).
const PublicSeedCluster = "seed-42"

// ParseRoots turns a name→base64 configuration into the roots a verifier pins.
//
// There is no implicit root. An empty map pins nothing, so nothing is attested;
// naming seed-42 is not enough, its key has to be there. That is what makes an
// offline cluster trust exactly what an online one does: the difference between
// them is connectivity, never authority.
func ParseRoots(configured map[string]string) (map[string]ed25519.PublicKey, error) {
	roots := make(map[string]ed25519.PublicKey, len(configured))
	for name, encoded := range configured {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("root with an empty name")
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil {
			return nil, fmt.Errorf("root %q: public key is not base64: %w", name, err)
		}
		if len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("root %q: public key is %d bytes, want %d", name, len(raw), ed25519.PublicKeySize)
		}
		roots[name] = ed25519.PublicKey(raw)
	}
	return roots, nil
}
