# The main app's own root controller and service.
# Sources: api/src/app.controller.ts, api/src/app.service.ts,
# api/test/app.e2e-spec.ts (the only e2e assertion in the api).
# Captured 2026-11-03 from the prototype.

@api @app @root
Feature: The root controller answers a bare probe with the e2e-asserted greeting
  As a client probing the main app
  I want the root path to answer immediately and observably
  So that liveness is checkable without knowing any other route

  Scenario: a bare probe of the main app gets the e2e-asserted greeting
    Given the main app is running on port 3001
    When a client requests GET /
    Then the response status is 200
    And the response body is "Hello World!"

  Scenario: the greeting leaves a metric trail
    Given the main app is running on port 3001
    When a client requests GET /
    Then the sheep counter is incremented
    And a heap-usage gauge is registered
    # app.controller.ts: counter.add(1) and the node-monitor gauge run on every request.
