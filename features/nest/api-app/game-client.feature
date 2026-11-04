# family: deployment
# The game-client service (api/src/game-client/): GameClientService +
# GameSession (outbound client of a local game server) and the mounted but
# empty GameClientController. Mounted on the main app (3001) and admin app
# (3002); reachable only in-process.
# Sources: api/src/game-client/game-client.service.ts,
# api/src/game-client/session.service.ts, api/src/game-client/game-client.controller.ts,
# api/src/game-client/game-client.constants.ts.
# Captured 2026-11-03 from the prototype.

@api @game-client
Feature: The game client speaks to the local game server in-process only
  As the house's game player
  I want a session that logs into the local game server and speaks its protocol
  So that the game is playable from the api's own tooling

  Scenario: a session logs into the local game server and follows its protocol
    Given the local game server is reachable
    When a game session starts
    Then the session posts its credentials to the game server's login endpoint
    And opens the game WebSocket with the issued access token
    And handles the auth, menu, stat and quit events

  Scenario: the mounted game-client controller exposes no routes
    Given the main app is running on port 3001
    When a client requests any path under /game-client
    Then the response status is 404
    # game-client.controller.ts is mounted but declares no routes.
