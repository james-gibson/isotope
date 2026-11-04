# The webhook service (api/src/webhook/webhook.controller.ts + payload
# converters). Mounted on the main app (3001).
# Sources: api/src/webhook/webhook.controller.ts,
# api/src/webhook/githubWebhook.service.ts, api/src/webhook/siteFartWebhook.service.ts,
# api/src/webhook/dto/github.markdown.dto.ts.
# Captured 2026-11-03 from the prototype.

@api @webhook
Feature: The /hookers surface converts inbound webhook payloads into stored markdown notes
  As an automation feeding the house
  I want webhook payloads converted into readable notes
  So that advisories and diagnostics are kept with the rest of the notes

  Scenario: a signed GitHub advisory or dependabot payload becomes a markdown note
    Given the main app is running on port 3001
    When a client posts a GitHub security-advisory payload to /hookers/github with a valid x-hub-signature-256 signature
    Then the payload is converted to a markdown report
    And a Note is stored from the report

  Scenario: an invalid GitHub signature still records the note before failing
    Given the main app is running on port 3001
    When a client posts a GitHub payload to /hookers/github with an invalid signature
    Then a Note is stored from the payload
    And the handler responds "Webhook Handler Error"
    # webhook.controller.ts: the note is created before the signature check throws.

  Scenario: an OBD diagnostics payload becomes a markdown report note
    Given the main app is running on port 3001
    When a client posts a Freematics OBD diagnostics payload to /hookers/honey
    Then a rich markdown report of the diagnostics is stored as a Note

  Scenario: a fail2ban webhook is logged only
    Given the main app is running on port 3001
    When a client posts a payload to /hookers/fail2ban
    Then the payload is logged
    And no Note is stored
