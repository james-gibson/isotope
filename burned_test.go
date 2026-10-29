package isotope

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The burned vector's fingerprint. It is asserted so a regenerated key shows up as
// a diff instead of quietly changing what the committed signature means.
const burnedFingerprint = "SHA256:w3+sxEvESgsmFobmbBlpM0yuslXz0pgS+2TLZOjJDgw"

func burnedPath(name string) string {
	return filepath.Join("testdata", "burned", name)
}

// TestBurnedVectorVerifiesOffline is the proof anyone can run: a committed note, a
// committed signature, a committed public key, no private material and no network.
//
// The key is burned (testdata/burned/README.md), so reproducing this result proves
// the mechanism rather than the identity of whoever signed it.
func TestBurnedVectorVerifiesOffline(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skipf("ssh-keygen verifies the vector: %v", err)
	}

	note, err := os.Open(burnedPath("note.md"))
	if err != nil {
		t.Fatalf("open committed note: %v", err)
	}
	defer func() { _ = note.Close() }()

	verify := exec.Command("ssh-keygen", "-Y", "verify",
		"-f", burnedPath("allowed_signers"),
		"-I", "isotope-burned",
		"-n", "lezz-note",
		"-s", burnedPath("note.md.sig"),
	)
	verify.Stdin = note
	out, err := verify.CombinedOutput()
	if err != nil {
		t.Fatalf("the committed vector does not verify: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), burnedFingerprint) {
		t.Fatalf("verified with a different key than the vector's:\n%s", out)
	}
}

// TestBurnedKeySignsAndThePublishedHalfVerifies: the committed private half really
// is the pair of the published half, so a reader can sign their own note with it
// and check it against the same file.
func TestBurnedKeySignsAndThePublishedHalfVerifies(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skipf("ssh-keygen signs the note: %v", err)
	}

	dir := t.TempDir()
	// ssh-keygen refuses a private key another user could read; git stores 0644.
	key, err := os.ReadFile(burnedPath("id_ed25519"))
	if err != nil {
		t.Fatalf("read burned key: %v", err)
	}
	keyPath := filepath.Join(dir, "id_ed25519")
	if writeErr := os.WriteFile(keyPath, key, 0o600); writeErr != nil {
		t.Fatalf("write burned key: %v", writeErr)
	}

	notePath := filepath.Join(dir, "note.md")
	const body = "a note signed with the burned key\n"
	if writeErr := os.WriteFile(notePath, []byte(body), 0o600); writeErr != nil {
		t.Fatalf("write note: %v", writeErr)
	}
	if out, signErr := exec.Command("ssh-keygen", "-Y", "sign", "-q", "-n", "lezz-note", "-f", keyPath, notePath).CombinedOutput(); signErr != nil {
		t.Fatalf("sign with the burned key: %v\n%s", signErr, out)
	}

	message, err := os.Open(notePath)
	if err != nil {
		t.Fatalf("open note: %v", err)
	}
	defer func() { _ = message.Close() }()
	verify := exec.Command("ssh-keygen", "-Y", "verify",
		"-f", burnedPath("allowed_signers"),
		"-I", "isotope-burned",
		"-n", "lezz-note",
		"-s", notePath+".sig",
	)
	verify.Stdin = message
	if out, verifyErr := verify.CombinedOutput(); verifyErr != nil {
		t.Fatalf("the burned private key does not match the published half: %v\n%s", verifyErr, out)
	}
}

// TestBurnedKeyIsATrustRoot pins the committed public half the way a deployment
// pins a person's GitHub key, and verifies a receipt the burned key signed.
//
// This is the whole model with no network and no CA: `ParseSSHEd25519` reads the
// published line, `ParseRoots` pins it, and a certificate issued by that root
// admits a cluster member.
func TestBurnedKeyIsATrustRoot(t *testing.T) {
	published, err := os.ReadFile(burnedPath("id_ed25519.pub"))
	if err != nil {
		t.Fatalf("read published half: %v", err)
	}
	key, err := ParseSSHEd25519(string(published))
	if err != nil {
		t.Fatalf("parse published half: %v", err)
	}
	const rootName = "testdata:isotope-burned"
	roots, err := ParseRoots(map[string]string{rootName: base64.StdEncoding.EncodeToString(key.PublicKey)})
	if err != nil {
		t.Fatalf("pin the burned key as a root: %v", err)
	}

	// The burned private half, in the OpenSSH shape ssh-keygen wrote it. Parsing it
	// also proves the published half is this key: the two halves must agree on the
	// public key, or the vector is broken.
	raw, err := os.ReadFile(burnedPath("id_ed25519"))
	if err != nil {
		t.Fatalf("read burned private key: %v", err)
	}
	seed := parseOpenSSHEd25519Private(t, raw)
	if !key.PublicKey.Equal(seed.Public().(ed25519.PublicKey)) {
		t.Fatalf("published half (%s) is not the private half's pair", ShortFingerprint(key.PublicKey))
	}

	// A member cluster, certified by the burned root, signing a receipt.
	memberPub, memberPriv := keypair(t)
	cert, err := SignCertificate(Certificate{
		Cluster:   "burned-notes",
		PublicKey: base64.StdEncoding.EncodeToString(memberPub),
		Issuer:    rootName,
		NotAfter:  time.Unix(1_900_000_000, 0).Unix(),
	}, seed)
	if err != nil {
		t.Fatalf("issue a certificate from the burned root: %v", err)
	}
	now := time.Unix(1_800_000_000, 0)
	receipt := SignReceiptEd25519(memberPriv, cert, "http://127.0.0.1:63000/mcp", now)
	if err := receipt.VerifyWith(Keyring{Roots: roots}, "http://127.0.0.1:63000/mcp", now, 0); err != nil {
		t.Fatalf("a cluster rooted at the burned key did not verify: %v", err)
	}
}

// parseOpenSSHEd25519Private reads the raw ed25519 private key out of an OpenSSH
// private key file (the "openssh-key-v1" container), which is what holds the
// burned vector's private half.
//
// It handles the unencrypted, uncommented shape ssh-keygen writes; anything else is
// refused rather than guessed at.
func parseOpenSSHEd25519Private(t *testing.T, raw []byte) ed25519.PrivateKey {
	t.Helper()
	const magic = "-----BEGIN OPENSSH PRIVATE KEY-----"
	text := string(raw)
	if !strings.Contains(text, magic) {
		t.Fatal("burned private key is not an OpenSSH private key")
	}
	body := text[len(magic):]
	if end := strings.Index(body, "-----END OPENSSH PRIVATE KEY-----"); end >= 0 {
		body = body[:end]
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(body), ""))
	if err != nil {
		t.Fatalf("decode burned private key: %v", err)
	}
	if !strings.HasPrefix(string(decoded), "openssh-key-v1\x00") {
		t.Fatal("burned private key is not in the openssh-key-v1 container")
	}
	// Fields: ciphername, kdfname, kdfoptions, nkeys, publickey, privatekey.
	rest := decoded[len("openssh-key-v1\x00"):]
	for _, name := range []string{"ciphername", "kdfname"} {
		value, next, readErr := readSSHString(rest)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		if name == "ciphername" && string(value) != "none" {
			t.Fatalf("burned key is encrypted (%q); the vector must be readable", value)
		}
		rest = next
	}
	if _, next, readErr := readSSHString(rest); readErr != nil { // kdfoptions
		t.Fatalf("read kdfoptions: %v", readErr)
	} else {
		rest = next
	}
	if len(rest) < 4 {
		t.Fatal("truncated private key")
	}
	rest = rest[4:]                                              // nkeys, always 1 here
	if _, next, readErr := readSSHString(rest); readErr != nil { // public key blob
		t.Fatalf("read public blob: %v", readErr)
	} else {
		rest = next
	}
	privateBlock, _, readErr := readSSHString(rest)
	if readErr != nil {
		t.Fatalf("read private block: %v", readErr)
	}
	// The private block repeats the check bytes, then holds keytype + key + comment.
	if len(privateBlock) < 8 {
		t.Fatal("truncated private block")
	}
	blk := privateBlock[8:]
	keyType, trimmed, readErr := readSSHString(blk)
	if readErr != nil {
		t.Fatalf("read key type: %v", readErr)
	}
	blk = trimmed
	if string(keyType) != "ssh-ed25519" {
		t.Fatalf("burned key type is %q", keyType)
	}
	// ed25519 stores the public key and then the 64-byte private key (seed || public).
	if _, next, pubErr := readSSHString(blk); pubErr != nil {
		t.Fatalf("read embedded public key: %v", pubErr)
	} else {
		blk = next
	}
	privateKey, _, readErr := readSSHString(blk)
	if readErr != nil {
		t.Fatalf("read private key material: %v", readErr)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		t.Fatalf("private key is %d bytes, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(privateKey)
}
