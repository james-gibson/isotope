package isotope

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sshPubLine renders a key the way ssh-keygen and GitHub publish it.
func sshPubLine(t *testing.T, pub ed25519.PublicKey, comment string) string {
	t.Helper()
	var blob []byte
	for _, field := range [][]byte{[]byte("ssh-ed25519"), pub} {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(field)))
		blob = append(blob, n[:]...)
		blob = append(blob, field...)
	}
	line := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob)
	if comment != "" {
		line += " " + comment
	}
	return line
}

// TestPersonalKeyIsARoot is the short version of the whole model: a note signed by
// a person's published key verifies offline, with no seed, no certificate and no
// network. The same key bytes are what a GitHub .keys file serves, so pinning one
// is a copy-paste, not a conversion.
func TestPersonalKeyIsARoot(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skipf("ssh-keygen is how a personal key signs: %v", err)
	}

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "james-gibson", "-f", keyPath).CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen cannot create a key here: %v\n%s", err, out)
	}

	// The published half, read exactly as https://github.com/<user>.keys serves it.
	published, err := os.ReadFile(keyPath + ".pub") //nolint:gosec // a key this test just made
	if err != nil {
		t.Fatalf("read published key: %v", err)
	}
	keys, err := ParseKeyList(string(published), "github:")
	if err != nil {
		t.Fatalf("parse published key: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("parsed %d keys, want 1", len(keys))
	}
	root := keys[0]
	if root.Name != "github:james-gibson" {
		t.Fatalf("root name = %q", root.Name)
	}
	if !strings.HasPrefix(root.Name, "github:") {
		t.Fatalf("root name = %q, want the publish prefix", root.Name)
	}

	// A personal note, signed by the same key.
	note := filepath.Join(dir, "note.md")
	const body = "today: trusts only what it pinned\n"
	if writeErr := os.WriteFile(note, []byte(body), 0o600); writeErr != nil {
		t.Fatalf("write note: %v", writeErr)
	}
	if out, signErr := exec.Command("ssh-keygen", "-Y", "sign", "-q", "-n", "lezz-note", "-f", keyPath, note).CombinedOutput(); signErr != nil {
		t.Fatalf("sign note: %v\n%s", signErr, out)
	}

	// Verification is offline and uses only the pinned public key.
	// allowed_signers takes principals, then keytype and key. A trailing comment
	// is parsed as key options, not as a comment — which is why the line is built
	// from the two fields rather than from the whole published line.
	keyFields := strings.Fields(sshPubLine(t, root.PublicKey, ""))
	allowed := filepath.Join(dir, "allowed_signers")
	if writeErr := os.WriteFile(allowed, []byte(root.Name+" "+strings.Join(keyFields, " ")+"\n"), 0o600); writeErr != nil {
		t.Fatalf("write allowed signers: %v", writeErr)
	}
	// `-Y verify` reads the *message* on stdin and the signature from -s, so the
	// note is what gets piped, not the signature.
	message, err := os.Open(note) //nolint:gosec // a note this test just wrote
	if err != nil {
		t.Fatalf("open note: %v", err)
	}
	defer func() { _ = message.Close() }()
	verify := exec.Command("ssh-keygen", "-Y", "verify", "-f", allowed, "-I", root.Name, "-n", "lezz-note", "-s", note+".sig")
	verify.Stdin = message
	if out, err := verify.CombinedOutput(); err != nil {
		t.Fatalf("verify note against the pinned key: %v\n%s", err, out)
	}
}

// TestParseSSHEd25519RejectsWhatItCannotUse: a pin that quietly accepts another
// algorithm would verify something the caller never pinned.
func TestParseSSHEd25519RejectsWhatItCannotUse(t *testing.T) {
	pub, _ := keypair(t)
	good := sshPubLine(t, pub, "me")
	if _, err := ParseSSHEd25519(good); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}

	for name, line := range map[string]string{
		"rsa":        "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ== me",
		"empty":      "",
		"one field":  "ssh-ed25519",
		"bad base64": "ssh-ed25519 !!!",
		"short key":  "ssh-ed25519 " + base64.StdEncoding.EncodeToString([]byte("nope")),
	} {
		if _, err := ParseSSHEd25519(line); err == nil {
			t.Errorf("%s: accepted %q", name, line)
		}
	}

	// The blob's own type must agree with the line: editing the prefix to claim
	// ed25519 must not relabel an RSA key.
	var rsaBlob []byte
	var rsaLen [4]byte
	binary.BigEndian.PutUint32(rsaLen[:], uint32(len("ssh-rsa")))
	rsaBlob = append(rsaBlob, rsaLen[:]...)
	rsaBlob = append(rsaBlob, []byte("ssh-rsa")...)
	rsaBlob = append(rsaBlob, 0, 0, 0, 3, 1, 2, 3)
	relabelled := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(rsaBlob)
	if _, err := ParseSSHEd25519(relabelled); err == nil {
		t.Fatal("a blob declaring another type was accepted")
	}
}

// TestParseKeyListSkipsOtherAlgorithms: a real .keys file has an RSA key in it
// too, and that must not make the ed25519 key unusable.
func TestParseKeyListSkipsOtherAlgorithms(t *testing.T) {
	pub, _ := keypair(t)
	body := strings.Join([]string{
		"# a comment",
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQ== rsa-key",
		sshPubLine(t, pub, "laptop"),
		"",
	}, "\n")

	keys, err := ParseKeyList(body, "github:")
	if err != nil {
		t.Fatalf("parse key file: %v", err)
	}
	if len(keys) != 1 || keys[0].Name != "github:laptop" {
		t.Fatalf("keys = %+v, want one named github:laptop", keys)
	}
}

// TestPersonalKeyRootsACluster is the bridge to the lab model: the same key a
// person publishes on GitHub can be a trust root and issue a cluster
// certificate, so "my notes" and "my cluster" chain to one identity the operator
// pinned — and a verifier with only the public key accepts both.
func TestPersonalKeyRootsACluster(t *testing.T) {
	personPub, personPriv := keypair(t)
	line := sshPubLine(t, personPub, "james-gibson")

	key, err := ParseSSHEd25519(line)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	const rootName = "github:james-gibson"
	roots, err := ParseRoots(map[string]string{rootName: base64.StdEncoding.EncodeToString(key.PublicKey)})
	if err != nil {
		t.Fatalf("pin as root: %v", err)
	}
	if got := roots[rootName]; !got.Equal(personPub) {
		t.Fatal("pinned root is not the published key")
	}

	// The person's key issues a certificate for a cluster, and a member of that
	// cluster signs a receipt a holder of *only the public key* can verify.
	clusterPub, clusterPriv := keypair(t)
	cert, err := SignCertificate(Certificate{
		Cluster:   "personal-notes",
		PublicKey: base64.StdEncoding.EncodeToString(clusterPub),
		Issuer:    rootName,
	}, personPriv)
	if err != nil {
		t.Fatalf("sign certificate: %v", err)
	}
	now := time.Unix(1_800_000_000, 0)
	receipt := SignReceiptEd25519(clusterPriv, cert, "http://127.0.0.1:62000/mcp", now)
	if err := receipt.VerifyWith(Keyring{Roots: roots}, "http://127.0.0.1:62000/mcp", now, 0); err != nil {
		t.Fatalf("a cluster rooted at a personal key did not verify: %v", err)
	}
}
