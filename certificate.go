package isotope

import (
	"crypto"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Evidence schemes. A claim about an endpoint can be proved two ways, and the
// scheme travels with the proof so a verifier never guesses.
const (
	// SchemeHMAC is a shared-secret MAC. Verification implies the ability to
	// mint, so it is only usable by the cluster itself or by a verifier the
	// cluster trusts with its secret — including a derived subkey handed to a
	// child (see DeriveClusterSecret).
	SchemeHMAC = "hmac-sha256"
	// SchemeEd25519 is a cluster's signature over its receipt, with the
	// certificate that chains the cluster to a root. Any verifier that pins the
	// root can check it without being able to mint.
	SchemeEd25519 = "ed25519"
)

// Certificate binds a cluster's signing key to an issuer: a trust root
// ("seed-42" for the public base, or a cluster's own name for a private root) or
// the cluster itself (self-signed, valid only where its key is pinned).
//
// The chain is deliberately one hop: root → cluster → receipt. A cluster cannot
// certify another cluster; rung 6 ("can certify others") does not yet have an
// operation behind it, and turning it into a minting capability is a decision
// that belongs in docs/trust-roots.md first.
type Certificate struct {
	Cluster   string   `json:"cluster"`
	PublicKey string   `json:"public_key"` // base64 std, ed25519 public key
	Issuer    string   `json:"issuer"`     // root name, or the cluster itself
	Scope     []string `json:"scope,omitempty"`
	NotAfter  int64    `json:"not_after,omitempty"` // unix seconds; 0 means no expiry
	Signature string   `json:"signature"`           // base64 std
}

// Certificate scopes. Empty means "attest" only, so a certificate minted for the
// directory role cannot be replayed as an attestation grant.
const (
	ScopeAttest   = "attest"
	ScopeRegistry = "registry"
)

// certificateDomain separates a certificate signature from any other use of the
// same key.
const certificateDomain = "isotope-cluster-cert-v1"

// SignCertificate issues a cluster certificate. issuerKey is the private key of
// the issuer named in the certificate: a root's key, or the cluster's own for a
// private root.
//
// The key is material, so it has to be somewhere: use SignCertificateWith when
// the root lives on a device and never leaves it.
func SignCertificate(cert Certificate, issuerKey ed25519.PrivateKey) (Certificate, error) {
	if len(issuerKey) != ed25519.PrivateKeySize {
		return Certificate{}, fmt.Errorf("issuer key has length %d, want %d", len(issuerKey), ed25519.PrivateKeySize)
	}
	return SignCertificateWith(cert, issuerKey)
}

// SignCertificateWith issues a cluster certificate using a signer rather than an
// in-memory private key.
//
// It produces the same signature over the same bytes as SignCertificate, so a
// verifier cannot tell whether the root was a file, a PKCS#11 slot, an SSH agent
// or a TPM; what changes is that a hardware signer never releases the private
// half. crypto.Signer is the seam every one of those speaks, and for Ed25519 the
// signer must sign the message itself (opts is Hash(0)).
func SignCertificateWith(cert Certificate, signer crypto.Signer) (Certificate, error) {
	if cert.Cluster == "" {
		return Certificate{}, errors.New("certificate needs a cluster")
	}
	if cert.Issuer == "" {
		return Certificate{}, errors.New("certificate needs an issuer")
	}
	if signer == nil {
		return Certificate{}, errors.New("certificate needs a signer")
	}
	sig, err := signer.Sign(rand.Reader, cert.marshal(), crypto.Hash(0))
	if err != nil {
		return Certificate{}, fmt.Errorf("sign certificate: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return Certificate{}, fmt.Errorf("signer returned a %d-byte signature, want %d", len(sig), ed25519.SignatureSize)
	}
	cert.Signature = base64.StdEncoding.EncodeToString(sig)
	return cert, nil
}

// marshal is the signed byte string. Fields are length-prefixed through joining
// with a separator that cannot appear in a cluster name or a base64 key.
func (c Certificate) marshal() []byte {
	scope := strings.Join(c.Scope, ",")
	return []byte(strings.Join([]string{
		certificateDomain,
		c.Cluster,
		c.PublicKey,
		c.Issuer,
		scope,
		fmt.Sprintf("%d", c.NotAfter),
	}, "\n"))
}

// VerifyCertificate checks the certificate against the roots a verifier pins, and
// returns the cluster's public key.
//
// A root name that is not pinned is not a failure of the certificate — it is
// simply not evidence here, which is why the error names the missing root rather
// than claiming a forgery.
func (c Certificate) VerifyCertificate(roots map[string]ed25519.PublicKey, now time.Time) (ed25519.PublicKey, error) {
	if c.Cluster == "" || c.PublicKey == "" || c.Issuer == "" {
		return nil, fmt.Errorf("%w: certificate is missing a field", ErrAttestation)
	}
	if c.NotAfter != 0 && now.Unix() > c.NotAfter {
		return nil, fmt.Errorf("%w: certificate for %q expired at %s", ErrAttestation, c.Cluster, time.Unix(c.NotAfter, 0).UTC().Format(time.RFC3339))
	}
	scope := c.Scope
	if len(scope) == 0 {
		scope = []string{ScopeAttest}
	}
	if !hasScope(scope, ScopeAttest) {
		return nil, fmt.Errorf("%w: certificate for %q does not grant %q (scope %v)", ErrAttestation, c.Cluster, ScopeAttest, scope)
	}

	root, pinned := roots[c.Issuer]
	if !pinned {
		if c.Issuer == c.Cluster {
			return nil, fmt.Errorf("%w: %q is self-signed and its key is not pinned", ErrAttestation, c.Cluster)
		}
		return nil, fmt.Errorf("%w: issuer %q is not a pinned root", ErrAttestation, c.Issuer)
	}

	sig, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil {
		return nil, fmt.Errorf("%w: certificate signature is not base64: %w", ErrAttestation, err)
	}
	if !ed25519.Verify(root, c.marshal(), sig) {
		return nil, fmt.Errorf("%w: certificate for %q is not signed by %q", ErrAttestation, c.Cluster, c.Issuer)
	}

	pub, err := base64.StdEncoding.DecodeString(c.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: certificate for %q carries no usable public key", ErrAttestation, c.Cluster)
	}
	return ed25519.PublicKey(pub), nil
}

func hasScope(scope []string, want string) bool {
	for _, s := range scope {
		if s == want {
			return true
		}
	}
	return false
}

// SignReceiptEd25519 signs a receipt for endpoint as cluster, using the cluster's
// private key and carrying the certificate that chains it to its root.
//
// This is the evidence a third party can check: it needs the pinned root, not the
// cluster's secret.
func SignReceiptEd25519(clusterKey ed25519.PrivateKey, cert Certificate, endpoint string, now time.Time) Attestation {
	issued := now.Unix()
	sig := ed25519.Sign(clusterKey, receiptBytes(cert.Cluster, endpoint, issued))
	return Attestation{
		Cluster:   cert.Cluster,
		IssuedAt:  issued,
		Scheme:    SchemeEd25519,
		Signature: base64.StdEncoding.EncodeToString(sig),
		Cert:      &cert,
	}
}

// receiptBytes is the Ed25519-signed byte string.
func receiptBytes(cluster, endpoint string, issuedAt int64) []byte {
	return []byte(strings.Join([]string{
		AttestationDomain,
		SchemeEd25519,
		cluster,
		endpoint,
		fmt.Sprintf("%d", issuedAt),
	}, "\n"))
}

// verifyEd25519Receipt checks the signature and the certificate chain, returning
// the reasons a verifier can show.
func (a Attestation) verifyEd25519(keyring Keyring, endpoint string, now time.Time, skew time.Duration) error {
	if a.Cert == nil {
		return fmt.Errorf("%w: ed25519 receipt carries no certificate", ErrAttestation)
	}
	if a.Cert.Cluster != a.Cluster {
		return fmt.Errorf("%w: receipt names %q but its certificate is for %q", ErrAttestation, a.Cluster, a.Cert.Cluster)
	}
	if a.Signature == "" {
		return fmt.Errorf("%w: ed25519 receipt has no signature", ErrAttestation)
	}
	if skew <= 0 {
		skew = DefaultAttestationSkew
	}
	if delta := now.Sub(time.Unix(a.IssuedAt, 0)); delta > skew || delta < -skew {
		return fmt.Errorf("%w: issued %s away from now (skew allowed %s)", ErrAttestation, delta.Round(time.Second), skew)
	}
	pub, err := a.Cert.VerifyCertificate(keyring.Roots, now)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(a.Signature)
	if err != nil {
		return fmt.Errorf("%w: receipt signature is not base64: %w", ErrAttestation, err)
	}
	if !ed25519.Verify(pub, receiptBytes(a.Cluster, endpoint, a.IssuedAt), sig) {
		return fmt.Errorf("%w: receipt for %q is not signed by its certificate's key", ErrAttestation, a.Cluster)
	}
	return nil
}

// DeriveClusterSecret returns a subkey of a cluster's secret, labeled with the
// child cluster it is handed to.
//
// A parent hands a child this value instead of its own secret: the child then
// registers receipts as itself and cannot mint as the parent, and revoking it is
// dropping one configured entry rather than rotating the parent. Derivation is
// deterministic, so a parent that lost the child's copy can recompute it.
//
// It is not a substitute for a certificate: a third party must be given the
// derived key to check a receipt, so this scopes trust within one administrative
// domain, it does not open it across one.
func DeriveClusterSecret(rootSecret, child string) string {
	mac := hmac.New(sha256.New, []byte(rootSecret))
	_, _ = mac.Write([]byte(subkeyDomain + "\n" + child))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

const subkeyDomain = "isotope-subkey-v1"
