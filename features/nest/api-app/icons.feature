# family: deployment
# The icons service (api/src/webhook/icons.controller.ts). Mounted on the
# main app (3001); icon-change notifications also reach MCP sessions — see
# api-app/mcp.feature.
# Sources: api/src/webhook/icons.controller.ts.
# Captured 2026-11-03 from the prototype.

@api @icons
Feature: The icons service exposes the deck's icon layout and its stream
  As a deck observer
  I want the current icon layout readable and its changes streamable
  So that a display can mirror the deck without polling the RPC log

  Scenario: the icon layout is readable
    Given the main app is running on port 3001
    When a client requests GET /icons
    Then the current icon layout is returned as JSON

  Scenario: layout updates stream over SSE
    Given the main app is running on port 3001
    When a client opens GET /icons/stream
    Then the current layout arrives first
    And later layout changes arrive as they happen
