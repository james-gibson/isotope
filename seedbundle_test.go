package isotope

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

const bundleRoot = "seed-42"

func bundleKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	return pub, priv
}

// TestSeedBundleRoundTrips: a root signs the bytes that travel, and anyone
// holding its pinned public key checks them with no network and no decryption.
func TestSeedBundleRoundTrips(t *testing.T) {
	pub, priv := bundleKeys(t)
	body := []byte(`{"roots":{"seed-42":"<key>"},"encrypted":true}`)
	now := time.Unix(1798761600, 0)

	sig, err := SignSeedBundle(priv, bundleRoot, body, now)
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}
	if sig.Issuer != bundleRoot || sig.CreatedAt != now.Unix() {
		t.Fatalf("signature = %+v, want issuer %q at %d", sig, bundleRoot, now.Unix())
	}
	if sig.Digest == "" {
		t.Fatal("signature carries no digest")
	}
	if err := sig.Verify(pub, body); err != nil {
		t.Fatalf("a fresh signature does not verify: %v", err)
	}
}

// TestSeedBundleRejectsATamperedBody: the ciphertext is the signed artifact, so
// changing one byte of it must fail before any age identity is involved.
func TestSeedBundleRejectsATamperedBody(t *testing.T) {
	pub, priv := bundleKeys(t)
	body := []byte("ENC[AES256_GCM,data:AAAA]")
	sig, err := SignSeedBundle(priv, bundleRoot, body, time.Now())
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}
	tampered := []byte("ENC[AES256_GCM,data:AAAB]")
	if err := sig.Verify(pub, tampered); err == nil {
		t.Fatal("a tampered bundle verified")
	}
}

// TestSeedBundleRejectsAnotherRoot: publishing a key is not trusting it. A
// bundle signed by a stranger's key does not verify against the pinned root.
func TestSeedBundleRejectsAnotherRoot(t *testing.T) {
	pub, _ := bundleKeys(t)
	_, otherPriv := bundleKeys(t)
	body := []byte("seed")
	sig, err := SignSeedBundle(otherPriv, bundleRoot, body, time.Now())
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}
	if err := sig.Verify(pub, body); err == nil {
		t.Fatal("a bundle signed by another key verified against the pinned root")
	}
}

// TestSeedBundleIssuerIsSigned: the issuer name is a name the operator pins, so
// editing it must invalidate the signature rather than relabel the artifact.
func TestSeedBundleIssuerIsSigned(t *testing.T) {
	pub, priv := bundleKeys(t)
	body := []byte("seed")
	sig, err := SignSeedBundle(priv, bundleRoot, body, time.Now())
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}
	sig.Issuer = "someone-else"
	if err := sig.Verify(pub, body); err == nil {
		t.Fatal("a renamed issuer still verified")
	}
}

// TestSeedBundleIsNotACertificate is the domain separation: bytes signed as a
// bundle must not verify as a certificate, or a root's signature over a seed
// could be replayed as a grant.
func TestSeedBundleIsNotACertificate(t *testing.T) {
	pub, priv := bundleKeys(t)
	cert, clusterKey := publicClusterCert(t, PublicSeedCluster, priv, "base-1", time.Now())
	_ = clusterKey

	sig, err := SignSeedBundle(priv, PublicSeedCluster, cert.marshal(), time.Now())
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}

	// Reusing the bundle signature as a certificate signature must not verify.
	forged := cert
	forged.Signature = sig.Signature
	if _, err := forged.VerifyCertificate(map[string]ed25519.PublicKey{PublicSeedCluster: pub}, time.Now()); err == nil {
		t.Fatal("a seed-bundle signature verified as a certificate")
	}
}

// TestSeedBundleNeedsASignerAndBody keeps the empty cases from becoming a
// signature over nothing, which would verify for anyone who wanted it to.
func TestSeedBundleNeedsASignerAndBody(t *testing.T) {
	_, priv := bundleKeys(t)
	if _, err := SignSeedBundle(priv, "", []byte("seed"), time.Now()); err == nil {
		t.Error("signed a bundle with no issuer")
	}
	if _, err := SignSeedBundle(priv, bundleRoot, nil, time.Now()); err == nil {
		t.Error("signed an empty bundle")
	}
	if _, err := SignSeedBundle(nil, bundleRoot, []byte("seed"), time.Now()); err == nil {
		t.Error("signed a bundle with no signer")
	}
	if err := (BundleSignature{}).Verify(priv.Public().(ed25519.PublicKey), []byte("seed")); err == nil {
		t.Error("an empty signature verified")
	}
}

// TestBundleSignatureIsBase64 keeps the wire shape parseable by a JSON reader.
func TestBundleSignatureIsBase64(t *testing.T) {
	_, priv := bundleKeys(t)
	sig, err := SignSeedBundle(priv, bundleRoot, []byte("seed"), time.Now())
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		t.Fatalf("signature is not base64: %v", err)
	}
	if len(raw) != ed25519.SignatureSize {
		t.Fatalf("signature is %d bytes, want %d", len(raw), ed25519.SignatureSize)
	}
}
