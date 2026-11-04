# family: deployment
# Capture of the api CLI (api/src/cli.ts, commander) — a client of the
# GraphQL api, not part of any server app.
# Sources: api/src/cli.ts.
# Captured 2026-11-03 from the prototype.

@api @cli
Feature: The api CLI drives the GraphQL api from a stored viewer identity
  As a terminal user of the house
  I want register, auth and note commands backed by a stored identity
  So that the api is drivable without a browser

  Scenario: register records a viewer identity and chains into auth
    Given the api is reachable at API_URL
    When I run "node dist/cli.js register" in the api directory
    Then the viewer id from the addView mutation is stored in the config file
    And the auth flow runs immediately afterwards
    # config: ~/.api/config.json, keys token and viewerId.

  Scenario: auth splits the paired tokens and stores them
    Given the api is reachable at API_URL
    When I run "node dist/cli.js auth" in the api directory
    Then the registerAuth result is split on ":"
    And both tokens are stored in the config file
    And the JWT expiry is printed

  Scenario: every request carries the viewer headers
    Given a viewer identity is stored in the config file
    When I run "node dist/cli.js status" in the api directory
    Then the request carries the x-viewer-id and x-site-token headers
    And the isValidAuth result is printed

  Scenario: notes are readable from the CLI
    Given a viewer identity is stored in the config file
    When I run "node dist/cli.js list-notes" in the api directory
    Then the notes for the stored owner are listed

  Scenario: compiled JSX is fetchable from the CLI
    Given a viewer identity is stored in the config file
    When I run "node dist/cli.js get-jsx" with a note id
    Then the note's mdx, jsx, sha and nonce are printed

  Scenario: the CLI's default endpoint is a port no api app serves
    Given API_URL is not set
    When I run any cli command
    Then the request goes to localhost:3000/graphql
    # api/src/cli.ts default: the api's apps listen on 3001, 3002, 3003 and 6001,
    # so API_URL must point at one of them.
