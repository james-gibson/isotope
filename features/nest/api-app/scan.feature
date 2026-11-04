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
    # Presupposition addressed by ground-truth isotope (canonical triple: signing
    # key "seed-42", the declared public constant of ADR 0004; family "nest"; v1)
    # rather than by magic name, per #40: the edge pins the
    # presupposed surface, so drift in the target breaks it loudly instead of
    # silently following a rename. Names as declared today:
    #   nest-v1:feb740bb18661244 = message-bus — the global SSE stream this
    #     scan broadcasts on (api/src/message-bus/stream.service.ts)
    Given the feature "nest-v1:feb740bb18661244"

  Scenario: a captured scan protocol event reaches the global SSE stream
    Given the main app is running on port 3001
    And a client is listening on GET /webhooks/sse
    When a client runs the captureProtocolEvent mutation with id "scan-1"
    Then the global SSE stream delivers the scan event
