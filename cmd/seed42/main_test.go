package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

	s, err := resolveSigner("file:" + path)
	if err != nil {
		t.Fatalf("resolveSigner: %v", err)
	}
	if !s.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("the file signer does not hold the key written to it")
	}

	// A 32-byte seed is the other shape an operator's store produces.
	s, err = resolveSigner("file:" + writeKey(t, priv.Seed()))
	if err != nil {
		t.Fatalf("resolveSigner(seed): %v", err)
	}
	if !s.Public().(ed25519.PublicKey).Equal(pub) {
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
	s, err := resolveSigner("file:" + writeKey(t, priv))
	if err != nil {
		t.Fatalf("resolveSigner: %v", err)
	}

	body := []byte("ENC[AES256_GCM,data:seed,iv:00]")
	sig, err := s.SignBundle(isotope.PublicSeedCluster, body, time.Now())
	if err != nil {
		t.Fatalf("SignBundle: %v", err)
	}
	if verifyErr := sig.Verify(pub, body); verifyErr != nil {
		t.Fatalf("bundle signature does not verify: %v", verifyErr)
	}

	cert, err := s.SignCertificate(isotope.Certificate{
		Cluster:   "base-1",
		PublicKey: base64.StdEncoding.EncodeToString(pub),
		Issuer:    isotope.PublicSeedCluster,
		Scope:     []string{isotope.ScopeAttest},
	})
	if err != nil {
		t.Fatalf("SignCertificate: %v", err)
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

// TestDecryptorsUnlockTheSameKey: `age:` and `sops:` are the shapes that make
// the YubiKey the authority — the issuer key never rests unencrypted, and only a
// touch on the plugin's PIV slot decrypts it. The tools are faked here so the
// wiring (arg shape, stdout parsing, base64/dotenv decoding) is what is tested.
func TestDecryptorsUnlockTheSameKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	keyBody := base64.StdEncoding.EncodeToString(priv)

	dir := t.TempDir()
	keyFile := filepath.Join(dir, "issuer.key")
	if err := os.WriteFile(keyFile, []byte("SEED42_PRIVATE_KEY="+keyBody+"\n"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	// A fake decryptor that behaves like `age -d FILE`: print the key.
	for _, tool := range []string{"age", "sops"} {
		script := "#!/bin/sh\ncat " + keyFile + "\n"
		path := filepath.Join(dir, tool)
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatalf("write %s: %v", tool, err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	for _, spec := range []string{"age:" + keyFile, "sops:" + keyFile} {
		s, err := resolveSigner(spec)
		if err != nil {
			t.Fatalf("resolveSigner(%s): %v", spec, err)
		}
		if !s.Public().(ed25519.PublicKey).Equal(pub) {
			t.Fatalf("%s unlocked a different key", spec)
		}
		body := []byte("ENC[AES256_GCM,data:seed,iv:00]")
		sig, err := s.SignBundle(isotope.PublicSeedCluster, body, time.Now())
		if err != nil {
			t.Fatalf("%s SignBundle: %v", spec, err)
		}
		if verifyErr := sig.Verify(pub, body); verifyErr != nil {
			t.Fatalf("%s signature does not verify: %v", spec, verifyErr)
		}
	}
}

// TestDecodeKeyMaterialForms covers what a decrypted document looks like in
// practice: raw base64, a dotenv line, or a YAML field.
func TestDecodeKeyMaterialForms(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(priv)
	for name, body := range map[string]string{
		"raw":    b64,
		"dotenv": "OTHER=x\nSEED42_PRIVATE_KEY=" + b64 + "\n",
		"yaml":   "private_key: " + b64 + "\n",
	} {
		key, err := decodeKeyMaterial(name, []byte(body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !key.Public().(ed25519.PublicKey).Equal(priv.Public()) {
			t.Fatalf("%s decoded a different key", name)
		}
	}
}

// TestAgentSignsOnlyItsTwoOperations is the boundary: the agent is the unlocked
// authority, so the socket must not offer raw signing to whoever can reach it.
func TestAgentSignsOnlyItsTwoOperations(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	socket := startAgent(t, localSigner{key: priv})

	client := &agentSigner{socket: socket}
	if got, ok := client.Public().(ed25519.PublicKey); !ok || !got.Equal(pub) {
		t.Fatal("the agent did not report the unlocked key")
	}

	body := []byte("ENC[AES256_GCM,data:seed,iv:00]")
	sig, err := client.SignBundle(isotope.PublicSeedCluster, body, time.Now())
	if err != nil {
		t.Fatalf("SignBundle: %v", err)
	}
	if verifyErr := sig.Verify(pub, body); verifyErr != nil {
		t.Fatalf("agent signature does not verify: %v", verifyErr)
	}

	cert, err := client.SignCertificate(isotope.Certificate{
		Cluster:   "base-1",
		PublicKey: base64.StdEncoding.EncodeToString(pub),
		Issuer:    isotope.PublicSeedCluster,
		Scope:     []string{isotope.ScopeAttest},
	})
	if err != nil {
		t.Fatalf("SignCertificate: %v", err)
	}
	if _, err := cert.VerifyCertificate(map[string]ed25519.PublicKey{isotope.PublicSeedCluster: pub}, time.Now()); err != nil {
		t.Fatalf("agent certificate does not verify: %v", err)
	}

	// Raw signing is not an operation the socket grants.
	if _, err := client.call(agentRequest{Op: "sign", Body: base64.StdEncoding.EncodeToString([]byte("anything"))}); err == nil {
		t.Fatal("the agent signed arbitrary bytes")
	}
}

// TestAgentSocketIsPrivate: another local user must not be able to ask the
// unlocked authority for signatures.
func TestAgentSocketIsPrivate(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	socket := startAgent(t, localSigner{key: priv})

	info, err := os.Stat(socket)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("socket mode = %o, want 600", got)
	}
}

// startAgent serves a signer on a short socket path and waits for it to bind.
//
// The path is built under os.TempDir() rather than t.TempDir(): a unix socket
// path is limited to ~104 bytes on darwin, and a test's nested temp directory is
// long enough to exceed it.
func startAgent(t *testing.T, s signer) string {
	t.Helper()
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("s42-test-%d.sock", time.Now().UnixNano()))
	t.Cleanup(func() { _ = os.Remove(socket) })

	done := make(chan error, 1)
	go func() { done <- serveAgent(s, socket, 5*time.Second) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil {
			return socket
		}
		select {
		case err := <-done:
			t.Fatalf("agent exited before binding: %v", err)
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("agent socket %s never appeared", socket)
	return ""
}
