# Mapping lore / local-time onto `plane`

**Status:** revised 2026-10-12. The earlier version of this doc put lore semantics into
`plane`'s integer charges; that was the wrong slot and did not work. This version reflects
what `plane` actually computes and where lore belongs in it.

---

## 1. The finding that reorganised everything

`plane` is **structure with stub numbers**. It accepts exactly one external numeric input —
a `--weights` file — and every other number in it is closed-form arithmetic on indices:

```go
// DefaultParams: "derived deterministically from the charge list and anchor;
//  nothing is searched or learned by gradient descent in this package."
Freq[j]  = 1 + 0.25*q  + 0.1*j            // + q, j
Phase[j] = 0.3*q  + 0.37*j + 0.011*base
Gain[j]  = base + 5*|q| + j               // → 47,48,49,50
Decay    = 0.95
base     = |anchor|; if base == 0 { base = 42 }    // ← hardcoded 42

// blockWeight
theta = π*(qa-qb)/(n+1) + 0.017*(i-k)
amp   = 1/(1 + |qa| + |qb|)
```

The prototype's `fortytwointegration.go` supplied a hand-guessed `DefaultWeights` table to
paper over this. **The real code has no weights, so lore supplies them.**

Two consequences:

- Lore does not need to change `plane`'s integer charges. Charges are a discrete symmetry
  label, and they are integers because `Out == 0` is the *invariance condition*
  (`TestEquivariance` holds only when the phases cancel, which forces `q_a + q_b == 0`).
  That is a structural constraint, not a preference.
- The semantic content goes in the **weight slots** — `encoder` params per plane and the
  signed `tensor` block weights — which are floats, already signed, and are the only place
  meaning can live.

## 2. The hook already exists

No code change is needed to run lore-supplied weights:

```
plane gen --charges <list>             → writes weights.json (stub numbers)
   edit weights.json: "encoder" + "tensor" blocks from lore
plane {sort,edge,triple} --weights weights.json   → LoadWeights uses them
```

Verified load-bearing **for `edge` and `Score` only — NOT for `sort`**: flipping one
`out == 0` block weight changes what `Edge` reports, and `Tensor.Score` sums those blocks.
But **`sort` never reads the weights at all**, so a lore-weighted ranking cannot be built on
it. See the "`sort` is weight-invariant" finding below.

`plane sort` is a **charge-and-probe** function:

- `admissibleEdges` (`plane/tree.go:211`) computes each edge weight as
  `1 / tensor.Norm(qj)` — a function of the **charge alone**, never of the pair `(j,k)`. The
  per-pair `blk.W` entries where lore would go are not consulted.
- `Compare`/`Distance`/`Triple` (`plane/key.go`) read `edgeAngle` (`plane/tree.go:187`), which
  is `arg(z_j) ± arg(z_k)` — charges and probe phases only. No tensor reference anywhere in
  `plane/key.go` or `plane/holonomy.go`.

Verified by holding the probe fixed and varying the weights: stub, all-weights-scaled-×10,
scaled-×0.1, a single entry raised to 100, raised to 100000, and **an all-zero tensor** all
produce the *identical* order. Changing the probe changes the order. So the sort is a function
of `(charges, probe)` and nothing else.

**Consequence for lore:** lore weights reach two different surfaces, and they are not
interchangeable:

- **The `encoder` block** (`freq`, `phase`, `gain`, `decay`) is the weight surface **`sort`
  actually reads**. Verified: holding the probe fixed, `gain ×10`, `freq` reversed, and
  `decay 0.5` each change the returned `Order`; only `phase +1.0` did not. This is the surface
  the mapping below uses, because `EncoderParams` produce `zeta` and `Distance`/`Compare` are
  computed from `zeta`.
- **The `tensor` blocks** are read by `Edge` and `Tensor.Score`, never by `sort`. `Score`
  couples only `Out == 0` pairs — cross-sector under the ±1 assignment — while the corpus's
  declared edges are almost entirely within-sector (film→film via `THEMATICALLY_LINKED`). Only
  Idiocracy's edges to real compendia would score at all.

An earlier version of this doc claimed the tensor was the load-bearing surface for the sort.
That was a misreading — two different probes, not two different weights — and it cost a
detour. The corrected rule: **`encoder` for ordering, `tensor` for pairing.**

## What the ordinal can and cannot be

`Distance` (`plane/key.go:13`) is **`cmplx.Abs(hat − z[c])`** — a magnitude. Its own comment
says so: *"the magnitude ... is invariant."* Gauge invariance is achieved *by discarding the
sign*. So the comparator ranks by magnitude against a reference plane, and a magnitude ranking
cannot reproduce a signed linear order.

Searched 11 encoder derivations against a 6-compendium corpus (`freq`, `gain`, `phase`, `decay`,
year-linear, log, rank, mixed). **None is monotonic in time**; the best is 1 inversion out of 15
possible pairs, and the two 1-inversion candidates invert at different places — so the residual
error is structural to the comparator, not to the derivation.

Practically: `plane`'s ordinal is a **rank relative to 42**, invariant under rotation. It is not
a temporal order, and cannot be made into one through the weight derivation. If exact temporal
order is the requirement, the zero-reference timestamps give it directly and locally; if an
ordinal rank relative to the anchor is the requirement, this is it. The "Constitutional
Convention before Aladdin" constraint is *approximately* satisfiable (the best derivations get
the leading pairs right) but never exactly.

## The calendar plane, and why it works

The sign is not lost — it is just not in `Distance`. `TripleKey.Phase` is **signed**:
`wrapPi(cmplx.Phase(hatA) − cmplx.Phase(hatB))`, both coordinates transported into C's frame.

So a calendar plane is the **reference frame C**, and each compendium's signed phase within
that frame carries its temporal position. Verified: with a calendar plane at C=0, one
compendium per plane, and `encoder.phase[j]` scaled from each compendium's year, the sign of
`Phase` matched the sign of the year difference on **16 of 16 pairs** (and `|Phase|` tracked
`|year difference|` to roughly linear, ~0.0012 rad/year).

That is the design working end to end: `encoder` for ordering (as above), `Distance` for
magnitude, `Phase` for the signed relation, and the calendar plane for the frame.

**Two caveats, both measured:**

- **The result is frame-dependent, and that is by design.** Probe `42` — the consensus anchor —
  gave 10/10. An arbitrary probe `zzz` also gave 10/10. But `history:gamestop_squeeze:2021`
  and `disney:aladdin:1992` gave **7/10 and 9/10**. Switching probes switches the frame, and
  the ordinal changes with it. "Ordinally against 42" therefore means: well-defined when the
  frame *is* 42, and deliberately different otherwise. Do not read a lore-entity probe as the
  canonical frame.
- **`|Phase|` is not monotonic in `|year difference|`** — 3 inversions across 9 pairs. The
  sign is exact; the magnitude is approximate. Treat it as an ordering, not a measurement.

**Caveat 1 — `out != 0` blocks are skipped.** The model emits `out != 0` blocks
(`out: -2`, `out: 2`) but `Tensor.Score` skips them. A lore interaction only matters if it
lives in an `out == 0` block. An inverse compendium expressed as a *negative weight* works
for the sort; expressed as a *charge sign change* does not.

**Caveat 2 — `Frustrated` at the default `tol` is uninformative; use `Distance`/`Phase`/`Holonomy`.** `edgeAngle` (`plane/tree.go:187`) reads the charges **and the probe's phase**, and the holonomy does vary with the probe — `|phi|` ranges 0.0058–1.5559 for probe `"42"` versus 0.2764–2.9740 for a lore universe. But `Frustrated` is `|phi| > tol` with a **default `tol` of `1e-9`**, so the boolean saturates: every non-tree cycle counts as frustrated and the count is flat at 22 for every probe, anchor and weight set. At a realistic tolerance it becomes strongly probe-dependent:

| `tol` | `"42"` (consensus anchor) | costco | aladdin | gamestop |
|---|---|---|---|---|
| `1e-9` (default) | 22 | 22 | 22 | 22 |
| `1.0` | 4 | 22 | 22 | 22 |
| `2.0` | **0** | 3 | 13 | 8 |

So the probe phase *does* directly rank invalid frames, and the consensus anchor carries none while lore frames carry 3–13. **Never read `Frustrated` at the default `tol`** — pick a tolerance for the gauge you are in, or read the raw `Holonomy`.

**Rotation symmetry.** `TripleKey` reports `Distance` (gauge-invariant: both terms transform as `exp(i·q_c·θ)` so the magnitude survives a global rotation), `Phase` (cyclic tiebreak), and `Holonomy` (probe-dependent). Those three are the real observables; `Quarantined` is a thresholded downstream flag. For the intended use — rotating the frame and having the invariants preserved — build on `Distance` and `Phase`.

## 3. The corrected architecture

| `plane` slot | now | lore supplies |
|---|---|---|
| `charges` | arbitrary ints | which compendia are **exact inverses** — sector labels only, no magnitude |
| `encoder` per plane | `1+0.25q+0.1j` etc. | the compendium's complex value: real tier + imaginary 42i distance |
| `tensor.w` | index arithmetic | how compendia compose — **signed**; negative for inverse compendia |
| `tree` / `quarantine` | derived | nothing. Genuinely structural, and this is where cross-linked cycles get handled |
| `anchor` | literal `42`; `base == 0 → 42` | **derived**: the sum of load-bearing compendia, so 42 stops being a constant |

The last row is the point: "the sum of all 42i compendiums meet on 42" is currently a
hardcoded fallback, and lore makes it computed.

## 4. Where each lore field lands

| lore field | `plane` slot | notes |
|---|---|---|
| `reality-gradient` (0.0–1.0) | `encoder.*` real tier | **(0, 42) load-bearing, = 42 consensus, > 42 private.** Now a graded float, because it is a weight slot rather than a charge |
| `zero-status` (`established` / `presumed` / `unestablished`) | `encoder.*` imaginary | matches "structurally sound but not proven positive". The prototype's `Real: always 42` is why it could not express this — the real axis was pinned |
| inverse compendium | negative `tensor.w` | sign lives in the weight, so it survives the `Out == 0` invariance |
| **absence** (the "unknown" rung) | *no plane at all* | not a zero-valued compendium. `charges "1,-1,0"` fails with `charge graph is disconnected: p2:q0` — a neutral compendium cannot enter the graph. Absence is the correct and cleaner representation |

## 5. Compendia are design-time, planes are run-time

A compendium and a plane are **not** the same thing, and conflating them is what made the
first draft fail:

- A **compendium** declares what it assumes and carries its own anchored facts. It is the
  unit of lore.
- A **plane** is a coordinate slot in a `plane` run. Its charge is a sector label; its
  encoder values are the compendium's contribution; its tensor weights are how it couples.

So a lore ranking run picks N compendia, assigns them plane indices, and the *probe* is the
lore entity you are asking about. That is the only sense in which "the lore compendium
becomes a plane."

**What this cannot do:** rank *within* a compendium. If the whole lore compendium is one
plane, Costco and Idiocracy share one coordinate slot and cannot be distinguished. Ranking
across compendia works; ranking within one needs a plane per entity, which is the broken
case from the first draft.

## 6. The four rungs, restated as support-set state

| rung | support state | representation |
|---|---|---|
| possible | every required compendium present; claim anchored in one | `out == 0` block, positive weight |
| plausible | compendia present; claim is a **declared extension** of an anchor (`IN_A_PRIOR_KEYFRAME`) | positive weight, keyframe carries the derivation |
| unlikely / unknown | a required compendium is **absent** — a gap | that compendium has no plane |
| really wrong | required compendium's **inverse** is present | negative `tensor.w` in the `out == 0` block |

Ford's Theatre: the fire does not contradict "Lincoln was assassinated"; it contradicts one
**keyframe**. So contradiction is evaluated at keyframe granularity, and an inverse
compendium holds *the specific defeated instantiation* — never a general negation. The
inverse compendium inherits the address and timestamp of the keyframe it defeats. Otherwise
"the box was impossible" decays into "he was not assassinated", which is the error the
example exists to prevent.

## 7. Frontmatter attributes, applied at declaration time

New lore docs must declare these in frontmatter **when the universe is declared**, not
retrofitted later. The rationale is that a compendium's assumptions are known at
declaration time and are expensive to reconstruct afterwards; retrofitting produces a graph
that is complete for new docs and empty for old ones, which is the same collision risk the
`presupposes` edges are meant to avoid.

```yaml
plane:
  # Which compendia this one's value is exact-inverse to. Sector labels only.
  charges: []            # ints
  # This compendium's value, mapped from reality-gradient / zero-status.
  encoder:
    real: <>             # (0,42) load-bearing | = 42 consensus | > 42 private
    imaginary: <>        # 42i distance; zero-status derived
  # Compendia whose existence this one presupposes. Cycles allowed.
  presupposes: []        # universe-ids
  # Signed interaction weights toward coupled compendia.
  tensor: []             # [{to: <universe-id>, w: <float>}]
  # Set only when a keyframe of this universe is defeated by another's.
  inverse-of: null       # universe-id + keyframe-id of the defeated instantiation
```

Existing docs are **not** backfilled by this convention — see `OUTSTANDING.md`. The
`lore/*.json` build artifacts are generated by the local-time pipeline, which currently
drops unknown frontmatter keys, so `plane:` will not appear in the JSON export until
`build-lore-json.js` is taught it. That is tracked separately.

## 8. What would need to change in `plane` to make this native

1. **Named planes** — a `NewSpace` variant taking names, so `SortResult.Names` comes back
   as `lore/idiocracy.md` rather than `p3:q-1`. Removes the caller's reverse map.
2. **A weight-supply path that is not hand-editing JSON** — `GenerateModel` should accept
   `EncoderParams` and tensor blocks, or a derivation function, so lore-derived numbers do
   not have to be written into a file by a script.
3. **`anchor` derived rather than defaulted** — replace `base == 0 → 42` with the sum of the
   load-bearing compendia, so 42 is an invariant instead of a literal.
4. **An `edge`/`sort` mode that reports `ErrIncomparable` when `Out != 0`**, rather than
   the raw `At` entry — see the caveat in §2.
