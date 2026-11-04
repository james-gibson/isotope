# The view service (api/src/view/).
# Mounted on the main app (3001). The 5-minute expire-unclaimed-views cron
# (view.cron.ts) is defined but registered nowhere — dead code.
# Sources: api/src/view/view.resolver.ts, api/prisma/schema.prisma (model View).
# Captured 2026-11-03 from the prototype.

@api @view
Feature: The view service stores named views
  As a viewer naming a surface
  I want views created and fetched by name and id
  So that surfaces like the kitchen monitor survive sessions

  Background:
    Given the Prisma model "View"

  Scenario: a view is created by name
    Given the main app is running on port 3001
    When a client runs the addView mutation with a name on /graphql
    Then a View row is stored with that name

  Scenario: a view is fetched by id
    Given a View row exists
    When a client queries getView with its id on /graphql
    Then the view is returned
