package isotope

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// keypair is a test's cluster or root identity.
func keypair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

func b64(raw []byte) string { return base64.StdEncoding.EncodeToString(raw) }

// privateRoot builds a self-signed cluster certificate: the private-cluster case,
// where the root *is* the cluster and its key is pinned on the hosts that trust
// it.
func privateRoot(t *testing.T, cluster string) (Certificate, ed25519.PrivateKey, string) {
	t.Helper()
	pub, priv := keypair(t)
	cert, err := SignCertificate(Certificate{
		Cluster:   cluster,
		PublicKey: b64(pub),
		Issuer:    cluster,
	}, priv)
	if err != nil {
		t.Fatalf("sign self-signed certificate: %v", err)
	}
	return cert, priv, b64(pub)
}

// publicClusterCert signs a cluster certificate under a test root that stands in
// for seed-42 (the embedded key is empty until the operator provisions it).
func publicClusterCert(t *testing.T, rootName string, rootKey ed25519.PrivateKey, cluster string, now time.Time) (Certificate, ed25519.PrivateKey) {
	t.Helper()
	pub, priv := keypair(t)
	cert, err := SignCertificate(Certificate{
		Cluster:   cluster,
		PublicKey: b64(pub),
		Issuer:    rootName,
		NotAfter:  now.Add(365 * 24 * time.Hour).Unix(),
	}, rootKey)
	if err != nil {
		t.Fatalf("sign certificate: %v", err)
	}
	return cert, priv
}

// TestPrivateClusterTrustsPublicCluster is the default edge the lab asked for: a
// private trust set resolves the public root, so a public cluster's members are
// accepted there — and the reverse is not implied.
func TestPrivateClusterTrustsPublicCluster(t *testing.T) {
	rootPub, rootPriv := keypair(t)
	now := time.Unix(1_800_000_000, 0)
	cert, clusterKey := publicClusterCert(t, PublicSeedCluster, rootPriv, "base-public", now)
	endpoint := "http://127.0.0.1:61000/mcp"

	receipt := SignReceiptEd25519(clusterKey, cert, endpoint, now)

	// A private cluster: it pins its own root, and the public root resolves by
	// default. Here the public root is this test's root.
	keyring := Keyring{Roots: map[string]ed25519.PublicKey{
		PublicSeedCluster: rootPub,
	}}

	if err := receipt.VerifyWith(keyring, endpoint, now, 0); err != nil {
		t.Fatalf("a private trust set rejected a public cluster's receipt: %v", err)
	}

	// The asymmetry: a trust set that does not resolve the public root does not
	// accept it. Nothing about the receipt changed — only who is pinned.
	closed := Keyring{Roots: map[string]ed25519.PublicKey{}}
	if err := receipt.VerifyWith(closed, endpoint, now, 0); err == nil {
		t.Fatal("a trust set with no roots accepted a certificate")
	} else if !properties(err, "not a pinned root") {
		t.Fatalf("error %q does not explain the missing root", err)
	}
}

// properties reports whether err mentions want, for readable assertions.
func properties(err error, want string) bool {
	return err != nil && strings.Contains(err.Error(), want)
}

// TestSelfSignedPrivateClusterNeedsItsKeyPinned: a cluster that is its own root is
// valid exactly where someone pinned it, and nowhere else.
func TestSelfSignedPrivateClusterNeedsItsKeyPinned(t *testing.T) {
	cert, key, pub := privateRoot(t, "home")
	now := time.Unix(1_800_000_000, 0)
	receipt := SignReceiptEd25519(key, cert, "http://127.0.0.1:61001/mcp", now)

	pinned := Keyring{Roots: map[string]ed25519.PublicKey{"home": mustDecode(t, pub)}}
	if err := receipt.VerifyWith(pinned, "http://127.0.0.1:61001/mcp", now, 0); err != nil {
		t.Fatalf("pinned private root rejected its own cluster: %v", err)
	}

	unpinned := Keyring{Roots: map[string]ed25519.PublicKey{}}
	if err := receipt.VerifyWith(unpinned, "http://127.0.0.1:61001/mcp", now, 0); err == nil {
		t.Fatal("an unpinned self-signed cluster was accepted")
	} else if !properties(err, "self-signed") {
		t.Fatalf("error %q does not explain the self-signed case", err)
	}
}

func mustDecode(t *testing.T, s string) ed25519.PublicKey {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	return ed25519.PublicKey(raw)
}

// TestCertificateScopeAndExpiry bounds what a certificate may be replayed for.
func TestCertificateScopeAndExpiry(t *testing.T) {
	rootPub, rootPriv := keypair(t)
	now := time.Unix(1_800_000_000, 0)
	roots := map[string]ed25519.PublicKey{"lab": rootPub}

	t.Run("registry-only certificate cannot attest", func(t *testing.T) {
		pub, _ := keypair(t)
		cert, err := SignCertificate(Certificate{
			Cluster: "lab-dir", PublicKey: b64(pub), Issuer: "lab", Scope: []string{ScopeRegistry},
		}, rootPriv)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if _, err := cert.VerifyCertificate(roots, now); err == nil {
			t.Fatal("a registry-scoped certificate was accepted as an attestation grant")
		} else if !properties(err, "does not grant") {
			t.Fatalf("error %q does not name the missing scope", err)
		}
	})

	t.Run("expired certificate", func(t *testing.T) {
		pub, _ := keypair(t)
		cert, err := SignCertificate(Certificate{
			Cluster: "lab-old", PublicKey: b64(pub), Issuer: "lab",
			NotAfter: now.Add(-time.Hour).Unix(),
		}, rootPriv)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if _, err := cert.VerifyCertificate(roots, now); err == nil {
			t.Fatal("an expired certificate was accepted")
		} else if !properties(err, "expired") {
			t.Fatalf("error %q does not say it expired", err)
		}
	})

	t.Run("certificate signed by the wrong root", func(t *testing.T) {
		_, otherRoot := keypair(t)
		pub, _ := keypair(t)
		cert, err := SignCertificate(Certificate{Cluster: "lab-x", PublicKey: b64(pub), Issuer: "lab"}, otherRoot)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		if _, err := cert.VerifyCertificate(roots, now); err == nil {
			t.Fatal("a certificate signed by another key was accepted")
		} else if !properties(err, "not signed by") {
			t.Fatalf("error %q does not name the signature failure", err)
		}
	})
}

// TestReceiptCannotBeReplayedElsewhere: a valid receipt is not a bearer token.
func TestReceiptCannotBeReplayedElsewhere(t *testing.T) {
	cert, key, pub := privateRoot(t, "home")
	now := time.Unix(1_800_000_000, 0)
	keyring := Keyring{Roots: map[string]ed25519.PublicKey{"home": mustDecode(t, pub)}}
	receipt := SignReceiptEd25519(key, cert, "http://127.0.0.1:61002/mcp", now)

	if err := receipt.VerifyWith(keyring, "http://127.0.0.1:61003/mcp", now, 0); err == nil {
		t.Fatal("a receipt verified for a different endpoint")
	}
	if err := receipt.VerifyWith(keyring, "http://127.0.0.1:61002/mcp", now.Add(10*time.Minute), 0); err == nil {
		t.Fatal("a stale receipt was accepted")
	}

	// Swapping in another certificate must not carry the signature with it: the
	// receipt is checked against the key the certificate names, not the one that
	// produced it.
	otherCert, otherKey, _ := privateRoot(t, "home")
	if err := receipt.VerifyWith(Keyring{Roots: map[string]ed25519.PublicKey{"home": mustDecode(t, b64(otherCertPub(t, otherKey)))}}, "http://127.0.0.1:61002/mcp", now, 0); err != nil {
		// The other key is not pinned, so this is expected to fail; the assertion
		// that matters is below.
		_ = err
	}
	swapped := receipt
	swapped.Cert = &otherCert
	if err := swapped.VerifyWith(keyring, "http://127.0.0.1:61002/mcp", now, 0); err == nil {
		t.Fatal("a receipt verified under a certificate it was not signed by")
	}
	if err := swapped.VerifyWith(Keyring{Roots: map[string]ed25519.PublicKey{"home": mustDecode(t, b64(otherCertPub(t, otherKey)))}}, "http://127.0.0.1:61002/mcp", now, 0); err == nil {
		t.Fatal("a receipt verified under another cluster's certificate and key")
	}
}

// otherCertPub returns the public half of a cluster key generated alongside a
// certificate, so a test can pin it.
func otherCertPub(t *testing.T, key ed25519.PrivateKey) []byte {
	t.Helper()
	return key.Public().(ed25519.PublicKey)
}

// TestDerivedSubkeyScopesAChild is the symmetric half: a parent hands a child a
// derived secret, the child registers as itself, and it cannot mint as the parent.
func TestDerivedSubkeyScopesAChild(t *testing.T) {
	root := "parent-secret"
	childKey := DeriveClusterSecret(root, "home-kb")
	otherKey := DeriveClusterSecret(root, "home-ci")

	if childKey == root || childKey == otherKey {
		t.Fatal("derivation returned the parent's secret or collided with a sibling")
	}
	// Deterministic: the parent can recompute it without storing anything.
	if again := DeriveClusterSecret(root, "home-kb"); again != childKey {
		t.Fatal("derivation is not deterministic")
	}

	now := time.Unix(1_800_000_000, 0)
	endpoint := "http://127.0.0.1:61004/mcp"
	childReceipt := SignAttestation(childKey, "home-kb", endpoint, now)

	keyring := Keyring{Secrets: map[string]string{"home-kb": childKey}}
	if err := childReceipt.VerifyWith(keyring, endpoint, now, 0); err != nil {
		t.Fatalf("the parent rejected its child's receipt: %v", err)
	}
	// The child cannot speak as the parent: the parent's name needs the parent's
	// secret, which it was never given.
	asParent := SignAttestation(childKey, "home", endpoint, now)
	parentKeyring := Keyring{Secrets: map[string]string{"home": root}}
	if err := asParent.VerifyWith(parentKeyring, endpoint, now, 0); err == nil {
		t.Fatal("a child minted a receipt as its parent")
	}
}

// TestParseRootsValidatesAndAddsThePublicSeed: a root entry is a key, not a name
// to be trusted hopefully.
func TestParseRootsValidatesAndAddsThePublicSeed(t *testing.T) {
	pub, _ := keypair(t)
	roots, err := ParseRoots(map[string]string{"home": b64(pub)})
	if err != nil {
		t.Fatalf("parse roots: %v", err)
	}
	if _, ok := roots["home"]; !ok {
		t.Fatal("a configured root is missing")
	}
	if PublicSeedKey() != nil {
		if _, ok := roots[PublicSeedCluster]; !ok {
			t.Fatal("the public seed is not resolved by default")
		}
	}

	for _, bad := range []map[string]string{
		{"": b64(pub)},
		{"home": "not base64!"},
		{"home": b64([]byte("short"))},
	} {
		if _, err := ParseRoots(bad); err == nil {
			t.Fatalf("ParseRoots accepted %v", bad)
		}
	}
}

// TestPublicSeedKeyIsEmptyUntilProvisioned: an empty key must never mean "trust
// everything", and the constant ships empty.
func TestPublicSeedKeyIsEmptyUntilProvisioned(t *testing.T) {
	key := PublicSeedKey()
	if key == nil {
		// Unprovisioned: no public root is resolved, and the private-root case
		// still works.
		roots, err := ParseRoots(nil)
		if err != nil {
			t.Fatalf("parse empty roots: %v", err)
		}
		if _, ok := roots[PublicSeedCluster]; ok {
			t.Fatal("an empty public key was resolved as a root")
		}
		return
	}
	if len(key) != ed25519.PublicKeySize {
		t.Fatalf("public seed key has length %d", len(key))
	}
}

// TestEd25519RegistrationReachesCertification drives the whole chain through
// VerifyRegistration: certificate to a pinned root, receipt, registry, mDNS.
func TestEd25519RegistrationReachesCertification(t *testing.T) {
	rootPub, rootPriv := keypair(t)
	now := time.Unix(1_800_000_000, 0)
	cert, clusterKey := publicClusterCert(t, PublicSeedCluster, rootPriv, testCluster, now)
	endpoint := "http://127.0.0.1:61005/mcp"

	registrySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]map[string]string{
			testCluster: {"adhd_mcp": endpoint},
		})
	}))
	defer registrySrv.Close()

	cfg := TrustConfig{
		Roots:       map[string]string{PublicSeedCluster: b64(rootPub)},
		RegistryURL: registrySrv.URL + "/cluster",
		Browser:     staticBrowser{ads: []Advertisement{localAd(testCluster, 19100)}},
		Now:         fixedClock(now),
	}
	receipt := SignReceiptEd25519(clusterKey, cert, endpoint, now)

	verdict := VerifyRegistration(context.Background(), cfg, Registration{Endpoint: endpoint, Attestation: &receipt})
	if verdict.Rung != 6 {
		t.Fatalf("rung = %d, want 6 (evidence: %v)", verdict.Rung, verdict.Reasons)
	}
	if !strings.Contains(strings.Join(verdict.Reasons, "; "), SchemeEd25519) {
		t.Fatalf("evidence does not name the scheme: %v", verdict.Reasons)
	}

	// Without the pinned root the same registration is just a self-declared claim.
	closed := cfg
	closed.Roots = map[string]string{}
	if got := VerifyRegistration(context.Background(), closed, Registration{Endpoint: endpoint, Attestation: &receipt}).Rung; got != 2 {
		t.Fatalf("rung without a root = %d, want 2", got)
	}
}
