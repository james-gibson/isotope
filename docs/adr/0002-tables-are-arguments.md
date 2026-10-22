# ADR 0002 — a data table is the argument of a step, not a set of features

**Status:** proposed, 2026-10-21
**Amends:** `canonicalization.go:212-240` (documents existing behaviour; proposes a named rule)
**Related:** ADR 0001 (identity is step text), ADR 0003 (outlines *are* sets of features)

## Context

A `| … |` table under a step currently reaches no part of the identity: `ParseGherkin`
has no handler for a table line, so the rows are dropped and only the step's own text is
hashed. Measured: two specs whose only difference is `| surface | base |` versus
`| surface | admin |` produce the same isotope, `eOBhPmeUtD6x`.

It is tempting to call this a hole — a table looks like a set of single-entry features,
so it should decompose like one. It does not, and the reason is structural rather than
an oversight:

- **A row is not a feature.** `| base |` under `surface` and `| base |` under `rung` are
  different promises, so a row's identity would have to be
  `(step text, headers, values)` — the triple. Hash the values alone and a column swap
  collides into the same token, which is *worse* than invisibility because it looks
  like coverage.
- **A table is one step's argument.** `Given a placement | surface | base |` is one
  behaviour taking one structured value. Decomposing it into cases would be
  decomposing a function argument into a test matrix.
- **Exclusion is what makes the fixture slot work.** A challenge that varies fixtures
  needs the values outside the identity — otherwise varying a fixture *is* editing the
  spec, and the whole point of `declare → canonicalise → sign → challenge → run` is
  lost.

So the current behaviour is right and the reason for it has never been written down.
That gap is the decision this ADR closes.

## Decision

**A step's data table is an argument, not behaviour.** It is excluded from the isotope
by design, and the exclusion is a documented rule rather than an artefact of parsing.

The composition rule, should a table ever need to be identity-bearing (a per-record
receipt, a row-addressed part): hash the **triple** — the owning step's text, the header
row, and the value row — never the values alone.

The authoring rule that follows: values that must vary per run live in tables or in
`<placeholder>` slots, never spelled into a step. A value spelled into a step is
behaviour (ADR 0001) and a volatile literal there is an error.

## Consequences

| want | mechanism | identity impact |
| --- | --- | --- |
| vary a fixture per run | table row, or `<placeholder>` | none — by design |
| declare that a fixture is required | the header row | none — the header is not hashed either |
| make a row individually addressable | decompose to a part with the triple as its raw | new part IDs; see ADR 0003's migration cost |

`<placeholder>` is safe because normalization deletes `<` and `>` (`:196`), so
`<workspace>` and `workspace` are the same token — stable across runs.

## Known limits

- **Exclusion is unconditional, so a declared fixture is indistinguishable from an
  undeclared one.** A spec can require `| surface | base |` and the token says nothing;
  a runner given a different table cannot tell from the isotope that it changed the
  setup. The declaration lives only in the file, and only for a reader who has the file.
- **Header changes are invisible too.** Renaming a column changes meaning and changes
  nothing in the identity, which is the same failure as a value change but easier to
  miss in review.
- **Two tables cannot be distinguished by count.** Adding rows is behaviourally
  significant for a data-driven step and produces no record.
- **The exclusion is currently untested.** No vector in `brine.py --self-test` asserts
  that table content is ignored, so an accidental future table handler would change
  every ID in the store with no failing test to announce it.

## What would reverse this

A requirement that fixtures be *accountable* rather than swappable — e.g. an attestation
that must show which fixture set was proven. That would move tables into the identity
and force the challenge to vary only placeholders, at the cost of the directness that
"vary the fixture" has today.
