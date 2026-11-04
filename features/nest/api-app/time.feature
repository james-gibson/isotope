# The time service (api/src/time/): TimeResolver + TimeCronService.
# Active on the main app (3001) and the admin app (3002, resolver only).
# Each resolver publishes a view event to PUB_SUB, but no @Subscription
# resolver exists anywhere in the api, so the publishes have no subscriber.
# Sources: api/src/time/time.resolver.ts, api/src/time/time.cron.ts.
# Captured 2026-11-03 from the prototype.

@api @time
Feature: The time service answers current-time queries and refreshes the WAN IP on a cron
  As a clock-dependent client
  I want the api's own time answers and a scheduled WAN IP refresh
  So that time displays work and the house DNS tracks the real address

  Background:
    Given the feature "ddns-updater"

  Scenario: the GraphQL endpoint answers the current-time queries
    Given the main app is running on port 3001
    When a client queries getCurrentGMTTime on /graphql
    Then the response is the current GMT time as a String
    # getCurrentMTTime answers the same way.

  Scenario: the ip-update cron runs the DDNS helper every 5 minutes
    Given the main app is running on port 3001
    When the 5-minute ip-update cron fires
    Then the wan_ip.sh helper is executed as a child process
