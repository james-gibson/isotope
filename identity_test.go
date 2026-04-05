package isotope

import (
	"strings"
	"testing"
)

func TestNewInstanceIdentityIsUnique(t *testing.T) {
	a := NewInstanceIdentity()
	b := NewInstanceIdentity()
	if a.Isotope == b.Isotope {
		t.Error("two independent identities must not share an isotope")
	}
}

func TestInstanceIdentityIsotopeIsBase64(t *testing.T) {
	id := NewInstanceIdentity()
	// base64url: no +, no /, no = padding
	if strings.ContainsAny(id.Isotope, "+/=") {
		t.Errorf("isotope should be base64url-encoded, got %q", id.Isotope)
	}
	if id.Isotope == "" {
		t.Error("isotope must not be empty")
	}
}

func TestComputeReceiptIsNonceBound(t *testing.T) {
	id := NewInstanceIdentity()
	r1 := id.ComputeReceipt("adhd/lights-status", "nonce-a")
	r2 := id.ComputeReceipt("adhd/lights-status", "nonce-b")
	if r1 == r2 {
		t.Error("receipts for different nonces must differ")
	}
}

func TestVerifyReceiptRoundTrip(t *testing.T) {
	id := NewInstanceIdentity()
	receipt := id.ComputeReceipt("adhd/lights-status", "nonce-x")
	if !id.VerifyReceipt(receipt, "adhd/lights-status", "nonce-x") {
		t.Error("VerifyReceipt must accept its own ComputeReceipt output")
	}
}

func TestVerifyReceiptRejectsReplay(t *testing.T) {
	id := NewInstanceIdentity()
	receipt := id.ComputeReceipt("adhd/lights-status", "nonce-1")
	if id.VerifyReceipt(receipt, "adhd/lights-status", "nonce-2") {
		t.Error("VerifyReceipt must reject a receipt bound to a different nonce")
	}
}

func TestVerifyReceiptRejectsRandomString(t *testing.T) {
	id := NewInstanceIdentity()
	if id.VerifyReceipt("random-garbage", "adhd/lights-status", "nonce-x") {
		t.Error("VerifyReceipt must reject a random receipt")
	}
}

func TestIsopeAndSecretAreDifferent(t *testing.T) {
	// Two calls to derive* with the same seed must yield different values
	// (domain separator ensures isotope != secret)
	a := deriveIsotope("seed-test")
	b := deriveIdentitySecret("seed-test")
	if a == b {
		t.Error("isotope and secret derived from same seed must differ")
	}
}
