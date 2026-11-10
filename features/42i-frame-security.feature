# features/42i-frame-security.feature
# Source: isotope/plane (encoder.go: zeta = z - anchor; space.go: ParseAnchor;
# tree.go: admissibleEdges; model.go: Quarantine, SortPlanes), isotope/cmd/plane
# (sort). Measured at commit 9cae211 with the embedded corpus.
#
# Why a quarter turn is a trust check, in two parts:
#
#   frame     the anchor is subtracted from the probe. A reading is produced in
#             a frame; the same reading presented in the other frame is displaced
#             by 42(1-i) — 59.4 at -45 degrees. That is not a rounding error, so
#             moving a reading between frames is visible.
#   closure   a cycle in the tree must transport back to itself. A transported
#             cycle that returns rotated is quarantined instead of ordered, and
#             the event carries the angle and a signature.
#
# The analogy that makes it easy to hold: a tumbler lock. The frame is the angle
# you must hold the pick at; the closure is whether it turns all the way back.
# A turn that does not close jams the lock — it does not open slightly.
#
# What is enforced and what is argued:
#   enforced  anchor subtraction, two-anchor parser, quarantine on |holonomy|>tol,
#             signed quarantine event, deterministic angle and signature
#   argued    that this constitutes a security boundary. The order and the
#             quarantine read charges and probe; the tensor weights do neither
#             (42i-ordering.feature). So the check is structural, not semantic.
#
# The part that decides whether any of it is secure: a check that is not applied
# is not a check. Every mechanism here returns data — a quarantine flag, a
# displacement, a signature, a verdict — and none of them refuses anything on its
# own. Security is the consumer refusing to proceed. Two measured cases:
# VerifyAgreement forms consensus over signatures it never verifies
# (isotope#20), and VerifyConsensusSignatures panics on a short signature before
# it can reject one (isotope#19). Both are filed; neither is fixed.
# 42i-marker: +42
@isotope @plane @security
Feature: The quarter turn is part of the trust check
  As a reader deciding whether to trust a reading
  I want the frame and the closure of the plane to be checked
  So that a reading cannot be rotated or moved into a better one

  Scenario: a reading is bound to the anchor that produced it
    Given one spec at anchor 42 and at anchor 42i
    Then the frame moves by 42 - 42i
    And the reading differs even though the probe is held fixed

  Scenario: the identity is not bound to the anchor
    Given one spec
    Then its isotope ID is the same at both anchors
    And changing the frame cannot make the identity say something else

  Scenario: a reading presented in the other frame is displaced and therefore visible
    Given a reading produced at anchor 42
    When it is presented as a reading at anchor 42i
    Then it is displaced by 59.4 at an angle of -45 degrees
    And the displacement is far larger than rounding

  Scenario: the mirrored handedness cannot be claimed
    Given the anchors 42i and -42i
    Then they differ in the sign of the angle rather than in magnitude
    And the parser refuses -42i
    And a mirrored reading cannot be expressed through the tool

  Scenario: a comparison declares one anchor
    Given a comparison across planes
    Then the run fixes a single anchor
    And anchors are never mixed inside one sort

  Scenario: a cycle that does not close is quarantined
    Given a plane whose transported cycle returns a non-zero angle
    When the planes are sorted
    Then the result is quarantined rather than ordered
    And the event carries the cycle, the planes and the angle

  Scenario: the quarantine is deterministic
    Given the same model and the same probe
    When the sort runs twice
    Then the angle is -1.435714 radians
    And the signature begins GfAh_EL0el

  Scenario: a different probe yields a different quarantine
    Given the same model with the probe 42i
    Then the angle differs from the probe 42 reading
    And the signature differs as well

  Scenario: a tree with no cycle has nothing to close
    Given three planes joined by a spanning tree
    Then no quarantine event arises
    And an order is produced

  Scenario: agreement alone forms consensus over signatures it never verifies
    Given three reports carrying the same unverified signature
    When agreement is evaluated
    Then consensus forms
    And no signature has been checked
    And the outlier detector finds nothing

  Scenario: verifying the signatures is what rejects a forgery
    Given the same reports with a key map
    When the signatures are verified
    Then verification fails
    And the rejection happens only because the second check ran

  Scenario: a malformed signature is rejected rather than crashing the verifier
    Given a report whose signature is shorter than a digest
    When the signatures are verified
    Then verification fails
    And the verifier does not panic

  Scenario: an unread check protects nothing
    Given a quarantine event, a frame displacement and a degraded verdict
    Then each is an observation rather than a refusal
    And it protects only where a consumer refuses to proceed

  Scenario: the displacement is public rather than secret
    Given the rotation between the two anchors
    Then its value is 42 - 42i and anyone can compute it
    And it separates frames rather than authenticating them
    And it can be inverted by anyone who knows which frame a reading came from
    And so the frame must be declared and honoured rather than inferred

  Scenario: the boundary is visibility rather than concealment
    Given a partial view of the plane
    Then the displacement reveals nothing about the subject
    And the position does
    And so security depends on who may see the position

  Scenario: the rotation cannot detect a clone
    Given a cloned identity held at -42i
    Then the displacement between the anchors is still 42 - 42i
    And it is the same as for an honest subject
    And so the transform cannot separate them

  Scenario: the byte commitment separates the holder from the clone
    Given two claimants presenting the same identity
    Then the recorded byte hash matches at most one of them
    And a record whose canon does not hash to its name is a spoof
    And that check is what resolves the pair
