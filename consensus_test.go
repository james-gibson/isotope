package isotope

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
)

// testKeys gives every named alarm a distinct signing key.
func testKeys(alarms ...AlarmID) map[AlarmID][]byte {
	keys := make(map[AlarmID][]byte, len(alarms))
	for _, a := range alarms {
		keys[a] = []byte("test-key-" + string(a))
	}
	return keys
}

// signAll recomputes each report's signature over its own Raw with the alarm's
// key, so a fixture can be fed to the verified path.
func signAll(reports []ConsensusReport, keys map[AlarmID][]byte) []ConsensusReport {
	out := make([]ConsensusReport, len(reports))
	for i, r := range reports {
		h := hmac.New(sha256.New, keys[r.AlarmID])
		h.Write([]byte(r.Isotope.Raw))
		r.Isotope.Signature = hex.EncodeToString(h.Sum(nil))
		out[i] = r
	}
	return out
}

func TestAgreementThreeAlarms(t *testing.T) {
	// All three alarms report same isotope
	isotope := Isotope{
		Family:    "iso-login",
		Version:   1,
		Signature: "abc123def456ghi789",
		Raw:       "FEATURE:name:user-auth\nSCENARIO:name:valid-login\nGIVEN:hash:xyz",
	}

	reports := []ConsensusReport{
		{
			AlarmID:   "alarm-a",
			Isotope:   isotope,
			Signature: isotope.Signature,
			Timestamp: time.Now(),
		},
		{
			AlarmID:   "alarm-b",
			Isotope:   isotope,
			Signature: isotope.Signature,
			Timestamp: time.Now().Add(100 * time.Millisecond),
		},
		{
			AlarmID:   "alarm-c",
			Isotope:   isotope,
			Signature: isotope.Signature,
			Timestamp: time.Now().Add(200 * time.Millisecond),
		},
	}

	keys := testKeys("alarm-a", "alarm-b", "alarm-c")
	status, err := VerifyConsensus(signAll(reports, keys), keys, 2) // Require 2/3 agreement
	if err != nil {
		t.Fatalf("VerifyConsensus: %v", err)
	}

	if !status.ConsensusFormed {
		t.Error("Expected consensus to form with 3/3 alarms agreeing")
	}

	if status.AgreedCount != 3 {
		t.Errorf("Expected 3 alarms to agree, got %d", status.AgreedCount)
	}

	if len(status.Outliers) != 0 {
		t.Errorf("Expected no outliers, got %d", len(status.Outliers))
	}
}

func TestIsotopeDisagreementDetection(t *testing.T) {
	// Alarm C reports different isotope
	isotope1 := Isotope{
		Family:    "iso-login",
		Version:   1,
		Signature: "abc123def456ghi789",
		Raw:       "FEATURE:name:user-auth\nSCENARIO:name:valid-login",
	}

	isotope2 := Isotope{
		Family:    "iso-login",
		Version:   1,
		Signature: "wrongwrongwrongwrong",
		Raw:       "FEATURE:name:user-auth\nSCENARIO:name:invalid-login",
	}

	reports := []ConsensusReport{
		{AlarmID: "alarm-a", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-b", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-c", Isotope: isotope2, Timestamp: time.Now()}, // Disagreer
	}

	keys := testKeys("alarm-a", "alarm-b", "alarm-c")
	status, err := VerifyConsensus(signAll(reports, keys), keys, 2)
	if err != nil {
		t.Fatalf("VerifyConsensus: %v", err)
	}

	if !status.ConsensusFormed {
		t.Error("Consensus should form with 2/3 agreement")
	}

	if status.AgreedCount != 2 {
		t.Errorf("Expected 2 alarms to agree, got %d", status.AgreedCount)
	}

	if len(status.Outliers) != 1 {
		t.Errorf("Expected 1 outlier, got %d", len(status.Outliers))
	}

	if status.Outliers[0] != "alarm-c" {
		t.Errorf("Expected alarm-c to be outlier, got %v", status.Outliers[0])
	}
}

func TestByzantineToleranceFiveAlarms(t *testing.T) {
	// 5 alarms can tolerate F=1 failures
	// If 4 agree and 1 disagrees, consensus forms
	isotope1 := Isotope{
		Family:    "iso-login",
		Version:   1,
		Signature: "sig1",
		Raw:       "canonical-form-1",
	}

	isotope2 := Isotope{
		Family:    "iso-login",
		Version:   1,
		Signature: "sig2",
		Raw:       "canonical-form-2",
	}

	reports := []ConsensusReport{
		{AlarmID: "alarm-a", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-b", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-c", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-d", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-e", Isotope: isotope2, Timestamp: time.Now()}, // Compromised
	}

	keys := testKeys("alarm-a", "alarm-b", "alarm-c", "alarm-d", "alarm-e")
	status, err := VerifyConsensus(signAll(reports, keys), keys, 3) // Require 3/5 quorum
	if err != nil {
		t.Fatalf("VerifyConsensus: %v", err)
	}

	if !status.ConsensusFormed {
		t.Error("Consensus should form with 4/5 alarms agreeing")
	}

	if status.ByzantineCount != 1 {
		t.Errorf("Expected Byzantine tolerance F=1, got F=%d", status.ByzantineCount)
	}

	t.Logf("5-alarm consensus: %d agree, %d disagree, can tolerate F=%d Byzantine failures",
		status.AgreedCount, status.DisagreedCount, status.ByzantineCount)
}

func TestConsensusRequiresQuorum(t *testing.T) {
	isotope := Isotope{
		Family:    "iso-login",
		Version:   1,
		Signature: "sig1",
		Raw:       "canonical",
	}

	// Only 2 alarms, require quorum of 3
	reports := []ConsensusReport{
		{AlarmID: "alarm-a", Isotope: isotope, Timestamp: time.Now()},
		{AlarmID: "alarm-b", Isotope: isotope, Timestamp: time.Now()},
	}

	keys := testKeys("alarm-a", "alarm-b")
	status, err := VerifyConsensus(signAll(reports, keys), keys, 3) // Require 3 alarms
	if err != nil {
		t.Fatalf("VerifyConsensus: %v", err)
	}

	if status.ConsensusFormed {
		t.Error("Consensus should not form with only 2 alarms (quorum=3)")
	}

	if status.AgreedCount != 2 {
		t.Errorf("Expected 2 alarms, got %d", status.AgreedCount)
	}
}

func TestTimestampOrdering(t *testing.T) {
	now := time.Now()

	// Valid: monotonically increasing
	validReports := []ConsensusReport{
		{AlarmID: "a", Timestamp: now},
		{AlarmID: "b", Timestamp: now.Add(100 * time.Millisecond)},
		{AlarmID: "c", Timestamp: now.Add(200 * time.Millisecond)},
	}

	err := TimestampOrdering(validReports)
	if err != nil {
		t.Errorf("TimestampOrdering should accept monotonic timestamps: %v", err)
	}

	// Invalid: out of order
	invalidReports := []ConsensusReport{
		{AlarmID: "a", Timestamp: now},
		{AlarmID: "b", Timestamp: now.Add(200 * time.Millisecond)},
		{AlarmID: "c", Timestamp: now.Add(100 * time.Millisecond)}, // Earlier than previous
	}

	err = TimestampOrdering(invalidReports)
	if err == nil {
		t.Error("TimestampOrdering should reject out-of-order timestamps")
	}
}

func TestDetectCompromisedAlarm(t *testing.T) {
	isotope1 := Isotope{
		Family:    "iso-test",
		Version:   1,
		Signature: "sig1",
		Raw:       "raw1",
	}

	isotope2 := Isotope{
		Family:    "iso-test",
		Version:   1,
		Signature: "sig2",
		Raw:       "raw2",
	}

	// Three alarms, one disagreeing
	reports := []ConsensusReport{
		{AlarmID: "alarm-a", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-b", Isotope: isotope1, Timestamp: time.Now()},
		{AlarmID: "alarm-c", Isotope: isotope2, Timestamp: time.Now()}, // Compromised
	}

	keys := testKeys("alarm-a", "alarm-b", "alarm-c")
	compromised, err := DetectCompromisedAlarm(signAll(reports, keys), keys, 2)
	if err != nil {
		t.Fatalf("DetectCompromisedAlarm: %v", err)
	}
	if compromised == nil {
		t.Error("Should detect compromised alarm")
	} else if *compromised != "alarm-c" {
		t.Errorf("Expected alarm-c to be compromised, got %s", *compromised)
	}
}

func TestEntropyCheckPredicate(t *testing.T) {
	tests := []struct {
		name     string
		results  []string
		expected bool
	}{
		{
			name:     "High entropy (many unique)",
			results:  []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			expected: true,
		},
		{
			name:     "Low entropy (few unique)",
			results:  []string{"a", "a", "a", "a", "a", "b", "b", "b", "b", "b"},
			expected: false, // 10 runs, 2 unique = 20% (threshold 50% = 5 unique needed)
		},
		{
			name:     "Too low entropy (constant output)",
			results:  []string{"a", "a", "a", "a", "a", "a", "a", "a", "a", "a"},
			expected: false,
		},
		{
			name:     "Insufficient data",
			results:  []string{"a", "b"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TestEntropyCheck(tt.results)
			if result != tt.expected {
				t.Errorf("TestEntropyCheck(%v) = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestAgreementPatternPredicate(t *testing.T) {
	isotope1 := Isotope{
		Family:    "iso-test",
		Version:   1,
		Signature: "sig1",
		Raw:       "raw1",
	}

	isotope2 := Isotope{
		Family:    "iso-test",
		Version:   1,
		Signature: "sig2",
		Raw:       "raw2",
	}

	// Three alarms, two agree
	reports := []ConsensusReport{
		{AlarmID: "a", Isotope: isotope1},
		{AlarmID: "b", Isotope: isotope1},
		{AlarmID: "c", Isotope: isotope2},
	}

	keys := testKeys("a", "b", "c")
	ok, err := TestAgreementPattern(signAll(reports, keys), keys)
	if err != nil {
		t.Fatalf("TestAgreementPattern: %v", err)
	}
	if !ok {
		t.Error("AgreementPattern should detect 2/3 consensus")
	}

	// Three alarms, all disagree
	reports2 := []ConsensusReport{
		{AlarmID: "a", Isotope: isotope1},
		{AlarmID: "b", Isotope: isotope2},
		{AlarmID: "c", Isotope: isotope2},
	}

	ok, err = TestAgreementPattern(signAll(reports2, keys), keys)
	if err != nil {
		t.Fatalf("TestAgreementPattern: %v", err)
	}
	if !ok {
		t.Error("AgreementPattern should accept 2/3 consensus (even if different isotopes)")
	}

	// Only two alarms (insufficient for Byzantine)
	reports3 := []ConsensusReport{
		{AlarmID: "a", Isotope: isotope1},
		{AlarmID: "b", Isotope: isotope1},
	}

	ok, err = TestAgreementPattern(signAll(reports3, keys), keys)
	if err != nil {
		t.Fatalf("TestAgreementPattern: %v", err)
	}
	if ok {
		t.Error("AgreementPattern should reject <3 alarms")
	}
}

func TestConsensusReporter(t *testing.T) {
	isotope := Isotope{
		Family:    "iso-test",
		Version:   1,
		Signature: "sig123",
		Raw:       "raw",
	}

	reporter := &ConsensusReporter{
		AlarmID:    "alarm-a",
		SigningKey: []byte("key"),
	}

	report := reporter.Report(isotope)

	if report.AlarmID != "alarm-a" {
		t.Errorf("Report should have correct alarm ID")
	}

	if report.Isotope.Signature != isotope.Signature {
		t.Errorf("Report should preserve isotope signature")
	}

	if report.Timestamp.IsZero() {
		t.Error("Report should have valid timestamp")
	}
}

// TestVerifyConsensusSignaturesMalformed pins isotope#17/#19: the signature is
// peer-supplied, so a short, odd, non-hex or wrong-length value must be
// rejected — not sliced into a panic. A well-formed but wrong signature is a
// mismatch; every malformed shape is a malformed signature.
func TestVerifyConsensusSignaturesMalformed(t *testing.T) {
	key := []byte("seed-42")
	alarm := AlarmID("alarm-1")
	keys := map[AlarmID][]byte{alarm: key}

	good := GenerateIsotope(
		[]CanonicalToken{{Type: "FEATURE", Key: "name", Value: "f"}},
		key, "iso-test", 1,
	).Signature

	cases := []struct {
		name string
		sig  string
		want string
	}{
		{"valid", good, ""},
		{"empty", "", "malformed"},
		{"three chars", "abc", "malformed"},
		{"short hex", "deadbeef", "malformed"},
		{"odd length", good[:63], "malformed"},
		{"non-hex", good[:62] + "zz", "malformed"},
		{"wrong but well-formed", strings.Repeat("a", len(good)), "mismatch"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := []ConsensusReport{{AlarmID: alarm, Isotope: Isotope{Raw: "FEATURE:name:f", Signature: tc.sig}}}
			err := VerifyConsensusSignatures(reports, keys)

			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("valid signature rejected: %v", err)
			case tc.want == "":
				return
			case err == nil:
				t.Fatalf("%q was accepted", tc.sig)
			case !strings.Contains(err.Error(), tc.want):
				t.Fatalf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestVerifyConsensusRejectsForgedAgreement is isotope#20: three reports that
// copy one signature string formed consensus, because agreement grouped on a
// field nobody verified. VerifyConsensus checks first, so the set is rejected
// and no ConsensusFormed verdict is returned.
func TestVerifyConsensusRejectsForgedAgreement(t *testing.T) {
	keys := testKeys("alarm-1", "alarm-2", "alarm-3")

	forged := make([]ConsensusReport, 3)
	for i := range forged {
		forged[i] = ConsensusReport{
			AlarmID:   AlarmID(fmt.Sprintf("alarm-%d", i+1)),
			Isotope:   Isotope{Family: "f", Version: 1, Raw: "FEATURE:name:f", Signature: "forged"},
			Signature: "forged",
			Timestamp: time.Now(),
		}
	}

	status, err := VerifyConsensus(forged, keys, 2)
	if err == nil {
		t.Fatalf("forged agreement accepted: %+v", status)
	}
	if status.ConsensusFormed {
		t.Fatal("a rejected set must not carry a ConsensusFormed verdict")
	}

	if _, err := DetectCompromisedAlarm(forged, keys, 2); err == nil {
		t.Fatal("DetectCompromisedAlarm accepted a forged set")
	}
}
