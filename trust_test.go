package isotope

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testSecret = "launch-secret-for-tests"
const testCluster = "demo-1234"

// fixedClock is a clock a test can reason about.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// staticBrowser is a Browser that always sees the same advertisements. It is
// how these tests avoid multicast: a real browse returns nothing on a host
// without local-network permission, which would let an mDNS test pass by never
// seeing anything.
type staticBrowser struct {
	ads []Advertisement
	err error
}

func (b staticBrowser) Browse(context.Context, string) ([]Advertisement, error) {
	return b.ads, b.err
}

// registry serves a /cluster directory with the given entries.
func registry(t *testing.T, entries map[string]clusterEntry) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cluster" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			t.Errorf("encode registry: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testConfig(baseURL string, ads ...Advertisement) TrustConfig {
	return TrustConfig{
		Secrets:       map[string]string{testCluster: testSecret},
		RegistryURL:   baseURL + "/cluster",
		Browser:       staticBrowser{ads: ads},
		Now:           fixedClock(time.Unix(1_800_000_000, 0)),
		BrowseTimeout: 50 * time.Millisecond,
	}
}

func localAd(cluster string, port int) Advertisement {
	// 127.0.0.1 is always one of this host's addresses, so the test does not
	// depend on which interface the machine has.
	return Advertisement{Cluster: cluster, Host: "127.0.0.1", Port: port}
}

// TestVerifyRegistrationRungLadder is the heart of #23: only attested evidence
// raises a rung, and presence alone raises nothing.
func TestVerifyRegistrationRungLadder(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	endpoint := "http://127.0.0.1:60099/mcp"
	entry := clusterEntry{AdhdMCP: endpoint}

	for _, tc := range []struct {
		name    string
		reg     Registration
		entries map[string]clusterEntry
		ads     []Advertisement
		want    int
		reason  string
	}{
		{
			name:    "no attestation is not evidence",
			reg:     Registration{Endpoint: endpoint},
			entries: map[string]clusterEntry{testCluster: entry},
			ads:     []Advertisement{localAd(testCluster, 19100)},
			want:    2,
			reason:  "no attestation",
		},
		{
			name:    "unknown cluster",
			reg:     Registration{Endpoint: endpoint, Attestation: &Attestation{Cluster: "someone-else", IssuedAt: now.Unix(), MAC: "x"}},
			entries: map[string]clusterEntry{"someone-else": entry},
			ads:     []Advertisement{localAd(testCluster, 19100)},
			want:    2,
			reason:  "not attested by this alarm",
		},
		{
			name:    "wrong secret",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation("not-the-secret", testCluster, endpoint, now))},
			entries: map[string]clusterEntry{testCluster: entry},
			ads:     []Advertisement{localAd(testCluster, 19100)},
			want:    2,
			reason:  "attestation",
		},
		{
			name:    "attested but not in the registry",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))},
			entries: map[string]clusterEntry{},
			ads:     []Advertisement{localAd(testCluster, 19100)},
			want:    2,
			reason:  "not in the discovery registry",
		},
		{
			name:    "registry names a different endpoint",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))},
			entries: map[string]clusterEntry{testCluster: {AdhdMCP: "http://127.0.0.1:60100/mcp"}},
			ads:     []Advertisement{localAd(testCluster, 19100)},
			want:    2,
			reason:  "which is not",
		},
		{
			name:    "attested and registry-confirmed, no mDNS: a roll looks like this",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))},
			entries: map[string]clusterEntry{testCluster: entry},
			want:    4,
			reason:  "mDNS did not confirm",
		},
		{
			name:    "a different cluster's advertisement proves nothing",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))},
			entries: map[string]clusterEntry{testCluster: entry},
			ads:     []Advertisement{localAd("other-cluster", 19100)},
			want:    4,
			reason:  "no advertisement names cluster",
		},
		{
			name:    "advertisement on another host proves nothing",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))},
			entries: map[string]clusterEntry{testCluster: entry},
			ads:     []Advertisement{{Cluster: testCluster, Host: "10.9.9.9", Port: 19100}},
			want:    4,
			reason:  "not this host",
		},
		{
			name:    "attested, registry-confirmed and advertised",
			reg:     Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))},
			entries: map[string]clusterEntry{testCluster: entry},
			ads:     []Advertisement{localAd(testCluster, 19100)},
			want:    6,
			reason:  "mDNS confirms",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := registry(t, tc.entries)
			cfg := testConfig(srv.URL, tc.ads...)
			verdict := VerifyRegistration(context.Background(), cfg, tc.reg)
			if verdict.Rung != tc.want {
				t.Fatalf("rung = %d, want %d (reasons: %v)", verdict.Rung, tc.want, verdict.Reasons)
			}
			if !strings.Contains(strings.Join(verdict.Reasons, "; "), tc.reason) {
				t.Fatalf("reasons %v do not mention %q", verdict.Reasons, tc.reason)
			}
		})
	}
}

// TestVerifyTrustWithoutAttestationStaysAtTheFloor is the compatibility path:
// the old entry point cannot prove anything, so it cannot rise above 2 even when
// a registry and an advertisement are both present.
func TestVerifyTrustWithoutAttestationStaysAtTheFloor(t *testing.T) {
	srv := registry(t, map[string]clusterEntry{testCluster: {AdhdMCP: "http://127.0.0.1:60099/mcp"}})
	// VerifyTrust uses the default config, so point the default at this server
	// through the environment the tests share: call the config-aware path with
	// no attestation instead, which is the same code.
	verdict := VerifyRegistration(context.Background(), TrustConfig{RegistryURL: srv.URL}, Registration{Endpoint: "http://127.0.0.1:60099/mcp"})
	if verdict.Rung != 2 {
		t.Fatalf("rung = %d, want 2", verdict.Rung)
	}
	if got := VerifyTrust(context.Background(), "http://127.0.0.1:60099/mcp"); got != 2 {
		t.Fatalf("VerifyTrust = %d, want 2 (it has no attestation to offer)", got)
	}
}

// TestAttestationBindsClusterAndEndpoint: a receipt is not a bearer token. The
// same MAC cannot be replayed for another endpoint, another cluster, or later.
func TestAttestationBindsClusterAndEndpoint(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	att := SignAttestation(testSecret, testCluster, "http://127.0.0.1:60099/mcp", now)

	if err := att.Verify(testSecret, "http://127.0.0.1:60099/mcp", now, 0); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}
	if err := att.Verify(testSecret, "http://127.0.0.1:60100/mcp", now, 0); err == nil {
		t.Fatal("receipt verified for a different endpoint")
	}
	if err := att.Verify("another-secret", "http://127.0.0.1:60099/mcp", now, 0); err == nil {
		t.Fatal("receipt verified under a different secret")
	}
	if err := att.Verify(testSecret, "http://127.0.0.1:60099/mcp", now.Add(10*time.Minute), 0); err == nil {
		t.Fatal("stale receipt accepted")
	}
	if err := att.Verify(testSecret, "http://127.0.0.1:60099/mcp", now.Add(30*time.Second), 0); err != nil {
		t.Fatalf("receipt within the skew window rejected: %v", err)
	}
	// An empty secret must never verify: an alarm with no launch secret cannot
	// attest anyone, and "" must not compare equal to a MAC of "".
	if err := att.Verify("", "http://127.0.0.1:60099/mcp", now, 0); !errors.Is(err, ErrAttestation) {
		t.Fatalf("empty secret = %v, want ErrAttestation", err)
	}
}

// TestMatchEndpointComparesHostAndPort is #23's second hole: the port agreeing
// was enough to match two different hosts.
func TestMatchEndpointComparesHostAndPort(t *testing.T) {
	for _, tc := range []struct {
		name     string
		endpoint string
		claimed  string
		want     bool
	}{
		{"same host and port", "http://127.0.0.1:60099/mcp", "http://127.0.0.1:60099/mcp", true},
		{"same host and port, different paths", "http://127.0.0.1:60099/mcp", "http://127.0.0.1:60099/other", true},
		{"bare host:port", "127.0.0.1:60099", "http://127.0.0.1:60099/mcp", true},
		{"different host, same port", "http://127.0.0.1:60099/mcp", "http://10.9.9.9:60099/mcp", false},
		{"different port, same host", "http://127.0.0.1:60099/mcp", "http://127.0.0.1:60100/mcp", false},
		{"wildcard endpoint reaches any host", ":60099", "http://10.9.9.9:60099/mcp", true},
		{"wildcard claim matches a specific endpoint", "http://127.0.0.1:60099/mcp", ":60099", true},
		{"unspecified address is a wildcard", "0.0.0.0:60099", "http://192.168.1.12:60099/mcp", true},
		{"loopback spellings agree", "http://localhost:60099/mcp", "http://127.0.0.1:60099/mcp", true},
		{"ipv6 loopback agrees with itself", "http://[::1]:60099/mcp", "http://[::1]:60099/mcp", true},
		{"ipv6 loopback is not a LAN address", "http://[::1]:60099/mcp", "http://192.168.1.12:60099/mcp", false},
		{"https default port", "https://example.test/mcp", "https://example.test:443/mcp", true},
		{"no port at all", "http://127.0.0.1/mcp", "http://127.0.0.1:60099/mcp", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchEndpoint(tc.endpoint, tc.claimed); got != tc.want {
				t.Fatalf("matchEndpoint(%q, %q) = %v, want %v", tc.endpoint, tc.claimed, got, tc.want)
			}
		})
	}
}

// TestVerifyRegistrationNeverDemotesOnAMissedBrowse is the rolling-restart
// constraint from #23: absence is expected during a roll, so a browse that
// misses costs the top rung and nothing else. It must never push a cluster below
// what its attestation and registry entry already proved.
func TestVerifyRegistrationNeverDemotesOnAMissedBrowse(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	endpoint := "http://127.0.0.1:60099/mcp"
	srv := registry(t, map[string]clusterEntry{testCluster: {AdhdMCP: endpoint}})
	reg := Registration{Endpoint: endpoint, Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now))}

	// Browse unseen, then seen, then unseen again: the rung returns to 4, never
	// to 2, and recovers to 6 on the next tick.
	for _, step := range []struct {
		name string
		ads  []Advertisement
		want int
	}{
		{"during the roll", nil, 4},
		{"after the roll", []Advertisement{localAd(testCluster, 19100)}, 6},
		{"a later miss", nil, 4},
		{"recovered", []Advertisement{localAd(testCluster, 19100)}, 6},
	} {
		t.Run(step.name, func(t *testing.T) {
			cfg := testConfig(srv.URL, step.ads...)
			got := VerifyRegistration(context.Background(), cfg, reg)
			if got.Rung != step.want {
				t.Fatalf("rung = %d, want %d (reasons: %v)", got.Rung, step.want, got.Reasons)
			}
		})
	}
}

// TestVerifyRegistrationRegistryDownIsNotEvidence: an unreachable registry
// cannot prove anything, so it earns the floor rather than a guess. This is the
// difference between a *liveness* probe (where unreachable is a warning) and an
// *evidence* probe (where it is nothing).
func TestVerifyRegistrationRegistryDownIsNotEvidence(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	endpoint := "http://127.0.0.1:60099/mcp"
	cfg := testConfig("http://127.0.0.1:1", localAd(testCluster, 19100))
	verdict := VerifyRegistration(context.Background(), cfg, Registration{
		Endpoint:    endpoint,
		Attestation: ptr(SignAttestation(testSecret, testCluster, endpoint, now)),
	})
	if verdict.Rung != 2 {
		t.Fatalf("rung = %d, want 2: %v", verdict.Rung, verdict.Reasons)
	}
}

func ptr[T any](v T) *T { return &v }
