# family: deployment
# The otel service (api/src/otel/): OtelModule -> HoneycombModule, loaded by
# the side-effect import in api/src/bootstrap.ts that every entrypoint pulls
# in before Nest starts. Env var NAMES only: HONEYCOMB_API_KEY,
# OTEL_SERVICE_NAME.
# Sources: api/src/bootstrap.ts, api/src/otel/otel.module.ts,
# api/src/otel/honeycomb.module.ts, api/src/otel/opentelemetry.module.ts.
# Captured 2026-11-03 from the prototype.

@api @otel
Feature: Tracing initializes with the app and exports only when a key is present
  As the operator watching the house
  I want tracing wired at boot without extra setup
  So that spans exist by default and export only when there is somewhere to send them

  Scenario: the base tracer initializes at boot
    When the api main entrypoint boots
    Then the base tracer and its bootstrap span are initialized

  Scenario: the honeycomb exporter starts only with an api key
    Given the honeycomb api key is set
    When the api main entrypoint boots
    Then traces export over OTLP HTTP with the node auto-instrumentations

  Scenario: without a key only the base tracer runs
    Given no honeycomb api key is set
    When the api main entrypoint boots
    Then no honeycomb exporter is initialized
