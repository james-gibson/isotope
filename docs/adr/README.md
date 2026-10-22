# Architecture decision records — isotope

Each ADR is one decision, the evidence for it, and what it costs. They are numbered
and never renumbered; a superseded ADR stays in place and points at its replacement.

| # | decision | status |
| --- | --- | --- |
| [0001](0001-identity-is-step-text.md) | the isotope identifies wording, not behaviour | proposed |
| [0002](0002-tables-are-arguments.md) | a data table is the argument of a step, not a set of features | proposed |
| [0003](0003-examples-rows-are-parts.md) | an outline is an array of cases; each row is a part | proposed |
| [0004](0004-key-schedule-is-an-input.md) | the signing key is a declared input with a named source | proposed |

**Status vocabulary:** *proposed* (written, not ratified) · *accepted* · *rejected* ·
*superseded by NNNN*.

All four were originated together on 2026-10-21 from a review of
[james-gibson/lezz.go#21](https://github.com/james-gibson/lezz.go/issues/21), which
places this canon at the centre of an attestation protocol it was never scoped for.
The evidence in them was produced by running the canon, not read off the prose about
it — the vectors are in each ADR so a later reader can re-derive rather than trust.

Format: **Context** (what is, with measurements) · **Decision** · **Consequences** ·
**Known limits** · **What would reverse this**. The limits sections are not
disclaimers; each one is a known way the decision produces a false reading, and they
exist so a later agent does not have to rediscover them.
