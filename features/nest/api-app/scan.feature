# family: deployment
# The scan service (api/src/scan/): ScanResolver + ScanService, registered via
# the webhook module's providers — main app (3001) only.
# Sources: api/src/scan/scan.resolver.ts, api/src/scan/scan.service.ts,
# api/src/message-bus/stream.service.ts (the global stream it broadcasts on).
# Captured 2026-11-03 from the prototype.

@api @scan
Feature: The scan service broadcasts captured protocol events to the global stream
  As a scanner device
  I want a scan event to reach every listening client
  So that a scan is visible the moment it happens

  Background:
    Given the feature "message-bus"

  Scenario: a captured scan protocol event reaches the global SSE stream
    Given the main app is running on port 3001
    And a client is listening on GET /webhooks/sse
    When a client runs the captureProtocolEvent mutation with id "scan-1"
    Then the global SSE stream delivers the scan event
