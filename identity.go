package isotope

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// InstanceIdentity holds the public isotope identifier and the private
// receipt-signing secret for a single ADHD (or proxy) instance.
// Generated once at process startup and retained for its lifetime.
type InstanceIdentity struct {
	// Isotope is the public identifier, safe to expose via the MCP API.
	// It is derived from the startup nonce but reveals nothing about secret.
	Isotope string

	secret string // never exported; used only for ComputeReceipt / VerifyReceipt
}

// NewInstanceIdentity generates a fresh identity from a cryptographically
// random 32-byte startup nonce. Call once at process startup.
func NewInstanceIdentity() InstanceIdentity {
	nonce := make([]byte, 32)
	_, _ = rand.Read(nonce) // failure is extremely unlikely; zero bytes still give domain-separated values
	seed := fmt.Sprintf("%x", nonce)
	return InstanceIdentity{
		Isotope: deriveIsotope(seed),
		secret:  deriveIdentitySecret(seed),
	}
}

// ComputeReceipt produces the receipt for a rung validation challenge.
//
//	receipt = base64url(SHA256(secret ":" featureID ":" nonce))
//
// Only an instance that knows its own secret can produce the correct receipt
// for a fresh nonce. SHA256 preimage resistance makes random guessing require
// ~2^256 attempts — effectively infeasible.
func (id InstanceIdentity) ComputeReceipt(featureID, nonce string) string {
	h := sha256.Sum256([]byte(id.secret + ":" + featureID + ":" + nonce))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// VerifyReceipt returns true when receipt matches what this instance would
// produce for the given featureID and nonce. Used by adhd.rung.verify to
// validate receipts submitted by a challenger.
func (id InstanceIdentity) VerifyReceipt(receipt, featureID, nonce string) bool {
	return receipt == id.ComputeReceipt(featureID, nonce)
}

func deriveIsotope(seed string) string {
	h := sha256.Sum256([]byte("isotope:" + seed))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func deriveIdentitySecret(seed string) string {
	h := sha256.Sum256([]byte("secret:" + seed))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
