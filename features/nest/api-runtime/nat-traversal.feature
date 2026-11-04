# The NAT traversal service (api/src/nat-traversal/nat-traversal.service.ts),
# mounted by the runtime app (6001). Binds a random UDP port, sends a
# STUN-style probe whose answer is ignored, discovers the public IP via a
# public echo service (with a documented fallback address), and registers a
# NAT mapping with a public DNS resolver.
# Sources: api/src/nat-traversal/nat-traversal.service.ts.
# Captured 2026-11-03 from the prototype.

@api @runtime @nat-traversal
Feature: The NAT traversal service discovers the node's public address
  As a peer behind home networking
  I want my public address and NAT mapping discovered at boot
  So that other peers can reach me without manual configuration

  Scenario: boot discovers the public address through NAT traversal
    Given the runtime app is starting
    When NAT traversal initializes
    Then a UDP socket is bound on a random port
    And a STUN-style probe is sent and its answer ignored
    And the public IP is discovered via a public IP echo service with a documented fallback address
    And a NAT mapping registration packet is sent to a public DNS resolver on port 53
