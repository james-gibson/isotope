# The runtime app (RuntimeModule, port 6001) — the fourth api app, booted by
# "npm run start:runtime" rather than the main entrypoint. The P2P services it
# mounts are captured in their own files: nat-traversal.feature,
# peer-registration.feature, node-management.feature, plus the shared surfaces
# in api-app/ (user, dht, message-bus).
# Sources: api/src/runtime.ts, api/src/runtime.module.ts.
# Captured 2026-11-03 from the prototype.

@api @runtime
Feature: The runtime app carries the peer-to-peer stack on port 6001
  As a peer node of the house
  I want a dedicated app that mounts the P2P stack
  So that peer coordination runs without the public site's app

  Scenario: the runtime entrypoint boots its own app
    When I run "npm run start:runtime" in the api directory
    Then the runtime app listens on port 6001
    And mounts the user, dht, message-bus, nat-traversal, peer-registration and node-management services
    And runs the 5-second ping cron
    And serves no admin or auth app
