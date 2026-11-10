# features/42i-parameterisation.feature
# Source: isotope/plane (doc.go, encoder.go, space.go), isotope/cmd/plane, and
# kb/scripts/lab/MATH.md + PLANE.md. Every figure in the other 42i specs is an
# output of this parameterisation, not a constant of the model.
#
# The claim this file exists to prevent: that 16, 32, 60, 220, or any single
# projection is "the" 42i number. The value is complex; the anchor chooses the
# origin; a different starting point yields a different reading of the same
# charge set. Values below were measured by running the package at commit
# 9cae211.
# 42i-marker: +42
@isotope @42i @parameterisation
Feature: The 42i numbers are a parameterised projection
  As someone deriving from the lab's figures
  I want to know which numbers are grounded and which are chosen
  So that a derivation does not chase a placeholder

  Scenario: exactly one anchor is fixed per run
    Given a plane run
    Then the anchor is 42 + 0i on the real axis, or 0 + 42i on the imaginary axis

  Scenario: the anchor cannot move while a run is measuring
    Given a plane run that has fixed its anchor
    When two values are compared within that run
    Then both were measured from the same origin
    And an origin moved mid-run would make their positions incomparable

  Scenario: the anchor's magnitude sets the base of the derived parameters
    Given charges "1,-1,-1"
    When the model is generated at anchor 42
    Then the base of the derivation is 42

  Scenario: both permitted anchors share a magnitude
    Given charges "1,-1,-1"
    When the model is generated at anchor 42 and again at anchor 42i
    Then the generated encoder parameters are identical
    And the only difference is the anchor itself, 42 + 0i against 0 + 42i

  Scenario: a value projected at anchor 42 sits near the real axis
    Given charges "1,-1,-1" with planes named "alpha,beta,gamma"
    When node "alpha" is projected at anchor 42
    Then the key is near -228.6 + 63.4i

  Scenario: the same value projected at anchor 42i lands elsewhere
    Given charges "1,-1,-1" with planes named "alpha,beta,gamma"
    When node "alpha" is projected at anchor 42i
    Then the key is near -9.1 - 95.8i
    And that reading differs from the one at anchor 42

  Scenario: nothing in the plane is fitted
    Given the plane package
    Then every encoder parameter is closed-form arithmetic on the charges and the anchor
    And no value is found by searching or learning

  Scenario: the weights file is the only external numeric input
    Given a generated weights file
    Then it carries the encoder parameters and the signed tensor block weights
    And every other number follows from the charges and the anchor

  Scenario: the prototype weight table is hand-chosen
    Given the default 42i weight table in fortytwointegration.go
    Then its eleven families carry hand-chosen weights
    And no weight is the result of a fit against measured departure

  Scenario: three schemes share the word rung
    Given the lab's rung vocabulary
    Then fortytwointegration.go assigns ceilings 0, 20, 60, 100, 140, 180, 220
    And trust-rungs.md assigns ceilings infinite, infinite, 80, 40, 16, 110, 220
    And the engine maps distance 0 to rung 0, 8 to 1, 12 to 1, 48 to 2 and 80 to 3

  Scenario: the catalogue's ceilings are not monotonic
    Given the catalogue rungs in trust-rungs.md
    Then rung 2 tolerates a distance of 80
    But rung 3 tolerates only 40

  Scenario: a rung carries the meaning of the file that produced it
    Given a rung reported by one component
    Then it is not combined with a ceiling taken from another component's table

  Scenario: distance and rung are readouts of a computed position
    Given an agent's 42i distance
    Then the distance is computed from the traffic log and the isotope records
    And the rung is the bucket the distance falls into
    And the rung is not an input to the distance

  Scenario: one relation is grounded
    Given a federation of N alarms
    Then it tolerates at most floor((N - 1) / 3) byzantine failures
    And consensus forms when the largest agreeing group reaches the required quorum

  Scenario: plane 42 is where the weights exist
    Given a model read at anchor 42
    Then its encoder parameters are the ones that exist
    And the anchor records them as present rather than planned

  Scenario: plane 42i is where we expect them
    Given a model read at anchor 42i
    Then its encoder parameters are the ones expected
    And the same weight set is read at the other origin

  Scenario: one weight set appears in both frames
    Given one charge list and one weights file
    When it is read at anchor 42 and at anchor 42i
    Then the encoder parameters are identical
    And the tensor blocks are identical
    And only the anchor differs

  Scenario: the displacement is the gap between existence and expectation
    Given the two frames
    Then the displacement is 42 - 42i
    And it does not depend on the weights
    And it is the same for every charge set

  Scenario: shipping a weight set moves it from expectation to existence
    Given an expected weight set at 42i
    When it ships
    Then it exists at 42
    And a new expectation forms at 42i
    And the previous set is superseded rather than destroyed

  Scenario: the frame gap is weight-independent
    Given the same charges read under two different weight sets
    Then the frame gap is 42 - 42i under both
    And it is the same for every statement
    And so the gap between existence and expectation does not depend on the weights

  Scenario: the content gap is weight-dependent
    Given two statements read under one weight set
    Then their content gap is definite
    When the weight set changes
    Then the content gap changes with it
    And so a comparison is only meaningful under a declared set

  Scenario: the key is transported so its gap is neither
    Given the key gap between two frames
    Then it is not 42 - 42i
    And it is not the content gap
    And only the zeta displacement is the frame gap

  Scenario: comparing two statements requires a declared weight set
    Given the pair "null" and "null exists"
    Then their identities do not depend on the weights
    And their distance does
    And the canonical weight set is what makes the distance reproducible

  Scenario: the weights in force are mock
    Given the plane's default build
    Then the encoder parameters are closed-form on the charges and the anchor
    And the tensor blocks are closed-form as well
    And the corpus weights are not shipped in the default build
    And no value is fitted to anything

  Scenario: a reading taken with mock weights is not evidence
    Given a comparison made under the default weights
    Then the comparison reflects the placeholder
    And it is not a measurement of the statements
    And only weight-independent quantities survive it

  Scenario: the mock set represents unvalidated input
    Given the mock weight set
    Then it stands for input the system has not validated
    And it marks the boundary rather than an interior state
    And it is the shape a shipped set would arrive in

  Scenario: the public reference is the hypothesis model
    Given the model packed into every binary
    Then its anchor is 0 + 42i
    And it is the generic hypothesis rather than the corpus
    And every build shares it

  Scenario: the validated weights are private
    Given the corpus weights
    Then they are encrypted and gated behind a build tag
    And they are not in the default build
    And so the public set is the hypothesis and the private set is the validation

  Scenario: the public reference carries nothing beyond its formula
    Given the public reference read in either frame
    Then the encoder parameters are the same
    And a frame change loses nothing
    And there was nothing validated to lose

  Scenario: agreement makes the reference and verification makes the value
    Given a set everyone agrees exists
    Then it is agreed rather than verified
    And its readings are reproducible between parties who share it
    And they are not evidence about the subject
