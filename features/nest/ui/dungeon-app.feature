# Capture of the website's runtime surface (ui/src).
# Sources: ui/src/App.jsx, ui/src/index.jsx, ui/src/atomic/ (Dungeon, GameOver,
# MDX/DynamicJsxLoader), ui/src/Note/NoteManager, ui/src/service-worker/,
# ui/package.json (dev proxy), apollo.config.js (repo root).
# Captured 2026-11-03 from the prototype.

@ui @app
Feature: The site routes visitors through an ASCII dungeon and CSP-bound remote JSX
  As a visitor with a note key
  I want the dungeon as navigation and my note rendered as live content
  So that the site is explorable and note content is executable without loosening the CSP

  Scenario: the app talks GraphQL same-origin
    Given the site is served
    When the app bootstraps
    Then its Apollo client points at /graphql on the same origin
    # in development the dev server proxies /graphql to the api on localhost:3001.

  Scenario: a visitor without a key lands in the dungeon
    Given a visitor opens the site root with no key id
    When the route shell renders
    Then the dungeon crawler renders as the navigation surface

  Scenario: a visitor with a key lands in the note manager
    Given a visitor opens /key/<keyId>
    When the route shell renders
    Then the note manager renders for that key

  Scenario: logout and game-over render the game-over view
    Given a visitor opens /logout or /game-over
    When the route shell renders
    Then the game-over view renders

  Scenario: remote JSX executes only under its content hash and nonce
    Given the api returned compiled JSX with its sha256 hash and nonce placeholder
    When the app loads the remote JSX
    Then the script executes only under the content-security-policy script hash and nonce
