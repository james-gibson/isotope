package isotope

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SeedBundleDomain separates a seed-bundle signature from every other use of a
// root key. A signature is not a message: without a domain, bytes signed here
// could be replayed as a certificate or a receipt.
const SeedBundleDomain = "isotope-seed-bundle-v1"

// BundleSignature is a detached signature over the exact bytes that travel.
//
// A seed is a declaration of what a workspace pins — roots, certificates, and
// the material a cluster needs to come up. It has to work online and offline,
// and on a host that cannot decrypt it: so the signature covers the bytes as
// distributed (which may be a SOPS/age ciphertext), not the plaintext inside.
// Anyone holding the root's public key can then prove the bundle is the one the
// root signed, without the network and without the age identity.
type BundleSignature struct {
	// Issuer is the root that signed, by the name its verifiers pin.
	Issuer string `json:"issuer"`
	// CreatedAt is when the signature was made, in Unix seconds. It is evidence
	// of when, not of freshness: expiry belongs to the certificates inside.
	CreatedAt int64 `json:"created_at"`
	// Digest is "sha256:<hex>" of the signed bytes. It is signed too, so a
	// mismatched body is named as such rather than as a bad signature.
	Digest string `json:"digest"`
	// Signature is base64(std) of the ed25519 signature over the bundle message.
	Signature string `json:"signature"`
}

// SignSeedBundle signs body with the root key.
//
// signer is a crypto.Signer rather than an ed25519.PrivateKey because the root
// key should not have to exist on the host: a PKCS#11 slot, a YubiKey PIV
// Ed25519 key or a TPM can sign the same bytes, and a verifier cannot tell the
// difference.
func SignSeedBundle(signer crypto.Signer, issuer string, body []byte, now time.Time) (BundleSignature, error) {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return BundleSignature{}, errors.New("seed bundle needs an issuer")
	}
	if signer == nil {
		return BundleSignature{}, errors.New("seed bundle needs a signer")
	}
	if len(body) == 0 {
		return BundleSignature{}, errors.New("seed bundle is empty")
	}
	created := now.Unix()
	sig, err := signer.Sign(rand.Reader, bundleMessage(issuer, created, body), crypto.Hash(0))
	if err != nil {
		return BundleSignature{}, fmt.Errorf("sign seed bundle: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return BundleSignature{}, fmt.Errorf("signer returned a %d-byte signature, want %d", len(sig), ed25519.SignatureSize)
	}
	return BundleSignature{
		Issuer:    issuer,
		CreatedAt: created,
		Digest:    bundleDigest(body),
		Signature: base64.StdEncoding.EncodeToString(sig),
	}, nil
}

// Verify checks the signature over body against a pinned root key. It needs no
// network, no registry and no decryption: the bytes are what they are.
func (s BundleSignature) Verify(pub ed25519.PublicKey, body []byte) error {
	if s.Issuer == "" {
		return fmt.Errorf("%w: seed bundle names no issuer", ErrAttestation)
	}
	if len(body) == 0 {
		return fmt.Errorf("%w: seed bundle is empty", ErrAttestation)
	}
	if want := bundleDigest(body); s.Digest != want {
		return fmt.Errorf("%w: seed bundle digest is %s, signature covers %s", ErrAttestation, want, s.Digest)
	}
	sig, err := base64.StdEncoding.DecodeString(s.Signature)
	if err != nil {
		return fmt.Errorf("%w: seed bundle signature is not base64: %w", ErrAttestation, err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: root key has length %d, want %d", ErrAttestation, len(pub), ed25519.PublicKeySize)
	}
	if !ed25519.Verify(pub, bundleMessage(s.Issuer, s.CreatedAt, body), sig) {
		return fmt.Errorf("%w: seed bundle is not signed by the pinned root", ErrAttestation)
	}
	return nil
}

// bundleMessage is the signed byte string. The digest binds the body while the
// issuer and time are signed beside it, so neither can be swapped without
// breaking the signature.
func bundleMessage(issuer string, createdAt int64, body []byte) []byte {
	sum := sha256.Sum256(body)
	return []byte(strings.Join([]string{
		SeedBundleDomain,
		issuer,
		fmt.Sprintf("%d", createdAt),
		hex.EncodeToString(sum[:]),
	}, "\n"))
}

func bundleDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}
