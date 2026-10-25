# Claims, routes and tension

**Status:** draft v2, 2026-10-20. Written to be argued with, not implemented wholesale. It
records the model that emerged from the routing/skills/scrubbing work and keeps a hard line
between what is in the tree and what is still a proposal.

**Revision 2** folds in three findings: the ladder of agreements the plane rests on
(isotope#12), the direction of rung authority and the separation of drift from violation
(isotope#4), and a verifier whose comment advertises more than it checks (isotope#13). The
first draft described the attestation chain; it did not say which part of it a *node* may
write. That is the gap this revision closes, and closing it turned up three contradictions in
the code that are more useful than the prose around them (§4).

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

### The first worked example is in this repository

`VerifyIsotopeChain` (`consensus.go:143`) carries the comment:

> checks that all links in the chain are accounted for and signed

It checks three things and no signature: at least two links, `links[0].Role == "requester"`,
non-decreasing timestamps. `ChainLink.Isotope` carries a `Signature`, and the function never
reads it. A chain with forged or absent signatures passes as long as roles and clocks line
up. The contrast is in the same file: `VerifyConsensusSignatures` does verify HMACs and fails
closed when a key is missing.

The claim then travels. `adhd/internal/mcpserver/server.go` describes the same function as
requiring "strictly increasing timestamps, signed so the sequence is non-repudiable" — a
stronger claim than the code, and one hop further from it. Whoever builds an authorizer from
that paragraph builds it from a comment.

Two consequences, and they are the reason this section comes first:

1. **A docstring is a claim about code; a test is the receipt.** The fix is not a choice
   between "verify" and "reword" made on taste. Write the test that forges a signature and
   expects failure. It fails today. That failure is the receipt for the docstring, and it
   then decides the design: if the function is to authenticate, it needs the key material
   passed in and must fail closed when a key is absent.
2. **The rule has no exception for documents.** This one included. Where it cites a symbol,
   the symbol is the receipt; where it says "should", it is a claim.

## 2. What the model presupposes

42i is a stack of **increasing agreements**, each presupposing the one before:

```text
null
null exists
0 is not null
-1 < 0
0 < 1
```

The ladder is not derived. It is conceded, and naming it separates what can be checked from
what is merely assumed.

| agreement | what it buys | where the code refuses to paper over it |
|---|---|---|
| `null` exists | absence is a state worth naming | an empty charge list is an error, not an empty plane |
| `0 is not null` | absence is *no plane*, never charge 0 | `1,-1,0` and `0,1,1,2` leave the neutral plane with no admissible edge (`same` or `opposite` charge only) → `ErrDisconnected`; planes in different components → `ErrIncomparable`, never a default phase of zero |
| `-1 < 0 < 1` | integer sector labels, not a score | charges are `int`; a tensor block's `Out = qa + qb` is a label, and magnitude lives in the weight table |
| `42` | a shared zero, so universes can be compared | `ParseAnchor` accepts `42` and `42i` and rejects everything else; one anchor per run, never mixed |

Once a shared zero exists, a universe needs exactly two things to take part:

1. **internal direction** — its own axis and ordering (the encoder);
2. **rotational positioning** — where that axis sits relative to the system (charge and
   signed tensor weights).

Direction without rotation cannot be compared; rotation without direction has nothing to
rotate. A constant has a position and no orientation, which is why it is the maximally lossy
move in §9.

**What the ladder does not buy.** `42` is a choice, not a measurement. Nothing here proves
that every universe agrees where zero is; the plane *fixes* it so the rest can be compared.
If the choice is wrong, every distance downstream is wrong in the same direction, and no
downstream test can detect it — a uniform error is invisible to a relative measurement. The
only defence is to state the agreement, which is what this section is for. A model that
cannot state its presuppositions cannot be falsified, only disagreed with.

This is also where invariant 3 comes from. `0 is not null` and "missing is not agreement" are
the same rule at two scales: a plane that has no edge is disconnected rather than neutral,
and a peer that has no claim disagrees rather than assents.

## 3. Three identifier spaces, deliberately different

| identifier | derivation | collision policy | what it names |
|---|---|---|---|
| isotope address | derived from Gherkin scenario text | **shared on purpose** | a promise; many agents may claim it |
| skill id | 32-bit hash of the canonical name | checked at registration | a capability, so a trace can attribute it |
| trace id | allocated per run | unique | one execution |

The collision policy *is* the trust policy, which is why the three are listed together. Since
the address is shared it can carry no trust, and a cache key must name
`(address, claimant, epoch)`. Since the skill id is collision-checked, a trace can attribute a
capability without ambiguity. Since the trace id is unique, it is the only identifier that
can name a receipt.

Survival is the selection function: a claimant that keeps passing its scenario across ticks
and rotations earns the claim; one that never transits is dead weight; one that survives
intermittently is the interesting class.

## 4. Trust is a chain, not a score; a rung is a budget, not a reward

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

### Authority has a direction

A rung is a *tolerance budget granted by the network*, and a local node may only spend it
down. The model needs exactly two writers, and only one of them may raise:

- **`Certify(rung, attestation)`** verifies the attestation and is the **only** writer that
  may raise a rung. It has no local caller.
- **`demote(certifiedRung, distance)`** returns `certifiedRung` unless `distance` exceeds
  that rung's ceiling, in which case `0`. It never returns more than its input.
- Distance must never raise a rung. Local transitions are **monotone non-increasing**.

### What the tree does instead

Three contradictions, all checkable, and all of them vanish if the budget model is adopted:

1. **Two quantities share a name.** `VerifyTrust` grants a rung (2/4/6) from registry and
   mDNS presence; `AgentState.Position.Rung` computes a rung from accumulated distance. Only
   the first is used for gating; the second is local accounting that is not wired to
   authorization. They share a name, a range (0–6) and a table, and they answer different
   questions.
2. **The local ladder runs backwards.** `RungForDistance` is first-fit low-to-high:
   distance `0` → rung 0, `1–20` → rung 1, `181–220` → rung 6, `221+` → **rung 0**. A
   failing test therefore raises the rung, a passing test lowers it, and enough failures
   restore a clean slate. The limit is absurd in both directions: a hosted service that passes
   every test settles at `Absolute Zero`, while one that fails on every axis climbs toward
   `Certification`. `CheckRungBoundary` narrates the same move as "DEMOTED … drifting toward
   rung N+1". Meanwhile the README and adhd's read-gating treat rung 6 as *Certification*, the
   tier that reads most. Both readings cannot be true: as written, the lattice grants read
   access in proportion to demonstrated departure.
3. **A grant is stored as a measurement.** `NewAgentStateAtRung` seeds `distance` to the
   smallest value that still classifies at the requested rung, so one passing test can
   subtract its way out of a certification that was never local to give.

The 220→0 cliff is the sharpest form of (2): the agent that has drifted furthest is
classified as pristine. This is not a bug to patch with a bigger constant. The `220` ceiling
and the `+16` per-failure weights share one scale, so moving one without the other silently
moves the boundary between passing and failing.

### Presence is not trust

The granted rung's floor is 2, not 0: `VerifyTrust` returns 2 when the endpoint is not found
in the registry at all, so "unregistered" and "self-registered" are indistinguishable, and a
stranger that can cause a registration is granted a rung on the strength of a claim. Worse,
the check is a presence check, so a partition and a loss of trust return the same answer. A
node that can drop a peer's mDNS record can demote it.

Liveness therefore belongs to its own channel (§5) and the grant must name its evidence:
which registry, which epoch, which attestation. A rung granted because a record exists is not
a receipt for the claim the rung authorizes.

### Promotion is a decoy

The local promote path is a **detector, not a grant**. Contact with an address that is not in
the routing table says the caller got an address from somewhere; that is the signal, not the
authorization. Grade it by proof tier, because the tiers are the receipts:

| evidence | reading | channel |
|---|---|---|
| a route trace mentions the address | the caller was pointed at it | drift, `+16` |
| the caller addressed it directly | the caller guessed, or was given a sheet | flag for review |
| a canary receipt appears outside the honeypot | the canary was spent where it must not be | **violation**: suspend the skill, revoke the hosting instance's rung |

Attribution for revocation goes **upward** to the instance that hosted the skill, not to
whoever repeated the address. Today the decoy is detection-only and files an event; it never
authorizes anything, which is the right starting point.

## 5. Three channels, three axes

One scalar cannot carry three different kinds of fact. Split them, and give each the axis its
arithmetic deserves:

| channel | who writes | arithmetic | recoverable | axis |
|---|---|---|---|---|
| drift | local recorder | additive, graded | yes, by passing | imaginary |
| violation | peer, signed | categorical | no | negative real |
| liveness | lease TTL | presence or not | on renewal | none |

- **Drift** is a departure from nominal: self-reported, graded, recoverable. It is a working
  hypothesis, so it lives on the imaginary axis.
- **Violation** is categorical and peer-attested: it happened. It is never additive and never
  recovered by drift. It belongs on the negative real axis — away from consensus — because
  "the alarm disagreed" is a fact about agreement, not a quantity of departure.
- **Liveness** is a lease. Losing presence clears the flag; the rung is unchanged.
  Unreachability is never encoded as drift.

The rule is not "weight violations heavily". It is that **a violation is not a quantity**. If
one can be expressed as a number of drift units, it is purchasable: enough passing tests
subtract the same units back. And if unreachability is drift, a partition is a demotion, so
an attacker who can partition demotes a peer without touching it.

Sign violation and honeypot reports, and **ignore unsigned ones**. An unsigned violation is a
framing primitive and a DoS primitive: anyone who can reach the port can spend someone else's
rung.

The plane already models this channel correctly and is the reference implementation:
`QuarantineEvent` is documented as "categorical, peer-attested … a receiving side must verify
the signature and ignore unsigned events", and `Quarantine` rebuilds the tree after dropping
the frustrated edge. It also carries a hazard worth naming: `DefaultQuarantineKey()` is a
deterministic development key. A constant key makes "peer-attested" mean "attested by anyone
who read the source", which is the same failure as a constant answer — it erases the evidence
it was supposed to carry. Production callers must supply a key, and a receiver must know
*whose*.

### Chaos windows exempt drift, never violations

The drift recorder reads `isotope_classification` from smoke-alarm `/status` — never a local
clock, and no second registry: two clocks make two histories. `registered-chaos` carries
drift weight `0`; `expired-window` and `unregistered` carry the real weight. The exemption
applies to channel 1 only; a violation fires inside a chaos window as loudly as outside it,
or the window becomes a licence.

Windows are **operator-granted**, checked as an explicit capability. A node that can declare
its own chaos window can make its own failures unobservable, which is the same move as
writing its own rung.

### The debug path is not on the wire

Debug access is a separate loopback-only listener or a `//go:build debug` binary, never a
branch inside the network-facing socket. It emits no drift and does not advertise
(`_adhd-isotope._tcp`). The invariant to test is not "debug works" but "the debug path is
unreachable through the network-facing listener" — a test that would fail the day the branch
moves.

## 6. Tension is the sensor, and partiality is what makes it work

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

## 7. The tumble: positional authority and a latency floor

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

## 8. Skills are capabilities with projections

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

## 9. Provenance is a route you can walk

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

## 10. The same algebra on a real task: PII scrubbing

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
relationship. "Trusted" is a chain, not a header: the view is a function of the verified
rung, which is why §4 has to name who may write that rung before this sentence means
anything.

## 11. What exists, and what is proposed

Implemented and tested:

- `adhd/internal/navtrace` — step schema, recorder, 32-bit skill id, alias registry;
- `adhd/internal/tearsheet` — sheet, tear, ledger, file-backed rotation seed;
- `adhd/internal/trust` — attestations, views, contradiction, tension, holonomy;
- `adhd/internal/mcpserver` — route-plan skill, tear skill, spec fidelity reporting, decoy
  registration (detection only);
- `adhd/internal/headless` — chain-rung eviction; address-keyed fidelity (characterised as a
  gap);
- `lezz.go` — prod skill env, `startProcessEnv`;
- `isotope/plane` — per-kind corpus gate (request, path, skill, knock); anchor parsing;
  charge-graph disconnection and incomparability; deterministic tensor, tree and holonomy;
  signed categorical quarantine (the violation channel, already realised).

Proposed, not built:

- identity at the HTTP layer so `ChainRung` is real, and routing gated on it;
- the rung authority split: `Certify` as the sole raiser, `demote` as the only local writer,
  and a single meaning for "rung" across `VerifyTrust` and `AgentState`;
- the three channels as three types, so a violation cannot be written as drift;
- a transport for `internal/trust` (gRPC or otherwise) and a peer set policy;
- the tumble schedule, receipts and the latency floor;
- cross-node trace propagation for recursive skills;
- the universe generator and held-out splits;
- payload opacity for hops (a hop currently reads the body it forwards);
- operator-granted chaos windows and the loopback-only debug listener;
- a key policy for quarantine signing that replaces `DefaultQuarantineKey` in production.

Known gaps that are bugs rather than proposals, filed separately:

- `VerifyIsotopeChain` advertises signatures it does not check (isotope#13);
- the rung ladder inverts capability and drift, and the 220 ceiling resets the worst agent
  to rung 0 (isotope#15);
- `VerifyTrust`'s floor of 2 grants a rung on presence, and cannot distinguish a partition
  from a stranger (isotope#16);
- the one function that does verify HMACs panics on a short signature and compares in
  non-constant time (isotope#17).

## 12. Invariants worth keeping

*About evidence*

1. Address is a claim; receipt is evidence.
2. A docstring is a claim about code; a test is the receipt.
3. Missing is not agreement.
4. Report contradictions; never resolve them silently.

*About trust*

5. Gate and rank on the chain; report the score.
6. Local rung transitions are monotone non-increasing; `Certify` is the only raiser.
7. Presence is a lease, not a rung; losing a lease never demotes.
8. Unsigned violations are ignored, never honored.
9. Chaos windows exempt drift, never violations, and a node cannot declare its own.

*About routes*

10. Provenance must be walkable.
11. Latency is bounded below by the attested hops.
12. Single-use tickets bound exposure; they do not prevent it.
13. A debug path emits no drift and is not advertised.

*About measurement*

14. Verify on state or trace, not on the answer.
15. Never let the improver be the examinee.
16. Encoding is deterministic: no map iteration in any encoder, fixed accumulation order.

## 13. Open questions

- **What is rung 0?** `VerifyTrust` grants 2 to anything it cannot find (isotope#16), so the
  bottom of the lattice is a participation trophy. Either 0 must mean *no receipt presented*
  rather than *not found*, or the lattice has no bottom and its lowest tier needs another
  name.
- **What does a passing test buy?** Under the budget model, a pass may restore drift; it may
  not restore a rung. That means recovery and promotion are separate events, and the
  existing call sites that return `rungChanged` are answering the wrong question.
- Is the routing floor the *target's* rung or the *path's* minimum? Both are defensible; the
  second is stricter and makes a low-rung hop a route downgrade.
- Should a hop see the body at all? Opaque forwarding changes the trust model from "trusted
  hops" to "trusted routing", and costs the hop any ability to cache or transform.
- Where does the tumble schedule live — the registry, the isotope definition, or 42i?
- Does a shared isotope address need a per-claimant nonce in the address itself, or is the
  receipt enough?
- For the universe: is the hydration log an authority (like 42i) or a receipt held by the
  examiner?

---

In its own terms this document is a **claim**. Its receipts are the symbols it names and the
tests that would fail if it were wrong; where it says "should", it is an intention, and where
it says "today", it is a reading of the tree on 2026-10-20.
