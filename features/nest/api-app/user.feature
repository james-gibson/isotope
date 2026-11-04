# family: deployment
# The user service (api/src/user/).
# Mounted on the main app (3001) and the runtime app (6001).
# Sources: api/src/user/user.resolver.ts, api/prisma/schema.prisma (model User).
# Captured 2026-11-03 from the prototype.

@api @user
Feature: The user service checks tokens and records users
  As a registered user of the house
  I want my token checkable and my record creatable and fetchable
  So that identity is more than a viewer id

  Background:
    Given the Prisma model "User"
    Given the auth service

  Scenario: an authorization token is checked for validity
    Given the main app is running on port 3001
    When a client queries isValidAuth on /graphql with an authorization header
    Then the response is that token's validity

  Scenario: a user is fetched by id
    Given a User row exists
    When a client queries getUser with its id on /graphql
    Then the user is returned

  Scenario: a user is added by name
    Given the main app is running on port 3001
    When a client runs the addUser mutation on /graphql
    Then a User row is stored
