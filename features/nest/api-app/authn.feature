# family: deployment
# The authn service (api/src/authn/).
# Sources: api/src/authn/authn.controller.ts.
# AuthNController is mounted by all four apps (3001, 3002, 3003, 6001). The
# authn service and its 5-minute wan_ip cron (authn.service.ts, authn.cron.ts)
# are defined but registered nowhere — dead code.
# Captured 2026-11-03 from the prototype.

@api @authn
Feature: The authn probe answers identically on every api app
  As any client of the house
  I want one trivially checkable probe shared by every app
  So that reachability of each surface is testable with one request shape

  Scenario: the authn probe answers on every api app
    Given any api app is running
    When a client requests GET /authn
    Then the response body is "hi"
