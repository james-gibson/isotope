# The auth service (api/src/auth/): AuthService (njwt issue/verify),
# AuthGuard (header-based, 423 on failure), AuthResolver.
# Mounted on the main app (3001); the auth app (3003) mounts the module but
# exposes no GraphQL — see api-auth/auth-app.feature. Token issuance is
# triggered through the viewer-count service's registerAuth — see
# api-app/viewer-count.feature.
# Sources: api/src/auth/auth.service.ts, api/src/auth/auth.guard.ts,
# api/src/auth/auth.resolver.ts.
# Captured 2026-11-03 from the prototype.

@api @auth
Feature: The auth service guards the GraphQL surface and verifies the token pair
  As a credentialed client
  I want uncredentialed access locked out and tokens verified against their keys
  So that the GraphQL surface is only usable with a real viewer identity

  Scenario: a guarded query without credentials is locked out
    Given the main app is running on port 3001
    And the client sends no authorization, x-site-token or x-viewer-id header
    When a client queries isTokenValid on /graphql
    Then the response status is 423

  Scenario: a guarded query with credentials answers its Boolean
    Given the main app is running on port 3001
    And the client sends a valid viewer token pair in the guard's headers
    When a client queries isTokenValid on /graphql
    Then the guard passes and the resolver answers a Boolean

  Scenario: refreshAuthToken is still a stub
    Given the main app is running on port 3001
    When a client runs the refreshAuthToken mutation on /graphql
    Then the response is "hello-world"
