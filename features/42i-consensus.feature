# features/42i-consensus.feature
# Source: isotope/consensus.go — VerifyAgreement, DetectCompromisedAlarm,
# VerifyConsensusSignatures, TimestampOrdering, VerifyIsotopeChain,
# GetConsensusGap. Values derived by running the package at commit 9cae211.
#
# Grounded vs chosen: F = floor((N-1)/3) and "consensus forms when the largest
# agreeing group reaches quorum" are standard results (Lamport/Shostak/Pease
# 1982), and the former is asserted by TestByzantineToleranceFiveAlarms. The gap
# weights (16 per dissenter, +32 for losing quorum) are parameters of this
# implementation, not derived from that result — see kb/scripts/lab/MATH.md.
#
# Known defect: isotope/devmode.go accumulates reports across calls without
# resetting, so a five-node cluster eventually reports F over a ten-report
# multiset (isotope#9). These scenarios drive VerifyAgreement with constructed
# reports and are unaffected; a devmode-derived count is not.
# 42i-marker: +42
@isotope @42i @consensus
Feature: Alarms reach consensus on an isotope before it is believed
  As a federation verifying what an alarm observed
  I want independent alarms to agree, and to prove who said what
  So that one compromised alarm cannot change the answer

  Scenario: unanimous agreement forms consensus
    Given three alarms reporting the same isotope
    When consensus is evaluated with a quorum of 2
    Then consensus is formed
    And 3 alarms agreed
    And no alarm is an outlier

  Scenario: a majority that meets quorum forms consensus
    Given two of three alarms report the same isotope
    When consensus is evaluated with a quorum of 2
    Then consensus is formed
    And 2 alarms agreed
    And 1 alarm is an outlier

  Scenario: a majority below quorum does not form consensus
    Given two of three alarms report the same isotope
    When consensus is evaluated with a quorum of 3
    Then consensus is not formed
    And 2 alarms agreed
    And 1 alarm is an outlier

  Scenario: a single outlier is identified as the compromised alarm
    Given three of four alarms report the same isotope
    When consensus is evaluated with a quorum of 3
    Then consensus is formed
    And the compromised alarm is "alarm-4"

  Scenario: several outliers within tolerance identify no single alarm
    Given two of four alarms report the same isotope
    When consensus is evaluated with a quorum of 3
    Then consensus is not formed
    And 2 alarms are outliers
    And no single alarm is identified as compromised

  Scenario: no reports means no consensus
    Given no alarm reports
    When consensus is evaluated with a quorum of 2
    Then consensus is not formed
    And 0 alarms agreed

  Scenario: byzantine tolerance is one third of the alarm count
    Given the byzantine tolerance of a federation
    Then 3 alarms tolerate 0 failures
    And 4 alarms tolerate 1 failure
    And 7 alarms tolerate 2 failures
    And 10 alarms tolerate 3 failures

  Scenario: a report whose signature does not verify is rejected
    Given an alarm report signed with a different key
    When the signatures are verified
    Then verification fails

  Scenario: an alarm with no known signing key cannot be verified
    Given an alarm report from an alarm with no signing key
    When the signatures are verified
    Then verification fails

  Scenario: reports must not go backwards in time
    Given reports whose timestamps decrease
    When the ordering is checked
    Then the ordering is rejected

  Scenario: reports sharing a timestamp are accepted
    Given reports with equal timestamps
    When the ordering is checked
    Then the ordering is accepted

  Scenario: a chain needs at least two links
    Given an isotope chain with one link
    When the chain is verified
    Then the chain is rejected

  Scenario: the first chain link must be the requester
    Given an isotope chain that starts with a secondary link
    When the chain is verified
    Then the chain is rejected

  Scenario: chain links must not go backwards in time
    Given an isotope chain whose second link is earlier than the first
    When the chain is verified
    Then the chain is rejected

  Scenario: perfect agreement adds no distance
    Given 3 of 3 alarms agree
    Then the consensus gap is 0

  Scenario: a dissenting alarm costs 16
    Given 2 of 3 alarms agree
    Then the consensus gap is 16

  Scenario: losing quorum costs 16 per dissenter and a further 32
    Given 1 of 3 alarms agree
    Then the consensus gap is 64

  Scenario: total disagreement costs the most
    Given 0 of 3 alarms agree
    Then the consensus gap is 80
