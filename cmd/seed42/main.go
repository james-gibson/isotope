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
	case "public":
		publicKey(os.Args[2:])
	case "agent":
		runAgent(os.Args[2:])
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
  seed42 public               print the issuer public key (the half to pin)
  seed42 agent -in SPEC       unlock once, then sign while the YubiKey is away

  -signer env                 $SEED42_PRIVATE_KEY, base64 (default)
  -signer file:PATH           a 0600 file holding the same base64 key
  -signer age:PATH            decrypt with age (a YubiKey plugin identity)
  -signer sops:PATH           decrypt with sops
  -signer agent:PATH          use a key a running "seed42 agent" unlocked

The authority is the device that can decrypt the issuer key; the issuer signs.
"agent" holds the unlocked key in memory only, so the device and the touch event
are needed once, not per signature.

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
	fmt.Println("private → the operator store (or an age/SOPS file a YubiKey unlocks), never a repo")
}

func certify(args []string) {
	fs := flag.NewFlagSet("certify", flag.ExitOnError)
	cluster := fs.String("cluster", "", "cluster identity to certify")
	clusterKey := fs.String("cluster-key", "", "the cluster's public key (base64); generated when empty")
	scope := fs.String("scope", isotope.ScopeAttest, "comma-separated scopes: attest, registry")
	days := fs.Int("days", 90, "certificate lifetime in days; 0 means no expiry")
	signerSpec := fs.String("signer", "env", "where the issuer key lives: env, file:PATH, age:PATH, sops:PATH or agent:PATH")
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

	s, err := resolveSigner(*signerSpec)
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
	signed, err := s.SignCertificate(cert)
	if err != nil {
		fail("sign certificate: %v", err)
	}

	fmt.Printf("cluster   %s\n", signed.Cluster)
	fmt.Printf("issuer    %s\n", signed.Issuer)
	fmt.Printf("public    %s\n", signed.PublicKey)
	fmt.Printf("expires   %s\n", expiryString(signed.NotAfter))
	fmt.Printf("cert      %s\n", base64.StdEncoding.EncodeToString(certJSON(signed)))
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
