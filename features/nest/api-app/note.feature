# The note service (api/src/note/): NoteResolver + NoteService.
# Mounted on the main app (3001) and the admin app (3002).
# Sources: api/src/note/note.resolver.ts, api/src/note/note.service.ts,
# api/prisma/schema.prisma (models Note, Link).
# Captured 2026-11-03 from the prototype.

@api @notes @mdx
Feature: Notes are stored with an audit link graph and served as CSP-bound compiled JSX
  As a note author on the site
  I want note writes to leave an auditable link trail and note reads to return executable JSX
  So that note history is reconstructable and remote MDX runs only under its own hash

  Background:
    Given the Prisma model "Link"
    Given the Prisma model "Note"

  Scenario: adding a note stores the note and its audit links
    Given the main app is running on port 3001
    When a client runs the addNote mutation on /graphql
    Then a Note row is created
    And noteLink and auditLink rows are created for it

  Scenario: updating a note copies the original and records an edit link
    Given a note exists
    When a client runs the updateNote mutation on /graphql
    Then the note is copied for the edit
    And a noteEdit link is recorded

  Scenario: referencing a note records the reference and audit links
    Given a note exists
    When a client runs the referenceNote mutation on /graphql
    Then noteReference, noteLink and auditLink rows are created

  Scenario: getJSX returns server-compiled JSX bound to its own content hash
    Given a note with MDX content exists
    When a client queries getJSX with id "random" on /graphql
    Then a random note's MDX is compiled to JSX
    And the response carries a Content-Security-Policy header allowing only scripts matching the JSX's sha256 hash
    And the response carries a nonce placeholder for the edge to rewrite

  Scenario: the same note surface ships on the admin app
    Given the admin app is running on port 3002
    When a client runs the addNote mutation on /graphql
    Then a Note row is created with its audit links
