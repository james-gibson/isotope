// Command seed42 provisions the public base seed's keypair.
//
// The private half is the lab's public root of trust: it certifies public
// clusters. It must never be committed, printed into a log, or pasted into a file
// a repository can see. The public half is not embedded in isotope either — it is
// pinned by the seeds and cluster configs that should accept public clusters, so
// an offline cluster trusts exactly what an online one does.
//
//	seed42 init            # generate a keypair; prints both halves once
//	seed42 certify ...     # sign a cluster certificate with the seed key
//
// init reads the private key from $SEED42_PRIVATE_KEY when it is already
// provisioned, so a machine can certify a new cluster without ever printing it.
package main

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/james-gibson/isotope"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "init":
		initKey()
	case "certify":
		certify(os.Args[2:])
	case "sign":
		signBundle(os.Args[2:])
	case "verify":
		verifyBundle(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `seed42 — the public base seed (seed-42)

  seed42 init                 generate a keypair (prints both halves once)
  seed42 certify -cluster ID  sign a cluster certificate with the seed key
  seed42 sign -in FILE        sign a seed bundle (a SOPS/age ciphertext is fine)
  seed42 verify -in FILE      verify a bundle's detached signature offline

  -signer env                 $SEED42_PRIVATE_KEY, base64 (default)
  -signer file:PATH           a 0600 file holding the same base64 key
  (a hardware slot is the same crypto.Signer seam; pkcs11: is not built yet)

After init, put the public half in the seeds and cluster configs that should accept
public clusters (seed.json roots), and the private half in the operator's store.
Nothing is embedded in isotope: an offline cluster has to trust the same set an
online one does. The private half is never committed.
`)
	os.Exit(2)
}

func initKey() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fail("generate key: %v", err)
	}
	fmt.Printf("public  %s\n", base64.StdEncoding.EncodeToString(pub))
	fmt.Printf("private %s\n", base64.StdEncoding.EncodeToString(priv))
	fmt.Println("\npublic  → the seeds and cluster configs that should accept public clusters")
	fmt.Println("private → SEED42_PRIVATE_KEY (keychain/secret store), never a repository")
}

func certify(args []string) {
	fs := flag.NewFlagSet("certify", flag.ExitOnError)
	cluster := fs.String("cluster", "", "cluster identity to certify")
	clusterKey := fs.String("cluster-key", "", "the cluster's public key (base64); generated when empty")
	scope := fs.String("scope", isotope.ScopeAttest, "comma-separated scopes: attest, registry")
	days := fs.Int("days", 90, "certificate lifetime in days; 0 means no expiry")
	signerSpec := fs.String("signer", "env", "where the root key lives: env (default) or file:PATH")
	_ = fs.Parse(args)

	if strings.TrimSpace(*cluster) == "" {
		fail("-cluster is required")
	}

	var pubBase64 string
	if *clusterKey != "" {
		pubBase64 = *clusterKey
	} else {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fail("generate cluster key: %v", err)
		}
		pubBase64 = base64.StdEncoding.EncodeToString(pub)
		fmt.Fprintf(os.Stderr, "generated a cluster key; the private half is NOT printed here — provision it with the cluster\n")
	}

	signer, err := resolveSigner(*signerSpec)
	if err != nil {
		fail("%v", err)
	}

	cert := isotope.Certificate{
		Cluster:   *cluster,
		PublicKey: pubBase64,
		Issuer:    isotope.PublicSeedCluster,
		Scope:     splitScopes(*scope),
	}
	if *days > 0 {
		cert.NotAfter = time.Now().AddDate(0, 0, *days).Unix()
	}
	signed, err := isotope.SignCertificateWith(cert, signer)
	if err != nil {
		fail("sign certificate: %v", err)
	}

	fmt.Printf("cluster   %s\n", signed.Cluster)
	fmt.Printf("issuer    %s\n", signed.Issuer)
	fmt.Printf("public    %s\n", signed.PublicKey)
	fmt.Printf("expires   %s\n", expiryString(signed.NotAfter))
	fmt.Printf("cert      %s\n", base64.StdEncoding.EncodeToString(certJSON(signed)))
}

// seedPrivateKey reads the seed's private key from the environment. It is never a
// file in a repository and never a flag, so it does not land in shell history.
func seedPrivateKey() (ed25519.PrivateKey, error) {
	return decodePrivateKey("$SEED42_PRIVATE_KEY", os.Getenv("SEED42_PRIVATE_KEY"))
}

// decodePrivateKey reads an ed25519 private key as base64, accepting the 64-byte
// private key or its 32-byte seed — the two shapes an operator's key store and a
// key file respectively hold.
func decodePrivateKey(what, encoded string) (ed25519.PrivateKey, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, fmt.Errorf("%s is not set; provision it from the keychain or a secret store", what)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s is not base64: %w", what, err)
	}
	switch len(raw) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	default:
		return nil, fmt.Errorf("%s is %d bytes, want %d (private key) or %d (seed)", what, len(raw), ed25519.PrivateKeySize, ed25519.SeedSize)
	}
}

// resolveSigner turns -signer into the crypto.Signer that signs certificates and
// seed bundles.
//
// The flag exists because the root key should not have to exist on the host:
// env and file are the two shapes that do, and a hardware slot (a YubiKey PIV
// Ed25519 key, a TPM) is the same interface with the private half never leaving
// the device. Nothing in the signed bytes changes between them, so a verifier
// cannot tell which one signed.
func resolveSigner(spec string) (crypto.Signer, error) {
	spec = strings.TrimSpace(spec)
	switch {
	case spec == "" || spec == "env":
		return seedPrivateKey()
	case strings.HasPrefix(spec, "file:"):
		path := strings.TrimSpace(strings.TrimPrefix(spec, "file:"))
		if path == "" {
			return nil, fmt.Errorf("-signer file: needs a path")
		}
		data, err := os.ReadFile(path) //nolint:gosec // the operator names the key file
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		return decodePrivateKey(path, string(data))
	default:
		return nil, fmt.Errorf("unknown signer %q: want env or file:PATH (a pkcs11: signer needs a device and is not built yet)", spec)
	}
}

// signBundle writes a detached signature for the bytes that travel. Those bytes
// may be a SOPS/age ciphertext: the signature covers what is distributed, so a
// host that can verify it needs neither the decryption key nor the network.
func signBundle(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	in := fs.String("in", "", "file to sign (plaintext or ciphertext)")
	out := fs.String("out", "", "signature output (default: <in>.sig)")
	issuer := fs.String("issuer", isotope.PublicSeedCluster, "root name the verifiers pin")
	signerSpec := fs.String("signer", "env", "where the root key lives: env (default) or file:PATH")
	_ = fs.Parse(args)

	if strings.TrimSpace(*in) == "" {
		fail("-in is required")
	}
	body, err := os.ReadFile(*in) //nolint:gosec // the operator names the bundle
	if err != nil {
		fail("read %s: %v", *in, err)
	}
	signer, err := resolveSigner(*signerSpec)
	if err != nil {
		fail("%v", err)
	}
	sig, err := isotope.SignSeedBundle(signer, *issuer, body, time.Now())
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
	if err := sig.Verify(pub, body); err != nil {
		fail("%s", err)
	}
	fmt.Printf("ok: %s signed %s at %s\n", sig.Issuer, *in, time.Unix(sig.CreatedAt, 0).UTC().Format(time.RFC3339))
}

// parsePublicKey reads the pinned root key: base64 inline, or a file holding
// base64 or an OpenSSH ed25519 line (what a GitHub keys file or `seed42 init`'s
// public half look like).
func parsePublicKey(spec string) (ed25519.PublicKey, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("-pub is required: a signature is only as good as the key you pinned")
	}
	body := spec
	if strings.HasPrefix(spec, "@") {
		data, err := os.ReadFile(strings.TrimSpace(strings.TrimPrefix(spec, "@")))
		if err != nil {
			return nil, fmt.Errorf("read public key: %w", err)
		}
		body = strings.TrimSpace(string(data))
	}
	if strings.HasPrefix(body, "ssh-ed25519 ") {
		key, err := isotope.ParseSSHEd25519(body)
		if err != nil {
			return nil, err
		}
		return key.PublicKey, nil
	}
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("public key is neither base64 nor an ssh-ed25519 line: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key is %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

func splitScopes(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func expiryString(unix int64) string {
	if unix == 0 {
		return "never"
	}
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}

func certJSON(c isotope.Certificate) []byte {
	// The certificate is printed as its own JSON so a verifier can pin it without
	// re-deriving the field order.
	body, err := json.Marshal(c)
	if err != nil {
		fail("marshal certificate: %v", err)
	}
	return body
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seed42: "+format+"\n", args...)
	os.Exit(1)
}
