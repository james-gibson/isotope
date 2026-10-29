package isotope

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// PublicSeedCluster is the lab's public base seed. It is the root that certifies
// public clusters, and every trust set resolves it by default — a private cluster
// trusts a public one, while the public one does not thereby trust the private
// one (docs/trust-roots.md).
const PublicSeedCluster = "seed-42"

// publicSeedKey is base64(seed-42's ed25519 public key).
//
// The private half is provisioned once by the operator and never committed; see
// cmd/seed42. Until it is provisioned this is empty, PublicSeedKey returns nil,
// and no public cluster can be certified — an empty key must never be treated as
// "trust everything".
const publicSeedKey = ""

// PublicSeedKey returns seed-42's public key, or nil when it has not been
// provisioned yet.
func PublicSeedKey() ed25519.PublicKey {
	if publicSeedKey == "" {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(publicSeedKey)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil
	}
	return ed25519.PublicKey(raw)
}

// ParseRoots turns a name→base64 configuration into the keyring's roots, adding
// the public seed when it is available.
//
// The public root is added even to a set that pinned private roots: "private
// clusters will also trust public clusters" is the default edge, and the reverse
// needs an explicit entry rather than an inference.
func ParseRoots(configured map[string]string) (map[string]ed25519.PublicKey, error) {
	roots := make(map[string]ed25519.PublicKey, len(configured)+1)
	if key := PublicSeedKey(); key != nil {
		roots[PublicSeedCluster] = key
	}
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
