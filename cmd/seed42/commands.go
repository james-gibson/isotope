package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"crypto/ed25519"

	"github.com/james-gibson/isotope"
)

// signBundle writes a detached signature for the bytes that travel. Those bytes
// may be a SOPS/age ciphertext: the signature covers what is distributed, so a
// host that can verify it needs neither the decryption key nor the network.
func signBundle(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	in := fs.String("in", "", "file to sign (plaintext or ciphertext)")
	out := fs.String("out", "", "signature output (default: <in>.sig)")
	issuer := fs.String("issuer", isotope.PublicSeedCluster, "root name the verifiers pin")
	signerSpec := fs.String("signer", "env", "where the issuer key lives: env, file:PATH, age:PATH, sops:PATH or agent:PATH")
	_ = fs.Parse(args)

	if strings.TrimSpace(*in) == "" {
		fail("-in is required")
	}
	body, err := os.ReadFile(*in) //nolint:gosec // the operator names the bundle
	if err != nil {
		fail("read %s: %v", *in, err)
	}
	s, err := resolveSigner(*signerSpec)
	if err != nil {
		fail("%v", err)
	}
	sig, err := s.SignBundle(*issuer, body, time.Now())
	if err != nil {
		fail("%v", err)
	}
	encoded, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		fail("marshal signature: %v", err)
	}
	encoded = append(encoded, '\n')
	path := *out
	if path == "" {
		path = *in + ".sig"
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil { //nolint:gosec // a signature is public
		fail("write %s: %v", path, err)
	}
	fmt.Fprintf(os.Stderr, "signed %s as %s -> %s\n", *in, *issuer, path)
}

// verifyBundle checks a detached signature against a pinned root key. It is
// deliberately offline: -pub is the key the operator pinned, not one it fetched.
func verifyBundle(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	in := fs.String("in", "", "file that was signed")
	sigPath := fs.String("sig", "", "signature file (default: <in>.sig)")
	pubSpec := fs.String("pub", "", "pinned root public key: base64, @file with base64, or @file with an ssh-ed25519 line")
	_ = fs.Parse(args)

	if strings.TrimSpace(*in) == "" {
		fail("-in is required")
	}
	body, err := os.ReadFile(*in) //nolint:gosec // the operator names the bundle
	if err != nil {
		fail("read %s: %v", *in, err)
	}
	path := *sigPath
	if path == "" {
		path = *in + ".sig"
	}
	raw, err := os.ReadFile(path) //nolint:gosec // the operator names the signature
	if err != nil {
		fail("read %s: %v", path, err)
	}
	var sig isotope.BundleSignature
	if unmarshalErr := json.Unmarshal(raw, &sig); unmarshalErr != nil {
		fail("parse %s: %v", path, unmarshalErr)
	}
	pub, err := parsePublicKey(*pubSpec)
	if err != nil {
		fail("%v", err)
	}
	if verifyErr := sig.Verify(pub, body); verifyErr != nil {
		fail("%s", verifyErr)
	}
	fmt.Printf("ok: %s signed %s at %s\n", sig.Issuer, *in, time.Unix(sig.CreatedAt, 0).UTC().Format(time.RFC3339))
}

// publicKey prints the issuer's public half, which is the value seeds and alarm
// configs pin. It works for a running agent too, so an operator can pin a key
// without unlocking it again.
func publicKey(args []string) {
	fs := flag.NewFlagSet("public", flag.ExitOnError)
	signerSpec := fs.String("signer", "env", "where the issuer key lives: env, file:PATH, age:PATH, sops:PATH or agent:PATH")
	_ = fs.Parse(args)

	s, err := resolveSigner(*signerSpec)
	if err != nil {
		fail("%v", err)
	}
	pub, ok := s.Public().(ed25519.PublicKey)
	if !ok {
		fail("signer has no ed25519 public key")
	}
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
}
