# ADR 0004 — the signing key is a declared input with a named source

**Status:** proposed, 2026-10-21
**Amends:** `canonicalization.go:273-286` (`GenerateIsotope`), `isotope.String()` `:268-270`
**Related:** `lezz.go/docs/adr/0001-isotope-tag-challenge-receipt.md`, `isotope/plane/key.go`

## Context

`GenerateIsotope` takes a caller-supplied `signingKey []byte` and produces
`HMAC-SHA256(canonical)`; `Isotope.String()` renders `family-vN:` plus the first 16 hex
characters (64 bits) of that HMAC. The store keeps the full 256-bit hex
(`signature.hex`); only the display token truncates.

The key is therefore a real input, and the current constant is a declared default, not
the design:

- `brine.json` pins `"signing_key": "seed-42"`;
- `.brine.state.json` records the key the store was built with;
- `brine_audit.py:175` recomputes every signature and compares with `hmac.compare_digest`;
- the audit's own docstring (`brine_audit.py:29`) already scopes it honestly — *"HMAC
  with a public constant (`seed-42`), so it detects corruption, not forgery"*.

So the canon does not overclaim; a consumer did (`lezz.go/docs/attestation.md:894`:
"a recertification cannot be forged").

Production intent is that the key comes from the `42i` plane structure rather than from
a constant. That makes the choice of plane quantity load-bearing, because
`plane.Key(n)` is **gauge-covariant**: `plane/key.go:112-127` states it transforms as
`exp(i·q_root·θ)` under a global rotation, so only *ratios and differences* are
gauge-invariant. Signing with the raw key gives the same spec a different token in a
rotated frame — and `lezz.go/docs/attestation.md` §*The weights are not portable* holds
that the relation between two corpora **is** a rotation.

## Decision

**The key source is a declared, recorded input. The key material is never recorded.**

| source | token stable across peers | who can forge | verdict |
| --- | --- | --- | --- |
| public constant | yes | anyone | dev only; detects corruption |
| gauge-invariant scalar (ratio to root, distance, holonomy) | **yes, within a gauge class** | any member of the class | **default candidate** |
| raw `Model.Key` / observed phase | **no** — frame-dependent | — | **forbidden** |
| fresh session rotation (group/KDF) | no | nobody | a receipt, not an isotope (see below) |

1. **Default candidate: a gauge-invariant quantity, quotienting the root phase.** It
   preserves the property the isotope exists for — two peers with the same declared
   spec produce the same token — while being unforgeable to anyone outside the class.
   The forgeability boundary becomes class membership, which is the correct unit:
   `lab-safety/lab/TRIPWIRE.md` already holds that a break-in re-keys the *neighbours*,
   not the world.

2. **Forbidden: the tick or the observed phase.** A rotation derived from a public grid
   is public by construction. `lezz.go/docs/attestation.md` §*Two rotations* already
   states this rule; this ADR makes it binding on the key schedule rather than on the
   session rotation alone.

3. **Fresh per-session keys are a different artifact.** They give per-node
   unforgeability and destroy stability: the same spec yields a new token every run,
   which ends comparability, communities, and "what changed since the fire". If that is
   wanted, the result is a **receipt**, not an isotope — the two must not share a name.

4. **Record the key id, never the key.** A store must say *which source and which class*
   signed a record, so a verifier can tell "same key" from "different key" without being
   handed key material. `brine.json`'s `signing_key` field becomes `key_id` +
   `key_source` (see `kb/projects/scripts/docs/adr/0001-...`).

## Consequences

| property | constant key | gauge-invariant class key | session key |
| --- | --- | --- | --- |
| two peers, same spec, same token | yes | yes, same class | **no** |
| forgeable by a stranger | yes | no | no |
| forgeable by a class member | yes | **yes** | no |
| usable as a community tag | yes | yes | no |
| usable as a per-node credential | no | no | yes |

The middle column is the one the protocol wants: a keyed community tag. It is still not
a node identity, and a consumer must not read it as one.

## Known limits

- **A class key is a mirror, not a witness, inside the class.** Any member can mint any
  token for any spec, so two members agreeing on a token proves shared membership, not
  independent observation — the same limit `TRIPWIRE.md` records one layer up.
- **Revocation granularity is the class.** Removing one member means re-keying the class
  or accepting that the removed member can still mint valid-looking tags for it.
- **The wire token is 64 bits.** Single-forgery probability is `2^-64` per attempt
  against a real key, which is sound for a tag and thin for a certificate; the store's
  256-bit hex must remain the auditing artifact.
- **`String()` truncation is a presentation choice that has been read as the security
  parameter.** Nothing in the format says the token is the evidence.
- **Key id is not proof of possession.** A record naming a class says who *should* have
  signed it, not who did; only the HMAC over the raw does that, and only within the
  class.

## What would reverse this

A requirement that a token be unforgeable *per node* — e.g. a receipt that gates
admission on its own. That forces the session-key row, and therefore forces the isotope
to stop being stable. Accepting that trade is a decision for
`lezz.go/docs/adr/0001-isotope-tag-challenge-receipt.md`, not for this one.
