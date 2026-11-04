# family: deployment
# The auth app (AuthModule, port 3003) — a separate, minimal Nest app booted
# by the main entrypoint. The auth SERVICE behaviours (guard, tokens) are
# captured in api-app/auth.feature; this file is the app's surface only.
# Sources: api/src/auth/auth.module.ts, api/src/authn/authn.controller.ts,
# api/src/smoke-alarm/smoke-alarm-config.module.ts.
# Captured 2026-11-03 from the prototype.

@api @auth
Feature: The auth app is a deliberately minimal surface on its own port
  As the house's auth boundary
  I want a tiny app with only the authn probe and config surface
  So that the auth surface exposes nothing else to attack

  Scenario: the auth app runs as its own process surface
    When the api main entrypoint boots
    Then the auth app listens on port 3003

  Scenario: the authn probe answers on the auth app
    Given the auth app is running on port 3003
    When a client requests GET /authn
    Then the response body is "hi"

  Scenario: the config surface is served on the auth app
    Given the auth app is running on port 3003
    When a client requests GET /config/generate
    Then a default SmokeAlarmConfig document is returned

  Scenario: the auth app mounts no root or GraphQL surface
    Given the auth app is running on port 3003
    When a client requests GET / on port 3003
    Then the response status is 404
    # GET /graphql is 404 the same way: the module registers no GraphQL module
    # and no root controller.
