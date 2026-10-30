package main

import (
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/james-gibson/isotope"
)

// The unlock agent is what makes "the authority stays offline while the issuer
// signs" true: the YubiKey (and its touch) is needed once, to decrypt the issuer
// key; the agent then holds that key in memory and answers signature requests
// over a unix socket. Unplugging the device does not stop the signing.
//
// The socket exposes two operations — sign a certificate, sign a bundle — and
// never raw signing. A key that signed arbitrary bytes would be a root that can
// be turned into anything by whoever can reach the socket.

const (
	agentMaxRequest  = 1 << 20 // a certificate or bundle body, with room to spare
	agentDialTimeout = 5 * time.Second
	agentReadTimeout = 30 * time.Second
)

type agentRequest struct {
	Op          string               `json:"op"`
	Issuer      string               `json:"issuer,omitempty"`
	Body        string               `json:"body,omitempty"` // base64 of the bytes to sign
	CreatedAt   int64                `json:"created_at,omitempty"`
	Certificate *isotope.Certificate `json:"certificate,omitempty"`
}

type agentResponse struct {
	Public      string                   `json:"public,omitempty"`
	Signature   *isotope.BundleSignature `json:"signature,omitempty"`
	Certificate *isotope.Certificate     `json:"certificate,omitempty"`
	Error       string                   `json:"error,omitempty"`
}

// defaultAgentSocket is a per-user socket path: two operators on one host must
// not be able to reach each other's unlocked key by guessing a name.
func defaultAgentSocket() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("seed42-%d.sock", os.Getuid()))
}

// serveAgent answers requests until idle expires or the listener fails. It
// returns nil when the idle window closes, which is a normal shutdown.
func serveAgent(s signer, socket string, idle time.Duration) error {
	if socket == "" {
		socket = defaultAgentSocket()
	}
	// A stale socket from a killed agent would otherwise refuse the bind.
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket %s: %w", socket, err)
	}
	ln, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("listen %s: %w", socket, err)
	}
	defer func() { _ = ln.Close() }()
	if err := os.Chmod(socket, 0o600); err != nil {
		return fmt.Errorf("chmod %s: %w", socket, err)
	}
	defer func() { _ = os.Remove(socket) }()

	unixLn, ok := ln.(*net.UnixListener)
	if !ok {
		return fmt.Errorf("listener is %T, want a unix listener", ln)
	}

	for {
		if err := unixLn.SetDeadline(time.Now().Add(idle)); err != nil {
			return fmt.Errorf("set accept deadline: %w", err)
		}
		conn, err := unixLn.Accept()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		handleAgentConn(s, conn)
	}
}

func handleAgentConn(s signer, conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(agentReadTimeout))

	var req agentRequest
	if err := json.NewDecoder(io.LimitReader(conn, agentMaxRequest)).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(agentResponse{Error: "bad request: " + err.Error()})
		return
	}
	resp := handleAgentRequest(s, req, time.Now())
	_ = json.NewEncoder(conn).Encode(resp)
}

// handleAgentRequest is the whole authority the socket grants. Anything not
// named here is refused, including a raw signature over caller-supplied bytes.
func handleAgentRequest(s signer, req agentRequest, now time.Time) agentResponse {
	switch req.Op {
	case "public":
		pub, ok := s.Public().(ed25519.PublicKey)
		if !ok {
			return agentResponse{Error: "signer has no ed25519 public key"}
		}
		return agentResponse{Public: base64.StdEncoding.EncodeToString(pub)}

	case "sign-bundle":
		body, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return agentResponse{Error: "body is not base64: " + err.Error()}
		}
		created := now
		if req.CreatedAt != 0 {
			created = time.Unix(req.CreatedAt, 0)
		}
		sig, err := s.SignBundle(req.Issuer, body, created)
		if err != nil {
			return agentResponse{Error: err.Error()}
		}
		return agentResponse{Signature: &sig}

	case "sign-certificate":
		if req.Certificate == nil {
			return agentResponse{Error: "no certificate"}
		}
		cert, err := s.SignCertificate(*req.Certificate)
		if err != nil {
			return agentResponse{Error: err.Error()}
		}
		return agentResponse{Certificate: &cert}

	default:
		return agentResponse{Error: fmt.Sprintf("unknown op %q: the agent signs certificates and bundles, not arbitrary bytes", req.Op)}
	}
}

// agentSigner is the client half: the same signer interface, backed by a running
// agent. Nothing about the resulting signature differs from a local key.
type agentSigner struct{ socket string }

func (a *agentSigner) call(req agentRequest) (agentResponse, error) {
	conn, err := net.DialTimeout("unix", a.socket, agentDialTimeout)
	if err != nil {
		return agentResponse{}, fmt.Errorf("unlock agent at %s: %w (is `seed42 agent` running?)", a.socket, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(agentReadTimeout))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return agentResponse{}, fmt.Errorf("write request: %w", err)
	}
	var resp agentResponse
	if err := json.NewDecoder(io.LimitReader(conn, agentMaxRequest)).Decode(&resp); err != nil {
		return agentResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

func (a *agentSigner) Public() crypto.PublicKey {
	resp, err := a.call(agentRequest{Op: "public"})
	if err != nil {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(resp.Public)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil
	}
	return ed25519.PublicKey(raw)
}

func (a *agentSigner) SignCertificate(cert isotope.Certificate) (isotope.Certificate, error) {
	resp, err := a.call(agentRequest{Op: "sign-certificate", Certificate: &cert})
	if err != nil {
		return isotope.Certificate{}, err
	}
	if resp.Certificate == nil {
		return isotope.Certificate{}, errors.New("agent returned no certificate")
	}
	return *resp.Certificate, nil
}

func (a *agentSigner) SignBundle(issuer string, body []byte, now time.Time) (isotope.BundleSignature, error) {
	resp, err := a.call(agentRequest{
		Op:        "sign-bundle",
		Issuer:    issuer,
		Body:      base64.StdEncoding.EncodeToString(body),
		CreatedAt: now.Unix(),
	})
	if err != nil {
		return isotope.BundleSignature{}, err
	}
	if resp.Signature == nil {
		return isotope.BundleSignature{}, errors.New("agent returned no signature")
	}
	return *resp.Signature, nil
}

// runAgent unlocks the issuer key once and serves it. The unlock is what touches
// the key: `age:`/`sops:` run the decryptor, whose identity may be a YubiKey
// plugin with a touch policy.
func runAgent(args []string) {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	in := fs.String("in", "", "how to unlock the issuer key: env, file:PATH, age:PATH or sops:PATH")
	socket := fs.String("socket", "", "socket to serve (default: "+defaultAgentSocket()+")")
	idle := fs.Duration("idle", 15*time.Minute, "close the socket after this long without a request")
	_ = fs.Parse(args)

	if *idle <= 0 {
		fail("-idle must be positive")
	}
	if strings.HasPrefix(*in, "agent:") {
		fail("an agent cannot unlock another agent")
	}
	s, err := resolveSigner(*in)
	if err != nil {
		fail("%v", err)
	}
	pub, ok := s.Public().(ed25519.PublicKey)
	if !ok {
		fail("unlocked signer has no ed25519 public key")
	}

	path := *socket
	if path == "" {
		path = defaultAgentSocket()
	}
	fmt.Fprintf(os.Stderr, "unlocked; public %s\n", base64.StdEncoding.EncodeToString(pub))
	fmt.Fprintf(os.Stderr, "agent listening on %s (idle %s)\n", path, *idle)

	if err := serveAgent(s, path, *idle); err != nil {
		fail("%v", err)
	}
}
