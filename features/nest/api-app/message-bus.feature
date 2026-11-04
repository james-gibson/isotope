# family: deployment
# The message bus (api/src/message-bus/): PingCronService, PingController,
# StreamService, RoutingService (empty stub). Mounted on the main app (3001)
# and the runtime app (6001); the admin app registers none of it.
# Sources: api/src/message-bus/ping.cron.ts, api/src/message-bus/ping.controller.ts,
# api/src/message-bus/stream.service.ts.
# Captured 2026-11-03 from the prototype.

@api @message-bus @sse
Feature: The message bus streams house events over SSE on a 5-second heartbeat
  As a listening client
  I want house events as an SSE stream with a regular heartbeat
  So that displays stay live and dead connections are obvious

  Background:
    Given the auth guard

  Scenario: the ping cron feeds the SSE pong stream every 5 seconds
    Given the main app is running on port 3001
    When a client opens GET /webhooks/sse/pong
    Then within 10 seconds the stream delivers an event
    And another event arrives within every 5 seconds after that

  Scenario: the global stream is served as STACK_TICK events
    Given the main app is running on port 3001
    When a client opens GET /webhooks/sse
    Then house events arrive typed STACK_TICK
    # the family also serves /webhooks/sse/ping (uuid events), /webhooks/sse/empty
    # (opens, emits nothing) and the AuthGuard-gated /webhooks/sse/sw.
