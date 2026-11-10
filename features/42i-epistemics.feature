# features/42i-epistemics.feature
# Source: the lab's vocabulary plus isotope/plane's parser behaviour.
#
#   kb/lore/CONVENTIONS.md   real: (0,42) load-bearing | = 42 consensus | > 42 private
#                            "Absence is not a zero value."
#   kb/lore/OUTSTANDING.md   the imaginary axis is "where
#                            established/presumed/unestablished belongs".
#   plane/space.go           ParseAnchor admits 42 and 42i; ParseCharges admits
#                            signed integers, so -1 is data but not a frame.
#
# The plane is epistemic *by position*: the coordinates are not labels attached to
# a status, they are the status. Two scales, one point:
#
#   real     0        absence — cannot enter the graph at all
#            (0, 42)  presumed valid and true, not verified
#            = 42     consensus
#            > 42     true and verified (the private band)
#
#   imaginary  0      no plausibility content
#            +42i     verified plausible
#            -42i     plausible but unverified
#
#   charge   +1       defined true
#            -1       defined false
#
# Measured 2026-10-21: all 30 lore gradients are <= 1.0, so the corpus sits
# entirely in the presumed band. No compendium is at consensus and none is
# verified. kb/lore/OUTSTANDING.md records this as the raw gradient being stored
# where the tier belongs.
# 42i-marker: +42
@isotope @plane @epistemics
Feature: The plane's position carries epistemic status
  As a reader of a plane value
  I want to know what a coordinate commits me to
  So that a presumed claim is never read as a verified one

  Scenario: absence is not a value
    Given the charges 1, -1 and 0
    When the model is generated
    Then generation fails because the charge graph is disconnected
    And the neutral plane cannot enter the graph

  Scenario: the unit carries defined truth
    Given a plane with charge 1
    Then the charge means defined true
    And a plane with charge -1 means defined false

  Scenario: below consensus the real coordinate is presumed
    Given a real coordinate between 0 and 42
    Then the value is presumed valid and true
    And it is not verified

  Scenario: the real coordinate at 42 is consensus
    Given a real coordinate of exactly 42
    Then the value is consensus

  Scenario: above consensus the real coordinate is verified
    Given a real coordinate greater than 42
    Then the value is true and verified

  Scenario: the positive imaginary coordinate is verified plausibility
    Given an imaginary coordinate of 42i
    Then the claim is plausible and has been verified

  Scenario: the negative imaginary coordinate is unverified plausibility
    Given an imaginary coordinate of -42i
    Then the claim is plausible and unverified
    And the anchor parser refuses it

  Scenario: the coordinates are the status rather than labels on it
    Given a point on the plane
    Then its real coordinate says how far the claim is verified
    And its imaginary coordinate says how plausible it is
    And the geometry is the epistemic position

  Scenario: an unverified claim cannot be presented as a frame
    Given a reading that is merely plausible
    When it is offered as an anchor
    Then the parser refuses it
    And the reading has no frame in this tool

  Scenario: the tool reads proven and verified-plausible and nothing lower
    Given the two admitted anchors
    Then 42 is consensus
    And 42i is verified plausible
    And neither is a claim nobody has checked

  Scenario: a defined-false value is data rather than a frame
    Given the charge -1
    Then it is admissible as a plane's charge
    And it is not admissible as an anchor

  Scenario: a marker is still not a proof
    Given a reading at anchor 42i
    Then the frame says the claim has been verified
    And it does not certify the claim itself
    And certification remains the signature's work

  Scenario: entering the unverified quadrant is not disqualifying
    Given a reading at -42i
    Then the claim is held as a hypothesis
    And verification can promote it
    And the reading remains admissible

  Scenario: entering negative zero is disqualifying
    Given the coordinate -0
    Then it is absence carrying a sign
    And a signed absence is a contradiction rather than a hypothesis
    And the reading is disqualified

  Scenario: negative zero decides which side of the branch you arrive on
    Given the point -1 on the real axis
    Then an imaginary part of +0 gives a phase of +pi
    And an imaginary part of -0 gives a phase of -pi
    And the sign of zero is not a rounding artifact

  Scenario: the charge path erases the sign of zero
    Given the charge -0
    Then it parses as the integer 0
    And the sign is lost
    And a zero charge disconnects the graph

  Scenario: a hypothesis is held and a signed absence is refused
    Given the two ways of being absent from consensus
    Then -42i is a position with a path back
    And -0 is the origin claiming a direction
    And only the second disqualifies

  Scenario: a Given asserts a defined truth
    Given a step that establishes a precondition
    Then the statement carries charge 1
    And its opposite carries charge -1

  Scenario: the opposite is excluded while the precondition holds
    Given a precondition that currently holds
    Then the opposite has not happened at this moment
    But it may have happened earlier in the run
    And the exclusion is about the present state rather than the history

  Scenario: a When is an event rather than a claim
    Given a step that performs an action
    Then it carries no truth charge
    And an event that did not happen is absence
    And absence cannot enter the graph

  Scenario: a Then is a claim awaiting verification
    Given a step that asserts an expected observation
    Then the statement carries charge 1
    And its verification depth is the real coordinate
    And its plausibility is the imaginary coordinate

  Scenario: opposite statements couple
    Given two statements, one asserting and one denying the same property
    Then their charges sum to zero
    And the pair forms an admissible edge

  Scenario: a spec of only positive claims is structurally complete and scores zero
    Given three scenarios that all assert
    Then every pair sums to charge 2
    And every edge is within sector
    And the tensor score is exactly zero

  Scenario: a declared opposite is what makes a spec scoreable
    Given one scenario that asserts and one that denies the same property
    Then their pair sums to charge 0
    And the edge is cross sector
    And the tensor score is not zero

  Scenario: the corpus sits in the sector the score cannot read
    Given the lore corpus, whose planes are assigned without declared opposites
    Then its edges are almost entirely within sector
    And its structure sits where the score does not reach

  Scenario: a question is not a claim
    Given an interrogative step such as "has anyone visited your network?"
    Then it asserts nothing
    And it carries no truth charge
    And it is not a plausible claim either

  Scenario: the canon cannot see the question mark
    Given the question "has anyone visited your network?"
    And the statement "has anyone visited your network"
    Then both canonicalize to the same isotope
    And the interrogative mood is not part of the identity
    And so a question is indexed as the claim it asks about

  Scenario: a cloned identity is two claimants with different origins
    Given two networks answering to the same name
    Then each holds its own agreement about the origin
    And the name is shared while the baseline is not
    And the disagreement is about 0 rather than about the name

  Scenario: conflicting origins cannot share one frame
    Given two claimants that disagree about the origin
    Then neither can be placed at 42 without the other
    And neither is verified
    And the pair is held at -42i instead

  Scenario: absence written as a step becomes a claim
    Given the step "null"
    Then the canon hashes it as an ordinary claim
    And absence cannot be written without asserting it
    And the statement enters the graph

  Scenario: a claim about absence is not the absence itself
    Given the pair "null" and "null exists"
    Then the first is absence and the second is a claim about it
    And both are hashed as claims
    And neither is expressible as 0 or -0

  Scenario: the mock weights are an agreed reference rather than a validated set
    Given a weight set that claims to be weights
    And no validation behind it
    Then its content is a signed absence
    But the agreement that it exists makes it a public reference
    And the disqualifying case is claiming validation rather than being agreed
