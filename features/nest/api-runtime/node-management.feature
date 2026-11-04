# family: deployment
# The node-management service (api/src/node-management/), mounted by the
# runtime app (6001). In-memory node map; messages go out over UDP via the NAT
# traversal service. registerSync is a stub (commented body).
# Sources: api/src/node-management/node-management.service.ts.
# Captured 2026-11-03 from the prototype.

@api @runtime @node-management
Feature: The node-management service tracks nodes and routes messages to them
  As the coordinator of the swarm
  I want known nodes tracked, messageable and findable by capability
  So that work can be routed to whichever node can do it

  Background:
    Given the feature "nat-traversal"

  Scenario: nodes are tracked with their NAT info
    Given the runtime app is running
    When a node is added and its NAT info updated
    Then the node is tracked until it is set offline

  Scenario: messages are sent to nodes over UDP
    Given a tracked node
    When a message is sent to the node
    Then the message goes out over UDP via the NAT traversal service

  Scenario: nodes are findable by capability
    Given tracked nodes advertise capabilities
    When a caller asks for nodes that can a capability
    Then every matching online node is returned
