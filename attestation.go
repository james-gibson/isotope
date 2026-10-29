package isotope

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Attestation is the cluster-scoped evidence a registrant presents alongside its
// endpoint: a receipt that the cluster's operator handed this process the
// cluster's launch secret.
//
// It exists because presence is not evidence. Before #23, `VerifyTrust` granted
// rung 6 to any endpoint that appeared in a registry anyone could write and
// behind any `_lezz-demo._tcp` advertisement anyone could publish — so a local
// process could claim a rung for an endpoint that did not exist. The receipt
// cannot be produced without the secret, and the secret never crosses the wire:
// only the MAC does.
//
// The MAC binds the cluster *and* the endpoint, so a leaked receipt cannot be
// replayed for a different endpoint, and IssuedAt bounds it in time.
type Attestation struct {
	// Cluster is the cluster identity the receipt was issued for. It is also the
	// registry key the endpoint is looked up under: one cluster, one grant.
	Cluster string `json:"cluster"`
	// IssuedAt is when the receipt was signed, in Unix seconds.
	IssuedAt int64 `json:"issued_at"`
	// MAC is base64url(HMAC-SHA256(secret, domain || cluster || endpoint || issuedAt)).
	MAC string `json:"mac"`
}

// AttestationDomain separates this MAC from any other use of the same secret.
const AttestationDomain = "isotope-attestation-v1"

// ErrAttestation is the family of receipt failures. Every one of them means the
// same thing to a caller: this process has not proved it belongs to the cluster
// it claims.
var ErrAttestation = errors.New("attestation rejected")

// SignAttestation produces the receipt a process of cluster presents for
// endpoint. The operator's secret comes from the cluster's launch: for a lezz
// demo it is minted when the cluster starts and handed to the smoke-alarm it
// starts and to the agents it spawns.
//
// The endpoint string is signed verbatim, so the verifier must be given the
// same string it will verify against — the registration's Endpoint field.
func SignAttestation(secret, cluster, endpoint string, now time.Time) Attestation {
	issued := now.Unix()
	return Attestation{
		Cluster:  cluster,
		IssuedAt: issued,
		MAC:      attestationMAC(secret, cluster, endpoint, issued),
	}
}

// Verify checks the receipt against secret for endpoint, accepting IssuedAt
// within skew of now. An empty secret never verifies: an alarm with no launch
// secret cannot attest anyone.
func (a Attestation) Verify(secret, endpoint string, now time.Time, skew time.Duration) error {
	if secret == "" {
		return fmt.Errorf("%w: this alarm has no cluster secret configured", ErrAttestation)
	}
	if a.Cluster == "" {
		return fmt.Errorf("%w: no cluster named", ErrAttestation)
	}
	if a.MAC == "" {
		return fmt.Errorf("%w: no mac", ErrAttestation)
	}
	if skew <= 0 {
		skew = DefaultAttestationSkew
	}
	if delta := now.Sub(time.Unix(a.IssuedAt, 0)); delta > skew || delta < -skew {
		return fmt.Errorf("%w: issued %s away from now (skew allowed %s)", ErrAttestation, delta.Round(time.Second), skew)
	}
	want := attestationMAC(secret, a.Cluster, endpoint, a.IssuedAt)
	// Constant time: the MAC is a secret-dependent value, and comparing it with
	// == leaks its prefix through timing.
	if !hmac.Equal([]byte(want), []byte(a.MAC)) {
		return fmt.Errorf("%w: mac does not match cluster %q and endpoint %q", ErrAttestation, a.Cluster, endpoint)
	}
	return nil
}

// DefaultAttestationSkew is how far a receipt's clock may differ from the
// verifier's. A demo runs both sides on one host, so this only covers a
// container whose clock is a little behind.
const DefaultAttestationSkew = 2 * time.Minute

// attestationMAC is the receipt's MAC. Fields are newline-separated so no value
// can be shifted into its neighbor by embedding a separator.
func attestationMAC(secret, cluster, endpoint string, issuedAt int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	// Hash writes cannot fail on sha256; the error is part of io.Writer.
	_, _ = mac.Write([]byte(strings.Join([]string{
		AttestationDomain,
		cluster,
		endpoint,
		strconv.FormatInt(issuedAt, 10),
	}, "\n")))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
