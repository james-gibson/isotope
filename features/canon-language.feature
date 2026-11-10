# features/canon-language.feature
# Source: isotope/canonicalization.go (keywordMap, normalizeStepText) and
# isotope/plane (encoder, ParseAnchor). Measured at commit 9cae211.
#
# The canon's documentation claims that one test written in several languages
# produces one isotope (README.md, USAGE.md). Measured, that is false for the
# step prose: only the *keywords* normalize. Filed as isotope#21.
#
#   english  TAgLxm5d0rrBytio   GIVEN:hash:120937abc278cbb8
#   spanish  bjAc0K_zDYIO8WEl   GIVEN:hash:b3eecf9b587d8ef0
#   french   TZJoGM73QSxMi7aL   GIVEN:hash:81a73d899e2766e7
#
# Same feature name, same scenario name, only the prose localized. The keyword
# normalized in all three; the hash did not.
#
# What *is* language-free is the tumbler: the frame rotation is 42 - 42i for
# every input, the charge is polarity rather than words, and the trace counts
# visits. Those layers read no words at all, which is why cross-language
# coordination belongs there and not on the name.
# 42i-marker: +42
@isotope @canon @language
Feature: The canon normalizes keywords and hashes prose
  As someone sharing a test between localized suites
  I want to know which layer is language-free
  So that I do not treat a translation as the same isotope

  Scenario: a localized keyword normalizes to the canonical step type
    Given a step beginning "Dado"
    Then the canonical step type is GIVEN
    And a step beginning "Alors" is THEN
    And the keyword itself carries no language

  Scenario: the step prose is hashed as written
    Given the step "the user is on the login page"
    And the step "que el usuario está en la página de inicio de sesión"
    Then the two hashes differ
    And article stripping does not make them equal

  Scenario: one test in three languages is three isotopes
    Given the same feature name and scenario name
    And the step prose localized into English, Spanish and French
    Then three different isotopes are produced
    And none is a duplicate of another

  Scenario: the documented claim holds for keywords only
    Given the canon's claim that all three produce the same signature
    Then the claim holds for the step type
    But it does not hold for the step hash
    And a translation is a new isotope

  Scenario: the tumbler layer is language-free
    Given the same test in any language
    Then the frame displacement is 42 - 42i
    And it is the same in every language
    And the rotation reads no words

  Scenario: the charge is polarity rather than vocabulary
    Given two statements in different languages
    Then each carries a charge from whether it asserts or denies
    And the words do not enter the charge

  Scenario: cross-language coordination belongs on the positional layer
    Given two suites that share a test but not a language
    Then their isotopes differ
    And their rotations, charges and visit counts do not
    And so coordination must use the positional layer

  Scenario: the keyword table is a discrete placeholder for a rotation
    Given the canonical step types GIVEN WHEN THEN AND BUT
    Then the table enumerates one rotation per language by hand
    And the rotation itself is continuous
    And a lookup table is what you write before you have the rotation

  Scenario: a hash has no angle
    Given an English step and its Spanish translation
    Then their isotope names differ in about half their bits
    And a one-letter edit differs by the same amount
    And so no partial rotation can be read from the name

  Scenario: with a reference marker the prose rotates
    Given a reference marker that fixes the origin
    And a projection of the prose into the plane
    Then a translation is a rotation about the marker
    And the rotation is measurable

  Scenario: the prose must be projected before it is hashed
    Given a pipeline that hashes the words and passes the hash as a probe
    Then the plane receives a name rather than a position
    And the prose is already destroyed by the time it arrives
    And so the rotation cannot be applied to it

  Scenario: a language rotation must be constant across tests
    Given the rotation between two languages measured on one test
    Then it must hold for every other test
    But a rotation that varies per test is a coincidence rather than a language relation
