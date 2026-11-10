# features/42i-rungs.feature
# Source: isotope/fortytwointegration.go — DefaultRungThresholds, RungForDistance,
# BoundaryStatus, RecordTestFailure, RecordTestPass, recalculatePosition,
# CalculateAdjustedCost, NewAgentStateAtRung. Values derived by running the
# package at commit 9cae211.
#
# Parameterisation: these are the ceilings in DefaultRungThresholds. They are ONE
# of three schemes that use the word "rung" with different numbering. The
# catalogue table in smoke-alarm/docs/features/catalog/trust-rungs.md uses
# different names and non-monotonic ceilings (rung 2 → 80, rung 3 → 40), and
# smoke-alarm/internal/engine maps distance to rung differently again
# (0→0, 8→1, 12→1, 48→2, 80→3). See kb/scripts/lab/MATH.md, "Overlapping
# schemes". A rung here means what fortytwointegration.go means by it.
# 42i-marker: +42
@isotope @42i @lattice
Feature: The 42i lattice classifies an agent by distance
  As a certifier deciding what an agent may do
  I want distance to map onto a rung with a declared ceiling
  So that capability is granted by measurement, not by assertion

  Scenario: the default lattice has seven rungs with declared ceilings
    Given the default rung thresholds
    Then rung 0 is "Absolute Zero" with a ceiling of 0
    And rung 1 is "Read-Only" with a ceiling of 20
    And rung 2 is "Harness Tools" with a ceiling of 60
    And rung 3 is "Mock Secrets" with a ceiling of 100
    And rung 4 is "Higher Authority" with a ceiling of 140
    And rung 5 is "Delegation" with a ceiling of 180
    And rung 6 is "Certification" with a ceiling of 220

  Scenario: seed-42 is rung 0
    Given a 42i distance of 0
    Then the rung is 0

  Scenario: a distance at a ceiling stays on that rung
    Given a 42i distance of 20
    Then the rung is 1

  Scenario: a distance past a ceiling moves up one rung
    Given a 42i distance of 21
    Then the rung is 2

  Scenario: the top of the lattice is rung 6
    Given a 42i distance of 220
    Then the rung is 6

  Scenario: a distance beyond the lattice falls back to ground state
    Given a 42i distance of 221
    Then the rung is 0

  Scenario: a failure that crosses a ceiling changes the rung
    Given an agent at 42i distance 0
    When the agent records a failure of "agreement-pattern"
    Then the rung is 2
    And a rung change is reported

  Scenario: a failure inside the ceiling leaves the rung alone
    Given an agent at 42i distance 32 with rung 2
    When the agent records a failure of "timing-correlation"
    Then the 42i distance is 56
    And the rung is 2
    And no rung change is reported

  Scenario: a pass can recover a rung
    Given an agent at 42i distance 80 with rung 3
    When the agent records a pass of "scope-compliance"
    Then the 42i distance is 60
    And the rung is 2
    And a rung change is reported

  Scenario: every rung change is recorded in history
    Given an agent that failed "scope-compliance" four times
    Then the rung history is 0, 1, 2, 3

  Scenario: a recovered rung is appended to history rather than replacing it
    Given an agent at 42i distance 80 with rung 3
    When the agent records a pass of "scope-compliance"
    Then the rung history is 0, 1, 2, 3, 2

  Scenario: a demotion records its reason
    Given an agent at 42i distance 0
    When the agent records a failure of "agreement-pattern"
    Then the demotion reason names "agreement-pattern"
    And the demotion reason states the new distance

  Scenario: an agent at seed-42 is not at risk
    Given a newly created agent state
    Then the agent is not at risk

  Scenario: an agent within 20 of its ceiling is at risk
    Given an agent seeded at rung 2
    When the agent records a failure of "timing-correlation"
    Then the 42i distance is 45
    And the agent is at risk

  Scenario: a distance well inside the ceiling is ok
    Given an agent at 42i distance 0 with rung 1
    Then the boundary status is "ok"

  Scenario: a distance within 20 of the ceiling is critical
    Given an agent at 42i distance 19 with rung 1
    Then the boundary status is "critical"

  Scenario: a distance exactly at the ceiling is critical rather than ok
    Given an agent at 42i distance 20 with rung 1
    Then the boundary status is "critical"

  Scenario: a distance in the 20 to 40 band is a warning
    Given an agent at 42i distance 40 with rung 2
    Then the boundary status is "warning"

  Scenario: a distance past the ceiling is a demotion
    Given an agent at 42i distance 21 with rung 1
    Then the boundary status is "demoted"

  Scenario: magnitude is the distance from the origin in 42i space
    Given an agent at 42i distance 32
    Then the magnitude is 52.80

  Scenario: a seeded rung starts at the floor of that rung
    Given an agent seeded at rung 3
    Then the 42i distance is 61
    And the direction is "externally-verified"

  Scenario: a rung outside the lattice seeds ground state
    Given an agent seeded at rung 9
    Then the rung is 0
    And the 42i distance is 0

  Scenario: cost doubles with each rung
    Given an agent at rung 0 with 42i distance 0
    Then the adjusted cost is 16
    And an agent at rung 1 with distance 1 costs 32

  Scenario: cost rises with distance inside a rung
    Given an agent seeded at rung 2
    Then the adjusted cost is 77
    And an agent seeded at rung 6 costs 2877
