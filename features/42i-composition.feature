# features/42i-composition.feature
# Source: isotope/fortytwointegration.go — RecordTestFailure, RecordTestPass,
# recalculatePosition, inferDirection, RungHistory. Values verified by running
# the package at commit 9cae211.
#
# The rule this file pins: a delta adds, a position does not compare across
# origins. Four ways of saying the same thing:
#
#   arithmetic   1 + 1 + 1 = 3                  a sum, no origin
#                42 + 1 + 1 + 1 = 45            a position, origin 42
#   shell        cd /; cd users; cd dev          a path, lands at /users/dev
#                the same two moves from /home   lands at /home/users/dev
#   brine        part_set_digest                 the multiset of parts
#                part_order_digest, chain        the path they were applied in
#   plane        total displacement              the complex position, anchor 42
#                42 + i*distance                 or 0 + 42i, a different reading
#
# A 42i distance is a position, a failure is a relative move, and the anchor is
# the working directory. Reordering the moves does not fail: `cd /; cd dev`
# lands at /dev, and a reordered spec still canonicalizes — to a different ID.
# See kb/scripts/lab/MATH.md and PLANE.md.
# 42i-marker: +42
@isotope @42i @composition
Feature: A 42i distance is a position, not a sum
  As a reader comparing two agents
  I want to know when two numbers are comparable
  So that I do not compare a path from one root with a path from another

  Scenario: the same delta from a different origin reaches a different position
    Given an agent seeded at rung 3
    When the agent records a failure of "agreement-pattern"
    Then the 42i distance is 93
    And the rung is 3
    And the magnitude is 102.04
    And an agent at seed-42 with the same failure is at distance 32 on rung 2

  Scenario: failures of 8 then 24 pass through rung 1 on the way
    Given an agent at seed-42
    When the agent records a failure of "isotope-variation"
    And the agent records a failure of "timing-correlation"
    Then the 42i distance is 32
    And the rung history is 0, 1, 2

  Scenario: the same failures in the other order reach rung 2 without rung 1
    Given an agent at seed-42
    When the agent records a failure of "timing-correlation"
    And the agent records a failure of "isotope-variation"
    Then the 42i distance is 32
    And the rung history is 0, 2
    And the distance matches the other ordering while the history does not

  Scenario: direction reads the multiset rather than the path
    Given the same two failures recorded in either order
    Then both agents read "coordinated-signaling"
    And the two distances are equal

  Scenario: the model has no move that fails
    Given an agent at seed-42
    When the agent records a failure of an undeclared family
    Then the 42i distance is 4
    And no error is reported
