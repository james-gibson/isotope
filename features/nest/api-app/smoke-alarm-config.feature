# The smoke-alarm config service (api/src/smoke-alarm/), evaluated through
# OpenFeature. Mounted on the main app (3001) and the auth app (3003).
# Sources: api/src/smoke-alarm/smoke-alarm-config.controller.ts,
# api/src/smoke-alarm/smoke-alarm-config.service.ts,
# api/src/smoke-alarm/smoke-alarm-config.types.ts, api/gofeature/goff-proxy.yaml,
# api/gofeature/goff/flags.yaml.
# Captured 2026-11-03 from the prototype.

@api @flags @config
Feature: Smoke-alarm configs are generated from OpenFeature evaluations that always fall back to defaults
  As a smoke-alarm instance
  I want a full config generated from my evaluation context
  So that a node can be provisioned from one URL

  Scenario: a config is generated from the query-string context
    Given the main app is running on port 3001
    When a client requests GET /config/generate with a targetingKey
    Then a full SmokeAlarmConfig document is returned
    # /config/adhd, /config/smoke-alarm and /config/fire-marshal answer the same way.

  Scenario: every flag evaluates to its in-code default
    Given no OpenFeature provider is registered
    And the goff flags file is empty
    When a client requests GET /config/generate
    Then every flag value in the document is the in-code default

  Scenario: the topology dump is served as the config surface's sibling
    Given the main app is running on port 3001
    When a client requests GET /config/d
    Then the prototypes, zones, applications and connection graph are returned

  Scenario: the vendored feature-flag relay is present but unwired
    Given the goff relay binary and its file-retriever config are vendored in the api tree
    When the api and the compose stacks boot
    Then nothing starts the relay
    And the api's OpenFeature provider wiring stays commented out
    And the relay's flags file is empty
