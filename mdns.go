package isotope

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

// Server is satisfied by *zeroconf.Server. Call Shutdown when the process exits.
type Server interface {
	Shutdown()
}

// Advertise registers an mDNS record so LAN peers can discover this isotope.
// instanceName is the DNS-SD instance label (e.g. "adhd").
// serviceType is the mDNS service type (e.g. "_adhd-isotope._tcp").
// addr is the listen address (e.g. ":9090" or "0.0.0.0:9090").
// role and trustRung are included as TXT record fields.
func Advertise(instanceName, serviceType, addr, role string, trustRung int) (Server, error) {
	port, err := parsePortInt(addr)
	if err != nil {
		return nil, fmt.Errorf("isotope advertise: %w", err)
	}
	txt := []string{
		"v=1",
		"role=" + role,
		"trust_rung=" + strconv.Itoa(trustRung),
	}
	srv, err := zeroconf.Register(instanceName, serviceType, "local.", port, txt, nil)
	if err != nil {
		return nil, fmt.Errorf("isotope advertise: zeroconf register: %w", err)
	}
	return srv, nil
}

// parsePortInt parses the port number from an address string.
func parsePortInt(addr string) (int, error) {
	if idx := strings.Index(addr, "://"); idx >= 0 {
		addr = addr[idx+3:]
	}
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("invalid address %q: %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("non-numeric port in %q: %w", addr, err)
	}
	return port, nil
}
