# ADR 0001 — the isotope identifies wording, not behaviour

**Status:** proposed, 2026-10-21
**Amends:** `canonicalization.go` (documents existing behaviour; proposes no code change)
**Related:** ADR 0002 (tables), ADR 0003 (outlines)

## Context

`Canonicalize` (`canonicalization.go:212-240`) emits three token kinds and nothing else:

```
FEATURE:name:<normalized feature name>
SCENARIO:name:<normalized scenario name>
<KEYWORD>:hash:<sha256(normalized step text)[:8]>
```

`ParseGherkin` (`:83-130`) only recognises `Feature:`, `Scenario:`, and step keywords,
and only accepts a step while a scenario is open (`:119`). Everything else on the page
is skipped without a diagnostic. So the identity is the **order of scenario and step
lines**; the behaviour such a spec describes is not in it unless that behaviour is
written as a step.

Measured against the Python port (`kb/projects/scripts/brine.py`, same vectors as
`./brine.py --self-test`):

| two specs differing only in | isotope |
| --- | --- |
| a data-table row (`base` vs `admin`) | **identical** `eOBhPmeUtD6x` |
| a `Background:` anchor (`0i + 42` vs `99i + 99`) | **identical** `901drSV8wbVz` |
| `the penalty is +8` vs `the penalty is −8` (U+2212) | **identical** `QwdySnsSHhOy` |
| `0 < 1` vs `0 > 1` | **identical** `RAO3uf6Ee4eN` |
| scenario order (two scenarios swapped) | different `JD1AiOZHzDaq` / `x3gJwfrTjkZK` |
| `42i + 0` vs `0i + 42` | different `p_SBx7Hooz4A` / `m2cscr45OUbc` |

`normalizeStepText` (`:173-201`) lowercases, deletes articles, then deletes every
character outside `[^\w\s-]` — so `+`, `<`, `>`, `:` and U+2212 vanish, which is why
the two weight-sign rows and the two ladder rows collide. Only an ASCII hyphen
survives, so `-8` is distinguishable and `−8` is not.

This matters now because `lezz.go/docs/attestation.md:890` claims *"if the fix changed
what the node does, the spec — and therefore the token — differs"*. The first two rows
above are counterexamples, and they are the common kind of change.

## Decision

**Declare the scope: the isotope identifies a promise's wording.** A behaviour that is
not written on a step line is not in the promise, and a consumer that needs it must be
told to look elsewhere.

The consequences for authors, stated as rules:

1. Anything a behaviour depends on must be a step (`Given`/`When`/`Then`/`And`/`But`),
   never a `Background:`, never a bare table.
2. Operators and signs in step text must be spelled, not symbolised — `plus 8` /
   `minus 8`, `less than` / `greater than`. The canon cannot carry the glyphs.
3. Scenario order is identity. Do not reorder a spec to tidy it; reorder *is* an edit
   (and see ADR 0002 / brine `classify()` for how to say so legibly).

No change to `Canonicalize`. The decision is to stop describing this as an identity for
behaviour, and to make the authoring rules explicit rather than implied.

## Consequences

| consumer | gets | must not assume |
| --- | --- | --- |
| a test-suite fingerprint | the wording of a spec, stably | that unwritten preconditions are covered |
| a route/community tag | a name for a declared feature set | that two nodes with the token behave alike |
| an attestation | a commitment to what was *declared* | that a change in it is the only change that matters |

## Known limits

- **The sign collision is silent.** `+8` and `−8` hash identically, so a spec that
  asserts the wrong direction of a `MATH.md` weight is the *same promise* as the right
  one. Neither the token nor brine's audit can see it.
- **Order comparison is not covered.** `0 < 1` and `0 > 1` are one token, so the
  fourth rung of the ladder in ADR-adjacent work (`lore/CONVENTIONS.md`, the `42i`
  origin) has no identity distinct from its inverse.
- **Non-Latin text is deleted** (`[^\w]` is ASCII), so two non-English steps can
  collide; specs must be English.
- **Keyword matching is a bare prefix** (`:119`), so a step beginning `Andromeda` is
  read as `And` + `romeda…`. `validate_spec.py` flags it; the canon does not.
- **A reorder is a new isotope**, which reads as a behaviour change unless the consumer
  also carries brine's digest pair.

## What would reverse this

Hashing table content as structured arguments (ADR 0002) and covering operator glyphs
in normalization would each narrow the gap. Doing both would make the isotope closer to
a behaviour fingerprint — and would then have to answer whether the fixture slot
survives, because a hashed table cannot be varied by a challenge without changing the
identity.
