package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/james-gibson/isotope"
)

// writeKey writes a base64 ed25519 key file the way an operator's store holds
// it, and returns its path.
func writeKey(t *testing.T, key []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "seed42.key")
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(key)+"\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return path
}

// TestResolveSignerReadsTheFileShape: the seam a hardware signer plugs into has
// two shapes today, and both must produce the same key.
func TestResolveSignerReadsTheFileShape(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	path := writeKey(t, priv)

	signer, err := resolveSigner("file:" + path)
	if err != nil {
		t.Fatalf("resolveSigner: %v", err)
	}
	if !signer.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("the file signer does not hold the key written to it")
	}

	// A 32-byte seed is the other shape an operator's store produces.
	signer, err = resolveSigner("file:" + writeKey(t, priv.Seed()))
	if err != nil {
		t.Fatalf("resolveSigner(seed): %v", err)
	}
	if !signer.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("the seed did not expand to the same key")
	}
}

func TestResolveSignerRejectsAnUnknownShape(t *testing.T) {
	if _, err := resolveSigner("pkcs11:slot=9c"); err == nil {
		t.Fatal("an unbuilt signer shape was accepted")
	} else if !strings.Contains(err.Error(), "pkcs11") {
		t.Fatalf("error %q does not name the shape", err)
	}
	if _, err := resolveSigner("file:"); err == nil {
		t.Fatal("a file signer with no path was accepted")
	}
}

// TestFileSignerSignsABundleAndANote is the whole point of the seam: the same
// signer produces a certificate and a seed-bundle signature, and each verifies
// against the pinned public half.
func TestFileSignerSignsABundleAndANote(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := resolveSigner("file:" + writeKey(t, priv))
	if err != nil {
		t.Fatalf("resolveSigner: %v", err)
	}

	body := []byte("ENC[AES256_GCM,data:seed,iv:00]")
	sig, err := isotope.SignSeedBundle(signer, isotope.PublicSeedCluster, body, time.Now())
	if err != nil {
		t.Fatalf("SignSeedBundle: %v", err)
	}
	if verifyErr := sig.Verify(pub, body); verifyErr != nil {
		t.Fatalf("bundle signature does not verify: %v", verifyErr)
	}

	cert, err := isotope.SignCertificateWith(isotope.Certificate{
		Cluster:   "base-1",
		PublicKey: base64.StdEncoding.EncodeToString(pub),
		Issuer:    isotope.PublicSeedCluster,
		Scope:     []string{isotope.ScopeAttest},
	}, signer)
	if err != nil {
		t.Fatalf("SignCertificateWith: %v", err)
	}
	if _, err := cert.VerifyCertificate(map[string]ed25519.PublicKey{isotope.PublicSeedCluster: pub}, time.Now()); err != nil {
		t.Fatalf("certificate does not verify: %v", err)
	}
}

// TestParsePublicKeyForms covers what an operator actually pastes: base64, a
// file holding base64, and the ssh-ed25519 line a GitHub keys file publishes.
func TestParsePublicKeyForms(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(pub)

	if got, err := parsePublicKey(b64); err != nil || !got.Equal(pub) {
		t.Fatalf("base64: %v", err)
	}

	path := filepath.Join(t.TempDir(), "seed42.pub")
	if err := os.WriteFile(path, []byte(b64+"\n"), 0o600); err != nil {
		t.Fatalf("write pub: %v", err)
	}
	if got, err := parsePublicKey("@" + path); err != nil || !got.Equal(pub) {
		t.Fatalf("@file: %v", err)
	}

	sshLine := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(sshBlob(t, pub)) + " seed-42"
	if err := os.WriteFile(path, []byte(sshLine+"\n"), 0o600); err != nil {
		t.Fatalf("write ssh pub: %v", err)
	}
	if got, err := parsePublicKey("@" + path); err != nil || !got.Equal(pub) {
		t.Fatalf("ssh line: %v", err)
	}

	if _, err := parsePublicKey(""); err == nil {
		t.Fatal("a signature was verifiable with no pinned key")
	}
}

// sshBlob builds the OpenSSH wire form: string type, string key.
func sshBlob(t *testing.T, pub ed25519.PublicKey) []byte {
	t.Helper()
	kind := []byte("ssh-ed25519")
	blob := make([]byte, 0, 4+len(kind)+4+len(pub))
	blob = append(blob, byte(len(kind)>>24), byte(len(kind)>>16), byte(len(kind)>>8), byte(len(kind)))
	blob = append(blob, kind...)
	blob = append(blob, byte(len(pub)>>24), byte(len(pub)>>16), byte(len(pub)>>8), byte(len(pub)))
	blob = append(blob, pub...)
	return blob
}
