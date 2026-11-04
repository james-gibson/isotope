# ADR 0005 — the port is a family value

**Status:** proposed, 2026-11-04
**Amends:** the `features/nest/` capture (56 identity-bearing port step-lines in 27 of 34 files); the `family.go` contract (names the value the deployment assigns); the `nest_steps.go` port comment (its law becomes canon-wide)
**Related:** ADR 0001 (the mechanism — step text is identity), ADR 0004 (declared inputs; frames and gauges), `docs/trust-roots.md` (the ladder scores the announcement), `docs/claims-routes-tension.md` (an advertisement is a claim), PR #40 (the first ledger edges addressed by ground-truth isotope)

## Context

`nest_steps.go` states the law: "The port is deliberately absent everywhere: a port is a deployment-collision decision an app announces, never a feature's declaration." The vocabulary obeys — `NestStepDefinitions` has no port kind. The captured corpus does not: **56 scenario step-lines in 27 of the 34 feature files carry a port number** (36× `3001`, 6× `5000`, 6× `3003`, 5× `3002`, 1× `6001`, 1× `53`, 1× `41234`), and ADR 0001 makes step text identity — so every one of those lines bakes a frame into the content address.

Measured under the ground-truth triple (key `seed-42` — the declared public constant of ADR 0004; family `nest`; v1 — the triple `family_test.go` and `nest_steps_test.go` canonicalize with):

| two specs differing only in | isotope |
| --- | --- |
| `port 3001` vs `port 3002` in a scenario line | **different**: `nest-v1:46dd4d496447a37b` vs `nest-v1:523afd5329b15ce6` |
| the port line in `Background:` vs absent entirely | **identical**: `nest-v1:636374acd54db752` |
| the family at canonicalization: `nest` vs `3001` vs `6001` | **same signature** `46dd4d496447a37b`; only the display prefix differs |

The HMAC covers the canonical token string and nothing else — the family argument never enters it. The frame slot therefore already exists, is deployment-assigned (`family.go`: "The tag declares the TYPE; the deployment assigns the value"), and all 34 nest specs declare their type as `deployment`. The corpus leaks into its step text what the family slot was designed to carry.

The leak has three shapes, all measured in the capture:

- **Restatement** — the apps announce once (`boot.feature:18-20`: main listens on 3001, admin on 3002, auth on 3003) and every service echoes the frame into its own scenarios (`webhook.feature` alone: four lines of `Given the main app is running on port 3001`).
- **Crossing** — a client spec carries the server's frame: `game-client.feature:25` restates `the main app is running on port 3001`. A service that serves on a port has a matching client on the other side; today the pairing is duplicated prose, and each side's identity carries the other's deployment decision. (The capture half-knows this: `dht.feature`'s *header comment* — canon-dropped — is where it names the tracker's `port 5000`.)
- **Straddling** — `note.feature` states both `port 3001` (`:19`) and `port 3002` (`:43`): one service, two frames, fused into a single identity that is neither deployment.

Against them stands `authn.feature` — the one service mounted on all four apps — already written port-free (`Given any api app is running`, `:16`). Its identity is migration-invariant: `nest-v1:12549f2967d60332`. The capture contains its own existence proof that the port-free spec is writable; it just did not apply it everywhere.

The duality is already enforced elsewhere in the canon, undocumented:

- `VerifyRegistration` (`trust.go:172`) scores exactly the server-announces/client-matches pair: rung 6 requires "attested, the cluster's registry entry names this endpoint (**host and port**), and mDNS confirms the cluster"; rung 2 is the self-declared endpoint with no match. `VerifyTrust` calls the result "the initial **42i trust rung** for a registering isotope".
- `Advertise` (`mdns.go`) registers the listen port with role and rung as TXT fields; `Registration` carries endpoint + attestation.
- `ChainLink` (`consensus.go:147`) pairs role-tagged isotopes — `requester`, `secondary`, `verifier` — each link an `Isotope`, which has a `Family`.
- `DefaultWeights` (`fortytwointegration.go:8`): "maps test isotope families to their 42i failure weights (**gaps in imaginary space**). Each failure moves an agent toward -42i (away from seed-42 ground truth)" — and the bookkeeping is keyed per family (`FailedTests[isotopeFamily]`).

So the port already exists in the system as an advertisement (server side), as a match scored by the ladder (client side), and as the accounting namespace (family). It exists as step text only in the 56 leaked lines.

## Decision

**The port is a family value.** The deployment announces it, the caller of `CanonicalizeGherkinToIsotope` passes it as `family`, and step text never carries it — with one boundary:

1. **The deployer's port is frame; the protocol's port is content.** `nat-traversal.feature:22` ("a NAT mapping registration packet is sent to a public DNS resolver on port 53") keeps its port: 53 is a protocol fact, the same in every deployment, part of the behaviour under test. `webhook.feature`'s `port 3001` goes: 3001 is where this deployment happened to put the app. The test: *could another deployment of the same spec answer somewhere else?* If yes, the port is frame.
2. **One spec, as many frames as deployments.** The same bytes canonicalize to the same signature under any family (vector 3 above), so authn's situation — one probe, four apps — becomes the general case: signature `12549f2967d60332`, displayed `3001-v1:…`, `3002-v1:…`, `3003-v1:…`, `6001-v1:…`. The spec is written once and signed into each deployment; `family.go`'s split finally does its whole job.
3. **The pair meets at the ledger edge and the ladder.** A client presupposes a server by content address — `Given the feature "<isotope>"` in `Background:`, the pattern of PR #40 — and resolves that address to a live endpoint through the advertisement (registry entry naming host:port, mDNS naming the cluster), which `VerifyRegistration` scores. The edge (sig ↔ sig) is the frame-independent part; the rendezvous (family ↔ advertisement) is the frame. In the plane's language this is the gauge story of ADR 0004 one level down: only ratios and differences are gauge-invariant, so what a client may *presuppose* is the address-ratio to its server, while the port is a coordinate of the frame the deployment happens to stand in. A deployment is a working hypothesis — the imaginary anchor `42i`, against the proven consensus `42` on the real axis (`cmd/plane/main.go:57`) — and the family value is where the hypothesis is written down.
4. **Drift gets a location.** Family already keys the 42i books (`DefaultWeights`, `FailedTests[family]`, `TotalDistance` toward -42i). When the family value carries the frame, a failing deployment drifts *at a named coordinate* instead of inside an undifferentiated corpus family.

The family-value syntax (`3001` vs `api-3001` vs a cluster name) is a deployment convention, not canon — `String()` displays whatever the caller passed. Ledger addressing keeps the ground-truth family `nest` (where PR #40's pins live); a deployment frame is a re-presentation of the same signature, never a different address.

## Consequences

- **One-time readdressing.** Migration strips the restatement and crossing lines; every spec that carried one gets a new address. First measured move: `app-root` `nest-v1:8d68b8366ff2fe51` → `nest-v1:d8bbfe971220f18d`. Pins follow — PR #40's two Background addresses (app-root, github) recompute when it migrates, which is precisely the lockfile semantics #40 announced. Port-free specs (authn and the six other clean files) move zero.
- **App-manifest features are the open item.** `boot.feature`, the `*-app.feature` files, and `dht-tracker/tracker.feature` exist to announce ports — `Then the main app listens on port 3001` *is* their surface. This ADR does not decide them; two readings survive: (a) the announcement becomes a `Background:` step (canon-dropped, readable for meaning, possibly a future `announces port` vocabulary kind), with the concrete assignment living in the family value and the deployment manifest; or (b) a deployment-manifest spec accepts a frame-relative identity — family per deployment, ports in step text because there they are the content. The 27-file migration should settle this before stripping these files.
- **The UDP surface stresses the rule.** `udp-ingestion.feature:17` puts the deployer-chosen port inside the stimulus ("sends an AES-256-CBC encrypted datagram to UDP port 41234"): in UDP the port *is* the address — there is no path to carry the identity instead. Under the decision rule, 41234 is frame (another deployment could ingest elsewhere), so the step wants a rewrite to "the ingestion service's announced UDP port" with the number in the family value.
- **Family polysemy becomes explicit.** The slot already carries behaviour-test families in `DefaultWeights` ("entropy-check", "timing-correlation"), the corpus ground truth ("nest") in tests, and now deployment frames. Unknown families fall to weight 4 (`fortytwointegration.go:166`) — cheap but blind: a weights table keyed by behaviour families will not find deployment families. The accounting needs either one namespace or a declared mapping.
- **The README becomes true.** `features/nest/README.md` claims "never a port"; after migration the claim holds over step text, and the counterexample moves to where the design already put it — the family argument at canonicalization.

## Known limits

- **The family value is a free-form claim.** Nothing at canonicalization validates `3001` as a port or prevents a collision inside a family (two signatures, one prefix). Collisions surface only at advertisement time — registry/mDNS, which is where the ladder already lives — but a typo'd family silently yields a valid-looking display ID.
- **64-bit display truncation** (ADR 0004's known limit) applies to frame-qualified addresses as well; the store's 256-bit hex remains the auditing artifact.
- **The frame/content boundary is a judgment, not an algorithm.** 53 is protocol; 3001 is frame; 41234 is frame inside a stimulus; a hypothetical spec testing "answers on 3001 and not 3002" has ports as content. The decision rule (*could another deployment answer elsewhere?*) is a test a person applies, and misapplied it either re-leaks the frame or strips real content.
- **Stripping without edging loses the duality.** When `game-client.feature` stops naming 3001, the client→server pairing must exist as a ledger edge (a `Background:` address, PR #40's pattern) or it exists nowhere. The migration must convert each crossing into an edge, not delete it.

## What would reverse this

A requirement that the deployment port be part of test identity — a spec whose behaviour *is* the frame ("answers on 3001 and nowhere else") — puts ports back into step text for that spec. Or a requirement that the family value be authoritative rather than a claim: then the port must ride a receipt (ADR 0004's session-key row, stability traded for unforgeability), because a display prefix anyone can compute with the public constant gates nothing.
