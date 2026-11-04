# family: deployment
# The peer-registration service (api/src/peer-registration/), mounted by the
# runtime app (6001). Tokens are uuid-based, expire after 5 minutes and are
# single-use; claims attach an RSA public key; data is verified RSA-SHA256.
# Sources: api/src/peer-registration/peer-registration.service.ts,
# api/src/peer-registration/interfaces/.
# Captured 2026-11-03 from the prototype.

@api @runtime @peer-registration
Feature: The peer-registration service issues single-use claim tokens for peers
  As an untrusted peer
  I want registration to hand me a one-shot token I can claim with my key
  So that later messages are provably mine

  Background:
    # Presupposition addressed by ground-truth isotope (canonical triple: signing
    # key "seed-42", the declared public constant of ADR 0004; family "nest"; v1)
    # rather than by magic name, per #40: the edge pins the
    # presupposed surface, so drift in the target breaks it loudly instead of
    # silently following a rename. Names as declared today:
    #   nest-v1:c9c11a1730f4b7c4 = nat-traversal — the UDP path registration
    #     rides; node-management presupposes the same spec
    Given the feature "nest-v1:c9c11a1730f4b7c4"

  Scenario: peer registration issues a single-use claim token
    Given the runtime app is running
    When a peer registers with a name, address, port and capabilities
    Then a peer id and claim token are returned
    And the token expires after 5 minutes and can be used once

  Scenario: claiming attaches the peer's public key
    Given a peer holds an unclaimed registration token
    When the peer claims it with an RSA public key
    Then the key is attached to the peer record with its NAT info

  Scenario: signed peer data is verified with RSA-SHA256
    Given a peer has claimed its token with a public key
    When the peer presents data signed with the matching private key
    Then isDataSigned verifies the signature against the attached key

  Scenario: a peer is findable by capability
    Given registered peers advertise capabilities
    When a caller asks for a peer that can a capability
    Then a matching peer is returned
