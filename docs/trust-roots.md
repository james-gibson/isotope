# Trust roots, cluster certificates, and derived subkeys

**Status:** decided, implemented in isotope. Extends
[isotope#23](../README.md#trust-verification) (attested evidence) rather than
replacing it.

## The shape

Trust is a set of **trusted sets**, and a set can name an upstream root instead
of enumerating its members. Two kinds of evidence, one ladder:

```
                    seed-42  (public root, key published)
                       │  signs
        ┌──────────────┼───────────────┐
   cluster: base    cluster: lab     cluster: ci        ← public clusters
        │ signs receipts over cluster|endpoint|time
   adhd, tuner, …                                        ← members

   cluster: home  (private root, self-signed, pinned locally)
     │ derives a subkey labelled for a child
   cluster: home-kb                                      ← derived, symmetric
```

- **Seed-signed certificates** answer "may this cluster speak for a root?" The
  verifier pins the root's public key and never talks to it.
- **Derived subkeys** answer "may this child speak for a cluster?" — symmetric,
  no PKI, and the child cannot mint as its parent.

`VerifyRegistration` accepts either. The ladder is unchanged: attested + registry
naming this host:port + mDNS naming the cluster is 6, attested + registry is 4,
everything else is 2. What changed is only what counts as *attested*.

## Roots

| root | identity | private key | trusted by default |
| --- | --- | --- | --- |
| public base | `seed-42` | the lab operator's; never in a repo | every trust set |
| a private cluster | its cluster id | that cluster's operator | only the hosts that pin it |

**`seed-42` is the public base seed.** Its public key is embedded in isotope
(`PublicSeedKey`), so a tool can verify a public cluster with no configuration at
all; the private half is provisioned once by the operator and never enters a
repository (see `cmd/seed42`). A trust set that resolves the public root accepts
public clusters and adds them to its own set.

**The edge is asymmetric on purpose.** A private cluster trusts a public one; the
public one does not thereby trust the private one. Publishing a key is not a
claim about who may publish back into it, and the lab's public surfaces must not
acquire a path to a stranger's cluster just because that stranger adopted
`seed-42`.

## Certificates

```json
{
  "cluster": "base-1234",
  "public_key": "base64(ed25519)",
  "issuer": "seed-42",
  "scope": ["attest", "registry"],
  "not_after": 1798761600,
  "signature": "base64(issuer signs: v1 || cluster || public_key || scope || not_after)"
}
```

- `issuer` is a root name. `issuer == cluster` is a **self-signed private root**:
  valid only where that key is pinned, which is exactly the private case.
- `scope` bounds what the cluster may attest for. Empty means `attest` only, so a
  certificate minted for a directory role cannot be replayed as an attestation
  grant.
- `not_after` is the only expiry. It is checked at verification, never trusted
  from the attestation.
- **Rotation needs both signatures.** A certificate that supersedes a key must be
  signed by the old key and the new one (same rule as
  `lezz.go/docs/attestation.md`), otherwise whoever holds the new key can claim
  to be whoever held the old one.

The chain is one hop: **root → cluster → receipt**. A cluster cannot certify
another cluster, because "can certify others" (rung 6) still has no operation
behind it — issuing rung 6 does not grant a minting capability. When it does,
depth becomes a decision, not an accident, and it belongs in this document first.

## Receipts

An attestation now carries its scheme:

| scheme | MAC | verified with | third parties can verify |
| --- | --- | --- | --- |
| `hmac-sha256` (default, `scheme` omitted) | HMAC over cluster \| endpoint \| issued_at | the cluster's secret, or a derived subkey | no — verification implies minting |
| `ed25519` | signature by the cluster's key, plus the cluster's certificate | the root the certificate chains to | yes |

Both bind cluster, endpoint and issue time, so neither can be replayed for
another endpoint, and both are bounded by the verifier's skew window.

The HMAC scheme stays for two reasons: every released producer uses it
(adhd v0.10.0, smoke-alarm v0.13.0), and it is the right shape for a **derived
subkey**, where the whole point is that the parent can compute the child's key.

## Derived subkeys

```
derived = HKDF-SHA256(secret = parent secret, info = "isotope-subkey-v1:" + child)
```

- A parent hands a child its derived key, never its own: the child registers
  receipts as `cluster: <child>` and cannot mint as the parent.
- The verifier is configured with `Secrets[child] = derived`, so revoking a child
  is dropping one entry, not rotating the parent.
- Derivation is deterministic, so a parent that has lost the child's key can
  recompute it from its own secret and the child's name. There is no other state.

This is the right tool inside one administrative domain and the wrong tool
across one: a third party cannot check a derived receipt without being given the
derived key, so it does not replace a certificate, it scopes one.

## What a verifier holds

```go
iso.TrustConfig{
    // Roots it pins by name: "seed-42" is added automatically when the embedded
    // public key is provisioned.
    Roots: map[string]string{"home": "base64(ed25519 public key)"},
    // Symmetric clusters, including derived children.
    Secrets: map[string]string{"home-kb": derived},
}
```

The asymmetry is a configuration property, not a rule in the code: a public
cluster's alarm pins `seed-42` (and gets it by default), and a private cluster's
alarm pins `seed-42` **and** its own root. Nothing stops an operator from pinning
a stranger's root — that operator has said, out loud, that they trust it. What
the code refuses is inferring that from proximity, discovery, or the registry.

## Non-goals, for now

- **No transitive trust.** A certificate chains to a pinned root, one hop. "A
  trusts B, B trusts C, therefore A trusts C" is how a lab loses control of its
  own surface.
- **No discovery of roots.** A root's key arrives out of band (embedded, or
  pinned in configuration). Reading it from mDNS or the registry would rebuild
  isotope#23 one level up.
- **No rung movement from scheme.** A certificate does not make a registration
  6 by itself; the registry and mDNS evidence still have to agree, because those
  are what say "this endpoint, here, now".
- **No key material in the repo.** `cmd/seed42` prints a keypair for the
  operator to provision; the public half may be committed, the private half
  never.
