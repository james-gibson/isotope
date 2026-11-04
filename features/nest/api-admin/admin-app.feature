# The admin app (AdminModule, port 3002) — a separate Nest app booted by the
# main entrypoint, not a route prefix on the main app. Its services are
# captured in their own files: the note surface (api-app/note.feature), the
# time resolver (api-app/time.feature) and the UDP sensor pair below.
# Sources: api/src/admin.module.ts.
# Captured 2026-11-03 from the prototype.

@api @admin
Feature: The admin app is its own process surface on port 3002
  As the operator of the admin surface
  I want the admin app distinct from the public app in process and surface
  So that admin behaviour is not reachable through the public port

  Scenario: the admin app runs as its own process surface
    When the api main entrypoint boots
    Then the admin app listens on port 3002
    And serves the note and time GraphQL resolvers
    And serves no SSE endpoints and runs no cron jobs
    # admin.module.ts registers no PingController and no ScheduleModule.
