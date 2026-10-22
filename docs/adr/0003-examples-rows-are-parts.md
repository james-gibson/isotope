# ADR 0003 — an outline is an array of cases; each row is a part

**Status:** proposed, 2026-10-21
**Amends:** `ParseGherkin` (`canonicalization.go:96-130`) — this one proposes a code change
**Related:** ADR 0001 (identity is step text), ADR 0002 (data tables are arguments)

## Context

`Scenario Outline:` is the one construct where the array model is literally correct: each
`Examples:` row is a full substitution of the outline's steps, so each row is a distinct
case with a distinct promise.

The canon currently does the opposite of decomposition:

- `Scenario Outline:` does not match the `Scenario:` prefix test (`:108`), so it is
  skipped and the **previous** scenario stays open;
- the outline's steps are therefore appended to whatever scenario came before it;
- its **name is dropped**;
- every `Examples:` row collapses into one case, since rows are not parsed at all.

Verified in `canon-constraints.md` (produced by running the canon): a spec with
`Scenario: first` followed by an outline yields `SCENARIO:name:first` with the outline's
`When`/`Then` appended to it, both example rows gone. The outline's own name never
appears.

So a file that reads as three cases is one case with a misleading name. This is the
construct `canon-constraints.md` calls "the single most dangerous for an isotope-bearing
spec", and it is the only one in that document which is a *corruption* rather than an
omission.

## Decision

**An outline decomposes, by row, into parts.** Each `Examples:` row is a case: a part
with its own `part_id`, its own record, and its own receipt. The outline's name is
retained as the scenario name for every row it generates; the row index is carried in
the part's metadata, not in the name (otherwise adding a row would renumber the ones
after it).

The collection is a **set**, not an array: row order carries no meaning, so the
generated parts contribute to `part_set_digest` (order-insensitive) and not to any
position-sensitive chain. This is the same pair brine already implements for scenarios
(`brine.py:543-550`), applied one level down.

Scope: one `Examples:` block first. Multiple blocks (`Examples:` twice, or
`Scenario Template:`) are a separate decision and are left out of this one.

## Consequences

| now | after | cost |
| --- | --- | --- |
| outline steps attach to the previous scenario | each row is its own part | every existing outline's ID changes |
| all rows collapse to one case | one receipt per row | store grows with the case count |
| adding a row changes one ID | adding a row changes the parent and adds a part | parents must be re-recorded |
| reordering rows is silent | reorder-classifiable via `part_set_digest` | — |

The ID churn is unavoidable: the fix changes what the bytes mean, so it needs a
`CANON_VERSION` bump and a re-record of every affected spec. That is a migration, not a
patch, and it should land with the version bump in the same change.

## Known limits

- **Every outline in every store changes identity.** Any receipt citing an old outline
  ID stops matching. Since a receipt is evidence of a run, this is the expensive half:
  the old runs remain valid history but can no longer be re-derived.
- **Row identity is `(outline name, row values)`.** If two rows are identical, they
  produce the same part ID — a set, deliberately, but a duplicate row is then a silent
  de-dupe rather than a visible error.
- **This does not make tables identity-bearing** (ADR 0002). An outline with a data
  table inside it still has an invisible table; the two decisions are orthogonal and
  should not be conflated in review.
- **`validate_spec.py` must move from warning to error** for outlines once the canon
  fixes them, or specs written before the fix will keep validating against the old
  reading.

## What would reverse this

Nothing in the current use makes a collapsed outline desirable. If the fix is judged too
expensive to migrate, the fallback is not "leave it" but "forbid outlines in
isotope-bearing specs" — an omission is at least honest, where a corruption is not.
