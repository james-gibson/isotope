# Capture of the standalone BitTorrent-style tracker prototype served by the
# repo-root package (npm run tracker|client|monitor -> node dist/cli.js, built
# from prototypes/dht/). This is a separate service from the NestJS api and from
# the api's Pegasus /s registry.
# Sources: prototypes/dht/tracker.ts, prototypes/dht/client.ts,
# prototypes/dht/monitor.ts, prototypes/dht/cli.ts, package.json (repo root).
# Wiring caveats: the current cli.ts no longer imports monitor.ts (npm run
# monitor works only via the stale built dist/cli.js), and the dht / lookup /
# announce / put / get subcommands are unwired (startNode is undefined) — see
# prototypes/dht/.pickled/README.md. Captured 2026-11-03 from
# the prototype.

@dht @tracker
Feature: The prototype tracker registers peers over HTTP and Socket.IO on port 5000
  As a swarm member
  I want a tracker that remembers peers, survives restarts and drops the stale
  So that a small swarm can find itself without a public tracker

  Scenario: the tracker starts on its own port
    When I run "npm run tracker"
    Then the tracker listens on port 5000 over HTTP and Socket.IO

  Scenario: an announce upserts the node and returns the peer list
    Given the tracker is running on port 5000
    When a client posts a node announcement to /announce
    Then the node is upserted with its data and address
    And the current peer list is returned
    And tracker:update and node:announce broadcasts go out over Socket.IO

  Scenario: stats report the tracker's identity and counters
    Given the tracker is running on port 5000
    When a client requests GET /stats
    Then the response reports the tracker id, node count, uptime and announce count

  Scenario: the node list is readable
    Given the tracker is running on port 5000
    When a client requests GET /nodes
    Then every registered node is returned

  Scenario: tracker state is saved periodically and on exit
    Given the tracker is running with a tracker id
    When 30 seconds pass
    Then the tracker state is saved to data/<trackerId>.tracker.json
    And the same state is saved on SIGINT

  Scenario: stale nodes are pruned on a 60-second pass
    Given a node has not announced for more than 30 minutes
    When the 60-second cleanup pass runs
    Then the stale node is removed

  Scenario: a client re-announces every 60 seconds
    Given the tracker is running on port 5000
    When I run "npm run client"
    Then the client announces on connect
    And re-announces every 60 seconds

  Scenario: the monitor renders the tracker dashboard
    Given the tracker is running on port 5000
    When I run "npm run monitor"
    Then a terminal dashboard renders the tracker info, peers and logs
    And the dashboard refreshes from /stats and /nodes every 5 seconds
