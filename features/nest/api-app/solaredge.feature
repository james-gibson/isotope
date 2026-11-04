# The solaredge service (api/src/solaredge/): SolarEdgeCronService —
# registered in AppModule but every @Cron poll is commented out, so nothing
# ever fires. SolarEdgeService (the empty shell) is registered nowhere.
# Sources: api/src/solaredge/solaredge.cron.ts, api/src/solaredge/solaredge.service.ts.
# Captured 2026-11-03 from the prototype.

@api @solaredge
Feature: The solaredge service is registered but inert
  As the operator
  I want the in-practice behaviour stated plainly
  So that nobody waits for solar telemetry that never arrives

  Scenario: the registered solaredge module never polls
    Given the main app is running on port 3001
    When the module's poll window arrives
    Then no SolarEdge dashboard endpoint is requested
    # solaredge.cron.ts: every @Cron decorator is commented out.
