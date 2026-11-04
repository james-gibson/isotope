# family: deployment
# The replay-protection service (api/src/message-bus/replay-protection-service.ts),
# registered by the admin app (3002) and used by the UDP ingestion service.
# Device ids are salted hashes; sequence state persists to
# data/sequence-records.json.
# Sources: api/src/message-bus/replay-protection-service.ts, api/src/admin.module.ts.
# Captured 2026-11-03 from the prototype.

@api @admin @replay-protection
Feature: The replay-protection service enforces per-device sequence numbers
  As the admin app
  I want datagram replays rejected and sequence state durable
  So that a captured datagram cannot be replayed against the house

  Scenario: a replayed datagram is rejected
    Given the admin app is running on port 3002
    And a datagram with sequence number <n> was already accepted from a device
    When the same device sends a datagram with sequence number <n> again
    Then the datagram is rejected as a replay

  Scenario: per-device sequence state survives a restart
    Given the admin app accepted datagrams from a device
    When the admin app restarts
    Then the device's last accepted sequence number is still enforced
