package isotope

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	lezzDiscoveryPort = 19100
	lezzMDNSService   = "_lezz-demo._tcp"
	// clusterTXTKey is the DNS-SD TXT field a lezz demo publishes its cluster
	// identity in. Without it, "a demo exists" is the only evidence mDNS offers,
	// which is what #23 was about.
	clusterTXTKey = "cluster"
)

// clusterEntry mirrors the ClusterInfo JSON published by lezz demo's /cluster endpoint.
type clusterEntry struct {
	AdhdMCP string `json:"adhd_mcp"`
}

// Advertisement is one DNS-SD record a browser saw. It is the browser's output,
// not zeroconf's, so a test can present the exact record it wants to be judged
// on — including a record for a cluster that is not the one being verified.
type Advertisement struct {
	Cluster string
	Host    string
	Port    int
}

// Browser finds DNS-SD advertisements. The zero value is not usable; use
// ZeroConfBrowser or a test double.
type Browser interface {
	Browse(ctx context.Context, service string) ([]Advertisement, error)
}

// ZeroConfBrowser is the real browser: a bounded multicast browse, returning
// every record it saw so the verifier can choose among them. Tests inject a
// Browser instead of this, because multicast is exactly what a test host may
// not have (macOS local-network permission blocks the browse while dns-sd still
// sees the service).
type ZeroConfBrowser struct {
	// Timeout bounds the browse. Zero uses DefaultBrowseTimeout.
	Timeout time.Duration
}

// DefaultBrowseTimeout is how long a verification waits for an advertisement.
const DefaultBrowseTimeout = 4 * time.Second

// Browse returns every advertisement of service seen within the window.
func (b ZeroConfBrowser) Browse(ctx context.Context, service string) ([]Advertisement, error) {
	timeout := b.Timeout
	if timeout <= 0 {
		timeout = DefaultBrowseTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("create mDNS resolver: %w", err)
	}
	entries := make(chan *zeroconf.ServiceEntry, 8)
	if err := resolver.Browse(ctx, service, "local.", entries); err != nil {
		return nil, fmt.Errorf("mDNS browse: %w", err)
	}

	var found []Advertisement
	for {
		select {
		case e := <-entries:
			found = append(found, advertisementFrom(e))
		case <-ctx.Done():
			return found, nil
		}
	}
}

// advertisementFrom maps a zeroconf entry onto the fields that decide trust: the
// cluster id it claims, the address it names, and the port it opens.
func advertisementFrom(e *zeroconf.ServiceEntry) Advertisement {
	ad := Advertisement{Port: e.Port}
	for _, txt := range e.Text {
		if key, value, ok := strings.Cut(txt, "="); ok && key == clusterTXTKey {
			ad.Cluster = value
		}
	}
	switch {
	case len(e.AddrIPv4) > 0:
		ad.Host = e.AddrIPv4[0].String()
	case len(e.AddrIPv6) > 0:
		ad.Host = e.AddrIPv6[0].String()
	}
	return ad
}

// TrustConfig is the verifier's own evidence configuration: which clusters it
// trusts, where their registry is, and how it learns about advertisements.
// Every field has a production default, so the zero value still verifies
// (against a registry on loopback and the real mDNS browser) — but it trusts no
// cluster, so nothing rises above rung 2.
type TrustConfig struct {
	// Secrets maps cluster identity to the launch secret its members were given,
	// including derived subkeys handed to child clusters. A symmetric cluster
	// that is not named here cannot be attested, whatever its registry says.
	Secrets map[string]string
	// Roots maps a trust root's name to its ed25519 public key (base64). A
	// certificate issued by a root named here is accepted, which is how a private
	// cluster trusts another cluster's members without holding their secret.
	// Nothing is implicit: an empty map pins nothing, so a cluster trusts exactly
	// what it was given, online or offline (docs/trust-roots.md).
	Roots map[string]string
	// RegistryURL is the discovery registry to read. Empty means
	// http://127.0.0.1:19100/cluster.
	RegistryURL string
	// Browser resolves advertisements. Nil uses ZeroConfBrowser.
	Browser Browser
	// Client is the HTTP client for the registry. Nil uses a 2s client.
	Client *http.Client
	// Now is the clock, injectable for tests. Nil uses time.Now.
	Now func() time.Time
	// Skew is the receipt clock tolerance. Zero uses DefaultAttestationSkew.
	Skew time.Duration
	// BrowseTimeout bounds the mDNS confirmation. Zero uses
	// DefaultBrowseTimeout.
	BrowseTimeout time.Duration
}

// TrustVerdict is the outcome of verification, with the reasons behind it. The
// reasons are part of the answer: a rung that cannot be explained cannot be
// audited, and "why is this 4?" is the question an operator asks first.
type TrustVerdict struct {
	Rung    int
	Reasons []string
}

// explain appends a reason and returns the verdict unchanged otherwise, so the
// ladder below reads as the evidence it demands.
func (v *TrustVerdict) explain(format string, args ...any) {
	v.Reasons = append(v.Reasons, fmt.Sprintf(format, args...))
}

// keyring resolves the configuration into the roots and secrets a receipt is
// checked against. There is no implicit root: what a cluster trusts is what its
// configuration (ultimately its seed) pinned.
func (c TrustConfig) keyring() (Keyring, error) {
	roots, err := ParseRoots(c.Roots)
	if err != nil {
		return Keyring{}, err
	}
	return Keyring{Roots: roots, Secrets: c.Secrets}, nil
}

// VerifyTrust determines the initial 42i trust rung for a registering isotope.
//
// This entry point has no attestation, so it can only reach rung 2: the endpoint
// is self-declared, and presence in a registry anyone can write is not evidence
// (isotope#23). Callers that can present a cluster receipt — everything lezz
// starts does — should use VerifyRegistration.
func VerifyTrust(ctx context.Context, endpoint string) int {
	return VerifyRegistration(ctx, TrustConfig{}, Registration{Endpoint: endpoint}).Rung
}

// VerifyRegistration derives the rung for a registration, requiring
// cluster-scoped attestation before presence counts for anything.
//
//	6 — Certification:   attested, the cluster's registry entry names this
//	                     endpoint (host and port), and mDNS confirms the cluster.
//	4 — Higher Authority: attested and registry-confirmed, but mDNS did not
//	                     confirm. A rolling restart looks exactly like this, so
//	                     the absence costs the top rung and nothing else.
//	2 — Harness Tools:    everything else. Unattested, wrong secret, unknown
//	                     cluster, no registry entry, or an entry for a different
//	                     host: unproven evidence earns the floor, not a guess.
//
// Trust is cluster-scoped: the receipt names a cluster, the registry lookup is
// that cluster's entry, and the advertisement must name that cluster too. One
// grant per cluster, as the rolling-restart constraint requires.
func VerifyRegistration(ctx context.Context, cfg TrustConfig, reg Registration) TrustVerdict {
	verdict := TrustVerdict{Rung: 2}
	endpoint := strings.TrimSpace(reg.Endpoint)
	if endpoint == "" {
		verdict.explain("no endpoint to verify")
		return verdict
	}

	att := reg.Attestation
	if att == nil {
		verdict.explain("no attestation: %s is self-declared, and a registry entry is not evidence (isotope#23)", endpoint)
		return verdict
	}
	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now()
	}
	keyring, err := cfg.keyring()
	if err != nil {
		verdict.explain("trust configuration is unusable: %v", err)
		return verdict
	}
	if verifyErr := att.VerifyWith(keyring, endpoint, now, cfg.Skew); verifyErr != nil {
		verdict.explain("attestation for cluster %q rejected: %v", att.Cluster, verifyErr)
		return verdict
	}
	verdict.explain("attested by cluster %q (%s)", att.Cluster, att.EffectiveScheme())

	// From here the evidence is proven to belong to a real cluster; presence in
	// its directory is what raises the rung, and only for the endpoint it names.
	entry, err := lookupCluster(ctx, cfg, att.Cluster)
	switch {
	case err != nil:
		verdict.explain("registry for cluster %q unavailable: %v", att.Cluster, err)
		return verdict
	case entry == nil:
		verdict.explain("cluster %q is not in the discovery registry", att.Cluster)
		return verdict
	case !matchEndpoint(endpoint, entry.AdhdMCP):
		verdict.explain("registry entry for %q names %s, which is not %s", att.Cluster, entry.AdhdMCP, endpoint)
		return verdict
	}
	verdict.Rung = 4
	verdict.explain("registry entry for %q names this endpoint", att.Cluster)

	confirmed, detail := confirmCluster(ctx, cfg, att.Cluster)
	if !confirmed {
		verdict.explain("mDNS did not confirm cluster %q: %s", att.Cluster, detail)
		return verdict
	}
	verdict.Rung = 6
	verdict.explain("mDNS confirms cluster %q: %s", att.Cluster, detail)
	return verdict
}

// lookupCluster reads the registry and returns the entry registered under name,
// or nil when the registry is reachable and has no such entry.
func lookupCluster(ctx context.Context, cfg TrustConfig, name string) (*clusterEntry, error) {
	url := cfg.RegistryURL
	if url == "" {
		url = fmt.Sprintf("http://127.0.0.1:%d/cluster", lezzDiscoveryPort)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry answered HTTP %d", resp.StatusCode)
	}
	var entries map[string]clusterEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if entry, ok := entries[name]; ok {
		return &entry, nil
	}
	return nil, nil
}

// confirmCluster asks mDNS whether the cluster is advertising a registry on this
// host. The advertisement must carry the cluster identity in its TXT record: "a
// demo is running somewhere" is what #23 rejected as evidence, and a record for
// another cluster proves nothing about this one.
func confirmCluster(ctx context.Context, cfg TrustConfig, cluster string) (bool, string) {
	browser := cfg.Browser
	if browser == nil {
		browser = ZeroConfBrowser{Timeout: cfg.BrowseTimeout}
	}
	browseCtx, cancel := context.WithTimeout(ctx, browseTimeout(cfg))
	defer cancel()

	ads, err := browser.Browse(browseCtx, lezzMDNSService)
	if err != nil {
		return false, err.Error()
	}
	if len(ads) == 0 {
		return false, "no advertisement seen"
	}
	registryHost := registryHostOf(cfg.RegistryURL)
	for _, ad := range ads {
		if ad.Cluster != cluster {
			continue
		}
		if ad.Port != 0 && ad.Port != lezzDiscoveryPort {
			continue
		}
		if !hostAnswersHere(ad.Host) {
			return false, fmt.Sprintf("cluster %q advertises %s, which is not this host", cluster, ad.Host)
		}
		if registryHost != "" && !isUnspecified(registryHost) && !strings.EqualFold(ad.Host, registryHost) {
			return false, fmt.Sprintf("cluster %q advertises %s, but its registry is read at %s", cluster, ad.Host, registryHost)
		}
		return true, fmt.Sprintf("%s:%d", ad.Host, ad.Port)
	}
	return false, fmt.Sprintf("no advertisement names cluster %q", cluster)
}

func browseTimeout(cfg TrustConfig) time.Duration {
	if cfg.BrowseTimeout > 0 {
		return cfg.BrowseTimeout
	}
	return DefaultBrowseTimeout
}

// registryHostOf extracts the host from a registry URL, or "" when unset.
func registryHostOf(registryURL string) string {
	if registryURL == "" {
		return ""
	}
	host := registryURL
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	if slash := strings.IndexByte(host, '/'); slash >= 0 {
		host = host[:slash]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// hostAnswersHere reports whether host is one of this machine's addresses. The
// registry is read over loopback, while the advertisement necessarily names the
// interface it is reachable on, so the two hosts are compared by "is this the
// same machine", not by string equality.
func hostAnswersHere(host string) bool {
	if host == "" {
		return false
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.Equal(ip) {
			return true
		}
	}
	return false
}

// matchEndpoint reports whether two endpoints name the same host and port.
//
// Port-only matching was #23's second hole: `http://10.9.9.9:60099` matched
// `http://127.0.0.1:60099/config` because the ports agreed. Now the hosts must
// agree too, with one deliberate exception — an endpoint bound to all
// interfaces (":9090", "0.0.0.0:9090") is reachable on every address this host
// has, so it matches any host. That exception is what keeps a wildcard adhd
// working; without it the check would only pass for loopback-bound instances.
func matchEndpoint(endpoint, claimed string) bool {
	endHost, endPort, ok := splitEndpoint(endpoint)
	if !ok {
		return false
	}
	claimHost, claimPort, ok := splitEndpoint(claimed)
	if !ok {
		return false
	}
	if endPort != claimPort {
		return false
	}
	if endHost == "" || claimHost == "" {
		return true // either side binds every interface at this port
	}
	return endHost == claimHost
}

// splitEndpoint normalises an endpoint into a comparable host and port. The host
// is "" for a wildcard bind, "loopback" for any loopback spelling, and the
// canonical IP text otherwise.
func splitEndpoint(s string) (host, port string, ok bool) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "://"); idx >= 0 {
		scheme := strings.ToLower(s[:idx])
		s = s[idx+3:]
		// Strip any userinfo so "user:pass@host:port" still parses.
		if at := strings.LastIndexByte(s, '@'); at >= 0 {
			s = s[at+1:]
		}
		if !strings.Contains(s, ":") && (scheme == "http" || scheme == "https") {
			if slash := strings.IndexByte(s, '/'); slash >= 0 {
				s = s[:slash]
			}
			if scheme == "http" {
				return s, "80", true
			}
			return s, "443", true
		}
	}
	if slash := strings.IndexByte(s, '/'); slash >= 0 {
		s = s[:slash]
	}
	h, p, err := net.SplitHostPort(s)
	if err != nil {
		return "", "", false
	}
	if _, convErr := strconv.Atoi(p); convErr != nil {
		return "", "", false
	}
	return normaliseHost(h), p, true
}

// normaliseHost folds the spellings that mean the same listener together.
func normaliseHost(h string) string {
	h = strings.Trim(h, "[]")
	if h == "" {
		return ""
	}
	if ip := net.ParseIP(h); ip != nil {
		switch {
		case ip.IsUnspecified():
			return ""
		case ip.IsLoopback():
			return "loopback"
		default:
			return ip.String()
		}
	}
	if strings.EqualFold(h, "localhost") {
		return "loopback"
	}
	return strings.ToLower(h)
}

// isUnspecified reports whether a registry host means "every interface".
func isUnspecified(host string) bool {
	host = strings.Trim(host, "[]")
	if host == "" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsUnspecified()
	}
	return false
}
