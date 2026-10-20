# Claims, routes and tension

**Status:** draft, 2026-10-20. Written to be argued with, not implemented wholesale. It
records the model that emerged from the routing/skills/scrubbing work and separates what is
already in the tree from what is still a proposal.

---

## 1. The one distinction everything rests on

**An advertised thing is a claim. A receipt is evidence.**

An isotope address says "invoke me and I'll fetch something". A tool description says
"counts the letter r". A cached answer says "the count is 3". A proxy chain says "this is
the way". None of these is evidence, and the honeypot is not an anomaly in the model — it is
the model: a perfectly formed claim with nothing behind it.

The design consequence is that no check may be made against the claim. It must be made
against:

- the **receipt** for identity (nonce → rung, signed by the party that vouches);
- the **route** for provenance (a path that can be re-walked, not a narrative);
- the **state** for correctness (the trace, not the answer).

## 2. Three identifier spaces, deliberately different

| identifier | derivation | collision policy | what it names |
|---|---|---|---|
| isotope address | derived from Gherkin scenario text | **shared on purpose** | a promise; many agents may claim it |
| skill id | 32-bit hash of the canonical name | checked at registration | a capability, so a trace can attribute it |
| trace id | allocated per run | unique | one execution |

The address is a namespace, not a trust boundary. Because several claimants may hold it, the
address carries zero trust by design; everything must be verified per claimant. This is also
why a cache key can never be the address alone — it is `(address, claimant, epoch)`.

Survival is the selection function: a claimant that keeps passing its scenario across ticks
and rotations earns the claim; one that never transits is dead weight; one that survives
intermittently is the interesting class.

## 3. Trust is a chain, not a score

Request-shape conformance (`ScoreRequest`, 0–100) describes manners. It is reported, ranked
nowhere, and gated on never. Trust comes from the attestation chain:

```
attestation = {subject, issuer, rung, epoch, receipt}
```

- a claim is accepted only if its receipt verifies;
- a newer epoch replaces; an older is ignored;
- a **same-epoch disagreement is recorded as a contradiction and reported**, never resolved.

That last rule is the plane's holonomy rule applied to trust: a contradiction is evidence,
not noise to smooth.

**Current state of the code:** the rolling request score is keyed by network address
(`internal/headless/server.go`), so it is bound to a claim that can change hands. Eviction
already ranks on `ChainRung` (chain-verified rung, 0 = unverified) rather than on score, and
a flood of perfect-scoring strangers cannot evict a chain-verified peer. But in production
every entry is rung 0, because the middleware cannot see a chain yet. Until identity reaches
the HTTP layer, ranked eviction degenerates to age — safe, and not yet what it claims to be.

## 4. Tension is the sensor, and partiality is what makes it work

A fully connected trust graph converges before disagreement can be observed. A sparse one —
a few peers per node — exposes partition, propagation delay and a node that stopped
vouching. A cycle localises a liar to the two edges touching it; a tree cannot.

`internal/trust` implements the pure part:

- `View.Merge(claim, verify)` — verified, epoch-ordered, contradictions kept;
- `Compare(a, b)` — disagreement over the union of subjects; **a missing claim is a
  disagreement**, because missing is not agreement;
- `Holonomy(views)` — one reading per edge plus the closing edge.

The transport is deliberately not decided. A gRPC service (`Share`, heartbeat) is the likely
shape: mutual identity on the link, bounded tick-aligned batches, drop rather than queue.
The link authenticates *who is speaking*; the attestation still authenticates *what is
claimed*.

## 5. The tumble: positional authority and a latency floor

The isotope rotates between nodes. At epoch *e* one node holds it, and answering requires
holding it. This is what makes a constant answer structurally impossible: the entry node
does not hold the isotope, so it cannot legitimately answer without a rotation receipt. It
is `path_stale` in a different costume.

Two invariants follow:

1. **Completeness** — the receipt chain covers every hop of the schedule; the composed
   rotation is the identity (zero holonomy).
2. **Latency floor** — `reported ≥ Σ hop_latency`. A total below the sum of its own attested
   hops is a contradiction: the rotation was skipped, not fast.

Measure time to a **verified tumble**, never time to first byte. First byte is the honeypot
metric; a constant maximises it.

## 6. Skills are capabilities with projections

Visibility is not a property of a skill, it is a projection chosen at the edge:

| projection | lifetime | example |
|---|---|---|
| ambient | the generation's processes | tear-ledger seed via transient env |
| visible | the tick's surface | `adhd.skill.proxy-hop-tear.call`, `adhd.skill.route-plan.call` |
| in-flight | one interaction | a question or an answer mid-trajectory |

The same capability may be ambient in one generation and visible in another, and both at
once. `internal/navtrace` records the projection that was actually used: `shown` vs `acted`,
`stop` reasons, a 32-bit `skill` id, and argument **keys** but never values. A non-action is
a data point, not a missing row.

Two implemented skills anchor the model:

- **route plan** — resolves a target to one callable hop chain, cached per tick;
- **proxy-hop tear sheet** — tears one hop off a sheet, spends a single-use nonce, forwards,
  returns the remainder. Replay is refused `-32000` + `skill_reuse`; expiry is `-32010` +
  `path_stale`.

The tear ledger is file-backed and atomically written, so the child of a rotation opens the
ledger its parent wrote (`ADHD_TEAR_LEDGER_PATH`, seeded by `prodSkillsFor` in lezz). A
reload-on-miss closes the handoff window without spending an extra tick.

## 7. Provenance is a route you can walk

A supplied answer implies a route. A protocol run produces another. The comparison is the
product:

- **same output, cycle composes to identity** — genuinely equivalent;
- **same output, cycle does not** — the supplied route skipped something;
- **same reading under a lossy projection** — observationally equivalent only.

A constant is the maximally lossy move: it makes every route equivalent and therefore cannot
verify anything. Lossy reads bound what can be detected: a one-bit predicate distinguishes
two classes; a constant distinguishes none.

Rule: **invertible moves preserve holonomy, lossy moves hide it, constants erase it.** Verify
on the state or a trace that retains it, never on the answer alone.

## 8. The same algebra on a real task: PII scrubbing

Scrubbing is a projection, so it has checkable laws:

| law | statement |
|---|---|
| idempotence | `S(S(x)) = S(x)` |
| residual | independent detector finds nothing |
| utility | non-PII content preserved |
| irreversibility | no recovery of removed spans outside the vault |
| stability | same input + policy → same spans |

Residual and utility are two-sided: a constant maximises one and destroys the other, so
report the pair, never a blend. The output is the lossy read; the **span set** is the state.

A hydrated universe closes the oracle gap: if the environment is generated from a universe
definition, the PII is known by construction. The join of hydration record and scrub trace
gives exact per-entity recall, planted near-misses for over-scrub, and a coverage grid by
class and rule. Two guardrails: the scrubber must not see the universe, and the survival test
must use a held-out universe or it measures the generator.

Selective disclosure is the same projection read the other way: **a view function over one
source, not a second copy.** Redaction, tokenisation plus a vault, or encryption — chosen per
relationship. "Trusted" is a chain, not a header: the view is a function of the verified rung.

## 9. What exists, and what is proposed

Implemented and tested:

- `adhd/internal/navtrace` — step schema, recorder, 32-bit skill id, alias registry;
- `adhd/internal/tearsheet` — sheet, tear, ledger, file-backed rotation seed;
- `adhd/internal/trust` — attestations, views, contradiction, tension, holonomy;
- `adhd/internal/mcpserver` — route-plan skill, tear skill, spec fidelity reporting;
- `adhd/internal/headless` — chain-rung eviction; address-keyed fidelity (characterised as a gap);
- `lezz.go` — prod skill env, `startProcessEnv`;
- `isotope/plane` — per-kind corpus gate (request, path, skill, knock).

Proposed, not built:

- identity at the HTTP layer so `ChainRung` is real, and routing gated on it;
- a transport for `internal/trust` (gRPC or otherwise) and a peer set policy;
- the tumble schedule, receipts and the latency floor;
- cross-node trace propagation for recursive skills;
- the universe generator and held-out splits;
- payload opacity for hops (a hop currently reads the body it forwards).

## 10. Invariants worth keeping

1. Address is a claim; receipt is evidence.
2. Missing is not agreement.
3. Report contradictions; never resolve them silently.
4. Gate and rank on the chain; report the score.
5. Provenance must be walkable.
6. Verify on state or trace, not on the answer.
7. Latency is bounded below by the attested hops.
8. Single-use tickets bound exposure; they do not prevent it.
9. Never let the improver be the examinee.

## 11. Open questions

- Is the routing floor the *target's* rung or the *path's* minimum? Both are defensible; the
  second is stricter and makes a low-rung hop a route downgrade.
- Should a hop see the body at all? Opaque forwarding changes the trust model from "trusted
  hops" to "trusted routing", and costs the hop any ability to cache or transform.
- Where does the tumble schedule live — the registry, the isotope definition, or 42i?
- Does a shared isotope address need a per-claimant nonce in the address itself, or is the
  receipt enough?
- For the universe: is the hydration log an authority (like 42i) or a receipt held by the
  examiner?
