package isotope

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
)

// Key is a public key pinned as a trust root, with the label it was published
// under.
//
// The label matters because a root is a *name* the caller chose: publishing a key
// ("this is james-gibson's GitHub key") is not a claim about what it may do, and
// pinning it is the claim. A person's published key works as a root exactly like
// a lab seed's does — the verification code cannot tell them apart, which is the
// point (docs/trust-roots.md).
type Key struct {
	// Name is the root's name in a trust set, e.g. "github:james-gibson".
	Name string
	// Comment is the trailing comment of the OpenSSH line, when present.
	Comment string
	// PublicKey is the ed25519 key.
	PublicKey ed25519.PublicKey
}

// ParseSSHEd25519 reads one OpenSSH public key line into a root key.
//
// GitHub publishes a user's SSH keys as lines of exactly this form at
// https://github.com/<user>.keys, so a personal key can be pinned with no
// conversion step and no network at verification time: fetch once, pin, verify
// offline forever.
//
// Only ed25519 is accepted. An RSA or ECDSA key is a different algorithm, and
// silently refusing is better than pretending the pin means something it does
// not.
func ParseSSHEd25519(line string) (Key, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return Key{}, fmt.Errorf("not an OpenSSH public key line: %q", truncate(line))
	}
	kind := fields[0]
	if kind != "ssh-ed25519" {
		return Key{}, fmt.Errorf("key type %q is not ssh-ed25519", kind)
	}
	raw, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return Key{}, fmt.Errorf("key body is not base64: %w", err)
	}

	// OpenSSH wire format: string type, string key. Both are uint32 length
	// prefixed. Re-declaring the type inside the blob is what stops a key being
	// relabelled by editing the line.
	blobType, rest, err := readSSHString(raw)
	if err != nil {
		return Key{}, err
	}
	if string(blobType) != "ssh-ed25519" {
		return Key{}, fmt.Errorf("key blob declares type %q, not ssh-ed25519", truncate(string(blobType)))
	}
	pub, rest, err := readSSHString(rest)
	if err != nil {
		return Key{}, err
	}
	if len(pub) != ed25519.PublicKeySize {
		return Key{}, fmt.Errorf("key is %d bytes, want %d", len(pub), ed25519.PublicKeySize)
	}
	if len(rest) != 0 {
		return Key{}, fmt.Errorf("%d trailing bytes after the key", len(rest))
	}

	key := Key{PublicKey: ed25519.PublicKey(pub)}
	if len(fields) > 2 {
		key.Comment = strings.Join(fields[2:], " ")
	}
	return key, nil
}

// ParseKeyList reads every ed25519 line in a published key file (for example the
// body of https://github.com/<user>.keys) and names each one namePrefix + the
// key's comment. GitHub strips comments from the .keys file, so a key published
// there is named by its fingerprint instead — stable across edits, and the same
// string `ssh-keygen -l` shows.
//
// Lines for other algorithms are skipped rather than failing the file: a GitHub
// key list legitimately contains an RSA key, and it does not make the ed25519 ones
// less usable.
func ParseKeyList(body, namePrefix string) ([]Key, error) {
	var keys []Key
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "ssh-ed25519 ") {
			continue
		}
		key, err := ParseSSHEd25519(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		name := key.Comment
		if name == "" {
			name = ShortFingerprint(key.PublicKey)
		}
		key.Name = namePrefix + name
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no ssh-ed25519 keys found")
	}
	return keys, nil
}

// readSSHString reads one length-prefixed field of the OpenSSH wire format.
func readSSHString(b []byte) (field, rest []byte, err error) {
	if len(b) < 4 {
		return nil, nil, fmt.Errorf("truncated key blob")
	}
	n := int(binary.BigEndian.Uint32(b[:4]))
	if n < 0 || 4+n > len(b) {
		return nil, nil, fmt.Errorf("key blob declares %d bytes, has %d", n, len(b)-4)
	}
	return b[4 : 4+n], b[4+n:], nil
}

// ShortFingerprint is the first 12 characters of a key's SHA256 fingerprint, as
// base64. It exists to name a key that was published without a comment, so the
// name is stable when its neighbors change.
func ShortFingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	encoded := base64.RawStdEncoding.EncodeToString(sum[:])
	if len(encoded) > 12 {
		encoded = encoded[:12]
	}
	return "SHA256:" + encoded
}

func truncate(s string) string {
	if len(s) > 32 {
		return s[:32] + "…"
	}
	return s
}
