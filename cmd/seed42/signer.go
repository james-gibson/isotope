package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/james-gibson/isotope"
)

// The custody chain this file implements:
//
//	YubiKey (authority)  ──touch──▶  age/SOPS file  ──▶  seed issuer key
//	                                                        │
//	                                     signs bundles and cluster certificates
//
// The second key is the seed issuer: its public half is what seeds and alarm
// configs pin, and its private half is useless without the YubiKey that unlocks
// it. Unlocking is a touch event on a PIV slot; once unlocked, the key is in
// memory and the YubiKey can be unplugged — which is what "the authority stays
// offline while the issuer signs" means in practice.
type signer interface {
	// Public is the seed issuer's public key, the half that gets pinned.
	Public() crypto.PublicKey
	// SignCertificate issues a cluster certificate.
	SignCertificate(isotope.Certificate) (isotope.Certificate, error)
	// SignBundle signs the bytes a seed travels as.
	SignBundle(issuer string, body []byte, now time.Time) (isotope.BundleSignature, error)
}

// localSigner is a key that is already in this process: read from the
// environment, a file, or freshly decrypted by age/SOPS.
type localSigner struct{ key ed25519.PrivateKey }

func (l localSigner) Public() crypto.PublicKey { return l.key.Public() }

func (l localSigner) SignCertificate(cert isotope.Certificate) (isotope.Certificate, error) {
	return isotope.SignCertificateWith(cert, l.key)
}

func (l localSigner) SignBundle(issuer string, body []byte, now time.Time) (isotope.BundleSignature, error) {
	return isotope.SignSeedBundle(l.key, issuer, body, now)
}

// resolveSigner turns -signer into a signer.
//
//	env             $SEED42_PRIVATE_KEY, base64 (default)
//	file:PATH       a 0600 file holding the same base64 key
//	age:PATH        decrypt PATH with age; the identity may be a YubiKey plugin,
//	                so the read needs the device and its touch policy
//	sops:PATH       decrypt PATH with sops
//	agent:PATH      ask a running `seed42 agent` that holds an unlocked key
//
// Nothing about the signature changes across these: the same bytes sign, so a
// verifier cannot tell whether the authority was on the host or on a key.
func resolveSigner(spec string) (signer, error) {
	spec = strings.TrimSpace(spec)
	switch {
	case spec == "" || spec == "env":
		key, err := seedPrivateKey()
		if err != nil {
			return nil, err
		}
		return localSigner{key: key}, nil
	case strings.HasPrefix(spec, "file:"):
		key, err := keyFromFile(strings.TrimSpace(strings.TrimPrefix(spec, "file:")))
		if err != nil {
			return nil, err
		}
		return localSigner{key: key}, nil
	case strings.HasPrefix(spec, "age:"):
		key, err := decryptedKey("age", strings.TrimSpace(strings.TrimPrefix(spec, "age:")))
		if err != nil {
			return nil, err
		}
		return localSigner{key: key}, nil
	case strings.HasPrefix(spec, "sops:"):
		key, err := decryptedKey("sops", strings.TrimSpace(strings.TrimPrefix(spec, "sops:")))
		if err != nil {
			return nil, err
		}
		return localSigner{key: key}, nil
	case strings.HasPrefix(spec, "agent:"):
		path := strings.TrimSpace(strings.TrimPrefix(spec, "agent:"))
		if path == "" {
			return nil, fmt.Errorf("-signer agent: needs a socket path")
		}
		return &agentSigner{socket: path}, nil
	default:
		return nil, fmt.Errorf("unknown signer %q: want env, file:PATH, age:PATH, sops:PATH or agent:PATH", spec)
	}
}

// decryptTimeout bounds an unlock. A hardware unlock includes a human touch, so
// it is generous — but a stuck plugin should not hang a certificate issue.
const decryptTimeout = 2 * time.Minute

// decryptedKey runs the external decryptor and parses the issuer key from its
// output.
//
// The decryptors are shelled out to on purpose: age's plugin protocol (and so a
// YubiKey's PIV ECDH) lives in `age`, and SOPS' age support lives in `sops`.
// Linking either would not remove the dependency, and would put the identity
// handling in this tool instead of the one the operator already audits.
func decryptedKey(tool, path string) (ed25519.PrivateKey, error) {
	if path == "" {
		return nil, fmt.Errorf("-signer %s: needs a path", tool)
	}
	body, err := runDecryptor(tool, path)
	if err != nil {
		return nil, err
	}
	return decodeKeyMaterial(tool+":"+path, body)
}

func runDecryptor(tool, path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), decryptTimeout)
	defer cancel()

	var args []string
	switch tool {
	case "age":
		args = append(args, "-d")
		// The YubiKey plugin's identity is an age identity file; naming it
		// explicitly keeps the unlock from depending on the caller's home.
		if id := strings.TrimSpace(os.Getenv("SEED42_AGE_IDENTITY_FILE")); id != "" {
			args = append(args, "-i", id)
		}
	case "sops":
		args = append(args, "-d")
	default:
		return nil, fmt.Errorf("unknown decryptor %q", tool)
	}
	args = append(args, path)

	cmd := exec.CommandContext(ctx, tool, args...) //nolint:gosec // tool is one of two fixed names
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%s -d %s: timed out after %s (was the key touched?)", tool, path, decryptTimeout)
		}
		return nil, fmt.Errorf("%s -d %s: %w: %s", tool, path, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// decodeKeyMaterial reads the issuer key out of a decrypted document. Two shapes
// are accepted because they are what the tools produce: the raw base64 key, or a
// key/value document (dotenv, or a YAML/JSON field) holding SEED42_PRIVATE_KEY.
func decodeKeyMaterial(what string, body []byte) (ed25519.PrivateKey, error) {
	text := strings.TrimSpace(string(body))
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		for _, key := range []string{"SEED42_PRIVATE_KEY=", "SEED42_PRIVATE_KEY:", "private_key:", "private_key="} {
			if strings.HasPrefix(line, key) {
				value := strings.TrimSpace(strings.TrimPrefix(line, key))
				value = strings.Trim(value, `"'`)
				return decodePrivateKey(what, value)
			}
		}
	}
	return decodePrivateKey(what, text)
}

func keyFromFile(path string) (ed25519.PrivateKey, error) {
	if path == "" {
		return nil, fmt.Errorf("-signer file: needs a path")
	}
	data, err := os.ReadFile(path) //nolint:gosec // the operator names the key file
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return decodePrivateKey(path, string(data))
}

// decodePrivateKey reads an ed25519 private key as base64, accepting the 64-byte
// private key or its 32-byte seed — the two shapes an operator's key store and a
// key file respectively hold.
func decodePrivateKey(what, encoded string) (ed25519.PrivateKey, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, fmt.Errorf("%s is empty; provision the seed issuer key", what)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s is not base64: %w", what, err)
	}
	switch len(raw) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	default:
		return nil, fmt.Errorf("%s is %d bytes, want %d (private key) or %d (seed)", what, len(raw), ed25519.PrivateKeySize, ed25519.SeedSize)
	}
}

// seedPrivateKey reads the issuer key from the environment. It is never a flag,
// so it does not land in shell history.
func seedPrivateKey() (ed25519.PrivateKey, error) {
	return decodePrivateKey("$SEED42_PRIVATE_KEY", os.Getenv("SEED42_PRIVATE_KEY"))
}

// parsePublicKey reads the pinned root key: base64 inline, or a file holding
// base64 or an OpenSSH ed25519 line (what a GitHub keys file or the public half
// printed by `seed42 init` look like).
func parsePublicKey(spec string) (ed25519.PublicKey, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("-pub is required: a signature is only as good as the key you pinned")
	}
	body := spec
	if strings.HasPrefix(spec, "@") {
		data, err := os.ReadFile(strings.TrimSpace(strings.TrimPrefix(spec, "@")))
		if err != nil {
			return nil, fmt.Errorf("read public key: %w", err)
		}
		body = strings.TrimSpace(string(data))
	}
	if strings.HasPrefix(body, "ssh-ed25519 ") {
		key, err := isotope.ParseSSHEd25519(body)
		if err != nil {
			return nil, err
		}
		return key.PublicKey, nil
	}
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("public key is neither base64 nor an ssh-ed25519 line: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key is %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}
