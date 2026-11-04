# family: deployment
# The pull-requests service (api/src/pull-requests/).
# Mounted on the main app (3001). Reads a saved GitHub GraphQL search
# response — no live GitHub call is made.
# Sources: api/src/pull-requests/pull-requests.resolver.ts,
# api/src/mocks/state_of_current_prs.ts.
# Captured 2026-11-03 from the prototype.

@api @pull-requests
Feature: The pull-requests service filters a saved GitHub search snapshot
  As a maintainer watching the queue
  I want the current pull requests answerable from the api
  So that the queue is visible without opening GitHub

  Scenario: the current pull requests are filtered from the saved snapshot
    Given the main app is running on port 3001
    When a client queries getCurrentPullRequests on /graphql
    Then the response is a JSON string of the snapshot's entries for the gateway and zeus repositories
    # src/mocks/state_of_current_prs.ts — a saved GitHub GraphQL search response.
