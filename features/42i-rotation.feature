# features/42i-rotation.feature
# Source: isotope/plane (encoder.go: zeta[j] = z[j] - s.Anchor; space.go:
# ParseAnchor; model.go: Key), isotope/cmd/plane (node), and the canon in
# brine.py (isotope_id takes no anchor). Verified at commit 9cae211 by holding
# the probe fixed and varying only the anchor.
#
# The composition this file pins: an isotope ID is invariant under the anchor,
# and the anchor is a quarter-turn away from its alternative. Together they make
# a rotation — one identity, two readings, a turn apart.
#
#   identity   base64url(sha256(canonical raw))   no anchor in the computation
#   anchor     42 + 0i (proven consensus)         0 + 42i (working hypothesis)
#   turn       i * 42 = 42i                       a quarter-turn of the axis
#   frame      zeta = z - anchor                  the anchor is subtracted
#   effect     every plane moves by 42 - 42i      |42(1-i)| = 59.4 at -45 degrees
#
# The key is a transported sum, not a translation-invariant one, so the reading
# moves with the frame even though the identity cannot. That is why a position is
# comparable only within one anchor (42i-parameterisation), and why the same spec
# can look better or worse at another starting point without having changed.
# 42i-marker: +42
@isotope @plane @rotation
Feature: The anchor and the identity compose into a rotation
  As a reader comparing two readings of one spec
  I want to see what the anchor moves and what it cannot
  So that I never mistake a change of frame for a change of test

  Scenario: the anchor is subtracted from the probe frame
    Given a node built at anchor 42
    Then each plane's coordinate is the encoder value minus 42 + 0i

  Scenario: the two anchors are a quarter turn apart
    Given the anchors 42 and 42i
    Then multiplying 42 by i gives 42i
    And the two anchors differ only in which axis they fix

  Scenario: changing the anchor translates every plane by one displacement
    Given three planes with charges 1, -1, -1
    And the probe held fixed
    When the anchor changes from 42 to 42i
    Then every plane's coordinate moves by 42 - 42i
    And the displacement has magnitude 59.4 at an angle of -45 degrees

  Scenario: the identity is the same in both frames
    Given one spec
    Then its isotope ID does not depend on the anchor
    And both frames name the same identity

  Scenario: the derived key is not translation invariant
    Given three planes with charges 1, -1, -1
    And the probe held fixed
    When the anchor changes from 42 to 42i
    Then the key changes
    And the magnitude of the key changes as well

  Scenario: one identity has two readings a quarter turn apart
    Given one spec read at anchor 42 and at anchor 42i
    Then the identity is unchanged
    And the readings differ
    And neither reading is the test

  Scenario: the mirror anchor is a different concept rather than a sign flip
    Given the anchors 42i and -42i constructed directly
    Then both displace the frame by 59.4
    But 42i is verified plausible and -42i is plausible but unverified
    And their keys differ
    And the parser refuses -42i

  Scenario: the parser admits two of the four quarter-turn anchors
    Given the anchor parser
    Then it accepts 42 and 42i
    And it rejects -42i and -42
    And so one handedness of the mirror is not expressible

  Scenario: the model's distinguished magnitudes are 0, 1 and 42
    Given the plane's convention
    Then 0 is the origin
    And 1 is the unit of charge
    And 42 is the seed
    And nothing between the unit and the seed is distinguished
    But the charge parser does not forbid other magnitudes

  Scenario: the rotation is the same for every agent
    Given charge configurations from one plane to five
    And probes including arbitrary strings
    When each configuration is read at anchor 42 and at anchor 42i
    Then every plane displaces by 42 - 42i
    And the magnitude is 59.397 in every case
    And no plane is exempt

  Scenario: the rotation cannot rank two statements
    Given two different specs
    Then each displaces by 42 - 42i between the anchors
    And the displacement is the same for both
    And so the rotation carries no similarity between them

  Scenario: the rotation is a common mode that hides the subject
    Given two unrelated statements
    Then both displace by 42 - 42i
    And an observer who sees only the displacement cannot tell them apart
    And an observer who sees the position can
