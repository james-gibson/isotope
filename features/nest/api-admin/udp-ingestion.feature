# family: deployment
# The UDP ingestion service (api/src/message-bus/udpService.ts) — a member of
# the message-bus module mounted ONLY by the admin app (3002). Sensor datagrams
# arrive encrypted; the key is derived from SENSOR_API_KEY (name only).
# Sources: api/src/message-bus/udpService.ts, api/src/admin.module.ts.
# Captured 2026-11-03 from the prototype.

@api @admin @sensors
Feature: The UDP ingestion service decrypts and logs sensor datagrams
  As the sensor fleet
  I want encrypted datagrams decrypted and their readings logged
  So that environmental readings arrive without an HTTP surface

  Scenario: an encrypted sensor datagram is decrypted and logged
    Given the admin app is running on port 3002
    And the sensor api key is configured
    When a device sends an AES-256-CBC encrypted datagram to UDP port 41234
    Then the datagram is decrypted with the key derived from the sensor api key
    And the temperature, pressure, battery and rssi readings are logged
