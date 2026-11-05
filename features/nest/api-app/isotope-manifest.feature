# family: deployment
# The isotope manifest service: the introspection surface every api app mounts
# (like authn). Each generated NestJS module registers the isotope it
# implements — ground-truth address, announced family, surface declarations,
# presupposed edges — and this service answers for them over JSON-RPC on its
# own endpoint, leaving stream-deck's /rpc method set closed.
# Specified (not captured) 2026-11-04 for the x-nestjs:json-rpc generator
# (gibsunas/lemon-x-files#5). The first nest feature written in the vocabulary
# surface form (the migration isotope#42 tracks) and port-free per
# nest_steps.go's law — "a port is a deployment-collision decision an app
# announces, never a feature's declaration" — and features/nest/README.md.
# Doctrine: the manifest returns claims, not evidence
# (docs/claims-routes-tension.md) — describe answers recomputably so a caller
# re-derives the address instead of trusting it; attestation, receipts and
# trust rungs never appear on this surface (ADR 0004, docs/trust-roots.md).

@api @isotope-manifest
Feature: The isotope manifest answers for the surfaces its app implements
  As a peer of the house
  I want each app to declare over JSON-RPC which isotopes it implements
  So that what an app serves is checkable against its specs instead of trusted

  Background:
    # Presupposition addressed by ground-truth isotope (canonical triple:
    # signing key "seed-42", the declared public constant of ADR 0004; family
    # "nest"; v1) rather than by magic name, per #40/#43: the edge pins the
    # presupposed surface, so drift in the target breaks loudly. Name as
    # declared today:
    #   nest-v1:8d68b8366ff2fe51 = app-root — the app scaffold this sits on
    #     (readdresses if the ADR 0005 migration (#41) lands — repin then)
    Given the feature "nest-v1:8d68b8366ff2fe51"

  Scenario: the manifest surface
    Given the route "/isotope"
    Given the JSON-RPC method "isotope.list"
    Given the JSON-RPC method "isotope.describe"

  Scenario: list answers for every isotope the app implements
    Given any api app is running
    When a client posts an isotope.list request to /isotope
    Then every isotope the app implements is returned
    And each entry carries its ground-truth address, announced family, version, feature name and transports

  Scenario: describe answers recomputably
    Given any api app is running
    When a client posts an isotope.describe request for a registered address to /isotope
    Then the response carries the spec text, the canonical token string and the full 256-bit signature
    And the response carries the surface declarations by kind and the presupposed edges
    # The caller recomputes HMAC-SHA256 over the canonical tokens with the
    # public constant, and re-canonicalizes the spec to check both — a route
    # that can be re-walked, not a narrative. The truncated 16-hex display
    # token is presentation; the 256-bit hex is the auditing artifact
    # (ADR 0004).

  Scenario: an unregistered address is refused
    Given any api app is running
    When a client posts an isotope.describe request for an unregistered address to /isotope
    Then the response is the -32602 invalid-params error

  Scenario: any other JSON-RPC method is refused
    Given any api app is running
    When a client posts a JSON-RPC 2.0 request with a different method to /isotope
    Then the response is the -32601 method-not-found error

  Scenario: the manifest claims nothing beyond content
    Given any api app is running
    When a client posts an isotope.list request to /isotope
    Then the response carries no attestation, no receipt and no trust rung
    # Freshness and trust are the receipt-and-ladder machinery's job (ADR
    # 0004's session-key row, docs/trust-roots.md) — never this surface's.
