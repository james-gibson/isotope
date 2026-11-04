# family: deployment
# The DHT service (api/src/dht/): the Pegasus module — MagnetService,
# MagnetService, PegasusService + PegasusController.
# Mounted on the main app (3001) and the runtime app (6001). This is the
# magnet-URI registry — a different thing from the standalone Socket.IO
# tracker on port 5000 (dht-tracker/tracker.feature).
# Sources: api/src/dht/pegasus.controller.ts, api/src/dht/pegasus.service.ts,
# api/src/dht/magnet.service.ts, api/src/dht/magnet.service.ts,
# api/prisma/schema.prisma (model MagnetUri).
# Captured 2026-11-03 from the prototype.

@api @dht @pegasus
Feature: The Pegasus registry issues and reconciles content-addressed magnet URIs over the /s surface
  As a peer of the house
  I want magnet URIs issued, stored and reconciled by hash
  So that content stays addressable across the tracker and its peers

  Background:
    Given the Prisma model "MagnetUri"
    Given the feature "dht-tracker"

  Scenario: posting a hash issues a magnet URI
    Given the main app is running on port 3001
    When a client posts a hash to /s/<hash>
    Then the response is a magnet URI with xt urn:magnet:<hash>
    And the URI carries the tracker and web-seed endpoints
    And a MagnetUri row is stored for the hash with a sarcastic salt

  Scenario: issuing the same hash twice keeps one stored record
    Given a MagnetUri row already exists for a hash
    When a client posts the same hash to /s/<hash>
    Then the registry still holds one row for that hash

  Scenario: the registry lists every issued URI
    Given MagnetUri rows exist
    When a client requests GET /s
    Then every issued URI is returned

  Scenario: a stored URI is looked up by hash
    Given a MagnetUri row exists for a hash
    When a client requests GET /s/eed/<hash>
    Then the stored URI for that hash is returned

  Scenario: a reconciliation pass validates every stored record
    Given MagnetUri rows exist and some have no sync status
    When a client requests PATCH /s
    Then every record's magnet URI and hash are validated
    And records with no sync status are flagged for peer sync

  Scenario: boot reconciles the registry before serving
    Given MagnetUri rows exist with no sync status
    When the main app boots on port 3001
    Then the records are validated and flagged for peer sync
