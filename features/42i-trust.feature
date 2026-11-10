# features/42i-trust.feature
# Source: isotope/trust.go — VerifyTrust, checkRegistry, confirmViaMDNS.
# VerifyTrust depends on a live lezz registry and mDNS, so these scenarios
# state the rung it assigns for each outcome rather than a wire-level exchange.
#
# The rungs here are VerifyTrust's own assignment (6 / 4 / 2), not the catalogue
# rungs in trust-rungs.md and not DefaultRungThresholds. Three schemes share the
# word; see kb/scripts/lab/MATH.md.
# 42i-marker: +42
@isotope @42i @trust
Feature: Trust is a starting rung, not a judgement
  As a federation admitting a new isotope
  I want an externally verified participant to start higher than a stranger
  So that certification is earned once and carried forward

  Scenario: a registry entry confirmed by mDNS certifies at rung 6
    Given an endpoint listed in the lezz registry
    And mDNS confirms the same endpoint
    When trust is verified
    Then the trust rung is 6

  Scenario: a registry entry without mDNS lands at rung 4
    Given an endpoint listed in the lezz registry
    And mDNS does not confirm it
    When trust is verified
    Then the trust rung is 4

  Scenario: an unknown endpoint self-registers at rung 2
    Given an endpoint that is in no cluster
    When trust is verified
    Then the trust rung is 2

  Scenario: trust verification is bounded
    Given an endpoint of any kind
    When trust is verified
    Then the check completes within four seconds

  Scenario: an externally verified agent starts at its certified rung
    Given an agent certified at rung 6
    Then the rung is 6
    And the 42i distance is 181
    And the direction is "externally-verified"
