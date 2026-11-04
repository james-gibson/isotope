# The viewer-count service (api/src/viewer-count/).
# Mounted on the main app (3001). registerAuth is the CLI's login flow — see
# cli/cli.feature.
# Sources: api/src/viewer-count/viewer-count.resolver.ts,
# api/src/auth/auth.service.ts (token signing).
# Captured 2026-11-03 from the prototype.

@api @viewer-count
Feature: The viewer-count service registers viewers and issues their token pair
  As a first-time visitor
  I want a viewer identity and its token pair from one mutation
  So that every later request can prove who it is

  Background:
    Given the Prisma model "View"
    Given the auth service

  Scenario: registering a viewer issues a fresh viewer id and records a View
    Given the main app is running on port 3001
    When a client runs the registerNewViewer mutation on /graphql
    Then the response is a fresh viewer id
    And a View row named "hello" is stored for that id

  Scenario: registerAuth returns the paired view and user tokens
    Given the main app is running on port 3001
    When a client runs the registerAuth mutation on /graphql
    Then the response is "<viewToken>:<token>"
    And the view token is signed with the view signing key
    And the user token is signed with the built-in demo user's private key

  Scenario: the viewer count query expires viewers after 30 seconds
    Given the main app is running on port 3001
    And a viewer id was counted more than 30 seconds ago
    When a client queries getCurrentViewerCount on /graphql
    Then the stale viewer is no longer counted
