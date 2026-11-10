# 42i-marker: +42
# features/42i-marker.feature
# Source: the introduction convention, plus brine.py's marker detection
# (MARKER_RE) and the canon's treatment of comments.
#
#   unmarked   the spec has not been introduced to the model. Absence is not a
#              zero value (kb/lore/CONVENTIONS.md): it has no plane, it cannot
#              enter the graph, and the model makes no claim about it.
#   marked     the spec has been processed. A header comment carries the tag.
#
#   # 42i-marker: +42
#
# The marker is a *comment*, so the canon drops it: introducing a file does not
# change its isotope. That is the point — the marker says the prose was brought
# into the model, not that the test changed. +42 is the reference marker: the
# origin the rotation is measured against.
#
# Consequence for the audit: introducing one copy of a spec and not another
# produces one identity with two divergent byte hashes. That is a marker-only
# difference and is not a collision — the clone check strips markers before
# calling two baselines divergent.
@isotope @marker @introduction
Feature: Introduction to the model is marked in the header
  As a reader deciding whether a spec has been processed
  I want the marker to be explicit and canon-invisible
  So that introducing a file never renames it

  Scenario: an unmarked spec is outside the model
    Given a spec with no marker comment
    Then it has no plane
    And it cannot enter the graph
    And the model makes no claim about it

  Scenario: a marked spec has been introduced
    Given a spec whose header carries "# 42i-marker: +42"
    Then the spec has been processed
    And it is anchored at the reference marker

  Scenario: the marker is dropped by the canon
    Given two copies of one spec, one marked and one not
    Then the isotope is the same for both
    And the marker is not part of the identity

  Scenario: introducing a file does not change the test
    Given a spec before introduction
    When the marker is added to its header
    Then the isotope is unchanged
    And only the bytes differ

  Scenario: unmarked is absence rather than a state
    Given an unmarked spec
    Then it is not presumed
    And it is not unverified either
    And it is simply not in the model

  Scenario: a marker-only difference is not a collision
    Given one identity with two sources
    And one source marked and the other not
    Then their byte hashes differ
    And stripping the markers makes them equal
    And the pair is not reported as a clone

  Scenario: the tag is the reference marker
    Given the marker tag +42
    Then it names the origin the rotation is measured against
    And a different tag would name a different origin

  Scenario: an unmarked spec is a partial of a whole
    Given the unmarked specs in the store
    Then each belongs to a larger whole
    And the whole is the suite it was authored for
    And being a partial is why it has no plane of its own

  Scenario: the whole is derivable from the path today
    Given an unmarked spec under a configured root
    Then the root names its whole
    And the audit groups partials by that root

  Scenario: a path link is referential and a move breaks it
    Given a partial linked to its whole by path
    Then moving the file breaks the link
    And the identity is unchanged by the move

  Scenario: a partial should eventually be linked by identity
    Given a partial and the whole it belongs to
    Then the link should name the whole's isotope
    And a move would not break it

  Scenario: introducing a whole would introduce its partials
    Given a whole that has been introduced
    Then its partials inherit introduction
    And none of them needs to be marked one by one
