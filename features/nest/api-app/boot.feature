# The api main entrypoint (api/src/main.ts) — not a NestJS service itself, but
# the fact that makes the per-service split real: one entrypoint boots three
# separate Nest apps. Each app's services are captured in their own feature
# files under api-app/, api-admin/ and api-auth/.
# Sources: api/src/main.ts, api/package.json (scripts start, start:prod),
# api/launch.sh (structure only — contains credentials, never opened).
# Captured 2026-11-03 from the prototype.

@api @boot
Feature: One api entrypoint boots three separate apps on three ports
  As the operator of the house api
  I want the main, admin and auth apps booted as separate surfaces
  So that each app fails, restarts and is attacked on its own

  Scenario: the main entrypoint boots the main, admin and auth apps
    When the api main entrypoint boots
    Then the main app listens on port 3001
    And the admin app listens on port 3002
    And the auth app listens on port 3003

  Scenario: the production start runs the same entrypoint from the build
    When I run "npm run start:prod" in the api directory
    Then the three apps boot from the built dist/main
