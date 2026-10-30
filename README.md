# isotope — Trust Protocol Library

`github.com/james-gibson/isotope`

A shared Go library implementing the **isotope protocol**: a framework for verifying the trustworthiness and capability of agents in the prototype lab before allowing them to act.

Not a binary. All lab tools depend on it.

---

## Trust after connection

**A caller can connect to an MCP service; whether it is trusted after connecting is
a separate question, and one most transport specifications do not answer.** This lab
measures it. A caller carries a **42i distance** — the extent to which it has
departed from **nominal** (the service level a call promises) against **linear time**
(the shared tick) — and the distance maps to a **rung** that limits what the caller
may read. A caller that keeps departing is removed from the routeable set.

```
 nominal ●────────────  the level a call promises
          ╲
           ╲ distance     +8 / +16 / +32 …
            ●───────────  where it landed (linear time →)
```

One measure, regardless of source — a stale credential, a clock skew, a tampered
receipt, a honeypot contact. Model and references: `james-gibson/lab-safety` →
`lab/SPLASH.md`, `lab/PLANE.md`, `lab/MATH.md`.

## What It Does

The isotope protocol answers one question — **how far has this agent drifted from the seed-42 ground truth?** — and expresses the answer as a *42i distance* that maps to a capability rung.

### The 42i Capability Lattice

Ground truth lives at `42 + 0i` (seed-42). Each test failure moves an agent along the imaginary axis, weighted by how fundamental the failure is. The magnitude `sqrt(42² + distance²)` gives a geometric picture of where the agent sits in capability space.

| Rung | Name             | Max distance | What it means                              |
|------|------------------|--------------|--------------------------------------------|
| 0    | Absolute Zero    | 0            | Loads without crashing                     |
| 1    | Read-Only        | 20           | Entropy and isotope variation pass         |
| 2    | Harness Tools    | 60           | Known tool scope, input correlation        |
| 3    | Mock Secrets     | 100          | Safe isotope handling, declared behaviour  |
| 4    | Higher Authority | 140          | Independent decisions, timing uncorrelated |
| 5    | Delegation       | 180          | Can delegate to peers                      |
| 6    | Certification    | 220          | Can certify others                         |

### Trust Verification

A rung is granted from **evidence**, not from presence. A registrant presents a
cluster *attestation* — an HMAC receipt over the cluster identity and its own
endpoint, signed with the cluster's launch secret — and only an attested
registration can rise above the floor:

| Evidence | Rung |
|----------|------|
| Attested, registry names this host:port, mDNS confirms the cluster | 6 — Certification |
| Attested and registry-confirmed, mDNS did not confirm | 4 — Higher Authority |
| Unattested, wrong secret, unknown cluster, or no matching registry entry | 2 — Harness Tools |

Attestation comes in two schemes, and a verifier accepts either:

| scheme | proof | who can verify |
| --- | --- | --- |
| `hmac-sha256` | shared secret (or a **derived subkey** handed to a child cluster) | only holders of that secret — verification implies the ability to mint |
| `ed25519` | the cluster's signature plus its **certificate** | any verifier that pins the issuing root |

The public base seed is **`seed-42`** — a name, not a key compiled into isotope. A
cluster pins the roots it accepts (in `seed.json`, or in the config it generates),
so one launched offline trusts exactly what one launched online does: seed-42's key
travels with the seed, never the build. A private trust set accepts public clusters
by pinning it; the public side does not thereby trust the private one. Certificates carry a scope and
an optional expiry, chain one hop (root → cluster → receipt), and a self-signed
private root is valid only where its key is pinned. See
[docs/trust-roots.md](docs/trust-roots.md) for the model, the reason each edge is
directional, and why a root's key is provisioned out of band rather than
discovered.

```go
// Ed25519: a certificate under a pinned root, and a receipt the cluster signs.
cert, _ := isotope.SignCertificate(isotope.Certificate{
    Cluster: "base-1", PublicKey: pub, Issuer: isotope.PublicSeedCluster,
}, rootKey)
att := isotope.SignReceiptEd25519(clusterKey, cert, endpoint, time.Now())

verdict := isotope.VerifyRegistration(ctx, isotope.TrustConfig{
    Roots: map[string]string{
        "seed-42": publicRootPub, // pinning it is what admits public clusters
        "home":    privateRootPub,
    },
}, isotope.Registration{Endpoint: endpoint, Attestation: &att})

// Symmetric: a parent scopes a child without handing over its own secret.
child := isotope.DeriveClusterSecret(parentSecret, "home-kb")
```

`cmd/seed42` provisions the public root (`seed42 init`), certifies a public
cluster (`seed42 certify -cluster base-1`), and signs a seed bundle
(`seed42 sign -in seed.json.enc`, verified offline with `seed42 verify`). The
private half comes from `$SEED42_PRIVATE_KEY` or `-signer file:PATH` and is never
committed; a hardware slot fits the same `crypto.Signer` seam.
| No registration | 0 — Absolute Zero |

An mDNS miss costs the top rung and nothing else: a rolling restart looks exactly
like that, so absence is a lease, never a revocation (isotope#23).

`VerifyTrust(ctx, endpoint)` still exists for callers with no attestation to
offer; it can only reach rung 2, because a registry that anyone can write and an
advertisement that anyone can publish are not evidence. Use
`VerifyRegistration(ctx, TrustConfig{Secrets: …}, Registration{Attestation: …})`
to verify attested registrations.

### Attestation

The cluster's operator mints one secret when the cluster starts and hands it to
the processes it trusts — in a lezz demo, to the smoke-alarm it runs and to the
agents it spawns. Neither side sends the secret; the registrant sends a receipt:

```go
att := isotope.SignAttestation(secret, "demo-1234", "http://127.0.0.1:60099/mcp", time.Now())
// POST /isotope/register {"endpoint": "...", "attestation": att}

verdict := isotope.VerifyRegistration(ctx, isotope.TrustConfig{
    Secrets: map[string]string{"demo-1234": secret},
}, isotope.Registration{Endpoint: endpoint, Attestation: &att})
fmt.Println(verdict.Rung, verdict.Reasons)
```

The MAC binds the cluster, the endpoint and the issue time, so a receipt cannot
be replayed for another endpoint, and a receipt for one cluster says nothing
about another. Endpoints are matched by **host:port**; an endpoint bound to all
interfaces (`:9090`, `0.0.0.0:9090`) matches any host, because that is what the
bind means.

---

## API Surface

### Gherkin Canonicalization

```go
// Parse and canonicalize Gherkin text to a signed isotope token
isotope, err := isotope.CanonicalizeGherkinToIsotope(gherkinText, signingKey, "iso-login", 1)
fmt.Println(isotope.String()) // "iso-login-v1:a3f2b1c4d5e6f7a8"
```

### Byzantine Consensus

```go
reporter := isotope.NewConsensusReporter(agentID, quorumSize)
reporter.RecordResult(result)
consensus, ok := reporter.Consensus()
```

### 42i Agent State

```go
// Start a fresh agent
state := isotope.NewAgentState(agentID)

// Or pre-position at a verified rung
state := isotope.NewAgentStateAtRung(agentID, 4)

// Record a test failure (increases 42i distance)
state.RecordTestFailure(testID, isotope.FailureEntropyCollapse)
fmt.Println(state.Position.Rung) // current rung
```

### Trust Verification

```go
// Unattested: always the floor, whatever the registry says.
isotope.VerifyTrust(ctx, "http://localhost:8088") // 2

// Attested: 2, 4 or 6, with the reasons behind the verdict.
att := isotope.SignAttestation(secret, cluster, endpoint, time.Now())
verdict := isotope.VerifyRegistration(ctx, isotope.TrustConfig{
    Secrets: map[string]string{cluster: secret},
}, isotope.Registration{Endpoint: endpoint, Attestation: &att})
```

### REST Client (smoke-alarm isotope endpoints)

```go
client := isotope.NewClient("http://localhost:8088")

// Register as an isotope
record, err := client.Register(ctx, isotope.Registration{
    Name:     "adhd",
    Role:     "prime",
    Endpoint: "http://localhost:9090",
    Protocol: "mcp",
})
fmt.Println(record.TrustRung, record.RungName)

// List all registered isotopes
records, err := client.List(ctx)
```

### mDNS Advertisement

```go
srv, err := isotope.Advertise("adhd", "_adhd-isotope._tcp", ":9090", "prime", 4)
defer srv.Shutdown()
```

---

## The `plane` Command

`cmd/plane` is a one-shot deriver that turns a charge assignment into a
U(1)-covariant, block-diagonal edge tensor, a spanning-tree transport over the
planes (never a star), and a three-point sort functional with holonomy and a
quarantine closing move. There is no hyperparameter search.

Build a single static binary:

```sh
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o plane ./cmd/plane
```

```
plane gen   --charges 0,1,1,2 --corpus corpus.txt --anchor 42 --out weights.json
plane node  <string>            # → {zeta:[...], key:...}
plane edge  <a> <b>             # → {blocks:[...], score, phase}
plane triple <a> <b> <c>        # → {key, holonomy, frustrated, quarantined}
plane sort                      # stdin, one plane selector per line; --anchor required
```

`gen` writes `weights.json`; the default is embedded into the binary via
`go:embed` (`plane/weights.json`), so the other subcommands run with no external
files. Exactly one anchor is fixed per run — `42` (real axis, proven consensus)
or `42i` (imaginary axis, working hypothesis) — and anchors are never mixed
within a sort. A disconnected charge graph returns an error rather than a
default phase of 0.

### §12 decisions (recorded)

| Config | Decision |
|---|---|
| **Charges** | Default embedded model uses `1,1,-1` (a connected charge graph). `0,1,1,2` is intentionally disconnected and exercises `ErrDisconnected`. |
| **Edge weight source** | Inverse tensor norm `1/‖T_q‖`, so routing prefers proven / high-rung planes. |
| **Quarantine attribution** | **Flag all three suspects** — the event records both endpoints of the frustrated non-tree edge *and* their lowest common ancestor. |
| **Anchor default** | `42` (proven consensus). `sort` requires `--anchor` to be passed explicitly; other subcommands default to `42`. |

### Interpretation note

The tensor, tree, transport, holonomy, and three-point key are all defined over
**planes**. A string encodes to a `Node`: a centered coordinate vector with one
complex value per plane. The tensor score compares two `Node`s; the
tree/transport/holonomy operate on planes for a given coordinate vector (the CLI
calls it the probe node, selected with `--node`, default `"42"`). The `edge`,
`triple`, and `sort` subcommands therefore take **plane selectors** (an index or
a plane name) rather than raw strings.

Determinism is non-negotiable: no map iteration in the encoder or tree builder,
stable tie-breaks via `slices.SortStableFunc`/`cmp.Compare`, fixed accumulation
order (runes, then planes; edges by index), cached transports, and bit-identical
output across runs.

---

## Result Ledger

The `ResultLedger` tracks test history, flakiness, regressions, and refactoring effectiveness across consensus runs:

```go
ledger := isotope.NewResultLedger()
ledger.Record(result)
summary := ledger.Summary()
```

---

## Installation

```sh
go get github.com/james-gibson/isotope
```

---

## In the Lab

isotope is the shared library that all lab tools depend on. See [lab-safety](https://github.com/james-gibson/lab-safety) for a full map of how the tools connect.

Tools that use isotope:
- **ocd-smoke-alarm** — serves `/isotope/register` and `/isotope/list`; calls
  `VerifyRegistration` on each registrant with the cluster secrets it was started
  with
- **adhd** — registers itself on startup; calls `Advertise` to broadcast `_adhd-isotope._tcp`
- **lezz.go** — calls `Client.List` to include trust levels in the demo cluster summary

---

## See Also

- [lab-safety — full ecosystem overview](https://github.com/james-gibson/lab-safety)
- [ocd-smoke-alarm](https://github.com/james-gibson/smoke-alarm)
- [adhd dashboard](https://github.com/james-gibson/adhd)
- [pickled-onions seeds](https://github.com/james-gibson/pickled-onions) — Gherkin world-builder that feeds the canonicalization pipeline
