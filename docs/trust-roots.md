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

| root | identity | private key | trusted where |
| --- | --- | --- | --- |
| public base | `seed-42` | the lab operator's; never in a repo | the trust sets that *pin its key* |
| a private cluster | its cluster id | that cluster's operator | only the hosts that pin it |

**Nothing is embedded.** `seed-42` is a name, not a key in the binary. A cluster
launched offline must have exactly the same trust as one launched online, and an
embedded root would make "trusted by seed-42" a property of the build rather than
of what the operator pinned. So the public key travels with the seed — in
`seed.json`, or in the alarm config a cluster generates — and verification never
reaches the network for it.

The consequence is deliberate: a trust set that does not carry seed-42's key
accepts no public cluster, and one that carries it accepts them all. "Private
clusters trust public clusters" is therefore a statement about what a private
seed pins, not a default the library supplies.

### Where a root lives: seed.json

A cluster is a workspace of workspaces serviced by clusters, and `seed.json` is
where a workspace says which roots it accepts. The pin travels with the project
and with the cluster it generates, so:

- a laptop with no network trusts its own cluster and any public cluster whose
  root is in its seed, and refuses everything else — the same set it would trust
  online;
- a public surface trusts only what its own seed pins, so adopting `seed-42`
  somewhere does not make a stranger's private cluster admissible there;
- re-pinning a root is a `seed.json` change, reviewable in the same diff as the
  rest of the project's placement.

Nothing about the network, the registry, or discovery participates in choosing a
root.

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

## A person is a root too

A GitHub account publishes its SSH keys as `ssh-ed25519` lines at
`https://github.com/<user>.keys`, and an ed25519 SSH key is an ed25519 key. So a
person's published key pins as a root directly (`ParseSSHEd25519`,
`ParseKeyList`), with no certificate authority, no seed and no conversion step.

That makes the shortest honest test of this whole document a note and a key:

```sh
# once, on the machine that holds the private key
ssh-keygen -Y sign -n lezz-note notes/today.md      # writes notes/today.md.sig

# anywhere, offline, with only the published key
curl -s https://github.com/james-gibson.keys > allowed_signers.body
printf '%s %s\n' "github:james-gibson" "$(grep -m1 '^ssh-ed25519 ' allowed_signers.body | cut -d' ' -f1,2)" > allowed_signers
ssh-keygen -Y verify -f allowed_signers -I github:james-gibson -n lezz-note -s notes/today.md.sig < notes/today.md
```

`Good "lezz-note" signature for github:james-gibson` is the whole trust model in
one line: a root the operator pinned, an artifact it signed, and a decision made
without the network. The same key can issue a cluster certificate
(`SignCertificate(cert, personKey)`), so "my notes" and "my cluster" chain to one
identity — which is what makes a personal root useful rather than decorative.

`safe`: publishing a key is not a claim about what it may do. Pinning it is.

### The burned vector

For a result anyone can reproduce without asking for a key, `testdata/burned/`
commits a **deliberately burned** keypair owned by Zaphod Beeblebrox: private half
included, comment says `DO NOT USE`, and nothing it signs is worth anything. A
burned key is one everybody has, which makes him the least trustworthy owner
available — and verification does not care who signed, only what the signature
proves. The committed `note.md` + `note.md.sig` verify with the committed public
half, offline:

```sh
cd testdata/burned
ssh-keygen -Y verify -f allowed_signers -I zaphod-beeblebrox -n lezz-note -s note.md.sig < note.md
```

`go test` asserts that vector, that the private half really is the published key's
pair, and that the same key roots a cluster whose member's receipt verifies. A test
that needs a signature that *should* be refused generates a fresh key instead — the
burned one is for the positive path, so its fingerprint is asserted rather than
regenerated.

## What a verifier holds

```go
iso.TrustConfig{
    // Every root it accepts, named. Nothing is added for it: no entry, no trust.
    Roots: map[string]string{
        "seed-42": "base64(ed25519 public key)", // public clusters
        "home":    "base64(ed25519 public key)", // this private root
    },
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
- **No discovery of roots, and no embedded ones.** A root's key arrives with the
  seed: pinned in `seed.json` or in the configuration a cluster generated. Reading
  it from mDNS or the registry would rebuild isotope#23 one level up; baking it
  into the binary would make offline and online clusters trust different things.
- **No rung movement from scheme.** A certificate does not make a registration
  6 by itself; the registry and mDNS evidence still have to agree, because those
  are what say "this endpoint, here, now".
- **No key material in the repo, and no public key in the library either.**
  `cmd/seed42` prints a keypair for the operator to provision; the public half
  goes into the seeds and cluster configs that should accept it, the private half
  never leaves the operator's store.
