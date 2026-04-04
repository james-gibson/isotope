# isotope — Trust Protocol Library

`github.com/james-gibson/isotope`

A shared Go library implementing the **isotope protocol**: a framework for verifying the trustworthiness and capability of agents in the prototype lab before allowing them to act.

Not a binary. All lab tools depend on it.

---

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

`VerifyTrust(ctx, endpoint)` determines a rung by checking the lezz cluster registry and mDNS:

| Condition | Rung |
|-----------|------|
| Registry + mDNS confirmed | 6 — Certification |
| Registry only | 4 — Higher Authority |
| Self-registered, no cluster | 2 — Harness Tools |
| No registration | 0 — Absolute Zero |

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
rung, err := isotope.VerifyTrust(ctx, "http://localhost:8088")
// Returns 2, 4, or 6
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
- **ocd-smoke-alarm** — serves `/isotope/register` and `/isotope/list`; calls `VerifyTrust` on each registrant
- **adhd** — registers itself on startup; calls `Advertise` to broadcast `_adhd-isotope._tcp`
- **lezz.go** — calls `Client.List` to include trust levels in the demo cluster summary

---

## See Also

- [lab-safety — full ecosystem overview](https://github.com/james-gibson/lab-safety)
- [ocd-smoke-alarm](https://github.com/james-gibson/smoke-alarm)
- [adhd dashboard](https://github.com/james-gibson/adhd)
- [pickled-onions seeds](https://github.com/james-gibson/pickled-onions) — Gherkin world-builder that feeds the canonicalization pipeline
