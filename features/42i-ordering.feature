# features/42i-ordering.feature
# Source: isotope/plane (tree.go: admissibleEdges, key.go: Distance/Compare),
# isotope/cmd/plane (sort, edge, node), isotope/docs/lore-plane-mapping.md.
#
# Measured at commit 9cae211 on a three-plane fixture: charges 1,-1,-1, names
# alpha,beta,gamma, anchor 42, probe 42, tolerance 1e-9.
#
# The gap this file records: an isotope ID names a projection, the projection
# carries weights, and the order the tool reports does not read them. Identity is
# referentially rich and applicatively inert — the complement of
# "the ID says nothing about what applying it does" (features/brine-store.feature).
#
# lore-plane-mapping.md corrects an earlier version of itself here: the tensor
# was claimed load-bearing for the sort and is not. Its own words: "two different
# probes, not two different weights — and it cost a detour."
# 42i-marker: +42
@isotope @plane @ordering
Feature: Ordering does not read the weights that carry meaning
  As someone building a ranking from lore-weighted planes
  I want to know which surface the order actually reads
  So that I do not encode meaning in a slot the sort ignores

  Scenario: the order is fixed by the charges and the probe
    Given three planes with charges 1, -1, -1
    When the planes are sorted at probe 42
    Then the order is 0, 2, 1
    And nothing is quarantined

  Scenario: scaling every tensor block does not reach the order
    Given three planes with charges 1, -1, -1
    When every tensor block weight is multiplied by 100000
    Then the order is still 0, 2, 1

  Scenario: scaling every tensor block does reach the edge pairing
    Given three planes with charges 1, -1, -1
    When every tensor block weight is multiplied by 100000
    Then the edge between "alpha" and "beta" carries the scaled block weights
    And the edge reports approximately 566.6 + 33328.5i where it reported 0.0057 + 0.3333i

  Scenario: moving the encoder moves the projected value
    Given three planes with charges 1, -1, -1
    When the encoder gain is multiplied by 10
    Then the projected key of "alpha" is approximately -1083.7 + 572.8i
    And it was approximately -228.6 + 63.4i before

  Scenario: moving the encoder did not reach the order on this fixture
    Given three planes with charges 1, -1, -1
    When the encoder gain is multiplied by 10
    Then the order is still 0, 2, 1

  Scenario: the two weight surfaces are not interchangeable
    Given a model carrying an encoder and a tensor
    Then ordering reads the encoder surface
    And pairing reads the tensor surface
    And a ranking cannot be built from the tensor alone

  Scenario: a rank is a function of the charge and the probe rather than of the weights
    Given a fixed charge list and a fixed probe
    Then the order is the same for any tensor
    And changing the probe can change the order
