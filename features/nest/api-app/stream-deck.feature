# family: deployment
# The stream-deck service (api/src/webhook/stream-deck.controller.ts +
# stream-deck.service.ts). Mounted on the main app (3001).
# Sources: api/src/webhook/stream-deck.controller.ts,
# api/src/webhook/stream-deck.service.ts, api/src/webhook/dto/stream-deck.dto.ts.
# Captured 2026-11-03 from the prototype.

@api @rpc @stream-deck
Feature: The stream-deck service records deck events as a bounded replayable log
  As a deck owner
  I want pedal and key events recorded, bounded and replayable
  So that the deck's history is observable and the palette can be driven by foot

  Scenario: a streamdeck.event JSON-RPC request is accepted and stored
    Given the main app is running on port 3001
    When a client posts a JSON-RPC 2.0 streamdeck.event request to /rpc
    Then the response reports ok, receivedAt and stored
    And the event is stored

  Scenario: any other JSON-RPC method is refused
    Given the main app is running on port 3001
    When a client posts a JSON-RPC 2.0 request with a different method to /rpc
    Then the response is the -32601 method-not-found error

  Scenario: a pedal press advances the deck palette
    Given the deck has a 15-key layout with a 7-colour palette
    When a client posts a pedal down event to /rpc
    Then the palette advances and the layout revision is bumped

  Scenario: the event log is newest-first and bounded at 100
    Given more than 100 events have been posted to /rpc
    When a client requests GET /rpc/events
    Then the 100 most recent events are returned newest-first

  Scenario: the SSE stream replays the last 25 events then follows live
    Given the main app is running on port 3001
    When a client opens GET /rpc/stream
    Then a snapshot of the last 25 events arrives first
    And events posted afterwards arrive live
