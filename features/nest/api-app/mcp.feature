# family: deployment
# The MCP bridge (api/src/webhook/mcp/): McpController, McpService,
# SessionStore. Mounted on the main app (3001) via the webhook module.
# MCP protocol version 2025-11-25.
# Sources: api/src/webhook/mcp/mcp.controller.ts, api/src/webhook/mcp/mcp.service.ts,
# api/src/webhook/mcp/session.store.ts.
# Captured 2026-11-03 from the prototype.

@api @mcp
Feature: The /mcp bridge exposes the deck as an MCP server over HTTP
  As an MCP client
  I want the deck's state and tools reachable as a session-scoped MCP server
  So that agents can observe and drive the deck like any other client

  Scenario: initialize creates a session and returns its id
    Given the main app is running on port 3001
    When a client posts an initialize request to /mcp
    Then a session is created
    And the response carries the Mcp-Session-Id header

  Scenario: the server advertises its nine tools
    Given an initialized MCP session
    When a client posts a tools/list request to /mcp
    Then the response lists nine tools
    And the tools include get_icons, set_icons, cycle_color, simulate_event, duplicate_event, orchestrate, register_webhook, list_webhooks and list_sessions

  Scenario: a ping answers with an empty object
    Given an initialized MCP session
    When a client posts a ping request to /mcp
    Then the response is an empty JSON object

  Scenario: the initialized notification is acknowledged
    Given an initialized MCP session
    When a client posts a notifications/initialized notification to /mcp
    Then the response status is 202

  Scenario: real deck events fan out to every open session
    Given two MCP sessions are open
    When a client posts a streamdeck.event to /rpc
    Then both sessions receive a deck event notification on their SSE streams

  Scenario: an unknown session's notification stream is refused
    Given no session exists for the id sent
    When a client opens GET /mcp
    Then the response status is 404
    # open streams heartbeat every 15 seconds to defeat proxy timeouts.

  Scenario: deleting a session tears it down
    Given an initialized MCP session
    When a client sends DELETE /mcp
    Then the response status is 204
